package scaffold

import (
	"strings"
	"unicode"
)

// DefaultNGramN is the character n-gram length for Jaccard similarity.
// Fixed by TestSimilarity_FixtureDispersion (must stay in sync with that test).
const DefaultNGramN = 5

// NormalizeForSimilarity normalizes text before n-gram extraction.
func NormalizeForSimilarity(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// CharNGrams builds a set of character n-grams (rune-based) from s.
func CharNGrams(s string, n int) map[string]struct{} {
	out := make(map[string]struct{})
	if n <= 0 {
		return out
	}
	runes := []rune(NormalizeForSimilarity(s))
	if len(runes) < n {
		if len(runes) > 0 {
			out[string(runes)] = struct{}{}
		}
		return out
	}
	for i := 0; i <= len(runes)-n; i++ {
		out[string(runes[i:i+n])] = struct{}{}
	}
	return out
}

// Jaccard returns |A∩B| / |A∪B|. Empty∩empty → 1; one empty → 0.
func Jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// SimilarityScore returns character n-gram Jaccard between left and right.
func SimilarityScore(left, right string, n int) float64 {
	return Jaccard(CharNGrams(left, n), CharNGrams(right, n))
}

// CollapseSpace collapses unicode spaces to a single space (optional helper for tests).
func CollapseSpace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
