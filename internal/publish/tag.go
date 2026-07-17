package publish

import "strings"

// DefaultTag is applied to published notes when config leaves tag unset.
const DefaultTag = "#lazy-notes"

// normalizeTag returns a hashtag form (e.g. "#lazy-notes"), or "" if empty.
func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	if !strings.HasPrefix(tag, "#") {
		tag = "#" + tag
	}
	return tag
}

// tagWithoutHash returns the tag name without a leading '#'.
func tagWithoutHash(tag string) string {
	return strings.TrimPrefix(normalizeTag(tag), "#")
}

// withTag appends tag to body if missing. Empty tag is a no-op.
func withTag(body, tag string) string {
	tag = normalizeTag(tag)
	if tag == "" {
		return body
	}
	if strings.Contains(body, tag) {
		return body
	}
	body = strings.TrimRight(body, "\n \t")
	if body == "" {
		return tag + "\n"
	}
	return body + "\n\n" + tag + "\n"
}
