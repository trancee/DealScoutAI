package pipeline

import (
	"os"
	"strings"
	"testing"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/parser"
)

func TestParsePriceResponse_Array(t *testing.T) {
	data := []byte(`{
		"response": {
			"products": [
				{"articleId": "123", "offers": {"offer": {"price": {"price": 99.95, "crossedOutPrice": 129.95}}}},
				{"articleId": "456", "offers": {"offer": {"price": {"price": 149.00}}}}
			]
		}
	}`)

	api := &config.PriceAPI{
		ProductsPath: "response.products",
		IDPath:       "articleId",
		PricePath:    "offers.offer.price.price",
		OldPricePath: "offers.offer.price.crossedOutPrice",
	}

	prices, err := ParsePriceResponse(data, api)
	if err != nil {
		t.Fatal(err)
	}

	if len(prices) != 2 {
		t.Fatalf("got %d prices, want 2", len(prices))
	}

	p123 := prices["123"]
	if p123.Price != 99.95 {
		t.Errorf("Price[123] = %.2f, want 99.95", p123.Price)
	}
	if p123.OldPrice == nil || *p123.OldPrice != 129.95 {
		t.Errorf("OldPrice[123] = %v, want 129.95", p123.OldPrice)
	}

	p456 := prices["456"]
	if p456.Price != 149.00 {
		t.Errorf("Price[456] = %.2f, want 149.00", p456.Price)
	}
	if p456.OldPrice != nil {
		t.Errorf("OldPrice[456] should be nil, got %v", *p456.OldPrice)
	}
}

func TestParsePriceResponse_Map(t *testing.T) {
	data := []byte(`{
		"SKU001": {"sku": "SKU001", "effectivePricing": {"userPrice": 599.00, "mainPrice": 699.00}, "description": {"title": "Laptop X"}, "cover": {"sizes": [{"link": "https://img.jpg"}]}},
		"SKU002": {"sku": "SKU002", "effectivePricing": {"userPrice": 399.00}, "description": {"title": "Laptop Y"}}
	}`)

	api := &config.PriceAPI{
		PricePath:    "effectivePricing.userPrice",
		OldPricePath: "effectivePricing.mainPrice",
		TitlePath:    "description.title",
		ImagePath:    "cover.sizes.0.link",
	}

	prices, err := ParsePriceResponse(data, api)
	if err != nil {
		t.Fatal(err)
	}

	if len(prices) != 2 {
		t.Fatalf("got %d prices, want 2", len(prices))
	}

	p1 := prices["SKU001"]
	if p1.Price != 599.00 {
		t.Errorf("Price = %.2f, want 599.00", p1.Price)
	}
	if p1.Title != "Laptop X" {
		t.Errorf("Title = %q, want Laptop X", p1.Title)
	}
	if p1.ImageURL != "https://img.jpg" {
		t.Errorf("ImageURL = %q", p1.ImageURL)
	}
}

func TestMergePrices(t *testing.T) {
	products := []parser.RawProduct{
		{Title: "", URL: "123", Price: 0},
		{Title: "", URL: "456", Price: 0},
		{Title: "", URL: "789", Price: 0},
	}

	prices := map[string]PriceInfo{
		"123": {Price: 99.95, Title: "Product A", ImageURL: "https://a.jpg"},
		"456": {Price: 149.00},
	}

	result := MergePrices(products, prices)

	if len(result) != 2 {
		t.Fatalf("got %d products, want 2 (unmatched dropped)", len(result))
	}

	if result[0].Price != 99.95 || result[0].Title != "Product A" || result[0].ImageURL != "https://a.jpg" {
		t.Errorf("product[0] = %+v", result[0])
	}
	if result[1].Price != 149.00 {
		t.Errorf("product[1].Price = %.2f, want 149.00", result[1].Price)
	}
}

func TestBuildPriceRequestBody(t *testing.T) {
	dir := t.TempDir()
	tplPath := dir + "/template.json"
	tpl := `{"ids": "{ids}", "articles": {articles}}`
	if err := os.WriteFile(tplPath, []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}

	products := []parser.RawProduct{
		{URL: "100"},
		{URL: "200"},
	}

	api := &config.PriceAPI{BodyTemplate: tplPath}
	body, err := BuildPriceRequestBody(products, api)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(body, "100,200") {
		t.Errorf("body should contain '100,200', got: %s", body)
	}
	if !strings.Contains(body, `"articleID":"100"`) {
		t.Errorf("body should contain article 100, got: %s", body)
	}
}

func TestPriceEnricher_EndToEnd(t *testing.T) {
	api := &config.PriceAPI{
		PricePath:    "price",
		OldPricePath: "oldPrice",
		TitlePath:    "title",
	}

	mockFetch := func(method, url, body string, headers map[string]string) ([]byte, error) {
		return []byte(`{
			"P1": {"sku": "P1", "price": 99.00, "oldPrice": 129.00, "title": "Phone A"},
			"P2": {"sku": "P2", "price": 199.00, "title": "Phone B"}
		}`), nil
	}

	products := []parser.RawProduct{
		{URL: "P1"},
		{URL: "P2"},
		{URL: "P3"},
	}

	enricher := NewPriceEnricher(api, mockFetch)
	result := enricher.Enrich(products)

	if len(result) != 2 {
		t.Fatalf("got %d products, want 2", len(result))
	}
	if result[0].Price != 99.00 || result[0].Title != "Phone A" {
		t.Errorf("product[0] = %+v", result[0])
	}
	if result[1].Price != 199.00 || result[1].Title != "Phone B" {
		t.Errorf("product[1] = %+v", result[1])
	}
}
