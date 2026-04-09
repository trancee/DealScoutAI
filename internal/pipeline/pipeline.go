package pipeline

import (
	"log/slog"
	"time"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/currency"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/fetcher"
	"github.com/trancee/DealScout/internal/storage"
)

// Options configures a pipeline run.
type Options struct {
	Seed     bool
	DryRun   bool
	ShopName string
	DumpDir  string
}

// ProductResult tracks a product that was evaluated during the run.
type ProductResult struct {
	Name     string
	Shop     string
	Price    float64
	Discount float64
	IsDeal   bool
	Reason   string
}

// Summary holds run statistics.
type Summary struct {
	ProductsChecked   int
	DealsFound        int
	NotificationsSent int
	Errors            int
	Duration          time.Duration
	Products          []ProductResult
}

// Run executes the full pipeline.
func Run(cfg *config.Config, db *storage.Database, opts Options) Summary {
	start := time.Now()
	var summary Summary

	conv := currency.New(db, cfg.Settings.ExchangeRateProvider, cfg.Settings.BaseCurrency, cfg.Settings.ExchangeRateCacheTTLHours)
	if err := conv.RefreshRates(); err != nil {
		slog.Error("failed to refresh exchange rates", "error", err)
		summary.Errors++
	}

	eval := deal.NewEvaluator(db, cfg.DealRules, cfg.Settings.NotificationCooldownHours)
	f := fetcher.New(cfg.Settings.FetchDelaySeconds, cfg.Settings.MaxRetries)

	shops := filterShops(cfg.Shops, opts.ShopName)
	deals := collectDeals(shops, f, conv, eval, cfg.Filters, opts.Seed, opts.DumpDir, &summary)

	summary.DealsFound = len(deals)
	sendNotifications(deals, cfg, db, opts, &summary)
	pruneHistory(db, cfg.Settings.PriceHistoryRetentionDays, &summary)

	summary.Duration = time.Since(start)
	logProducts(summary.Products)
	logSummary(summary)

	return summary
}
