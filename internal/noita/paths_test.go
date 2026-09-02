package noita

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeSave writes a save folder containing a player.xml with the given
// modification time, and returns its path.
func makeSave(t *testing.T, root, name string, modified time.Time) string {
	t.Helper()

	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	player := filepath.Join(dir, "player.xml")
	if err := os.WriteFile(player, []byte("<Entity/>"), 0o644); err != nil {
		t.Fatalf("write %s: %v", player, err)
	}
	if err := os.Chtimes(player, modified, modified); err != nil {
		t.Fatalf("chtimes %s: %v", player, err)
	}
	return dir
}

func TestActiveSavePicksNewestPlayerXML(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)

	makeSave(t, root, "save00", old)
	want := makeSave(t, root, "save01", time.Now())
	makeSave(t, root, "save02", old.Add(time.Hour))

	got, err := ActiveSave(root)
	if err != nil {
		t.Fatalf("ActiveSave: %v", err)
	}
	if got != want {
		t.Errorf("ActiveSave = %s, want %s", got, want)
	}
}

// A fresh Proton prefix has the folder but has never been played in. That is
// not an error worth surfacing — Detect has to be able to move on to the next
// candidate — so it must not be mistaken for a usable save.
func TestActiveSaveIgnoresSaveFolderWithoutPlayerXML(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "save00"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := makeSave(t, root, "save01", time.Now())

	got, err := ActiveSave(root)
	if err != nil {
		t.Fatalf("ActiveSave: %v", err)
	}
	if got != want {
		t.Errorf("ActiveSave = %s, want %s", got, want)
	}
}

// save_rec sits beside the save folders and is not one, so a regex looser than
// ^save\d+$ would happily return it and then look for wands in a gif folder.
func TestActiveSaveIgnoresSaveRec(t *testing.T) {
	root := t.TempDir()

	recPlayer := filepath.Join(root, "save_rec", "player.xml")
	if err := os.MkdirAll(filepath.Dir(recPlayer), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recPlayer, []byte("<Entity/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ActiveSave(root); !errors.Is(err, ErrNotFound) {
		t.Errorf("ActiveSave error = %v, want ErrNotFound", err)
	}
}

func TestActiveSaveWithNoSavesIsErrNotFound(t *testing.T) {
	if _, err := ActiveSave(t.TempDir()); !errors.Is(err, ErrNotFound) {
		t.Errorf("ActiveSave error = %v, want ErrNotFound", err)
	}
}

func TestAtResolvesBothFolders(t *testing.T) {
	root := t.TempDir()
	save := makeSave(t, root, "save00", time.Now())

	install, err := At(root)
	if err != nil {
		t.Fatalf("At: %v", err)
	}

	if install.Save != save {
		t.Errorf("Save = %s, want %s", install.Save, save)
	}
	if want := filepath.Join(root, "save_rec", "screenshots_animated"); install.Gifs != want {
		t.Errorf("Gifs = %s, want %s", install.Gifs, want)
	}
	if want := filepath.Join(save, "player.xml"); install.PlayerXML() != want {
		t.Errorf("PlayerXML = %s, want %s", install.PlayerXML(), want)
	}
}

func TestAtOnMissingRootIsErrNotFound(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := At(missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("At error = %v, want ErrNotFound", err)
	}
}

// The gif folder is not inside the save folder. Getting this wrong would make
// the watcher silently see no clips, which looks identical to the player not
// having recorded any.
func TestGifsAreNotUnderTheSaveFolder(t *testing.T) {
	root := t.TempDir()
	makeSave(t, root, "save00", time.Now())

	install, err := At(root)
	if err != nil {
		t.Fatalf("At: %v", err)
	}
	if strings.HasPrefix(install.Gifs, install.Save+string(os.PathSeparator)) {
		t.Errorf("Gifs %s is under Save %s; clips are shared across saves", install.Gifs, install.Save)
	}
}

// Every candidate must be actionable on its own: a path a player can look at,
// and a sentence saying what it is. An empty Note is a candidate that can only
// ever produce an unexplained miss.
func TestCandidatesAreWellFormed(t *testing.T) {
	for _, c := range Candidates() {
		if !filepath.IsAbs(c.Path) {
			t.Errorf("candidate path %q is not absolute", c.Path)
		}
		if strings.TrimSpace(c.Note) == "" {
			t.Errorf("candidate %q has no note", c.Path)
		}
	}
}

func TestExplainNotFoundListsWhatWasTried(t *testing.T) {
	probes := []Probe{
		{Candidate: Candidate{Path: "/one", Note: "first"}},
		{Candidate: Candidate{Path: "/two", Note: "second"}},
	}

	msg := ExplainNotFound(probes)
	for _, want := range []string{"/one", "/two", "--root"} {
		if !strings.Contains(msg, want) {
			t.Errorf("ExplainNotFound missing %q:\n%s", want, msg)
		}
	}
}

// On macOS there are no candidates at all, so the message has to stand on its
// own rather than print an empty list.
func TestExplainNotFoundWithNoCandidates(t *testing.T) {
	msg := ExplainNotFound(nil)
	if strings.Contains(msg, "Looked in:") {
		t.Errorf("ExplainNotFound printed an empty list:\n%s", msg)
	}
	if !strings.Contains(msg, "--root") {
		t.Errorf("ExplainNotFound did not offer --root:\n%s", msg)
	}
}
