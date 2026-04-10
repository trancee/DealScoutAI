package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/currency"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/fetcher"
	"github.com/trancee/DealScout/internal/jsonpath"
	"github.com/trancee/DealScout/internal/parser/cleaners"
)

func collectDeals(shops []config.Shop, f *fetcher.Fetcher, conv *currency.Converter, eval *deal.Evaluator, filters map[string]config.Filter, seedMode bool, dumpDir string, cache *responseCache, summary *Summary) []deal.Deal {
	var (
		mu    sync.Mutex
		deals []deal.Deal
		wg    sync.WaitGroup
		sem   = make(chan struct{}, 5) // default concurrency
	)

	for _, shop := range shops {
		wg.Add(1)
		sem <- struct{}{}

		go func(shop config.Shop) {
			defer wg.Done()
			defer func() { <-sem }()

			clearShopDumpDir(dumpDir, shop.Name)
			shopDeals, shopProducts, products, errors := processShop(shop, f, conv, eval, filters, seedMode, dumpDir, cache)

			mu.Lock()
			deals = append(deals, shopDeals...)
			summary.Products = append(summary.Products, shopProducts...)
			summary.ProductsChecked += products
			summary.Errors += errors
			mu.Unlock()
		}(shop)
	}

	wg.Wait()
	return deals
}

func processShop(shop config.Shop, f *fetcher.Fetcher, conv *currency.Converter, eval *deal.Evaluator, filters map[string]config.Filter, seedMode bool, dumpDir string, cache *responseCache) ([]deal.Deal, []ProductResult, int, int) {
	var (
		deals     []deal.Deal
		evaluated []ProductResult
		products  int
		errors    int
	)

	shopClean := cleaners.ShopCleaner(shop.Cleaner)
	urlClean := cleaners.URLCleaner(shop.Cleaner)

	if shop.TokenEndpoint != nil {
		token, err := fetchBearerToken(f, shop.TokenEndpoint)
		if err != nil {
			slog.Error("token fetch failed", "shop", shop.Name, "error", err)
			return nil, nil, 0, 1
		}
		if shop.Headers == nil {
			shop.Headers = make(map[string]string)
		}
		shop.Headers["Authorization"] = "Bearer " + token
	}

	for _, cat := range shop.Categories {
		cat, priceReplacements := resolvePricePlaceholders(cat, eval.Rules(), shop.PriceBuckets)
		catFilter := buildFilter(cat.Category, filters)

		for urlIdx, catURL := range categoryURLs(cat) {
			cacheKey := fmt.Sprintf("%s_%d", cat.Category, urlIdx)
			fetchCat := cat
			fetchCat.URL = catURL

			for page := range cat.MaxPages {
				data, err := fetchPageData(f, shop, fetchCat, page, priceReplacements, cacheKey, cache, dumpDir)
				if err != nil {
					slog.Error("fetch failed", "shop", shop.Name, "category", cat.Category, "page", page, "error", err)
					errors++
					break
				}

				rawProducts, err := parsePageProducts(data, cat, shop, f, dumpDir, cache)
				if err != nil {
					slog.Error("parse failed", "shop", shop.Name, "category", cat.Category, "error", err)
					errors++
					continue
				}

				for _, p := range rawProducts {
					products++

					if urlClean != nil {
						p.URL = urlClean(p.URL)
					}

					cleaned, priceCHF, oldPrice, skip := transformProduct(p, cat, shopClean, catFilter, conv)
					if skip {
						continue
					}

					pr, d := evaluateProduct(cleaned, priceCHF, oldPrice, p, cat, shop, eval, seedMode)
					evaluated = append(evaluated, pr)
					if d != nil {
						deals = append(deals, *d)
					}
				}
			}
		}
	}

	return deals, evaluated, products, errors
}

func fetchPage(f *fetcher.Fetcher, shop config.Shop, cat config.ShopCategory, page int, priceReplacements map[string]string) ([]byte, error) {
	switch cat.Pagination.Type {
	case "offset":
		offset := page * cat.Pagination.PerPage
		tpl, err := os.ReadFile(cat.BodyTemplate)
		if err != nil {
			return nil, fmt.Errorf("loading template %s: %w", cat.BodyTemplate, err)
		}
		replacements := map[string]string{"{offset}": fmt.Sprintf("%d", offset)}
		for k, v := range priceReplacements {
			replacements[k] = v
		}
		return f.Post(cat.URL, string(tpl), replacements, shop.Headers)
	case "page_param":
		pageNum := cat.Pagination.Start + page
		url := strings.ReplaceAll(cat.URL, "{page}", fmt.Sprintf("%d", pageNum))
		return f.Get(url, shop.Headers)
	default:
		if cat.BodyTemplate != "" {
			tpl, err := os.ReadFile(cat.BodyTemplate)
			if err != nil {
				return nil, fmt.Errorf("loading template %s: %w", cat.BodyTemplate, err)
			}
			return f.Post(cat.URL, string(tpl), priceReplacements, shop.Headers)
		}
		return f.Get(cat.URL, shop.Headers)
	}
}

func buildFilter(category string, filters map[string]config.Filter) cleaners.FilterFunc {
	f, ok := filters[category]
	if !ok {
		return nil
	}
	return cleaners.NewFilter(f)
}

// fetchBearerToken calls a token endpoint and extracts a token string via a JSON path.
func fetchBearerToken(f *fetcher.Fetcher, ep *config.TokenEndpoint) (string, error) {
	data, err := f.Get(ep.URL)
	if err != nil {
		return "", fmt.Errorf("fetching token from %s: %w", ep.URL, err)
	}
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}
	token := jsonpath.String(raw, ep.TokenPath)
	if token == "" {
		return "", fmt.Errorf("token not found at path %q", ep.TokenPath)
	}
	return token, nil
}

func stripJSONP(data []byte, callback string) []byte {
	prefix := []byte(callback + "(")
	if !bytes.HasPrefix(data, prefix) {
		return data
	}
	data = data[len(prefix):]
	// Strip trailing ");", ")", or ")\n"
	data = bytes.TrimRight(data, ";\n\r ")
	if len(data) > 0 && data[len(data)-1] == ')' {
		data = data[:len(data)-1]
	}
	return data
}

func categoryURLs(cat config.ShopCategory) []string {
	if len(cat.URLs) > 0 {
		return cat.URLs
	}
	if cat.URL != "" {
		return []string{cat.URL}
	}
	return nil
}

func filterShops(shops []config.Shop, name string) []config.Shop {
	if name == "" {
		return shops
	}
	for _, s := range shops {
		if s.Name == name {
			return []config.Shop{s}
		}
	}
	return nil
}

func fetchMethod(cat config.ShopCategory) string {
	if cat.BodyTemplate != "" || cat.Pagination.Type == "offset" {
		return "POST"
	}
	return "GET"
}

func fetchBody(cat config.ShopCategory, page int, priceReplacements map[string]string) string {
	if cat.BodyTemplate == "" {
		return ""
	}
	tpl, err := os.ReadFile(cat.BodyTemplate)
	if err != nil {
		return ""
	}
	body := string(tpl)
	if cat.Pagination.Type == "offset" {
		offset := page * cat.Pagination.PerPage
		body = strings.ReplaceAll(body, "{offset}", fmt.Sprintf("%d", offset))
	}
	for k, v := range priceReplacements {
		body = strings.ReplaceAll(body, k, v)
	}
	return body
}
