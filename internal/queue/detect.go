package queue

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TheShrug/noitacheatsheet-companion/internal/noita"
)

// Detect builds the queue entry for one clip, the moment it is first seen
// complete. Call it once per clip: it snapshots player.xml right then, which
// is what makes a queue entry self-contained — see Entry.Wands.
//
// Its second return is false when clipPath doesn't look like a Noita clip
// filename (the screenshots_animated folder also holds hand-named gifs and a
// stray .xml); that is expected and not an error. A non-nil error means
// something that should have been readable wasn't — player.xml missing or
// unparsable — which is worth surfacing rather than silently skipping.
func Detect(install noita.Install, clipPath string, readWands WandReader, now time.Time) (Entry, bool, error) {
	info, ok := ParseClipFilename(filepath.Base(clipPath))
	if !ok {
		return Entry{}, false, nil
	}

	key := RunKey{Seed: info.Seed, SessionStart: info.Timestamp}

	// If the current run's seed matches this clip's, we know the run's real
	// start time and use it. Otherwise the clip is from a run that has
	// already ended (or never was current — e.g. a backlog scan at
	// startup), and there is no XML-free way to recover that run's actual
	// start; the clip's own timestamp is the best available stand-in, and
	// docs/noita-save-layout.md is explicit that grouping is meant to work
	// from a directory listing alone.
	if current, ok := ReadCurrentRun(install.Save); ok && current.Seed == info.Seed {
		key.SessionStart = current.SessionStart
	}

	playerXML := install.PlayerXML()
	f, err := os.Open(playerXML)
	if err != nil {
		return Entry{}, true, fmt.Errorf("reading %s for %s: %w", playerXML, filepath.Base(clipPath), err)
	}
	defer f.Close()

	wands, err := readWands(f)
	if err != nil {
		return Entry{}, true, fmt.Errorf("parsing wands from %s: %w", playerXML, err)
	}

	return Entry{
		Path:       clipPath,
		Run:        key,
		DetectedAt: now,
		Wands:      wands,
	}, true, nil
}
