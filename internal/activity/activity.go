package activity

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"github.com/cli/go-gh/pkg/api"
	"github.com/pterm/pterm"
	"github.com/ssulei7/gh-dormant-users/internal/commits"
	"github.com/ssulei7/gh-dormant-users/internal/issues"
	"github.com/ssulei7/gh-dormant-users/internal/pullrequests"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

var (
	activeUsers map[string]bool
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
	for _, repo := range repositories {
		commits := commits.GetCommitsSinceDate(organization, repo.Name, date, client)
		commitProgressBar.Increment()
		if len(commits) == 0 {
			continue
		}
		for _, commit := range commits {
			for i := range usersList {
				user := &usersList[i]
				if commit.Author.Login == user.Login {
					if !user.Active && !activeUsers[user.Login] {
						user.MakeActive()
						activeUsers[user.Login] = true
					}
				}
			}
		}
		commitProgressBar.Increment()
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

	activeInactiveBars := []pterm.Bar{
		{Label: "Active", Value: activeCount},
		{Label: "Inactive", Value: inactiveCount},
	}

	pterm.DefaultBarChart.WithBars(activeInactiveBars).WithShowValue().Render()
}

func GenerateUserReportCSV(users users.Users, filePath string) error {
	log.Default().Println("Generating CSV report: " + filePath)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Username", "Email", "Active"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, user := range users {
		record := []string{user.Login, user.Email, strconv.FormatBool(user.Active)}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func issueActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	issueActivityProgressBar, _ := pterm.DefaultProgressbar.WithTotal(len(repositories)).WithTitle("Checking for issue activity...").Start()
	defer issueActivityProgressBar.Stop()
	for _, repo := range repositories {
		issueActivityProgressBar.Increment()
		issues := issues.GetIssuesSinceDate(organization, repo.Name, date, client)
		if len(issues) == 0 {
			continue
		}
		for _, issue := range issues {
			for i := range users {
				user := &users[i]
				if issue.User.Login == user.Login {
					if !user.Active && !activeUsers[user.Login] {
						user.MakeActive()
						activeUsers[user.Login] = true
					}
				}
			}
		}
	}
}

func issueCommentActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	issueCommentProgressBar, _ := pterm.DefaultProgressbar.WithTotal(len(repositories)).WithTitle("Checking for issue comment activity...").Start()
	defer issueCommentProgressBar.Stop()
	for _, repo := range repositories {
		issueCommentProgressBar.Increment()
		issueComments := issues.GetIssueCommentsSinceDate(organization, repo.Name, date, client)
		if len(issueComments) == 0 {
			continue
		}
		for _, issueComment := range issueComments {
			for i := range users {
				user := &users[i]
				if issueComment.User.Login == user.Login {
					if !user.Active && !activeUsers[user.Login] {
						user.MakeActive()
						activeUsers[user.Login] = true
					}
				}
			}
		}
	}
}

func pullRequestCommentActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	pullRequestCommentProgressBar, _ := pterm.DefaultProgressbar.WithTotal(len(repositories)).WithTitle("Checking for pull request comment activity...").Start()
	defer pullRequestCommentProgressBar.Stop()
	for _, repo := range repositories {
		pullRequestCommentProgressBar.Increment()
		pullRequestComments := pullrequests.GetPullRequestCommentsSinceDate(organization, repo.Name, date, client)
		if len(pullRequestComments) == 0 {
			continue
		}
		for _, pullRequestComment := range pullRequestComments {
			for i := range users {
				user := &users[i]
				if pullRequestComment.User.Login == user.Login {
					if !user.Active && !activeUsers[user.Login] {
						user.MakeActive()
						activeUsers[user.Login] = true
					}
				}
			}
		}
	}
}
