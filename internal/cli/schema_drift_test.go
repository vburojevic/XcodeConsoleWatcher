package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Ensures generated schema is in sync with checked-in schema file.
func TestSchemaDrift(t *testing.T) {
	globals, stdout, _ := testGlobals("ndjson")
	cmd := &SchemaCmd{}
	require.NoError(t, cmd.Run(globals))

	expectedPath := filepath.Join("..", "..", "schemas", "generated.schema.json")
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err, "schemas/generated.schema.json missing; run make schema")

	require.JSONEq(t, string(expected), stdout.String())
}
