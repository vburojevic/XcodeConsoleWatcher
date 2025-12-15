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

func TestWatchMaxLogs_WithStubXcrun(t *testing.T) {
	stubDir := t.TempDir()
	xcrunPath := filepath.Join(stubDir, "xcrun")

	// Stub xcrun simctl calls used by WatchCmd:
	// - list devices --json (device resolution)
	// - spawn <udid> log stream (streaming)
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

if [ "$#" -ge 5 ] && [ "$1" = "simctl" ] && [ "$2" = "spawn" ] && [ "$4" = "log" ] && [ "$5" = "stream" ]; then
  echo '{"timestamp":"2025-12-15 00:00:00.000000+0000","messageType":"Error","processImagePath":"/Applications/MyApp.app/MyApp","processID":123,"threadID":1,"subsystem":"com.example.myapp","category":"network","eventMessage":"Watch error","eventType":"logEvent","processImageUUID":"UUID-123","senderImagePath":""}'
  # Keep the process alive; WatchCmd should stop us after --max-logs.
  # Use exec so the process we spawned is the one that sleeps (no orphan child).
  exec sleep 60
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
	cmd := &WatchCmd{
		Booted:              true,
		App:                 "com.example.myapp",
		OnError:             "/usr/bin/true",
		TriggerNoShell:      true,
		Cooldown:            "0s",
		TriggerTimeout:      "2s",
		MaxParallelTriggers: 1,
		TriggerOutput:       "discard",
		MaxLogs:             1,
	}

	require.NoError(t, cmd.Run(globals))

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.NotEmpty(t, lines)

	types := make(map[string]bool)
	for _, line := range lines {
		var v map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &v))
		typ, _ := v["type"].(string)
		if typ != "" {
			types[typ] = true
		}
	}

	require.True(t, types["log"], "expected at least one log entry")
	require.True(t, types["trigger"], "expected trigger event")
	require.True(t, types["trigger_result"], "expected trigger_result event")
	require.True(t, types["cutoff_reached"], "expected cutoff_reached on --max-logs")

	var last map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &last))
	require.Equal(t, "cutoff_reached", last["type"])
	require.Equal(t, "max_logs", last["reason"])
}
