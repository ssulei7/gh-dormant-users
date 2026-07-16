package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressBarDefersWarningsUntilCompletion(t *testing.T) {
	var output bytes.Buffer
	progressBar := newProgressBar(2, "Checking for activity...", &output)

	progressBar.Increment()
	progressBar.DeferWarning("Skipped issue comments for repo-b")
	progressBar.DeferWarning("Skipped issue comments for repo-a")

	beforeCompletion := output.String()
	if strings.Contains(beforeCompletion, "Skipped issue comments") {
		t.Fatal("warning was rendered before progress completed")
	}

	progressBar.Complete()
	rendered := output.String()
	finalProgress := strings.LastIndex(rendered, "100% (2/2)")
	firstWarning := strings.Index(rendered, "2 non-fatal warnings")
	if finalProgress == -1 || firstWarning == -1 || firstWarning < finalProgress {
		t.Fatalf("warnings were not rendered after final progress: %q", rendered)
	}
	if strings.Index(rendered, "repo-a") > strings.Index(rendered, "repo-b") {
		t.Fatalf("warnings were not sorted deterministically: %q", rendered)
	}
}

func TestProgressBarDeduplicatesDeferredWarnings(t *testing.T) {
	var output bytes.Buffer
	progressBar := newProgressBar(1, "Checking...", &output)
	progressBar.DeferWarning("repository unavailable")
	progressBar.DeferWarning("repository unavailable")
	progressBar.Complete()

	rendered := output.String()
	if !strings.Contains(rendered, "repository unavailable (2 occurrences)") {
		t.Fatalf("warning occurrences were not aggregated: %q", rendered)
	}
}
