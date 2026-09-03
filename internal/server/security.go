package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

// csrfField is the form field every state-changing request carries the
// process's token in.
const csrfField = "csrf_token"

// newToken mints the process's CSRF token: 32 bytes from crypto/rand, which
// is not guessable by a page that has guessed the port. One per process
// rather than one per session, because there are no sessions — the only
// client is the person sitting at the machine.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes for the CSRF token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// guard is the check every request passes through, whatever it asks for.
//
// The threat is not somebody on the network — nothing outside the machine can
// open this socket. It is a web page the player is already looking at, which
// can make their browser send requests to a guessed port on 127.0.0.1 and
// which must not be able to confirm a clip on their behalf. Three things stop
// that, and each is independent of the others:
//
//  1. the Host header, which is what a DNS rebinding attack has to forge to
//     turn a name it controls into an address it does not;
//  2. Origin/Referer on anything that changes state, which a browser sets
//     itself and a page cannot lie about;
//  3. the CSRF token, which a cross-origin page cannot read, and so cannot
//     put in a form it submits.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A gif served here is a gif. Never let a browser decide otherwise.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if !s.hostAllowed(r.Host) {
			http.Error(w, "The review queue answers requests addressed to 127.0.0.1 only. Open "+s.URL()+" directly.", http.StatusForbidden)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if !s.originAllowed(r) {
				http.Error(w, "That request came from another site. The review queue only accepts its own pages.", http.StatusForbidden)
				return
			}
			if !s.csrfValid(r) {
				http.Error(w, "That request did not carry this session's token. Reload "+s.URL()+" and try again.", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// hostAllowed reports whether the Host header names this server.
//
// It is an allowlist of the three spellings of loopback a browser can produce
// for our own port, not a test for "is this address loopback": a name that
// resolves to 127.0.0.1 is exactly what a rebinding attack arrives with, and
// it would pass any test of the address while failing this one.
//
// The port has to match too. A Host naming our address but a port we are not
// listening on did not come from a page served by us.
func (s *Server) hostAllowed(host string) bool {
	name, port, err := net.SplitHostPort(host)
	if err != nil {
		// No port at all. A browser always sends one for a port that is not
		// the scheme's default, and ours never is.
		return false
	}
	if port != strconv.Itoa(s.port) {
		return false
	}

	switch name {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

// originAllowed checks where a state-changing request says it came from.
//
// A missing Origin *and* Referer is allowed: some browsers omit both on an
// ordinary same-origin form post, and a curl with neither is a person on this
// machine, who can already read the queue file directly. That is why the CSRF
// token exists as well — a request that skips this check still has to carry
// something only our own page could have shown it.
//
// A header that is present and names anywhere else is refused outright.
func (s *Server) originAllowed(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		return s.isOwnOrigin(origin)
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		return s.isOwnOrigin(u.Scheme + "://" + u.Host)
	}

	return true
}

// isOwnOrigin reports whether an Origin string is this server's own. The
// scheme has to be http: we serve no https, so an https origin claiming our
// host is somebody else's.
func (s *Server) isOwnOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" {
		return false
	}
	return s.hostAllowed(u.Host)
}

// csrfValid reports whether the request carries this process's token.
//
// The comparison is constant-time, which matters less here than it would
// facing the network — but a local attacker gets to make as many attempts as
// they like, and the constant-time version is the same length to write.
func (s *Server) csrfValid(r *http.Request) bool {
	// ParseForm reads the body; handlers read r.PostForm afterwards rather
	// than the body again. A body that is not a form fails here, which is
	// the right answer for a request that has no token in it either.
	if err := r.ParseForm(); err != nil {
		return false
	}

	got := r.PostFormValue(csrfField)
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.csrf)) == 1
}
