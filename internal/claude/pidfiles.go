package claude

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// pidFileEntry represents a parsed PID-keyed JSON file from a state directory.
type pidFileEntry struct {
	PID     int
	Path    string
	ModTime time.Time
	Data    []byte
}

// readPIDFiles reads a directory of <PID>.json files, cleaning up entries older
// than maxAge and returning the rest. Callers handle unmarshalling and filtering.
func readPIDFiles(dir string, maxAge time.Duration) ([]pidFileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []pidFileEntry
	now := time.Now()

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}

		path := filepath.Join(dir, e.Name())

		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > maxAge {
			_ = os.Remove(path)
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		result = append(result, pidFileEntry{
			PID:     pid,
			Path:    path,
			ModTime: info.ModTime(),
			Data:    data,
		})
	}

	return result, nil
}
