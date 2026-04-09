package pipeline

import (
	"testing"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/currency"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/parser"
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

func TestTransformProduct(t *testing.T) {
	db := mustOpenDB(t)
	conv := currency.New(db, "", "CHF", 24)

	cat := config.ShopCategory{Currency: "CHF"}

	p := parser.RawProduct{Title: "SAMSUNG Galaxy A16 128GB", Price: 149.0}
	cleaned, priceCHF, skip := transformProduct(p, cat, nil, nil, conv)

	if skip {
		t.Fatal("should not skip")
	}
	if priceCHF != 149.0 {
		t.Errorf("priceCHF = %f, want 149.0", priceCHF)
	}
	if cleaned != "Samsung Galaxy A16 128GB" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestTransformProductWithDivisor(t *testing.T) {
	db := mustOpenDB(t)
	conv := currency.New(db, "", "CHF", 24)

	cat := config.ShopCategory{Currency: "CHF", PriceDivisor: 100}

	p := parser.RawProduct{Title: "Test Phone", Price: 14900.0}
	_, priceCHF, skip := transformProduct(p, cat, nil, nil, conv)

	if skip {
		t.Fatal("should not skip")
	}
	if priceCHF != 149.0 {
		t.Errorf("priceCHF = %f, want 149.0", priceCHF)
	}
}

func TestTransformProductFiltered(t *testing.T) {
	db := mustOpenDB(t)
	conv := currency.New(db, "", "CHF", 24)

	cat := config.ShopCategory{Currency: "CHF"}
	filter := func(name string) bool { return true } // skip everything

	p := parser.RawProduct{Title: "Test Phone", Price: 100.0}
	_, _, skip := transformProduct(p, cat, nil, filter, conv)

	if !skip {
		t.Error("should be skipped by filter")
	}
}

func TestEvaluateProductDeal(t *testing.T) {
	db := mustOpenDB(t)
	rules := map[string]config.DealRule{
		"smartphone": {MinPrice: 50, MaxPrice: 350, MinDiscountPct: 10},
	}
	eval := deal.NewEvaluator(db, rules, 24)
	shop := config.Shop{Name: "TestShop"}
	cat := config.ShopCategory{Category: "smartphone"}

	p := parser.RawProduct{Title: "Test Phone", Price: 149.0, URL: "https://example.com"}

	pr, d := evaluateProduct("Test Phone", 149.0, p, cat, shop, eval, false)

	if d == nil {
		t.Fatal("expected deal for first-seen product in range")
	}
	if !pr.IsDeal {
		t.Error("ProductResult.IsDeal should be true")
	}
	if pr.Price != 149.0 {
		t.Errorf("pr.Price = %f, want 149.0", pr.Price)
	}
}

func TestEvaluateProductSeedMode(t *testing.T) {
	db := mustOpenDB(t)
	rules := map[string]config.DealRule{
		"smartphone": {MinPrice: 50, MaxPrice: 350, MinDiscountPct: 10},
	}
	eval := deal.NewEvaluator(db, rules, 24)
	shop := config.Shop{Name: "TestShop"}
	cat := config.ShopCategory{Category: "smartphone"}

	p := parser.RawProduct{Title: "Test Phone", Price: 149.0}

	pr, d := evaluateProduct("Test Phone", 149.0, p, cat, shop, eval, true)

	if d != nil {
		t.Error("seed mode should not produce deals")
	}
	if pr.IsDeal {
		t.Error("ProductResult.IsDeal should be false in seed mode")
	}
}
