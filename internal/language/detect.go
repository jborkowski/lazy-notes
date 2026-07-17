package language

import (
	"strings"

	"github.com/abadojack/whatlanggo"
)

// Text detection uses github.com/abadojack/whatlanggo (pure Go, trigram models).
//
// When text is empty and audioPath is set, audio language ID is not implemented yet.
// TODO: probe audio metadata (afinfo) or run whisper LID on audioPath.

// Detect returns pl, en, or es when possible, otherwise fallback.
// allowed is normalized to supported codes; empty fallback defaults to en.
func Detect(text string, audioPath string, allowed []string, fallback string) string {
	allowed = normalizeAllowedList(allowed)
	fallback = effectiveFallback(fallback)

	if strings.TrimSpace(text) != "" {
		if lang := detectFromText(text, allowed); lang != "" {
			return lang
		}
		return fallback
	}

	if strings.TrimSpace(audioPath) != "" {
		// Phase 1: no audio LID; whisper/afinfo can be wired here later.
		_ = audioPath
	}
	return fallback
}

func detectFromText(text string, allowed []string) string {
	whitelist := whatlangWhitelist(allowed)
	if len(whitelist) == 0 {
		return ""
	}

	blacklist := map[whatlanggo.Lang]bool{}
	for range supportedWhatlangLangs {
		opts := whatlanggo.Options{
			Whitelist: whitelist,
			Blacklist: blacklist,
		}
		info := whatlanggo.DetectWithOptions(text, opts)
		if info.Lang < 0 {
			break
		}

		code := fromWhatlang(info.Lang)
		if code != "" && InAllowed(code, allowed) {
			return code
		}

		blacklist[info.Lang] = true
	}
	return ""
}

var supportedWhatlangLangs = []whatlanggo.Lang{
	whatlanggo.Pol,
	whatlanggo.Eng,
	whatlanggo.Spa,
}

func whatlangWhitelist(allowed []string) map[whatlanggo.Lang]bool {
	wl := make(map[whatlanggo.Lang]bool)
	for _, a := range allowed {
		switch Normalize(a) {
		case "pl":
			wl[whatlanggo.Pol] = true
		case "en":
			wl[whatlanggo.Eng] = true
		case "es":
			wl[whatlanggo.Spa] = true
		}
	}
	return wl
}

func fromWhatlang(lang whatlanggo.Lang) string {
	switch lang {
	case whatlanggo.Pol:
		return "pl"
	case whatlanggo.Eng:
		return "en"
	case whatlanggo.Spa:
		return "es"
	default:
		return Normalize(lang.Iso6391())
	}
}
