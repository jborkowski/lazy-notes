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

// withConfigDB loads config and opens the state DB for a command handler.
func withConfigDB(fn func(*config.Config, *db.DB) error) error {
	cfg, err := loadConfig()
	if err != nil {
		return exitErr(err)
	}
	database, err := openDB()
	if err != nil {
		return exitErr(err)
	}
	defer database.Close()
	return fn(cfg, database)
}

func printModesInstalled(installed []string) {
	if len(installed) > 0 {
		fmt.Fprintf(os.Stdout, "Installed modes: %v\n", installed)
		return
	}
	fmt.Fprintln(os.Stdout, "Modes already installed (use --force to overwrite)")
}

func printSyncResult(result *lnsync.Result) {
	fmt.Fprintf(os.Stdout, "scanned=%d submitted=%d harvested=%d published=%d skipped=%d errors=%d watermark=%d\n",
		result.Scanned, result.Submitted, result.Harvested, result.Published,
		result.Skipped, result.Errors, result.Watermark)
}
