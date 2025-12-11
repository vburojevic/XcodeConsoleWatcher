package cli

// validateFlags centralizes common flag combinations to keep behavior consistent.
func validateFlags(globals *Globals, dryRunJSON bool, tmux bool) error {
	// dry-run-json requires ndjson and no tmux
	if dryRunJSON && tmux {
		return outputErrorCommon(globals, "INVALID_FLAGS", "--dry-run-json cannot be combined with --tmux", "drop --tmux or remove --dry-run-json")
	}
	if dryRunJSON && globals != nil && globals.Format != "ndjson" {
		return outputErrorCommon(globals, "INVALID_FLAGS", "--dry-run-json requires ndjson output", "add --format ndjson or remove --dry-run-json")
	}
	// quiet + text is confusing for agents; steer to ndjson
	if globals != nil && globals.Format == "text" && globals.Quiet {
		return outputErrorCommon(globals, "INVALID_FLAGS", "--quiet is only supported with ndjson output", "switch to --format ndjson or drop --quiet")
	}
	return nil
}
