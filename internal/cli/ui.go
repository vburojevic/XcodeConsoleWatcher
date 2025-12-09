package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vedranburojevic/xcw/internal/domain"
	"github.com/vedranburojevic/xcw/internal/simulator"
	"github.com/vedranburojevic/xcw/internal/tui"
)

// UICmd launches an interactive TUI for viewing logs
type UICmd struct {
	Simulator        string   `short:"s" default:"booted" help:"Simulator name, UDID, or 'booted' for auto-detect"`
	App              string   `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Pattern          string   `short:"p" help:"Regex pattern to filter log messages"`
	Exclude          string   `short:"x" help:"Regex pattern to exclude from log messages"`
	ExcludeSubsystem []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	Subsystem        []string `help:"Filter by subsystem (can be repeated)"`
	Category         []string `help:"Filter by category (can be repeated)"`
	BufferSize       int      `default:"1000" help:"Number of recent logs to buffer"`
}

// Run executes the UI command
func (c *UICmd) Run(globals *Globals) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Find the simulator
	globals.Debug("Finding simulator: %s", c.Simulator)
	mgr := simulator.NewManager()
	device, err := mgr.FindDevice(ctx, c.Simulator)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}
	globals.Debug("Found device: %s (UDID: %s)", device.Name, device.UDID)

	// Compile pattern regex if provided
	var pattern *regexp.Regexp
	if c.Pattern != "" {
		pattern, err = regexp.Compile(c.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Compile exclude pattern regex if provided
	var excludePattern *regexp.Regexp
	if c.Exclude != "" {
		excludePattern, err = regexp.Compile(c.Exclude)
		if err != nil {
			return fmt.Errorf("invalid exclude regex pattern: %w", err)
		}
	}

	// Create streamer
	streamer := simulator.NewStreamer(mgr)
	opts := simulator.StreamOptions{
		BundleID:          c.App,
		Subsystems:        c.Subsystem,
		Categories:        c.Category,
		MinLevel:          domain.ParseLogLevel(globals.Level),
		Pattern:           pattern,
		ExcludePattern:    excludePattern,
		ExcludeSubsystems: c.ExcludeSubsystem,
		BufferSize:        c.BufferSize,
	}

	globals.Debug("Starting log stream for TUI...")
	if err := streamer.Start(ctx, device.UDID, opts); err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}
	defer streamer.Stop()

	// Create TUI model
	model := tui.New(c.App, device.Name, streamer.Logs(), streamer.Errors())

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
