package publish

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/jborkowski/lazy-notes/internal/paths"
)

// WriteMarkdown writes n to notesDir as {recording_id}-{slug-title}.md with YAML frontmatter.
func WriteMarkdown(notesDir string, n Note) (path string, err error) {
	dir, err := expandPath(notesDir)
	if err != nil {
		return "", err
	}
	if err := paths.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("ensure notes dir %q: %w", dir, err)
	}

	title := ResolveTitle(n)
	name := fmt.Sprintf("%d-%s.md", n.RecordingID, slugTitle(title))
	path = filepath.Join(dir, name)

	content := buildMarkdownFile(n, time.Now().UTC())
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write note %q: %w", path, err)
	}
	return path, nil
}

func buildMarkdownFile(n Note, created time.Time) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: %d\n", n.RecordingID))
	if n.Language != "" {
		b.WriteString(fmt.Sprintf("language: %q\n", n.Language))
	}
	if n.ModeKey != "" {
		b.WriteString(fmt.Sprintf("mode: %q\n", n.ModeKey))
	}
	if n.SourceSW != "" {
		b.WriteString(fmt.Sprintf("sw_id: %q\n", n.SourceSW))
	}
	b.WriteString(fmt.Sprintf("created: %q\n", created.Format(time.RFC3339)))
	b.WriteString("---\n")
	if body := strings.TrimSpace(n.Body); body != "" {
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		b.WriteString("\n")
		b.WriteString(body)
	}
	return b.String()
}

func slugTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	prevDash := false
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "note"
	}
	const maxLen = 60
	if len(slug) > maxLen {
		slug = strings.Trim(slug[:maxLen], "-")
	}
	if slug == "" {
		return "note"
	}
	return slug
}

func expandPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("notes dir is empty")
	}
	if len(p) > 0 && p[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand path %q: %w", p, err)
		}
		return filepath.Join(home, p[1:]), nil
	}
	return p, nil
}
