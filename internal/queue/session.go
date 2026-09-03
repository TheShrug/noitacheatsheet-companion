package queue

import (
	"encoding/xml"
	"io"
	"os"
	"path"
	"path/filepath"
)

// CurrentRun is the run world_state.xml says is active right now: its seed
// and when it started. Reading it is the only way to tell a clip from the
// run in progress apart from one recorded during an earlier run that
// happened to share the same seed.
type CurrentRun struct {
	Seed         string
	SessionStart string // YYYYMMDD-HHMMSS
}

// ReadCurrentRun reads world_state.xml and, through it, the current run's
// stats file, to learn the run happening right now in the given save folder.
// Its second return is false whenever that isn't available — the file is
// missing, unparsable, or names a stats file that itself is missing or
// unparsable — never an error: an unreadable world_state.xml just means the
// caller can't confirm a clip belongs to the current run, not that anything
// is broken.
func ReadCurrentRun(save string) (CurrentRun, bool) {
	sessionFile, err := os.Open(filepath.Join(save, "world_state.xml"))
	if err != nil {
		return CurrentRun{}, false
	}
	defer sessionFile.Close()

	raw, ok := findAttr(sessionFile, "session_stat_file")
	if !ok {
		return CurrentRun{}, false
	}

	// raw looks like "??STA/sessions/20260808-135431" — a Noita path token
	// we don't resolve, followed by the session start. Only the last
	// component matters, so take it with the "path" package (raw always
	// uses "/", regardless of host OS) rather than trying to strip a
	// specific prefix.
	sessionStart := path.Base(raw)

	statsFile, err := os.Open(filepath.Join(save, "stats", "sessions", sessionStart+"_stats.xml"))
	if err != nil {
		return CurrentRun{}, false
	}
	defer statsFile.Close()

	seed, ok := findAttr(statsFile, "world_seed")
	if !ok {
		return CurrentRun{}, false
	}

	return CurrentRun{Seed: seed, SessionStart: sessionStart}, true
}

// findAttr scans r for the first element carrying the given attribute,
// anywhere in the document, and returns its value. This is deliberately
// loose about element names and nesting: docs/noita-save-layout.md only
// pins down the attributes we depend on (session_stat_file,
// world_seed), not the exact element hierarchy around them, and a save
// format we can't fully verify is safer to read this way than to encode
// an assumed shape that silently stops matching.
func findAttr(r io.Reader, attr string) (string, bool) {
	d := xml.NewDecoder(r)
	for {
		tok, err := d.Token()
		if err != nil {
			return "", false
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		for _, a := range start.Attr {
			if a.Name.Local == attr {
				return a.Value, true
			}
		}
	}
}
