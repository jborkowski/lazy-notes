class LazyNotes < Formula
  desc "Pull Hugging Face voice memories into SuperWhisper"
  homepage "https://github.com/jborkowski/lazy-notes"
  version "0.1.2"
  license "MIT"

  # Prefer a locally packed tarball from `make install`; otherwise build the
  # tagged GitHub source (for `brew tap … https://github.com/jborkowski/lazy-notes`).
  local_tarball = begin
    Tap.fetch("jborkowski/lazy-notes").path/"lazy-notes-src.tar.gz"
  rescue NameError, LoadError, StandardError
    nil
  end

  if local_tarball&.exist?
    url "file://#{local_tarball}"
    sha256 local_tarball.sha256
  else
    url "https://github.com/jborkowski/lazy-notes.git",
        tag:      "v0.1.2",
        revision: "9ea74f707c6a1dfbfb519920ad9f5312ed99a1d7"
  end

  depends_on "go" => :build
  depends_on "duckdb"
  depends_on "ffmpeg"
  depends_on "hf" # Hugging Face CLI (`hf auth login`)
  depends_on "antoniorodr/memo/memo"
  depends_on "openclaw/tap/gogcli" # Google Drive via `gog` CLI

  # Official SuperWhisper CLI (history / export / search)
  resource "superwhisper-cli" do
    url "https://github.com/superultrainc/superwhisper-cli-release/releases/download/v0.1.0/superwhisper-v0.1.0-macos-universal.tar.gz"
    sha256 "455175349ecd226384e89820f668c080701b7900d841243a3c2707a9aec73860"
  end

  def install
    root = (buildpath/"build-src").directory? ? buildpath/"build-src" : buildpath

    cd root do
      system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "./cmd/lazy-notes"
    end

    config_dir = root/"config"
    (pkgshare/"config").install Dir[config_dir/"*"] if config_dir.directory?

    resource("superwhisper-cli").stage do
      bin.install "superwhisper"
    end
  end

  service do
    run [opt_bin/"lazy-notes", "daemon"]
    keep_alive true
    process_type :background
    log_path var/"log/lazy-notes.log"
    error_log_path var/"log/lazy-notes.err.log"
    # launchd has a minimal PATH; duckdb/ffmpeg/memo/hf/gog live under Homebrew.
    # HF_TOKEN_PATH points at the canonical lazy-notes token file (not HF_HOME).
    environment_variables PATH: std_service_path_env,
                          LAZY_NOTES_DATA_DIR: opt_pkgshare/"config",
                          HOME: Dir.home,
                          HF_TOKEN_PATH: "#{Dir.home}/.config/lazy-notes/hf_token"
  end

  def caveats
    <<~EOS
      Formula installs:
        lazy-notes       — sync daemon
        superwhisper     — SuperWhisper CLI (history/export)

      Runtime dependencies:
        hf               — Hugging Face CLI (auth / hub)
        memo             — Apple Notes publisher (antoniorodr/memo)
        gog              — Google Workspace CLI (Drive upload / watch)

      Also required (Makefile installs via cask if missing):
        superwhisper.app — brew install --cask superwhisper

      Where output goes:
        Harvested notes → publish.notes_dir (see config.toml)
        Apple Notes     → memo folder publish.memo_folder (default Lazy Notes)
        Google Drive    → publish.drive_folder_id when publish.drive_enabled
        Tag             → publish.tag (default #lazy-notes)

      Optional watchers (config [watch]):
        Apple Notes SQLite → watch.apple_notes_enabled
        Drive local dir    → watch.drive_local_dir
        Drive folder (gog) → watch.drive_folder_id

      Google Drive auth (once):
        gog auth credentials set ~/Downloads/client_secret_….json
        gog auth add you@example.com --services drive

      Service:
        brew services start jborkowski/lazy-notes/lazy-notes
        brew services stop  jborkowski/lazy-notes/lazy-notes
        make start|stop|restart|logs|status

      HF token (private dataset), in order:
        ~/.config/lazy-notes/hf_token
        HF_TOKEN / hf auth login

      lazy-notes onboard && make start
      lazy-notes doctor
    EOS
  end

  test do
    assert_match "lazy-notes", shell_output("#{bin}/lazy-notes --help")
    assert_equal version.to_s, shell_output("#{bin}/lazy-notes --version").strip
    assert_match "superwhisper", shell_output("#{bin}/superwhisper --help")
    assert_predicate Formula["hf"].opt_bin/"hf", :exist?
    assert_predicate Formula["gogcli"].opt_bin/"gog", :exist?
  end
end
