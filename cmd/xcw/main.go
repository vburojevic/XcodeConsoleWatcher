package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/vburojevic/xcw/internal/cli"
	"github.com/vburojevic/xcw/internal/config"
)

const quickStart = `xcw - iOS Simulator log streaming for AI agents

Quick start:
  xcw list                              List simulators
  xcw apps -s "iPhone 17 Pro"           List apps
  xcw tail -s "iPhone 17 Pro" -a BUNDLE_ID

For help:
  xcw --help                            All commands and flags
  xcw help --json                       Machine-readable docs (for AI agents)
`

func main() {
	// Show quick start if no args provided
	if len(os.Args) == 1 {
		fmt.Print(quickStart)
		return
	}

	// Load configuration from files/environment
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.Default()
	}

	var c cli.CLI

	// Apply config defaults before parsing
	// These will be overridden by CLI flags if specified
	vars := kong.Vars{
		"config_format":    cfg.Format,
		"config_level":     cfg.Level,
		"config_simulator": cfg.Defaults.Simulator,
		"config_since":     cfg.Defaults.Since,
	}

	ctx := kong.Parse(&c,
		kong.Name("xcw"),
		kong.Description("XcodeConsoleWatcher: Tail iOS Simulator logs for AI agents\n\nAI agents: run 'xcw help --json' for complete machine-readable documentation"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		vars,
	)

	// Create globals with config fallbacks
	globals := cli.NewGlobalsWithConfig(&c, cfg)
	err = ctx.Run(globals)
	if err != nil {
		os.Exit(1)
	}
}
