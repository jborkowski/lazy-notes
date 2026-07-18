package hf

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ExtractAudioByID finds which shard contains recordingID and writes audio to destPath.
// Used to re-submit poisoned rows after KeepAudio deleted the local cache.
func (c *Client) ExtractAudioByID(ctx context.Context, recordingID int64, preferOriginal bool, destPath string, keepShards bool) error {
	if recordingID <= 0 {
		return fmt.Errorf("recording_id %d invalid", recordingID)
	}
	shards, err := c.ListShards(ctx)
	if err != nil {
		return err
	}
	var lastErr error
	for _, shard := range shards {
		local, err := c.EnsureShard(ctx, shard.Path)
		if err != nil {
			lastErr = err
			continue
		}
		err = c.ExtractAudio(ctx, local, recordingID, preferOriginal, destPath)
		if !keepShards {
			_ = os.Remove(local)
		}
		if err == nil {
			return nil
		}
		lastErr = err
		if strings.Contains(err.Error(), "not found") {
			continue
		}
		return err
	}
	if lastErr != nil {
		return fmt.Errorf("recording_id %d: %w", recordingID, lastErr)
	}
	return fmt.Errorf("recording_id %d not found in any shard", recordingID)
}
