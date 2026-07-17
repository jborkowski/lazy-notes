package publish

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const memoBodyEnv = "LAZY_NOTES_MEMO_BODY"

// MemoInvocation documents the memo CLI pattern used by PushMemo.
//
// memo v0.6.0 exposes only interactive add: `memo notes -a -f <folder>`, which
// opens $EDITOR on a temp markdown file. There is no `memo notes add --title`
// or stdin pipe flag, so PushMemo sets EDITOR to a short-lived helper script
// that writes the prepared markdown into that temp file before memo converts it
// to HTML and creates the Apple Note.
const MemoInvocation = `memo notes -a -f "<folder>" with EDITOR=<helper-script> and LAZY_NOTES_MEMO_BODY=<markdown>`

// PushMemo creates n in Apple Notes under folder using the memo CLI.
func PushMemo(ctx context.Context, memoBin, folder string, n Note) error {
	if strings.TrimSpace(memoBin) == "" {
		memoBin = "memo"
	}
	if strings.TrimSpace(folder) == "" {
		return fmt.Errorf("memo folder is empty")
	}

	if err := ensureMemoFolder(ctx, folder); err != nil {
		return err
	}

	title := ResolveTitle(n)
	markdown := buildMemoMarkdown(title, n.Body, n.Tag)

	editorPath, cleanup, err := writeMemoEditorScript()
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, memoBin, "notes", "-a", "-f", folder)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("EDITOR=%s", editorPath),
		fmt.Sprintf("%s=%s", memoBodyEnv, markdown),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("memo notes -a -f %q: %w: %s", folder, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func buildMemoMarkdown(title, body, tag string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(withTag(body, tag))
	if body == "" {
		return "# " + title + "\n"
	}
	if strings.HasPrefix(body, "#") {
		return body + "\n"
	}
	return "# " + title + "\n\n" + body + "\n"
}

func writeMemoEditorScript() (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "lazy-notes-memo-editor-*.sh")
	if err != nil {
		return "", nil, fmt.Errorf("create memo editor script: %w", err)
	}
	path = f.Name()

	script := fmt.Sprintf(`#!/bin/sh
printf '%%s' "$%s" > "$1"
`, memoBodyEnv)
	if _, err := f.WriteString(script); err != nil {
		f.Close()
		os.Remove(path)
		return "", nil, fmt.Errorf("write memo editor script: %w", err)
	}
	if err := f.Chmod(0o700); err != nil {
		f.Close()
		os.Remove(path)
		return "", nil, fmt.Errorf("chmod memo editor script: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", nil, fmt.Errorf("close memo editor script: %w", err)
	}

	cleanup = func() { _ = os.Remove(path) }
	return path, cleanup, nil
}

func ensureMemoFolder(ctx context.Context, folder string) error {
	script := fmt.Sprintf(`
tell application "Notes"
	if not (exists folder %q) then
		make new folder with properties {name:%q}
	end if
end tell
`, folder, folder)

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ensure Apple Notes folder %q: %w: %s", folder, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// MemoBinPath resolves memoBin on PATH when it is not an absolute path.
func MemoBinPath(memoBin string) (string, error) {
	if memoBin == "" {
		memoBin = "memo"
	}
	if filepath.IsAbs(memoBin) {
		return memoBin, nil
	}
	path, err := exec.LookPath(memoBin)
	if err != nil {
		return "", fmt.Errorf("memo binary %q not found on PATH", memoBin)
	}
	return path, nil
}
