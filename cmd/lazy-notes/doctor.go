package main

import (
	"fmt"
	"os"

	"github.com/jborkowski/lazy-notes/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorOffline bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check deps, config, auth, and watchers",
	Long: `Run readiness checks for lazy-notes: binaries (duckdb, ffmpeg, hf, superwhisper, memo, gog),
config, Hugging Face auth, publish targets, and optional Apple Notes / Drive watchers.

Exit codes: 0 = ok or warnings only, 1 = one or more failures.`,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		report := doctor.Run(cmd.Context(), doctor.Options{
			SkipHFAccess: doctorOffline,
		})
		doctor.WriteReport(os.Stdout, report)
		if report.Failed() {
			return exitErr(fmt.Errorf("doctor: %d failure(s)", countFails(report)))
		}
		return nil
	},
}

func countFails(r doctor.Report) int {
	n := 0
	for _, c := range r.Checks {
		if c.Severity == doctor.Fail {
			n++
		}
	}
	return n
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorOffline, "offline", false, "Skip live Hugging Face access probe")
	rootCmd.AddCommand(doctorCmd)
}
