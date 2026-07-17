package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lazy-notes",
	Short: "Pull voice memories from Hugging Face into SuperWhisper",
	Long: `lazy-notes incrementally syncs audio from a Hugging Face dataset
and opens each clip in SuperWhisper with a language-specific Note mode.`,
	SilenceUsage: true,
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}

func exitErr(err error) error {
	fmt.Fprintln(os.Stderr, err)
	return err
}
