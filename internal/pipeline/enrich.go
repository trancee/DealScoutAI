package pipeline

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/jsonpath"
	"github.com/trancee/DealScout/internal/parser"
)

// FetchFunc abstracts HTTP fetching for testability.
// GET: body is empty. POST: body is the request body.
type FetchFunc func(method, url, body string, headers map[string]string) ([]byte, error)

// PriceEnricher fetches prices from a secondary API and merges them into products.
type PriceEnricher struct {
	api   *config.PriceAPI
	fetch FetchFunc
}

// NewPriceEnricher creates an enricher for the given price API config.
func NewPriceEnricher(api *config.PriceAPI, fetch FetchFunc) *PriceEnricher {
	return &PriceEnricher{api: api, fetch: fetch}
}

// Enrich fetches prices and merges them into products.
// Products without a price match are dropped.
func (pe *PriceEnricher) Enrich(products []parser.RawProduct) []parser.RawProduct {
	if pe.api == nil || len(products) == 0 {
		return products
	}

	slog.Debug("enriching prices", "products", len(products))

	data, err := pe.fetchPriceData(products)
	if err != nil {
		slog.Error("price API fetch failed", "error", err)
		return products
	}

	priceMap, err := ParsePriceResponse(data, pe.api)
	if err != nil {
		slog.Error("parsing price API response", "error", err)
		return products
	}

	return MergePrices(products, priceMap)
}

func (pe *PriceEnricher) fetchPriceData(products []parser.RawProduct) ([]byte, error) {
	if pe.api.BodyTemplate != "" {
		body, err := BuildPriceRequestBody(products, pe.api)
		if err != nil {
			return nil, err
		}
		return pe.fetch("POST", pe.api.URL, body, pe.api.Headers)
	}

	ids := collectIDs(products)
	url := strings.ReplaceAll(pe.api.URL, "{ids}", strings.Join(ids, ","))
	return pe.fetch("GET", url, "", pe.api.Headers)
}

// PriceInfo holds enrichment data for a single product.
type PriceInfo struct {
	Price    float64
	OldPrice *float64
	Title    string
	ImageURL string
}

// BuildPriceRequestBody constructs the POST body for a price API call.
func BuildPriceRequestBody(products []parser.RawProduct, api *config.PriceAPI) (string, error) {
	tpl, err := os.ReadFile(api.BodyTemplate)
	if err != nil {
		return "", fmt.Errorf("loading price API template %s: %w", api.BodyTemplate, err)
	}

	ids := collectIDs(products)

	var articles []map[string]any
	for _, id := range ids {
		articles = append(articles, map[string]any{
			"articleID":         id,
			"insertCode":        "UO",
			"calculatePrice":    true,
			"checkAvailability": true,
			"findExclusions":    true,
		})
	}
	articlesJSON, _ := json.Marshal(articles)

	body := string(tpl)
	body = strings.ReplaceAll(body, "{ids}", strings.Join(ids, ","))
	body = strings.ReplaceAll(body, "{articles}", string(articlesJSON))
	return body, nil
}

// ParsePriceResponse extracts price info from a secondary API response.
func ParsePriceResponse(data []byte, api *config.PriceAPI) (map[string]PriceInfo, error) {
	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing price response: %w", err)
	}

	var items []interface{}
	isMap := false

	if api.ProductsPath != "" {
		raw := jsonpath.Walk(root, api.ProductsPath)
		switch v := raw.(type) {
		case []interface{}:
			items = v
		case map[string]interface{}:
			isMap = true
			for _, val := range v {
				if val != nil {
					items = append(items, val)
				}
			}
		default:
			return nil, fmt.Errorf("price products path %q resolved to %T", api.ProductsPath, raw)
		}
	} else if m, ok := root.(map[string]interface{}); ok {
		isMap = true
		for _, val := range m {
			if val != nil {
				items = append(items, val)
			}
		}
	}

	result := make(map[string]PriceInfo, len(items))
	for _, item := range items {
		var id string
		if api.IDPath != "" {
			id = jsonpath.String(item, api.IDPath)
			id = strings.TrimLeft(id, "0")
		} else if isMap {
			id = jsonpath.String(item, "sku")
		}
		if id == "" {
			continue
		}

		price, err := jsonpath.Float(item, api.PricePath)
		if err != nil {
			continue
		}

		info := PriceInfo{Price: price}
		if api.OldPricePath != "" {
			if old, err := jsonpath.Float(item, api.OldPricePath); err == nil && old > 0 {
				info.OldPrice = &old
			}
		}
		if api.TitlePath != "" {
			info.Title = jsonpath.String(item, api.TitlePath)
		}
		if api.ImagePath != "" {
			info.ImageURL = jsonpath.String(item, api.ImagePath)
		}

		result[id] = info
	}

	return result, nil
}

// MergePrices joins price data back into products by URL/ID.
// Products without a match are dropped.
func MergePrices(products []parser.RawProduct, prices map[string]PriceInfo) []parser.RawProduct {
	var enriched []parser.RawProduct
	for _, p := range products {
		if info, ok := prices[p.URL]; ok {
			p.Price = info.Price
			if info.OldPrice != nil {
				p.OldPrice = info.OldPrice
			}
			if info.Title != "" {
				p.Title = info.Title
			}
			if info.ImageURL != "" {
				p.ImageURL = info.ImageURL
			}
			enriched = append(enriched, p)
		}
	}
	return enriched
}

func collectIDs(products []parser.RawProduct) []string {
	var ids []string
	for _, p := range products {
		if p.URL != "" {
			ids = append(ids, p.URL)
		}
	}
	return ids
}
