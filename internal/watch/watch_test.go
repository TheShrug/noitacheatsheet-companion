package watch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeClock lets a test control Watcher.now without a real sleep, since
// stablePeriod is a full second and the suite runs with -race.
type fakeClock struct {
	t time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Now()} }

func (c *fakeClock) now() time.Time { return c.t }

func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestWatcher(dir string, clock *fakeClock) *Watcher {
	w := New(dir)
	w.now = clock.now
	return w
}

func TestPollDoesNotSurfaceAFreshlySeenClip(t *testing.T) {
	dir := t.TempDir()
	writeGif(t, dir, "clip.gif", validGIF())

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Poll = %v on first sighting, want none until size has held for stablePeriod", done)
	}
}

func TestPollSurfacesAClipOnceItIsStable(t *testing.T) {
	dir := t.TempDir()
	path := writeGif(t, dir, "clip.gif", validGIF())

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	clock.advance(stablePeriod)
	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 1 || done[0] != path {
		t.Fatalf("Poll = %v, want [%s]", done, path)
	}

	// A third poll must not report it again.
	clock.advance(stablePeriod)
	done, err = w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Poll = %v on third call, want none — a clip is reported exactly once", done)
	}
}

func TestPollDoesNotSurfaceAClipBeforeStablePeriodElapses(t *testing.T) {
	dir := t.TempDir()
	writeGif(t, dir, "clip.gif", validGIF())

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	clock.advance(stablePeriod / 2)
	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Poll = %v before stablePeriod elapsed, want none", done)
	}
}

func TestPollDoesNotSurfaceAGrowingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.gif")
	if err := os.WriteFile(path, []byte("GIF89a\x01"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}

	// Noita appends more frames before the size has held for stablePeriod.
	clock.advance(stablePeriod)
	if err := os.WriteFile(path, []byte("GIF89a\x01\x02\x03\x04\x05"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 0 {
		t.Fatalf("Poll = %v for a file that just grew, want none", done)
	}

	// Now it holds still and finishes — the clock must have reset, so this
	// alone is not enough yet.
	if err := os.WriteFile(path, append([]byte("GIF89a\x01\x02\x03\x04\x05"), gifTrailer), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	clock.advance(stablePeriod / 2)
	done, err = w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 0 {
		t.Fatalf("Poll = %v too soon after the last size change, want none", done)
	}

	clock.advance(stablePeriod)
	done, err = w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 1 || done[0] != path {
		t.Fatalf("Poll = %v, want [%s] once the final size has held", done, path)
	}
}

func TestPollDoesNotSurfaceAStableButIncompleteFile(t *testing.T) {
	dir := t.TempDir()
	// Stable size, valid header, but Noita paused mid-write with no trailer.
	path := filepath.Join(dir, "clip.gif")
	if err := os.WriteFile(path, []byte("GIF89a\x01\x02\x03"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}
	clock.advance(stablePeriod * 3)
	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Poll = %v for a size-stable file with no trailer, want none", done)
	}
}

func TestPollPicksUpAClipAlreadyPresentAtStartup(t *testing.T) {
	dir := t.TempDir()
	path := writeGif(t, dir, "clip.gif", validGIF())

	clock := newFakeClock()
	// The Watcher is created after the file already exists, simulating
	// clips left over from before the app started.
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}
	clock.advance(stablePeriod)
	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 1 || done[0] != path {
		t.Fatalf("Poll = %v, want [%s] for a clip present at startup", done, path)
	}
}

func TestPollIgnoresNonGifFiles(t *testing.T) {
	dir := t.TempDir()
	// A stray file left by a 2021 mod, sharing a gif's basename and ending
	// in the same trailer byte by coincidence — must never be surfaced.
	writeGif(t, dir, "clip.xml", append([]byte("<Entity>"), gifTrailer))

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}
	clock.advance(stablePeriod * 3)
	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Poll = %v, want none — a .xml file must never be surfaced", done)
	}
}

func TestPollDropsAPendingFileDeletedBeforeItStabilizes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clip.gif")
	if err := os.WriteFile(path, []byte("GIF89a\x01"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(w.pending) != 1 {
		t.Fatalf("pending = %d entries, want 1 before delete", len(w.pending))
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}

	clock.advance(stablePeriod)
	done, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll after delete: %v", err)
	}
	if len(done) != 0 {
		t.Errorf("Poll = %v for a deleted file, want none", done)
	}
	if len(w.pending) != 0 {
		t.Errorf("pending = %d entries after delete, want 0", len(w.pending))
	}
}

func TestPollForgetsAReportedFileOnceDeletedAndAllowsARewrite(t *testing.T) {
	dir := t.TempDir()
	path := writeGif(t, dir, "clip.gif", validGIF())

	clock := newFakeClock()
	w := newTestWatcher(dir, clock)

	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}
	clock.advance(stablePeriod)
	done, err := w.Poll()
	if err != nil || len(done) != 1 {
		t.Fatalf("Poll = %v, %v, want the clip reported once", done, err)
	}
	if len(w.reported) != 1 {
		t.Fatalf("reported = %d entries, want 1", len(w.reported))
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll after delete: %v", err)
	}
	if len(w.reported) != 0 {
		t.Errorf("reported = %d entries after delete, want 0 — deleting must not resurrect the clip nor block a future one with the same name", len(w.reported))
	}

	// A new clip written under the same name is a genuinely new event.
	writeGif(t, dir, "clip.gif", validGIF())
	if _, err := w.Poll(); err != nil {
		t.Fatalf("Poll: %v", err)
	}
	clock.advance(stablePeriod)
	done, err = w.Poll()
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(done) != 1 || done[0] != path {
		t.Fatalf("Poll = %v, want [%s] for the rewritten clip", done, path)
	}
}

func TestPollErrorsOnAMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	w := New(dir)

	if _, err := w.Poll(); err == nil {
		t.Error("Poll error = nil, want an error for a missing directory")
	}
}

func TestRunStopsWhenContextIsCanceled(t *testing.T) {
	dir := t.TempDir()
	clips := make(chan string)

	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { done <- Run(ctx, dir, time.Millisecond, clips) }()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after ctx was canceled")
	}
}
