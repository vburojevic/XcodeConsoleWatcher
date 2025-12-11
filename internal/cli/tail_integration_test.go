package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
)

// Basic compilation-level test to ensure emit helpers are wired.
// Full integration would require simulator, so we keep it minimal here.
func TestTailEmitHelperWiring(t *testing.T) {
	require.NotEmpty(t, defaultHints())
}

func TestNDJSONWriterLifecycleSnippets(t *testing.T) {
	buf := &bytes.Buffer{}
	w := output.NewNDJSONWriter(buf)

	require.NoError(t, w.WriteHeartbeat(&output.Heartbeat{
		Type:              "heartbeat",
		SchemaVersion:     output.SchemaVersion,
		Timestamp:         "2025-12-11T00:00:00Z",
		UptimeSeconds:     5,
		LogsSinceLast:     2,
		TailID:            "tail-1",
		LatestSession:     2,
		LastSeenTimestamp: "2025-12-11T00:00:00Z",
	}))
	require.NoError(t, w.WriteCutoff("max_duration", "tail-1", 2, 42))
	require.NoError(t, w.WriteReconnect("reconnecting", "tail-1", "warn"))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 3)

	// Heartbeat
	var hb map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &hb))
	require.Equal(t, "heartbeat", hb["type"])
	require.EqualValues(t, output.SchemaVersion, hb["schemaVersion"])
	require.EqualValues(t, 2, hb["latest_session"])

	// Cutoff
	var cutoff map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &cutoff))
	require.Equal(t, "cutoff_reached", cutoff["type"])
	require.EqualValues(t, output.SchemaVersion, cutoff["schemaVersion"])
	require.Equal(t, "max_duration", cutoff["reason"])

	// Reconnect notice
	var rc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(lines[2]), &rc))
	require.Equal(t, "reconnect_notice", rc["type"])
	require.Equal(t, "warn", rc["severity"])
}

func TestEmitterSessionLifecycle(t *testing.T) {
	buf := &bytes.Buffer{}
	em := output.NewEmitter(buf)

	start := domain.NewSessionStartWithMeta(1, 123, 0, "com.example", "Sim", "UDID", "tail-1", "1.0", "1", "uuid-1", "")
	end := domain.NewSessionEndWithMeta(1, 123, domain.SessionSummary{TotalLogs: 5}, "tail-1")

	require.NoError(t, em.SessionStart(start))
	require.NoError(t, em.SessionEnd(end))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	var s map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &s))
	require.Equal(t, "session_start", s["type"])
	require.EqualValues(t, output.SchemaVersion, s["schemaVersion"])
}
