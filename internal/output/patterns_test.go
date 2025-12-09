package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatternStore(t *testing.T) {
	t.Run("creates store with default path when empty", func(t *testing.T) {
		store := NewPatternStore("")
		require.NotNil(t, store)
		assert.Contains(t, store.path, ".xcw")
		assert.Contains(t, store.path, "patterns.json")
	})

	t.Run("creates store with custom path", func(t *testing.T) {
		customPath := "/tmp/custom-patterns.json"
		store := NewPatternStore(customPath)
		require.NotNil(t, store)
		assert.Equal(t, customPath, store.path)
	})

	t.Run("initializes with empty patterns", func(t *testing.T) {
		store := NewPatternStore("/tmp/nonexistent-patterns.json")
		assert.Equal(t, 0, store.Count())
	})
}

func TestPatternStore_RecordPattern(t *testing.T) {
	store := NewPatternStore("")
	store.Clear()

	t.Run("returns true for new pattern", func(t *testing.T) {
		isNew := store.RecordPattern("error pattern 1", 5)
		assert.True(t, isNew)
		assert.Equal(t, 1, store.Count())
	})

	t.Run("returns false for existing pattern", func(t *testing.T) {
		isNew := store.RecordPattern("error pattern 1", 3)
		assert.False(t, isNew)
		assert.Equal(t, 1, store.Count())
	})

	t.Run("updates total count for existing pattern", func(t *testing.T) {
		p := store.GetPattern("error pattern 1")
		require.NotNil(t, p)
		assert.Equal(t, 8, p.TotalCount) // 5 + 3
	})

	t.Run("tracks first and last seen times", func(t *testing.T) {
		p := store.GetPattern("error pattern 1")
		require.NotNil(t, p)
		assert.False(t, p.FirstSeen.IsZero())
		assert.False(t, p.LastSeen.IsZero())
		assert.True(t, p.LastSeen.After(p.FirstSeen) || p.LastSeen.Equal(p.FirstSeen))
	})
}

func TestPatternStore_IsKnown(t *testing.T) {
	store := NewPatternStore("")
	store.Clear()

	t.Run("returns false for unknown pattern", func(t *testing.T) {
		assert.False(t, store.IsKnown("unknown pattern"))
	})

	t.Run("returns true for known pattern", func(t *testing.T) {
		store.RecordPattern("known pattern", 1)
		assert.True(t, store.IsKnown("known pattern"))
	})
}

func TestPatternStore_GetAllPatterns(t *testing.T) {
	store := NewPatternStore("")
	store.Clear()

	store.RecordPattern("pattern 1", 1)
	store.RecordPattern("pattern 2", 2)
	store.RecordPattern("pattern 3", 3)

	patterns := store.GetAllPatterns()
	assert.Len(t, patterns, 3)
}

func TestPatternStore_Clear(t *testing.T) {
	store := NewPatternStore("")
	store.RecordPattern("pattern 1", 1)
	store.RecordPattern("pattern 2", 2)

	store.Clear()
	assert.Equal(t, 0, store.Count())
}

func TestPatternStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "patterns.json")

	// Create and populate store
	store := NewPatternStore(path)
	store.RecordPattern("error <n>", 5)
	store.RecordPattern("timeout at <addr>", 3)

	// Save
	err := store.Save()
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(path)
	require.NoError(t, err)

	// Read file content to verify structure
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var file patternsFile
	err = json.Unmarshal(data, &file)
	require.NoError(t, err)
	assert.Equal(t, 1, file.Version)
	assert.Len(t, file.Patterns, 2)

	// Create new store and load
	store2 := NewPatternStore(path)
	assert.Equal(t, 2, store2.Count())
	assert.True(t, store2.IsKnown("error <n>"))
	assert.True(t, store2.IsKnown("timeout at <addr>"))

	p := store2.GetPattern("error <n>")
	require.NotNil(t, p)
	assert.Equal(t, 5, p.TotalCount)
}

func TestPatternStore_LoadNonexistent(t *testing.T) {
	store := NewPatternStore("/nonexistent/path/patterns.json")
	err := store.Load()
	assert.NoError(t, err) // Should not error for missing file
	assert.Equal(t, 0, store.Count())
}

func TestPatternStore_AnnotatePatterns(t *testing.T) {
	store := NewPatternStore("")
	store.Clear()

	// Add a known pattern
	store.RecordPattern("known error", 10)

	// Patterns to annotate
	patterns := []PatternMatch{
		{Pattern: "known error", Count: 3, Samples: []string{"sample 1"}},
		{Pattern: "new error", Count: 2, Samples: []string{"sample 2"}},
	}

	enhanced := store.AnnotatePatterns(patterns)

	assert.Len(t, enhanced, 2)

	// Known pattern
	assert.False(t, enhanced[0].IsNew)
	assert.NotNil(t, enhanced[0].FirstSeen)
	assert.Equal(t, 10, enhanced[0].TotalCount)

	// New pattern
	assert.True(t, enhanced[1].IsNew)
	assert.Nil(t, enhanced[1].FirstSeen)
	assert.Equal(t, 0, enhanced[1].TotalCount)
}

func TestPatternStore_RecordPatterns(t *testing.T) {
	store := NewPatternStore("")
	store.Clear()

	// Pre-record one pattern
	store.RecordPattern("existing error", 5)

	patterns := []PatternMatch{
		{Pattern: "existing error", Count: 3, Samples: []string{"sample"}},
		{Pattern: "new error", Count: 2, Samples: []string{"sample"}},
	}

	enhanced := store.RecordPatterns(patterns)

	assert.Len(t, enhanced, 2)

	// Existing pattern - not new, count updated
	assert.False(t, enhanced[0].IsNew)
	assert.Equal(t, 8, enhanced[0].TotalCount) // 5 + 3

	// New pattern
	assert.True(t, enhanced[1].IsNew)
	assert.Equal(t, 2, enhanced[1].TotalCount)

	// Verify store was updated
	assert.Equal(t, 2, store.Count())
	assert.True(t, store.IsKnown("new error"))
}

func TestPatternStore_Concurrency(t *testing.T) {
	store := NewPatternStore("")
	store.Clear()

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				store.RecordPattern("concurrent pattern", 1)
				store.IsKnown("concurrent pattern")
				store.GetPattern("concurrent pattern")
				store.Count()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no race conditions
	assert.True(t, store.IsKnown("concurrent pattern"))
	p := store.GetPattern("concurrent pattern")
	require.NotNil(t, p)
	assert.Equal(t, 1000, p.TotalCount)
}

func TestEnhancedPatternMatch(t *testing.T) {
	now := time.Now()
	enhanced := EnhancedPatternMatch{
		PatternMatch: PatternMatch{
			Pattern: "test pattern",
			Count:   5,
			Samples: []string{"sample 1", "sample 2"},
		},
		IsNew:      false,
		FirstSeen:  &now,
		TotalCount: 15,
	}

	// Verify JSON marshaling
	data, err := json.Marshal(enhanced)
	require.NoError(t, err)

	var decoded EnhancedPatternMatch
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "test pattern", decoded.Pattern)
	assert.Equal(t, 5, decoded.Count)
	assert.False(t, decoded.IsNew)
	assert.Equal(t, 15, decoded.TotalCount)
}
