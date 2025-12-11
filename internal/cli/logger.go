package cli

import "fmt"

// agentLogger prefixes debug output with tail/session identifiers so AI agents
// can correlate logs even when multiple tails run concurrently.
type agentLogger struct {
	globals   *Globals
	tailID    string
	sessionFn func() int
}

func newAgentLogger(globals *Globals, tailID string, sessionFn func() int) *agentLogger {
	return &agentLogger{
		globals:   globals,
		tailID:    tailID,
		sessionFn: sessionFn,
	}
}

func (l *agentLogger) Debug(format string, args ...interface{}) {
	if l.globals == nil {
		return
	}
	prefix := ""
	if l.tailID != "" {
		prefix = fmt.Sprintf("[tail=%s", l.tailID)
		if l.sessionFn != nil {
			prefix += fmt.Sprintf(" session=%d", l.sessionFn())
		}
		prefix += "] "
	}
	l.globals.Debug(prefix+format, args...)
}
