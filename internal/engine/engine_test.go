package engine

import (
	"testing"
	"time"
)

func TestShouldBackOff_RecentCrashes(t *testing.T) {
	e := &Engine{
		crashTimes: []time.Time{
			time.Now().Add(-2 * time.Second),
			time.Now().Add(-1 * time.Second),
			time.Now(),
		},
	}
	if !e.shouldBackOff() {
		t.Error("expected backoff with 3 recent crashes")
	}
}

func TestShouldBackOff_OldCrashes(t *testing.T) {
	e := &Engine{
		crashTimes: []time.Time{
			time.Now().Add(-60 * time.Second),
			time.Now().Add(-60 * time.Second),
			time.Now().Add(-60 * time.Second),
		},
	}
	if e.shouldBackOff() {
		t.Error("expected no backoff with old crashes")
	}
}

func TestShouldBackOff_BelowThreshold(t *testing.T) {
	e := &Engine{
		crashTimes: []time.Time{
			time.Now().Add(-1 * time.Second),
			time.Now(),
		},
	}
	if e.shouldBackOff() {
		t.Error("expected no backoff with only 2 crashes")
	}
}

func TestShouldBackOff_Empty(t *testing.T) {
	e := &Engine{}
	if e.shouldBackOff() {
		t.Error("expected no backoff with no crashes")
	}
}

func TestShouldBackOff_MixedOldAndNew(t *testing.T) {
	e := &Engine{
		crashTimes: []time.Time{
			time.Now().Add(-60 * time.Second),
			time.Now().Add(-60 * time.Second),
			time.Now().Add(-1 * time.Second),
			time.Now(),
		},
	}
	if e.shouldBackOff() {
		t.Error("expected no backoff with only 2 recent crashes")
	}
}

func TestShouldBackOff_PrunesOldEntries(t *testing.T) {
	e := &Engine{
		crashTimes: []time.Time{
			time.Now().Add(-120 * time.Second),
			time.Now().Add(-90 * time.Second),
			time.Now().Add(-60 * time.Second),
			time.Now().Add(-1 * time.Second),
		},
	}
	e.shouldBackOff()
	if len(e.crashTimes) != 1 {
		t.Errorf("expected 1 recent crash after pruning, got %d", len(e.crashTimes))
	}
}
