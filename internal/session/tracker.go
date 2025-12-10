package session

import (
	"sync"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// Tracker monitors log entries for PID changes to detect app relaunches
type Tracker struct {
	mu             sync.Mutex
	currentSession int
	currentPID     int
	sessionStart   time.Time
	logCount       int
	errorCount     int
	faultCount     int
	app            string
	simulator      string
	udid           string
	initialized    bool
}

// SessionChange contains events emitted when a session changes
type SessionChange struct {
	EndSession   *domain.SessionEnd
	StartSession *domain.SessionStart
}

// NewTracker creates a new session tracker
func NewTracker(app, simulator, udid string) *Tracker {
	return &Tracker{
		app:       app,
		simulator: simulator,
		udid:      udid,
	}
}

// CheckEntry processes a log entry and returns a SessionChange if the app was relaunched
func (t *Tracker) CheckEntry(entry *domain.LogEntry) *SessionChange {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Only track entries matching our app's bundle ID
	if entry.Subsystem != t.app && !t.matchesApp(entry) {
		// Still increment counts if we're tracking
		if t.initialized {
			t.logCount++
			t.updateCounts(entry)
		}
		return nil
	}

	pid := entry.PID

	// First entry - initialize session
	if !t.initialized {
		t.initialized = true
		t.currentSession = 1
		t.currentPID = pid
		t.sessionStart = time.Now()
		t.logCount = 1
		t.updateCounts(entry)

		// Return initial session start
		return &SessionChange{
			StartSession: domain.NewSessionStart(
				t.currentSession,
				pid,
				0, // no previous PID
				t.app,
				t.simulator,
				t.udid,
			),
		}
	}

	// PID changed - app was relaunched
	if pid != t.currentPID && pid > 0 {
		previousPID := t.currentPID
		previousSession := t.currentSession

		// Create session end summary
		summary := domain.SessionSummary{
			TotalLogs:       t.logCount,
			Errors:          t.errorCount,
			Faults:          t.faultCount,
			DurationSeconds: int(time.Since(t.sessionStart).Seconds()),
		}

		// Start new session
		t.currentSession++
		t.currentPID = pid
		t.sessionStart = time.Now()
		t.logCount = 1
		t.errorCount = 0
		t.faultCount = 0
		t.updateCounts(entry)

		return &SessionChange{
			EndSession: domain.NewSessionEnd(previousSession, previousPID, summary),
			StartSession: domain.NewSessionStart(
				t.currentSession,
				pid,
				previousPID,
				t.app,
				t.simulator,
				t.udid,
			),
		}
	}

	// Same session - just increment counts
	t.logCount++
	t.updateCounts(entry)
	return nil
}

// matchesApp checks if entry is from our app by process name or subsystem prefix
func (t *Tracker) matchesApp(entry *domain.LogEntry) bool {
	// Check if subsystem starts with our bundle ID
	if len(entry.Subsystem) >= len(t.app) && entry.Subsystem[:len(t.app)] == t.app {
		return true
	}
	return false
}

// updateCounts updates error/fault counts based on log level
func (t *Tracker) updateCounts(entry *domain.LogEntry) {
	switch entry.Level {
	case domain.LogLevelError:
		t.errorCount++
	case domain.LogLevelFault:
		t.faultCount++
	}
}

// CurrentSession returns the current session number
func (t *Tracker) CurrentSession() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.currentSession
}

// GetFinalSummary returns a summary for the current session (for stream end)
func (t *Tracker) GetFinalSummary() *domain.SessionEnd {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		return nil
	}

	return domain.NewSessionEnd(
		t.currentSession,
		t.currentPID,
		domain.SessionSummary{
			TotalLogs:       t.logCount,
			Errors:          t.errorCount,
			Faults:          t.faultCount,
			DurationSeconds: int(time.Since(t.sessionStart).Seconds()),
		},
	)
}

// Stats returns current session statistics
func (t *Tracker) Stats() (session, pid, logs, errors, faults int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.currentSession, t.currentPID, t.logCount, t.errorCount, t.faultCount
}
