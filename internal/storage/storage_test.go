package storage_test

import (
	"testing"

	"github.com/trancee/DealScout/internal/storage"
)

func mustOpen(t *testing.T) *storage.Database {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenCreatesTables(t *testing.T) {
	db := mustOpen(t)

	tables := db.Tables()
	want := map[string]bool{
		"products":           false,
		"price_history":      false,
		"deal_notifications": false,
		"exchange_rates":     false,
	}

	for _, name := range tables {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("table %q not created", name)
		}
	}
}

func TestUpsertProduct(t *testing.T) {
	db := mustOpen(t)

	// First insert — should be new.
	id1, isNew, err := db.UpsertProduct("Samsung Galaxy A15", "smartphone")
	if err != nil {
		t.Fatalf("first UpsertProduct: %v", err)
	}
	if !isNew {
		t.Error("first insert: isNew = false, want true")
	}
	if id1 == 0 {
		t.Error("first insert: id = 0, want non-zero")
	}

	// Same name+category — should return existing.
	id2, isNew2, err := db.UpsertProduct("Samsung Galaxy A15", "smartphone")
	if err != nil {
		t.Fatalf("second UpsertProduct: %v", err)
	}
	if isNew2 {
		t.Error("second insert: isNew = true, want false")
	}
	if id2 != id1 {
		t.Errorf("second insert: id = %d, want %d", id2, id1)
	}

	// Different category — should be new.
	id3, isNew3, err := db.UpsertProduct("Samsung Galaxy A15", "laptop")
	if err != nil {
		t.Fatalf("third UpsertProduct: %v", err)
	}
	if !isNew3 {
		t.Error("different category: isNew = false, want true")
	}
	if id3 == id1 {
		t.Error("different category: id should differ from first")
	}
}

func TestPriceHistory(t *testing.T) {
	db := mustOpen(t)

	id, _, _ := db.UpsertProduct("iPhone 16", "smartphone")

	// No history yet.
	_, found, err := db.LastPrice(id, "Galaxus")
	if err != nil {
		t.Fatalf("LastPrice (empty): %v", err)
	}
	if found {
		t.Error("LastPrice (empty): found = true, want false")
	}

	// Append first price.
	oldPrice := 999.0
	if err := db.AppendPriceHistory(id, "Galaxus", 899.0, "CHF", &oldPrice, "https://example.com/1"); err != nil {
		t.Fatalf("AppendPriceHistory: %v", err)
	}

	price, found, err := db.LastPrice(id, "Galaxus")
	if err != nil {
		t.Fatalf("LastPrice: %v", err)
	}
	if !found {
		t.Fatal("LastPrice: found = false, want true")
	}
	if price != 899.0 {
		t.Errorf("LastPrice = %f, want 899.0", price)
	}

	// Append second price — LastPrice should return the newest.
	if err := db.AppendPriceHistory(id, "Galaxus", 849.0, "CHF", nil, "https://example.com/2"); err != nil {
		t.Fatalf("AppendPriceHistory (second): %v", err)
	}

	price2, _, _ := db.LastPrice(id, "Galaxus")
	if price2 != 849.0 {
		t.Errorf("LastPrice after second append = %f, want 849.0", price2)
	}

	// Different shop — should not see Galaxus prices.
	_, found3, _ := db.LastPrice(id, "Amazon")
	if found3 {
		t.Error("LastPrice for different shop: found = true, want false")
	}
}

func TestNotificationCooldown(t *testing.T) {
	db := mustOpen(t)

	id, _, _ := db.UpsertProduct("Pixel 9", "smartphone")

	// No notification yet — cooldown should be false.
	notified, err := db.WasNotifiedWithinCooldown(id, "Galaxus", 24)
	if err != nil {
		t.Fatalf("WasNotifiedWithinCooldown (empty): %v", err)
	}
	if notified {
		t.Error("cooldown (empty): got true, want false")
	}

	// Record a notification.
	if err := db.RecordNotification(id, "Galaxus", 299.0); err != nil {
		t.Fatalf("RecordNotification: %v", err)
	}

	// Now within cooldown.
	notified2, err := db.WasNotifiedWithinCooldown(id, "Galaxus", 24)
	if err != nil {
		t.Fatalf("WasNotifiedWithinCooldown (after): %v", err)
	}
	if !notified2 {
		t.Error("cooldown (after notification): got false, want true")
	}

	// Different shop — not notified.
	notified3, _ := db.WasNotifiedWithinCooldown(id, "Amazon", 24)
	if notified3 {
		t.Error("cooldown (different shop): got true, want false")
	}
}

func TestExchangeRateCache(t *testing.T) {
	db := mustOpen(t)

	// No rate yet.
	_, fresh, err := db.ExchangeRate("EUR", 24)
	if err != nil {
		t.Fatalf("ExchangeRate (empty): %v", err)
	}
	if fresh {
		t.Error("ExchangeRate (empty): fresh = true, want false")
	}

	// Upsert a rate.
	if err := db.UpsertExchangeRate("EUR", 0.93); err != nil {
		t.Fatalf("UpsertExchangeRate: %v", err)
	}

	rate, fresh2, err := db.ExchangeRate("EUR", 24)
	if err != nil {
		t.Fatalf("ExchangeRate (after upsert): %v", err)
	}
	if !fresh2 {
		t.Error("ExchangeRate (after upsert): fresh = false, want true")
	}
	if rate != 0.93 {
		t.Errorf("ExchangeRate = %f, want 0.93", rate)
	}

	// Update the rate.
	if err := db.UpsertExchangeRate("EUR", 0.95); err != nil {
		t.Fatalf("UpsertExchangeRate (update): %v", err)
	}

	rate2, _, _ := db.ExchangeRate("EUR", 24)
	if rate2 != 0.95 {
		t.Errorf("ExchangeRate after update = %f, want 0.95", rate2)
	}

	// Different currency — not found.
	_, fresh3, _ := db.ExchangeRate("GBP", 24)
	if fresh3 {
		t.Error("ExchangeRate (GBP): fresh = true, want false")
	}
}

func TestPruneOldPriceHistory(t *testing.T) {
	db := mustOpen(t)

	id, _, _ := db.UpsertProduct("AirPods Pro", "headphones")

	// Insert a recent record.
	if err := db.AppendPriceHistory(id, "Galaxus", 199.0, "CHF", nil, "https://example.com/1"); err != nil {
		t.Fatal(err)
	}

	// Insert an old record by manipulating timestamp directly.
	db.ExecRaw("INSERT INTO price_history (product_id, shop, price, currency, url, timestamp) VALUES (?, ?, ?, ?, ?, datetime('now', '-100 days'))",
		id, "Galaxus", 249.0, "CHF", "https://example.com/2")

	// Prune with 90-day retention.
	deleted, err := db.PruneOldPriceHistory(90)
	if err != nil {
		t.Fatalf("PruneOldPriceHistory: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Recent record should still exist.
	price, found, _ := db.LastPrice(id, "Galaxus")
	if !found {
		t.Fatal("recent record was pruned")
	}
	if price != 199.0 {
		t.Errorf("remaining price = %f, want 199.0", price)
	}
}
