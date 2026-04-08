package deal_test

import (
	"testing"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/storage"
)

func mustOpenDB(t *testing.T) *storage.Database {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func defaultRules() map[string]config.DealRule {
	return map[string]config.DealRule{
		"smartphone": {MinPrice: 50, MaxPrice: 350, MinDiscountPct: 10},
	}
}

func TestFirstSeenInRange(t *testing.T) {
	db := mustOpenDB(t)
	ev := deal.NewEvaluator(db, defaultRules(), 24)

	result := ev.Evaluate("Samsung Galaxy A15", "smartphone", "Galaxus", 149.0, nil, "https://example.com/1", "https://img.com/1.jpg")

	if result.Deal == nil {
		t.Fatalf("expected deal, got skipped: %s", result.Reason)
	}
	if !result.Deal.IsFirstSeen {
		t.Error("IsFirstSeen = false, want true")
	}
	if result.Deal.Price != 149.0 {
		t.Errorf("Price = %f, want 149.0", result.Deal.Price)
	}
}

func TestFirstSeenOutOfRange(t *testing.T) {
	db := mustOpenDB(t)
	ev := deal.NewEvaluator(db, defaultRules(), 24)

	// Price above max.
	result := ev.Evaluate("iPhone 16 Pro", "smartphone", "Galaxus", 999.0, nil, "url", "img")
	if result.Deal != nil {
		t.Error("should not be a deal — price above max")
	}

	// Price below min.
	result2 := ev.Evaluate("Cheap Phone", "smartphone", "Galaxus", 10.0, nil, "url", "img")
	if result2.Deal != nil {
		t.Error("should not be a deal — price below min")
	}
}

func TestReturningProductSufficientDiscount(t *testing.T) {
	db := mustOpenDB(t)
	ev := deal.NewEvaluator(db, defaultRules(), 24)

	// First evaluation — first seen, in range, is a deal.
	ev.Evaluate("Samsung Galaxy A15", "smartphone", "Galaxus", 300.0, nil, "url", "img")

	// Second evaluation — price dropped 17% (300 → 249), should be deal.
	result := ev.Evaluate("Samsung Galaxy A15", "smartphone", "Galaxus", 249.0, nil, "url", "img")
	if result.Deal == nil {
		t.Fatalf("expected deal on price drop, got skipped: %s", result.Reason)
	}
	if result.Deal.IsFirstSeen {
		t.Error("IsFirstSeen should be false")
	}
	if result.Deal.DiscountPct < 16.0 || result.Deal.DiscountPct > 18.0 {
		t.Errorf("DiscountPct = %f, want ~17%%", result.Deal.DiscountPct)
	}
}

func TestReturningProductInsufficientDiscount(t *testing.T) {
	db := mustOpenDB(t)
	ev := deal.NewEvaluator(db, defaultRules(), 24)

	// First evaluation.
	ev.Evaluate("Samsung Galaxy A15", "smartphone", "Galaxus", 300.0, nil, "url", "img")

	// Second evaluation — price dropped only 3% (300 → 290), below min_discount_pct.
	result := ev.Evaluate("Samsung Galaxy A15", "smartphone", "Galaxus", 290.0, nil, "url", "img")
	if result.Deal != nil {
		t.Error("should not be deal — insufficient discount")
	}
}

func TestSanityBoundsViolation(t *testing.T) {
	db := mustOpenDB(t)
	ev := deal.NewEvaluator(db, defaultRules(), 24)

	// Price outside [min*0.1, max*10] = [5, 3500].
	result := ev.Evaluate("Bug Phone", "smartphone", "Galaxus", 0.01, nil, "url", "img")
	if result.Deal != nil {
		t.Error("should reject — below sanity bound")
	}
	if result.Reason != "sanity" {
		t.Errorf("Reason = %q, want %q", result.Reason, "sanity")
	}

	result2 := ev.Evaluate("Mega Phone", "smartphone", "Galaxus", 5000.0, nil, "url", "img")
	if result2.Deal != nil {
		t.Error("should reject — above sanity bound")
	}
}

func TestNoCategoryRule(t *testing.T) {
	db := mustOpenDB(t)
	ev := deal.NewEvaluator(db, defaultRules(), 24)

	result := ev.Evaluate("Random Item", "unknown_cat", "Galaxus", 100.0, nil, "url", "img")
	if result.Deal != nil {
		t.Error("should skip — no category rule")
	}
	if result.Reason != "no_rule" {
		t.Errorf("Reason = %q, want %q", result.Reason, "no_rule")
	}
}

func TestCooldownDedup(t *testing.T) {
	db := mustOpenDB(t)
	ev := deal.NewEvaluator(db, defaultRules(), 24)

	// First evaluation — first seen, deal.
	result := ev.Evaluate("Samsung Galaxy A15", "smartphone", "Galaxus", 149.0, nil, "url", "img")
	if result.Deal == nil {
		t.Fatal("first evaluation should be a deal")
	}

	// Record the notification.
	id, _, _ := db.UpsertProduct("Samsung Galaxy A15", "smartphone")
	_ = db.RecordNotification(id, "Galaxus", 149.0)

	// Third evaluation — new product+shop (first seen) but cooldown blocks it.
	// We need a different shop to trigger first-seen again on same product.
	_ = db.RecordNotification(id, "Amazon", 140.0)
	result2 := ev.Evaluate("Samsung Galaxy A15", "smartphone", "Amazon", 140.0, nil, "url", "img")
	if result2.Deal != nil {
		t.Error("should be skipped — within cooldown")
	}
	if result2.Reason != "cooldown" {
		t.Errorf("Reason = %q, want %q", result2.Reason, "cooldown")
	}
}
