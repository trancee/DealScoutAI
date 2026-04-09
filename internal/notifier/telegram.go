package notifier

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/trancee/DealScout/internal/deal"
)

// Notifier sends deal notifications to Telegram.
type Notifier struct {
	botToken   string
	channelID  string
	topics     map[string]int
	apiBase    string
	client     *http.Client
	lastSendAt time.Time
}

// New creates a Telegram Notifier.
func New(botToken, channelID string, topics map[string]int) *Notifier {
	return &Notifier{
		botToken:  botToken,
		channelID: channelID,
		topics:    topics,
		apiBase:   "https://api.telegram.org",
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// WithProxy configures an HTTP/HTTPS/SOCKS5 proxy for Telegram API requests.
func (n *Notifier) WithProxy(proxyURL string) *Notifier {
	if proxyURL == "" {
		return n
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		slog.Error("invalid proxy URL", "proxy", proxyURL, "error", err)
		return n
	}
	n.client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(parsed),
		},
	}
	slog.Info("telegram proxy configured", "proxy", proxyURL)
	return n
}

// WithAPIBase overrides the Telegram API base URL (for testing).
func (n *Notifier) WithAPIBase(base string) *Notifier {
	n.apiBase = base
	return n
}

// SendTestMessage sends a plain text test message to each configured topic.
// Returns the number of topics successfully notified and any errors.
func (n *Notifier) SendTestMessage() (int, error) {
	sent := 0
	var lastErr error

	if len(n.topics) == 0 {
		text := EscapeMarkdownV2("✅ DealScout test message — connection working!")
		if err := n.sendMessage(text, 0); err != nil {
			return 0, fmt.Errorf("sending to General topic: %w", err)
		}
		return 1, nil
	}

	for category, topicID := range n.topics {
		text := EscapeMarkdownV2(fmt.Sprintf("✅ DealScout test message — %s topic (thread %d)", category, topicID))
		if err := n.sendMessage(text, topicID); err != nil {
			lastErr = fmt.Errorf("sending to %s (topic %d): %w", category, topicID, err)
			slog.Error("test message failed", "category", category, "topic_id", topicID, "error", err)
		} else {
			slog.Info("test message sent", "category", category, "topic_id", topicID)
			sent++
		}
		time.Sleep(time.Second)
	}

	if lastErr != nil && sent == 0 {
		return 0, lastErr
	}
	return sent, lastErr
}

// Send posts a deal notification to the appropriate Telegram topic.
func (n *Notifier) Send(d deal.Deal) error {
	n.rateLimit()

	caption := FormatCaption(d)
	topicID := n.topics[d.Category]

	return n.sendMessage(caption, topicID)
}

func (n *Notifier) sendMessage(text string, topicID int) error {
	params := url.Values{
		"chat_id":           {n.channelID},
		"text":              {text},
		"parse_mode":        {"MarkdownV2"},
		"message_thread_id": {fmt.Sprintf("%d", topicID)},
	}

	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", n.apiBase, n.botToken)
	resp, err := n.client.PostForm(apiURL, params)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendMessage %d: %s", resp.StatusCode, extractTelegramError(body))
	}
	return nil
}

func (n *Notifier) rateLimit() {
	elapsed := time.Since(n.lastSendAt)
	if elapsed < time.Second {
		time.Sleep(time.Second - elapsed)
	}
	n.lastSendAt = time.Now()
}

// FormatCaption builds a MarkdownV2 caption for a deal notification.
func FormatCaption(d deal.Deal) string {
	name := EscapeMarkdownV2(d.ProductName)
	price := EscapeMarkdownV2(fmt.Sprintf("CHF %.2f", d.Price))
	shop := EscapeMarkdownV2(d.Shop)

	var lines []string
	lines = append(lines, fmt.Sprintf("🔥 %s", name))
	lines = append(lines, fmt.Sprintf("💰 %s", price))

	if !d.IsFirstSeen && d.DiscountPct > 0 {
		drop := EscapeMarkdownV2(fmt.Sprintf("-%.0f%% (was CHF %.2f)", d.DiscountPct, d.LastDBPrice))
		lines = append(lines, fmt.Sprintf("📉 %s", drop))
	}

	if d.OldPrice != nil {
		shopPrice := EscapeMarkdownV2(fmt.Sprintf("CHF %.2f", *d.OldPrice))
		lines = append(lines, fmt.Sprintf("🏷️ Shop says: %s", shopPrice))
	}

	lines = append(lines, fmt.Sprintf("🏪 %s", shop))

	if d.URL != "" {
		escapedURL := EscapeMarkdownV2(d.URL)
		lines = append(lines, fmt.Sprintf("🔗 [View Product](%s)", escapedURL))
	}

	return strings.Join(lines, "\n")
}

// EscapeMarkdownV2 escapes special characters for Telegram MarkdownV2.
func EscapeMarkdownV2(s string) string {
	replacer := strings.NewReplacer(
		"_", `\_`,
		"*", `\*`,
		"[", `\[`,
		"]", `\]`,
		"(", `\(`,
		")", `\)`,
		"~", `\~`,
		"`", "\\`",
		">", `\>`,
		"#", `\#`,
		"+", `\+`,
		"-", `\-`,
		"=", `\=`,
		"|", `\|`,
		"{", `\{`,
		"}", `\}`,
		".", `\.`,
		"!", `\!`,
	)
	return replacer.Replace(s)
}

// extractTelegramError extracts the "description" field from a Telegram API error response.
func extractTelegramError(body []byte) string {
	var result struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.Description != "" {
		return result.Description
	}
	if len(body) > 200 {
		return string(body[:200]) + "..."
	}
	return string(body)
}
