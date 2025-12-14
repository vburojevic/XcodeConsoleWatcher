package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultResumeStatePath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got, err := defaultResumeStatePath("com.example.myapp")
	require.NoError(t, err)

	want := filepath.Join(tmp, ".xcw", "resume", "com.example.myapp.json")
	require.Equal(t, want, got)

	info, err := os.Stat(filepath.Dir(got))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestLoadResumeStateMissingFile(t *testing.T) {
	tmp := t.TempDir()
	got, err := loadResumeState(filepath.Join(tmp, "missing.json"))
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestSaveAndLoadResumeStateRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "resume.json")

	st := &resumeState{
		Type:              "resume_state",
		SchemaVersion:     1,
		App:               "com.example.myapp",
		UDID:              "ABC123",
		LastSeenTimestamp: "2025-12-14T22:00:00Z",
		LastLogTimestamp:  "2025-12-14T22:00:01.123456789Z",
		UpdatedAt:         "2025-12-14T22:00:02Z",
	}
	require.NoError(t, saveResumeState(path, st))

	loaded, err := loadResumeState(path)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.Equal(t, st, loaded)
}

func TestParseRFC3339Any(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, err := parseRFC3339Any("")
		require.NoError(t, err)
		require.True(t, got.IsZero())
	})

	t.Run("rfc3339", func(t *testing.T) {
		in := "2025-12-14T22:00:00Z"
		got, err := parseRFC3339Any(in)
		require.NoError(t, err)
		require.Equal(t, time.Date(2025, 12, 14, 22, 0, 0, 0, time.UTC), got)
	})

	t.Run("rfc3339nano", func(t *testing.T) {
		in := "2025-12-14T22:00:00.123456789Z"
		got, err := parseRFC3339Any(in)
		require.NoError(t, err)
		require.Equal(t, time.Date(2025, 12, 14, 22, 0, 0, 123456789, time.UTC), got)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseRFC3339Any("not-a-time")
		require.Error(t, err)
	})
}
