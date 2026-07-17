package paths

import (
	"os"
	"path/filepath"
)

const AppName = "lazy-notes"

func ConfigPath() string {
	if v := os.Getenv("LAZY_NOTES_CONFIG"); v != "" {
		return expand(v)
	}
	return filepath.Join(home(), ".config", AppName, "config.toml")
}

func ConfigDir() string {
	return filepath.Dir(ConfigPath())
}

func StateDir() string {
	if v := os.Getenv("LAZY_NOTES_STATE_DIR"); v != "" {
		return expand(v)
	}
	return filepath.Join(home(), ".local", "share", AppName)
}

func DBPath() string {
	return filepath.Join(StateDir(), "state.sqlite")
}

func CacheDir() string {
	if v := os.Getenv("LAZY_NOTES_CACHE_DIR"); v != "" {
		return expand(v)
	}
	return filepath.Join(home(), ".cache", AppName)
}

func AudioCacheDir() string {
	return filepath.Join(CacheDir(), "audio")
}

// DataDir returns the directory that contains config.example.toml and prompts/.
// Order: LAZY_NOTES_DATA_DIR, then search near the executable / cwd for a config/ tree.
func DataDir() string {
	if v := os.Getenv("LAZY_NOTES_DATA_DIR"); v != "" {
		return expand(v)
	}
	if dir, ok := findDataDir(); ok {
		return dir
	}
	return filepath.Join(home(), ".config", AppName)
}

func ExampleConfigName() string { return "config.example.toml" }

func findDataDir() (string, bool) {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "config"),
			filepath.Join(exeDir, "..", "config"),
			filepath.Join(exeDir, "..", "..", "config"),
			filepath.Join(exeDir, "..", "share", AppName, "config"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "config"),
			filepath.Join(wd, "..", "config"),
		)
	}
	for _, c := range candidates {
		example := filepath.Join(c, ExampleConfigName())
		if st, err := os.Stat(example); err == nil && !st.IsDir() {
			abs, err := filepath.Abs(c)
			if err != nil {
				return c, true
			}
			return abs, true
		}
	}
	return "", false
}

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func expand(p string) string {
	return Expand(p)
}

// Expand resolves a leading ~ to the user home directory.
func Expand(p string) string {
	if p == "" {
		return ""
	}
	if p == "~" {
		return home()
	}
	if len(p) > 0 && p[0] == '~' {
		return filepath.Join(home(), p[1:])
	}
	return p
}

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
