package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"swarmsim/locale"
)

var scaffold = flag.Bool("scaffold", false, "print missing keys as Go map entries with EN values as placeholders")

func main() {
	flag.Parse()

	ref := locale.GetTranslations(locale.DE)
	enMap := locale.GetTranslations(locale.EN)
	langs := []locale.Lang{locale.EN, locale.FR, locale.ES, locale.PT, locale.IT, locale.UK}

	totalMissing := 0
	for _, lang := range langs {
		langMap := locale.GetTranslations(lang)
		missing := 0
		for key := range ref {
			if _, ok := langMap[key]; !ok {
				if *scaffold {
					enVal := enMap[key]
					if enVal == "" {
						enVal = ref[key] // fall back to DE if EN also missing
					}
					fmt.Printf("\t\t%q: %q, // TODO: translate\n", key, enVal)
				} else {
					fmt.Printf("[%s] MISSING: %s\n", lang, key)
				}
				missing++
			}
		}
		extra := 0
		for key := range langMap {
			if _, ok := ref[key]; !ok {
				fmt.Printf("[%s] EXTRA:   %s\n", lang, key)
				extra++
			}
		}

		// Check for untranslated (identical to EN) values
		if lang != locale.EN {
			identical := 0
			for key, val := range langMap {
				if enVal, ok := enMap[key]; ok && val == enVal {
					// Skip keys that are legitimately the same in all languages
					// (technical terms, formulas, algorithm names, etc.)
					if isLegitimatelySame(key) {
						continue
					}
					fmt.Printf("[%s] SAME AS EN: %s = %q\n", lang, key, val)
					identical++
				}
			}
			if identical > 0 {
				fmt.Printf("[%s] WARNING: %d keys identical to EN (possibly untranslated)\n\n", lang, identical)
			}
		}

		if missing > 0 || extra > 0 {
			fmt.Printf("[%s] Summary: %d missing, %d extra\n\n", lang, missing, extra)
		} else {
			fmt.Printf("[%s] OK — all %d keys present\n\n", lang, len(ref))
		}
		totalMissing += missing
	}

	if totalMissing > 0 {
		fmt.Printf("TOTAL: %d missing keys across all languages\n", totalMissing)
		os.Exit(1)
	}
	fmt.Println("All languages complete!")
}

// isLegitimatelySame returns true for keys whose values are expected to be
// identical across languages (algorithm names, formulas, short labels, etc.).
func isLegitimatelySame(key string) bool {
	prefixes := []string{"help.math.", "algoexpl.", "tooltip.algo:", "tooltip.preset:"}
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}
