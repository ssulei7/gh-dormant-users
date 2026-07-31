package activity

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ssulei7/gh-dormant-users/internal/users"
)

func TestGenerateUserReportCSV(t *testing.T) {
	userList := users.Users{
		{Login: "inactive", Email: "inactive@example.com"},
		{Login: "active", Email: "active@example.com"},
	}
	userList[1].MarkActiveWithType("issues")
	userList[1].AddActivityType("commits")

	path := filepath.Join(t.TempDir(), "report.csv")
	if err := GenerateUserReportCSV(userList, path); err != nil {
		t.Fatalf("GenerateUserReportCSV returned error: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open report: %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("record count = %d, want 3", len(records))
	}
	if got := strings.Join(records[0], ","); got != "Username,Email,Active,ActivityTypes" {
		t.Fatalf("header = %q", got)
	}
	if got := strings.Join(records[1], ","); got != "inactive,inactive@example.com,false,none" {
		t.Fatalf("inactive row = %q", got)
	}
	if records[2][0] != "active" || records[2][1] != "active@example.com" || records[2][2] != "true" {
		t.Fatalf("active row = %v", records[2])
	}
	activityTypes := strings.Split(records[2][3], ",")
	sort.Strings(activityTypes)
	if strings.Join(activityTypes, ",") != "commits,issues" {
		t.Fatalf("activity types = %v", activityTypes)
	}
}

func TestGenerateUserReportCSVReturnsCreateError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "report.csv")
	err := GenerateUserReportCSV(users.Users{{Login: "octocat"}}, path)
	if err == nil {
		t.Fatal("GenerateUserReportCSV returned nil error")
	}
}
