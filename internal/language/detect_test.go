package language_test

import (
	"testing"

	"github.com/jborkowski/lazy-notes/internal/language"
)

func TestDetectPolishEnglishSpanishOrAuto(t *testing.T) {
	allowed := []string{"pl", "en", "es"}

	pl := language.Detect("To jest krótka notatka po polsku o spotkaniu z szefem.", "", allowed, "auto")
	if pl != "pl" {
		t.Fatalf("polish: got %q want pl", pl)
	}

	en := language.Detect("This is a short English voice note about the meeting.", "", allowed, "auto")
	if en != "en" {
		t.Fatalf("english: got %q want en", en)
	}

	es := language.Detect("Esta es una nota de voz corta en español sobre la reunión.", "", allowed, "auto")
	if es != "es" {
		t.Fatalf("spanish: got %q want es", es)
	}

	unk := language.Detect("", "/tmp/x.m4a", allowed, "auto")
	if unk != language.Auto {
		t.Fatalf("unknown audio: got %q want auto", unk)
	}
}
