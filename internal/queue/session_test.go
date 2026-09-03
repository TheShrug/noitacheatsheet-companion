package queue

import (
	"os"
	"path/filepath"
	"testing"
)

func writeWorldState(t *testing.T, save, sessionStatFile string) {
	t.Helper()
	xml := `<Entity><WorldStateComponent session_stat_file="` + sessionStatFile + `"/></Entity>`
	if err := os.WriteFile(filepath.Join(save, "world_state.xml"), []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeStats(t *testing.T, save, sessionStart, seed string) {
	t.Helper()
	dir := filepath.Join(save, "stats", "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	xml := `<stats playtime="25.0333" world_seed="` + seed + `"/>`
	path := filepath.Join(dir, sessionStart+"_stats.xml")
	if err := os.WriteFile(path, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadCurrentRunFollowsWorldStateToStats(t *testing.T) {
	save := t.TempDir()
	writeWorldState(t, save, "??STA/sessions/20260808-135431")
	writeStats(t, save, "20260808-135431", "409296132")

	run, ok := ReadCurrentRun(save)
	if !ok {
		t.Fatal("ReadCurrentRun returned ok = false, want true")
	}
	if run.SessionStart != "20260808-135431" {
		t.Errorf("SessionStart = %q, want %q", run.SessionStart, "20260808-135431")
	}
	if run.Seed != "409296132" {
		t.Errorf("Seed = %q, want %q", run.Seed, "409296132")
	}
}

func TestReadCurrentRunWithNoWorldStateIsNotOK(t *testing.T) {
	if _, ok := ReadCurrentRun(t.TempDir()); ok {
		t.Error("ReadCurrentRun returned ok = true with no world_state.xml present")
	}
}

func TestReadCurrentRunWithCorruptWorldStateIsNotOK(t *testing.T) {
	save := t.TempDir()
	if err := os.WriteFile(filepath.Join(save, "world_state.xml"), []byte("not xml at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadCurrentRun(save); ok {
		t.Error("ReadCurrentRun returned ok = true for unparsable world_state.xml")
	}
}

// world_state.xml can point at a session whose stats file has not been
// written yet, or was already rotated away. That is not this package's
// business to explain — it just means "not confirmably the current run".
func TestReadCurrentRunWithMissingStatsFileIsNotOK(t *testing.T) {
	save := t.TempDir()
	writeWorldState(t, save, "??STA/sessions/20260808-135431")

	if _, ok := ReadCurrentRun(save); ok {
		t.Error("ReadCurrentRun returned ok = true with no matching stats file")
	}
}
