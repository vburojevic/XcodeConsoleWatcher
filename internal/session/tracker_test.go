package session

import (
	"testing"

	"github.com/vburojevic/xcw/internal/domain"
)

func TestTrackerDetectsBinaryUUIDChange(t *testing.T) {
	tr := NewTracker("com.example.app", "Sim", "UDID", "tail-1", "", "")

	// first entry initializes session
	change := tr.CheckEntry(&domain.LogEntry{
		PID:              111,
		ProcessImageUUID: "UUID-1",
		Level:            domain.LogLevelInfo,
	})
	if change == nil || change.StartSession == nil || change.StartSession.Session != 1 {
		t.Fatalf("expected initial session start")
	}

	// same PID but new binary uuid should trigger new session
	change = tr.CheckEntry(&domain.LogEntry{
		PID:              111,
		ProcessImageUUID: "UUID-2",
		Level:            domain.LogLevelInfo,
	})
	if change == nil || change.StartSession == nil || change.StartSession.Session != 2 {
		t.Fatalf("expected session rollover on binary UUID change")
	}
	if change.EndSession == nil || change.EndSession.Session != 1 {
		t.Fatalf("expected previous session to close")
	}
}
