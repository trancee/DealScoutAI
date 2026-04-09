package pipeline

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/fetcher"
	"github.com/trancee/DealScout/internal/jsonpath"
	"github.com/trancee/DealScout/internal/parser"
)

// enrichPrices fetches prices from a secondary API and merges them into products.
// Products without a price match are returned with Price=0 (filtered by sanity bounds later).
func enrichPrices(products []parser.RawProduct, priceAPI *config.PriceAPI, f *fetcher.Fetcher, cat config.ShopCategory, shop config.Shop, dumpDir string, cache *responseCache) []parser.RawProduct {
	if priceAPI == nil {
		return products
	}

	slog.Debug("enriching prices", "products", len(products))

	var ids []string
	for _, p := range products {
		if p.URL != "" {
			ids = append(ids, p.URL)
		}
	}
	if len(ids) == 0 {
		return products
	}

	priceCacheKey := cat.Category + "_prices"
	data, cached := cache.get(shop.Name, priceCacheKey, 0)
	if !cached {
		var err error
		if priceAPI.BodyTemplate != "" {
			body, buildErr := buildPriceRequestBody(products, priceAPI)
			if buildErr != nil {
				slog.Error("building price API request", "error", buildErr)
				return products
			}
			data, err = f.Post(priceAPI.URL, body, nil, priceAPI.Headers)
			if err != nil {
				slog.Error("price API fetch failed", "url", priceAPI.URL, "error", err)
				return products
			}
			dumpResponse(dumpDir, shop.Name, priceCacheKey, 0,
				"POST", priceAPI.URL, priceAPI.Headers, body, data)
		} else {
			url := strings.ReplaceAll(priceAPI.URL, "{ids}", strings.Join(ids, ","))
			data, err = f.Get(url, priceAPI.Headers)
			if err != nil {
				slog.Error("price API fetch failed", "url", url, "error", err)
				return products
			}
			dumpResponse(dumpDir, shop.Name, priceCacheKey, 0,
				"GET", url, priceAPI.Headers, "", data)
		}
		cache.put(shop.Name, priceCacheKey, 0, data)
	}

	priceMap, err := parsePriceResponse(data, priceAPI)
	if err != nil {
		slog.Error("parsing price API response", "error", err)
		return products
	}

	return mergePrices(products, priceMap)
}

type priceInfo struct {
	price    float64
	oldPrice *float64
	title    string
	imageURL string
}

func buildPriceRequestBody(products []parser.RawProduct, priceAPI *config.PriceAPI) (string, error) {
	tpl, err := os.ReadFile(priceAPI.BodyTemplate)
	if err != nil {
		return "", fmt.Errorf("loading price API template %s: %w", priceAPI.BodyTemplate, err)
	}

	var ids []string
	for _, p := range products {
		if p.URL != "" {
			ids = append(ids, p.URL)
		}
	}

	// Build article objects for the {articles} placeholder (Conrad-style).
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

func parsePriceResponse(data []byte, priceAPI *config.PriceAPI) (map[string]priceInfo, error) {
	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing price response: %w", err)
	}

	var items []interface{}
	isMap := false

	if priceAPI.ProductsPath != "" {
		raw := jsonpath.Walk(root, priceAPI.ProductsPath)
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
			return nil, fmt.Errorf("price products path %q resolved to %T", priceAPI.ProductsPath, raw)
		}
	} else if m, ok := root.(map[string]interface{}); ok {
		isMap = true
		for _, val := range m {
			if val != nil {
				items = append(items, val)
			}
		}
	}

	result := make(map[string]priceInfo, len(items))
	for _, item := range items {
		var id string
		if priceAPI.IDPath != "" {
			id = jsonpath.String(item, priceAPI.IDPath)
			id = strings.TrimLeft(id, "0")
		} else if isMap {
			id = jsonpath.String(item, "sku")
		}
		if id == "" {
			continue
		}

		price, err := jsonpath.Float(item, priceAPI.PricePath)
		if err != nil {
			continue
		}

		info := priceInfo{price: price}
		if priceAPI.OldPricePath != "" {
			if old, err := jsonpath.Float(item, priceAPI.OldPricePath); err == nil && old > 0 {
				info.oldPrice = &old
			}
		}
		if priceAPI.TitlePath != "" {
			info.title = jsonpath.String(item, priceAPI.TitlePath)
		}
		if priceAPI.ImagePath != "" {
			info.imageURL = jsonpath.String(item, priceAPI.ImagePath)
		}

		result[id] = info
	}

	return result, nil
}

func mergePrices(products []parser.RawProduct, prices map[string]priceInfo) []parser.RawProduct {
	var enriched []parser.RawProduct
	for _, p := range products {
		if info, ok := prices[p.URL]; ok {
			p.Price = info.price
			if info.oldPrice != nil {
				p.OldPrice = info.oldPrice
			}
			if info.title != "" {
				p.Title = info.title
			}
			if info.imageURL != "" {
				p.ImageURL = info.imageURL
			}
			enriched = append(enriched, p)
		}
	}
	return enriched
}
