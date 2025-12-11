package cli

import (
	"errors"
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
)

// outputErrorCommon normalizes error emission across commands, respecting
// ndjson vs text formats so AI agents always get machine-readable failures.
func outputErrorCommon(globals *Globals, code, message string, hint ...string) error {
	if globals != nil && globals.Format == "ndjson" {
		output.NewNDJSONWriter(globals.Stdout).WriteError(code, message, hint...)
	} else if globals != nil {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s", code, message)
		if len(hint) > 0 && hint[0] != "" {
			fmt.Fprintf(globals.Stderr, " (hint: %s)", hint[0])
		}
		fmt.Fprintln(globals.Stderr)
	}
	return errors.New(message)
}
