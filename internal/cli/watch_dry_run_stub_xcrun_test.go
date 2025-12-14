package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/config"
)

func TestWatchDryRunJSON_WithStubXcrun(t *testing.T) {
	stubDir := t.TempDir()
	xcrunPath := filepath.Join(stubDir, "xcrun")

	script := `#!/bin/sh
set -eu

if [ "$#" -ge 4 ] && [ "$1" = "simctl" ] && [ "$2" = "list" ] && [ "$3" = "devices" ] && [ "$4" = "--json" ]; then
  cat <<'EOF'
{
  "devices": {
    "com.apple.CoreSimulator.SimRuntime.iOS-17-0": [
      {
        "udid": "TEST-UDID-123",
        "name": "iPhone 17 Pro",
        "state": "Booted",
        "isAvailable": true,
        "deviceTypeIdentifier": "com.apple.CoreSimulator.SimDeviceType.iPhone-17-Pro",
        "dataPath": "/tmp",
        "logPath": "/tmp"
      }
    ]
  }
}
EOF
  exit 0
fi

echo "stub: unsupported xcrun args: $*" >&2
exit 1
`
	require.NoError(t, os.WriteFile(xcrunPath, []byte(script), 0o755))

	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	globals := &Globals{
		Format: "ndjson",
		Level:  "debug",
		Quiet:  false,
		Stdout: &stdout,
		Stderr: &stderr,
		Config: config.Default(),
	}
	cmd := &WatchCmd{
		Booted:              true,
		App:                 "com.example.myapp",
		Cooldown:            "5s",
		TriggerTimeout:      "30s",
		MaxParallelTriggers: 5,
		TriggerOutput:       "discard",
		DryRunJSON:          true,
	}

	require.NoError(t, cmd.Run(globals))

	var out map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out))

	stream, ok := out["stream"].(map[string]any)
	require.True(t, ok, "expected stream object")
	require.Equal(t, "com.example.myapp", stream["BundleID"])

	require.Equal(t, "5s", out["cooldown"])
	require.Equal(t, "30s", out["trigger_timeout"])
	require.Equal(t, "discard", out["trigger_output"])
}
