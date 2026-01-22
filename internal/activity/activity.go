package activity

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/commits"
	"github.com/ssulei7/gh-dormant-users/internal/issues"
	"github.com/ssulei7/gh-dormant-users/internal/pullrequests"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

var (
	activeUsers    map[string]bool
	activeUsersMux sync.RWMutex
)

func init() {
	activeUsers = make(map[string]bool)
}

// activityTypeSet for quick lookup
type activityTypeSet map[string]bool

func newActivityTypeSet(types []string) activityTypeSet {
	set := make(activityTypeSet)
	for _, t := range types {
		set[t] = true
	}
	return set
}

// CheckActivity checks all activity types in a single pass through repositories.
// This is more efficient than separate passes as it reduces progress bar overhead
// and allows for better batching of API calls per repository.
func CheckActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient, activityTypes []string) {
	for _, user := range users {
		activeUsers[user.Login] = false
	}

	typeSet := newActivityTypeSet(activityTypes)

	// Calculate total work: repos * number of activity types enabled
	totalWork := len(repositories) * len(activityTypes)
	progressBar := ui.NewProgressBar(totalWork, "Checking for activity...")

	var wg sync.WaitGroup
	var progressMux sync.Mutex
	repoChan := make(chan repository.Repository, len(repositories))

	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoChan {
				// Process all activity types for this repo
				checkRepoActivity(users, organization, repo.Name, date, client, typeSet, progressBar, &progressMux)
			}
		}()
	}

	for _, repo := range repositories {
		repoChan <- repo
	}
	close(repoChan)
	wg.Wait()
	progressBar.Complete()
}

// checkRepoActivity checks all enabled activity types for a single repository.
func checkRepoActivity(usersList users.Users, organization string, repoName string, date string, client api.RESTClient, typeSet activityTypeSet, progressBar *ui.ProgressBar, progressMux *sync.Mutex) {
	// Check commits
	if typeSet["commits"] {
		commitList := commits.GetCommitsSinceDate(organization, repoName, date, client)
		for _, commit := range commitList {
			markUserActive(usersList, commit.Author.Login, "commits")
		}
		progressMux.Lock()
		progressBar.Increment()
		progressMux.Unlock()
	}

	// Check issues
	if typeSet["issues"] {
		issueList := issues.GetIssuesSinceDate(organization, repoName, date, client)
		for _, issue := range issueList {
			markUserActive(usersList, issue.User.Login, "issues")
		}
		progressMux.Lock()
		progressBar.Increment()
		progressMux.Unlock()
	}

	// Check issue comments
	if typeSet["issue-comments"] {
		issueCommentList := issues.GetIssueCommentsSinceDate(organization, repoName, date, client)
		for _, comment := range issueCommentList {
			markUserActive(usersList, comment.User.Login, "issue-comments")
		}
		progressMux.Lock()
		progressBar.Increment()
		progressMux.Unlock()
	}

	// Check PR comments
	if typeSet["pr-comments"] {
		prCommentList := pullrequests.GetPullRequestCommentsSinceDate(organization, repoName, date, client)
		for _, comment := range prCommentList {
			markUserActive(usersList, comment.User.Login, "pr-comments")
		}
		progressMux.Lock()
		progressBar.Increment()
		progressMux.Unlock()
	}
}

// markUserActive marks a user as active with the given activity type.
func markUserActive(usersList users.Users, login string, activityType string) {
	for i := range usersList {
		user := &usersList[i]
		if user.Login == login {
			user.AddActivityType(activityType)
			activeUsersMux.Lock()
			if !user.Active && !activeUsers[user.Login] {
				user.MakeActive()
				activeUsers[user.Login] = true
			}
			activeUsersMux.Unlock()
			break // Found the user, no need to continue
		}
	}
}

func GenerateBarChartOfActiveUsers() {
	activeCount := 0
	inactiveCount := 0
	for _, active := range activeUsers {
		if active {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	bars := []ui.Bar{
		{Label: "Active", Value: activeCount},
		{Label: "Inactive", Value: inactiveCount},
	}

	ui.BarChart(bars)
}

func GenerateUserReportCSV(users users.Users, filePath string) error {
	ui.Info("Generating CSV report: %s", filePath)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Username", "Email", "Active", "ActivityTypes"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, user := range users {
		var atSlice []string
		if !user.Active {
			atSlice = []string{"none"}
		} else {
			atSlice = user.GetActivityTypes()
		}
		record := []string{user.Login, user.Email, strconv.FormatBool(user.Active), strings.Join(atSlice, ",")}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	ui.Success("Report saved to %s", filePath)
	return nil
}
