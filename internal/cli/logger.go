package cli

import "go.uber.org/zap"

// agentLogger wraps zap for verbose debug with tail/session context.
type agentLogger struct {
	sugared   *zap.SugaredLogger
	globals   *Globals
	tailID    string
	sessionFn func() int
}

func newAgentLogger(globals *Globals, tailID string, sessionFn func() int) *agentLogger {
	if globals == nil || !globals.Verbose {
		return &agentLogger{}
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	cfg.Encoding = "json"
	logger, _ := cfg.Build()
	return &agentLogger{
		sugared:   logger.Sugar(),
		globals:   globals,
		tailID:    tailID,
		sessionFn: sessionFn,
	}
}

func (l *agentLogger) Debug(format string, args ...interface{}) {
	if l.sugared == nil {
		return
	}
	session := 0
	if l.sessionFn != nil {
		session = l.sessionFn()
	}
	l.sugared.With("tail_id", l.tailID, "session", session).Debugf(format, args...)
}
