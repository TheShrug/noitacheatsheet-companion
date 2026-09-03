package queue

import (
	"path/filepath"
	"testing"
	"time"
)

// The ID is what the review queue's URLs are built from, so it has to name
// the same clip after a restart and after other entries have come and gone.
func TestIDIsStableAndPerClip(t *testing.T) {
	a := Entry{Path: "/clips/one.gif"}
	b := Entry{Path: "/clips/two.gif"}

	if a.ID() == b.ID() {
		t.Error("two clips share an ID")
	}
	if a.ID() != (Entry{Path: a.Path, DetectedAt: time.Now()}).ID() {
		t.Error("the ID changed with a field that is not the clip's path")
	}
	if a.ID() == "" {
		t.Error("the ID is empty")
	}

	// It is a handle, not a path: nothing in it can be walked back to one.
	if filepath.Base(a.ID()) != a.ID() {
		t.Errorf("ID %q contains a path separator", a.ID())
	}
}

func TestIDSurvivesSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")

	q := New()
	q.Add(Entry{Path: "/clips/one.gif", DetectedAt: time.Now()})
	if err := q.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	want := q.Entries()[0].ID()
	got := Load(path).Entries()[0].ID()
	if got != want {
		t.Errorf("ID after a restart is %q, was %q — a page left open would confirm the wrong clip", got, want)
	}
}

func TestApproveMarksOnlyTheNamedEntry(t *testing.T) {
	q := New()
	q.Add(Entry{Path: "/clips/one.gif"})
	q.Add(Entry{Path: "/clips/two.gif"})

	target := q.Entries()[1]
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	if !q.Approve(target.ID(), at) {
		t.Fatal("Approve did not find an entry that is in the queue")
	}

	entries := q.Entries()
	if entries[0].ApprovedAt != nil {
		t.Error("approving one entry approved another")
	}
	if entries[1].ApprovedAt == nil || !entries[1].ApprovedAt.Equal(at) {
		t.Errorf("ApprovedAt = %v, want %v", entries[1].ApprovedAt, at)
	}
}

// Consent is given once. A second click is not a second decision, so the
// recorded time stays the one the player actually acted at.
func TestApproveKeepsTheFirstTime(t *testing.T) {
	q := New()
	q.Add(Entry{Path: "/clips/one.gif"})
	id := q.Entries()[0].ID()

	first := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	q.Approve(id, first)
	q.Approve(id, first.Add(time.Hour))

	if got := q.Entries()[0].ApprovedAt; !got.Equal(first) {
		t.Errorf("ApprovedAt = %v, want %v", got, first)
	}
}

func TestApproveAndFindMissAnEntryThatIsGone(t *testing.T) {
	q := New()
	q.Add(Entry{Path: "/clips/one.gif"})

	if _, ok := q.Find("not-an-id"); ok {
		t.Error("Find returned an entry for an ID that is not in the queue")
	}
	if q.Approve("not-an-id", time.Now()) {
		t.Error("Approve claimed to have marked an entry that is not in the queue")
	}
}

func TestApprovalRoundTripsThroughTheQueueFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	q := New()
	q.Add(Entry{Path: "/clips/one.gif", DetectedAt: at})
	q.Approve(q.Entries()[0].ID(), at)
	if err := q.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := Load(path).Entries()[0]
	if got.ApprovedAt == nil || !got.ApprovedAt.Equal(at) {
		t.Errorf("ApprovedAt after a restart = %v, want %v", got.ApprovedAt, at)
	}
}

// An entry nobody has confirmed carries no approved_at at all, rather than a
// zero time that reads as "approved in year one".
func TestUnapprovedEntryStaysNilAcrossTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")

	q := New()
	q.Add(Entry{Path: "/clips/one.gif"})
	if err := q.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if got := Load(path).Entries()[0]; got.ApprovedAt != nil {
		t.Errorf("ApprovedAt = %v, want nil", got.ApprovedAt)
	}
}
