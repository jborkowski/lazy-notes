package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/jborkowski/lazy-notes/internal/config"
)

func TestReportFailedAndWarned(t *testing.T) {
	ok := Report{Checks: []Check{{Severity: OK}, {Severity: Skip}}}
	if ok.Failed() || ok.Warned() {
		t.Fatal("ok report should not fail or warn")
	}

	warn := Report{Checks: []Check{{Severity: OK}, {Severity: Warn}}}
	if warn.Failed() || !warn.Warned() {
		t.Fatal("expected warn-only report")
	}

	fail := Report{Checks: []Check{{Severity: Warn}, {Severity: Fail}}}
	if !fail.Failed() || fail.Warned() {
		t.Fatal("fail should win over warn")
	}
}

func TestWriteReport(t *testing.T) {
	var buf bytes.Buffer
	WriteReport(&buf, Report{Checks: []Check{
		{Name: "bin.duckdb", Severity: OK, Detail: "/opt/homebrew/bin/duckdb"},
		{Name: "hf.token", Severity: Fail, Detail: "no token found", Fix: "echo hf_... > ~/.config/lazy-notes/hf_token"},
		{Name: "bin.memo", Severity: Skip, Detail: "memo publish disabled"},
	}})
	out := buf.String()
	for _, want := range []string{"[ok]", "[fail]", "[skip]", "fix:", "result: FAIL"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCheckNotesDir(t *testing.T) {
	dir := t.TempDir()
	c := checkNotesDir(dir)
	if c.Severity != OK {
		t.Fatalf("expected OK, got %+v", c)
	}

	c = checkNotesDir("")
	if c.Severity != Fail {
		t.Fatalf("expected Fail for empty dir, got %+v", c)
	}
}

func TestCheckConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("dataset = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := checkConfigFile(path)
	if c.Severity != OK {
		t.Fatalf("expected OK, got %+v", c)
	}

	c = checkConfigFile(filepath.Join(t.TempDir(), "missing.toml"))
	if c.Severity != Fail {
		t.Fatalf("expected Fail, got %+v", c)
	}
}

func TestCheckWatchDriveMisconfigured(t *testing.T) {
	cfg := config.Defaults()
	cfg.Watch.DriveEnabled = true
	cfg.Watch.DriveLocalDir = ""
	cfg.Watch.DriveFolderID = ""
	checks := checkWatch(&cfg)
	found := false
	for _, c := range checks {
		if c.Name == "watch.drive" && c.Severity == Fail {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected watch.drive fail, got %+v", checks)
	}
}
