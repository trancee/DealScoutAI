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

var conradSpecRe = regexp.MustCompile(`\s*[-,]\s+|\W\+\s+|EU |\d+\s*GB|\s*\d+G|\s+\(Version 20[12]\d\)|\s+\(Grade [A-Z]\)|\s+(((Senioren-|senior |Industrie |Outdoor )?Smartphone)|\s*CH$|Satellite|Ex-geschütztes Handy|Fusion( Holiday Edition)?|Refurbished|\(PRODUCT\) RED™|Weiß)`)

func cleanConrad(name string) string {
	name = strings.NewReplacer(
		"Enterprise Edition", "EE",
		"Renewd® ", "",
		"refurbished", "",
		"5G Smartphone", "",
		"Samsung XCover", "Samsung Galaxy XCover",
		"Edge20", "Edge 20",
		"Edge Neo 40", "Edge 40 Neo",
	).Replace(name)

	// Remove duplicate brand prefix (e.g., "Nokia Nokia 105")
	parts := strings.SplitN(name, " ", 3)
	if len(parts) >= 2 && strings.EqualFold(parts[0], parts[1]) {
		name = parts[0] + " " + strings.Join(parts[2:], " ")
	}

	if loc := conradSpecRe.FindStringSubmatchIndex(name); loc != nil {
		name = name[:loc[0]]
	}

	return strings.TrimSpace(name)
}

var folettiSpecRe = regexp.MustCompile(`(?i)\s*[-,]+\s+|\s*\(?(\d+(\s*GB)?[+/])?\d+\s*GB\)?|\s*[45]G|(2|4|6|8|12)/(64|128|256?B?)(GB)?|\s+\(?20[12]\d\)?|\s*\d+([,.]\d+)?\s*(cm|inch|\")|\d{4,5}\s*mAh|\s+20[12]\d|\s+(Hybrid|Dual\W(SIM|Sim)|\s*CH( -|$)|inkl\.|LTE|NFC|smartphone)`)

func cleanFoletti(name string) string {
	name = strings.NewReplacer("Enterprise Edition", "EE", "Enterprise", "EE", "Renewd ", "", "SMARTPHONE ", "", "Smartphone ", "", "Smartfon ", "").Replace(name)

	if loc := folettiSpecRe.FindStringSubmatchIndex(name); loc != nil {
		name = name[:loc[0]]
	}

	return strings.TrimSpace(name)
}

var interdiscountSpecRe = regexp.MustCompile(`\(\d+\s*GB?|\(\d\.\d{1,2}"|\s+[2345]G| LTE`)

func cleanInterdiscount(name string) string {
	name = strings.NewReplacer("Enterprise Edition", "EE").Replace(name)

	if loc := interdiscountSpecRe.FindStringSubmatchIndex(name); loc != nil {
		name = name[:loc[0]]
	}

	// Remove duplicate brand prefix (e.g., "NOKIA Nokia 105")
	parts := strings.SplitN(name, " ", 3)
	if len(parts) >= 2 && strings.EqualFold(parts[0], parts[1]) {
		name = parts[0] + " " + strings.Join(parts[2:], " ")
	}

	return strings.TrimSpace(name)
}

var mediamarktSpecRe = regexp.MustCompile(` - |(64|128)\s*GB|\s+[2345]G|\s+CH$`)

func cleanMediamarkt(name string) string {
	name = strings.NewReplacer("ONE PLUS", "ONEPLUS", "Enterprise Edition", "EE").Replace(name)

	if loc := mediamarktSpecRe.FindStringSubmatchIndex(name); loc != nil {
		name = name[:loc[0]]
	}

	return strings.TrimSpace(name)
}
