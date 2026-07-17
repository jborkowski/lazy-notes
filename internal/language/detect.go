package language

import (
	"strings"

	"github.com/abadojack/whatlanggo"
)

// Detect returns "pl", "en", "es" when confident from text; otherwise "auto".
// "auto" means: SuperWhisper mode with language=auto and the conditional prompt
// (three <if=...> blocks). No external audio LID.
func Detect(text string, audioPath string, allowed []string, fallback string) string {
	_ = audioPath // SW auto mode handles unknown audio language
	allowed = normalizeAllowedList(allowed)
	if fallback == "" {
		fallback = Auto
	}

	if strings.TrimSpace(text) != "" {
		if lang := detectFromText(text, allowed); lang != "" {
			return lang
		}
	}
	if fallback == "en" || fallback == "pl" || fallback == "es" {
		// Prefer explicit auto when unrecognized — caller maps via ModeKey("auto").
		return Auto
	}
	if Normalize(fallback) == Auto || fallback == Auto {
		return Auto
	}
	return Auto
}

// Auto is the routing key for the SuperWhisper fallback mode (conditional prompt).
const Auto = "auto"

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
		if !info.IsReliable() && info.Confidence < 0.5 {
			// try next / give up
		}
		if info.Lang < 0 {
			break
		}
		code := fromWhatlang(info.Lang)
		if code != "" && InAllowed(code, allowed) {
			if info.IsReliable() || info.Confidence >= 0.45 {
				return code
			}
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

// FromSuperWhisper maps SW language labels to pl|en|es|"".
func FromSuperWhisper(label string) string {
	return Normalize(label)
}
