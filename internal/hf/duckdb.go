package hf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// duckdbJSON runs `duckdb -json -c "<sql>"` and returns stdout.
func duckdbJSON(ctx context.Context, sql string) ([]byte, error) {
	out, err := duckdbRun(ctx, "-json", "-c", sql)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// duckdbCSV runs `duckdb -csv -noheader -c "<sql>"` and returns stdout (one cell per row typical).
func duckdbCSV(ctx context.Context, sql string) ([]byte, error) {
	return duckdbRun(ctx, "-csv", "-noheader", "-c", sql)
}

func duckdbRun(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "duckdb", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("duckdb: %s", msg)
	}
	return out, nil
}

func duckdbQueryRows(ctx context.Context, sql string, dest any) error {
	out, err := duckdbJSON(ctx, sql)
	if err != nil {
		return err
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return nil
	}
	if err := json.Unmarshal(out, dest); err != nil {
		return fmt.Errorf("duckdb json: %w", err)
	}
	return nil
}
