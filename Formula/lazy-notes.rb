class LazyNotes < Formula
  desc "Pull Hugging Face voice memories into SuperWhisper"
  homepage "https://github.com/jborkowski/lazy-notes"
  version "0.1.0"
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
    # Checked out at the release tag. `revision` is filled on main after the
    # tag exists (Formula lives in-repo; avoids a self-hash chicken-egg).
    url "https://github.com/jborkowski/lazy-notes.git", tag: "v0.1.0"
  end

  depends_on "go" => :build
  depends_on "duckdb"
  depends_on "ffmpeg"
  depends_on "antoniorodr/memo/memo"

  # Official SuperWhisper CLI (history / export / search)
  resource "superwhisper-cli" do
    url "https://github.com/superultrainc/superwhisper-cli-release/releases/download/v0.1.0/superwhisper-v0.1.0-macos-universal.tar.gz"
    sha256 "455175349ecd226384e89820f668c080701b7900d841243a3c2707a9aec73860"
  end

  def install
    root = (buildpath/"build-src").directory? ? buildpath/"build-src" : buildpath

    cd root do
      system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/lazy-notes"
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
    environment_variables LAZY_NOTES_DATA_DIR: opt_pkgshare/"config",
                          HOME: Dir.home,
                          HF_HOME: "#{Dir.home}/.config/cache/huggingface"
  end

  def caveats
    <<~EOS
      Formula installs:
        lazy-notes       — sync daemon
        superwhisper     — SuperWhisper CLI (history/export)

      Also required (Makefile installs via cask if missing):
        superwhisper.app — brew install --cask superwhisper

      Where output goes:
        Harvested notes → publish.notes_dir (see config.toml)
        Apple Notes     → memo folder publish.memo_folder (default Lazy Notes)
        Tag             → publish.tag (default #lazy-notes)

      Service:
        brew services start jborkowski/lazy-notes/lazy-notes
        brew services stop  jborkowski/lazy-notes/lazy-notes
        make start|stop|restart|logs|status

      HF token (private dataset): hf auth login  or  HF_TOKEN

      lazy-notes setup && make start
    EOS
  end

  test do
    assert_match "lazy-notes", shell_output("#{bin}/lazy-notes --help")
    assert_match "superwhisper", shell_output("#{bin}/superwhisper --help")
  end
end
