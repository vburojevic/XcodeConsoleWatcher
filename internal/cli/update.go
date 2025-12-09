package cli

import (
	"encoding/json"
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
)

// UpdateCmd shows how to upgrade xcw
type UpdateCmd struct{}

// UpdateOutput represents the NDJSON output for update instructions
type UpdateOutput struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schemaVersion"`
	Version       string `json:"current_version"`
	Commit        string `json:"commit"`
	Homebrew      string `json:"homebrew"`
	GoInstall     string `json:"go_install"`
	ReleasesURL   string `json:"releases_url"`
}

const (
	homebrewCmd  = "brew update && brew upgrade xcw"
	goInstallCmd = "go install github.com/vburojevic/xcw/cmd/xcw@latest"
	releasesURL  = "https://github.com/vburojevic/xcw/releases"
)

// Run executes the update command
func (c *UpdateCmd) Run(globals *Globals) error {
	if globals.Format == "ndjson" {
		return c.outputNDJSON(globals)
	}
	return c.outputText(globals)
}

func (c *UpdateCmd) outputNDJSON(globals *Globals) error {
	out := UpdateOutput{
		Type:          "update",
		SchemaVersion: output.SchemaVersion,
		Version:       Version,
		Commit:        Commit,
		Homebrew:      homebrewCmd,
		GoInstall:     goInstallCmd,
		ReleasesURL:   releasesURL,
	}

	encoder := json.NewEncoder(globals.Stdout)
	return encoder.Encode(out)
}

func (c *UpdateCmd) outputText(globals *Globals) error {
	fmt.Fprintln(globals.Stdout, "xcw update instructions")
	fmt.Fprintln(globals.Stdout)
	fmt.Fprintf(globals.Stdout, "Current version: %s (%s)\n", Version, Commit)
	fmt.Fprintln(globals.Stdout)
	fmt.Fprintln(globals.Stdout, "To upgrade via Homebrew:")
	fmt.Fprintf(globals.Stdout, "  %s\n", homebrewCmd)
	fmt.Fprintln(globals.Stdout)
	fmt.Fprintln(globals.Stdout, "To upgrade via Go:")
	fmt.Fprintf(globals.Stdout, "  %s\n", goInstallCmd)
	fmt.Fprintln(globals.Stdout)
	fmt.Fprintln(globals.Stdout, "For release notes, see:")
	fmt.Fprintf(globals.Stdout, "  %s\n", releasesURL)

	return nil
}
