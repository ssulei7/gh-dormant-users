package activity

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/commits"
	"github.com/ssulei7/gh-dormant-users/internal/githubapi"
	"github.com/ssulei7/gh-dormant-users/internal/issues"
	"github.com/ssulei7/gh-dormant-users/internal/pullrequests"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

// ActivityChecker encapsulates activity checking state
type ActivityChecker struct {
	activeUsers map[string]bool
	userIndex   map[string]*users.User
	workers     int
	mu          sync.RWMutex
}

// NewActivityChecker creates a new ActivityChecker
func NewActivityChecker(workerCount ...int) *ActivityChecker {
	workers := 5
	if len(workerCount) > 0 && workerCount[0] > 0 {
		workers = workerCount[0]
	}
	return &ActivityChecker{
		activeUsers: make(map[string]bool),
		userIndex:   make(map[string]*users.User),
		workers:     workers,
	}
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
func (ac *ActivityChecker) CheckActivity(usersList users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient, activityTypes []string) error {
	// Build user index for O(1) lookups
	for i := range usersList {
		user := &usersList[i]
		ac.userIndex[user.Login] = user
		ac.activeUsers[user.Login] = false
	}

	typeSet := newActivityTypeSet(activityTypes)
	since, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return err
	}

	// Calculate total work: repos * number of activity types enabled
	totalWork := len(repositories) * len(activityTypes)
	progressBar := ui.NewProgressBar(totalWork, "Checking for activity...")

	var wg sync.WaitGroup
	var progressMux sync.Mutex
	repoChan := make(chan repository.Repository)
	done := make(chan struct{})
	var stopOnce sync.Once
	var firstErr error
	var errorMux sync.Mutex

	for i := 0; i < ac.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case repo, ok := <-repoChan:
					if !ok {
						return
					}
					if err := ac.checkRepoActivity(organization, repo, date, since, client, typeSet, progressBar, &progressMux); err != nil {
						errorMux.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errorMux.Unlock()
						stopOnce.Do(func() { close(done) })
						return
					}
				}
			}
		}()
	}

enqueue:
	for _, repo := range repositories {
		select {
		case <-done:
			break enqueue
		case repoChan <- repo:
		}
	}
	close(repoChan)
	wg.Wait()
	progressBar.Complete()
	return firstErr
}

// checkRepoActivity checks all enabled activity types for a single repository.
func (ac *ActivityChecker) checkRepoActivity(organization string, repo repository.Repository, date string, since time.Time, client api.RESTClient, typeSet activityTypeSet, progressBar *ui.ProgressBar, progressMux *sync.Mutex) error {
	// Check commits
	if typeSet["commits"] {
		if repo.Size > 0 && (repo.PushedAt == nil || !repo.PushedAt.Before(since)) {
			commitList, err := commits.GetCommitsSinceDate(organization, repo.Name, date, client)
			if err != nil {
				if !skipUnavailableRepositoryEndpoint(progressBar, repo.Name, "commits", err) {
					return err
				}
			}
			for _, commit := range commitList {
				ac.markUserActive(commit.Author.Login, "commits")
			}
		}
		incrementProgress(progressBar, progressMux)
	}

	// Check issues
	if typeSet["issues"] {
		issueList, err := issues.GetIssuesSinceDate(organization, repo.Name, date, client)
		if err != nil {
			if !skipUnavailableRepositoryEndpoint(progressBar, repo.Name, "issues", err) {
				return err
			}
		}
		for _, issue := range issueList {
			ac.markUserActive(issue.User.Login, "issues")
		}
		incrementProgress(progressBar, progressMux)
	}

	// Check issue comments
	if typeSet["issue-comments"] {
		issueCommentList, err := issues.GetIssueCommentsSinceDate(organization, repo.Name, date, client)
		if err != nil {
			if !skipUnavailableRepositoryEndpoint(progressBar, repo.Name, "issue comments", err) {
				return err
			}
		}
		for _, comment := range issueCommentList {
			ac.markUserActive(comment.User.Login, "issue-comments")
		}
		incrementProgress(progressBar, progressMux)
	}

	// Check PR comments
	if typeSet["pr-comments"] {
		prCommentList, err := pullrequests.GetPullRequestCommentsSinceDate(organization, repo.Name, date, client)
		if err != nil {
			if !skipUnavailableRepositoryEndpoint(progressBar, repo.Name, "pull request comments", err) {
				return err
			}
		}
		for _, comment := range prCommentList {
			ac.markUserActive(comment.User.Login, "pr-comments")
		}
		incrementProgress(progressBar, progressMux)
	}
	return nil
}

func skipUnavailableRepositoryEndpoint(progressBar *ui.ProgressBar, repoName string, activityType string, err error) bool {
	if !githubapi.IsRepositoryUnavailable(err) {
		return false
	}
	progressBar.DeferWarning("Skipped %s for %s: repository endpoint is unavailable", activityType, repoName)
	return true
}

func incrementProgress(progressBar *ui.ProgressBar, progressMux *sync.Mutex) {
	progressMux.Lock()
	progressBar.Increment()
	progressMux.Unlock()
}

// markUserActive marks a user as active with the given activity type using O(1) lookup.
func (ac *ActivityChecker) markUserActive(login string, activityType string) {
	user, exists := ac.userIndex[login]
	if !exists {
		return
	}

	// Use atomic method on user (handles its own locking)
	user.MarkActiveWithType(activityType)

	// Update activeUsers map
	ac.mu.Lock()
	ac.activeUsers[login] = true
	ac.mu.Unlock()
}

// GenerateBarChart generates a bar chart of active/inactive users
func (ac *ActivityChecker) GenerateBarChart() {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	activeCount := 0
	inactiveCount := 0
	for _, active := range ac.activeUsers {
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

	for i := range users {
		user := &users[i]
		var atSlice []string
		if !user.IsActive() {
			atSlice = []string{"none"}
		} else {
			atSlice = user.GetActivityTypes()
		}
		record := []string{user.Login, user.Email, strconv.FormatBool(user.IsActive()), strings.Join(atSlice, ",")}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	ui.Success("Report saved to %s", filePath)
	return nil
}
