// Package voicememos scans the Voice Memos export inbox and stages audio for SuperWhisper.
package voicememos

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Item is one inbox audio file ready to upsert.
type Item struct {
	Path       string
	ExternalID string
	Title      string
	Ext        string
	ModTime    time.Time
	Duration   float64 // seconds; 0 if unknown
}

// ScanOptions controls inbox scanning.
type ScanOptions struct {
	Dir                string
	Extensions         []string
	MinDurationSeconds float64
	// DurationOf optionally measures audio duration; nil skips the min-duration filter.
	DurationOf func(path string) (float64, error)
}

// Scan lists supported audio files in dir (non-recursive).
func Scan(opts ScanOptions) ([]Item, error) {
	dir := strings.TrimSpace(opts.Dir)
	if dir == "" {
		return nil, fmt.Errorf("voice memos export dir is empty")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read voice memos inbox %q: %w", dir, err)
	}

	extOK := extensionSet(opts.Extensions)
	var out []Item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !extOK[ext] {
			continue
		}
		path := filepath.Join(dir, name)
		info, err := e.Info()
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", path, err)
		}

		var dur float64
		if opts.DurationOf != nil && opts.MinDurationSeconds > 0 {
			d, err := opts.DurationOf(path)
			if err == nil {
				dur = d
				if dur < opts.MinDurationSeconds {
					continue
				}
			}
		}

		externalID, err := ContentHash(path)
		if err != nil {
			return nil, err
		}
		out = append(out, Item{
			Path:       path,
			ExternalID: externalID,
			Title:      titleFromName(name),
			Ext:        ext,
			ModTime:    info.ModTime(),
			Duration:   dur,
		})
	}
	return out, nil
}

func extensionSet(exts []string) map[string]bool {
	out := make(map[string]bool)
	if len(exts) == 0 {
		out[".m4a"] = true
		return out
	}
	for _, e := range exts {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out[e] = true
	}
	return out
}

func titleFromName(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.TrimSpace(base)
	if base == "" {
		return "Voice Memo"
	}
	return base
}

// ContentHash returns a stable external_id for an inbox file (sha256 hex of contents).
func ContentHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %q: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Stage copies src into destDir as {externalID}{ext}.
func Stage(src, destDir, externalID, ext string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("ensure audio cache: %w", err)
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	dest := filepath.Join(destDir, externalID+ext)
	if err := copyFile(src, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source %q: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create staged %q: %w", dest, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy to %q: %w", dest, err)
	}
	return out.Close()
}
