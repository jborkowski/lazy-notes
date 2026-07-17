package hf

// Meta holds recording metadata scanned from a parquet shard.
type Meta struct {
	RecordingID     int64
	Title           string
	CreatedAt       string
	DurationSeconds float64
	Transcription   string
	AudioPath       string
	PreferOriginal  bool
}

// Shard describes one parquet file in the dataset repo.
type Shard struct {
	Path string
	Size int64
	OID  string // git blob oid from Hub tree API
}

// Client accesses a HuggingFace dataset via the Hub API and DuckDB CLI.
type Client struct {
	Repo     string // e.g. j14i/voice-memories
	Token    string
	CacheDir string
}
