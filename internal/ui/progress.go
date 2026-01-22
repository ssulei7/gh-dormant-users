package ui

import (
	"fmt"
	"os"
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
}

// NewProgressBar creates a new progress bar with the given total and title
func NewProgressBar(total int, title string) *ProgressBar {
	pb := &ProgressBar{
		total: total,
		title: title,
		width: 30,
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
	fmt.Fprintln(os.Stderr) // Move to next line
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

	fmt.Fprint(os.Stderr, output)
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
