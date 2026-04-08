package cleaners_test

import (
	"testing"

	"github.com/trancee/DealScout/internal/config"
	"github.com/trancee/DealScout/internal/parser/cleaners"
)

func TestGalaxusShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("galaxus")
	if clean == nil {
		t.Fatal("ShopCleaner(galaxus) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Samsung Galaxy A15 128 GB Black", "Samsung Galaxy A15 128 GB Black"},
		{"Samsung Galaxy A15 (128 GB, Black, 6.50\", Dual SIM, 50 Mpx, 4G)", "Samsung Galaxy A15 128 GB Black"},
		{"Apple iPhone 16 Pro (256 GB, Space Black, 6.30\", Single SIM, 48 Mpx, 5G)", "Apple iPhone 16 Pro 256 GB Space Black"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := clean(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAmazonShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("amazon")
	if clean == nil {
		t.Fatal("ShopCleaner(amazon) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Samsung Galaxy A15, Android Smartphone, 6.5 Zoll, 128 GB Speicher", "Samsung Galaxy A15"},
		{"Apple iPhone 16 Pro Max 256GB - Space Black", "Apple iPhone 16 Pro Max 256GB - Space Black"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := clean(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnknownShopCleanerReturnsNil(t *testing.T) {
	if cleaners.ShopCleaner("unknown_shop") != nil {
		t.Error("expected nil for unknown shop")
	}
}

func TestAckermannShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("ackermann")
	if clean == nil {
		t.Fatal("ShopCleaner(ackermann) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Smartphone »225, 4G Blau«, Blau, 6,1 cm/2,4 Zoll, 0,128 GB Speicherplatz", "225"},
		{"Smartphone »Galaxy A16 128 GB Blau/Schwarz«", "Galaxy A16"},
		{"Smartphone »iPhone 16 Pro 256 GB Desert Titanium«", "iPhone 16 Pro"},
		{"Smartphone »Redmi Note 14 Pro 5G 256 GB Phantom Purple«", "Redmi Note 14 Pro"},
		{"Smartphone »moto g85 5G 256 GB Cobalt Blue«", "moto g85"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := clean(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCategoryFilterSkipBrands(t *testing.T) {
	f := cleaners.NewFilter(config.Filter{
		SkipBrands:     []string{"Doro", "Emporia"},
		ExclusionRegex: "",
	})

	if !f("Doro 8200 4G") {
		t.Error("should skip Doro product")
	}
	if !f("Emporia Smart.5 mini") {
		t.Error("should skip Emporia product")
	}
	if f("Samsung Galaxy A15") {
		t.Error("should NOT skip Samsung product")
	}
}

func TestCategoryFilterExclusionRegex(t *testing.T) {
	f := cleaners.NewFilter(config.Filter{
		SkipBrands:     nil,
		ExclusionRegex: "(?i)Cover|Hülle|Schutzfolie",
	})

	if !f("Samsung Galaxy A15 Hülle Silikon") {
		t.Error("should skip Hülle product")
	}
	if !f("iPhone 16 Pro Cover Case") {
		t.Error("should skip Cover product")
	}
	if f("Samsung Galaxy A15 128GB") {
		t.Error("should NOT skip regular product")
	}
}

func TestCategoryFilterCombined(t *testing.T) {
	f := cleaners.NewFilter(config.Filter{
		SkipBrands:     []string{"Doro"},
		ExclusionRegex: "(?i)Hülle",
	})

	if !f("Doro 8200") {
		t.Error("should skip by brand")
	}
	if !f("Samsung Hülle") {
		t.Error("should skip by regex")
	}
	if f("Samsung Galaxy A15") {
		t.Error("should NOT skip")
	}
}

func TestCleanPipeline(t *testing.T) {
	shopClean := cleaners.ShopCleaner("galaxus")
	filter := cleaners.NewFilter(config.Filter{
		SkipBrands:     []string{"Doro"},
		ExclusionRegex: "(?i)Hülle",
	})

	products := []string{
		"Samsung Galaxy A15 (128 GB, Black, 6.50\", Dual SIM, 50 Mpx, 4G)",
		"Doro 8200 (64 GB, Black, 6.10\", Dual SIM, 16 Mpx, 4G)",
		"Samsung Galaxy Hülle (Black)",
	}

	var kept []string
	for _, name := range products {
		cleaned := name
		if shopClean != nil {
			cleaned = shopClean(cleaned)
		}
		if !filter(cleaned) {
			kept = append(kept, cleaned)
		}
	}

	if len(kept) != 1 {
		t.Fatalf("kept %d products, want 1: %v", len(kept), kept)
	}
	if kept[0] != "Samsung Galaxy A15 128 GB Black" {
		t.Errorf("kept[0] = %q, want %q", kept[0], "Samsung Galaxy A15 128 GB Black")
	}
}

func TestBrackShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("brack")
	if clean == nil {
		t.Fatal("ShopCleaner(brack) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Samsung Galaxy A54 5G 128 GB Awesome Graphite", "Samsung Galaxy A54"},
		{"Apple iPhone 15 128 GB Schwarz", "Apple iPhone 15"},
		{"Samsung Galaxy XCover 7 Enterprise Edition", "Samsung Galaxy XCover 7 EE"},
		{"Fairphone Fairphone 5 256 GB Schwarz Matt", "Fairphone 5"},
		{"Google Pixel 9a", "Google Pixel 9a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := clean(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
