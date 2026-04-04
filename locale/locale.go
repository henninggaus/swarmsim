// Package locale provides a lightweight i18n system for SwarmSim.
//
// Usage:
//
//	locale.T("tab.arena")        // simple lookup
//	locale.Tf("stats.bots", 42)  // formatted lookup (fmt.Sprintf)
//	locale.CycleLang()           // toggle language in UI
//
// Fallback chain: current language -> DE -> raw key.
// This ensures the app never breaks with incomplete translations.
//
//go:generate go run ../cmd/locale-check/ -scaffold
package locale

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Lang identifies a supported language.
type Lang string

const (
	DE Lang = "de"
	EN Lang = "en"
	FR Lang = "fr"
	ES Lang = "es"
	PT Lang = "pt"
	IT Lang = "it"
	UK Lang = "uk"
)

// mu protects the current language variable from concurrent access.
// T() acquires a read lock; SetLang/CycleLang acquire a write lock.
var mu sync.RWMutex

// current is the active language. Default: German (original UI language).
var current = DE

// langOrder defines the cycle order for CycleLang.
var langOrder = []Lang{DE, EN, FR, ES, PT, IT, UK}

// translations maps language code to key->string.
var translations = map[Lang]map[string]string{
	DE: deStrings,
	EN: enStrings,
	FR: frStrings,
	ES: esStrings,
	PT: ptStrings,
	IT: itStrings,
	UK: ukStrings,
}

// SetLang changes the active language.
func SetLang(l Lang) {
	mu.Lock()
	current = l
	mu.Unlock()
}

// GetLang returns the active language.
func GetLang() Lang {
	mu.RLock()
	l := current
	mu.RUnlock()
	return l
}

// CycleLang advances to the next language in the cycle.
func CycleLang() {
	mu.Lock()
	for i, l := range langOrder {
		if l == current {
			current = langOrder[(i+1)%len(langOrder)]
			mu.Unlock()
			return
		}
	}
	current = DE
	mu.Unlock()
}

// LangDisplayName returns a short display name for the active language.
func LangDisplayName() string {
	mu.RLock()
	s := strings.ToUpper(string(current))
	mu.RUnlock()
	return s
}

// T returns the translated string for the given key.
// Fallback: current lang -> DE -> raw key (makes missing translations visible).
func T(key string) string {
	mu.RLock()
	lang := current
	mu.RUnlock()

	if m, ok := translations[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	// Fallback to German (the original source language)
	if s, ok := translations[DE][key]; ok {
		return s
	}
	return key
}

// Tf is T + fmt.Sprintf for format strings.
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

// Tn returns a pluralized, formatted translation.
// It looks up "key.one" when n==1, "key.other" otherwise.
// The count n is prepended to args for fmt.Sprintf.
func Tn(key string, n int, args ...any) string {
	suffix := ".other"
	if n == 1 {
		suffix = ".one"
	}
	tmpl := T(key + suffix)
	allArgs := append([]any{n}, args...)
	return fmt.Sprintf(tmpl, allArgs...)
}

// GetTranslations returns the translation map for a language (for tooling).
func GetTranslations(l Lang) map[string]string {
	return translations[l]
}

const configFile = "swarmsim_lang.cfg"

// langConfigPath returns the path to the language config file.
func langConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "swarmsim", configFile), nil
}

// SaveLang writes the current language to a config file in the user's config dir.
func SaveLang() {
	mu.RLock()
	lang := string(current)
	mu.RUnlock()

	path, err := langConfigPath()
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(lang), 0644)
}

// LoadLang reads the language from the config file, if it exists.
func LoadLang() {
	path, err := langConfigPath()
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lang := Lang(string(data))
	if _, ok := translations[lang]; ok {
		mu.Lock()
		current = lang
		mu.Unlock()
	}
}
