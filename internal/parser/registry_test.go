package parser_test

import (
	"os"
	"testing"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/parser"
)

func TestRegistryHTML(t *testing.T) {
	html, _ := os.ReadFile("testdata/amazon_listing.html")

	cat := config.ShopCategory{
		Category: "smartphones",
		Parsing: config.Parsing{
			Selectors: map[string]string{
				"product_card": "div[data-component-type='s-search-result']",
				"title":        "h2 a span",
				"price":        "span.a-price span.a-offscreen",
				"url":          "h2 a[href]",
				"image":        "img.s-image[src]",
			},
		},
	}

	products, err := parser.Parse(cat, html, "https://www.amazon.de")
	if err != nil {
		t.Fatalf("Parse (html): %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}
}

func TestRegistryJSON(t *testing.T) {
	data, _ := os.ReadFile("testdata/galaxus_response.json")

	cat := config.ShopCategory{
		Category: "smartphones",
		Parsing: config.Parsing{
			Fields: map[string]string{
				"products": "data.productType.filterProducts.products.results",
				"title":    "product.name",
				"price":    "offer.price.amountInclusive",
				"url":      "product.productId",
				"image":    "product.imageUrl",
			},
		},
	}

	products, err := parser.Parse(cat, data, "")
	if err != nil {
		t.Fatalf("Parse (json): %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}
}

func TestRegistryUnknownType(t *testing.T) {
	cat := config.ShopCategory{Category: "test"}
	_, err := parser.Parse(cat, []byte("data"), "")
	if err == nil {
		t.Fatal("expected error for unknown source type")
	}
}
