package analysis

import (
	"os/exec"
)

// IsCopilotCLIAvailable checks if the GitHub Copilot CLI is installed and available
func IsCopilotCLIAvailable() bool {
	// The Copilot CLI is a standalone binary called "copilot"
	// It can be installed via: brew install copilot-cli, npm install -g @github/copilot, or winget install GitHub.Copilot
	_, err := exec.LookPath("copilot")
	return err == nil
}
