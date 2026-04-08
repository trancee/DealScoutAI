package pipeline

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/fetcher"
	"github.com/trancee/DealScout/internal/parser"
)

// enrichPrices fetches prices from a secondary API and merges them into products.
// Products without a price match are returned with Price=0 (filtered by sanity bounds later).
func enrichPrices(products []parser.RawProduct, priceAPI *config.PriceAPI, f *fetcher.Fetcher, cat config.ShopCategory, shop config.Shop, dumpDir string) []parser.RawProduct {
	if priceAPI == nil {
		return products
	}

	slog.Debug("enriching prices", "products", len(products))

	body, err := buildPriceRequestBody(products, priceAPI, cat)
	if err != nil {
		slog.Error("building price API request", "error", err)
		return products
	}

	data, err := f.Post(priceAPI.URL, body, nil, priceAPI.Headers)
	if err != nil {
		slog.Error("price API fetch failed", "url", priceAPI.URL, "error", err)
		return products
	}

	dumpResponse(dumpDir, shop.Name, cat.Category+"_prices", 0,
		"POST", priceAPI.URL, priceAPI.Headers, body, data)

	priceMap, err := parsePriceResponse(data, priceAPI)
	if err != nil {
		slog.Error("parsing price API response", "error", err)
		return products
	}

	return mergePrices(products, priceMap, priceAPI.IDField)
}

type priceInfo struct {
	price    float64
	oldPrice *float64
}

func buildPriceRequestBody(products []parser.RawProduct, priceAPI *config.PriceAPI, cat config.ShopCategory) (string, error) {
	if priceAPI.BodyTemplate == "" {
		return "", fmt.Errorf("price_api.body_template is required")
	}

	tpl, err := os.ReadFile(priceAPI.BodyTemplate)
	if err != nil {
		return "", fmt.Errorf("loading price API template %s: %w", priceAPI.BodyTemplate, err)
	}

	// Build the articles array from product IDs for the {articles} placeholder.
	var articles []map[string]any
	for _, p := range products {
		id := p.URL // URL field is used to carry the product ID from the search API
		if id == "" {
			continue
		}
		articles = append(articles, map[string]any{
			"articleID":         id,
			"insertCode":        "UO",
			"calculatePrice":    true,
			"checkAvailability": true,
			"findExclusions":    true,
		})
	}

	articlesJSON, err := json.Marshal(articles)
	if err != nil {
		return "", fmt.Errorf("marshaling articles: %w", err)
	}

	body := strings.ReplaceAll(string(tpl), "{articles}", string(articlesJSON))
	return body, nil
}

func parsePriceResponse(data []byte, priceAPI *config.PriceAPI) (map[string]priceInfo, error) {
	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing price response: %w", err)
	}

	productsRaw := walkPath(root, priceAPI.ProductsPath)
	arr, ok := productsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("price products path %q did not resolve to array", priceAPI.ProductsPath)
	}

	result := make(map[string]priceInfo, len(arr))
	for _, item := range arr {
		id := walkString(item, priceAPI.IDPath)
		id = strings.TrimLeft(id, "0")
		if id == "" {
			continue
		}

		price, err := walkFloat(item, priceAPI.PricePath)
		if err != nil {
			continue
		}

		info := priceInfo{price: price}
		if priceAPI.OldPricePath != "" {
			if old, err := walkFloat(item, priceAPI.OldPricePath); err == nil && old > 0 {
				info.oldPrice = &old
			}
		}

		result[id] = info
	}

	return result, nil
}

func mergePrices(products []parser.RawProduct, prices map[string]priceInfo, idField string) []parser.RawProduct {
	var enriched []parser.RawProduct
	for _, p := range products {
		id := p.URL // product ID stored in URL field
		if info, ok := prices[id]; ok {
			p.Price = info.price
			if info.oldPrice != nil {
				p.OldPrice = info.oldPrice
			}
			enriched = append(enriched, p)
		}
	}
	return enriched
}

// walkPath and walkString/walkFloat are reused from the json parser.
// Import them via the parser package's exported functions.
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
	val := walkPath(data, path)
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func walkFloat(data interface{}, path string) (float64, error) {
	val := walkPath(data, path)
	if val == nil {
		return 0, fmt.Errorf("nil")
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected type %T", val)
	}
}
