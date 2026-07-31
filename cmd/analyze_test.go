package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeAnalyzer struct {
	listed          bool
	checked         bool
	available       bool
	prompt          string
	promptErr       error
	response        string
	analysisErr     error
	buildCalls      int
	analysisCalls   int
	templateName    string
	customPrompt    string
	analyzedCSVPath string
}

func (f *fakeAnalyzer) ListTemplates() {
	f.listed = true
}

func (f *fakeAnalyzer) CheckCopilotStatus() {
	f.checked = true
}

func (f *fakeAnalyzer) BuildPrompt(csvPath string, templateName string, customPrompt string) (string, error) {
	f.buildCalls++
	f.analyzedCSVPath = csvPath
	f.templateName = templateName
	f.customPrompt = customPrompt
	return f.prompt, f.promptErr
}

func (f *fakeAnalyzer) IsCopilotAvailable() bool {
	return f.available
}

func (f *fakeAnalyzer) AnalyzeCSV(csvPath string, templateName string, customPrompt string) (string, error) {
	f.analysisCalls++
	f.analyzedCSVPath = csvPath
	f.templateName = templateName
	f.customPrompt = customPrompt
	return f.response, f.analysisErr
}

type fakeSpinner struct {
	started     bool
	stopMessage string
	failMessage string
}

func (f *fakeSpinner) Start() {
	f.started = true
}

func (f *fakeSpinner) Stop(message string) {
	f.stopMessage = message
}

func (f *fakeSpinner) StopFail(message string) {
	f.failMessage = message
}

func configureAnalyzeTest(t *testing.T, analyzer *fakeAnalyzer, spinner *fakeSpinner) {
	t.Helper()
	oldAnalyzer := newCSVAnalyzer
	oldSpinner := newSpinner
	newCSVAnalyzer = func() csvAnalyzer { return analyzer }
	newSpinner = func(string) analysisSpinner { return spinner }
	t.Cleanup(func() {
		newCSVAnalyzer = oldAnalyzer
		newSpinner = oldSpinner
	})
}

func writeCSVFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "users.csv")
	if err := os.WriteFile(path, []byte("Username,Email,Active,ActivityTypes\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func executeAnalyze(args ...string) error {
	command := newAnalyzeCommand()
	command.SetArgs(args)
	return command.Execute()
}

func TestAnalyzeCommandListsTemplates(t *testing.T) {
	analyzer := &fakeAnalyzer{}
	configureAnalyzeTest(t, analyzer, &fakeSpinner{})

	if err := executeAnalyze("--list-templates"); err != nil {
		t.Fatalf("execute analyze: %v", err)
	}
	if !analyzer.listed {
		t.Fatal("ListTemplates was not called")
	}
}

func TestAnalyzeCommandChecksCopilot(t *testing.T) {
	analyzer := &fakeAnalyzer{}
	configureAnalyzeTest(t, analyzer, &fakeSpinner{})

	if err := executeAnalyze("--check-copilot"); err != nil {
		t.Fatalf("execute analyze: %v", err)
	}
	if !analyzer.checked {
		t.Fatal("CheckCopilotStatus was not called")
	}
}

func TestAnalyzeCommandRequiresCSVFile(t *testing.T) {
	configureAnalyzeTest(t, &fakeAnalyzer{}, &fakeSpinner{})

	err := executeAnalyze()
	if err == nil || !strings.Contains(err.Error(), "CSV file path is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestAnalyzeCommandRejectsMissingCSVFile(t *testing.T) {
	configureAnalyzeTest(t, &fakeAnalyzer{}, &fakeSpinner{})

	err := executeAnalyze("--file", filepath.Join(t.TempDir(), "missing.csv"))
	if err == nil || !strings.Contains(err.Error(), "CSV file not found") {
		t.Fatalf("error = %v", err)
	}
}

func TestAnalyzeCommandPromptOnly(t *testing.T) {
	path := writeCSVFixture(t)
	analyzer := &fakeAnalyzer{prompt: "generated prompt"}
	configureAnalyzeTest(t, analyzer, &fakeSpinner{})

	err := executeAnalyze("--file", path, "--template", "custom", "--prompt", "focus", "--prompt-only")
	if err != nil {
		t.Fatalf("execute analyze: %v", err)
	}
	if analyzer.buildCalls != 1 || analyzer.templateName != "custom" || analyzer.customPrompt != "focus" {
		t.Fatalf("build call = %#v", analyzer)
	}
}

func TestAnalyzeCommandWrapsPromptError(t *testing.T) {
	path := writeCSVFixture(t)
	analyzer := &fakeAnalyzer{promptErr: errors.New("boom")}
	configureAnalyzeTest(t, analyzer, &fakeSpinner{})

	err := executeAnalyze("--file", path, "--prompt-only")
	if err == nil || !strings.Contains(err.Error(), "failed to build prompt: boom") {
		t.Fatalf("error = %v", err)
	}
}

func TestAnalyzeCommandRequiresCopilot(t *testing.T) {
	path := writeCSVFixture(t)
	configureAnalyzeTest(t, &fakeAnalyzer{available: false}, &fakeSpinner{})

	err := executeAnalyze("--file", path)
	if err == nil || !strings.Contains(err.Error(), "GitHub Copilot CLI is not available") {
		t.Fatalf("error = %v", err)
	}
}

func TestAnalyzeCommandRunsAnalysis(t *testing.T) {
	path := writeCSVFixture(t)
	analyzer := &fakeAnalyzer{available: true, response: "analysis"}
	spinner := &fakeSpinner{}
	configureAnalyzeTest(t, analyzer, spinner)

	if err := executeAnalyze("--file", path, "--template", "risk"); err != nil {
		t.Fatalf("execute analyze: %v", err)
	}
	if analyzer.analysisCalls != 1 || analyzer.templateName != "risk" {
		t.Fatalf("analysis call = %#v", analyzer)
	}
	if !spinner.started || spinner.stopMessage != "Analysis complete" || spinner.failMessage != "" {
		t.Fatalf("spinner = %#v", spinner)
	}
}

func TestAnalyzeCommandStopsSpinnerOnFailure(t *testing.T) {
	path := writeCSVFixture(t)
	analyzer := &fakeAnalyzer{available: true, analysisErr: errors.New("boom")}
	spinner := &fakeSpinner{}
	configureAnalyzeTest(t, analyzer, spinner)

	err := executeAnalyze("--file", path)
	if err == nil || !strings.Contains(err.Error(), "analysis failed: boom") {
		t.Fatalf("error = %v", err)
	}
	if !spinner.started || spinner.failMessage != "Analysis failed" || spinner.stopMessage != "" {
		t.Fatalf("spinner = %#v", spinner)
	}
}
