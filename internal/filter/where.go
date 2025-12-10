package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/vburojevic/xcw/internal/domain"
)

// WhereClause represents a parsed --where condition
type WhereClause struct {
	Field    string
	Operator string
	Value    string
	regex    *regexp.Regexp // Compiled regex for ~ and !~ operators
}

// ParseWhereClause parses a where clause like "level=error" or "message~timeout"
// Supported operators: =, !=, ~, !~, >=, <=, ^, $
func ParseWhereClause(clause string) (*WhereClause, error) {
	// Try operators in order of length (longest first to avoid partial matches)
	operators := []string{"!~", ">=", "<=", "!=", "~", "=", "^", "$"}

	for _, op := range operators {
		idx := strings.Index(clause, op)
		if idx > 0 {
			field := strings.TrimSpace(clause[:idx])
			value := strings.TrimSpace(clause[idx+len(op):])

			if field == "" || value == "" {
				return nil, fmt.Errorf("invalid where clause: %s", clause)
			}

			wc := &WhereClause{
				Field:    field,
				Operator: op,
				Value:    value,
			}

			// Pre-compile regex for ~ and !~ operators
			if op == "~" || op == "!~" {
				re, err := regexp.Compile(value)
				if err != nil {
					return nil, fmt.Errorf("invalid regex in where clause '%s': %w", clause, err)
				}
				wc.regex = re
			}

			return wc, nil
		}
	}

	return nil, fmt.Errorf("no valid operator found in where clause: %s (use =, !=, ~, !~, >=, <=, ^, $)", clause)
}

// Match checks if a log entry matches this where clause
func (wc *WhereClause) Match(entry *domain.LogEntry) bool {
	// Get the field value from the entry
	fieldValue := wc.getFieldValue(entry)

	switch wc.Operator {
	case "=":
		return fieldValue == wc.Value
	case "!=":
		return fieldValue != wc.Value
	case "~": // Contains (regex)
		if wc.regex != nil {
			return wc.regex.MatchString(fieldValue)
		}
		return strings.Contains(fieldValue, wc.Value)
	case "!~": // Not contains (regex)
		if wc.regex != nil {
			return !wc.regex.MatchString(fieldValue)
		}
		return !strings.Contains(fieldValue, wc.Value)
	case "^": // Starts with
		return strings.HasPrefix(fieldValue, wc.Value)
	case "$": // Ends with
		return strings.HasSuffix(fieldValue, wc.Value)
	case ">=": // Greater or equal (for levels)
		return wc.compareLevel(entry, true)
	case "<=": // Less or equal (for levels)
		return wc.compareLevel(entry, false)
	}

	return false
}

// getFieldValue extracts the field value from a log entry
func (wc *WhereClause) getFieldValue(entry *domain.LogEntry) string {
	switch strings.ToLower(wc.Field) {
	case "level":
		return string(entry.Level)
	case "subsystem":
		return entry.Subsystem
	case "category":
		return entry.Category
	case "process":
		return entry.Process
	case "message":
		return entry.Message
	case "pid":
		return strconv.Itoa(entry.PID)
	default:
		return ""
	}
}

// compareLevel handles >= and <= comparisons for log levels
func (wc *WhereClause) compareLevel(entry *domain.LogEntry, greaterOrEqual bool) bool {
	if strings.ToLower(wc.Field) != "level" {
		return false
	}

	targetLevel := domain.ParseLogLevel(wc.Value)
	entryPriority := entry.Level.Priority()
	targetPriority := targetLevel.Priority()

	if greaterOrEqual {
		return entryPriority >= targetPriority
	}
	return entryPriority <= targetPriority
}

// WhereFilter is a filter that applies multiple where clauses (AND logic)
type WhereFilter struct {
	clauses []*WhereClause
}

// NewWhereFilter creates a filter from multiple where clause strings
func NewWhereFilter(whereClauses []string) (*WhereFilter, error) {
	if len(whereClauses) == 0 {
		return nil, nil
	}

	filter := &WhereFilter{}
	for _, clause := range whereClauses {
		wc, err := ParseWhereClause(clause)
		if err != nil {
			return nil, err
		}
		filter.clauses = append(filter.clauses, wc)
	}

	return filter, nil
}

// Match returns true if the entry matches ALL where clauses (AND logic)
func (f *WhereFilter) Match(entry *domain.LogEntry) bool {
	for _, clause := range f.clauses {
		if !clause.Match(entry) {
			return false
		}
	}
	return true
}
