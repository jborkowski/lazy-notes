package superwhisper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jborkowski/lazy-notes/internal/config"
)

// InstallModes writes SuperWhisper mode JSON files for each configured language
// and merges their keys into settings.json modeKeys.
func InstallModes(cfg *config.Config, configDir string, force bool) ([]string, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if err := os.MkdirAll(ModesDir(), 0o755); err != nil {
		return nil, fmt.Errorf("create modes dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(SettingsPath()), 0o755); err != nil {
		return nil, fmt.Errorf("create settings dir: %w", err)
	}

	var written []string
	modeKeys := make([]string, 0, len(cfg.Languages))

	for _, lang := range cfg.Languages {
		key := cfg.ModeKey(lang)
		modeKeys = append(modeKeys, key)

		prompt, err := cfg.ResolvePrompt(lang, configDir)
		if err != nil {
			return written, fmt.Errorf("resolve prompt %q: %w", lang, err)
		}

		modePath := filepath.Join(ModesDir(), key+".json")
		if !force {
			if _, err := os.Stat(modePath); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return written, fmt.Errorf("stat mode %q: %w", modePath, err)
			}
		}

		mode := config.BuildModeJSON(
			key,
			modeDisplayName(lang),
			lang,
			cfg.VoiceModel(lang),
			cfg.LanguageModel(lang),
			prompt,
		)

		data, err := json.MarshalIndent(mode, "", "  ")
		if err != nil {
			return written, fmt.Errorf("marshal mode %q: %w", key, err)
		}
		data = append(data, '\n')

		if err := os.WriteFile(modePath, data, 0o644); err != nil {
			return written, fmt.Errorf("write mode %q: %w", modePath, err)
		}
		written = append(written, modePath)
	}

	if err := mergeModeKeys(SettingsPath(), modeKeys); err != nil {
		return written, fmt.Errorf("merge mode keys: %w", err)
	}

	return written, nil
}

func modeDisplayName(lang string) string {
	return fmt.Sprintf("Lazy Note %s", strings.ToUpper(lang))
}

func mergeModeKeys(settingsPath string, keys []string) error {
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	existing := modeKeysFromSettings(settings)
	seen := make(map[string]struct{}, len(existing)+len(keys))
	for _, key := range existing {
		seen[key] = struct{}{}
	}

	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		existing = append(existing, key)
		seen[key] = struct{}{}
	}

	settings["modeKeys"] = existing
	return writeSettings(settingsPath, settings)
}

func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read settings %q: %w", path, err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse settings %q: %w", path, err)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]any) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write settings %q: %w", path, err)
	}
	return nil
}

func modeKeysFromSettings(settings map[string]any) []string {
	raw, ok := settings["modeKeys"]
	if !ok || raw == nil {
		return nil
	}

	switch keys := raw.(type) {
	case []string:
		return append([]string(nil), keys...)
	case []any:
		out := make([]string, 0, len(keys))
		for _, item := range keys {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
