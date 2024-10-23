package activity

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"github.com/cli/go-gh/pkg/api"
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

func CheckActivity(users users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	commitActivity(users, organization, repositories, date, client)
	issueActivity(users, organization, repositories, date, client)
	issueCommentActivity(users, organization, repositories, date, client)
	pullRequestCommentActivity(users, organization, repositories, date, client)
}

func commitActivity(usersList users.Users, organization string, repositories repository.Repositories, date string, client api.RESTClient) {
	log.Default().Println("Checking for commit activity in organization: " + organization)
	for _, repo := range repositories {
		log.Default().Println("Checking for commit activity in repository: " + repo.Name)
		commits := commits.GetCommitsSinceDate(organization, repo.Name, date, client)
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
	}
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
	log.Default().Println("Checking for issue activity in organization: " + organization)
	for _, repo := range repositories {
		log.Default().Println("Checking for issue activity in repository: " + repo.Name)
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
	log.Default().Println("Checking for issue comment activity in organization: " + organization)
	for _, repo := range repositories {
		log.Default().Println("Checking for issue comment activity in repository: " + repo.Name)
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
	log.Default().Println("Checking for pull request comment activity in organization: " + organization)
	for _, repo := range repositories {
		log.Default().Println("Checking for pull request comment activity in repository: " + repo.Name)
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
