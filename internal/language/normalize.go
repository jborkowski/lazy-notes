package language

import "strings"

var codeAliases = map[string]string{
	"pl":       "pl",
	"pol":      "pl",
	"polish":   "pl",
	"en":       "en",
	"eng":      "en",
	"english":  "en",
	"es":       "es",
	"spa":      "es",
	"spanish":  "es",
	"espanol":  "es",
	"español":  "es",
}

// Normalize maps language names and ISO codes to pl, en, or es.
// Unknown codes return an empty string.
func Normalize(code string) string {
	s := strings.ToLower(strings.TrimSpace(code))
	if s == "" {
		return ""
	}
	if v, ok := codeAliases[s]; ok {
		return v
	}
	return ""
}

// InAllowed reports whether lang matches any entry in allowed after normalization.
func InAllowed(lang string, allowed []string) bool {
	n := Normalize(lang)
	if n == "" {
		return false
	}
	for _, a := range allowed {
		if Normalize(a) == n {
			return true
		}
	}
	return false
}

func normalizeAllowedList(allowed []string) []string {
	if len(allowed) == 0 {
		return []string{"pl", "en", "es"}
	}
	seen := make(map[string]struct{}, len(allowed))
	out := make([]string, 0, len(allowed))
	for _, a := range allowed {
		n := Normalize(a)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	if len(out) == 0 {
		return []string{"pl", "en", "es"}
	}
	return out
}

func effectiveFallback(fallback string) string {
	if n := Normalize(fallback); n != "" {
		return n
	}
	return "en"
}
