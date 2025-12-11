package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFlags(t *testing.T) {
	globals := &Globals{Format: "ndjson", Quiet: false, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	require.Error(t, validateFlags(globals, true, true))

	globals = &Globals{Format: "text", Quiet: true, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	require.Error(t, validateFlags(globals, false, false))

	globals = &Globals{Format: "ndjson", Quiet: false, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	require.NoError(t, validateFlags(globals, false, false))
}
