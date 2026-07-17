package hf

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultToken reads a Hugging Face token for private datasets.
// Order:
//  1. HF_TOKEN / HUGGING_FACE_HUB_TOKEN
//  2. HF_TOKEN_PATH (file)
//  3. ~/.config/lazy-notes/hf_token  (canonical for brew service / daemon)
//  4. $HF_HOME/token and other hf CLI locations
func DefaultToken() string {
	for _, key := range []string{"HF_TOKEN", "HUGGING_FACE_HUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if p := strings.TrimSpace(os.Getenv("HF_TOKEN_PATH")); p != "" {
		if v := readTokenFile(p); v != "" {
			return v
		}
	}
	for _, p := range tokenFileCandidates() {
		if v := readTokenFile(p); v != "" {
			return v
		}
	}
	return ""
}

func tokenFileCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	var out []string
	if home != "" {
		// Canonical lazy-notes path first — brew services set HF_HOME and must
		// not steal precedence from the user's ~/.config/lazy-notes/hf_token.
		out = append(out, filepath.Join(home, ".config", "lazy-notes", "hf_token"))
	}
	if hfHome := strings.TrimSpace(os.Getenv("HF_HOME")); hfHome != "" {
		out = append(out, filepath.Join(hfHome, "token"))
	}
	if home != "" {
		out = append(out,
			filepath.Join(home, ".config", "huggingface", "token"),
			filepath.Join(home, ".cache", "huggingface", "token"),
			filepath.Join(home, ".config", "cache", "huggingface", "token"),
			filepath.Join(home, ".huggingface", "token"),
		)
	}
	return out
}

func readTokenFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// stored_tokens style is INI; plain token files are a single line.
	text := strings.TrimSpace(string(b))
	if text == "" {
		return ""
	}
	if !strings.Contains(text, "\n") && !strings.Contains(text, "=") {
		return text
	}
	// Prefer first hf_… token-looking line.
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "hf_") {
			return line
		}
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			val := strings.TrimSpace(parts[1])
			if strings.HasPrefix(val, "hf_") {
				return val
			}
		}
	}
	return ""
}

// TokenSource describes where DefaultToken found a credential (never the secret).
func TokenSource() string {
	for _, key := range []string{"HF_TOKEN", "HUGGING_FACE_HUB_TOKEN"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return "env:" + key
		}
	}
	if p := strings.TrimSpace(os.Getenv("HF_TOKEN_PATH")); p != "" {
		if readTokenFile(p) != "" {
			return "file:" + p
		}
	}
	for _, p := range tokenFileCandidates() {
		if readTokenFile(p) != "" {
			return "file:" + p
		}
	}
	return ""
}
