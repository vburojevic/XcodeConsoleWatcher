package tmux

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// ClearPane clears the pane content and scrollback history
func (m *Manager) ClearPane() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pane == nil {
		return ErrNoPaneAvailable
	}

	paneTarget := fmt.Sprintf("%s:0.0", m.config.SessionName)

	// Send reset terminal state + clear screen
	_, err := m.tmux.Command("send-keys", "-t", paneTarget, "-R")
	if err != nil {
		return fmt.Errorf("failed to reset terminal: %w", err)
	}

	// Clear the scrollback history
	_, err = m.tmux.Command("clear-history", "-t", paneTarget)
	if err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	// Send clear command
	_, err = m.tmux.Command("send-keys", "-t", paneTarget, "clear", "Enter")
	if err != nil {
		return fmt.Errorf("failed to clear screen: %w", err)
	}

	return nil
}

// ClearPaneWithBanner clears the pane and displays a session marker
func (m *Manager) ClearPaneWithBanner(message string) error {
	if err := m.ClearPane(); err != nil {
		return err
	}

	// Display session marker
	banner := fmt.Sprintf(
		"═══════════════════════════════════════════════════════════\n"+
			"  XcodeConsoleWatcher - %s\n"+
			"  Session: %s | Started: %s\n"+
			"═══════════════════════════════════════════════════════════",
		message,
		m.config.SessionName,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	return m.WriteLines(strings.Split(banner, "\n"))
}

// WriteSessionBanner writes a visual banner when app is relaunched
func (m *Manager) WriteSessionBanner(session int, app string, pid int, prevSummary *domain.SessionSummary) error {
	// Build previous session summary if available
	prevInfo := ""
	if prevSummary != nil {
		prevInfo = fmt.Sprintf("Previous: %d logs, %d errors | ", prevSummary.TotalLogs, prevSummary.Errors)
	}

	banner := fmt.Sprintf(
		"\n══════════════════════════════════════════════════════════════\n"+
			"  🚀 SESSION %d: %s (PID: %d)\n"+
			"  %s%s\n"+
			"══════════════════════════════════════════════════════════════",
		session,
		app,
		pid,
		prevInfo,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	return m.WriteLines(strings.Split(banner, "\n"))
}

// WriteLine writes a single line to the tmux pane using echo
func (m *Manager) WriteLine(line string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pane == nil {
		return ErrNoPaneAvailable
	}

	// Escape special characters for shell
	escaped := escapeTmuxString(line)
	paneTarget := fmt.Sprintf("%s:0.0", m.config.SessionName)

	// Use send-keys with echo
	_, err := m.tmux.Command("send-keys", "-t", paneTarget, fmt.Sprintf("echo '%s'", escaped), "Enter")
	return err
}

// WriteLines writes multiple lines efficiently
func (m *Manager) WriteLines(lines []string) error {
	for _, line := range lines {
		if err := m.WriteLine(line); err != nil {
			return err
		}
	}
	return nil
}

// escapeTmuxString escapes special characters for tmux send-keys
func escapeTmuxString(s string) string {
	// Escape single quotes for shell
	s = strings.ReplaceAll(s, "'", "'\"'\"'")
	// Escape backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return s
}

// Writer implements io.Writer for streaming logs to tmux pane
type Writer struct {
	manager *Manager
	buffer  strings.Builder
}

// NewWriter creates a new writer that streams to tmux pane
func NewWriter(manager *Manager) *Writer {
	return &Writer{
		manager: manager,
	}
}

// Write implements io.Writer - writes data to tmux pane
func (w *Writer) Write(p []byte) (n int, err error) {
	w.buffer.Write(p)

	// Process complete lines
	content := w.buffer.String()
	lines := strings.Split(content, "\n")

	// Keep incomplete last line in buffer
	if !strings.HasSuffix(content, "\n") && len(lines) > 0 {
		w.buffer.Reset()
		w.buffer.WriteString(lines[len(lines)-1])
		lines = lines[:len(lines)-1]
	} else {
		w.buffer.Reset()
	}

	// Write complete lines to pane
	for _, line := range lines {
		if line == "" {
			continue
		}
		if err := w.manager.WriteLine(line); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

// Flush writes any remaining buffered content
func (w *Writer) Flush() error {
	if w.buffer.Len() > 0 {
		err := w.manager.WriteLine(w.buffer.String())
		w.buffer.Reset()
		return err
	}
	return nil
}

// Ensure Writer implements io.Writer
var _ io.Writer = (*Writer)(nil)
