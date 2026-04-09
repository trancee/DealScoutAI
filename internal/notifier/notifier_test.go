package notifier_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/trancee/DealScout/internal/deal"
	"github.com/trancee/DealScout/internal/notifier"
)

func TestEscapeMarkdownV2(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Samsung Galaxy A15", "Samsung Galaxy A15"},
		{"Price: CHF 119.00", `Price: CHF 119\.00`},
		{"(128 GB)", `\(128 GB\)`},
		{"50% off!", `50% off\!`},
		{"hello-world", `hello\-world`},
		{"test_underscore", `test\_underscore`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := notifier.EscapeMarkdownV2(tt.input)
			if got != tt.want {
				t.Errorf("EscapeMarkdownV2(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatCaptionFirstSeen(t *testing.T) {
	d := deal.Deal{
		ProductName: "Samsung Galaxy A15",
		Category:    "smartphones",
		Shop:        "Galaxus",
		Price:       149.0,
		URL:         "https://example.com/product",
		ImageURL:    "https://img.com/1.jpg",
		IsFirstSeen: true,
	}

	caption := notifier.FormatCaption(d)

	if !strings.Contains(caption, "Samsung Galaxy A15") {
		t.Error("caption missing product name")
	}
	if !strings.Contains(caption, "149") {
		t.Error("caption missing price")
	}
	if !strings.Contains(caption, "Galaxus") {
		t.Error("caption missing shop")
	}
	if !strings.Contains(caption, "View Product") {
		t.Error("caption missing link")
	}
	// First-seen should NOT have discount line.
	if strings.Contains(caption, "📉") {
		t.Error("first-seen should not have discount line")
	}
}

func TestFormatCaptionWithDiscount(t *testing.T) {
	oldPrice := 179.0
	d := deal.Deal{
		ProductName: "Samsung Galaxy A15",
		Category:    "smartphones",
		Shop:        "Galaxus",
		Price:       149.0,
		OldPrice:    &oldPrice,
		DiscountPct: 14.0,
		LastDBPrice: 173.0,
		URL:         "https://example.com/product",
		IsFirstSeen: false,
	}

	caption := notifier.FormatCaption(d)

	if !strings.Contains(caption, "📉") {
		t.Error("caption missing discount line")
	}
	if !strings.Contains(caption, "14%") {
		t.Error("caption missing discount percentage")
	}
	if !strings.Contains(caption, "🏷️") {
		t.Error("caption missing shop price line")
	}
}

func TestSendUsesMessage(t *testing.T) {
	var calledEndpoint string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledEndpoint = r.URL.Path
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	n := notifier.New("tok", "-123", map[string]int{"smartphones": 42}).WithAPIBase(server.URL)

	d := deal.Deal{
		ProductName: "Test Phone",
		Category:    "smartphones",
		Shop:        "TestShop",
		Price:       100.0,
		ImageURL:    "https://img.com/test.jpg",
		IsFirstSeen: true,
	}

	if err := n.Send(d); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(calledEndpoint, "sendMessage") {
		t.Errorf("endpoint = %q, want sendMessage", calledEndpoint)
	}
}

func TestSendNoImageUsesMessage(t *testing.T) {
	var calledEndpoint string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledEndpoint = r.URL.Path
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	n := notifier.New("tok", "-123", map[string]int{"smartphones": 0}).WithAPIBase(server.URL)

	d := deal.Deal{
		ProductName: "Test Phone",
		Category:    "smartphones",
		Shop:        "TestShop",
		Price:       100.0,
		ImageURL:    "",
		IsFirstSeen: true,
	}

	if err := n.Send(d); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(calledEndpoint, "sendMessage") {
		t.Errorf("endpoint = %q, want sendMessage when no image", calledEndpoint)
	}
}
