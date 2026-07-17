# lazy-notes

Make → Homebrew. Lazy HF → SuperWhisper (only **new** clips).

```bash
make install && make setup && make start
make sync
```

**Formula deps:** `go` (build), `duckdb`, `ffmpeg`, cask `superwhisper` (app), plus ships **SuperWhisper CLI** binary (`superwhisper`).

**Where output goes:** SuperWhisper writes transcripts into its app DB (`~/Library/Application Support/superwhisper/database/`). Inspect with `superwhisper history` / `export`. Harvest/publish into Notes is Phase 2 (not built).

**Lazy sync:** first run jumps to latest id; then max 5/pass; deletes audio + parquet after use.

**HF auth:** `hf auth login` or `HF_TOKEN`.
