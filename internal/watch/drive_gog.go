package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jborkowski/lazy-notes/internal/excmd"
)

// DriveGogOptions configures Google Drive change polling via gog-cli.
type DriveGogOptions struct {
	GogBin      string
	Account     string
	FolderID    string // optional; when set, only trigger for changes under this parent
	StateFile   string
	Interval    time.Duration
	LookPath    func(string) (string, error) // optional; defaults to exec.LookPath
	CommandContext func(context.Context, string, ...string) *exec.Cmd
}

// WatchDriveGog polls Drive changes with `gog drive changes poll` and invokes
// onChange when a non-empty batch arrives (optionally filtered by FolderID).
func WatchDriveGog(ctx context.Context, opts DriveGogOptions, onChange func(reason string)) error {
	bin := strings.TrimSpace(opts.GogBin)
	if bin == "" {
		bin = "gog"
	}
	look := opts.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look(bin); err != nil {
		return fmt.Errorf("gog binary %q not found on PATH (brew install gogcli): %w", bin, err)
	}

	stateFile := opts.StateFile
	if stateFile == "" {
		return fmt.Errorf("drive changes state file is empty")
	}
	if err := os.MkdirAll(filepath.Dir(stateFile), 0o755); err != nil {
		return fmt.Errorf("ensure drive state dir: %w", err)
	}

	interval := opts.Interval
	if interval <= 0 {
		interval = 60 * time.Second
	}

	cmdCtx := opts.CommandContext
	if cmdCtx == nil {
		cmdCtx = exec.CommandContext
	}

	slog.Info("polling Google Drive changes via gog",
		"bin", bin,
		"folder_id", opts.FolderID,
		"state_file", stateFile,
		"interval", interval,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Immediate first poll, then on interval.
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		n, err := pollDriveChangesOnce(ctx, cmdCtx, bin, opts.Account, opts.FolderID, stateFile)
		if err != nil {
			slog.Warn("gog drive changes poll failed", "err", err)
		} else if n > 0 {
			onChange(fmt.Sprintf("drive-gog:%d", n))
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func pollDriveChangesOnce(
	ctx context.Context,
	cmdCtx func(context.Context, string, ...string) *exec.Cmd,
	bin, account, folderID, stateFile string,
) (int, error) {
	args := []string{
		"drive", "changes", "poll",
		"--state-file", stateFile,
		"--max-iterations", "1",
		"--interval", "0s",
		"--json",
		"--results-only",
	}
	if account != "" {
		args = append([]string{"--account", account}, args...)
	}

	stdout, err := excmd.RunCmd(cmdCtx(ctx, bin, args...))
	if err != nil {
		return 0, err
	}

	raw := bytes.TrimSpace(stdout)
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}

	changes, err := parseDriveChanges(raw)
	if err != nil {
		return 0, err
	}
	if folderID == "" {
		return len(changes), nil
	}
	matched := 0
	for _, ch := range changes {
		if changeMatchesFolder(ch, folderID) {
			matched++
		}
	}
	return matched, nil
}

type driveChange struct {
	FileID  string   `json:"fileId"`
	Removed bool     `json:"removed"`
	File    *struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Parents  []string `json:"parents"`
		MimeType string   `json:"mimeType"`
	} `json:"file"`
}

func parseDriveChanges(raw []byte) ([]driveChange, error) {
	// gog may emit a bare array, an object with "changes", or a single change.
	var asArray []driveChange
	if err := json.Unmarshal(raw, &asArray); err == nil {
		return asArray, nil
	}

	var envelope struct {
		Changes []driveChange `json:"changes"`
		Change  *driveChange  `json:"change"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode drive changes JSON: %w", err)
	}
	if len(envelope.Changes) > 0 {
		return envelope.Changes, nil
	}
	if envelope.Change != nil {
		return []driveChange{*envelope.Change}, nil
	}
	return nil, nil
}

func changeMatchesFolder(ch driveChange, folderID string) bool {
	if folderID == "" {
		return true
	}
	if ch.File == nil {
		return false
	}
	for _, p := range ch.File.Parents {
		if p == folderID {
			return true
		}
	}
	return false
}
