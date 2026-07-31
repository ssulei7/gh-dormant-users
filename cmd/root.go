package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:           "gh-dormant-users",
	Short:         "A CLI tool to report upon and take action on dormant GitHub users within GHEC / GHES",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		// Display welcome box with cyan text
		ui.CyanBox("GitHub Dormant Users ૮(-.-)ა")

		// Show available commands
		cmd.Help()
	},
}

func init() {
	reportCmd.Flags().String("org-name", "", "The name of the organization to report upon")
	reportCmd.Flags().BoolP("email", "e", false, "Check if user has an email")
	reportCmd.Flags().String("date", "", "The date from which to start looking for activity. Max 3 months in the past.")
	reportCmd.Flags().StringSlice("activity-types", []string{"commits", "issues", "issue-comments", "pr-comments"}, "Comma-separated list of activity types to check (commits, issues, issue-comments, pr-comments)")
	reportCmd.Flags().String("request-mode", "bounded", "API request mode: bounded (default) or safe (serial)")
	reportCmd.Flags().Int("initial-concurrency", 5, "Initial concurrent API requests in bounded mode")
	reportCmd.Flags().Int("max-concurrency", 15, "Adaptive concurrency ceiling in bounded mode (1-15)")
	reportCmd.Flags().Float64("requests-per-second", 10, "Global API request rate cap (0-15)")
	reportCmd.Flags().Int("rate-limit-reserve", 10, "Percentage of the primary rate limit to preserve (0-99)")
	reportCmd.Flags().String("cache-dir", "", "Directory for the strict ETag response cache")
	reportCmd.Flags().Bool("no-cache", false, "Disable the persistent response cache")
	reportCmd.Flags().Bool("clear-cache", false, "Clear the response cache before collecting data")
	if err := reportCmd.MarkFlagRequired("org-name"); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
	if err := reportCmd.MarkFlagRequired("date"); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}
	os.Exit(0)
}
