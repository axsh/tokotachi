package scaffold

import (
	"os"
	"strings"
)

// DetectLocale detects the runtime locale.
// Priority: explicitLang > LC_ALL > LANG > "" (empty = no locale).
// Returns a 2-letter language code (e.g. "ja", "en") or empty string.
func DetectLocale(explicitLang string) string {
	if explicitLang != "" {
		return normalizeLocale(explicitLang)
	}

	if lcAll := os.Getenv("LC_ALL"); lcAll != "" {
		return normalizeLocale(lcAll)
	}

	if lang := os.Getenv("LANG"); lang != "" {
		return normalizeLocale(lang)
	}

	return ""
}

// normalizeLocale extracts a 2-letter language code from a locale string.
// Examples: "ja_JP.UTF-8" -> "ja", "en_US" -> "en", "C" -> "", "ja" -> "ja"
func normalizeLocale(locale string) string {
	if locale == "C" || locale == "POSIX" {
		return ""
	}

	// Already a short code
	if len(locale) == 2 {
		return strings.ToLower(locale)
	}

	// Extract language part before underscore or dot
	lang := locale
	if idx := strings.IndexAny(lang, "_."); idx > 0 {
		lang = lang[:idx]
	}

	if len(lang) < 2 {
		return ""
	}

	return strings.ToLower(lang[:2])
}

// MergeLocaleFiles merges base template files with locale overlay files.
// Locale files with the same relative path override the base version.
// Locale-only files (not in base) are added to the result.
func MergeLocaleFiles(baseFiles []DownloadedFile, localeFiles []DownloadedFile) []DownloadedFile {
	if len(localeFiles) == 0 {
		return baseFiles
	}

	// Build a map from base files
	merged := make(map[string]DownloadedFile, len(baseFiles))
	order := make([]string, 0, len(baseFiles))

	for _, f := range baseFiles {
		merged[f.RelativePath] = f
		order = append(order, f.RelativePath)
	}

	// Override or add locale files
	for _, f := range localeFiles {
		if _, exists := merged[f.RelativePath]; !exists {
			order = append(order, f.RelativePath)
		}
		merged[f.RelativePath] = f
	}

	// Reconstruct slice preserving order
	result := make([]DownloadedFile, 0, len(merged))
	for _, p := range order {
		result = append(result, merged[p])
	}

	return result
}
