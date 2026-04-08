package parser_test

import (
	"testing"

	"github.com/trancee/DealScout/internal/parser"
)

const embeddedHTML = `<html><body>
<script type="application/json" id="product-data">{
  "items": {
    "SKU001": {
      "brand": {"name": "Samsung"},
      "description": {"title": "Galaxy A15 128 GB"},
      "pricing": {"current": 149.0, "original": 199.0},
      "link": "/product/sku001",
      "img": "/images/sku001.jpg"
    },
    "SKU002": {
      "brand": {"name": "Apple"},
      "description": {"title": "iPhone 16 256 GB"},
      "pricing": {"current": 899.0},
      "link": "/product/sku002",
      "img": "/images/sku002.jpg"
    }
  }
}</script>
</body></html>`

func TestParseEmbeddedJSON(t *testing.T) {
	fields := map[string]string{
		"products":     "items",
		"title_prefix": "brand.name",
		"title":        "description.title",
		"price":        "pricing.current",
		"old_price":    "pricing.original",
		"url":          "link",
		"image":        "img",
	}

	products, err := parser.ParseEmbeddedJSON([]byte(embeddedHTML), "script#product-data", fields)
	if err != nil {
		t.Fatalf("ParseEmbeddedJSON: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}

	// Map products are unordered, so find by title content.
	var samsung, apple *parser.RawProduct
	for i := range products {
		switch products[i].Title {
		case "Samsung Galaxy A15 128 GB":
			samsung = &products[i]
		case "Apple iPhone 16 256 GB":
			apple = &products[i]
		}
	}

	if samsung == nil {
		t.Fatal("Samsung product not found")
	}
	if samsung.Price != 149.0 {
		t.Errorf("Samsung price = %f, want 149.0", samsung.Price)
	}
	if samsung.OldPrice == nil || *samsung.OldPrice != 199.0 {
		t.Errorf("Samsung old_price = %v, want 199.0", samsung.OldPrice)
	}

	if apple == nil {
		t.Fatal("Apple product not found")
	}
	if apple.OldPrice != nil {
		t.Errorf("Apple old_price should be nil, got %f", *apple.OldPrice)
	}
}

func TestParseEmbeddedJSONMissingScript(t *testing.T) {
	_, err := parser.ParseEmbeddedJSON([]byte("<html></html>"), "script#missing", nil)
	if err == nil {
		t.Error("expected error for missing script")
	}
}
