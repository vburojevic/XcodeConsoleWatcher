package cli

import (
	"errors"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
	"github.com/vedranburojevic/xcw/internal/output"
	"github.com/vedranburojevic/xcw/internal/simulator"
	"github.com/vedranburojevic/xcw/internal/tmux"
)

// WatchCmd watches logs and triggers commands on specific patterns
type WatchCmd struct {
	Simulator        string   `short:"s" default:"booted" help:"Simulator name, UDID, or 'booted' for auto-detect"`
	App              string   `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Pattern          string   `short:"p" help:"Regex pattern to filter log messages"`
	Exclude          string   `short:"x" help:"Regex pattern to exclude from log messages"`
	ExcludeSubsystem []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	OnError          string   `help:"Command to run when error-level log detected"`
	OnFault          string   `help:"Command to run when fault-level log detected"`
	OnPattern        []string `help:"Pattern:command pairs (e.g., 'crash:notify.sh') - can be repeated"`
	Cooldown         string   `default:"5s" help:"Minimum time between trigger executions"`
	Tmux             bool     `help:"Output to tmux session"`
	Session          string   `help:"Custom tmux session name (default: xcw-<simulator>)"`
}

// triggerConfig holds parsed trigger configuration
type triggerConfig struct {
	pattern *regexp.Regexp
	command string
}

// Run executes the watch command
func (c *WatchCmd) Run(globals *Globals) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Parse cooldown duration
	cooldown, err := time.ParseDuration(c.Cooldown)
	if err != nil {
		return c.outputError(globals, "INVALID_COOLDOWN", fmt.Sprintf("invalid cooldown duration: %s", err))
	}

	// Parse pattern triggers
	var triggers []triggerConfig
	for _, pt := range c.OnPattern {
		parts := strings.SplitN(pt, ":", 2)
		if len(parts) != 2 {
			return c.outputError(globals, "INVALID_TRIGGER", fmt.Sprintf("invalid pattern:command format: %s", pt))
		}
		re, err := regexp.Compile(parts[0])
		if err != nil {
			return c.outputError(globals, "INVALID_TRIGGER_PATTERN", fmt.Sprintf("invalid trigger pattern: %s", err))
		}
		triggers = append(triggers, triggerConfig{pattern: re, command: parts[1]})
	}

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := mgr.FindDevice(ctx, c.Simulator)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
	}

	// Determine output destination
	var outputWriter io.Writer = globals.Stdout
	var tmuxMgr *tmux.Manager

	if c.Tmux {
		sessionName := c.Session
		if sessionName == "" {
			sessionName = tmux.GenerateSessionName(device.Name)
		}

		if tmux.IsTmuxAvailable() {
			cfg := &tmux.Config{
				SessionName:   sessionName,
				SimulatorName: device.Name,
				Detached:      true,
			}

			tmuxMgr, err = tmux.NewManager(cfg)
			if err == nil {
				if err := tmuxMgr.GetOrCreateSession(); err == nil {
					outputWriter = tmux.NewWriter(tmuxMgr)
					tmuxMgr.ClearPaneWithBanner(fmt.Sprintf("Watching: %s (%s) [TRIGGER MODE]", device.Name, c.App))

					if globals.Format == "ndjson" {
						fmt.Fprintf(globals.Stdout, `{"type":"tmux","session":"%s","attach":"%s"}`+"\n",
							sessionName, tmuxMgr.AttachCommand())
					} else {
						fmt.Fprintf(globals.Stdout, "Tmux session: %s\n", sessionName)
						fmt.Fprintf(globals.Stdout, "Attach with: %s\n", tmuxMgr.AttachCommand())
					}
				}
			}
		}
	}

	if tmuxMgr != nil {
		defer tmuxMgr.Cleanup()
	}

	// Output watch info
	if !globals.Quiet && tmuxMgr == nil {
		if globals.Format == "ndjson" {
			fmt.Fprintf(globals.Stdout, `{"type":"info","message":"Watching logs from %s","simulator":"%s","mode":"trigger"}`+"\n",
				device.Name, device.UDID)
		} else {
			fmt.Fprintf(globals.Stderr, "Watching logs from %s (%s)\n", device.Name, device.UDID)
			fmt.Fprintf(globals.Stderr, "App: %s\n", c.App)
			if c.OnError != "" {
				fmt.Fprintf(globals.Stderr, "On error: %s\n", c.OnError)
			}
			if c.OnFault != "" {
				fmt.Fprintf(globals.Stderr, "On fault: %s\n", c.OnFault)
			}
			for _, t := range triggers {
				fmt.Fprintf(globals.Stderr, "On pattern '%s': %s\n", t.pattern.String(), t.command)
			}
			fmt.Fprintf(globals.Stderr, "Cooldown: %s\n", c.Cooldown)
			fmt.Fprintln(globals.Stderr, "Press Ctrl+C to stop")
		}
	}

	// Compile pattern regex if provided
	var pattern *regexp.Regexp
	if c.Pattern != "" {
		pattern, err = regexp.Compile(c.Pattern)
		if err != nil {
			return c.outputError(globals, "INVALID_PATTERN", fmt.Sprintf("invalid regex pattern: %s", err))
		}
	}

	// Compile exclude pattern
	var excludePattern *regexp.Regexp
	if c.Exclude != "" {
		excludePattern, err = regexp.Compile(c.Exclude)
		if err != nil {
			return c.outputError(globals, "INVALID_EXCLUDE_PATTERN", fmt.Sprintf("invalid exclude pattern: %s", err))
		}
	}

	// Create streamer
	streamer := simulator.NewStreamer(mgr)
	opts := simulator.StreamOptions{
		BundleID:          c.App,
		MinLevel:          domain.ParseLogLevel(globals.Level),
		Pattern:           pattern,
		ExcludePattern:    excludePattern,
		ExcludeSubsystems: c.ExcludeSubsystem,
		BufferSize:        100,
	}

	if err := streamer.Start(ctx, device.UDID, opts); err != nil {
		return c.outputError(globals, "STREAM_FAILED", err.Error())
	}
	defer streamer.Stop()

	// Track last trigger times for cooldown
	lastErrorTrigger := time.Time{}
	lastFaultTrigger := time.Time{}
	lastPatternTriggers := make(map[int]time.Time)

	// Create output writer
	var writer interface {
		Write(entry *domain.LogEntry) error
	}

	if globals.Format == "ndjson" {
		writer = output.NewNDJSONWriter(outputWriter)
	} else {
		writer = output.NewTextWriter(outputWriter)
	}

	// Process logs
	for {
		select {
		case <-ctx.Done():
			return nil

		case entry := <-streamer.Logs():
			// Output the log entry
			if err := writer.Write(&entry); err != nil {
				return err
			}

			now := time.Now()

			// Check error trigger
			if c.OnError != "" && entry.Level == domain.LogLevelError {
				if now.Sub(lastErrorTrigger) >= cooldown {
					c.runTrigger(globals, "error", c.OnError, &entry)
					lastErrorTrigger = now
				}
			}

			// Check fault trigger
			if c.OnFault != "" && entry.Level == domain.LogLevelFault {
				if now.Sub(lastFaultTrigger) >= cooldown {
					c.runTrigger(globals, "fault", c.OnFault, &entry)
					lastFaultTrigger = now
				}
			}

			// Check pattern triggers
			for i, t := range triggers {
				if t.pattern.MatchString(entry.Message) {
					if now.Sub(lastPatternTriggers[i]) >= cooldown {
						c.runTrigger(globals, "pattern:"+t.pattern.String(), t.command, &entry)
						lastPatternTriggers[i] = now
					}
				}
			}

		case err := <-streamer.Errors():
			if !globals.Quiet {
				if globals.Format == "ndjson" {
					fmt.Fprintf(outputWriter, `{"type":"warning","message":"%s"}`+"\n", err.Error())
				} else {
					fmt.Fprintf(globals.Stderr, "Warning: %s\n", err.Error())
				}
			}
		}
	}
}

// runTrigger executes a trigger command
func (c *WatchCmd) runTrigger(globals *Globals, triggerType, command string, entry *domain.LogEntry) {
	// Output trigger notification
	if globals.Format == "ndjson" {
		fmt.Fprintf(globals.Stdout, `{"type":"trigger","trigger":"%s","command":"%s","message":"%s"}`+"\n",
			triggerType, command, escapeJSON(entry.Message))
	} else if !globals.Quiet {
		fmt.Fprintf(globals.Stderr, "[TRIGGER:%s] Running: %s\n", triggerType, command)
	}

	// Set environment variables for the command
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = append(os.Environ(),
		"XCW_TRIGGER="+triggerType,
		"XCW_LEVEL="+string(entry.Level),
		"XCW_MESSAGE="+entry.Message,
		"XCW_SUBSYSTEM="+entry.Subsystem,
		"XCW_PROCESS="+entry.Process,
		"XCW_TIMESTAMP="+entry.Timestamp.Format(time.RFC3339),
	)

	// Run command in background (don't block log processing)
	go func() {
		if err := cmd.Run(); err != nil {
			if globals.Format == "ndjson" {
				fmt.Fprintf(globals.Stdout, `{"type":"trigger_error","command":"%s","error":"%s"}`+"\n",
					command, escapeJSON(err.Error()))
			} else if !globals.Quiet {
				fmt.Fprintf(globals.Stderr, "[TRIGGER ERROR] %s: %s\n", command, err.Error())
			}
		}
	}()
}

func (c *WatchCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		w := output.NewNDJSONWriter(globals.Stdout)
		w.WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}

// escapeJSON escapes special characters for JSON string
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
