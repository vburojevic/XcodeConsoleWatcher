package cli

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

// Ensure embedded flag structs keep flag names/aliases working for agents.
func TestTailFlagsParse(t *testing.T) {
	var c CLI
	parser, err := kong.New(&c)
	require.NoError(t, err)

	_, err = parser.Parse([]string{
		"tail",
		"-s", "iPhone 17 Pro",
		"-a", "com.example.app",
		"--filter", "timeout",
		"--where", "level=error",
		"--dedupe",
		"--output", "out.ndjson",
		"--heartbeat", "5s",
		"--summary-interval", "1m",
		"--session-idle", "30s",
	})
	require.NoError(t, err)

	require.Equal(t, "iPhone 17 Pro", c.Tail.Simulator)
	require.Equal(t, "com.example.app", c.Tail.App)
	require.Equal(t, "timeout", c.Tail.Pattern)
	require.Contains(t, c.Tail.Where, "level=error")
	require.True(t, c.Tail.Dedupe)
	require.Equal(t, "out.ndjson", c.Tail.Output)
	require.Equal(t, "5s", c.Tail.Heartbeat)
	require.Equal(t, "1m", c.Tail.SummaryInterval)
	require.Equal(t, "30s", c.Tail.SessionIdle)
}
