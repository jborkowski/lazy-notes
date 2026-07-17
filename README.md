# lazy-notes

Make → Homebrew. Lazy HF and/or Voice Memos export inbox → SuperWhisper → harvest → Notes (disk + Apple Notes).

```bash
make install && lazy-notes onboard && make start
make sync
lazy-notes doctor    # re-check deps / auth / watchers
lazy-notes publish   # retry harvest/publish backlog
```

## Install

### Make (recommended on this machine)

`make install` creates a local Homebrew tap, packs this checkout, and builds from source:

```bash
make install && lazy-notes onboard && make start
```

It also pulls in:

- **memo** ([antoniorodr/memo](https://github.com/antoniorodr/memo)) — pushes notes into Apple Notes
- **gog** ([gogcli](https://github.com/openclaw/gogcli)) — Google Drive upload and change polling
- **hf** ([Hugging Face CLI](https://huggingface.co/docs/huggingface_hub/guides/cli)) — `hf auth login` for private datasets
- **SuperWhisper CLI** (`superwhisper`) — submits audio and exposes `history` for harvest
- **go**, **duckdb**, **ffmpeg**, and the **SuperWhisper** macOS app (cask)

### Homebrew tap (from GitHub)

Install the formula from this repo without cloning first:

```bash
brew tap antoniorodr/memo
brew tap openclaw/tap
brew tap jborkowski/lazy-notes https://github.com/jborkowski/lazy-notes
brew install --build-from-source jborkowski/lazy-notes/lazy-notes
lazy-notes onboard
brew services start jborkowski/lazy-notes/lazy-notes
```

Formula source: [`Formula/lazy-notes.rb`](https://github.com/jborkowski/lazy-notes/blob/main/Formula/lazy-notes.rb).

Binary archives for macOS (arm64 / amd64 / universal) are attached to [GitHub Releases](https://github.com/jborkowski/lazy-notes/releases).

## Release

```bash
make release VERSION=X.Y.Z
```

Bump Formula → tag → upload GitHub archives → pin Formula revision. See [`scripts/release.sh`](scripts/release.sh).

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
4. **Uploads** to Google Drive via `gog` when `publish.drive_enabled = true` (folder `publish.drive_folder_id`)
5. **Tags** each note with `publish.tag` (default `#lazy-notes`)

Use `lazy-notes publish` to process submitted/harvested backlog; `make sync` and the daemon run harvest/publish automatically when `publish.enabled = true`.

### Inputs (what gets transcribed)

| Source | Config | Notes |
|--------|--------|-------|
| Hugging Face dataset | `dataset` + HF token | Default path; stays enabled when Voice Memos is on |
| Voice Memos.app | `[voice_memos]` export inbox | Drop finished `.m4a` into `export_dir` (Shortcuts or manual). **Not** NoteStore. **Not** the Voice Memos Group Container. |

```toml
[voice_memos]
enabled = true
export_dir = "~/.local/share/lazy-notes/voice-memos-inbox"
```

### Optional wake watchers

The daemon can also react to filesystem / Drive / inbox events (debounced) instead of waiting for the next interval. These **wake sync only** — they do not import Apple Notes bodies or read Voice Memos.app’s private container:

| Setting | Purpose |
|---------|---------|
| `voice_memos.enabled` + `watch_enabled` | Watch the Voice Memos **export inbox** for new `.m4a` files |
| `watch.apple_notes_enabled` | Watch Apple Notes `NoteStore.sqlite` (wake only) |
| `watch.drive_local_dir` | Watch a local Google Drive desktop sync directory |
| `watch.drive_folder_id` | Poll Drive folder changes via `gog drive changes poll` |

Google Drive auth (once):

```bash
gog auth credentials set ~/Downloads/client_secret_….json
gog auth add you@example.com --services drive
```

## Commands

| Command | Purpose |
|---------|---------|
| `lazy-notes onboard` | Step-by-step first-run setup, then `doctor` |
| `lazy-notes doctor` | Check deps, config, HF auth, memo/gog, watchers |
| `lazy-notes setup` | Config, SuperWhisper CLI, Note modes (non-interactive) |
| `lazy-notes sync` | One HF + Voice Memos inbox → SuperWhisper pass (+ harvest/publish when enabled) |
| `lazy-notes publish` | Harvest submitted + publish harvested backlog only |
| `lazy-notes status` | Watermark and per-status counts (incl. harvested/published) |
| `lazy-notes daemon` | Sync on interval (via `make start` / brew services) |

### Onboarding & doctor

```bash
lazy-notes onboard          # numbered steps 1–9, ends with doctor
lazy-notes doctor           # ok / warn / fail with fix hints
lazy-notes doctor --offline # skip live Hugging Face access probe
```

`onboard` is idempotent: re-run after editing `config.toml` or installing brew deps.

## Lazy sync

First run jumps to the latest recording id; then max 5 clips per pass; deletes audio and parquet after use unless configured otherwise.

## HF auth

Private datasets need a token. Lookup order:

1. `HF_TOKEN` / `HUGGING_FACE_HUB_TOKEN`
2. `HF_TOKEN_PATH` (brew service sets this to `~/.config/lazy-notes/hf_token`)
3. `~/.config/lazy-notes/hf_token` — **canonical file for the daemon**
4. `hf auth login` locations (`$HF_HOME/token`, `~/.cache/huggingface/token`, …)

```bash
mkdir -p ~/.config/lazy-notes
chmod 700 ~/.config/lazy-notes
echo 'hf_...' > ~/.config/lazy-notes/hf_token
chmod 600 ~/.config/lazy-notes/hf_token
```

Or set `hf_token_file` in `config.toml`.
