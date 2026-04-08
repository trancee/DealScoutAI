package cleaners

import (
	"regexp"
	"strings"
)

var galaxusParenRe = regexp.MustCompile(`\s*\(.*\)\s*$`)

func cleanGalaxus(name string) string {
	// Galaxus format: "Brand Model (Storage, Color, Screen, SIM, Camera, Network)"
	// Extract key specs from parentheses before removing them.
	match := galaxusParenRe.FindString(name)
	base := galaxusParenRe.ReplaceAllString(name, "")

	if match == "" {
		return strings.TrimSpace(base)
	}

	// Parse parenthesized specs — keep storage and color (first two fields).
	inner := strings.Trim(match, " ()")
	parts := strings.Split(inner, ", ")

	var kept []string
	for i, part := range parts {
		if i >= 2 {
			break
		}
		kept = append(kept, strings.TrimSpace(part))
	}

	result := base
	if len(kept) > 0 {
		result += " " + strings.Join(kept, " ")
	}

	return strings.TrimSpace(result)
}

var amazonSuffixRe = regexp.MustCompile(`,\s*(Android|Smartphone|Handy|Mobiltelefon|Mobile Phone).*$`)

func cleanAmazon(name string) string {
	// Amazon format: "Brand Model, Android Smartphone, specs..."
	// Strip everything from the first category-indicator comma onward.
	name = amazonSuffixRe.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

var ackermannGuillemetsRe = regexp.MustCompile(`»(.*?)«`)
var ackermannSpecSuffixRe = regexp.MustCompile(`(?i)(,\s*)?\d+\s*GB|(,\s*)?\(?[2345]G\)?| LTE`)

func cleanAckermann(name string) string {
	// Ackermann format: "Type »Brand Model Specs« extra description"
	// Extract the content between guillemets as the product name.
	if matches := ackermannGuillemetsRe.FindStringSubmatch(name); len(matches) > 1 {
		name = matches[1]
	}

	// Strip storage/network suffixes (e.g., "128 GB", "5G", "LTE").
	if loc := ackermannSpecSuffixRe.FindStringIndex(name); loc != nil {
		name = name[:loc[0]]
	}

	return strings.TrimSpace(name)
}

var brackSpecRe = regexp.MustCompile(`(\s*[-,]\s+)|(\d+\s*GB?)|\s+CH$`)

func cleanBrack(name string) string {
	name = strings.NewReplacer("Enterprise Edition", "EE", "Fairphone Fairphone", "Fairphone").Replace(name)

	if loc := brackSpecRe.FindStringSubmatchIndex(name); loc != nil {
		name = name[:loc[0]]
	}

	return strings.TrimSpace(name)
}
