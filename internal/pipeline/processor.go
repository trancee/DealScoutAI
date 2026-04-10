package pipeline

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/currency"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/fetcher"
	"github.com/trancee/DealScout/internal/jsonpath"
	"github.com/trancee/DealScout/internal/parser"
	"github.com/trancee/DealScout/internal/parser/cleaners"
)

// ShopResult holds the output of processing a single shop.
type ShopResult struct {
	Deals    []deal.Deal
	Products []ProductResult
	Count    int
	Errors   int
}

// ShopProcessor handles the full lifecycle of fetching, parsing, cleaning,
// and evaluating products for a single shop.
type ShopProcessor struct {
	shop      config.Shop
	fetcher   *fetcher.Fetcher
	conv      *currency.Converter
	eval      *deal.Evaluator
	filters   map[string]config.Filter
	seedMode  bool
	cache     *responseCache
	dumpDir   string
	shopClean cleaners.CleanFunc
	urlClean  cleaners.CleanFunc
}

// NewShopProcessor creates a processor for a single shop run.
func NewShopProcessor(shop config.Shop, f *fetcher.Fetcher, conv *currency.Converter, eval *deal.Evaluator, filters map[string]config.Filter, seedMode bool, dumpDir string, cache *responseCache) *ShopProcessor {
	return &ShopProcessor{
		shop:      shop,
		fetcher:   f,
		conv:      conv,
		eval:      eval,
		filters:   filters,
		seedMode:  seedMode,
		cache:     cache,
		dumpDir:   dumpDir,
		shopClean: cleaners.ShopCleaner(shop.Cleaner),
		urlClean:  cleaners.URLCleaner(shop.Cleaner),
	}
}

// Process runs the full pipeline for this shop and returns results.
func (sp *ShopProcessor) Process() ShopResult {
	var result ShopResult

	if err := sp.acquireToken(); err != nil {
		slog.Error("token fetch failed", "shop", sp.shop.Name, "error", err)
		result.Errors = 1
		return result
	}

	for _, cat := range sp.shop.Categories {
		sp.processCategory(cat, &result)
	}

	return result
}

func (sp *ShopProcessor) acquireToken() error {
	if sp.shop.TokenEndpoint == nil {
		return nil
	}
	data, err := sp.fetcher.Get(sp.shop.TokenEndpoint.URL)
	if err != nil {
		return fmt.Errorf("fetching token from %s: %w", sp.shop.TokenEndpoint.URL, err)
	}
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}
	token := jsonpath.String(raw, sp.shop.TokenEndpoint.TokenPath)
	if token == "" {
		return fmt.Errorf("token not found at path %q", sp.shop.TokenEndpoint.TokenPath)
	}
	if sp.shop.Headers == nil {
		sp.shop.Headers = make(map[string]string)
	}
	sp.shop.Headers["Authorization"] = "Bearer " + token
	return nil
}

func (sp *ShopProcessor) processCategory(cat config.ShopCategory, result *ShopResult) {
	cat, priceReplacements := resolvePricePlaceholders(cat, sp.eval.Rules(), sp.shop.PriceBuckets)
	catFilter := buildFilter(cat.Category, sp.filters)

	for urlIdx, catURL := range categoryURLs(cat) {
		cacheKey := fmt.Sprintf("%s_%d", cat.Category, urlIdx)
		fetchCat := cat
		fetchCat.URL = catURL

		for page := range cat.MaxPages {
			data, err := sp.fetchPageData(fetchCat, page, priceReplacements, cacheKey)
			if err != nil {
				slog.Error("fetch failed", "shop", sp.shop.Name, "category", cat.Category, "page", page, "error", err)
				result.Errors++
				break
			}

			rawProducts, err := sp.parseProducts(data, cat)
			if err != nil {
				slog.Error("parse failed", "shop", sp.shop.Name, "category", cat.Category, "error", err)
				result.Errors++
				continue
			}

			for _, p := range rawProducts {
				result.Count++

				if sp.urlClean != nil {
					p.URL = sp.urlClean(p.URL)
				}

				cleaned, priceCHF, oldPrice, skip := transformProduct(p, cat, sp.shopClean, catFilter, sp.conv)
				if skip {
					continue
				}

				pr, d := evaluateProduct(cleaned, priceCHF, oldPrice, p, cat, sp.shop, sp.eval, sp.seedMode)
				result.Products = append(result.Products, pr)
				if d != nil {
					result.Deals = append(result.Deals, *d)
				}
			}
		}
	}
}

func (sp *ShopProcessor) fetchPageData(cat config.ShopCategory, page int, priceReplacements map[string]string, cacheKey string) ([]byte, error) {
	data, cached := sp.cache.get(sp.shop.Name, cacheKey, page)
	if !cached {
		var err error
		data, err = fetchPage(sp.fetcher, sp.shop, cat, page, priceReplacements)
		if err != nil {
			return nil, err
		}
		sp.cache.put(sp.shop.Name, cacheKey, page, data)
	}

	dumpResponse(sp.dumpDir, sp.shop.Name, cacheKey, page,
		fetchMethod(cat), buildRequestURL(cat, page), sp.shop.Headers, fetchBody(cat, page, priceReplacements), data)

	return data, nil
}

func (sp *ShopProcessor) parseProducts(data []byte, cat config.ShopCategory) ([]parser.RawProduct, error) {
	if cat.JSONPCallback != "" {
		data = stripJSONP(data, cat.JSONPCallback)
	}

	rawProducts, err := parser.Parse(cat, data, sp.shop.BaseURL)
	if err != nil {
		return nil, err
	}

	if cat.PriceAPI != nil {
		enricher := NewPriceEnricher(cat.PriceAPI, sp.makeFetchFunc(cat))
		rawProducts = enricher.Enrich(rawProducts)
	}

	parser.ResolveProductURLs(rawProducts, sp.shop.BaseURL, cat.URLTemplate)

	return rawProducts, nil
}

// makeFetchFunc creates a FetchFunc that integrates caching and dumping.
func (sp *ShopProcessor) makeFetchFunc(cat config.ShopCategory) FetchFunc {
	priceCacheKey := cat.Category + "_prices"
	return func(method, url, body string, headers map[string]string) ([]byte, error) {
		data, cached := sp.cache.get(sp.shop.Name, priceCacheKey, 0)
		if cached {
			return data, nil
		}

		var err error
		if method == "POST" {
			data, err = sp.fetcher.Post(url, body, nil, headers)
		} else {
			data, err = sp.fetcher.Get(url, headers)
		}
		if err != nil {
			return nil, err
		}

		dumpResponse(sp.dumpDir, sp.shop.Name, priceCacheKey, 0, method, url, headers, body, data)
		sp.cache.put(sp.shop.Name, priceCacheKey, 0, data)
		return data, nil
	}
}
