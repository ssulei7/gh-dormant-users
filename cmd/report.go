package cmd

import (
	"os"

	"github.com/cli/go-gh"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
	client, err := gh.RESTClient(nil)
	if err != nil {
		pterm.Error.Printf("Failed to create REST client: %v\n", err)
		os.Exit(1)
	}

	// Validate date is no longer than 3 months, and turn into an ISO string
	isDateValid := dateUtil.ValidateDate(date)
	if !isDateValid {
		pterm.Error.Println("Date must be within the last 3 months")
		os.Exit(1)
	}

	// Convert date to iso 8601 format
	isoDate := dateUtil.GetISODate(date)
	users := users.GetOrganizationUsers(orgName, email, client)

	repositories := repository.GetOrgRepositories(orgName, client)

	activityTypes, _ := cmd.Flags().GetStringSlice("activity-types")

	// Now, check for activity in the organization's repositories
	box := pterm.DefaultBox.WithTitle("Organization Info").
		WithLeftPadding(1).
		WithRightPadding(1).
		WithBottomPadding(1).
		WithTopPadding(1)
	box.Printfln("Number of users: %v\nNumber of repositories: %v", len(users), len(repositories))
	pterm.Info.Println("Checking for activity...")
	activity.CheckActivity(users, orgName, repositories, isoDate, client, activityTypes)
	activity.GenerateBarChartOfActiveUsers()
	activity.GenerateUserReportCSV(users, orgName+"-dormant-users.csv")
}
