package pipeline

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/notifier"
	"github.com/trancee/DealScout/internal/storage"
)

func sendNotifications(deals []deal.Deal, cfg *config.Config, db *storage.Database, opts Options, summary *Summary) {
	if opts.Seed || opts.DryRun {
		if opts.DryRun {
			for _, d := range deals {
				slog.Info("deal (dry-run)", "product", d.ProductName, "shop", d.Shop, "price", d.Price, "discount", d.DiscountPct)
			}
		}
		return
	}

	n := notifier.New(cfg.Secrets.TelegramBotToken, cfg.Secrets.TelegramChannel, cfg.Settings.TelegramTopics)
	for _, d := range deals {
		if err := n.Send(d); err != nil {
			slog.Error("notification failed", "product", d.ProductName, "error", err)
			summary.Errors++
			continue
		}
		id, _, _ := db.UpsertProduct(d.ProductName, d.Category)
		if err := db.RecordNotification(id, d.Shop, d.Price); err != nil {
			slog.Error("record notification failed", "product", d.ProductName, "error", err)
		}
		summary.NotificationsSent++
	}
}

func pruneHistory(db *storage.Database, retentionDays int, summary *Summary) {
	deleted, err := db.PruneOldPriceHistory(retentionDays)
	if err != nil {
		slog.Error("prune failed", "error", err)
		summary.Errors++
		return
	}
	if deleted > 0 {
		slog.Info("pruned old price history", "deleted", deleted)
	}
}

func logProducts(products []ProductResult) {
	if len(products) == 0 {
		return
	}

	sort.Slice(products, func(i, j int) bool {
		if products[i].Shop != products[j].Shop {
			return products[i].Shop < products[j].Shop
		}
		return products[i].Price < products[j].Price
	})

	fmt.Println()
	fmt.Println("Products found:")
	fmt.Printf("%-4s %-50s %-15s %10s %8s  %-25s %s\n", "DEAL", "PRODUCT", "SHOP", "PRICE", "DROP", "REASON", "URL")
	fmt.Printf("%-4s %-50s %-15s %10s %8s  %-25s %s\n", "----", "-------", "----", "-----", "----", "------", "---")

	for _, p := range products {
		marker := "    "
		if p.IsDeal {
			marker = " 🔥 "
		}

		discount := ""
		if p.Discount > 0 {
			discount = fmt.Sprintf("-%.0f%%", p.Discount)
		}

		reason := ""
		if !p.IsDeal && p.Reason != "" {
			reason = p.Reason
		}

		name := p.Name
		if len(name) > 50 {
			name = name[:47] + "..."
		}
		shop := p.Shop
		if len(shop) > 15 {
			shop = shop[:12] + "..."
		}

		fmt.Printf("%s %-50s %-15s %10.2f %8s  %-25s %s\n", marker, name, shop, p.Price, discount, reason, p.URL)
	}
	fmt.Println()
}

func logSummary(s Summary) {
	slog.Info("run complete",
		"products_checked", s.ProductsChecked,
		"deals_found", s.DealsFound,
		"notifications_sent", s.NotificationsSent,
		"errors", s.Errors,
		"duration", s.Duration.Round(time.Millisecond),
	)
}
