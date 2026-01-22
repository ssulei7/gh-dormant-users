package ui

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerModel is a bubbletea model for displaying a spinner
type spinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
	result   string
	failed   bool
	done     chan struct{}
}

// SpinnerDoneMsg signals the spinner to stop
type SpinnerDoneMsg struct {
	Success bool
	Message string
}

func newSpinnerModel(message string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(cyan)
	return spinnerModel{
		spinner: s,
		message: message,
		done:    make(chan struct{}),
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case SpinnerDoneMsg:
		m.quitting = true
		m.result = msg.Message
		m.failed = !msg.Success
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.quitting {
		if m.failed {
			return errorStyle.Render(errorPrefix+m.result) + "\n"
		}
		return successStyle.Render(successPrefix+m.result) + "\n"
	}
	return fmt.Sprintf("%s %s\n", m.spinner.View(), m.message)
}

// Spinner provides a controlled spinner that runs in a separate goroutine
type Spinner struct {
	program *tea.Program
	done    chan SpinnerDoneMsg
}

// StartSpinner creates and starts a new spinner with the given message.
// The spinner runs in a background goroutine and must be stopped with Success() or Fail().
func StartSpinner(message string) *Spinner {
	model := newSpinnerModel(message)
	done := make(chan SpinnerDoneMsg, 1)

	program := tea.NewProgram(model,
		tea.WithOutput(os.Stderr),
	)

	go func() {
		msg := <-done
		program.Send(msg)
	}()

	go func() {
		_, _ = program.Run()
	}()

	// Small delay to let spinner initialize
	time.Sleep(10 * time.Millisecond)

	return &Spinner{
		program: program,
		done:    done,
	}
}

// Success stops the spinner with a success message
func (s *Spinner) Success(message string) {
	s.done <- SpinnerDoneMsg{Success: true, Message: message}
	// Wait a moment for the program to render the final state
	time.Sleep(50 * time.Millisecond)
}

// Fail stops the spinner with a failure message
func (s *Spinner) Fail(message string) {
	s.done <- SpinnerDoneMsg{Success: false, Message: message}
	// Wait a moment for the program to render the final state
	time.Sleep(50 * time.Millisecond)
}

// SimpleSpinner is a non-interactive spinner for simpler use cases
// that doesn't require the full tea.Program overhead
type SimpleSpinner struct {
	message  string
	frames   []string
	frame    int
	running  atomic.Bool
	stopChan chan struct{}
}

// NewSimpleSpinner creates a new simple spinner
func NewSimpleSpinner(message string) *SimpleSpinner {
	return &SimpleSpinner{
		message:  message,
		frames:   []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		frame:    0,
		stopChan: make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *SimpleSpinner) Start() {
	s.running.Store(true)
	spinStyle := lipgloss.NewStyle().Foreground(cyan)

	go func() {
		for s.running.Load() {
			select {
			case <-s.stopChan:
				return
			default:
				fmt.Fprintf(os.Stderr, "\r%s %s", spinStyle.Render(s.frames[s.frame]), s.message)
				s.frame = (s.frame + 1) % len(s.frames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner and prints a success message
func (s *SimpleSpinner) Stop(message string) {
	s.running.Store(false)
	close(s.stopChan)
	// Clear the line and print success
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(s.message)+5))
	Success("%s", message)
}

// StopFail stops the spinner and prints a failure message
func (s *SimpleSpinner) StopFail(message string) {
	s.running.Store(false)
	close(s.stopChan)
	// Clear the line and print error
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(s.message)+5))
	Error("%s", message)
}
