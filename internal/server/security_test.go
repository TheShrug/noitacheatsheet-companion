package server

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheShrug/noitacheatsheet-companion/internal/queue"
)

// Requirement 1 of ADR 3: loopback, and never anything else. This asserts it
// against the listener the app would actually serve on, because a config
// value saying "127.0.0.1" proves nothing about what net.Listen was handed.
func TestListenBindsLoopbackAndNothingElse(t *testing.T) {
	s := newServer(t, 0)

	ln, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is %T, want *net.TCPAddr", ln.Addr())
	}
	if addr.IP.IsUnspecified() {
		t.Fatalf("bound to %s — that is every interface on the machine", addr.IP)
	}
	if !addr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Errorf("bound to %s, want 127.0.0.1", addr.IP)
	}

	// Port 0 means "whatever the OS gives us", and the URL printed on
	// startup has to be the one that was actually bound.
	if s.port != addr.Port {
		t.Errorf("server port is %d, listener is on %d", s.port, addr.Port)
	}
	if want := fmt.Sprintf("http://127.0.0.1:%d", addr.Port); s.URL() != want {
		t.Errorf("URL() = %q, want %q", s.URL(), want)
	}
}

func TestDefaultPortIsTheDocumentedOne(t *testing.T) {
	if DefaultPort != 7331 {
		t.Errorf("DefaultPort = %d, want 7331 — README, ADR 3 and the tray's Open queue all say 7331", DefaultPort)
	}
}

// Requirement 2: the Host header is what a DNS rebinding attack has to forge,
// so a request that does not address us by one of our own three names is
// refused before any handler sees it.
func TestRejectsRequestsNotAddressedToLoopback(t *testing.T) {
	s := newServer(t, DefaultPort)

	tests := []struct {
		host string
		want int
	}{
		{fmt.Sprintf("127.0.0.1:%d", DefaultPort), http.StatusOK},
		{fmt.Sprintf("localhost:%d", DefaultPort), http.StatusOK},
		{fmt.Sprintf("[::1]:%d", DefaultPort), http.StatusOK},
		{fmt.Sprintf("clips.attacker.example:%d", DefaultPort), http.StatusForbidden},
		{fmt.Sprintf("127.0.0.1.attacker.example:%d", DefaultPort), http.StatusForbidden},
		// Loopback, but not a port we are listening on, so not a page of
		// ours that produced this.
		{"127.0.0.1:9999", http.StatusForbidden},
		// No port at all, and a bare name.
		{"127.0.0.1", http.StatusForbidden},
		{"localhost", http.StatusForbidden},
		{"", http.StatusForbidden},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = tt.host

		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code != tt.want {
			t.Errorf("GET / with Host %q = %d, want %d", tt.host, rec.Code, tt.want)
		}
	}
}

// Requirement 3: a browser sets Origin itself and a page cannot lie about it,
// so a state-changing request that names somewhere else is refused even when
// it somehow carries a valid token.
func TestConfirmRejectsForeignOriginAndReferer(t *testing.T) {
	s, entry := newServerWithClip(t)
	ownOrigin := fmt.Sprintf("http://127.0.0.1:%d", DefaultPort)

	tests := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"our own origin", map[string]string{"Origin": ownOrigin}, http.StatusSeeOther},
		{"our own referer", map[string]string{"Referer": ownOrigin + "/"}, http.StatusSeeOther},
		{"neither header", nil, http.StatusSeeOther},
		{"another site's origin", map[string]string{"Origin": "https://noita.example"}, http.StatusForbidden},
		{"another site's referer", map[string]string{"Referer": "https://noita.example/x"}, http.StatusForbidden},
		// Our address, but not over a scheme we serve, and not on our port.
		{"https on our host", map[string]string{"Origin": fmt.Sprintf("https://127.0.0.1:%d", DefaultPort)}, http.StatusForbidden},
		{"our host, another port", map[string]string{"Origin": "http://127.0.0.1:9999"}, http.StatusForbidden},
		// What a sandboxed iframe or a file:// page sends.
		{"opaque origin", map[string]string{"Origin": "null"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		req := confirmRequest(s, entry.ID())
		for k, v := range tt.headers {
			req.Header.Set(k, v)
		}

		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code != tt.want {
			t.Errorf("confirm with %s = %d, want %d", tt.name, rec.Code, tt.want)
		}
	}
}

// Requirement 4: the token is the check a cross-origin page cannot pass even
// if it guesses the port, forges nothing and sends no headers at all, because
// it cannot read the page the token is rendered into.
func TestConfirmRequiresTheCSRFToken(t *testing.T) {
	s, entry := newServerWithClip(t)

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"the process token", s.csrf, http.StatusSeeOther},
		{"no token", "", http.StatusForbidden},
		{"a guessed token", "not-the-token", http.StatusForbidden},
		// Right length, wrong value: the comparison is on the bytes, not on
		// whether a field is filled in.
		{"a token of the right shape", strings.Repeat("A", len(s.csrf)), http.StatusForbidden},
	}

	for _, tt := range tests {
		form := url.Values{"id": {entry.ID()}}
		if tt.token != "" {
			form.Set(csrfField, tt.token)
		}

		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, formRequest(form))

		if rec.Code != tt.want {
			t.Errorf("confirm with %s = %d, want %d", tt.name, rec.Code, tt.want)
		}
	}

	// And the refusals have to have refused, not just redirected oddly: the
	// entry is approved exactly once, by the one request that was allowed.
	approved := 0
	for _, e := range s.queue.Entries() {
		if e.ApprovedAt != nil {
			approved++
		}
	}
	if approved != 1 {
		t.Errorf("%d entries approved, want 1 — only the request carrying the token should have confirmed anything", approved)
	}
}

func TestCSRFTokenDiffersBetweenProcesses(t *testing.T) {
	a, b := newServer(t, DefaultPort), newServer(t, DefaultPort)
	if a.csrf == b.csrf {
		t.Error("two servers minted the same CSRF token — it is not random")
	}
	if len(a.csrf) < 32 {
		t.Errorf("CSRF token is %d characters, too short to be unguessable", len(a.csrf))
	}
}

// Requirement 5: a request names an entry, never a file. The queue points at
// a folder full of the player's own recordings, and one handler that joins a
// request string onto a directory would hand any of them — or anything else
// readable — to a web page that guessed the port.
func TestClipIDIsNeverTreatedAsAPath(t *testing.T) {
	s, entry := newServerWithClip(t)

	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("SEKRIT"), 0o600); err != nil {
		t.Fatal(err)
	}

	clipDir := filepath.Dir(entry.Path)
	attempts := []string{
		secret,
		filepath.Join(clipDir, "..", "secret.txt"),
		"../../secret.txt",
		"..%2f..%2fsecret.txt",
		filepath.Base(entry.Path), // the clip's own name, rather than its ID
	}

	for _, attempt := range attempts {
		req := httptest.NewRequest(http.MethodGet, "/clip/"+url.PathEscape(attempt), nil)
		req.Host = testHost()

		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Errorf("GET /clip/%s = 200 — a request reached the filesystem", attempt)
		}
		if strings.Contains(rec.Body.String(), "SEKRIT") {
			t.Errorf("GET /clip/%s served a file outside the queue", attempt)
		}
	}

	// The ID for a queued entry still works, so the above is a real check
	// and not the route being broken.
	req := httptest.NewRequest(http.MethodGet, "/clip/"+entry.ID(), nil)
	req.Host = testHost()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /clip/<id> = %d, want 200", rec.Code)
	}
}

func TestConfirmIDIsNeverTreatedAsAPath(t *testing.T) {
	s, entry := newServerWithClip(t)

	form := url.Values{csrfField: {s.csrf}, "id": {entry.Path}}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, formRequest(form))

	if rec.Code != http.StatusNotFound {
		t.Errorf("confirm by path = %d, want 404 — an entry is named by ID, not by where it lives", rec.Code)
	}
	if s.queue.Entries()[0].ApprovedAt != nil {
		t.Error("a confirm naming a path approved an entry")
	}
}

// Requirement 6: fetching a route must not change anything. The methods are
// in the routing patterns, so the wrong one never reaches a handler.
func TestFetchingConfirmChangesNothing(t *testing.T) {
	s, entry := newServerWithClip(t)

	for _, path := range []string{"/confirm", "/confirm?id=" + entry.ID() + "&" + csrfField + "=" + s.csrf} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = testHost()

		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET %s = %d, want 405", path, rec.Code)
		}
	}

	if s.queue.Entries()[0].ApprovedAt != nil {
		t.Error("a GET confirmed a clip")
	}
	if _, err := os.Stat(s.queuePath); err == nil {
		t.Error("a GET wrote the queue file")
	}
}

// --- helpers ---

func testHost() string {
	return fmt.Sprintf("127.0.0.1:%d", DefaultPort)
}

// newServer builds a server over an empty queue whose file does not exist,
// so any test can tell a write from a non-write by whether it appeared.
func newServer(t *testing.T, port int) *Server {
	t.Helper()

	s, err := New(queue.New(), filepath.Join(t.TempDir(), "queue.json"), port)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

// newServerWithClip builds a server holding one entry whose gif is really on
// disk, and returns the entry so a test can name it by ID.
func newServerWithClip(t *testing.T, wands ...queue.Wand) (*Server, queue.Entry) {
	t.Helper()

	clip := filepath.Join(t.TempDir(), "noita-20260901-141530-1234567890-4200.gif")
	if err := os.WriteFile(clip, gifBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	entry := queue.Entry{
		Path:       clip,
		Run:        queue.RunKey{Seed: "1234567890", SessionStart: "20260901-140000"},
		DetectedAt: time.Date(2026, 9, 1, 14, 15, 0, 0, time.UTC),
		Wands:      wands,
	}

	s := newServer(t, DefaultPort)
	s.queue.Add(entry)
	return s, entry
}

// gifBytes is the smallest thing that is honestly a gif: the header and the
// trailer, which is also what internal/watch checks for.
var gifBytes = []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")

func confirmRequest(s *Server, id string) *http.Request {
	return formRequest(url.Values{csrfField: {s.csrf}, "id": {id}})
}

func formRequest(form url.Values) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/confirm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = testHost()
	return req
}
