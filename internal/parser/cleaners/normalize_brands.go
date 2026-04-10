package cleaners

import (
	"strings"
)

// BrandNormalizer transforms a product name for a specific brand.
type BrandNormalizer func(name string) string

// brandAliases maps alternative first-word brands to their canonical brand,
// with an optional prefix to prepend before dispatching.
var brandAliases = map[string]struct {
	canonical string
	prefix    string
}{
	"iphone": {canonical: "apple", prefix: "Apple"},
	"galaxy": {canonical: "samsung", prefix: "Samsung"},
	"moto":   {canonical: "motorola", prefix: "Motorola"},
	"poco":   {canonical: "xiaomi", prefix: "Xiaomi"},
	"redmi":  {canonical: "xiaomi", prefix: "Xiaomi"},
	"mi":     {canonical: "xiaomi", prefix: "Xiaomi"},
}

// smartphoneNormalizers maps lowercase brand names to normalizer functions for smartphones.
var smartphoneNormalizers = map[string]BrandNormalizer{
	"apple":     normalizeAppleSmartphone,
	"blackview": normalizeBlackview,
	"fairphone": normalizeFairphone,
	"gigaset":   normalizeGigaset,
	"google":    normalizeGoogle,
	"honor":     normalizeHonor,
	"hmd":       normalizeHMD,
	"huawei":    normalizeHuawei,
	"motorola":  normalizeMotorola,
	"nothing":   normalizeNothing,
	"oukitel":   normalizeOukitel,
	"oneplus":   normalizeOnePlus,
	"oppo":      normalizeOppo,
	"oscal":     normalizeOscal,
	"realme":    normalizeRealme,
	"samsung":   normalizeSamsungSmartphone,
	"sony":      normalizeSony,
	"vivo":      normalizeVivo,
	"wiko":      normalizeWiko,
	"xiaomi":    normalizeXiaomi,
	"zte":       normalizeZTE,
}

// notebookNormalizers maps lowercase brand names to normalizer functions for notebooks.
var notebookNormalizers = map[string]BrandNormalizer{
	"acer":  normalizeAcerNotebook,
	"apple": normalizeAppleNotebook,
	"asus":  normalizeAsusNotebook,
	"dell":  normalizeDellNotebook,
	"hp":    normalizeHPNotebook,
}

// categoryNormalizers maps category names to their brand normalizer registries.
var categoryNormalizers = map[string]map[string]BrandNormalizer{
	"smartphones": smartphoneNormalizers,
	"notebooks":   notebookNormalizers,
}

// productNormalizer dispatches to per-brand normalizers based on category and brand.
func productNormalizer(name, category string) string {
	s := strings.Split(name, " ")
	brand := strings.ToLower(s[0])

	// Strip leading "the" (e.g. "The Fairphone").
	if brand == "the" && len(s) > 1 {
		name = strings.Join(s[1:], " ")
		s = strings.Split(name, " ")
		brand = strings.ToLower(s[0])
	}

	// Resolve brand aliases (e.g. "iPhone" → prepend "Apple", dispatch to "apple").
	if alias, ok := brandAliases[brand]; ok {
		if alias.prefix != "" {
			name = alias.prefix + " " + name
		}
		brand = alias.canonical
	}

	// Dispatch to category-specific brand normalizer.
	if registry, ok := categoryNormalizers[category]; ok {
		if fn, ok := registry[brand]; ok {
			name = fn(name)
		}
	}

	return strings.TrimSpace(name)
}
