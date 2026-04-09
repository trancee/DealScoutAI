package cleaners

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var nameMapping = map[string]string{
	"Ee": "EE", "Fe": "FE", "Gt": "GT", "Hd": "HD",
	"Htc": "HTC", "Iphone": "iPhone", "Lg": "LG",
	"Oneplus": "OnePlus", "Se": "SE", "Tcl": "TCL",
	"Zte": "ZTE", "Xcover": "XCover", "Xl": "XL",
	"Xr": "XR", "Xs": "XS", "Hmd": "HMD", "Nfc": "NFC",
	"Tecno": "TECNO", "Umidigi": "UMIDIGI",
}

var titleCaser = cases.Title(language.Und, cases.NoLower)
var multiSpaceRe = regexp.MustCompile(`\s{2,}`)

// NormalizeName converts an ALL CAPS or mixed-case product name to a
// consistent title-case form with brand-specific corrections.
func NormalizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}

	// Collapse whitespace.
	name = multiSpaceRe.ReplaceAllString(name, " ")

	// Normalize each word: title-case words that are fully uppercase (2+ chars)
	// but skip words containing digits (model numbers like CE4, A16).
	words := strings.Split(name, " ")
	for i, w := range words {
		if len(w) > 1 && w == strings.ToUpper(w) && w != strings.ToLower(w) && !containsDigit(w) {
			words[i] = titleCaser.String(strings.ToLower(w))
		}
	}
	name = strings.Join(words, " ")

	// Apply brand/model corrections.
	for wrong, right := range nameMapping {
		if strings.Contains(name, wrong) {
			name = replaceWord(name, wrong, right)
		}
	}

	return strings.TrimSpace(name)
}

func replaceWord(s, old, new string) string {
	words := strings.Split(s, " ")
	for i, w := range words {
		if w == old {
			words[i] = new
		}
	}
	return strings.Join(words, " ")
}

func containsDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}
