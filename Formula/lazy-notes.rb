class LazyNotes < Formula
  desc "Pull Hugging Face voice memories into SuperWhisper"
  homepage "https://github.com/jborkowski/lazy-notes"
  version "0.1.0"
  license "MIT"

  # `make install` packs this checkout into the local tap as lazy-notes-src.tar.gz
  tapdir = Tap.fetch("jborkowski/lazy-notes").path
  tarball = tapdir/"lazy-notes-src.tar.gz"
  raise "Missing #{tarball}; run: make install" unless tarball.exist?

  url "file://#{tarball}"
  sha256 tarball.sha256

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
        Transcripts land in SuperWhisper's DB:
          ~/Library/Application Support/superwhisper/database/
        Read them with:
          superwhisper history --json
          superwhisper export -f markdown
        Harvested notes are written to publish.notes_dir (see config.toml).
        memo (antoniorodr/memo/memo) pushes notes into Apple Notes when
        publish.memo_enabled is set.

      HF token (private dataset): hf auth login  or  HF_TOKEN

      make setup && make start
    EOS
  end

  test do
    assert_match "lazy-notes", shell_output("#{bin}/lazy-notes --help")
    assert_match "superwhisper", shell_output("#{bin}/superwhisper --help")
  end
end
