package cleaners_test

import (
	"testing"

	"github.com/trancee/DealScout/internal/parser/cleaners"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Apple iPhone SE 2020", "Apple iPhone SE (2020)"},
		{"APPLE IPHONE 8", "Apple iPhone 8"},
		{"SAMSUNG Galaxy A16", "Samsung Galaxy A16"},
		{"XIAOMI Redmi Note 14 Pro", "Xiaomi Redmi Note 14 Pro"},
		{"samsung galaxy a16", "Samsung Galaxy A16"},
		{"Apple iPhone 16 Pro", "Apple iPhone 16 Pro"},
		{"NOKIA 225", "Nokia 225"},
		{"ONEPLUS Nord CE4 Lite", "OnePlus Nord CE4 Lite"},
		{"ONE PLUS Nord", "One Plus Nord"},
		{"Google Pixel 9a", "Google Pixel 9a"},
		{"HMD Arc", "HMD Arc"},
		{"ZTE Blade V70", "ZTE Blade V70"},
		{"ZTE Blade V70 Vita stone gray", "ZTE Blade V70 Vita"},
		{"ZTE Blade V70 stardust gray", "ZTE Blade V70"},
		{"ZTE BLADE A35E SILVERY GRAY 64 GB Silvery Gray Dual SIM", "ZTE Blade A35e"},
		{"Samsung Galaxy A17 A176", "Samsung Galaxy A17"},
		// {"Samsung Galaxy A30s Dual-SIM", "Samsung Galaxy A30s"},
		{"Samsung Galaxy XCover 7 EE", "Samsung Galaxy XCover 7 EE"},
		// {"Samsung SM-A175FZKBEUE", "Samsung Galaxy A17"},
		// {"Samsung Galaxy A52s 128 Black Refurbished C+", "Samsung Galaxy A52s"},
		{"  extra   spaces  ", "extra spaces"},
		{"SONIM TECHNOLOGIES XP100", "Sonim Technologies XP100"},
		{"CROSSCALL Core-S5", "Crosscall Core-S5"},

		// Fairphone
		{"The Fairphone (Gen 6)", "Fairphone 6"},

		// realme
		{"realme Note70T obsidian black", "realme Note 70T"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleaners.NormalizeName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
