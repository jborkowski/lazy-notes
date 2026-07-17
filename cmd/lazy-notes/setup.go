package main

import (
	"fmt"
	"os"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/hf"
	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
	"github.com/spf13/cobra"
)

var setupForce bool

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install config, SuperWhisper CLI, and Note modes",
	Long: `Ensure example config, SuperWhisper CLI, and language-specific SuperWhisper modes are installed.

For a guided first-run checklist with doctor at the end, prefer: lazy-notes onboard`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		configPath := paths.ConfigPath()
		dataDir := paths.DataDir()

		if err := config.EnsureExample(configPath, dataDir); err != nil {
			return exitErr(fmt.Errorf("ensure config: %w", err))
		}
		fmt.Fprintf(os.Stdout, "Config: %s\n", configPath)

		if err := superwhisper.EnsureCLI(ctx); err != nil {
			return exitErr(fmt.Errorf("ensure superwhisper CLI: %w", err))
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return exitErr(fmt.Errorf("load config: %w", err))
		}

		installed, err := superwhisper.InstallModes(cfg, paths.ConfigDir(), setupForce)
		if err != nil {
			return exitErr(fmt.Errorf("install modes: %w", err))
		}
		printModesInstalled(installed)

		token := hf.ResolveToken(cfg.HfTokenFile)
		client := hf.NewClient(cfg.Dataset, token, paths.CacheDir()+"/hf")
		if src := hf.TokenSource(); src != "" {
			fmt.Fprintf(os.Stdout, "HF token: %s\n", src)
		}
		if err := client.VerifyAccess(ctx); err != nil {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "Hugging Face auth required (dataset is private):")
			fmt.Fprintln(os.Stderr, "  1. echo hf_... > ~/.config/lazy-notes/hf_token   # canonical for brew service")
			fmt.Fprintln(os.Stderr, "  2. or: export HF_TOKEN=hf_...")
			fmt.Fprintln(os.Stderr, "  3. or: hf auth login")
			fmt.Fprintln(os.Stderr, "  4. or set hf_token_file in config.toml")
			return exitErr(fmt.Errorf("hf auth: %w", err))
		}
		fmt.Fprintf(os.Stdout, "HF access OK: %s\n", cfg.Dataset)

		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Next steps:")
		fmt.Fprintln(os.Stdout, "  lazy-notes onboard   # step-by-step checklist + doctor")
		fmt.Fprintln(os.Stdout, "  lazy-notes doctor    # re-check deps / auth / watchers")
		fmt.Fprintf(os.Stdout, "  Edit %s (publish / Drive / watch)\n", configPath)
		fmt.Fprintf(os.Stdout, "  Notes → %s + Apple Notes folder %q\n",
			cfg.NotesDir(), cfg.Publish.MemoFolder)
		fmt.Fprintln(os.Stdout, "  make sync && make start")

		return nil
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupForce, "force", false, "Overwrite existing SuperWhisper modes")
	rootCmd.AddCommand(setupCmd)
}
