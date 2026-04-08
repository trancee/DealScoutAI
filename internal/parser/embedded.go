package parser

import (
	"bytes"
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

// ParseEmbeddedJSON extracts JSON from an HTML <script> tag identified by
// a CSS selector, then parses products using dot-notation field paths.
func ParseEmbeddedJSON(html []byte, scriptSelector string, fields map[string]string) ([]RawProduct, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML for embedded JSON: %w", err)
	}

	scriptContent := doc.Find(scriptSelector).First().Text()
	if scriptContent == "" {
		return nil, fmt.Errorf("no content found for selector %q", scriptSelector)
	}

	return ParseJSON([]byte(scriptContent), fields)
}
