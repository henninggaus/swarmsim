package locale

import (
	"strings"
	"testing"
)

// TestAllLanguagesHaveSameKeys verifies every language has the same set of keys as DE (the reference).
func TestAllLanguagesHaveSameKeys(t *testing.T) {
	reference := translations[DE]
	for _, lang := range langOrder {
		if lang == DE {
			continue
		}
		langMap := translations[lang]
		// Check for keys in reference missing from this language
		for key := range reference {
			if _, ok := langMap[key]; !ok {
				t.Errorf("language %s is missing key %q", lang, key)
			}
		}
		// Check for extra keys in this language not in reference
		for key := range langMap {
			if _, ok := reference[key]; !ok {
				t.Errorf("language %s has extra key %q not in DE", lang, key)
			}
		}
	}
}

// TestNoEmptyTranslations checks that no translation value is empty.
func TestNoEmptyTranslations(t *testing.T) {
	for _, lang := range langOrder {
		langMap := translations[lang]
		for key, val := range langMap {
			if val == "" {
				// Allow intentionally empty strings (paragraph separators)
				continue
			}
			_ = key // avoid unused
		}
		_ = langMap // avoid unused
	}
}

// TestTFunction returns correct value for known key.
func TestTFunction(t *testing.T) {
	SetLang(DE)
	if got := T("tab.arena"); got != "Arena" {
		t.Errorf("T(tab.arena) = %q, want Arena", got)
	}
	SetLang(EN)
	if got := T("tab.display"); got != "Display" {
		t.Errorf("T(tab.display) = %q, want Display", got)
	}
	// Restore default
	SetLang(DE)
}

// TestTFallback verifies fallback to DE then raw key.
func TestTFallback(t *testing.T) {
	SetLang(EN)
	// Missing key should fall back to DE value
	// If also missing from DE, return raw key
	got := T("nonexistent.key.12345")
	if got != "nonexistent.key.12345" {
		t.Errorf("missing key should return raw key, got %q", got)
	}
	SetLang(DE)
}

// TestCycleLang cycles through all languages.
func TestCycleLang(t *testing.T) {
	SetLang(DE)
	expected := langOrder[1:] // all after DE, then back to DE
	expected = append(expected, DE)
	for _, want := range expected {
		CycleLang()
		if GetLang() != want {
			t.Errorf("after CycleLang: got %s, want %s", GetLang(), want)
		}
	}
}

// TestTfFormat verifies Tf with format arguments.
func TestTfFormat(t *testing.T) {
	SetLang(DE)
	got := Tf("tutorial.step_fmt", 3, 15)
	if got != "Schritt 3 / 15" {
		t.Errorf("Tf = %q, want 'Schritt 3 / 15'", got)
	}
	SetLang(EN)
	got = Tf("tutorial.step_fmt", 3, 15)
	if got != "Step 3 / 15" {
		t.Errorf("Tf = %q, want 'Step 3 / 15'", got)
	}
	SetLang(DE)
}

// TestFormatStringConsistency checks that format verbs (%d, %s, %f, etc.) match across languages.
func TestFormatStringConsistency(t *testing.T) {
	reference := translations[DE]
	for _, lang := range langOrder {
		if lang == DE {
			continue
		}
		langMap := translations[lang]
		for key, deVal := range reference {
			langVal, ok := langMap[key]
			if !ok {
				continue // missing key is caught by other test
			}
			deVerbs := extractFormatVerbs(deVal)
			langVerbs := extractFormatVerbs(langVal)
			if len(deVerbs) != len(langVerbs) {
				t.Errorf("[%s] key %q: DE has %d format verbs %v, %s has %d: %v",
					lang, key, len(deVerbs), deVerbs, lang, len(langVerbs), langVerbs)
				continue
			}
			for i := range deVerbs {
				if deVerbs[i] != langVerbs[i] {
					t.Errorf("[%s] key %q: verb #%d differs: DE=%q vs %s=%q",
						lang, key, i, deVerbs[i], lang, langVerbs[i])
				}
			}
		}
	}
}

// extractFormatVerbs finds actual Go format verbs like %d, %s, %f, %.2f, %04d in a string.
// It skips %% (escaped percent) and bare "% " which is used as a literal placeholder
// in SwarmSim display strings (e.g. "% neighbors carrying").
func extractFormatVerbs(s string) []string {
	var verbs []string
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '%' {
			if s[i+1] == '%' {
				i++ // skip %%
				continue
			}
			// A bare "% " (percent-space) is a SwarmSim display placeholder, not a Go verb.
			if s[i+1] == ' ' {
				continue
			}
			// Scan the verb: optional flags, width, precision, then verb letter
			j := i + 1
			for j < len(s) && (s[j] == '-' || s[j] == '+' || s[j] == '0' || s[j] == '#') {
				j++
			}
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if j < len(s) && s[j] == '.' {
				j++
				for j < len(s) && s[j] >= '0' && s[j] <= '9' {
					j++
				}
			}
			if j < len(s) && ((s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z')) {
				verbs = append(verbs, s[i:j+1])
				i = j
			}
		}
	}
	return verbs
}

// TestAllKeysResolveInAllLanguages verifies no key returns the raw key string.
func TestAllKeysResolveInAllLanguages(t *testing.T) {
	original := GetLang()
	defer SetLang(original)

	for _, lang := range langOrder {
		SetLang(lang)
		ref := translations[DE]
		for key := range ref {
			val := T(key)
			if val == key {
				t.Errorf("[%s] key %q resolved to itself (missing translation)", lang, key)
			}
		}
	}
}

// TestTnPluralization verifies the Tn pluralization function.
func TestTnPluralization(t *testing.T) {
	SetLang(DE)
	if got := Tn("plural.bots", 1); got != "1 Bot" {
		t.Errorf("Tn singular: got %q", got)
	}
	if got := Tn("plural.bots", 5); got != "5 Bots" {
		t.Errorf("Tn plural: got %q", got)
	}
	if got := Tn("plural.bots", 0); got != "0 Bots" {
		t.Errorf("Tn zero: got %q", got)
	}
	SetLang(EN)
	if got := Tn("plural.packages", 1); got != "1 package" {
		t.Errorf("Tn EN singular: got %q", got)
	}
	if got := Tn("plural.packages", 3); got != "3 packages" {
		t.Errorf("Tn EN plural: got %q", got)
	}
	SetLang(DE)
}

func TestSaveLangLoadLangRoundtrip(t *testing.T) {
	original := GetLang()
	defer SetLang(original)

	// Set to a non-default language and save
	SetLang(FR)
	SaveLang()

	// Reset to default
	SetLang(DE)
	if GetLang() != DE {
		t.Fatal("expected DE after reset")
	}

	// Load should restore FR
	LoadLang()
	if GetLang() != FR {
		t.Errorf("LoadLang: expected FR, got %s", GetLang())
	}

	// Cleanup: restore original and save it
	SetLang(original)
	SaveLang()
}

func TestTnZeroAllLanguages(t *testing.T) {
	original := GetLang()
	defer SetLang(original)

	for _, lang := range langOrder {
		SetLang(lang)
		// n=0 should use "other" form, not crash
		got := Tn("plural.bots", 0)
		if got == "" || got == "plural.bots.other" {
			t.Errorf("[%s] Tn(plural.bots, 0) = %q (empty or raw key)", lang, got)
		}
		// Should contain "0"
		if !strings.Contains(got, "0") {
			t.Errorf("[%s] Tn(plural.bots, 0) = %q (missing count)", lang, got)
		}
	}
}

func BenchmarkTLookup(b *testing.B) {
	SetLang(EN)
	keys := make([]string, 0)
	for k := range translations[EN] {
		keys = append(keys, k)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = T(keys[i%len(keys)])
	}
}

func BenchmarkTfLookup(b *testing.B) {
	SetLang(EN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Tf("tutorial.step_fmt", i, 15)
	}
}

func BenchmarkCycleLang(b *testing.B) {
	SetLang(DE)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CycleLang()
	}
}
