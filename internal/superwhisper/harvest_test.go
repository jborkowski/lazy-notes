package superwhisper

import (
	"testing"
	"time"
)

func TestMatchRecordingExcludesClaimedSwID(t *testing.T) {
	submitted := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	shared := Transcript{
		ID:        "5262f3959be449a0aef6943f78cebf56",
		ModeName:  "Lazy Note",
		RawResult: "reflash my ZMK keyboard firmware please",
		CreatedAt: "2026-07-18 12:00:10.000",
		FromFile:  true,
	}
	other := Transcript{
		ID:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ModeName:  "Lazy Note",
		RawResult: "different note about lunch",
		CreatedAt: "2026-07-18 12:00:20.000",
		FromFile:  true,
	}

	got := MatchRecording([]Transcript{shared, other}, "lazy-note-auto", submitted, nil)
	if got == nil || got.ID != shared.ID {
		t.Fatalf("without exclude matched %#v, want shared id", got)
	}

	exclude := map[string]struct{}{shared.ID: {}}
	got = MatchRecording([]Transcript{shared, other}, "lazy-note-auto", submitted, exclude)
	if got == nil || got.ID != other.ID {
		t.Fatalf("with exclude matched %#v, want other id", got)
	}

	got = MatchRecording([]Transcript{shared}, "lazy-note-auto", submitted, exclude)
	if got != nil {
		t.Fatalf("expected nil when only claimed transcript remains, got %#v", got)
	}
}

func TestMatchRecordingSkipsEmptyBody(t *testing.T) {
	submitted := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	empty := Transcript{
		ID:        "empty",
		ModeName:  "Lazy Note",
		RawResult: "   ",
		CreatedAt: "2026-07-18 12:00:10.000",
		FromFile:  true,
	}
	if got := MatchRecording([]Transcript{empty}, "lazy-note-auto", submitted, nil); got != nil {
		t.Fatalf("expected nil for empty body, got %#v", got)
	}
}
