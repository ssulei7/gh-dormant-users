package users

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/githubapi"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

const emailBatchSize = 50

type User struct {
	Login         string `json:"login"`
	ID            int    `json:"id"`
	Email         string `json:"email"`
	Active        bool
	ActivityTypes map[string]bool
	mu            sync.Mutex
}

type Users []User

func GetOrganizationUsers(organization string, email bool, restClient api.RESTClient, gqlClient api.GQLClient) (Users, error) {
	ui.Info("Starting to fetch users for organization: %s", organization)
	spinner := ui.NewSimpleSpinner("Fetching users...")
	spinner.Start()

	url := fmt.Sprintf("orgs/%s/members?per_page=100", organization)
	userList, err := githubapi.GetAll[User](restClient, url)
	if err != nil {
		spinner.StopFail("Failed to fetch users")
		return nil, fmt.Errorf("fetch organization users: %w", err)
	}
	users := Users(userList)

	if email {
		ui.Info("Getting public profile emails")
		if gqlClient == nil {
			spinner.StopFail("Failed to get user emails")
			return nil, fmt.Errorf("GraphQL client is required to fetch user emails")
		}
		if err := getUserEmails(users, gqlClient); err != nil {
			spinner.StopFail("Failed to get user emails")
			return nil, err
		}
	}

	spinner.Stop("Fetched users successfully")
	return users, nil
}

func (u *User) MakeActive() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Active = true
}

func (u *User) MakeInactive() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Active = false
}

func (u *User) IsActive() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.Active
}

func (u *User) AddActivityType(t string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ActivityTypes == nil {
		u.ActivityTypes = make(map[string]bool)
	}
	u.ActivityTypes[t] = true
}

func (u *User) MarkActiveWithType(t string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ActivityTypes == nil {
		u.ActivityTypes = make(map[string]bool)
	}
	u.ActivityTypes[t] = true
	u.Active = true
}

func (u *User) GetActivityTypes() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ActivityTypes == nil {
		return nil
	}
	var activityTypes []string
	for activityType := range u.ActivityTypes {
		activityTypes = append(activityTypes, activityType)
	}
	return activityTypes
}

func getUserEmails(users Users, client api.GQLClient) error {
	for start := 0; start < len(users); start += emailBatchSize {
		end := min(start+emailBatchSize, len(users))
		if err := getUserEmailBatch(users[start:end], client); err != nil {
			return fmt.Errorf("fetch user email batch starting at %d: %w", start, err)
		}
	}
	return nil
}

func getUserEmailBatch(users Users, client api.GQLClient) error {
	declarations := make([]string, 0, len(users))
	fields := make([]string, 0, len(users))
	variables := make(map[string]interface{}, len(users))
	aliases := make(map[string]struct{}, len(users))
	for index := range users {
		variable := fmt.Sprintf("login%d", index)
		alias := fmt.Sprintf("user%d", index)
		user := &users[index]
		declarations = append(declarations, fmt.Sprintf("$%s:String!", variable))
		fields = append(fields, fmt.Sprintf("%s:user(login:$%s){login email}", alias, variable))
		variables[variable] = user.Login
		aliases[alias] = struct{}{}
	}

	query := fmt.Sprintf("query(%s){%s}", strings.Join(declarations, ","), strings.Join(fields, " "))
	type profile struct {
		Login string `json:"login"`
		Email string `json:"email"`
	}
	result := make(map[string]*profile, len(users))
	if err := client.Do(query, variables, &result); err != nil && !isPartialUserLookupError(err, aliases) {
		return err
	}

	emailByLogin := make(map[string]string, len(result))
	for _, user := range result {
		if user != nil {
			emailByLogin[user.Login] = user.Email
		}
	}
	for index := range users {
		users[index].Email = emailByLogin[users[index].Login]
	}
	return nil
}

func isPartialUserLookupError(err error, aliases map[string]struct{}) bool {
	var gqlError api.GQLError
	if !errors.As(err, &gqlError) || len(gqlError.Errors) == 0 {
		return false
	}
	for _, item := range gqlError.Errors {
		if len(item.Path) == 0 {
			return false
		}
		alias, ok := item.Path[0].(string)
		if !ok {
			return false
		}
		if _, ok := aliases[alias]; !ok {
			return false
		}
	}
	return true
}
