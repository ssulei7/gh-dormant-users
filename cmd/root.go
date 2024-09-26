package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gh-dormant-users",
	Short: "A CLI tool to report upon and take action on dormant GitHub users within GHEC / GHES",
}

func init() {
	reportCmd.Flags().String("org-name", "", "The name of the organization to report upon")
	reportCmd.Flags().String("date", "", "The date from which to start looking for activity")
	reportCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")
	if err := reportCmd.MarkFlagRequired("org-name"); err != nil {
		log.Fatal(err)
	}
	if err := reportCmd.MarkFlagRequired("date"); err != nil {
		log.Fatal(err)
	}
	rootCmd.AddCommand(reportCmd)
	fmt.Println("Added a flag to the report command")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
