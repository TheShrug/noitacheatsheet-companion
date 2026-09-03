package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// configDirName is this app's directory under the user's config directory —
// never inside Noita's own folders. See SECURITY.md's "Writes" row.
const configDirName = "noitacheatsheet-companion"

const queueFileName = "queue.json"

// ConfigPath is where the queue is persisted:
// <os.UserConfigDir()>/noitacheatsheet-companion/queue.json.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding the config directory: %w", err)
	}
	return filepath.Join(dir, configDirName, queueFileName), nil
}

// Load reads the queue from path. A missing or corrupt file is an empty
// queue, never an error: the file not existing is the normal state before
// the first clip is ever seen, and a half-written file left by a crash
// should not stop the app that wrote it from starting again.
func Load(path string) *Queue {
	data, err := os.ReadFile(path)
	if err != nil {
		return New()
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return New()
	}

	return &Queue{entries: entries}
}

// Save writes the queue to path atomically: it writes a temp file in the
// same directory, then renames it over the target, so a crash mid-write
// leaves either the old file or the new one, never a truncated one.
func (q *Queue) Save(path string) error {
	q.mu.Lock()
	entries := append([]Entry(nil), q.entries...)
	q.mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the queue: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".queue-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating a temp file next to %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmpPath, path, err)
	}
	return nil
}
