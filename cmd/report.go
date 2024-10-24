package cmd

import (
	"log"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/config"
	"github.com/ssulei7/gh-dormant-users/internal/activity"
	dateUtil "github.com/ssulei7/gh-dormant-users/internal/date"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
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
	email, _ := cmd.Flags().GetBool("email")
	date, _ := cmd.Flags().GetString("date")
	config.Verbose = cmd.Flags().Changed("verbose")
	client, err := gh.RESTClient(nil)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	// Validate date is no longer than 3 months, and turn into an ISO string
	isDateValid := dateUtil.ValidateDate(date)
	if !isDateValid {
		log.Fatal("Date must be within the last 3 months")
	}

	// Convert date to iso 8601 format
	isoDate := dateUtil.GetISODate(date)
	users := users.GetOrganizationUsers(orgName, email, client)

	repositories := repository.GetOrgRepositories(orgName, client)

	// Now, check for activity in the organization's repositories
	log.Default().Printf("Checking for activity in organization: %s with %v repositories with %v users\n", orgName, len(repositories), len(users))
	activity.CheckActivity(users, orgName, repositories, isoDate, client)
	activity.GenerateUserReportCSV(users, orgName+"-dormant-users.csv")
}
