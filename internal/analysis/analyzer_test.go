package analysis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()

	if analyzer == nil {
		t.Error("NewAnalyzer() returned nil")
	}
}

func TestAnalyzerIsCopilotAvailable(t *testing.T) {
	analyzer := NewAnalyzer()

	// This just tests that the method returns a boolean without panicking
	_ = analyzer.IsCopilotAvailable()
}

func TestBuildPrompt_FileNotFound(t *testing.T) {
	analyzer := NewAnalyzer()

	_, err := analyzer.BuildPrompt("/nonexistent/file.csv", "summary", "")
	if err == nil {
		t.Error("BuildPrompt() should return error for nonexistent file")
	}
}

func TestBuildPrompt_InvalidTemplate(t *testing.T) {
	// Create a temporary CSV file
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(csvPath, []byte("Username,Email,Active\nuser1,user1@test.com,true"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	analyzer := NewAnalyzer()

	_, err = analyzer.BuildPrompt(csvPath, "invalid_template", "")
	if err == nil {
		t.Error("BuildPrompt() should return error for invalid template")
	}
}

func TestAnalyzeCSV_CopilotNotAvailable(t *testing.T) {
	analyzer := &Analyzer{copilotAvailable: false}

	_, err := analyzer.AnalyzeCSV("test.csv", "summary", "")
	if err == nil {
		t.Error("AnalyzeCSV() should return error when Copilot is not available")
	}
}

func TestAnalyzeCSV_InvalidTemplate(t *testing.T) {
	// Create a temporary CSV file
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(csvPath, []byte("Username,Email,Active\nuser1,user1@test.com,true"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	analyzer := &Analyzer{copilotAvailable: true}

	_, err = analyzer.AnalyzeCSV(csvPath, "invalid_template", "")
	if err == nil {
		t.Error("AnalyzeCSV() should return error for invalid template")
	}
}

func TestAnalyzeCSV_FileNotFound(t *testing.T) {
	analyzer := &Analyzer{copilotAvailable: true}

	_, err := analyzer.AnalyzeCSV("/nonexistent/file.csv", "summary", "")
	if err == nil {
		t.Error("AnalyzeCSV() should return error for nonexistent file")
	}
}

func TestAnalyzeCSV_CustomTemplateWithoutPrompt(t *testing.T) {
	// Create a temporary CSV file
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(csvPath, []byte("Username,Email,Active\nuser1,user1@test.com,true"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	analyzer := &Analyzer{copilotAvailable: true}

	_, err = analyzer.AnalyzeCSV(csvPath, "custom", "")
	if err == nil {
		t.Error("AnalyzeCSV() should return error for custom template without prompt")
	}
}

func TestBuildPrompt_CustomTemplateWithoutPrompt(t *testing.T) {
	// Create a temporary CSV file
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(csvPath, []byte("Username,Email,Active\nuser1,user1@test.com,true"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	analyzer := NewAnalyzer()

	_, err = analyzer.BuildPrompt(csvPath, "custom", "")
	if err == nil {
		t.Error("BuildPrompt() should return error for custom template without prompt")
	}
}

func TestBuildPrompt_Success(t *testing.T) {
	// Create a temporary CSV file
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csvContent := "Username,Email,Active,ActivityTypes\nuser1,user1@test.com,true,commits\nuser2,user2@test.com,false,none"
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	analyzer := NewAnalyzer()

	prompt, err := analyzer.BuildPrompt(csvPath, "summary", "")
	if err != nil {
		t.Errorf("BuildPrompt() returned error: %v", err)
	}
	if prompt == "" {
		t.Error("BuildPrompt() returned empty prompt")
	}
	// Now we check for formatted stats, not raw CSV
	if !contains(prompt, "Total Users: 2") {
		t.Error("BuildPrompt() prompt should contain aggregated stats")
	}
}

func TestBuildPrompt_CustomTemplateSuccess(t *testing.T) {
	// Create a temporary CSV file
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csvContent := "Username,Email,Active,ActivityTypes\nuser1,user1@test.com,true,commits"
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	analyzer := NewAnalyzer()
	customPrompt := "Analyze the activity patterns"

	prompt, err := analyzer.BuildPrompt(csvPath, "custom", customPrompt)
	if err != nil {
		t.Errorf("BuildPrompt() returned error: %v", err)
	}
	if !contains(prompt, customPrompt) {
		t.Error("BuildPrompt() prompt should contain custom prompt")
	}
	// Check for aggregated stats instead of raw CSV
	if !contains(prompt, "Total Users: 1") {
		t.Error("BuildPrompt() prompt should contain aggregated stats")
	}
}

func TestBuildPrompt_AllTemplates(t *testing.T) {
	// Create a temporary CSV file with all required columns
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csvContent := "Username,Email,Active,ActivityTypes\nuser1,user1@test.com,true,commits"
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV: %v", err)
	}

	analyzer := NewAnalyzer()

	templates := []string{"summary", "trends", "risk", "recommendations"}
	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			prompt, err := analyzer.BuildPrompt(csvPath, tmpl, "")
			if err != nil {
				t.Errorf("BuildPrompt(%s) returned error: %v", tmpl, err)
			}
			if prompt == "" {
				t.Errorf("BuildPrompt(%s) returned empty prompt", tmpl)
			}
			// Check for aggregated stats instead of raw CSV
			if !contains(prompt, "Total Users") {
				t.Errorf("BuildPrompt(%s) prompt should contain aggregated stats", tmpl)
			}
		})
	}
}

func TestListTemplates(t *testing.T) {
	analyzer := NewAnalyzer()

	// This just tests that the method doesn't panic
	// Output goes to stdout so we can't easily test it
	analyzer.ListTemplates()
}

func TestCheckCopilotStatus(t *testing.T) {
	analyzer := NewAnalyzer()

	// This just tests that the method doesn't panic
	// Output goes to stdout so we can't easily test it
	analyzer.CheckCopilotStatus()
}

func TestCheckCopilotStatus_Available(t *testing.T) {
	analyzer := &Analyzer{copilotAvailable: true}
	// Just test it doesn't panic
	analyzer.CheckCopilotStatus()
}

func TestCheckCopilotStatus_NotAvailable(t *testing.T) {
	analyzer := &Analyzer{copilotAvailable: false}
	// Just test it doesn't panic
	analyzer.CheckCopilotStatus()
}
