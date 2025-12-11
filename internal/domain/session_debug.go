package domain

// SessionDebug is an optional verbose event describing session transitions.
type SessionDebug struct {
	Type          string          `json:"type"` // session_debug
	SchemaVersion int             `json:"schemaVersion"`
	TailID        string          `json:"tail_id,omitempty"`
	Session       int             `json:"session"`
	PrevSession   int             `json:"prev_session,omitempty"`
	PID           int             `json:"pid"`
	PrevPID       int             `json:"prev_pid,omitempty"`
	Reason        string          `json:"reason"` // e.g., relaunch, idle_timeout
	Summary       *SessionSummary `json:"summary,omitempty"`
}
