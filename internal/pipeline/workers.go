package pipeline

import (
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

func collectDeals(shops []config.Shop, f *fetcher.Fetcher, conv *currency.Converter, eval *deal.Evaluator, filters map[string]config.Filter, seedMode bool, summary *Summary) []deal.Deal {
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

			shopDeals, products, errors := processShop(shop, f, conv, eval, filters, seedMode)

			mu.Lock()
			deals = append(deals, shopDeals...)
			summary.ProductsChecked += products
			summary.Errors += errors
			mu.Unlock()
		}(shop)
	}

	wg.Wait()
	return deals
}

func processShop(shop config.Shop, f *fetcher.Fetcher, conv *currency.Converter, eval *deal.Evaluator, filters map[string]config.Filter, seedMode bool) ([]deal.Deal, int, int) {
	var (
		deals    []deal.Deal
		products int
		errors   int
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

			rawProducts, err := parser.Parse(cat, data, shop.BaseURL)
			if err != nil {
				slog.Error("parse failed", "shop", shop.Name, "category", cat.Category, "error", err)
				errors++
				continue
			}

			if cat.PriceAPI != nil {
				rawProducts = enrichPrices(rawProducts, cat.PriceAPI, f, cat, shop)
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
				if !seedMode && result.Deal != nil {
					deals = append(deals, *result.Deal)
				}
			}
		}
	}

	return deals, products, errors
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
