package publish

import "testing"

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"#lazy-notes", "#lazy-notes"},
		{"lazy-notes", "#lazy-notes"},
		{"  #lazy-notes  ", "#lazy-notes"},
	}
	for _, tt := range tests {
		if got := normalizeTag(tt.in); got != tt.want {
			t.Fatalf("normalizeTag(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWithTag(t *testing.T) {
	got := withTag("Hello", "#lazy-notes")
	want := "Hello\n\n#lazy-notes\n"
	if got != want {
		t.Fatalf("withTag = %q, want %q", got, want)
	}

	already := "Hello\n\n#lazy-notes\n"
	if got := withTag(already, "#lazy-notes"); got != already {
		t.Fatalf("withTag should not duplicate, got %q", got)
	}

	if got := withTag("Hello", ""); got != "Hello" {
		t.Fatalf("empty tag should be no-op, got %q", got)
	}
}
