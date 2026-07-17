// Package excmd runs external commands and captures stdout/stderr for CLIs.
package excmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes name with args and returns stdout. On failure, stderr (or stdout)
// is included in the error message when present.
func Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return RunCmd(exec.CommandContext(ctx, name, args...))
}

// RunCmd runs cmd, capturing stdout and stderr.
func RunCmd(cmd *exec.Cmd) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			return stdout.Bytes(), err
		}
		return stdout.Bytes(), fmt.Errorf("%w: %s", err, msg)
	}
	return stdout.Bytes(), nil
}
