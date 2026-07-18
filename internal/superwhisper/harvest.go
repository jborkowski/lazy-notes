package superwhisper

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHarvestLimit = 20
	createdAtLayout     = "2006-01-02 15:04:05.000"
	submitSkew          = 60 * time.Second
)

// Transcript is one SuperWhisper history entry used after file submit.
type Transcript struct {
	ID         string
	FolderName string
	ModeName   string
	RawResult  string
	LLMResult  string
	CreatedAt  string
	FromFile   bool
	Duration   int
}

// BestText returns llmResult if non-empty else rawResult.
func (t Transcript) BestText() string {
	if strings.TrimSpace(t.LLMResult) != "" {
		return t.LLMResult
	}
	return t.RawResult
}

type historyEntry struct {
	ID         string `json:"id"`
	FolderName string `json:"folderName"`
	ModeName   string `json:"modeName"`
	RawResult  string `json:"rawResult"`
	LLMResult  string `json:"llmResult"`
	CreatedAt  string `json:"createdAt"`
	Datetime   string `json:"datetime"`
	FromFile   bool   `json:"fromFile"`
	Duration   int    `json:"duration"`
}

func (e historyEntry) transcript() Transcript {
	createdAt := e.CreatedAt
	if createdAt == "" {
		createdAt = e.Datetime
	}
	return Transcript{
		ID:         e.ID,
		FolderName: e.FolderName,
		ModeName:   e.ModeName,
		RawResult:  e.RawResult,
		LLMResult:  e.LLMResult,
		CreatedAt:  createdAt,
		FromFile:   e.FromFile,
		Duration:   e.Duration,
	}
}

// HarvestRecent runs `superwhisper history --json -l <limit>` (use CLIPath()),
// parses JSON array, returns fromFile==true items with non-empty BestText(),
// newest first. limit default 20.
func HarvestRecent(ctx context.Context, limit int) ([]Transcript, error) {
	if limit <= 0 {
		limit = defaultHarvestLimit
	}

	cli, ok := CLIPath()
	if !ok {
		return nil, fmt.Errorf("superwhisper CLI not found")
	}

	cmd := exec.CommandContext(ctx, cli, "history", "--json", "-l", strconv.Itoa(limit))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("superwhisper history: %w", err)
	}

	var entries []historyEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("parse superwhisper history: %w", err)
	}

	outList := make([]Transcript, 0, len(entries))
	for _, entry := range entries {
		t := entry.transcript()
		if !t.FromFile {
			continue
		}
		if strings.TrimSpace(t.BestText()) == "" {
			continue
		}
		outList = append(outList, t)
	}
	return outList, nil
}

// MatchRecording tries to match a submitted local recording to a transcript:
// prefer same modeName contains "Lazy Note", fromFile true, recent CreatedAt,
// and optional duration proximity. Return best match or nil.
// exclude is a set of SuperWhisper IDs already claimed by other recordings;
// those transcripts are never reused (prevents backlog clamp onto one result).
func MatchRecording(transcripts []Transcript, modeKey string, submittedAt time.Time, exclude map[string]struct{}) *Transcript {
	if len(transcripts) == 0 {
		return nil
	}

	expectedMode := expectedModeName(modeKey)
	earliest := submittedAt.UTC().Add(-5 * time.Minute)

	var best *Transcript
	bestScore := -1

	for i := range transcripts {
		t := &transcripts[i]
		if !t.FromFile || strings.TrimSpace(t.BestText()) == "" {
			continue
		}
		if _, taken := exclude[strings.TrimSpace(t.ID)]; taken {
			continue
		}

		created, ok := parseCreatedAt(t.CreatedAt)
		if !ok || created.UTC().Before(earliest) {
			continue
		}

		score := matchScore(t, expectedMode, created.UTC(), submittedAt.UTC())
		if score < 0 {
			continue
		}
		if best == nil || score > bestScore {
			best = t
			bestScore = score
		}
	}

	return best
}

func matchScore(t *Transcript, expectedMode string, created, submittedAt time.Time) int {
	if !strings.Contains(t.ModeName, "Lazy Note") {
		return -1
	}

	score := 0
	if expectedMode != "" && strings.EqualFold(strings.TrimSpace(t.ModeName), expectedMode) {
		score += 200
	} else if expectedMode != "" && strings.Contains(strings.ToUpper(t.ModeName), langSuffix(expectedMode)) {
		score += 120
	} else {
		score += 60
	}

	delta := created.Sub(submittedAt)
	if delta < 0 {
		delta = -delta
	}
	if delta <= 30*time.Second {
		score += 80
	} else if delta <= 2*time.Minute {
		score += 50
	} else if delta <= 10*time.Minute {
		score += 20
	} else if delta <= 2*time.Hour {
		score += 5
	}

	if t.Duration > 0 {
		score += 1
	}

	return score
}

func expectedModeName(modeKey string) string {
	key := strings.ToLower(strings.TrimSpace(modeKey))
	if key == "lazy-note-auto" || strings.HasSuffix(key, "-auto") || key == "auto" {
		return "Lazy Note"
	}
	lang := langSuffix(modeKey)
	if lang == "" {
		return "Lazy Note"
	}
	return fmt.Sprintf("Lazy Note %s", lang)
}

func langSuffix(key string) string {
	parts := strings.Split(strings.ToLower(key), "-")
	if len(parts) == 0 {
		return ""
	}
	suffix := parts[len(parts)-1]
	switch suffix {
	case "pl", "en", "es":
		return strings.ToUpper(suffix)
	default:
		return ""
	}
}

func parseCreatedAt(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}

	layouts := []string{
		createdAtLayout,
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	// SuperWhisper history timestamps are UTC without a zone suffix.
	for _, layout := range layouts {
		if ts, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

// WaitAndHarvest sleeps briefly then polls HarvestRecent until MatchRecording
// finds a transcript or polls are exhausted.
// exclude lists SuperWhisper IDs already claimed; they are skipped every poll.
func WaitAndHarvest(ctx context.Context, modeKey string, submittedAt time.Time, wait time.Duration, polls int, exclude map[string]struct{}) (*Transcript, error) {
	if polls <= 0 {
		polls = 1
	}
	if wait < 0 {
		wait = 0
	}

	if err := sleep(ctx, wait); err != nil {
		return nil, err
	}

	for attempt := 0; attempt < polls; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		transcripts, err := HarvestRecent(ctx, defaultHarvestLimit)
		if err != nil {
			return nil, err
		}
		if match := MatchRecording(transcripts, modeKey, submittedAt, exclude); match != nil {
			return match, nil
		}

		if attempt+1 < polls {
			if err := sleep(ctx, wait); err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
