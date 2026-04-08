package parser

import (
	"fmt"

	"github.com/trancee/DealScout/internal/config"
)

// Parse extracts products from raw response data using the shop category's configuration.
// Routes to:
//   - Embedded JSON parser (if JSONSelector + Fields are set)
//   - HTML parser (if Selectors are set)
//   - JSON parser (if Fields are set)
func Parse(shopCat config.ShopCategory, data []byte, baseURL string) ([]RawProduct, error) {
	switch {
	case shopCat.JSONSelector != "" && len(shopCat.Fields) > 0:
		return ParseEmbeddedJSON(data, shopCat.JSONSelector, shopCat.Fields)
	case len(shopCat.Selectors) > 0:
		return ParseHTML(data, shopCat.Selectors, baseURL)
	case len(shopCat.Fields) > 0:
		return ParseJSON(data, shopCat.Fields)
	default:
		return nil, fmt.Errorf("shop category %q has neither selectors nor fields configured", shopCat.Category)
	}
}
