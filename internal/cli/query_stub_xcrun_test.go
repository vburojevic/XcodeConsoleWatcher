package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/config"
)

func TestQueryIncludesFault_WhenMaxLevelUnset_WithStubXcrun(t *testing.T) {
	stubDir := t.TempDir()
	xcrunPath := filepath.Join(stubDir, "xcrun")

	// Stub xcrun simctl calls used by QueryCmd:
	// - list devices --json (device resolution)
	// - spawn <udid> log show (query)
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

if [ "$#" -ge 5 ] && [ "$1" = "simctl" ] && [ "$2" = "spawn" ] && [ "$4" = "log" ] && [ "$5" = "show" ]; then
  echo '{"timestamp":"2025-12-15 00:00:00.000000+0000","messageType":"Fault","processImagePath":"/Applications/MyApp.app/MyApp","processID":123,"threadID":1,"subsystem":"com.example.myapp","category":"network","eventMessage":"Catastrophic failure","eventType":"logEvent","processImageUUID":"UUID-123","senderImagePath":""}'
  echo '{"timestamp":"2025-12-15 00:00:01.000000+0000","messageType":"Error","processImagePath":"/Applications/MyApp.app/MyApp","processID":123,"threadID":1,"subsystem":"com.example.myapp","category":"network","eventMessage":"Regular error","eventType":"logEvent","processImageUUID":"UUID-123","senderImagePath":""}'
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
		Quiet:  true,
		Stdout: &stdout,
		Stderr: &stderr,
		Config: config.Default(),
	}
	cmd := &QueryCmd{
		Booted: true,
		App:    "com.example.myapp",
		Since:  "5m",
		Limit:  10,
	}

	require.NoError(t, cmd.Run(globals))

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.NotEmpty(t, lines)

	var sawFault bool
	for _, line := range lines {
		var v map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &v))
		if v["type"] != "log" {
			continue
		}
		if v["level"] == "Fault" {
			sawFault = true
		}
	}

	require.True(t, sawFault, "expected Fault log to be included when --max-level is unset")
}
