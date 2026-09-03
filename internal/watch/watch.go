// Package watch polls Noita's clip folder for gifs and reports each one
// exactly once, but only after Noita has finished writing it.
//
// It polls with os.ReadDir on a timer rather than using fsnotify: one
// directory, human-scale event rates, and a dependency is something a player
// has to trust just to run this tool. Polling also gets startup discovery —
// clips already in the folder before the app starts — for free, since the
// first poll sees them the same way a later one sees a new arrival.
//
// Completeness is structural, not guessed. A gif is only surfaced once all
// three hold:
//
//  1. its size is unchanged across two polls at least stablePeriod apart
//  2. it starts with a GIF header (GIF89a or GIF87a)
//  3. its last byte is 0x3B, the GIF trailer
//
// (1) is the cheap gate, checked from a directory listing so most ticks never
// open a file. (2) and (3) are the real test, verified against 94 real clips
// from a real screenshots_animated folder — see issue #3.
package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// stablePeriod is how long a gif's size must hold before it is even worth
// checking for the GIF trailer. Chosen to be comfortably longer than one
// filesystem metadata tick, not tuned against a real write's duration.
const stablePeriod = time.Second

// snapshot is what a pending file looked like the last time its size was
// judged worth comparing against.
type snapshot struct {
	size      int64
	checkedAt time.Time
}

// Watcher tracks *.gif files under a folder across repeated polls, and
// reports each one exactly once — the moment it is confirmed complete.
//
// A Watcher holds no open file handles or filesystem watches between calls
// to Poll: it only ever reads a directory listing and, for a candidate that
// looks stable, the file's first six and last one bytes. That is what keeps
// it safe to point at a folder Noita is actively writing into.
type Watcher struct {
	dir      string
	pending  map[string]snapshot
	reported map[string]bool
	now      func() time.Time
}

// New creates a Watcher for the given folder. It does not touch the
// filesystem until Poll is called.
func New(dir string) *Watcher {
	return &Watcher{
		dir:      dir,
		pending:  make(map[string]snapshot),
		reported: make(map[string]bool),
		now:      time.Now,
	}
}

// Poll checks the folder once and returns the full paths of any clips that
// have become complete since the last call. Clips still being written, or
// not yet stable for long enough to trust, are silently skipped — they will
// be reconsidered on a later Poll.
//
// A file that disappears — deleted or renamed outside the app — is dropped
// from tracking without being reported. That applies to reported clips too:
// if a reported clip's name later goes missing, the name is forgotten, so a
// new file later written under the same name is judged fresh rather than
// treated as a repeat of one the player already removed.
func (w *Watcher) Poll() ([]string, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, err
	}

	current := make(map[string]int64, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".gif") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			// Deleted or renamed between ReadDir and Info — treat it as not
			// present this poll rather than failing the whole call.
			continue
		}
		current[e.Name()] = info.Size()
	}

	forgetMissing(w.pending, current)
	forgetMissing(w.reported, current)

	var done []string
	now := w.now()
	for name, size := range current {
		if w.reported[name] {
			continue
		}

		prev, seen := w.pending[name]
		switch {
		case !seen || size != prev.size:
			w.pending[name] = snapshot{size: size, checkedAt: now}
		case now.Sub(prev.checkedAt) < stablePeriod:
			// Stable so far, but not for long enough yet.
		default:
			path := filepath.Join(w.dir, name)
			if isCompleteGIF(path) {
				delete(w.pending, name)
				w.reported[name] = true
				done = append(done, path)
			} else {
				// Size is stable but the trailer isn't there — a slow or
				// paused write. Reset the clock and look again later.
				w.pending[name] = snapshot{size: size, checkedAt: now}
			}
		}
	}

	return done, nil
}

// forgetMissing deletes every key from tracked that no longer has an entry
// in current, so a file removed outside the app is neither re-reported nor
// held onto forever.
func forgetMissing[V any](tracked map[string]V, current map[string]int64) {
	for name := range tracked {
		if _, ok := current[name]; !ok {
			delete(tracked, name)
		}
	}
}

// Run polls dir every interval and sends the path of each newly complete
// clip on clips, until ctx is done. It returns ctx.Err() when that happens.
//
// A ReadDir failure — the folder briefly gone, e.g. between polls — is not
// treated as fatal: it is retried on the next tick, since Noita's folders
// are outside this app's control and a miss here does not mean it will
// still be missing a moment later.
func Run(ctx context.Context, dir string, interval time.Duration, clips chan<- string) error {
	w := New(dir)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			found, err := w.Poll()
			if err != nil {
				continue
			}
			for _, path := range found {
				select {
				case clips <- path:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}
