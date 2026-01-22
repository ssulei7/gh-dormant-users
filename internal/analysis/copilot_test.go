package analysis

import (
	"testing"
)

func TestIsCopilotCLIAvailable(t *testing.T) {
	// This test just verifies the function runs without panic
	// The actual result depends on whether copilot is installed on the system
	_ = IsCopilotCLIAvailable()
}
