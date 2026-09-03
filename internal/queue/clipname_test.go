package queue

import "testing"

func TestParseClipFilenameMatchesNoitaGrammar(t *testing.T) {
	info, ok := ParseClipFilename("noita-20201123-153055-776668009-01028241.gif")
	if !ok {
		t.Fatal("ParseClipFilename returned ok = false for a well-formed name")
	}
	if info.Timestamp != "20201123-153055" {
		t.Errorf("Timestamp = %q, want %q", info.Timestamp, "20201123-153055")
	}
	if info.Seed != "776668009" {
		t.Errorf("Seed = %q, want %q", info.Seed, "776668009")
	}
}

// screenshots_animated also holds clips the player renamed by hand to keep
// them, and a stray .xml sitting beside a clip of the same basename. Both
// must be skipped, not treated as a parse error.
func TestParseClipFilenameSkipsFilesThatDoNotMatch(t *testing.T) {
	for _, name := range []string{
		"asdf.gif",
		"get-past-trigger.gif",
		"noita-20201123-153055-776668009-01028241.xml",
		"noita-2020112-153055-776668009-01028241.gif", // short date
		"screenshot.png",
	} {
		if _, ok := ParseClipFilename(name); ok {
			t.Errorf("ParseClipFilename(%q) = ok, want not matched", name)
		}
	}
}
