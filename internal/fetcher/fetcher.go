package fetcher

import (
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
}

// Fetcher performs rate-limited HTTP requests with retries and User-Agent rotation.
type Fetcher struct {
	delaySeconds   int
	maxRetries     int
	retryBaseDelay time.Duration
	client         *http.Client
	uaIndex        atomic.Uint64
	lastRequest    time.Time
}

// New creates a Fetcher with the given delay between requests and max retry count.
func New(delaySeconds int, maxRetries int) *Fetcher {
	return &Fetcher{
		delaySeconds:   delaySeconds,
		maxRetries:     maxRetries,
		retryBaseDelay: time.Second,
		client:         &http.Client{Timeout: 30 * time.Second},
	}
}

// WithRetryBaseDelay sets the base delay for exponential backoff (for testing).
func (f *Fetcher) WithRetryBaseDelay(d time.Duration) *Fetcher {
	f.retryBaseDelay = d
	return f
}

// Get fetches a URL via HTTP GET and returns the response body.
func (f *Fetcher) Get(url string) ([]byte, error) {
	return f.doWithRetries("GET", url, "", nil)
}

// Post fetches a URL via HTTP POST with a body built from a template string
// with placeholder replacements applied. Optional headers are added to the request.
func (f *Fetcher) Post(url string, bodyTemplate string, replacements map[string]string, headers map[string]string) ([]byte, error) {
	body := bodyTemplate
	for placeholder, value := range replacements {
		body = strings.ReplaceAll(body, placeholder, value)
	}
	return f.doWithRetries("POST", url, body, headers)
}

func (f *Fetcher) doWithRetries(method, url, body string, headers map[string]string) ([]byte, error) {
	var lastErr error

	for attempt := range f.maxRetries {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * f.retryBaseDelay
			time.Sleep(backoff)
		}

		f.rateLimit()

		respBody, statusCode, err := f.do(method, url, body, headers)
		if err != nil {
			lastErr = err
			continue
		}

		if statusCode >= 200 && statusCode < 300 {
			return respBody, nil
		}

		lastErr = fmt.Errorf("HTTP %d for %s %s", statusCode, method, url)

		if isRetryable(statusCode) {
			continue
		}

		return nil, lastErr
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: %w", f.maxRetries, lastErr)
}

func (f *Fetcher) do(method, url, body string, headers map[string]string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", f.nextUserAgent())
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	slog.Debug("http request",
		"method", method,
		"url", url,
		"headers", req.Header,
		"body_length", len(body),
		"body_preview", truncate(body, 500),
	)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	slog.Debug("http response",
		"method", method,
		"url", url,
		"status", resp.StatusCode,
		"body_length", len(respBody),
		"body_preview", truncate(string(respBody), 500),
	)

	return respBody, resp.StatusCode, nil
}

func (f *Fetcher) nextUserAgent() string {
	idx := f.uaIndex.Add(1) - 1
	return userAgents[idx%uint64(len(userAgents))]
}

func (f *Fetcher) rateLimit() {
	if f.delaySeconds <= 0 {
		return
	}
	elapsed := time.Since(f.lastRequest)
	delay := time.Duration(f.delaySeconds) * time.Second
	if elapsed < delay {
		time.Sleep(delay - elapsed)
	}
	f.lastRequest = time.Now()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}
