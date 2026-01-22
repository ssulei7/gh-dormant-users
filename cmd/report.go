package cmd

import (
	"fmt"
	"os"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/activity"
	dateUtil "github.com/ssulei7/gh-dormant-users/internal/date"
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

	// Validate date is no longer than 3 months
	if err := dateUtil.ValidateDate(date); err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}

	// Convert date to iso 8601 format
	isoDate, err := dateUtil.GetISODate(date)
	if err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}

	users, err := users.GetOrganizationUsers(orgName, email, client)
	if err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}

	repositories, err := repository.GetOrgRepositories(orgName, client)
	if err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}

	activityTypes, _ := cmd.Flags().GetStringSlice("activity-types")

	// Now, check for activity in the organization's repositories
	ui.BoxWithTitle("Organization Info", fmt.Sprintf("Number of users: %v\nNumber of repositories: %v", len(users), len(repositories)))
	ui.Info("Checking for activity...")

	checker := activity.NewActivityChecker()
	checker.CheckActivity(users, orgName, repositories, isoDate, client, activityTypes)
	checker.GenerateBarChart()

	if err := activity.GenerateUserReportCSV(users, orgName+"-dormant-users.csv"); err != nil {
		ui.Error("Failed to generate report: %v", err)
		os.Exit(1)
	}
}
