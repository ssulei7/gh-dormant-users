package activity

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/cli/go-gh/pkg/api"
	"github.com/pterm/pterm"
	"github.com/ssulei7/gh-dormant-users/internal/commits"
	"github.com/ssulei7/gh-dormant-users/internal/issues"
	"github.com/ssulei7/gh-dormant-users/internal/pullrequests"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

var (
	activeUsers    map[string]bool
	activeUsersMux sync.RWMutex
)

func init() {
	activeUsers = make(map[string]bool)
}

func CheckActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient, activityTypes []string) {
	for _, user := range users {
		activeUsers[user.Login] = false
	}
	for _, activityType := range activityTypes {
		switch activityType {
		case "commits":
			commitActivity(users, organization, repositories, date, client)
		case "issues":
			issueActivity(users, organization, repositories, date, client)
		case "issue-comments":
			issueCommentActivity(users, organization, repositories, date, client)
		case "pr-comments":
			pullRequestCommentActivity(users, organization, repositories, date, client)
		}
	}
}

func commitActivity(usersList users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	commitProgressBar, _ := pterm.DefaultProgressbar.WithTotal(len(repositories)).WithTitle("Checking for commit activity...").Start()
	defer commitProgressBar.Stop()

	var wg sync.WaitGroup
	var progressMux sync.Mutex
	repoChan := make(chan repository.Repository, len(repositories))

	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoChan {
				commitList := commits.GetCommitsSinceDate(organization, repo.Name, date, client)
				if len(commitList) == 0 {
					progressMux.Lock()
					commitProgressBar.Increment()
					progressMux.Unlock()
					continue
				}
				for _, commit := range commitList {
					for i := range usersList {
						user := &usersList[i]
						if commit.Author.Login == user.Login {
							user.AddActivityType("commits")
							activeUsersMux.Lock()
							if !user.Active && !activeUsers[user.Login] {
								user.MakeActive()
								activeUsers[user.Login] = true
							}
							activeUsersMux.Unlock()
						}
					}
				}
				progressMux.Lock()
				commitProgressBar.Increment()
				progressMux.Unlock()
			}
		}()
	}

	for _, repo := range repositories {
		repoChan <- repo
	}
	close(repoChan)
	wg.Wait()
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

	activeInactiveBars := []pterm.Bar{
		{Label: "Active", Value: activeCount},
		{Label: "Inactive", Value: inactiveCount},
	}

	pterm.DefaultBarChart.WithBars(activeInactiveBars).WithShowValue().Render()
}

func GenerateUserReportCSV(users users.Users, filePath string) error {
	pterm.Info.Printf("Generating CSV report: %s\n", filePath)
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

	return nil
}

func issueActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	issueActivityProgressBar, _ := pterm.DefaultProgressbar.WithTotal(len(repositories)).WithTitle("Checking for issue activity...").Start()
	defer issueActivityProgressBar.Stop()

	var wg sync.WaitGroup
	var progressMux sync.Mutex
	repoChan := make(chan repository.Repository, len(repositories))

	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoChan {
				issueList := issues.GetIssuesSinceDate(organization, repo.Name, date, client)
				if len(issueList) == 0 {
					progressMux.Lock()
					issueActivityProgressBar.Increment()
					progressMux.Unlock()
					continue
				}
				for _, issue := range issueList {
					for i := range users {
						user := &users[i]
						if issue.User.Login == user.Login {
							user.AddActivityType("issues")
							activeUsersMux.Lock()
							if !user.Active && !activeUsers[user.Login] {
								user.MakeActive()
								activeUsers[user.Login] = true
							}
							activeUsersMux.Unlock()
						}
					}
				}
				progressMux.Lock()
				issueActivityProgressBar.Increment()
				progressMux.Unlock()
			}
		}()
	}

	for _, repo := range repositories {
		repoChan <- repo
	}
	close(repoChan)
	wg.Wait()
}

func issueCommentActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	issueCommentProgressBar, _ := pterm.DefaultProgressbar.WithTotal(len(repositories)).WithTitle("Checking for issue comment activity...").Start()
	defer issueCommentProgressBar.Stop()

	var wg sync.WaitGroup
	var progressMux sync.Mutex
	repoChan := make(chan repository.Repository, len(repositories))

	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoChan {
				issueCommentList := issues.GetIssueCommentsSinceDate(organization, repo.Name, date, client)
				if len(issueCommentList) == 0 {
					progressMux.Lock()
					issueCommentProgressBar.Increment()
					progressMux.Unlock()
					continue
				}
				for _, issueComment := range issueCommentList {
					for i := range users {
						user := &users[i]
						if issueComment.User.Login == user.Login {
							user.AddActivityType("issue-comments")
							activeUsersMux.Lock()
							if !user.Active && !activeUsers[user.Login] {
								user.MakeActive()
								activeUsers[user.Login] = true
							}
							activeUsersMux.Unlock()
						}
					}
				}
				progressMux.Lock()
				issueCommentProgressBar.Increment()
				progressMux.Unlock()
			}
		}()
	}

	for _, repo := range repositories {
		repoChan <- repo
	}
	close(repoChan)
	wg.Wait()
}

func pullRequestCommentActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	pullRequestCommentProgressBar, _ := pterm.DefaultProgressbar.WithTotal(len(repositories)).WithTitle("Checking for pull request comment activity...").Start()
	defer pullRequestCommentProgressBar.Stop()

	var wg sync.WaitGroup
	var progressMux sync.Mutex
	repoChan := make(chan repository.Repository, len(repositories))

	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range repoChan {
				pullRequestCommentList := pullrequests.GetPullRequestCommentsSinceDate(organization, repo.Name, date, client)
				if len(pullRequestCommentList) == 0 {
					progressMux.Lock()
					pullRequestCommentProgressBar.Increment()
					progressMux.Unlock()
					continue
				}
				for _, pullRequestComment := range pullRequestCommentList {
					for i := range users {
						user := &users[i]
						if pullRequestComment.User.Login == user.Login {
							user.AddActivityType("pr-comments")
							activeUsersMux.Lock()
							if !user.Active && !activeUsers[user.Login] {
								user.MakeActive()
								activeUsers[user.Login] = true
							}
							activeUsersMux.Unlock()
						}
					}
				}
				progressMux.Lock()
				pullRequestCommentProgressBar.Increment()
				progressMux.Unlock()
			}
		}()
	}

	for _, repo := range repositories {
		repoChan <- repo
	}
	close(repoChan)
	wg.Wait()
}
