package main

import (
	"fmt"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/paths"
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
