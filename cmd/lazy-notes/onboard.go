package main

import (
	"fmt"
	"os"

	"github.com/jborkowski/lazy-notes/internal/onboard"
	"github.com/spf13/cobra"
)

var (
	onboardForce   bool
	onboardOffline bool
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Step-by-step first-run setup, then doctor",
	Long: `Walk through numbered onboarding steps:

  1. Config file
  2. State and notes directories
  3. SuperWhisper CLI
  4. Note modes and prompts
  5. Hugging Face auth
  6. Publish targets (Apple Notes / Drive)
  7. Optional watchers
  8. Doctor readiness check
  9. Next actions

Prefer this over setup for a new machine. Re-run anytime; it is idempotent.
Ends with the same checks as lazy-notes doctor.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := onboard.Run(cmd.Context(), onboard.Options{
			ForceModes:   onboardForce,
			SkipHFAccess: onboardOffline,
			Stdout:       os.Stdout,
			Stderr:       os.Stderr,
		})
		if err != nil {
			return exitErr(err)
		}
		if result.Doctor.Failed() {
			return exitErr(fmt.Errorf("onboard finished with doctor failures — fix them, then: lazy-notes doctor"))
		}
		return nil
	},
}

func init() {
	onboardCmd.Flags().BoolVar(&onboardForce, "force", false, "Overwrite existing SuperWhisper modes")
	onboardCmd.Flags().BoolVar(&onboardOffline, "offline", false, "Skip live Hugging Face access probe in doctor")
	rootCmd.AddCommand(onboardCmd)
}
