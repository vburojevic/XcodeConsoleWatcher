package domain

import "time"

// SessionStart is emitted when a new app session begins (detected via PID change)
type SessionStart struct {
	Type          string `json:"type"`                      // "session_start"
	SchemaVersion int    `json:"schemaVersion"`             // 1
	Alert         string `json:"alert,omitempty"`           // "APP_RELAUNCHED" when previous session existed
	Session       int    `json:"session"`                   // Session number (1, 2, 3...)
	PID           int    `json:"pid"`                       // Current process ID
	PreviousPID   int    `json:"previous_pid,omitempty"`    // Previous PID (if app relaunched)
	App           string `json:"app"`                       // Bundle identifier
	Simulator     string `json:"simulator"`                 // Simulator name
	UDID          string `json:"udid"`                      // Simulator UDID
	Timestamp     string `json:"timestamp"`                 // ISO8601 timestamp
}

// SessionEnd is emitted when an app session ends (PID changes or stream stops)
type SessionEnd struct {
	Type          string         `json:"type"`          // "session_end"
	SchemaVersion int            `json:"schemaVersion"` // 1
	Session       int            `json:"session"`       // Session number that ended
	PID           int            `json:"pid"`           // Process ID that ended
	Summary       SessionSummary `json:"summary"`       // Summary of the session
}

// SessionSummary contains statistics about a completed session
type SessionSummary struct {
	TotalLogs       int `json:"total_logs"`
	Errors          int `json:"errors"`
	Faults          int `json:"faults"`
	DurationSeconds int `json:"duration_seconds"`
}

// NewSessionStart creates a new SessionStart event
func NewSessionStart(session, pid, previousPID int, app, simulator, udid string) *SessionStart {
	s := &SessionStart{
		Type:          "session_start",
		SchemaVersion: 1,
		Session:       session,
		PID:           pid,
		App:           app,
		Simulator:     simulator,
		UDID:          udid,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
	if previousPID > 0 {
		s.Alert = "APP_RELAUNCHED"
		s.PreviousPID = previousPID
	}
	return s
}

// NewSessionEnd creates a new SessionEnd event
func NewSessionEnd(session, pid int, summary SessionSummary) *SessionEnd {
	return &SessionEnd{
		Type:          "session_end",
		SchemaVersion: 1,
		Session:       session,
		PID:           pid,
		Summary:       summary,
	}
}
