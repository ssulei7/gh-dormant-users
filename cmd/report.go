package cmd

import (
	"fmt"
	"os"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/activity"
	dateUtil "github.com/ssulei7/gh-dormant-users/internal/date"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a report",
	Run:   generateDormantUserReport,
}

func generateDormantUserReport(cmd *cobra.Command, args []string) {
	// First, get all users in an organization using the gh module
	orgName, _ := cmd.Flags().GetString("org-name")
	email, _ := cmd.Flags().GetBool("email")
	date, _ := cmd.Flags().GetString("date")
	client, err := gh.RESTClient(nil)
	if err != nil {
		ui.Error("Failed to create REST client: %v", err)
		os.Exit(1)
	}

	// Detect rate limit for this token and configure limiter
	limiter.DetectRateLimit(client)

	// Validate date is no longer than 3 months, and turn into an ISO string
	isDateValid := dateUtil.ValidateDate(date)
	if !isDateValid {
		ui.Error("Date must be within the last 3 months")
		os.Exit(1)
	}

	// Convert date to iso 8601 format
	isoDate := dateUtil.GetISODate(date)
	users := users.GetOrganizationUsers(orgName, email, client)

	repositories := repository.GetOrgRepositories(orgName, client)

	activityTypes, _ := cmd.Flags().GetStringSlice("activity-types")

	// Now, check for activity in the organization's repositories
	ui.BoxWithTitle("Organization Info", fmt.Sprintf("Number of users: %v\nNumber of repositories: %v", len(users), len(repositories)))
	ui.Info("Checking for activity...")
	activity.CheckActivity(users, orgName, repositories, isoDate, client, activityTypes)
	activity.GenerateBarChartOfActiveUsers()
	activity.GenerateUserReportCSV(users, orgName+"-dormant-users.csv")
}
