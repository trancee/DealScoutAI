package deal

import (
	"log/slog"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/storage"
)

// Deal represents a detected deal ready for notification.
type Deal struct {
	ProductName string
	Category    string
	Shop        string
	Price       float64
	OldPrice    *float64
	DiscountPct float64
	LastDBPrice float64
	URL         string
	ImageURL    string
	IsFirstSeen bool
}

// Evaluator applies deal rules and deduplication logic.
type Evaluator struct {
	db            *storage.Database
	rules         map[string]config.DealRule
	cooldownHours int
}

// NewEvaluator creates a deal Evaluator.
func NewEvaluator(db *storage.Database, rules map[string]config.DealRule, cooldownHours int) *Evaluator {
	return &Evaluator{db: db, rules: rules, cooldownHours: cooldownHours}
}

// Result represents the outcome of evaluating a product.
type Result struct {
	Deal    *Deal
	Skipped bool
	Reason  string
}

// Evaluate checks if a product qualifies as a deal.
// It stores the product and price history regardless of deal outcome.
func (e *Evaluator) Evaluate(cleanedName, category, shop string, priceCHF float64, oldPrice *float64, url, imageURL string) Result {
	rule, ok := e.rules[category]
	if !ok {
		return Result{Skipped: true, Reason: "no_rule"}
	}

	if !inSanityBounds(priceCHF, rule) {
		slog.Warn("price outside sanity bounds", "product", cleanedName, "price", priceCHF)
		return Result{Skipped: true, Reason: "sanity"}
	}

	productID, isNew, err := e.db.UpsertProduct(cleanedName, category)
	if err != nil {
		slog.Error("upsert product failed", "product", cleanedName, "error", err)
		return Result{Skipped: true, Reason: "db_error"}
	}

	lastPrice, hasHistory := e.lastPriceForShop(productID, shop)

	if err := e.db.AppendPriceHistory(productID, shop, priceCHF, "CHF", oldPrice, url); err != nil {
		slog.Error("append price history failed", "product", cleanedName, "error", err)
	}

	if !inDealRange(priceCHF, rule) {
		return Result{Skipped: true, Reason: "out_of_range"}
	}

	isFirstSeen := isNew || !hasHistory

	if isFirstSeen {
		return e.checkCooldownAndReturn(productID, cleanedName, category, shop, priceCHF, oldPrice, 0, 0, url, imageURL, true)
	}

	discountPct := ((lastPrice - priceCHF) / lastPrice) * 100
	if discountPct < rule.MinDiscountPct {
		return Result{Skipped: true, Reason: "insufficient_discount"}
	}

	return e.checkCooldownAndReturn(productID, cleanedName, category, shop, priceCHF, oldPrice, discountPct, lastPrice, url, imageURL, false)
}

func (e *Evaluator) checkCooldownAndReturn(productID int64, name, category, shop string, price float64, oldPrice *float64, discountPct, lastDBPrice float64, url, imageURL string, firstSeen bool) Result {
	notified, err := e.db.WasNotifiedWithinCooldown(productID, shop, e.cooldownHours)
	if err != nil {
		slog.Error("cooldown check failed", "product", name, "error", err)
	}
	if notified {
		return Result{Skipped: true, Reason: "cooldown"}
	}

	return Result{
		Deal: &Deal{
			ProductName: name,
			Category:    category,
			Shop:        shop,
			Price:       price,
			OldPrice:    oldPrice,
			DiscountPct: discountPct,
			LastDBPrice: lastDBPrice,
			URL:         url,
			ImageURL:    imageURL,
			IsFirstSeen: firstSeen,
		},
	}
}

func (e *Evaluator) lastPriceForShop(productID int64, shop string) (float64, bool) {
	price, found, err := e.db.LastPrice(productID, shop)
	if err != nil {
		return 0, false
	}
	return price, found
}

func inSanityBounds(price float64, rule config.DealRule) bool {
	lower := rule.MinPrice * 0.1
	upper := rule.MaxPrice * 10
	return price >= lower && price <= upper
}

func inDealRange(price float64, rule config.DealRule) bool {
	return price >= rule.MinPrice && price <= rule.MaxPrice
}
