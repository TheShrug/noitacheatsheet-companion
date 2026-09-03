package queue

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TheShrug/noitacheatsheet-companion/internal/noita"
)

func stubWands(wands ...Wand) WandReader {
	return func(io.Reader) ([]Wand, error) { return wands, nil }
}

func writePlayerXML(t *testing.T, save string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(save, "player.xml"), []byte("<Entity/>"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectUsesCurrentRunSessionStartWhenSeedMatches(t *testing.T) {
	save := t.TempDir()
	writePlayerXML(t, save)
	writeWorldState(t, save, "??STA/sessions/20260808-135431")
	writeStats(t, save, "20260808-135431", "776668009")

	install := noita.Install{Save: save}
	clip := filepath.Join(save, "..", "noita-20260808-140000-776668009-00000001.gif")

	entry, ok, err := Detect(install, clip, stubWands(Wand{Name: "Bolt"}), time.Now())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ok {
		t.Fatal("Detect returned ok = false for a well-formed clip name")
	}

	want := RunKey{Seed: "776668009", SessionStart: "20260808-135431"}
	if entry.Run != want {
		t.Errorf("Run = %+v, want %+v (session start from world_state.xml, not the clip's own timestamp)", entry.Run, want)
	}
	if len(entry.Wands) != 1 || entry.Wands[0].Name != "Bolt" {
		t.Errorf("Wands = %v, want [{Bolt}]", entry.Wands)
	}
	if entry.Path != clip {
		t.Errorf("Path = %q, want %q", entry.Path, clip)
	}
}

// A clip from a run that has already ended — or a backlog clip found before
// world_state.xml reflects any run at all — has no XML-free way to recover
// its true session start, so the clip's own capture timestamp stands in.
func TestDetectFallsBackToClipTimestampWhenSeedDoesNotMatchCurrentRun(t *testing.T) {
	save := t.TempDir()
	writePlayerXML(t, save)
	writeWorldState(t, save, "??STA/sessions/20260808-135431")
	writeStats(t, save, "20260808-135431", "111111111") // a different, current run

	clip := filepath.Join(save, "..", "noita-20260101-090000-222222222-00000001.gif")

	entry, ok, err := Detect(noita.Install{Save: save}, clip, stubWands(), time.Now())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ok {
		t.Fatal("Detect returned ok = false for a well-formed clip name")
	}

	want := RunKey{Seed: "222222222", SessionStart: "20260101-090000"}
	if entry.Run != want {
		t.Errorf("Run = %+v, want %+v", entry.Run, want)
	}
}

func TestDetectSkipsAFilenameThatDoesNotMatchTheGrammar(t *testing.T) {
	save := t.TempDir()
	writePlayerXML(t, save)

	entry, ok, err := Detect(noita.Install{Save: save}, filepath.Join(save, "..", "asdf.gif"), stubWands(), time.Now())
	if err != nil {
		t.Fatalf("Detect returned an error for a hand-named file: %v", err)
	}
	if ok {
		t.Errorf("Detect returned ok = true for a hand-named file, entry = %+v", entry)
	}
}

func TestDetectErrorsWhenPlayerXMLIsUnreadable(t *testing.T) {
	save := t.TempDir() // no player.xml written

	clip := filepath.Join(save, "..", "noita-20260808-140000-776668009-00000001.gif")
	_, ok, err := Detect(noita.Install{Save: save}, clip, stubWands(), time.Now())
	if !ok {
		t.Fatal("Detect returned ok = false; the filename is well-formed, the error should come from player.xml")
	}
	if err == nil {
		t.Fatal("Detect returned no error with no player.xml present")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Detect error = %v, want it to wrap os.ErrNotExist", err)
	}
}
