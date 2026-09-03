package watch

import (
	"os"
	"path/filepath"
	"testing"
)

func writeGif(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// validGIF returns the smallest byte sequence that passes isCompleteGIF: a
// header, an arbitrary payload, and the trailer byte.
func validGIF() []byte {
	return append([]byte("GIF89a\x01\x02\x03"), gifTrailer)
}

func TestIsCompleteGIFAcceptsAValidClip(t *testing.T) {
	dir := t.TempDir()
	path := writeGif(t, dir, "clip.gif", validGIF())

	if !isCompleteGIF(path) {
		t.Error("isCompleteGIF = false, want true for a valid GIF89a clip ending in the trailer")
	}
}

func TestIsCompleteGIFAcceptsGIF87aHeader(t *testing.T) {
	dir := t.TempDir()
	content := append([]byte("GIF87a\x01"), gifTrailer)
	path := writeGif(t, dir, "clip.gif", content)

	if !isCompleteGIF(path) {
		t.Error("isCompleteGIF = false, want true for a GIF87a header")
	}
}

func TestIsCompleteGIFRejectsMissingTrailer(t *testing.T) {
	dir := t.TempDir()
	// A file still being written: valid header, no trailer yet.
	content := append([]byte("GIF89a\x01\x02\x03"), 0x00)
	path := writeGif(t, dir, "clip.gif", content)

	if isCompleteGIF(path) {
		t.Error("isCompleteGIF = true, want false when the last byte is not the GIF trailer")
	}
}

func TestIsCompleteGIFRejectsWrongHeader(t *testing.T) {
	dir := t.TempDir()
	content := append([]byte("<Entity>"), gifTrailer)
	path := writeGif(t, dir, "notagif.gif", content)

	if isCompleteGIF(path) {
		t.Error("isCompleteGIF = true, want false for a file that does not start with a GIF header")
	}
}

func TestIsCompleteGIFRejectsTooShortFile(t *testing.T) {
	dir := t.TempDir()
	path := writeGif(t, dir, "clip.gif", []byte("GIF8"))

	if isCompleteGIF(path) {
		t.Error("isCompleteGIF = true, want false for a file shorter than the header")
	}
}

func TestIsCompleteGIFRejectsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.gif")

	if isCompleteGIF(path) {
		t.Error("isCompleteGIF = true, want false for a file that does not exist")
	}
}
