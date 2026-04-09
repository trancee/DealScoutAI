package pipeline

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// responseCache provides a file-based TTL cache for HTTP responses.
type responseCache struct {
	dir string
	ttl time.Duration
}

func newResponseCache(cacheDir string, ttlMinutes int) *responseCache {
	if cacheDir == "" || ttlMinutes <= 0 {
		return nil
	}
	return &responseCache{
		dir: cacheDir,
		ttl: time.Duration(ttlMinutes) * time.Minute,
	}
}

// get returns cached response bytes if the cache file exists and is fresh.
func (c *responseCache) get(shop, category string, page int) ([]byte, bool) {
	if c == nil {
		return nil, false
	}

	path := c.path(shop, category, page)
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}

	if time.Since(info.ModTime()) > c.ttl {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	slog.Debug("cache hit", "shop", shop, "category", category, "page", page, "age", time.Since(info.ModTime()).Round(time.Second))
	return data, true
}

// put writes response bytes to the cache.
func (c *responseCache) put(shop, category string, page int, data []byte) {
	if c == nil {
		return
	}

	path := c.path(shop, category, page)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		slog.Warn("cache mkdir failed", "dir", dir, "error", err)
		return
	}

	if err := os.WriteFile(path, data, 0o640); err != nil {
		slog.Warn("cache write failed", "path", path, "error", err)
	}
}

func (c *responseCache) path(shop, category string, page int) string {
	filename := fmt.Sprintf("%s.%d.bin", sanitizeName(category), page+1)
	return filepath.Join(c.dir, sanitizeName(shop), filename)
}
