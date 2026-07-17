package db

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusSubmitted  Status = "submitted"
	StatusHarvested  Status = "harvested"
	StatusPublished  Status = "published"
	StatusError      Status = "error"
)

// Recording sources. HF and Voice Memos share the status machine but not ID space.
const (
	SourceHF        = "hf"
	SourceVoiceMemo = "voicememo"
)

type Recording struct {
	RecordingID int64
	Source      string // hf | voicememo
	ExternalID  string // Voice Memo stable key; empty for HF
	CreatedAt   string
	Title       string
	AudioPath   string
	Language    string
	ModeKey     string
	Status      Status
	SubmittedAt *time.Time
	Error       string
	SwID        string
	NotePath    string
	PublishedAt *time.Time
	Body        string
}
