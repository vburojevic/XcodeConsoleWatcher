package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpJsonSynced(t *testing.T) {
	globals, stdout, _ := testGlobals("ndjson")
	cmd := &HelpCmd{JSON: true}
	require.NoError(t, cmd.Run(globals))

	expectedPath := filepath.Join("..", "..", "docs", "help.json")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err, "docs/help.json missing; run make docs")

	require.JSONEq(t, string(expected), stdout.String())
}
