package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCSVStats_ValidCSV(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csvContent := `Username,Email,Active,ActivityTypes
user1,user1@test.com,true,commits
user2,user2@test.com,true,"commits,issues"
user3,,false,none
user4,user4@test.com,false,none
user5,,true,issues`

	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	stats, err := ParseCSVStats(csvPath)
	if err != nil {
		t.Fatalf("ParseCSVStats() returned error: %v", err)
	}

	if stats.TotalUsers != 5 {
		t.Errorf("TotalUsers = %d, want 5", stats.TotalUsers)
	}
	if stats.ActiveUsers != 3 {
		t.Errorf("ActiveUsers = %d, want 3", stats.ActiveUsers)
	}
	if stats.DormantUsers != 2 {
		t.Errorf("DormantUsers = %d, want 2", stats.DormantUsers)
	}
	if stats.UsersWithEmail != 3 {
		t.Errorf("UsersWithEmail = %d, want 3", stats.UsersWithEmail)
	}

	expectedDormantPct := 40.0
	if stats.DormantPercent != expectedDormantPct {
		t.Errorf("DormantPercent = %.1f, want %.1f", stats.DormantPercent, expectedDormantPct)
	}

	// Check activity counts
	if stats.ActivityCounts["commits"] != 2 {
		t.Errorf("ActivityCounts[commits] = %d, want 2", stats.ActivityCounts["commits"])
	}
	if stats.ActivityCounts["issues"] != 2 {
		t.Errorf("ActivityCounts[issues] = %d, want 2", stats.ActivityCounts["issues"])
	}
}

func TestParseCSVStats_FileNotFound(t *testing.T) {
	_, err := ParseCSVStats("/nonexistent/file.csv")
	if err == nil {
		t.Error("ParseCSVStats() should return error for nonexistent file")
	}
}

func TestParseCSVStats_EmptyCSV(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "empty.csv")
	err := os.WriteFile(csvPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	_, err = ParseCSVStats(csvPath)
	if err == nil {
		t.Error("ParseCSVStats() should return error for empty CSV")
	}
}

func TestParseCSVStats_HeaderOnly(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "header_only.csv")
	err := os.WriteFile(csvPath, []byte("Username,Email,Active,ActivityTypes\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	stats, err := ParseCSVStats(csvPath)
	if err != nil {
		t.Fatalf("ParseCSVStats() returned error: %v", err)
	}

	if stats.TotalUsers != 0 {
		t.Errorf("TotalUsers = %d, want 0", stats.TotalUsers)
	}
}

func TestParseCSVStats_MissingColumn(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "missing_col.csv")
	// Missing 'active' column
	err := os.WriteFile(csvPath, []byte("Username,Email\nuser1,test@test.com"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	_, err = ParseCSVStats(csvPath)
	if err == nil {
		t.Error("ParseCSVStats() should return error for missing required column")
	}
}

func TestParseCSVStats_CaseInsensitiveHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "case.csv")
	csvContent := `USERNAME,EMAIL,ACTIVE,ACTIVITYTYPES
user1,user1@test.com,true,commits`

	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	stats, err := ParseCSVStats(csvPath)
	if err != nil {
		t.Fatalf("ParseCSVStats() returned error: %v", err)
	}

	if stats.TotalUsers != 1 {
		t.Errorf("TotalUsers = %d, want 1", stats.TotalUsers)
	}
}

func TestParseCSVStats_SampleLimits(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "large.csv")

	// Create CSV with 20 active and 20 dormant users
	csvContent := "Username,Email,Active,ActivityTypes\n"
	for i := 0; i < 20; i++ {
		csvContent += "active" + string(rune('a'+i)) + ",,true,commits\n"
	}
	for i := 0; i < 20; i++ {
		csvContent += "dormant" + string(rune('a'+i)) + ",,false,none\n"
	}

	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	stats, err := ParseCSVStats(csvPath)
	if err != nil {
		t.Fatalf("ParseCSVStats() returned error: %v", err)
	}

	// Should only keep 10 samples each
	if len(stats.TopActiveUsers) != 10 {
		t.Errorf("TopActiveUsers length = %d, want 10", len(stats.TopActiveUsers))
	}
	if len(stats.TopDormantUsers) != 10 {
		t.Errorf("TopDormantUsers length = %d, want 10", len(stats.TopDormantUsers))
	}
}

func TestCSVStats_FormatForPrompt(t *testing.T) {
	stats := &CSVStats{
		TotalUsers:     100,
		ActiveUsers:    30,
		DormantUsers:   70,
		DormantPercent: 70.0,
		UsersWithEmail: 50,
		ActivityCounts: map[string]int{
			"commits": 25,
			"issues":  15,
		},
		TopActiveUsers: []UserSummary{
			{Username: "user1", HasEmail: true, Active: true, ActivityTypes: "commits"},
		},
		TopDormantUsers: []UserSummary{
			{Username: "user2", HasEmail: false, Active: false, ActivityTypes: "none"},
		},
	}

	output := stats.FormatForPrompt()

	// Check key elements are present
	expectedStrings := []string{
		"Total Users: 100",
		"Active Users: 30",
		"Dormant Users: 70",
		"70.0%",
		"commits",
		"issues",
		"user1",
		"user2",
	}

	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("FormatForPrompt() output missing %q", expected)
		}
	}
}

func TestCSVStats_FormatForPrompt_EmptyStats(t *testing.T) {
	stats := &CSVStats{
		TotalUsers:      0,
		ActivityCounts:  make(map[string]int),
		TopActiveUsers:  []UserSummary{},
		TopDormantUsers: []UserSummary{},
	}

	// Should not panic
	output := stats.FormatForPrompt()
	if output == "" {
		t.Error("FormatForPrompt() returned empty string")
	}
}

func TestParseCSVStats_NoneActivityType(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "none_activity.csv")
	csvContent := `Username,Email,Active,ActivityTypes
user1,,true,none
user2,,true,commits`

	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	stats, err := ParseCSVStats(csvPath)
	if err != nil {
		t.Fatalf("ParseCSVStats() returned error: %v", err)
	}

	// "none" should not be counted as an activity type
	if _, exists := stats.ActivityCounts["none"]; exists {
		t.Error("ActivityCounts should not include 'none'")
	}
	if stats.ActivityCounts["commits"] != 1 {
		t.Errorf("ActivityCounts[commits] = %d, want 1", stats.ActivityCounts["commits"])
	}
}
