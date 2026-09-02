// Package noita locates the two things an upload needs on a player's machine:
// the active save's player.xml, and the folder Noita's own in-game recorder
// writes animated gifs to.
//
// PARITY: this file is a port of NoitaCheatSheet/Shared/NoitaSavePaths.cs in
// TheShrug/NoitaSpellCasters, which is the site's copy of the same knowledge.
// The two must move together — if Nolla moves a folder, both change in the same
// pass. See .claude/skills/parity/SKILL.md and ADR 1.
//
// The C# copy lists one canonical path per platform, because it only has to
// print something for a human to paste into a file picker. This copy has to
// find the folder on a machine it has never seen, so it probes a list of
// candidates instead. That difference is deliberate, not drift.
package noita

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// SteamAppID is Noita's Steam AppID, which is also the directory name Proton
// keeps its prefix under.
const SteamAppID = "881100"

// ErrNotFound means no candidate root existed on this machine. It is not
// necessarily a bug in the candidate list — the player may simply not have
// Noita installed, or may keep it somewhere unusual.
var ErrNotFound = errors.New("no Noita data folder found")

// Platform is the OS family a Noita install sits on. Noita ships on Windows
// only; Linux and macOS run it inside a Wine or Proton prefix, which puts a
// whole Windows filesystem under a prefix root.
type Platform int

const (
	Windows Platform = iota
	Linux
	MacOS
)

func (p Platform) String() string {
	switch p {
	case Windows:
		return "Windows"
	case Linux:
		return "Linux (Proton)"
	case MacOS:
		return "macOS (Wine bottle)"
	default:
		return "unknown"
	}
}

// Inside a Wine or Proton prefix the LocalLow tree hangs off drive_c, so the
// Linux and macOS shapes are this with a different prefix root and user.
const inPrefix = "drive_c/users/%s/AppData/LocalLow/Nolla_Games_Noita"

// Candidate is one place Noita's data root might be, plus what a player needs
// to know when it isn't there. Note is never empty: a bare path that does not
// exist tells someone nothing about why.
type Candidate struct {
	Platform Platform
	Path     string
	Note     string
}

// Install is a Noita data root that actually exists on this machine, with the
// two folders an upload needs resolved underneath it.
type Install struct {
	Platform Platform

	// Root is the Nolla_Games_Noita folder.
	Root string

	// Save is the save folder whose player.xml is newest — see ActiveSave.
	Save string

	// Gifs is save_rec/screenshots_animated, which sits beside the save
	// folders rather than inside one. Clips are NOT per-save: the recorder
	// writes every run's gifs into this one directory.
	Gifs string
}

// PlayerXML is the file the wand list is parsed out of.
func (i Install) PlayerXML() string { return filepath.Join(i.Save, "player.xml") }

// Candidates lists every place this OS might keep Noita's data root, in the
// order they should be tried. It does not touch the filesystem.
func Candidates() []Candidate {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	switch runtime.GOOS {
	case "windows":
		return []Candidate{{
			Platform: Windows,
			Path:     filepath.Join(home, "AppData", "LocalLow", "Nolla_Games_Noita"),
			Note:     "Noita's own save location. Present whether the game came from Steam or GOG.",
		}}

	case "linux":
		// There is no native Linux build; Noita runs under Proton. The Steam
		// root differs by distribution, and again for the Flatpak, so all
		// four are tried.
		steamRoots := []string{
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".steam", "root"),
			filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"),
		}

		// Proton creates the prefix owned by "steamuser". Prefixes made by
		// older Proton versions, or by plain Wine, use the login name.
		users := []string{"steamuser"}
		if u, err := user.Current(); err == nil && u.Username != "" && u.Username != "steamuser" {
			users = append(users, u.Username)
		}

		var out []Candidate
		for _, root := range steamRoots {
			for _, name := range users {
				out = append(out, Candidate{
					Platform: Linux,
					Path: filepath.Join(root, "steamapps", "compatdata", SteamAppID, "pfx",
						filepath.FromSlash(fmt.Sprintf(inPrefix, name))),
					Note: "Proton prefix for AppID " + SteamAppID + ", under " + root + ".",
				})
			}
		}
		return out

	case "darwin":
		// Deliberately empty. Noita has no macOS build at all, so there is no
		// path to guess: it lives inside whichever CrossOver or Whisky bottle
		// the player made, under a name only they know. Guessing here would
		// produce a confident wrong answer, so --root is the answer instead.
		return nil

	default:
		return nil
	}
}

// Probe is one candidate and whether it is on this machine. This is what lets
// the tool say "I looked in these four places" rather than only "not found".
type Probe struct {
	Candidate Candidate
	Exists    bool
}

// ProbeAll checks every candidate against the filesystem, in order.
func ProbeAll() []Probe {
	candidates := Candidates()
	out := make([]Probe, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, Probe{Candidate: c, Exists: isDir(c.Path)})
	}
	return out
}

// Detect finds the Noita data root on this machine and resolves the two
// folders underneath it.
func Detect() (Install, error) {
	for _, p := range ProbeAll() {
		if !p.Exists {
			continue
		}

		install, err := At(p.Candidate.Path)
		if err != nil {
			// The root is there but unusable — no save folder yet, most
			// likely a prefix that has never been played in. Keep looking;
			// another one may be the real install.
			continue
		}

		install.Platform = p.Candidate.Platform
		return install, nil
	}
	return Install{}, ErrNotFound
}

// At resolves an explicitly given Nolla_Games_Noita folder. This is the path
// --root takes, and the only way to reach a macOS bottle.
func At(root string) (Install, error) {
	if !isDir(root) {
		return Install{}, fmt.Errorf("%w: %s is not a directory", ErrNotFound, root)
	}

	save, err := ActiveSave(root)
	if err != nil {
		return Install{}, err
	}

	return Install{
		Platform: platformForGOOS(),
		Root:     root,
		Save:     save,
		Gifs:     filepath.Join(root, "save_rec", "screenshots_animated"),
	}, nil
}

var saveDir = regexp.MustCompile(`^save\d+$`)

// ActiveSave picks the save folder to read wands from.
//
// Noita numbers them save00, save01, … and which one is "active" is not
// written down anywhere we can read. The newest player.xml is the best
// available proxy: the game rewrites it as the run progresses, so the save
// being played is the one touched last. A player with two saves who has not
// played the other one today gets the right answer; one alt-tabbing between
// two runs in the same minute may not.
func ActiveSave(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", root, err)
	}

	var best string
	var bestMod int64
	for _, e := range entries {
		if !e.IsDir() || !saveDir.MatchString(e.Name()) {
			continue
		}

		info, err := os.Stat(filepath.Join(root, e.Name(), "player.xml"))
		if err != nil {
			// A save folder with no player.xml has never been played into.
			// Not an error, just not a candidate.
			continue
		}

		if mod := info.ModTime().UnixNano(); best == "" || mod > bestMod {
			best, bestMod = filepath.Join(root, e.Name()), mod
		}
	}

	if best == "" {
		return "", fmt.Errorf("%w: no saveNN folder under %s contains a player.xml", ErrNotFound, root)
	}
	return best, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func platformForGOOS() Platform {
	switch runtime.GOOS {
	case "windows":
		return Windows
	case "darwin":
		return MacOS
	default:
		return Linux
	}
}

// ExplainNotFound is the message a player sees when nothing was found. It
// lists what was actually tried, because "Noita not found" with no list is
// unactionable for the one person who can fix it.
func ExplainNotFound(probes []Probe) string {
	var b strings.Builder
	b.WriteString("Could not find Noita's data folder.\n")

	if len(probes) == 0 {
		b.WriteString("\nThere is no default path to try on this platform — Noita has no native\n")
		b.WriteString("build here, so it lives inside whichever Wine bottle you created.\n")
	} else {
		b.WriteString("\nLooked in:\n")
		for _, p := range probes {
			b.WriteString("  - " + p.Candidate.Path + "\n")
		}
	}

	b.WriteString("\nPoint at it directly with --root, giving the folder that contains save00\n")
	b.WriteString("and save_rec. On Windows that is normally:\n")
	b.WriteString("  %UserProfile%\\AppData\\LocalLow\\Nolla_Games_Noita\n")
	return b.String()
}
