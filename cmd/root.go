package cmd

import (
	"log"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-dormant-users",
	Short: "A CLI tool to report upon and take action on dormant GitHub users within GHEC / GHES",
	Run: func(cmd *cobra.Command, args []string) {
		// Create cyan text with emoticon
		text := pterm.NewRGB(0, 255, 255).Sprint("GitHub Dormant Users ૮(-.-)ა")

		// Create a box with padding and the cyan text
		box := pterm.DefaultBox.WithLeftPadding(5).WithRightPadding(5).WithBottomPadding(3).WithTopPadding(3).Sprint(text)

		// Center the box
		pterm.DefaultCenter.Println(box)

		// Show available commands
		cmd.Help()
	},
}

func init() {
	reportCmd.Flags().String("org-name", "", "The name of the organization to report upon")
	reportCmd.Flags().BoolP("email", "e", false, "Check if user has an email")
	reportCmd.Flags().String("date", "", "The date from which to start looking for activity. Max 3 months in the past.")
	reportCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")
	if err := reportCmd.MarkFlagRequired("org-name"); err != nil {
		log.Fatal(err)
	}
	if err := reportCmd.MarkFlagRequired("date"); err != nil {
		log.Fatal(err)
	}
	rootCmd.AddCommand(reportCmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
