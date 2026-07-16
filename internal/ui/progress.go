package ui

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
)

// ProgressBar provides a simple, thread-safe progress bar
type ProgressBar struct {
	total     int
	current   int
	title     string
	width     int
	mu        sync.Mutex
	lastLen   int
	completed bool
	writer    io.Writer
	warnings  map[string]int
}

// NewProgressBar creates a new progress bar with the given total and title
func NewProgressBar(total int, title string) *ProgressBar {
	return newProgressBar(total, title, os.Stderr)
}

func newProgressBar(total int, title string, writer io.Writer) *ProgressBar {
	pb := &ProgressBar{
		total:    total,
		title:    title,
		width:    30,
		writer:   writer,
		warnings: make(map[string]int),
	}
	pb.render()
	return pb
}

// Increment advances the progress bar by one
func (pb *ProgressBar) Increment() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if pb.completed {
		return
	}

	pb.current++
	pb.render()
}

// Complete marks the progress bar as complete
func (pb *ProgressBar) Complete() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.completed = true
	pb.current = pb.total
	pb.render()
	fmt.Fprintln(pb.writer)
	pb.renderWarnings()
}

// render draws the progress bar to stderr
func (pb *ProgressBar) render() {
	if pb.total <= 0 {
		return
	}

	percent := float64(pb.current) / float64(pb.total)
	if percent > 1 {
		percent = 1
	}

	filled := int(percent * float64(pb.width))
	empty := pb.width - filled

	// Build the bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Format: Title [████████░░░░░░░░] 25% (25/100)
	output := fmt.Sprintf("\r%s [%s] %3.0f%% (%d/%d)",
		pb.title,
		bar,
		percent*100,
		pb.current,
		pb.total,
	)

	// Clear any remaining characters from previous render
	if len(output) < pb.lastLen {
		output += strings.Repeat(" ", pb.lastLen-len(output))
	}
	pb.lastLen = len(output)

	fmt.Fprint(pb.writer, output)
}

// DeferWarning queues a warning until the progress bar has completed.
func (pb *ProgressBar) DeferWarning(format string, a ...interface{}) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.warnings[fmt.Sprintf(format, a...)]++
}

func (pb *ProgressBar) renderWarnings() {
	if len(pb.warnings) == 0 {
		return
	}

	messages := make([]string, 0, len(pb.warnings))
	total := 0
	for message, count := range pb.warnings {
		messages = append(messages, message)
		total += count
	}
	sort.Strings(messages)

	if total == 1 {
		fmt.Fprintln(pb.writer, warningStyle.Render(warningPrefix)+messages[0])
		return
	}

	fmt.Fprintf(
		pb.writer,
		"%s%d non-fatal warnings occurred during %s:\n",
		warningStyle.Render(warningPrefix),
		total,
		strings.TrimSuffix(pb.title, "..."),
	)
	for _, message := range messages {
		count := pb.warnings[message]
		suffix := ""
		if count > 1 {
			suffix = fmt.Sprintf(" (%d occurrences)", count)
		}
		fmt.Fprintf(pb.writer, "  - %s%s\n", message, suffix)
	}
}

// SetTitle updates the progress bar title
func (pb *ProgressBar) SetTitle(title string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.title = title
	pb.render()
}

// GetProgress returns current progress as a percentage
func (pb *ProgressBar) GetProgress() float64 {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	if pb.total <= 0 {
		return 0
	}
	return float64(pb.current) / float64(pb.total) * 100
}
