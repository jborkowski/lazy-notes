package main

import (
	"fmt"
	"os"

	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
	"github.com/spf13/cobra"
)

var modesForce bool

var modesCmd = &cobra.Command{
	Use:   "modes",
	Short: "Install or refresh SuperWhisper Note modes from config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return exitErr(err)
		}

		installed, err := superwhisper.InstallModes(cfg, paths.ConfigDir(), modesForce)
		if err != nil {
			return exitErr(fmt.Errorf("install modes: %w", err))
		}
		if len(installed) > 0 {
			fmt.Fprintf(os.Stdout, "Installed modes: %v\n", installed)
		} else {
			fmt.Fprintln(os.Stdout, "Modes already installed (use --force to overwrite)")
		}
		return nil
	},
}

func init() {
	modesCmd.Flags().BoolVar(&modesForce, "force", false, "Overwrite existing SuperWhisper modes")
	rootCmd.AddCommand(modesCmd)
}
