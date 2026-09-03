package watch

import (
	"bytes"
	"io"
	"os"
)

// gifTrailer is the single byte a GIF stream ends with once it is fully
// written. Verified against all 94 clips in a real screenshots_animated
// folder — see issue #3.
const gifTrailer = 0x3B

// gifHeaders are the two valid opening six bytes of a GIF file. Noita's own
// clips all start GIF89a, but GIF87a is accepted too since it is the other
// value the format actually defines.
var gifHeaders = [][]byte{[]byte("GIF89a"), []byte("GIF87a")}

// isCompleteGIF reports whether path both starts with a GIF header and ends
// with the GIF trailer byte. This is the structural test — size stability
// alone is checked by the caller first, as a cheap gate before this opens
// the file at all.
//
// Any error — the file vanishing, being shorter than a header, a permission
// hiccup — is reported as incomplete rather than propagated: Poll treats
// that identically to "not finished yet," which is the safe default for a
// file this app never wrote and does not own.
func isCompleteGIF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 6)
	if _, err := io.ReadFull(f, header); err != nil {
		return false
	}

	valid := false
	for _, want := range gifHeaders {
		if bytes.Equal(header, want) {
			valid = true
			break
		}
	}
	if !valid {
		return false
	}

	if _, err := f.Seek(-1, io.SeekEnd); err != nil {
		// Only fails if the file is shorter than one byte, which the header
		// read above already ruled out — kept as a guard, not expected.
		return false
	}
	last := make([]byte, 1)
	if _, err := io.ReadFull(f, last); err != nil {
		return false
	}
	return last[0] == gifTrailer
}
