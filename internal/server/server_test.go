package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheShrug/noitacheatsheet-companion/internal/queue"
)

func TestIndexShowsTheWandsStoredWithTheClip(t *testing.T) {
	s, _ := newServerWithClip(t, queue.Wand{Name: "Bolt Storm"}, queue.Wand{Name: "Chainsaw"})

	body := get(t, s, "/")

	for _, want := range []string{"Bolt Storm", "Chainsaw", "1234567890", "20260901-140000"} {
		if !strings.Contains(body, want) {
			t.Errorf("the page does not mention %q", want)
		}
	}

	// There is no player.xml anywhere near this test. The page rendering at
	// all is the proof that wands come from the snapshot taken when the clip
	// was detected, not from re-reading the save at review time (issue #4).
}

func TestIndexEscapesWhatItReadFromTheSave(t *testing.T) {
	s, _ := newServerWithClip(t, queue.Wand{Name: "<script>alert(1)</script>"})

	body := get(t, s, "/")

	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Error("a wand name was rendered as markup")
	}
	if !strings.Contains(body, "alert(1)") {
		t.Error("the wand name is missing entirely — it should be shown, escaped")
	}
}

func TestIndexGroupsClipsByRun(t *testing.T) {
	s := newServer(t, DefaultPort)
	dir := t.TempDir()

	for _, e := range []struct{ name, seed, start string }{
		{"noita-20260901-140000-111-1.gif", "111", "20260901-135900"},
		{"noita-20260901-140100-111-2.gif", "111", "20260901-135900"},
		{"noita-20260901-150000-222-1.gif", "222", "20260901-145900"},
	} {
		path := filepath.Join(dir, e.name)
		if err := os.WriteFile(path, gifBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		s.queue.Add(queue.Entry{
			Path:       path,
			Run:        queue.RunKey{Seed: e.seed, SessionStart: e.start},
			DetectedAt: time.Now(),
		})
	}

	body := get(t, s, "/")

	if got := strings.Count(body, "<section class=\"run\">"); got != 2 {
		t.Errorf("the page has %d run sections, want 2", got)
	}
	if got := strings.Count(body, "<article class=\"clip\">"); got != 3 {
		t.Errorf("the page has %d clips, want 3", got)
	}
}

// A clip the player deleted must not be offered, because confirming it would
// promise something that is no longer there.
func TestIndexDropsClipsWhoseFileHasGone(t *testing.T) {
	s, entry := newServerWithClip(t)

	if err := os.Remove(entry.Path); err != nil {
		t.Fatal(err)
	}

	body := get(t, s, "/")

	if strings.Contains(body, entry.ID()) {
		t.Error("the page still offers a clip whose file has been deleted")
	}
	if entries := s.queue.Entries(); len(entries) != 0 {
		t.Errorf("the queue still holds %d entries, want 0", len(entries))
	}

	// Pruning happens in memory. Rendering a page is not a reason to write
	// to disk; the next confirmation persists the pruned queue anyway.
	if _, err := os.Stat(s.queuePath); err == nil {
		t.Error("rendering the page wrote the queue file")
	}
}

func TestClipRouteServesTheGifOnDisk(t *testing.T) {
	s, entry := newServerWithClip(t)

	req := httptest.NewRequest(http.MethodGet, "/clip/"+entry.ID(), nil)
	req.Host = testHost()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /clip/<id> = %d, want 200", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), gifBytes) {
		t.Error("the bytes served are not the file's — nothing here transcodes")
	}
	if got := rec.Header().Get("Content-Type"); got != "image/gif" {
		t.Errorf("Content-Type = %q, want image/gif", got)
	}
}

func TestConfirmMarksTheEntryAndPersistsIt(t *testing.T) {
	s, entry := newServerWithClip(t)
	before := time.Now()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, confirmRequest(s, entry.ID()))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("confirm = %d, want 303", rec.Code)
	}

	got := s.queue.Entries()[0]
	if got.ApprovedAt == nil {
		t.Fatal("the entry was not marked approved")
	}
	if got.ApprovedAt.Before(before) {
		t.Errorf("ApprovedAt = %v, before the request was made", got.ApprovedAt)
	}

	// It survives a restart: the queue file is the app's only state.
	reloaded := queue.Load(s.queuePath).Entries()
	if len(reloaded) != 1 || reloaded[0].ApprovedAt == nil {
		t.Fatalf("the approval was not written to %s", s.queuePath)
	}

	// And the page stops offering to confirm what has been confirmed.
	body := get(t, s, "/")
	if strings.Contains(body, "Confirm this clip") {
		t.Error("a confirmed clip is still showing a confirm button")
	}
	if !strings.Contains(body, "Confirmed") {
		t.Error("the page does not say the clip was confirmed")
	}
}

// Confirming is consent, given once. A second click is not a second decision,
// and must not quietly restamp when it was given.
func TestConfirmingTwiceKeepsTheFirstTime(t *testing.T) {
	s, entry := newServerWithClip(t)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, confirmRequest(s, entry.ID()))
	first := *s.queue.Entries()[0].ApprovedAt

	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, confirmRequest(s, entry.ID()))

	if got := *s.queue.Entries()[0].ApprovedAt; !got.Equal(first) {
		t.Errorf("ApprovedAt moved from %v to %v", first, got)
	}
}

func TestEmptyQueueRendersAnExplanation(t *testing.T) {
	s := newServer(t, DefaultPort)

	body := get(t, s, "/")

	if !strings.Contains(body, "No clips have been queued yet") {
		t.Error("an empty queue should say so, not render a blank page")
	}
}

// Every form the page renders carries the token, or the button it belongs to
// is broken the moment the CSRF check is doing its job.
func TestEveryFormCarriesTheToken(t *testing.T) {
	s, _ := newServerWithClip(t)

	body := get(t, s, "/")

	forms := strings.Count(body, "<form")
	tokens := strings.Count(body, s.csrf)
	if forms == 0 {
		t.Fatal("the page rendered no form, so there is nothing to confirm with")
	}
	if tokens != forms {
		t.Errorf("%d forms but %d tokens rendered", forms, tokens)
	}
}

func get(t *testing.T, s *Server, path string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = testHost()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", path, rec.Code)
	}
	return rec.Body.String()
}
