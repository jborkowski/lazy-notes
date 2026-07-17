package main

import (
	"fmt"

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
		printModesInstalled(installed)
		return nil
	},
}

func init() {
	modesCmd.Flags().BoolVar(&modesForce, "force", false, "Overwrite existing SuperWhisper modes")
	rootCmd.AddCommand(modesCmd)
}
