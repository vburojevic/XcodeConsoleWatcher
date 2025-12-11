package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	t.Helper()
	dec := json.NewDecoder(buf)
	var m map[string]interface{}
	require.NoError(t, dec.Decode(&m))
	return m
}

func TestWriteAgentHints(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewNDJSONWriter(buf)

	err := w.WriteAgentHints("tail-123", 2, []string{"h1", "h2"})
	require.NoError(t, err)

	m := decodeLine(t, buf)
	require.Equal(t, "agent_hints", m["type"])
	require.EqualValues(t, 1, m["schemaVersion"])
	require.Equal(t, "tail-123", m["tail_id"])
	require.EqualValues(t, 2, m["session"])
	require.EqualValues(t, 1, m["contract_version"])
	require.Equal(t, "tail_id + latest session only", m["recommended_scope"])
	hints, ok := m["hints"].([]interface{})
	require.True(t, ok)
	require.Len(t, hints, 2)
}

func TestWriteClearBuffer(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewNDJSONWriter(buf)

	err := w.WriteClearBuffer("session_end", "tail-xyz", 3)
	require.NoError(t, err)

	m := decodeLine(t, buf)
	require.Equal(t, "clear_buffer", m["type"])
	require.Equal(t, "session_end", m["reason"])
	require.Equal(t, "tail-xyz", m["tail_id"])
	require.EqualValues(t, 3, m["session"])
	hints, ok := m["hints"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, hints)
}

func TestWriteReadyIncludesTailAndSession(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewNDJSONWriter(buf)

	err := w.WriteReady("2025-12-11T10:00:00Z", "iPhone 17 Pro", "UDID", "com.example.app", "tail-abc", 1)
	require.NoError(t, err)

	m := decodeLine(t, buf)
	require.Equal(t, "ready", m["type"])
	require.Equal(t, "tail-abc", m["tail_id"])
	require.EqualValues(t, 1, m["session"])
	require.EqualValues(t, 1, m["contract_version"])
}

func TestHeartbeatContractFields(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewNDJSONWriter(buf)

	hb := &Heartbeat{Type: "heartbeat", SchemaVersion: SchemaVersion, Timestamp: "2025-12-11T10:01:00Z", UptimeSeconds: 5, LogsSinceLast: 2, TailID: "tail-1", ContractVersion: 1, LatestSession: 4}
	require.NoError(t, w.WriteHeartbeat(hb))

	m := decodeLine(t, buf)
	require.Equal(t, "heartbeat", m["type"])
	require.EqualValues(t, 1, m["contract_version"])
	require.EqualValues(t, 4, m["latest_session"])
	require.Equal(t, "tail-1", m["tail_id"])
}
