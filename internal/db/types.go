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

type Recording struct {
	RecordingID int64
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
