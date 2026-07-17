package main

import (
	"fmt"
	"os"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/paths"
	lnsync "github.com/jborkowski/lazy-notes/internal/sync"
)

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

func openDB() (*db.DB, error) {
	database, err := db.Open(paths.DBPath())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return database, nil
}

func printSyncResult(result *lnsync.Result) {
	fmt.Fprintf(os.Stdout, "scanned=%d submitted=%d harvested=%d published=%d skipped=%d errors=%d watermark=%d\n",
		result.Scanned, result.Submitted, result.Harvested, result.Published,
		result.Skipped, result.Errors, result.Watermark)
}
