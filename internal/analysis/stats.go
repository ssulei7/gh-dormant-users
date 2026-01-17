package analysis

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
)

// CSVStats holds pre-aggregated statistics from a dormant users CSV
type CSVStats struct {
	TotalUsers       int
	ActiveUsers      int
	DormantUsers     int
	DormantPercent   float64
	UsersWithEmail   int
	ActivityCounts   map[string]int // counts per activity type
	TopActiveUsers   []UserSummary  // sample of active users (max 10)
	TopDormantUsers  []UserSummary  // sample of dormant users (max 10)
}

// UserSummary is a condensed view of a user for samples
type UserSummary struct {
	Username      string
	HasEmail      bool
	Active        bool
	ActivityTypes string
}

// ParseCSVStats reads a CSV file and returns aggregated statistics
func ParseCSVStats(csvPath string) (*CSVStats, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV file: %w", err)
	}

	if len(records) < 1 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Find column indices from header
	header := records[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Validate required columns
	requiredCols := []string{"username", "active", "activitytypes"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	stats := &CSVStats{
		ActivityCounts:  make(map[string]int),
		TopActiveUsers:  make([]UserSummary, 0, 10),
		TopDormantUsers: make([]UserSummary, 0, 10),
	}

	// Process data rows
	for _, row := range records[1:] {
		if len(row) <= colIndex["username"] {
			continue
		}

		stats.TotalUsers++

		username := row[colIndex["username"]]
		activeStr := strings.ToLower(strings.TrimSpace(row[colIndex["active"]]))
		activityTypes := ""
		if idx, ok := colIndex["activitytypes"]; ok && len(row) > idx {
			activityTypes = row[idx]
		}

		hasEmail := false
		if idx, ok := colIndex["email"]; ok && len(row) > idx && row[idx] != "" {
			hasEmail = true
			stats.UsersWithEmail++
		}

		isActive := activeStr == "true"
		if isActive {
			stats.ActiveUsers++

			// Count activity types
			if activityTypes != "" && activityTypes != "none" {
				for _, activity := range strings.Split(activityTypes, ",") {
					activity = strings.TrimSpace(activity)
					if activity != "" {
						stats.ActivityCounts[activity]++
					}
				}
			}

			// Collect sample of active users (max 10)
			if len(stats.TopActiveUsers) < 10 {
				stats.TopActiveUsers = append(stats.TopActiveUsers, UserSummary{
					Username:      username,
					HasEmail:      hasEmail,
					Active:        true,
					ActivityTypes: activityTypes,
				})
			}
		} else {
			stats.DormantUsers++

			// Collect sample of dormant users (max 10)
			if len(stats.TopDormantUsers) < 10 {
				stats.TopDormantUsers = append(stats.TopDormantUsers, UserSummary{
					Username:      username,
					HasEmail:      hasEmail,
					Active:        false,
					ActivityTypes: activityTypes,
				})
			}
		}
	}

	if stats.TotalUsers > 0 {
		stats.DormantPercent = float64(stats.DormantUsers) / float64(stats.TotalUsers) * 100
	}

	return stats, nil
}

// FormatForPrompt formats the stats as a concise string for the AI prompt
func (s *CSVStats) FormatForPrompt() string {
	var sb strings.Builder

	sb.WriteString("## Dormant Users Report Statistics\n\n")

	// Overview
	sb.WriteString("### Overview\n")
	sb.WriteString(fmt.Sprintf("- Total Users: %d\n", s.TotalUsers))
	sb.WriteString(fmt.Sprintf("- Active Users: %d (%.1f%%)\n", s.ActiveUsers, 100-s.DormantPercent))
	sb.WriteString(fmt.Sprintf("- Dormant Users: %d (%.1f%%)\n", s.DormantUsers, s.DormantPercent))
	sb.WriteString(fmt.Sprintf("- Users with Email: %d (%.1f%%)\n\n", s.UsersWithEmail,
		float64(s.UsersWithEmail)/float64(s.TotalUsers)*100))

	// Activity breakdown
	if len(s.ActivityCounts) > 0 {
		sb.WriteString("### Activity Type Distribution (among active users)\n")

		// Sort by count descending
		type activityCount struct {
			name  string
			count int
		}
		var sorted []activityCount
		for name, count := range s.ActivityCounts {
			sorted = append(sorted, activityCount{name, count})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})

		for _, ac := range sorted {
			pct := float64(ac.count) / float64(s.ActiveUsers) * 100
			sb.WriteString(fmt.Sprintf("- %s: %d users (%.1f%% of active)\n", ac.name, ac.count, pct))
		}
		sb.WriteString("\n")
	}

	// Sample active users
	if len(s.TopActiveUsers) > 0 {
		sb.WriteString("### Sample Active Users (up to 10)\n")
		for _, u := range s.TopActiveUsers {
			emailStatus := "no email"
			if u.HasEmail {
				emailStatus = "has email"
			}
			sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", u.Username, emailStatus, u.ActivityTypes))
		}
		sb.WriteString("\n")
	}

	// Sample dormant users
	if len(s.TopDormantUsers) > 0 {
		sb.WriteString("### Sample Dormant Users (up to 10)\n")
		for _, u := range s.TopDormantUsers {
			emailStatus := "no email"
			if u.HasEmail {
				emailStatus = "has email"
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", u.Username, emailStatus))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
