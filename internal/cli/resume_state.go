package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type resumeState struct {
	Type              string `json:"type"` // "resume_state"
	SchemaVersion     int    `json:"schemaVersion"`
	App               string `json:"app"`
	UDID              string `json:"udid,omitempty"`
	LastSeenTimestamp string `json:"last_seen_timestamp,omitempty"`
	LastLogTimestamp  string `json:"last_log_timestamp,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

func defaultResumeStatePath(app string) (string, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return "", errors.New("app is required for resume state path")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".xcw", "resume")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := app + ".json"
	return filepath.Join(dir, filename), nil
}

func loadResumeState(path string) (*resumeState, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("resume state path is required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var st resumeState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func saveResumeState(path string, st *resumeState) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("resume state path is required")
	}
	if st == nil {
		return errors.New("resume state is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func parseRFC3339Any(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	// Try nano first (what we emit), fall back to second precision.
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
