package hf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

type treeEntry struct {
	Type string `json:"type"`
	OID  string `json:"oid"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

// ListShards lists data/*.parquet files via the Hub tree API.
func (c *Client) ListShards(ctx context.Context) ([]Shard, error) {
	url := fmt.Sprintf("%s/api/datasets/%s/tree/main/data", hubBase, c.Repo)
	req, err := c.hubRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, hubError(resp)
	}

	var entries []treeEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode tree: %w", err)
	}

	var shards []Shard
	for _, e := range entries {
		if e.Type != "file" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Path), ".parquet") {
			continue
		}
		shards = append(shards, Shard{
			Path: e.Path,
			Size: e.Size,
			OID:  e.OID,
		})
	}
	sort.Slice(shards, func(i, j int) bool {
		return shards[i].Path > shards[j].Path
	})
	return shards, nil
}

// EnsureShard downloads shardPath from the Hub resolve URL when missing from CacheDir.
func (c *Client) EnsureShard(ctx context.Context, shardPath string) (string, error) {
	local := c.cachePath(shardPath)
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}

	url := fmt.Sprintf("%s/datasets/%s/resolve/main/%s", hubBase, c.Repo, shardPath)
	req, err := c.hubRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", hubError(resp)
	}

	if err := writeFileAtomic(local, resp.Body); err != nil {
		return "", err
	}
	return local, nil
}

func hubError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("huggingface api %s (private dataset — set HF_TOKEN or run: hf auth login): %s", resp.Status, msg)
	default:
		return fmt.Errorf("huggingface api: %s", msg)
	}
}

// VerifyAccess checks that the token can list dataset shards (private repos need auth).
func (c *Client) VerifyAccess(ctx context.Context) error {
	if c.Token == "" {
		return fmt.Errorf("no Hugging Face token (private dataset %s). Set HF_TOKEN, or: hf auth login, or put token in ~/.config/lazy-notes/hf_token", c.Repo)
	}
	shards, err := c.ListShards(ctx)
	if err != nil {
		return err
	}
	if len(shards) == 0 {
		return fmt.Errorf("dataset %s: auth ok but no parquet shards under data/", c.Repo)
	}
	return nil
}
