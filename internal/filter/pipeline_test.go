package filter

import (
	"regexp"
	"testing"

	"github.com/vburojevic/xcw/internal/domain"
)

func TestPipeline_MatchOrder(t *testing.T) {
	pat := regexp.MustCompile("ok")
	ex1 := regexp.MustCompile("ignore")
	where, err := NewWhereFilter([]string{"level=Error"})
	if err != nil {
		t.Fatalf("where build failed: %v", err)
	}
	p := NewPipeline(pat, []*regexp.Regexp{ex1}, where)

	entry := &domain.LogEntry{Message: "ok message", Level: domain.LogLevelError}
	if !p.Match(entry) {
		t.Fatalf("expected entry to match pipeline")
	}

	entry2 := &domain.LogEntry{Message: "ignore this ok message", Level: domain.LogLevelError}
	if p.Match(entry2) {
		t.Fatalf("expected exclude to drop entry")
	}

	entry3 := &domain.LogEntry{Message: "ok message", Level: domain.LogLevelInfo}
	if p.Match(entry3) {
		t.Fatalf("expected where to drop non-error entry")
	}
}

func TestPipeline_NilIsAllowAll(t *testing.T) {
	if NewPipeline(nil, nil, nil) != nil {
		t.Fatalf("expected nil pipeline when no filters provided")
	}
	p := NewPipeline(nil, nil, nil)
	entry := &domain.LogEntry{Message: "anything"}
	if !p.Match(entry) {
		t.Fatalf("nil pipeline should allow all")
	}
}
