package main

import (
	"fmt"
	"os"
	"strings"

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
	Long:  "Ensure example config, SuperWhisper CLI, and language-specific SuperWhisper modes are installed.",
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
		if len(installed) > 0 {
			fmt.Fprintf(os.Stdout, "Installed modes: %v\n", installed)
		} else {
			fmt.Fprintln(os.Stdout, "Modes already installed (use --force to overwrite)")
		}

		token := hf.DefaultToken()
		if token == "" && cfg.HfTokenFile != "" {
			path := cfg.HfTokenFile
			if strings.HasPrefix(path, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					path = home + path[1:]
				}
			}
			if b, err := os.ReadFile(path); err == nil {
				token = strings.TrimSpace(string(b))
			}
		}
		client := hf.NewClient(cfg.Dataset, token, paths.CacheDir()+"/hf")
		if src := hf.TokenSource(); src != "" {
			fmt.Fprintf(os.Stdout, "HF token: %s\n", src)
		}
		if err := client.VerifyAccess(ctx); err != nil {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "Hugging Face auth required (dataset is private):")
			fmt.Fprintln(os.Stderr, "  1. hf auth login")
			fmt.Fprintln(os.Stderr, "  2. or: export HF_TOKEN=hf_...")
			fmt.Fprintln(os.Stderr, "  3. or: echo hf_... > ~/.config/lazy-notes/hf_token")
			fmt.Fprintln(os.Stderr, "  4. or set hf_token_file in config.toml")
			return exitErr(fmt.Errorf("hf auth: %w", err))
		}
		fmt.Fprintf(os.Stdout, "HF access OK: %s\n", cfg.Dataset)

		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Next steps:")
		fmt.Fprintf(os.Stdout, "  1. Edit %s\n", configPath)
		fmt.Fprintln(os.Stdout, "  2. make sync")
		fmt.Fprintln(os.Stdout, "  3. make start")

		return nil
	},
}

func init() {
	setupCmd.Flags().BoolVar(&setupForce, "force", false, "Overwrite existing SuperWhisper modes")
	rootCmd.AddCommand(setupCmd)
}
