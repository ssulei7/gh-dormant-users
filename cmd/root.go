package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "gh-dormant-users",
	Short: "A CLI tool to report upon and take action on dormant GitHub users within GHEC / GHES",
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
