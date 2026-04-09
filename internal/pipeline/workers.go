package pipeline

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/currency"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/fetcher"
	"github.com/trancee/DealScout/internal/parser"
	"github.com/trancee/DealScout/internal/parser/cleaners"
)

func collectDeals(shops []config.Shop, f *fetcher.Fetcher, conv *currency.Converter, eval *deal.Evaluator, filters map[string]config.Filter, seedMode bool, dumpDir string, summary *Summary) []deal.Deal {
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
			shopDeals, shopProducts, products, errors := processShop(shop, f, conv, eval, filters, seedMode, dumpDir)

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

func processShop(shop config.Shop, f *fetcher.Fetcher, conv *currency.Converter, eval *deal.Evaluator, filters map[string]config.Filter, seedMode bool, dumpDir string) ([]deal.Deal, []ProductResult, int, int) {
	var (
		deals     []deal.Deal
		evaluated []ProductResult
		products  int
		errors    int
	)

	shopClean := cleaners.ShopCleaner(shop.Cleaner)

	for _, cat := range shop.Categories {
		catFilter := buildFilter(cat.Category, filters)

		for page := range cat.MaxPages {
			data, err := fetchPage(f, shop, cat, page)
			if err != nil {
				slog.Error("fetch failed", "shop", shop.Name, "category", cat.Category, "page", page, "error", err)
				errors++
				break
			}

			dumpResponse(dumpDir, shop.Name, cat.Category, page,
				fetchMethod(cat), buildRequestURL(cat, page), shop.Headers, fetchBody(cat, page), data)

			if cat.JSONPCallback != "" {
				data = stripJSONP(data, cat.JSONPCallback)
			}

			rawProducts, err := parser.Parse(cat, data, shop.BaseURL)
			if err != nil {
				slog.Error("parse failed", "shop", shop.Name, "category", cat.Category, "error", err)
				errors++
				continue
			}

			if cat.PriceAPI != nil {
				rawProducts = enrichPrices(rawProducts, cat.PriceAPI, f, cat, shop, dumpDir)
			}

			for _, p := range rawProducts {
				products++

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
					continue
				}

				priceCHF, err := conv.Convert(p.Price, cat.Currency)
				if err != nil {
					slog.Warn("currency conversion failed", "product", cleaned, "error", err)
					errors++
					continue
				}

				result := eval.Evaluate(cleaned, cat.Category, shop.Name, priceCHF, p.OldPrice, p.URL, p.ImageURL)

				pr := ProductResult{
					Name:   cleaned,
					Shop:   shop.Name,
					Price:  priceCHF,
					Reason: result.Reason,
				}

				if !seedMode && result.Deal != nil {
					pr.IsDeal = true
					pr.Discount = result.Deal.DiscountPct
					deals = append(deals, *result.Deal)
				}

				evaluated = append(evaluated, pr)
			}
		}
	}

	return deals, evaluated, products, errors
}

func fetchPage(f *fetcher.Fetcher, shop config.Shop, cat config.ShopCategory, page int) ([]byte, error) {
	switch cat.Pagination.Type {
	case "offset":
		offset := page * cat.Pagination.PerPage
		tpl, err := os.ReadFile(cat.BodyTemplate)
		if err != nil {
			return nil, fmt.Errorf("loading template %s: %w", cat.BodyTemplate, err)
		}
		replacements := map[string]string{"{offset}": fmt.Sprintf("%d", offset)}
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
			return f.Post(cat.URL, string(tpl), nil, shop.Headers)
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

func fetchBody(cat config.ShopCategory, page int) string {
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
	return body
}
