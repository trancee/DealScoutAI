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
		{"Samsung Smartphone »Galaxy A16 128 GB Blau/Schwarz«", "Samsung Galaxy A16"},
		{"Apple Smartphone »iPhone 16 Pro 256 GB Desert Titanium«", "Apple iPhone 16 Pro"},
		{"Xiaomi Smartphone »Redmi Note 14 Pro 5G 256 GB Phantom Purple«", "Xiaomi Redmi Note 14 Pro"},
		{"Motorola Smartphone »moto g85 5G 256 GB Cobalt Blue«", "Motorola moto g85"},
		{"TCL Handy »4042S 4G mit Cradle« Grau", "TCL 4042S"},
		{"ZTE Smartphone »Blade V70 Max« Galactic Gray", "ZTE Blade V70 Max"},
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

func TestConradShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("conrad")
	if clean == nil {
		t.Fatal("ShopCleaner(conrad) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Xiaomi Redmi 9AT Smartphone  32 GB 16.6 cm (6.53 Zoll) Gletscherblau Android 10 Dual-SIM", "Xiaomi Redmi 9AT"},
		{"Samsung Galaxy A54 5G Smartphone 128 GB 16.3 cm", "Samsung Galaxy A54"},
		{"Samsung XCover 7 Enterprise Edition 128 GB Schwarz", "Samsung Galaxy XCover 7 EE"},
		{"Samsung Galaxy A16 EU-Ware 128 GB Schwarz", "Samsung Galaxy A16"},
		{"Nokia Nokia 105 (2023)", "Nokia 105 (2023)"},
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

func TestFolettiShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("foletti")
	if clean == nil {
		t.Fatal("ShopCleaner(foletti) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Samsung Galaxy A16 128 GB Schwarz", "Samsung Galaxy A16"},
		{"Apple iPhone 16 Pro 256 GB Desert Titanium", "Apple iPhone 16 Pro"},
		{"Xiaomi Redmi 15C 17.5 cm (6.9) 4G USB Type-C 4 GB 128 GB 6000 mAh Black", "Xiaomi Redmi 15C"},
		{"Motorola moto g05 - 6.67 Zoll - Dual SIM", "Motorola moto g05"},
		{"Samsung Galaxy XCover 7 Enterprise Edition", "Samsung Galaxy XCover 7 EE"},
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

func TestInterdiscountShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("interdiscount")
	if clean == nil {
		t.Fatal("ShopCleaner(interdiscount) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"SAMSUNG Galaxy A16 (128 GB, Schwarz, 6.7\", 50 MP, 5G)", "SAMSUNG Galaxy A16"},
		{"APPLE iPhone 15 (128 GB, Schwarz, 6.1\", 48 MP, 5G)", "APPLE iPhone 15"},
		{"NOKIA Nokia 225 4G (64 MB, 2.4\", 0.3 MP, Blau)", "NOKIA 225"},
		{"SAMSUNG Galaxy XCover 7 Enterprise Edition (128 GB)", "SAMSUNG Galaxy XCover 7 EE"},
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

func TestMediamarktShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("mediamarkt")
	if clean == nil {
		t.Fatal("ShopCleaner(mediamarkt) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"SAMSUNG Galaxy A16 128 GB Schwarz", "SAMSUNG Galaxy A16"},
		{"APPLE iPhone 15 - 128 GB Schwarz", "APPLE iPhone 15"},
		{"XIAOMI Redmi Note 14 Pro 5G 256 GB", "XIAOMI Redmi Note 14 Pro"},
		{"ONE PLUS Nord CE4 Lite 5G 256 GB", "ONEPLUS Nord CE4 Lite"},
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

func TestMobilezoneShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("mobilezone")
	if clean == nil {
		t.Fatal("ShopCleaner(mobilezone) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Samsung Galaxy A16 128GB Black", "Samsung Galaxy A16"},
		{"Apple iPhone 16 Pro 256GB Desert Titanium", "Apple iPhone 16 Pro"},
		{"Xiaomi Redmi Note 14 Pro 5G 256GB Phantom Purple", "Xiaomi Redmi Note 14 Pro"},
		{"Samsung Galaxy Xcover5 64GB Black", "Samsung Galaxy XCover 5"},
		{"Google Pixel 9a 128GB Iris Dual Sim", "Google Pixel 9a"},
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

func TestOrderflowShopCleaner(t *testing.T) {
	clean := cleaners.ShopCleaner("orderflow")
	if clean == nil {
		t.Fatal("ShopCleaner(orderflow) returned nil")
	}

	tests := []struct {
		input, want string
	}{
		{"Samsung Galaxy A16 128 GB Schwarz", "Samsung Galaxy A16"},
		{"Apple iPhone 16 Pro 256 GB Desert Titanium 5G", "Apple iPhone 16 Pro"},
		{"Motorola Mobility moto g15 128GB Dual SIM", "moto g15"},
		{"Samsung Galaxy XCover 7 Enterprise Edition 128 GB", "Samsung Galaxy XCover 7 EE"},
		{"Nokia 225 4G 128 MB Blau", "Nokia 225 128 MB Blau"},
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
