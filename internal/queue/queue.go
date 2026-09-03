// Package queue keeps the local list of clips waiting for review, grouped by
// the Noita run that produced them. It is a list, not a spool: nothing here
// uploads anything, and an entry only leaves the queue by being pruned or by
// the player reviewing it elsewhere.
package queue

import (
	"io"
	"os"
	"sync"
	"time"
)

// Wand is the minimal shape this package needs to store and display a wand
// alongside a clip. The real parse of player.xml lands in internal/noita
// (issue #2, not yet on this branch); this type is this package's own,
// deliberately small, seam for it rather than a dependency on that work.
type Wand struct {
	Name string `json:"name"`
}

// WandReader parses whatever player.xml holds into wands. It exists so this
// package can be built and tested against a stub before the real parser
// lands, and so it never depends on that package's concrete types.
type WandReader func(r io.Reader) ([]Wand, error)

// RunKey identifies one Noita run. Seed alone is not unique — a replayed
// fixed seed produces two runs sharing it — so a run is (seed, session
// start). Both fields are decimal/timestamp strings taken verbatim from the
// filename or the save files, never parsed into numbers: nothing here does
// arithmetic on them, only comparison, and a string avoids any question of
// width or a leading zero.
type RunKey struct {
	Seed         string `json:"seed"`
	SessionStart string `json:"session_start"`
}

// Entry is one clip in the queue.
type Entry struct {
	// Path is where the clip lives on disk. A missing file at this path is
	// how an entry is recognized as stale — see Queue.Prune.
	Path string `json:"path"`

	Run RunKey `json:"run"`

	// DetectedAt is when this entry was added, not when the clip was
	// recorded — the gif filename already carries that.
	DetectedAt time.Time `json:"detected_at"`

	// Wands is a snapshot of player.xml taken when the clip was first seen,
	// not read again at review time. A wand discarded since stays correct,
	// and the clip stays reviewable after a new run starts.
	Wands []Wand `json:"wands"`
}

// Queue is an in-memory list of Entry, safe for concurrent use. It is
// deliberately unopinionated about persistence and detection — see Load/Save
// and Detect — so it can be tested without touching a filesystem or a save.
type Queue struct {
	mu      sync.Mutex
	entries []Entry
}

// New returns an empty queue.
func New() *Queue {
	return &Queue{}
}

// Add appends an entry to the queue.
func (q *Queue) Add(e Entry) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, e)
}

// Entries returns a copy of the queue's entries, in the order they were
// added.
func (q *Queue) Entries() []Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Entry, len(q.entries))
	copy(out, q.entries)
	return out
}

// Prune drops entries whose clip file no longer exists — deleted by the
// player, or by Noita itself — and returns how many were dropped. This is
// the only way an entry leaves the queue on its own; nothing else in this
// package deletes clips or marks them reviewed.
func (q *Queue) Prune() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	kept := q.entries[:0]
	dropped := 0
	for _, e := range q.entries {
		if _, err := os.Stat(e.Path); err != nil {
			dropped++
			continue
		}
		kept = append(kept, e)
	}
	q.entries = kept
	return dropped
}

// Run is one group of entries sharing a RunKey.
type Run struct {
	Key     RunKey
	Entries []Entry
}

// Runs groups the queue's entries by RunKey, preserving the order runs were
// first seen in.
func (q *Queue) Runs() []Run {
	q.mu.Lock()
	defer q.mu.Unlock()

	var order []RunKey
	byKey := make(map[RunKey][]Entry)
	for _, e := range q.entries {
		if _, seen := byKey[e.Run]; !seen {
			order = append(order, e.Run)
		}
		byKey[e.Run] = append(byKey[e.Run], e)
	}

	runs := make([]Run, len(order))
	for i, k := range order {
		runs[i] = Run{Key: k, Entries: byKey[k]}
	}
	return runs
}
