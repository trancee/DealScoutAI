package cleaners

import "strings"

// CleanFunc transforms a product name (strip artifacts, normalize).
type CleanFunc func(name string) string

// FilterFunc returns true if a product name should be SKIPPED.
type FilterFunc func(name string) bool

var shopCleaners = map[string]CleanFunc{
	"galaxus":        cleanGalaxus,
	"amazon":         cleanAmazon,
	"ackermann":      cleanAckermann,
	"brack":          cleanBrack,
	"conrad":         cleanConrad,
	"foletti":        cleanFoletti,
	"interdiscount":  cleanInterdiscount,
	"mediamarkt":     cleanMediamarkt,
	"mobilezone":     cleanMobilezone,
	"orderflow":      cleanOrderflow,
	"alltron":        cleanAlltron,
	"cashconverters": cleanCashConverters,
	"hopcash":        cleanHopCash,
}

// ShopCleaner returns a cleaning function for the given shop, or nil if none.
func ShopCleaner(shopName string) CleanFunc {
	return shopCleaners[strings.ToLower(shopName)]
}

// CategoryCleaner returns a cleaning function for the given category, or nil if none.
func CategoryCleaner(category string) CleanFunc {
	return nil
}
