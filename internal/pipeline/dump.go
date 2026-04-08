package pipeline

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/trancee/DealScout/internal/config"
)

// clearShopDumpDir removes all files in the shop's dump directory.
func clearShopDumpDir(dumpDir, shopName string) {
	if dumpDir == "" {
		return
	}
	dir := filepath.Join(dumpDir, sanitizeName(shopName))
	if err := os.RemoveAll(dir); err != nil {
		slog.Warn("failed to clear dump dir", "dir", dir, "error", err)
	}
}

// dumpResponse writes the HTTP response to a file with a curl command as the first line.
func dumpResponse(dumpDir, shopName, category string, page int, method, url string, headers map[string]string, body string, response []byte) {
	if dumpDir == "" {
		return
	}

	dir := filepath.Join(dumpDir, sanitizeName(shopName))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		slog.Warn("failed to create dump dir", "dir", dir, "error", err)
		return
	}

	filename := fmt.Sprintf("%s.%d.txt", sanitizeName(category), page+1)
	path := filepath.Join(dir, filename)

	curlCmd := buildCurlCommand(method, url, headers, body)

	var content []byte
	content = append(content, []byte("# "+curlCmd+"\n\n")...)
	content = append(content, response...)

	if err := os.WriteFile(path, content, 0o640); err != nil {
		slog.Warn("failed to write dump file", "path", path, "error", err)
	}
}

func buildCurlCommand(method, url string, headers map[string]string, body string) string {
	var parts []string
	parts = append(parts, "curl")

	if method != "GET" {
		parts = append(parts, "-X", method)
	}

	for k, v := range headers {
		parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, v))
	}

	if body != "" {
		escaped := strings.ReplaceAll(body, "'", "'\\''")
		if len(escaped) > 500 {
			escaped = escaped[:500] + "..."
		}
		parts = append(parts, "-d", fmt.Sprintf("'%s'", escaped))
	}

	parts = append(parts, fmt.Sprintf("'%s'", url))

	return strings.Join(parts, " ")
}

func buildRequestURL(cat config.ShopCategory, page int) string {
	switch cat.Pagination.Type {
	case "offset":
		return cat.URL
	case "page_param":
		pageNum := cat.Pagination.Start + page
		return strings.ReplaceAll(cat.URL, "{page}", fmt.Sprintf("%d", pageNum))
	default:
		return cat.URL
	}
}

func sanitizeName(name string) string {
	r := strings.NewReplacer(" ", "_", "/", "_", "\\", "_")
	return strings.ToLower(r.Replace(name))
}
