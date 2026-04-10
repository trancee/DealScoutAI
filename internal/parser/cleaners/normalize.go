package cleaners

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// removed `Chrome`: Lenovo Chrome 14M9610
var colorRegex = regexp.MustCompile(`(?i)(in )?([/(,]?(Ancora|Aqua|Arctic|Astral|Astro|Atlantic|Aurora|Awesome|Azul|Bamboo|Bianco|Black|(Dunkel|Hell ?|Nebel|Pazifik)?Blau|Bleu|Blue?|Burgund|Butter|Champagne|Charcoal|Cloudy?|Clover|Copper|Cosmic|Cosmo|Dark|Denim|Diamond|Dusk|Electric|Elegant|Frost(ed)?|Galactic|Gelb|Glacier|Glazed|Glowing|Gold(ig|farben)?|Gradient|Granite|Graphite|Gr[ae]y|Green|(Dunkel)?Grau|Gravity|Gris|(Dunkel)?Grün|Himalaya|Holunderweiss|Ic[ey]|Interstellar|Lagoon|Lake|Lavende[lr]|Light|(Dunkel)?Lila|Luminous|Magenta|Marine|Matte|Metallic|Meteorite|Meteor|Midday|Midnight|Mint(green)?|Minze|Misty?|Mitternacht(sschwarz)?|Moonlight|Mystic|Nachtgrün|Navy|Nero|Night|Noir|Obsidian|Ocean|Onyx|Orange|Orchid| Oro|Pacific|Pastel|Peacock|Pearl|Pebble|Pepper|Perlmutweiss|Petrol|Piano|Pink(gold)?|Polar(stern)?|Prism|Purple|Red( Edition)?|Rosa|Rose|Ros[aée]gold|Rosso|Rot|Sage|Sakura|Salbeigrün|Sandy|Schwarz|Shadow|Silber|Silvery?|Sky|Space(grey)?|Stardust|Stargaze|Starlight|Starry|Star|Steel|Sterling|Sternenblau|Stone|Sunburst|Sunrise|Sunset|Titanium|Titan|Türkis|Twilight|Violette?|Violet|Waldgrün|Weiss|Weiß|White|Yellow|Zeus)\b[\s/]?)(Azur|Black|Blau|Bleen|Blue|Bronze|Cream|Dream|Gold|Green|Gr[ae]y|Grün|Grau|Lime|Navy|Onyx|Pink|Rose|Schwarz|Silber|Silver|White|Weiss)?[)]?`)

var nameMapping = map[string]string{
	"Ai": "AI", "Amd": "AMD", "(Amd)": "(AMD)", "Asus": "ASUS",
	"Cmf":           "CMF",
	"Ee":            "EE",
	"Expertbook":    "ExpertBook",
	"Fe":            "FE",
	"Fs":            "FS",
	"Gt":            "GT",
	"Hd":            "HD",
	"Hmd":           "HMD",
	"Hp":            "HP",
	"Htc":           "HTC",
	"Ial":           "", // Intel Arrow Lake
	"Ideapad":       "IdeaPad",
	"Iphone":        "iPhone",
	"Lg":            "LG",
	"Loq":           "LOQ",
	"Lte":           "LTE",
	"Macbook":       "MacBook",
	"Motorola Moto": "motorola moto",
	"Nfc":           "NFC",
	"Oled":          "OLED",
	"Oneplus":       "OnePlus",
	"Omnibook":      "OmniBook",
	"Probook":       "ProBook",
	"Pc":            "PC",
	"Se":            "SE",
	"Tcl":           "TCL",
	"Thinkpad":      "ThinkPad",
	"Thinkbook":     "ThinkBook",
	"Tuf":           "TUF",
	"VivoBook":      "Vivobook",
	"Xcover":        "XCover", "Xl": "XL", "Xr": "XR", "Xs": "XS",
	"Zte": "ZTE",
}

var titleCaser = cases.Title(language.Und, cases.NoLower)
var multiSpaceRe = regexp.MustCompile(`\s{2,}`)

// NormalizeName converts an ALL CAPS or mixed-case product name to a
// consistent title-case form with brand-specific corrections.
func NormalizeName(name, category string) string {
	// fmt.Println("---", name)
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}

	if loc := colorRegex.FindStringSubmatchIndex(strings.NewReplacer("ß", "ss", "é", "e").Replace(name)); loc != nil {
		// fmt.Printf("%v\t%-30s %s\n", loc, name[:loc[0]], name)
		name = name[:loc[0]]
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

	name = productNormalizer(name, category)

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
