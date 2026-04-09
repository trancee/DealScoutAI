package cleaners_test

import (
	"testing"

	"github.com/trancee/DealScout/internal/parser/cleaners"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"APPLE IPHONE 8", "Apple iPhone 8"},
		{"SAMSUNG Galaxy A16", "Samsung Galaxy A16"},
		{"XIAOMI Redmi Note 14 Pro", "Xiaomi Redmi Note 14 Pro"},
		{"samsung galaxy a16", "samsung galaxy a16"},
		{"Apple iPhone 16 Pro", "Apple iPhone 16 Pro"},
		{"NOKIA 225", "Nokia 225"},
		{"ONEPLUS Nord CE4 Lite", "OnePlus Nord CE4 Lite"},
		{"ONE PLUS Nord", "One Plus Nord"},
		{"Google Pixel 9a", "Google Pixel 9a"},
		{"HMD Arc", "HMD Arc"},
		{"ZTE Blade V70", "ZTE Blade V70"},
		{"Samsung Galaxy XCover 7 EE", "Samsung Galaxy XCover 7 EE"},
		{"  extra   spaces  ", "extra spaces"},
		{"SONIM TECHNOLOGIES XP100", "Sonim Technologies XP100"},
		{"CROSSCALL Core-S5", "Crosscall Core-S5"},
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
