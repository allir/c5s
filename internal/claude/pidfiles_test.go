package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadPIDFiles(t *testing.T) {
	t.Run("NonexistentDir", func(t *testing.T) {
		entries, err := readPIDFiles("/tmp/does-not-exist-ever-"+t.Name(), time.Hour)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if entries != nil {
			t.Fatalf("expected nil slice, got %v", entries)
		}
	})

	t.Run("EmptyDir", func(t *testing.T) {
		dir := t.TempDir()
		entries, err := readPIDFiles(dir, time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(entries))
		}
	})

	t.Run("ValidFiles", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "123.json"), []byte(`{"foo":"bar"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "456.json"), []byte(`{"baz":"qux"}`), 0o644); err != nil {
			t.Fatal(err)
		}

		entries, err := readPIDFiles(dir, time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}

		found := map[int]string{}
		for _, e := range entries {
			found[e.PID] = string(e.Data)
		}
		if found[123] != `{"foo":"bar"}` {
			t.Errorf("PID 123: got data %q", found[123])
		}
		if found[456] != `{"baz":"qux"}` {
			t.Errorf("PID 456: got data %q", found[456])
		}
	})

	t.Run("NonJSONIgnored", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "789.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}

		entries, err := readPIDFiles(dir, time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].PID != 789 {
			t.Errorf("expected PID 789, got %d", entries[0].PID)
		}
	})

	t.Run("DirectoriesIgnored", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "100.json"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "200.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}

		entries, err := readPIDFiles(dir, time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].PID != 200 {
			t.Errorf("expected PID 200, got %d", entries[0].PID)
		}
	})

	t.Run("NonNumericIgnored", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "abc.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "300.json"), []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}

		entries, err := readPIDFiles(dir, time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].PID != 300 {
			t.Errorf("expected PID 300, got %d", entries[0].PID)
		}
	})

	t.Run("StaleCleanup", func(t *testing.T) {
		dir := t.TempDir()
		stalePath := filepath.Join(dir, "400.json")
		if err := os.WriteFile(stalePath, []byte(`{}`), 0o644); err != nil {
			t.Fatal(err)
		}
		twoHoursAgo := time.Now().Add(-2 * time.Hour)
		if err := os.Chtimes(stalePath, twoHoursAgo, twoHoursAgo); err != nil {
			t.Fatal(err)
		}

		entries, err := readPIDFiles(dir, time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(entries))
		}
		if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
			t.Errorf("stale file should have been removed, but stat returned: %v", err)
		}
	})
}
