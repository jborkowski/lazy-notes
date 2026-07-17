# lazy-notes

Make → Homebrew. Lazy HF → SuperWhisper → harvest → Notes (disk + Apple Notes).

```bash
make install && make setup && make start
make sync
lazy-notes publish   # retry harvest/publish backlog
```

## Install

### Make (recommended on this machine)

`make install` creates a local Homebrew tap, packs this checkout, and builds from source:

```bash
make install && make setup && make start
```

It also pulls in:

- **memo** ([antoniorodr/memo](https://github.com/antoniorodr/memo)) — pushes notes into Apple Notes
- **SuperWhisper CLI** (`superwhisper`) — submits audio and exposes `history` for harvest
- **go**, **duckdb**, **ffmpeg**, and the **SuperWhisper** macOS app (cask)

### Homebrew tap (from GitHub)

Install the formula from this repo without cloning first:

```bash
brew tap antoniorodr/memo
brew tap jborkowski/lazy-notes https://github.com/jborkowski/lazy-notes
brew install --build-from-source jborkowski/lazy-notes/lazy-notes
lazy-notes setup
brew services start jborkowski/lazy-notes/lazy-notes
```

Formula source: [`Formula/lazy-notes.rb`](https://github.com/jborkowski/lazy-notes/blob/main/Formula/lazy-notes.rb).

Binary archives for macOS (arm64 / amd64 / universal) are attached to [GitHub Releases](https://github.com/jborkowski/lazy-notes/releases).

## Brew service

The formula registers a background daemon (`lazy-notes daemon`):

| Command | Purpose |
|---------|---------|
| `make start` / `brew services start jborkowski/lazy-notes/lazy-notes` | Start daemon |
| `make stop` / `brew services stop …` | Stop daemon |
| `make restart` / `brew services restart …` | Restart daemon |
| `make status` | Brew service info + `lazy-notes status` |
| `make logs` | Tail brew service stdout/stderr logs |

Logs (Homebrew prefix):

- `var/log/lazy-notes.log`
- `var/log/lazy-notes.err.log`

## Where output goes

SuperWhisper processes each clip, but **output does not stay only in SuperWhisper**.

After SuperWhisper finishes, lazy-notes:

1. **Harvests** the transcript from SuperWhisper CLI history
2. **Writes** markdown to `publish.notes_dir` (default `~/.local/share/lazy-notes/notes`)
3. **Pushes** to Apple Notes via `memo` (folder `publish.memo_folder`, default `Lazy Notes`)
4. **Tags** each note with `publish.tag` (default `#lazy-notes`)

Use `lazy-notes publish` to process submitted/harvested backlog; `make sync` and the daemon run harvest/publish automatically when `publish.enabled = true`.

## Commands

| Command | Purpose |
|---------|---------|
| `lazy-notes setup` | Config, SuperWhisper CLI, Note modes |
| `lazy-notes sync` | One HF → SuperWhisper pass (+ harvest/publish when enabled) |
| `lazy-notes publish` | Harvest submitted + publish harvested backlog only |
| `lazy-notes status` | Watermark and per-status counts (incl. harvested/published) |
| `lazy-notes daemon` | Sync on interval (via `make start` / brew services) |

## Lazy sync

First run jumps to the latest recording id; then max 5 clips per pass; deletes audio and parquet after use unless configured otherwise.

## HF auth

`hf auth login` or `HF_TOKEN`.
