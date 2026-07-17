package hf

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const hubBase = "https://huggingface.co"

// NewClient returns a Client. Empty token falls back to DefaultToken().
func NewClient(repo, token, cacheDir string) *Client {
	if token == "" {
		token = DefaultToken()
	}
	return &Client{
		Repo:     repo,
		Token:    token,
		CacheDir: cacheDir,
	}
}

func (c *Client) cachePath(shardPath string) string {
	return filepath.Join(c.CacheDir, filepath.FromSlash(shardPath))
}

func (c *Client) hubRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return req, nil
}

func sqlStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func writeFileAtomic(path string, r io.Reader) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".download-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		tmp.Close()
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmp, r); err != nil {
		return fmt.Errorf("download %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true
	return nil
}
