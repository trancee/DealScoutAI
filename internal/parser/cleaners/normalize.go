package cleaners

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var colorRegex = regexp.MustCompile(`(?i)(in )?([/(,]?(Ancora|Aqua|Arctic|Astral|Astro|Atlantic|Aurora|Awesome|Azul|Bamboo|Bianco|Black|(Hell ?|Pazifik)?Blau|Bleu|Blue?|Burgund|Butter|Champagne|Charcoal|Chrome|Cloudy?|Clover|Copper|Cosmic|Cosmo|Dark|Denim|Diamond|Dusk|Electric|Elegant|Frost(ed)?|Galactic|Gelb|Glacier|Glazed|Glowing|Gold(ig)?|Gradient|Granite|Graphite|Gr[ae]y|Green|Grau|Gravity|Gris|Grün|Himalaya|Holunderweiss|Ic[ey]|Interstellar|Lagoon|Lake|Lavende[lr]|Light|(Dunkel)?Lila|Luminous|Magenta|Marine|Matte|Metallic|Meteorite|Meteor|Midday|Midnight|Mint(green)?|Misty?|Mitternacht(sschwarz)?|Moonlight|Mystic|Nachtgrün|Navy|Nero|Night|Noir|Obsidian|Ocean|Onyx|Orange|Orchid| Oro|Pacific|Pastel|Peacock|Pearl|Pebble|Pepper|Perlmutweiss|Petrol|Piano|Pink(gold)?|Polar(stern)?|Prism|Purple|Red( Edition)?|Rosa|Rose|Ros[aée]gold|Rosso|Rot|Sage|Sakura|Salbeigrün|Sandy|Schwarz|Shadow|Silber|Silvery?|Sky|Space(grey)?|Stardust|Stargaze|Starlight|Starry|Star|Steel|Sterling|Sternenblau|Stone|Sunburst|Sunrise|Sunset|Titanium|Titan|Türkis|Twilight|Violette?|Violet|Waldgrün|Weiss|Weiß|White|Yellow|Zeus)\b[\s/]?)(Azur|Black|Blau|Bleen|Blue|Bronze|Cream|Dream|Gold|Green|Gr[ae]y|Grün|Grau|Lime|Navy|Onyx|Pink|Rose|Schwarz|Silber|Silver|White|Weiss)?[)]?`)

var nameMapping = map[string]string{
	"Ee": "EE", "Fe": "FE", "Gt": "GT", "Hd": "HD",
	"Htc": "HTC", "Iphone": "iPhone", "Lg": "LG",
	"Lte": "LTE", "Motorola Moto": "Motorola moto",
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

	name = productNormalizer(name)

	// Apply brand/model corrections.
	for wrong, right := range nameMapping {
		if strings.Contains(name, wrong) {
			name = replaceWord(name, wrong, right)
		}
	}

	return strings.TrimSpace(name)
}

func productNormalizer(name string) string {
	s := strings.Split(name, " ")
	brand := strings.ToLower(s[0])

	if brand == "the" {
		name = strings.Join(s[1:], " ")

		s = strings.Split(name, " ")
		brand = strings.ToLower(s[0])
	}

	if brand == "iphone" {
		name = "Apple" + " " + name

		s = strings.Split(name, " ")
		brand = strings.ToLower(s[0])
	}

	if brand == "apple" {
		name = strings.Split(name, "/")[0]

		name = regexp.MustCompile(`(?i)\s+SE\s*\(?(2016|2020|2022)\)?`).ReplaceAllString(name, " SE ($1)")
		name = regexp.MustCompile(`(?i)\(?1(\.|st)\s*Gen(eration)?\.?\)?`).ReplaceAllString(name, "(2016)")
		name = regexp.MustCompile(`(?i)\(?2(\.|nd)\s*Gen(eration)?\.?\)?`).ReplaceAllString(name, "(2020)")
		name = regexp.MustCompile(`(?i)\(?3(\.|rd)\s*Gen(eration)?\.?\)?`).ReplaceAllString(name, "(2022)")
		name = regexp.MustCompile(`(?i)\s*(\d+)\s*(S)`).ReplaceAllString(name, " $1$2")

		name = strings.ReplaceAll(name, "SE 2", "SE (2020)")
		name = strings.ReplaceAll(name, "SE2", "SE (2020)")
		name = strings.ReplaceAll(name, "11 (2020)", "11")

		name = strings.ReplaceAll(name, "(2016) 2016", "(2016)")
		name = strings.ReplaceAll(name, "(2020) 2020", "(2020)")
		name = strings.ReplaceAll(name, "(2022) 2022", "(2022)")
		name = strings.ReplaceAll(name, "(2016) (2016)", "(2016)")
		name = strings.ReplaceAll(name, "(2020) (2020)", "(2020)")
		name = strings.ReplaceAll(name, "(2020) (2022)", "(2020)")
		name = strings.ReplaceAll(name, "(2022) (2022)", "(2022)")
	}

	if brand == "blackview" {
		name = regexp.MustCompile(`(?i)BL\s*(\d+)`).ReplaceAllString(name, "BL$1")
		name = regexp.MustCompile(`(?i)BV\s*(\d+)`).ReplaceAllString(name, "BV$1")
		// name = regexp.MustCompile(`(?i)(B[LV]\d+)\s*s`).ReplaceAllString(name, "$1s")
		name = regexp.MustCompile(`(?i)\(\d+\)`).ReplaceAllString(name, "")
	}

	if brand == "fairphone" {
		name = regexp.MustCompile(`(?i)Fairphone\s*\(?Gen(eration)?\.?\s*(\d+)\)?`).ReplaceAllString(name, "Fairphone $2")
	}

	if brand == "gigaset" {
		name = strings.ReplaceAll(name, " Ls", "LS")

		name = regexp.MustCompile(`(?i)G[LSX]\d+(LS)?`).ReplaceAllStringFunc(name, strings.ToUpper)
		name = regexp.MustCompile(`(?i)senior$`).ReplaceAllStringFunc(name, strings.ToLower)
		name = regexp.MustCompile(`(?i)LITE$`).ReplaceAllStringFunc(name, strings.ToUpper)
	}

	if brand == "google" {
		name = regexp.MustCompile(`(?i)\d+[A]$`).ReplaceAllStringFunc(name, strings.ToLower)
	}

	if brand == "honor" {
		name = regexp.MustCompile(`(?i)HONOR`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = regexp.MustCompile(`(?i)Magic\s*(\d)\s*(\w)?`).ReplaceAllString(name, "Magic$1 $2")
		name = regexp.MustCompile(`(?i)HONOR\s*(\d+[X]?)\s*(\w)?`).ReplaceAllString(name, "HONOR $1 $2")
	}

	if brand == "hmd" {
		name = regexp.MustCompile(`(?i)HMD`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = regexp.MustCompile(`(?i)TA-\d{4}`).ReplaceAllString(name, "")
	}

	if brand == "huawei" {
		name = regexp.MustCompile(`(?i)HUAWEI`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = regexp.MustCompile(`(?i)Ascend\s*([Y])\s*(\d+)`).ReplaceAllString(name, "Ascend $1$2")
		name = regexp.MustCompile(`(?i)Magic\s*(\d+)`).ReplaceAllString(name, "Magic$1")
		name = regexp.MustCompile(`(?i)Mate\s*(\d+)`).ReplaceAllString(name, "Mate $1")
		name = regexp.MustCompile(`(?i)nova\s*([Y]?\d+)?`).ReplaceAllString(name, "nova $1")
		name = regexp.MustCompile(`(?i)P\s*(smart[+]?)\s*(\+?)`).ReplaceAllString(name, "P smart$2 $3")
		name = regexp.MustCompile(`(?i)P-?\s*(\d+)`).ReplaceAllString(name, "P$1")
		name = regexp.MustCompile(`(?i)\s*SE$`).ReplaceAllString(name, " SE")

		name = regexp.MustCompile(`(?i)lite`).ReplaceAllString(name, "lite")

		name = regexp.MustCompile(`(?i)\d+i$`).ReplaceAllStringFunc(name, strings.ToLower)

		name = regexp.MustCompile(`(?i)NE$`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = strings.ReplaceAll(name, " Gx8", " GX8")
	}

	if brand == "moto" {
		name = "Motorola" + " " + name

		s = strings.Split(name, " ")
		brand = strings.ToLower(s[0])
	}

	if brand == "motorola" {
		name = strings.ToLower(name)

		if len(s) > 2 && s[1] == "Moto" && s[2] == "Edge" {
			name = strings.ReplaceAll(name, "moto ", "")
		}

		name = strings.ReplaceAll(name, "2nd gen", "???")

		name = regexp.MustCompile(`(?i)edge\s*(\d+)\s*(\w*)`).ReplaceAllString(name, "edge $1 $2")
		name = regexp.MustCompile(`(?i)^motorola\s*(moto\s*)?([eg]\s*)?([eg])\s*(\d+)`).ReplaceAllString(name, "motorola moto $3$4")

		name = strings.ReplaceAll(name, "???", "2nd gen")

		name = strings.ReplaceAll(name, "defy (2021)", "defy")
	}

	if brand == "nothing" {
		name = regexp.MustCompile(`-?\(?(\d+[Aa]?)\)?`).ReplaceAllString(name, "($1)")
		name = regexp.MustCompile(`\(\d+[Aa]?\)`).ReplaceAllStringFunc(name, strings.ToLower)
	}

	if brand == "oukitel" {
		name = regexp.MustCompile(`(?i)OUKITEL`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = regexp.MustCompile(`(?i)WP\s*(\d+)`).ReplaceAllString(name, "WP$1")
	}

	if brand == "oneplus" {
		name = regexp.MustCompile(`(?i)\s*CPH\d{4}`).ReplaceAllString(name, "")

		// OnePlus Nord CE4 Lite
		name = regexp.MustCompile(`(?i)CE\s*(\d+)`).ReplaceAllString(name, "CE$1")

		name = regexp.MustCompile(`(?i)Nord\s*(\d+)`).ReplaceAllString(name, "Nord $1")
	}

	if brand == "oppo" {
		name = regexp.MustCompile(`(?i)OPPO`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = regexp.MustCompile(`(?i)\s*CPH\d{4}`).ReplaceAllString(name, "")

		name = regexp.MustCompile(`(?i)([A])\s*(\d+)\s*([ks])?\s*\(?(\d{4})?\)?`).ReplaceAllString(name, "$1$2$3 $4")
		name = regexp.MustCompile(`[KS]\s*$`).ReplaceAllStringFunc(name, strings.ToLower)

		name = regexp.MustCompile(`(?i)Reno\s*(\d+)\s*(\w)?`).ReplaceAllString(name, "Reno$1 $2")
		name = regexp.MustCompile(`(?i)OPPO\s*(\d+)\s*(\w)?`).ReplaceAllString(name, "OPPO Reno$1 $2")
	}

	if brand == "oscal" {
		name = regexp.MustCompile(`(?i)OSCAL`).ReplaceAllStringFunc(name, strings.ToUpper)
	}

	if brand == "realme" {
		name = regexp.MustCompile(`(?i)realme`).ReplaceAllStringFunc(name, strings.ToLower)

		name = regexp.MustCompile(`(?i)narzo`).ReplaceAllStringFunc(name, strings.ToLower)

		// realme Note70T
		name = regexp.MustCompile(`(?i)Note\s*(\d+[T]?)`).ReplaceAllString(name, "Note $1")

		name = regexp.MustCompile(`(?i)GT\s*(\d+)`).ReplaceAllString(name, "GT $1")
		name = strings.ReplaceAll(name, "Neo ", "NEO ")

		name = regexp.MustCompile(`(?i)\d+i`).ReplaceAllStringFunc(name, strings.ToLower)
		name = regexp.MustCompile(`(?i)\d+[y]`).ReplaceAllStringFunc(name, strings.ToUpper)
	}

	if brand == "galaxy" {
		name = "Samsung" + " " + name

		s = strings.Split(name, " ")
		brand = strings.ToLower(s[0])
	}

	if brand == "samsung" {
		name = regexp.MustCompile(`(?i)samsung`).ReplaceAllString(name, "Samsung")

		name = regexp.MustCompile(`[^FE][E]$`).ReplaceAllStringFunc(name, strings.ToLower)
		name = regexp.MustCompile(`(?i)\s*FE$`).ReplaceAllString(name, " FE")

		// SM-A175FZKBEUE
		// SM-A546B/DS-128GB
		// model := regexp.MustCompile(`(?i).*?SM-([AFGMNS]\d{2})\w+`).ReplaceAllString(name, " Galaxy $1")
		// name = regexp.MustCompile(`(?i)\s+(SM-)?[AFGMNS]\d{3}[BFPR]?[N]?((/DSN?)?([-]?\d+?GB)|\w+)?`).ReplaceAllString(name, model)
		name = regexp.MustCompile(`(?i)\s+(SM-)?[AFGMNS]\d{3}[BFPR]?[N]?((/DSN?)?([-]?\d+?GB)|\w+)?`).ReplaceAllString(name, "")
		// fmt.Println("name:", name, ";", "model:", model)
		name = regexp.MustCompile(`(?i)\s+(GT-)?[N]\d{4}`).ReplaceAllString(name, "")

		name = regexp.MustCompile(`(?i)Z\s*Flip\s*(\d+)`).ReplaceAllString(name, "Z Flip$1")
		name = regexp.MustCompile(`(?i)Z\s*Fold\s*(\d+)`).ReplaceAllString(name, "Z Fold $1")
		name = regexp.MustCompile(`Note\s*(\d+)`).ReplaceAllString(name, "Note$1")
		name = regexp.MustCompile(`(?i)(Note\d+)\s+Plus`).ReplaceAllString(name, "$1+")
		name = regexp.MustCompile(`(?i)( Galaxy)? (Tab )?([AFMS])\s*(\d+| Duos)`).ReplaceAllString(name, " Galaxy $2$3$4")

		name = regexp.MustCompile(`(?i)FieldPro`).ReplaceAllString(name, "Field Pro")

		name = regexp.MustCompile(`(?i)X\s*Cover\s*(\d)`).ReplaceAllString(name, "XCover $1")
		name = regexp.MustCompile(`(?i)\s+[AFMS]\d+$`).ReplaceAllStringFunc(name, strings.ToUpper)
		name = regexp.MustCompile(`\s+\d[s]$`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = strings.ReplaceAll(name, "Samsung Note", "Samsung Galaxy Note")

		// name = regexp.MustCompile(`\(\d{4}\)`).ReplaceAllString(name, "") // remove year annotation
	}

	if brand == "sony" {
		name = regexp.MustCompile(`[DF]\d{4}`).ReplaceAllString(name, "")

		name = regexp.MustCompile(`(?i)\s*(III|II|IV|V|VI|VII)$`).ReplaceAllStringFunc(name, strings.ToUpper)
		name = regexp.MustCompile(`(?i)\s*(III|II|IV|V|VI|VII)$`).ReplaceAllString(name, " $1")
	}

	if brand == "vivo" {
		name = regexp.MustCompile(`(?i)vivo`).ReplaceAllStringFunc(name, strings.ToLower)

		name = regexp.MustCompile(`(?i)NEX`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = regexp.MustCompile(`\d+[CEIS]\b`).ReplaceAllStringFunc(name, strings.ToLower)
	}

	if brand == "wiko" {
		name = regexp.MustCompile(`(?i)View\s*(\d+)`).ReplaceAllString(name, "View $1")
	}

	if brand == "poco" || brand == "redmi" || brand == "mi" {
		name = "Xiaomi" + " " + name

		s = strings.Split(name, " ")
		brand = strings.ToLower(s[0])
	}

	if brand == "xiaomi" {
		name = regexp.MustCompile(`^Xiaomi\s*(\d+)\s*\(?(\d{4})\)?`).ReplaceAllString(name, "Xiaomi Redmi $1 ($2)")
		name = regexp.MustCompile(`(?i)(POCO\s*|Pocophone\s*)?([CFMX]\d+)`).ReplaceAllString(name, "POCO $2")
		name = regexp.MustCompile(`(Redmi\s*)?Note\s*(\d+)`).ReplaceAllString(name, "Redmi Note $2")
		// name = regexp.MustCompile(`Note\s*(\d+)`).ReplaceAllString(name, "Note $1")
		name = regexp.MustCompile(`(?i)Redmi\s*(\d+)\s*([ABC])`).ReplaceAllString(name, "Redmi $1$2")

		// name = regexp.MustCompile(`(?i)\d+[A]$`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = regexp.MustCompile(`\s*\(?(\d{4})\)?`).ReplaceAllString(name, " $1")

		name = regexp.MustCompile(`(?i)Mi(\d+)`).ReplaceAllString(name, "Mi $1")

		name = regexp.MustCompile(`\d+[I]`).ReplaceAllStringFunc(name, strings.ToLower)
		name = regexp.MustCompile(`\d+[t]`).ReplaceAllStringFunc(name, strings.ToUpper)

		name = strings.ReplaceAll(name, "Mi Redmi ", "Mi ")
	}

	if brand == "zte" {
		name = regexp.MustCompile(`(?i)nubia`).ReplaceAllString(name, "nubia")

		name = regexp.MustCompile(`(Blade\s*)?([AV])\s*(\d+)(\s*([EeSs]))?`).ReplaceAllString(name, "Blade $2$3$5")
		name = regexp.MustCompile(`[ES]\s*$`).ReplaceAllStringFunc(name, strings.ToLower)
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
