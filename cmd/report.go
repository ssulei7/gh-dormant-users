package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a report",
	Run:   generateDormantUserReport,
}

func generateDormantUserReport(cmd *cobra.Command, args []string) {
	// First, get all users in an orgainzation using the gh module
	orgName, _ := cmd.Flags().GetString("org-name")
	users := users.GetOrganizationUsers(orgName)

	log.Printf("Found users: %v", users)

}
