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
	name := fmt.Sprintf("%s-%s.md", recordingFileID(n.RecordingID), slugTitle(title))
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
	if tagName := tagWithoutHash(n.Tag); tagName != "" {
		b.WriteString(fmt.Sprintf("tags: [%q]\n", tagName))
	}
	b.WriteString(fmt.Sprintf("created: %q\n", created.Format(time.RFC3339)))
	b.WriteString("---\n")
	body := withTag(n.Body, n.Tag)
	if body = strings.TrimSpace(body); body != "" {
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		b.WriteString("\n")
		b.WriteString(body)
	}
	return b.String()
}

// recordingFileID formats the note filename id. Voice Memo negatives become vm{N}.
func recordingFileID(id int64) string {
	if id < 0 {
		return fmt.Sprintf("vm%d", -id)
	}
	return fmt.Sprintf("%d", id)
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
	return paths.Expand(p), nil
}
