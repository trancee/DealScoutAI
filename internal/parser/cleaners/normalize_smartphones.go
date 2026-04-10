package cleaners

import (
	"regexp"
	"strings"
)

// --- Smartphone brand normalizers ---

func normalizeAppleSmartphone(name string) string {
	name = strings.Split(name, "/")[0]

	name = regexp.MustCompile(`(?i)\s+SE\s*\(?(2016|2020|2022)\)?`).ReplaceAllString(name, " SE ($1)")
	name = regexp.MustCompile(`(?i)\(?1(\.|st)\s*Gen(eration)?\.?\)?`).ReplaceAllString(name, "(2016)")
	name = regexp.MustCompile(`(?i)\(?2(\.|nd)\s*Gen(eration)?\.?\)?`).ReplaceAllString(name, "(2020)")
	name = regexp.MustCompile(`(?i)\(?3(\.|rd)\s*Gen(eration)?\.?\)?`).ReplaceAllString(name, "(2022)")
	name = regexp.MustCompile(`(?i)\s*(\d+)\s*(S)`).ReplaceAllString(name, " $1$2")

	name = strings.ReplaceAll(name, "SE 3", "SE (2022)")
	name = strings.ReplaceAll(name, "SE3", "SE (2022)")
	name = strings.ReplaceAll(name, "SE 2", "SE (2020)")
	name = strings.ReplaceAll(name, "SE2", "SE (2020)")
	name = strings.ReplaceAll(name, "11 (2020)", "11")

	name = strings.ReplaceAll(name, "(2016) 2016", "(2016)")
	name = strings.ReplaceAll(name, "(2020) 2020", "(2020)")
	name = strings.ReplaceAll(name, "(2022) 2022", "(2022)")
	name = strings.ReplaceAll(name, "(2016) (2016)", "(2016)")
	name = strings.ReplaceAll(name, "(2020) (202laptop0)", "(2020)")
	name = strings.ReplaceAll(name, "(2020) (2022)", "(2020)")
	name = strings.ReplaceAll(name, "(2022) (2022)", "(2022)")
	return name
}

func normalizeBlackview(name string) string {
	name = regexp.MustCompile(`(?i)BL\s*(\d+)`).ReplaceAllString(name, "BL$1")
	name = regexp.MustCompile(`(?i)BV\s*(\d+)`).ReplaceAllString(name, "BV$1")
	name = regexp.MustCompile(`(?i)\(\d+\)`).ReplaceAllString(name, "")
	return name
}

func normalizeFairphone(name string) string {
	return regexp.MustCompile(`(?i)Fairphone\s*\(?Gen(eration)?\.?\s*(\d+)\)?`).ReplaceAllString(name, "Fairphone $2")
}

func normalizeGigaset(name string) string {
	name = strings.ReplaceAll(name, " Ls", "LS")
	name = regexp.MustCompile(`(?i)G[LSX]\d+(LS)?`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = regexp.MustCompile(`(?i)senior$`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`(?i)LITE$`).ReplaceAllStringFunc(name, strings.ToUpper)
	return name
}

func normalizeGoogle(name string) string {
	return regexp.MustCompile(`(?i)\d+[A]$`).ReplaceAllStringFunc(name, strings.ToLower)
}

func normalizeHonor(name string) string {
	name = regexp.MustCompile(`(?i)HONOR`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = regexp.MustCompile(`(?i)Magic\s*(\d)\s*(\w)?`).ReplaceAllString(name, "Magic$1 $2")
	name = regexp.MustCompile(`(?i)HONOR\s*(\d+[X]?)\s*(\w)?`).ReplaceAllString(name, "HONOR $1 $2")
	return name
}

func normalizeHMD(name string) string {
	name = regexp.MustCompile(`(?i)HMD`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = regexp.MustCompile(`(?i)TA-\d{4}`).ReplaceAllString(name, "")
	return name
}

func normalizeHuawei(name string) string {
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
	return name
}

func normalizeMotorola(name string) string {
	s := strings.Split(name, " ")
	name = strings.ToLower(name)
	if len(s) > 2 && s[1] == "Moto" && s[2] == "Edge" {
		name = strings.ReplaceAll(name, "moto ", "")
	}
	name = strings.ReplaceAll(name, "2nd gen", "???")
	name = regexp.MustCompile(`(?i)edge\s*(\d+)\s*(\w*)`).ReplaceAllString(name, "edge $1 $2")
	name = regexp.MustCompile(`(?i)^motorola\s*(moto\s*)?([eg]\s*)?([eg])\s*(\d+)`).ReplaceAllString(name, "motorola moto $3$4")
	name = strings.ReplaceAll(name, "???", "2nd gen")
	name = strings.ReplaceAll(name, "pb7e0037se", "edge 60 fusion")
	name = strings.ReplaceAll(name, "defy (2021)", "defy")
	return name
}

func normalizeNothing(name string) string {
	name = regexp.MustCompile(`-?\(?(\d+[Aa]?)\)?`).ReplaceAllString(name, "($1)")
	name = regexp.MustCompile(`\(\d+[Aa]?\)`).ReplaceAllStringFunc(name, strings.ToLower)
	return name
}

func normalizeOukitel(name string) string {
	name = regexp.MustCompile(`(?i)OUKITEL`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = regexp.MustCompile(`(?i)WP\s*(\d+)`).ReplaceAllString(name, "WP$1")
	return name
}

func normalizeOnePlus(name string) string {
	name = regexp.MustCompile(`(?i)\s*CPH\d{4}`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)CE\s*(\d+)`).ReplaceAllString(name, "CE$1")
	name = regexp.MustCompile(`(?i)Nord\s*(\d+)`).ReplaceAllString(name, "Nord $1")
	return name
}

func normalizeOppo(name string) string {
	name = regexp.MustCompile(`(?i)OPPO`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = regexp.MustCompile(`(?i)\s*CPH\d{4}`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)([A])\s*(\d+)\s*([ks])?\s*\(?(\d{4})?\)?`).ReplaceAllString(name, "$1$2$3 $4")
	name = regexp.MustCompile(`[KS]\s*$`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`(?i)Reno\s*(\d+)\s*(\w)?`).ReplaceAllString(name, "Reno$1 $2")
	name = regexp.MustCompile(`(?i)OPPO\s*(\d+)\s*(\w)?`).ReplaceAllString(name, "OPPO Reno$1 $2")
	return name
}

func normalizeOscal(name string) string {
	return regexp.MustCompile(`(?i)OSCAL`).ReplaceAllStringFunc(name, strings.ToUpper)
}

func normalizeRealme(name string) string {
	name = regexp.MustCompile(`(?i)realme`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`(?i)narzo`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`(?i)Note\s*(\d+[T]?)`).ReplaceAllString(name, "Note $1")
	name = regexp.MustCompile(`(?i)GT\s*(\d+)`).ReplaceAllString(name, "GT $1")
	name = strings.ReplaceAll(name, "Neo ", "NEO ")
	name = regexp.MustCompile(`(?i)\d+i`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`(?i)\d+[y]`).ReplaceAllStringFunc(name, strings.ToUpper)
	return name
}

func normalizeSamsungSmartphone(name string) string {
	name = regexp.MustCompile(`(?i)samsung`).ReplaceAllString(name, "Samsung")
	name = regexp.MustCompile(`[^FE][E]$`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`(?i)\s*FE$`).ReplaceAllString(name, " FE")
	name = regexp.MustCompile(`(?i)\s+(SM-)?[AFGMNS]\d{3}[BFPR]?[N]?((/DSN?)?([-]?\d+?GB)|\w+)?`).ReplaceAllString(name, "")
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
	return name
}

func normalizeSony(name string) string {
	name = regexp.MustCompile(`[DF]\d{4}`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)\s*(III|II|IV|V|VI|VII)$`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = regexp.MustCompile(`(?i)\s*(III|II|IV|V|VI|VII)$`).ReplaceAllString(name, " $1")
	return name
}

func normalizeVivo(name string) string {
	name = regexp.MustCompile(`(?i)vivo`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`(?i)NEX`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = regexp.MustCompile(`\d+[CEIS]\b`).ReplaceAllStringFunc(name, strings.ToLower)
	return name
}

func normalizeWiko(name string) string {
	return regexp.MustCompile(`(?i)View\s*(\d+)`).ReplaceAllString(name, "View $1")
}

func normalizeXiaomi(name string) string {
	name = regexp.MustCompile(`^Xiaomi\s*(\d+)\s*\(?(\d{4})\)?`).ReplaceAllString(name, "Xiaomi Redmi $1 ($2)")
	name = regexp.MustCompile(`(?i)(POCO\s*|Pocophone\s*)?([CFMX]\d+)`).ReplaceAllString(name, "POCO $2")
	name = regexp.MustCompile(`(Redmi\s*)?Note\s*(\d+)`).ReplaceAllString(name, "Redmi Note $2")
	name = regexp.MustCompile(`(?i)Redmi\s*(\d+)\s*([ABC])`).ReplaceAllString(name, "Redmi $1$2")
	name = regexp.MustCompile(`\s*\(?(\d{4})\)?`).ReplaceAllString(name, " $1")
	name = regexp.MustCompile(`(?i)Mi(\d+)`).ReplaceAllString(name, "Mi $1")
	name = regexp.MustCompile(`\d+[I]`).ReplaceAllStringFunc(name, strings.ToLower)
	name = regexp.MustCompile(`\d+[t]`).ReplaceAllStringFunc(name, strings.ToUpper)
	name = strings.ReplaceAll(name, "Mi Redmi ", "Mi ")
	return name
}

func normalizeZTE(name string) string {
	name = regexp.MustCompile(`(?i)nubia`).ReplaceAllString(name, "nubia")
	name = regexp.MustCompile(`(Blade\s*)?([AV])\s*(\d+)(\s*([EeSs]))?`).ReplaceAllString(name, "Blade $2$3$5")
	name = regexp.MustCompile(`[ES]\s*$`).ReplaceAllStringFunc(name, strings.ToLower)
	return name
}
