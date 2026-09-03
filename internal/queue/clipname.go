package queue

import "regexp"

// clipFilenameRe matches Noita's own clip names:
//
//	noita-<YYYYMMDD>-<HHMMSS>-<world_seed>-<frame>.gif
//
// Verified against docs/noita-save-layout.md: the seed field matches
// stats/sessions/*_stats.xml's world_seed for four separate runs.
var clipFilenameRe = regexp.MustCompile(`^noita-(\d{8})-(\d{6})-(\d+)-(\d+)\.gif$`)

// ClipInfo is what a clip's own filename says about it, with no XML read.
type ClipInfo struct {
	// Timestamp is when the clip was written, as YYYYMMDD-HHMMSS. This is
	// the clip's own capture time, not necessarily its run's start time —
	// see ReadCurrentRun.
	Timestamp string

	// Seed is the run's world_seed, verbatim from the filename.
	Seed string
}

// ParseClipFilename reads a clip's run key out of its filename alone. The
// screenshots_animated folder also holds hand-named clips (players rename
// gifs they want to keep) and a stray .xml, so a filename that doesn't match
// is reported, not treated as an error — the caller skips it and moves on.
func ParseClipFilename(name string) (ClipInfo, bool) {
	m := clipFilenameRe.FindStringSubmatch(name)
	if m == nil {
		return ClipInfo{}, false
	}
	return ClipInfo{
		Timestamp: m[1] + "-" + m[2],
		Seed:      m[3],
	}, true
}
