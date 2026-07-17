# lazy-notes

Make → Homebrew. Lazy HF → SuperWhisper → harvest → Notes (disk + Apple Notes).

```bash
make install && make setup && make start
make sync
lazy-notes publish   # retry harvest/publish backlog
```

## Install

`make install` pulls in:

- **memo** ([antoniorodr/memo](https://github.com/antoniorodr/memo)) — pushes notes into Apple Notes
- **SuperWhisper CLI** (`superwhisper`) — submits audio and exposes `history` for harvest
- **go**, **duckdb**, **ffmpeg**, and the **SuperWhisper** macOS app (cask)

## Where output goes

SuperWhisper processes each clip, but **output does not stay only in SuperWhisper**.

After SuperWhisper finishes, lazy-notes:

1. **Harvests** the transcript from SuperWhisper CLI history
2. **Writes** markdown to `publish.notes_dir` (default `~/.local/share/lazy-notes/notes`)
3. **Pushes** to Apple Notes via `memo` (folder `publish.memo_folder`, default `Lazy Notes`)

Use `lazy-notes publish` to process submitted/harvested backlog; `make sync` and the daemon run harvest/publish automatically when `publish.enabled = true`.

## Commands

| Command | Purpose |
|---------|---------|
| `lazy-notes setup` | Config, SuperWhisper CLI, Note modes |
| `lazy-notes sync` | One HF → SuperWhisper pass (+ harvest/publish when enabled) |
| `lazy-notes publish` | Harvest submitted + publish harvested backlog only |
| `lazy-notes status` | Watermark and per-status counts (incl. harvested/published) |
| `lazy-notes daemon` | Sync on interval (via `make start`) |

## Lazy sync

First run jumps to the latest recording id; then max 5 clips per pass; deletes audio and parquet after use unless configured otherwise.

## HF auth

`hf auth login` or `HF_TOKEN`.
