package parser

import (
	"encoding/json"
	"fmt"
	"strings"
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

	productsRaw := walkPath(root, productsPath)

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
		title := walkString(item, fields["title"])
		if titlePrefix := walkString(item, fields["title_prefix"]); titlePrefix != "" && title != "" {
			title = titlePrefix + " " + title
		}
		if title == "" {
			continue
		}

		price, err := walkFloat(item, fields["price"])
		if err != nil {
			continue
		}

		product := RawProduct{
			Title:    title,
			Price:    price,
			URL:      walkString(item, fields["url"]),
			ImageURL: walkString(item, fields["image"]),
		}

		if oldPricePath := fields["old_price"]; oldPricePath != "" {
			if oldPrice, err := walkFloat(item, oldPricePath); err == nil {
				product.OldPrice = &oldPrice
			}
		}

		products = append(products, product)
	}

	return products, nil
}

func walkPath(data interface{}, path string) interface{} {
	if path == "" || data == nil {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
		if current == nil {
			return nil
		}
	}

	return current
}

func walkString(data interface{}, path string) string {
	if path == "" {
		return ""
	}
	val := walkPath(data, path)
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func walkFloat(data interface{}, path string) (float64, error) {
	if path == "" {
		return 0, fmt.Errorf("empty path")
	}
	val := walkPath(data, path)
	if val == nil {
		return 0, fmt.Errorf("path %q resolved to nil", path)
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case string:
		return ParsePrice(v)
	default:
		return 0, fmt.Errorf("unexpected type %T at path %q", val, path)
	}
}
