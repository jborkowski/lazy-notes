package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at link time via -ldflags "-X main.version=…".
// Default "dev" is for local `go build` / `go run` without release flags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "lazy-notes",
	Short: "Pull voice memories from Hugging Face into SuperWhisper",
	Long: `lazy-notes incrementally syncs audio from a Hugging Face dataset
and opens each clip in SuperWhisper with a language-specific Note mode.`,
	Version:      version,
	SilenceUsage: true,
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

func exitErr(err error) error {
	fmt.Fprintln(os.Stderr, err)
	return err
}
