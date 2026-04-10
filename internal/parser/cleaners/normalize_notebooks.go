package cleaners

import (
	"regexp"
	"strings"
)

// --- Notebook brand normalizers ---

func normalizeAcerNotebook(name string) string {
	name = regexp.MustCompile(`\(?([A-Z]{1,3}[0-9]{2,3}(?:[A-Z]{1,2})?(?:-[0-9]{2,3}[A-Z]*))\)?`).ReplaceAllString(name, "($1)")
	name = regexp.MustCompile(`\)-(.*?)\).*`).ReplaceAllString(name, ")")
	return name
}

func normalizeAppleNotebook(name string) string {
	name = regexp.MustCompile(`\s(\d{2})\s`).ReplaceAllString(name, " $1\" ")
	name = regexp.MustCompile(`(\d{2})"?.*?(\d{4}).*?(M\d)`).ReplaceAllString(name, "($1\", $3, $2)")
	name = regexp.MustCompile(`\(?(\d{2})"?.*?(M\d).*?(\d{4})`).ReplaceAllString(name, "($1\", $2, $3)")
	name = regexp.MustCompile(`\).*$`).ReplaceAllString(name, ")")

	if inch := regexp.MustCompile(`\s(\d{2})"\s`).FindStringSubmatch(name); inch != nil {
		name = strings.ReplaceAll(name, inch[0], " ")
		name = regexp.MustCompile(`\((.*?)\)`).ReplaceAllString(name, "("+inch[1]+"\", $1)")
	}
	return name
}

func normalizeAsusNotebook(name string) string {
	name = regexp.MustCompile(`\(?([A-Z]{1,2}\d{3,5}[A-Z]{0,3}(?:-[A-Z0-9]+)?)\)?`).ReplaceAllString(name, "($1)")
	name = regexp.MustCompile(`-(.*?)\).*`).ReplaceAllString(name, ")")
	return name
}

func normalizeDellNotebook(name string) string {
	return regexp.MustCompile(`\((.*?)\)?$`).ReplaceAllString(name, "($1)")
}

func normalizeHPNotebook(name string) string {
	return regexp.MustCompile(`(?i)(\s+Next\s+Gen\s+)?(AI\s+PC).*`).ReplaceAllString(name, "")
}
