package hf

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// ExtractAudio writes the audio blob for recordingID from localParquet to destPath.
// When preferOriginal is true, audio_original.bytes is used; otherwise audio.bytes.
//
// DuckDB SQL (hex fallback; FORMAT RAW is unavailable in current DuckDB CLI):
//
//	SELECT hex(CASE WHEN prefer_original THEN audio_original.bytes ELSE audio.bytes END)
//	FROM read_parquet(...) WHERE recording_id = ?
func (c *Client) ExtractAudio(ctx context.Context, localParquet string, recordingID int64, preferOriginal bool, destPath string) error {
	col := "audio.bytes"
	if preferOriginal {
		col = "audio_original.bytes"
	}
	sql := fmt.Sprintf(`SELECT hex(%s) FROM read_parquet(%s) WHERE recording_id = %d`,
		col, sqlStringLiteral(localParquet), recordingID)

	out, err := duckdbCSV(ctx, sql)
	if err != nil {
		return err
	}
	hexStr := strings.TrimSpace(string(out))
	if hexStr == "" {
		return fmt.Errorf("recording_id %d not found in %s", recordingID, localParquet)
	}

	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return fmt.Errorf("decode audio hex: %w", err)
	}

	if err := ensureParentDir(destPath); err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0o644)
}
