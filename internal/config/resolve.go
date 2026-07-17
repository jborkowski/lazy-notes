package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func expandPath(p string) string {
	if len(p) > 0 && p[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[1:])
	}
	return p
}

func (m ModesConfig) forLang(lang string) string {
	switch lang {
	case "pl":
		return m.PL
	case "en":
		return m.EN
	case "es":
		return m.ES
	case "auto":
		return m.Auto
	default:
		return ""
	}
}

func (m ModelConfig) override(lang string) *LangModelOverride {
	switch lang {
	case "pl":
		if m.PL.VoiceModelID != "" || m.PL.LanguageModelID != "" {
			return &m.PL
		}
	case "en":
		if m.EN.VoiceModelID != "" || m.EN.LanguageModelID != "" {
			return &m.EN
		}
	case "es":
		if m.ES.VoiceModelID != "" || m.ES.LanguageModelID != "" {
			return &m.ES
		}
	}
	return nil
}

func (p PromptsConfig) forLang(lang string) (PromptSpec, bool) {
	switch lang {
	case "pl":
		if p.PL.File != "" || p.PL.Text != "" {
			return p.PL, true
		}
	case "en":
		if p.EN.File != "" || p.EN.Text != "" {
			return p.EN, true
		}
	case "es":
		if p.ES.File != "" || p.ES.Text != "" {
			return p.ES, true
		}
	case "auto":
		if p.Auto.File != "" || p.Auto.Text != "" {
			return p.Auto, true
		}
	}
	return PromptSpec{}, false
}

// ModeKey returns the SuperWhisper mode key for lang, or the fallback (auto) key.
func (c *Config) ModeKey(lang string) string {
	if lang == "auto" {
		if c.Modes.Auto != "" {
			return c.Modes.Auto
		}
		return c.Modes.Fallback
	}
	if key := c.Modes.forLang(lang); key != "" {
		return key
	}
	if c.Modes.Auto != "" {
		return c.Modes.Auto
	}
	return c.Modes.Fallback
}

// VoiceModel returns the voice model for lang (per-language override, then global).
func (c *Config) VoiceModel(lang string) string {
	if ov := c.Model.override(lang); ov != nil && ov.VoiceModelID != "" {
		return ov.VoiceModelID
	}
	return c.Model.VoiceModelID
}

// LanguageModel returns the language model for lang (per-language override, then global).
func (c *Config) LanguageModel(lang string) string {
	if ov := c.Model.override(lang); ov != nil && ov.LanguageModelID != "" {
		return ov.LanguageModelID
	}
	return c.Model.LanguageModelID
}

// NotesDir returns the expanded post-processed notes directory.
func (c *Config) NotesDir() string {
	return expandPath(c.Publish.NotesDir)
}

// MemoBin returns the memo CLI binary name or path.
func (c *Config) MemoBin() string {
	return c.Publish.MemoBin
}

// ResolvePrompt returns prompt markdown for lang from inline Text or a file under configDir.
func (c *Config) ResolvePrompt(lang string, configDir string) (string, error) {
	spec, ok := c.Prompts.forLang(lang)
	if !ok {
		return "", fmt.Errorf("no prompt configured for language %q", lang)
	}
	if spec.Text != "" {
		return spec.Text, nil
	}
	if spec.File == "" {
		return "", fmt.Errorf("prompt for language %q has neither file nor text", lang)
	}

	path := spec.File
	if !filepath.IsAbs(path) {
		path = filepath.Join(configDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt %q: %w", path, err)
	}
	return string(data), nil
}
