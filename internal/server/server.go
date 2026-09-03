// Package server serves the review queue over HTTP, on the loopback
// interface and nowhere else.
//
// This is the consent step from docs/PHILOSOPHY.md principle 2: the player
// sees each clip and the wands that were on the save when it was recorded,
// and confirms them one at a time. Nothing here uploads. Confirming sets a
// timestamp in the local queue file, because the endpoint to send a clip to
// does not exist yet (ADR 2, and issue #7 which is blocked on it) — the
// consent step is built first so there is never a version of this app that
// can upload without one.
//
// It is also a listening socket on someone else's computer, reachable by
// anything else running as that user and guessable by any web page they
// visit. The checks in security.go are what makes that acceptable, and
// ADR 3 calls them non-optional rather than hardening for later.
package server

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/TheShrug/noitacheatsheet-companion/internal/queue"
)

// DefaultPort is the port the review queue listens on unless told otherwise.
// Fixed and printed on startup, the same contract every other app in the
// fleet follows (ADR 3).
const DefaultPort = 7331

//go:embed queue.html
var templates embed.FS

// Server renders the review queue and takes the player's confirmations. One
// Server holds one CSRF token, minted at New and good for the life of the
// process.
type Server struct {
	queue     *queue.Queue
	queuePath string

	// port is the port this server answers on. Listen sets it to the port
	// actually bound, before any request is served, and nothing writes it
	// afterwards. Requests are checked against it — see hostAllowed.
	port int

	csrf string
	tmpl *template.Template

	// confirmMu serializes confirm-and-save, so two clicks arriving together
	// cannot each snapshot the queue and write the older snapshot last,
	// dropping the other's approval from the file.
	confirmMu sync.Mutex
}

// New builds a server over a loaded queue and the file that queue came from,
// which it writes back whenever the player confirms a clip.
//
// port is the port to bind; pass 0 to let the OS choose one, in which case
// Listen reports what it got.
func New(q *queue.Queue, queuePath string, port int) (*Server, error) {
	token, err := newToken()
	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFS(templates, "queue.html")
	if err != nil {
		return nil, fmt.Errorf("parsing the review queue template: %w", err)
	}

	return &Server{
		queue:     q,
		queuePath: queuePath,
		port:      port,
		csrf:      token,
		tmpl:      tmpl,
	}, nil
}

// Listen binds the review queue's socket.
//
// The address is the literal 127.0.0.1 and a port. It is never ":port" and
// never "0.0.0.0:port", both of which bind every interface on the machine and
// would put a player's clip folder on whatever network they are on. There is
// no flag to change that, deliberately: SECURITY.md's "Network — in" row is a
// promise, and this line is the whole of it.
func (s *Server) Listen() (net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(s.port))
	if err != nil {
		return nil, fmt.Errorf("binding 127.0.0.1:%d: %w\nAnother program on this machine may already be using that port; pass --port to choose a different one", s.port, err)
	}

	s.port = ln.Addr().(*net.TCPAddr).Port
	return ln, nil
}

// URL is where the player should point their browser. It is right only once
// Listen has returned, which is what makes it right when the OS chose the
// port.
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

// Serve answers requests on ln until it is closed.
func (s *Server) Serve(ln net.Listener) error {
	srv := &http.Server{
		Handler: s.Handler(),
		// A local socket still has local clients, and one that opens a
		// connection and never finishes its headers should not hold a
		// goroutine for the life of the process.
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.Serve(ln)
}

// Handler is the routing table, wrapped in the checks from security.go.
//
// Every route is either a GET that reads or the one POST that confirms.
// Naming the method in the pattern is what keeps that true: a GET of
// /confirm never reaches the handler, it is a 405, so no route can change
// anything by being fetched.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /clip/{id}", s.handleClip)
	mux.HandleFunc("POST /confirm", s.handleConfirm)
	return s.guard(mux)
}

// page is what queue.html renders. Everything on it is a plain string or a
// nested view: the template does no formatting and no lookups, so what the
// page can say is decided here in Go rather than in the markup.
type page struct {
	CSRFField string
	CSRFToken string
	Runs      []runView
	Waiting   int
}

type runView struct {
	Seed         string
	SessionStart string
	Clips        []clipView
}

type clipView struct {
	ID         string
	Name       string
	DetectedAt string
	Approved   bool
	ApprovedAt string
	Wands      []string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Prune first, so the page never offers a clip that has been deleted
	// since it was queued. It deliberately does not save the pruned queue:
	// a GET writes nothing to disk, and the next confirmation persists it
	// anyway.
	s.queue.Prune()

	data := page{
		CSRFField: csrfField,
		CSRFToken: s.csrf,
	}
	for _, run := range s.queue.Runs() {
		view := runView{Seed: run.Key.Seed, SessionStart: run.Key.SessionStart}
		for _, e := range run.Entries {
			view.Clips = append(view.Clips, clipViewOf(e))
			if e.ApprovedAt == nil {
				data.Waiting++
			}
		}
		data.Runs = append(data.Runs, view)
	}

	// Render into memory first: a template error halfway through Execute
	// would otherwise have already written a 200 and half a page.
	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, data); err != nil {
		http.Error(w, "The review queue page could not be rendered.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(buf.Bytes())
}

func clipViewOf(e queue.Entry) clipView {
	// Wand names come from the snapshot taken when the clip was detected,
	// never from re-reading player.xml: the save has moved on since, and the
	// wand the player held then is the one the clip shows (issue #4).
	names := make([]string, 0, len(e.Wands))
	for _, wand := range e.Wands {
		names = append(names, wand.Name)
	}

	view := clipView{
		ID:         e.ID(),
		Name:       filepath.Base(e.Path),
		DetectedAt: e.DetectedAt.Format("2006-01-02 15:04"),
		Wands:      names,
	}
	if e.ApprovedAt != nil {
		view.Approved = true
		view.ApprovedAt = e.ApprovedAt.Format("2006-01-02 15:04")
	}
	return view
}

// handleClip serves one clip exactly as it is on disk. No transcode and no
// thumbnail: the file is a gif and a browser renders gifs.
//
// The request names an entry, not a file. The path served is the one stored
// in that entry, so a request cannot reach a byte of the filesystem that is
// not already in the queue — which is what matters when the queue points into
// a folder full of the player's own recordings.
func (s *Server) handleClip(w http.ResponseWriter, r *http.Request) {
	entry, ok := s.queue.Find(r.PathValue("id"))
	if !ok {
		http.Error(w, "That clip is not in the queue. Reload the page — it may have been deleted since.", http.StatusNotFound)
		return
	}

	f, err := os.Open(entry.Path)
	if err != nil {
		http.Error(w, "That clip could not be read from disk. Reload the page.", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, "That clip could not be read from disk. Reload the page.", http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, filepath.Base(entry.Path), info.ModTime(), f)
}

// handleConfirm records the player's consent for one clip.
//
// It marks the entry and saves the queue. It does not upload: there is
// nowhere to upload to yet (ADR 2). When there is, this is the only place in
// the app a clip may be sent from.
func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	// guard already parsed the form to check the CSRF token, so this reads
	// the values that parse left behind rather than the body again.
	id := r.PostFormValue("id")

	s.confirmMu.Lock()
	defer s.confirmMu.Unlock()

	if !s.queue.Approve(id, time.Now()) {
		http.Error(w, "That clip is not in the queue. Reload the page — it may have been deleted since.", http.StatusNotFound)
		return
	}

	if err := s.queue.Save(s.queuePath); err != nil {
		http.Error(w, "The clip was confirmed, but the queue could not be written to disk, so it will be waiting again after a restart:\n"+err.Error(), http.StatusInternalServerError)
		return
	}

	// See Other, so reloading the page it lands on does not re-post.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
