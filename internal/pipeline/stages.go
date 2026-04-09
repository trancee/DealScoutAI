package pipeline

import (
	"log/slog"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/currency"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/fetcher"
	"github.com/trancee/DealScout/internal/parser"
	"github.com/trancee/DealScout/internal/parser/cleaners"
)

// fetchPageData fetches a page from cache or network, stores in cache, and dumps.
func fetchPageData(f *fetcher.Fetcher, shop config.Shop, cat config.ShopCategory, page int, priceReplacements map[string]string, cacheKey string, cache *responseCache, dumpDir string) ([]byte, error) {
	data, cached := cache.get(shop.Name, cacheKey, page)
	if !cached {
		var err error
		data, err = fetchPage(f, shop, cat, page, priceReplacements)
		if err != nil {
			return nil, err
		}
		cache.put(shop.Name, cacheKey, page, data)
	}

	dumpResponse(dumpDir, shop.Name, cacheKey, page,
		fetchMethod(cat), buildRequestURL(cat, page), shop.Headers, fetchBody(cat, page, priceReplacements), data)

	return data, nil
}

// parsePageProducts parses raw response data into products, handling JSONP, enrichment, and URL resolution.
func parsePageProducts(data []byte, cat config.ShopCategory, shop config.Shop, f *fetcher.Fetcher, dumpDir string, cache *responseCache) ([]parser.RawProduct, error) {
	if cat.JSONPCallback != "" {
		data = stripJSONP(data, cat.JSONPCallback)
	}

	rawProducts, err := parser.Parse(cat, data, shop.BaseURL)
	if err != nil {
		return nil, err
	}

	if cat.PriceAPI != nil {
		rawProducts = enrichPrices(rawProducts, cat.PriceAPI, f, cat, shop, dumpDir, cache)
	}

	parser.ResolveProductURLs(rawProducts, shop.BaseURL, cat.URLTemplate)

	return rawProducts, nil
}

// transformProduct cleans, normalizes, filters, divides price, and converts currency.
// Returns the cleaned name, CHF price, and whether the product should be skipped.
func transformProduct(p parser.RawProduct, cat config.ShopCategory, shopClean cleaners.CleanFunc, catFilter cleaners.FilterFunc, conv *currency.Converter) (string, float64, bool) {
	if cat.PriceDivisor > 0 {
		p.Price /= cat.PriceDivisor
		if p.OldPrice != nil {
			divided := *p.OldPrice / cat.PriceDivisor
			p.OldPrice = &divided
		}
	}

	cleaned := p.Title
	if shopClean != nil {
		cleaned = shopClean(cleaned)
	}
	cleaned = cleaners.NormalizeName(cleaned)

	if catFilter != nil && catFilter(cleaned) {
		return "", 0, true
	}

	priceCHF, err := conv.Convert(p.Price, cat.Currency)
	if err != nil {
		slog.Warn("currency conversion failed", "product", cleaned, "error", err)
		return "", 0, true
	}

	return cleaned, priceCHF, false
}

// evaluateProduct runs deal evaluation and builds a ProductResult.
func evaluateProduct(cleaned string, priceCHF float64, p parser.RawProduct, cat config.ShopCategory, shop config.Shop, eval *deal.Evaluator, seedMode bool) (ProductResult, *deal.Deal) {
	result := eval.Evaluate(cleaned, cat.Category, shop.Name, priceCHF, p.OldPrice, p.URL, p.ImageURL)

	pr := ProductResult{
		Name:   cleaned,
		Shop:   shop.Name,
		Price:  priceCHF,
		URL:    p.URL,
		Reason: result.Reason,
	}

	if !seedMode && result.Deal != nil {
		pr.IsDeal = true
		pr.Discount = result.Deal.DiscountPct
		return pr, result.Deal
	}

	return pr, nil
}
