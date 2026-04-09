package parser

import (
	"encoding/json"
	"fmt"

	"github.com/trancee/DealScout/internal/jsonpath"
)

// ParseJSON extracts products from raw JSON using dot-notation field paths.
// Required fields: "products" (path to array), "title", "price".
// Optional fields: "old_price", "url", "image".
func ParseJSON(data []byte, fields map[string]string) ([]RawProduct, error) {
	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	productsPath := fields["products"]
	if productsPath == "" {
		return nil, fmt.Errorf("missing required field: products")
	}

	productsRaw := jsonpath.Walk(root, productsPath)

	var items []interface{}
	switch v := productsRaw.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		for _, val := range v {
			if val != nil {
				items = append(items, val)
			}
		}
	default:
		return nil, fmt.Errorf("products path %q resolved to %T, want array or map", productsPath, productsRaw)
	}

	var products []RawProduct

	for _, item := range items {
		title := jsonpath.String(item, fields["title"])
		if titlePrefix := jsonpath.String(item, fields["title_prefix"]); titlePrefix != "" && title != "" {
			title = titlePrefix + " " + title
		}

		var price float64
		if fields["price"] != "" {
			if p, err := jsonpath.Float(item, fields["price"]); err == nil {
				price = p
			}
		}

		product := RawProduct{
			Title:    title,
			Price:    price,
			URL:      jsonpath.String(item, fields["url"]),
			ImageURL: jsonpath.String(item, fields["image"]),
		}

		if oldPricePath := fields["old_price"]; oldPricePath != "" {
			if oldPrice, err := jsonpath.Float(item, oldPricePath); err == nil {
				product.OldPrice = &oldPrice
			}
		}

		products = append(products, product)
	}

	return products, nil
}
