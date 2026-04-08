package cleaners

import "strings"

// CleanFunc transforms a product name (strip artifacts, normalize).
type CleanFunc func(name string) string

// FilterFunc returns true if a product name should be SKIPPED.
type FilterFunc func(name string) bool

// ShopCleaner returns a cleaning function for the given shop, or nil if none.
func ShopCleaner(shopName string) CleanFunc {
	switch strings.ToLower(shopName) {
	case "galaxus":
		return cleanGalaxus
	case "amazon":
		return cleanAmazon
	case "ackermann":
		return cleanAckermann
	case "brack":
		return cleanBrack
	case "conrad":
		return cleanConrad
	case "foletti":
		return cleanFoletti
	case "interdiscount":
		return cleanInterdiscount
	case "mediamarkt":
		return cleanMediamarkt
	default:
		return nil
	}
}

// CategoryCleaner returns a cleaning function for the given category, or nil if none.
func CategoryCleaner(category string) CleanFunc {
	return nil
}
