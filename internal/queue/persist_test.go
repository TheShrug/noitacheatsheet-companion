package queue

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "queue.json")

	q := New()
	q.Add(Entry{
		Path:       "clip.gif",
		Run:        RunKey{Seed: "1", SessionStart: "20260101-000000"},
		DetectedAt: time.Now().Truncate(time.Second),
		Wands:      []Wand{{Name: "Bolt"}},
	})

	if err := q.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := Load(path)
	got := loaded.Entries()
	want := q.Entries()
	if len(got) != len(want) {
		t.Fatalf("Load returned %d entries, want %d", len(got), len(want))
	}
	if got[0].Path != want[0].Path || got[0].Run != want[0].Run || got[0].Wands[0].Name != want[0].Wands[0].Name {
		t.Errorf("Load = %+v, want %+v", got[0], want[0])
	}
	if !got[0].DetectedAt.Equal(want[0].DetectedAt) {
		t.Errorf("DetectedAt = %v, want %v", got[0].DetectedAt, want[0].DetectedAt)
	}
}

func TestLoadMissingFileIsEmptyQueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist", "queue.json")
	q := Load(path)
	if entries := q.Entries(); len(entries) != 0 {
		t.Errorf("Load of a missing file returned %d entries, want 0", len(entries))
	}
}

func TestLoadCorruptFileIsEmptyQueue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	q := Load(path)
	if entries := q.Entries(); len(entries) != 0 {
		t.Errorf("Load of a corrupt file returned %d entries, want 0", len(entries))
	}
}

// A crash mid-write must never leave a half-written queue.json in place of a
// good one. Save writes to a temp file and renames over the target, so the
// directory should hold only the final file once Save returns.
func TestSaveLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queue.json")

	q := New()
	q.Add(Entry{Path: "clip.gif"})
	if err := q.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "queue.json" {
		t.Errorf("directory after Save = %v, want only queue.json", entries)
	}
}

func TestConfigPathIsUnderTheDedicatedConfigDir(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}
	if filepath.Base(path) != "queue.json" {
		t.Errorf("ConfigPath = %s, want it to end in queue.json", path)
	}
	if filepath.Base(filepath.Dir(path)) != "noitacheatsheet-companion" {
		t.Errorf("ConfigPath = %s, want it under a noitacheatsheet-companion directory", path)
	}
}
