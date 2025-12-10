package filter

import (
	"sync"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// DedupeFilter collapses repeated identical messages
type DedupeFilter struct {
	mu           sync.Mutex
	window       time.Duration   // Time window for deduplication (0 = consecutive only)
	seen         map[string]*dedupeEntry
	lastMessage  string
	lastEmitTime time.Time
}

type dedupeEntry struct {
	count     int
	firstSeen time.Time
	lastSeen  time.Time
}

// NewDedupeFilter creates a new deduplication filter
// window=0 means only collapse consecutive identical messages
// window>0 means collapse identical messages within the time window
func NewDedupeFilter(window time.Duration) *DedupeFilter {
	return &DedupeFilter{
		window: window,
		seen:   make(map[string]*dedupeEntry),
	}
}

// DedupeResult holds the result of a dedupe check
type DedupeResult struct {
	ShouldEmit  bool      // Whether this entry should be emitted
	Count       int       // Number of duplicates (1 = first occurrence)
	FirstSeen   time.Time // First occurrence timestamp
	LastSeen    time.Time // Last occurrence timestamp (same as FirstSeen if count=1)
}

// Check determines if a log entry should be emitted or suppressed
func (f *DedupeFilter) Check(entry *domain.LogEntry) DedupeResult {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := entry.Message
	now := time.Now()

	// Clean up old entries if using window mode
	if f.window > 0 {
		f.cleanOldEntries(now)
	}

	// Check if we've seen this message
	if existing, ok := f.seen[key]; ok {
		existing.count++
		existing.lastSeen = now

		// In window mode, always suppress duplicates within window
		if f.window > 0 {
			return DedupeResult{
				ShouldEmit: false,
				Count:      existing.count,
				FirstSeen:  existing.firstSeen,
				LastSeen:   existing.lastSeen,
			}
		}

		// In consecutive mode, only suppress if same as last message
		if f.lastMessage == key {
			return DedupeResult{
				ShouldEmit: false,
				Count:      existing.count,
				FirstSeen:  existing.firstSeen,
				LastSeen:   existing.lastSeen,
			}
		}
	}

	// New message or different from last (in consecutive mode)
	f.seen[key] = &dedupeEntry{
		count:     1,
		firstSeen: now,
		lastSeen:  now,
	}
	f.lastMessage = key
	f.lastEmitTime = now

	return DedupeResult{
		ShouldEmit: true,
		Count:      1,
		FirstSeen:  now,
		LastSeen:   now,
	}
}

// GetPendingDuplicates returns entries with count > 1 that haven't been reported
// This can be called periodically to emit duplicate summaries
func (f *DedupeFilter) GetPendingDuplicates() map[string]*dedupeEntry {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make(map[string]*dedupeEntry)
	for key, entry := range f.seen {
		if entry.count > 1 {
			result[key] = &dedupeEntry{
				count:     entry.count,
				firstSeen: entry.firstSeen,
				lastSeen:  entry.lastSeen,
			}
		}
	}
	return result
}

// Reset clears the deduplication state
func (f *DedupeFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = make(map[string]*dedupeEntry)
	f.lastMessage = ""
}

// cleanOldEntries removes entries outside the time window
func (f *DedupeFilter) cleanOldEntries(now time.Time) {
	cutoff := now.Add(-f.window)
	for key, entry := range f.seen {
		if entry.lastSeen.Before(cutoff) {
			delete(f.seen, key)
		}
	}
}
