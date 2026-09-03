package queue

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("gif"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestPruneDropsEntriesWhoseFileIsGone(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "present.gif")
	touch(t, present)
	gone := filepath.Join(dir, "gone.gif")
	touch(t, gone)
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}

	q := New()
	q.Add(Entry{Path: present})
	q.Add(Entry{Path: gone})

	dropped := q.Prune()
	if dropped != 1 {
		t.Errorf("Prune dropped %d entries, want 1", dropped)
	}

	entries := q.Entries()
	if len(entries) != 1 || entries[0].Path != present {
		t.Errorf("Entries after Prune = %v, want only %s", entries, present)
	}
}

func TestPruneOnAnUntouchedQueueDropsNothing(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "present.gif")
	touch(t, present)

	q := New()
	q.Add(Entry{Path: present})

	if dropped := q.Prune(); dropped != 0 {
		t.Errorf("Prune dropped %d entries, want 0", dropped)
	}
}

func TestRunsGroupsEntriesByRunKey(t *testing.T) {
	runA := RunKey{Seed: "1", SessionStart: "20260101-000000"}
	runB := RunKey{Seed: "2", SessionStart: "20260102-000000"}

	q := New()
	q.Add(Entry{Path: "a1.gif", Run: runA})
	q.Add(Entry{Path: "b1.gif", Run: runB})
	q.Add(Entry{Path: "a2.gif", Run: runA})

	runs := q.Runs()
	if len(runs) != 2 {
		t.Fatalf("Runs returned %d groups, want 2", len(runs))
	}

	// First-seen order: runA's first entry came before runB's.
	if runs[0].Key != runA {
		t.Errorf("Runs[0].Key = %+v, want %+v", runs[0].Key, runA)
	}
	if len(runs[0].Entries) != 2 {
		t.Errorf("runA has %d entries, want 2", len(runs[0].Entries))
	}
	if runs[1].Key != runB {
		t.Errorf("Runs[1].Key = %+v, want %+v", runs[1].Key, runB)
	}
}

// A replayed fixed seed produces two runs sharing a seed — this is exactly
// why RunKey carries session start too, and this test pins that a seed match
// alone must not merge two runs.
func TestRunsKeepsReplayedSeedsSeparate(t *testing.T) {
	first := RunKey{Seed: "42", SessionStart: "20260101-000000"}
	replay := RunKey{Seed: "42", SessionStart: "20260102-000000"}

	q := New()
	q.Add(Entry{Path: "a.gif", Run: first})
	q.Add(Entry{Path: "b.gif", Run: replay})

	if runs := q.Runs(); len(runs) != 2 {
		t.Errorf("Runs returned %d groups for two runs sharing a seed, want 2", len(runs))
	}
}

func TestEntriesReturnsACopy(t *testing.T) {
	q := New()
	q.Add(Entry{Path: "a.gif", DetectedAt: time.Now()})

	entries := q.Entries()
	entries[0].Path = "mutated.gif"

	if got := q.Entries()[0].Path; got != "a.gif" {
		t.Errorf("mutating a returned slice affected the queue: Path = %q", got)
	}
}
