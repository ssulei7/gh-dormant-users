package analysis

import (
	"os/exec"
)

// IsCopilotCLIAvailable checks if the GitHub Copilot CLI is installed and available
func IsCopilotCLIAvailable() bool {
	_, err := exec.LookPath("gh")
	if err != nil {
		return false
	}

	// Check if copilot extension is installed
	cmd := exec.Command("gh", "extension", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if copilot extension is in the list
	return containsCopilotExtension(string(output))
}

// containsCopilotExtension checks if the gh copilot extension is installed
func containsCopilotExtension(output string) bool {
	return len(output) > 0 && (contains(output, "github/gh-copilot") || contains(output, "copilot"))
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

// searchSubstring searches for substr in s
func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
