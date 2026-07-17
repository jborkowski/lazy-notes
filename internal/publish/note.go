package publish

import (
	"fmt"
	"strings"
)

// Note is a harvested voice recording ready for export.
type Note struct {
	Title       string
	Body        string
	RecordingID int64
	Language    string
	ModeKey     string
	SourceSW    string
	// Tag is appended to the note body (e.g. "#lazy-notes") for search in
	// markdown files and Apple Notes.
	Tag string
}

// ResolveTitle returns an explicit title or derives one from the note body.
func ResolveTitle(n Note) string {
	if t := strings.TrimSpace(n.Title); t != "" {
		return t
	}
	if t := firstLine(n.Body); t != "" {
		return t
	}
	if n.RecordingID < 0 {
		return fmt.Sprintf("Voice memo %d", -n.RecordingID)
	}
	return fmt.Sprintf("Voice note %d", n.RecordingID)
}

func firstLine(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return strings.TrimSpace(body[:i])
	}
	return body
}
