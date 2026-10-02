package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPath(t *testing.T) {
	t.Run("absolute XDG_DATA_HOME", func(t *testing.T) {
		dataHome := t.TempDir()
		t.Setenv("XDG_DATA_HOME", dataHome)

		got, err := DefaultPath()
		if err != nil {
			t.Fatalf("DefaultPath: %v", err)
		}
		want := filepath.Join(dataHome, "kana", "kana.db")
		if got != want {
			t.Fatalf("DefaultPath = %q, want %q", got, want)
		}
	})

	t.Run("relative XDG_DATA_HOME is ignored", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("XDG_DATA_HOME", "relative/data")

		got, err := DefaultPath()
		if err != nil {
			t.Fatalf("DefaultPath: %v", err)
		}
		want := filepath.Join(home, ".local", "share", "kana", "kana.db")
		if got != want {
			t.Fatalf("DefaultPath = %q, want %q", got, want)
		}
	})

	t.Run("falls back to home directory", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("XDG_DATA_HOME", "")

		got, err := DefaultPath()
		if err != nil {
			t.Fatalf("DefaultPath: %v", err)
		}
		want := filepath.Join(home, ".local", "share", "kana", "kana.db")
		if got != want {
			t.Fatalf("DefaultPath = %q, want %q", got, want)
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func requireExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func requireGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be gone, stat err = %v", path, err)
	}
}

func TestMigrateLegacy(t *testing.T) {
	t.Run("legacy present, target missing: moves main file and siblings", func(t *testing.T) {
		srcDir := t.TempDir()
		legacy := filepath.Join(srcDir, "kana.db")
		writeFile(t, legacy, "main-db-contents")
		writeFile(t, legacy+"-wal", "wal-contents")
		writeFile(t, legacy+"-shm", "shm-contents")

		// Target lives under a directory that does not exist yet, to also
		// cover "target directory is created when missing".
		targetDir := filepath.Join(t.TempDir(), "nested", "kana")
		target := filepath.Join(targetDir, "kana.db")

		moved, err := MigrateLegacy(legacy, target)
		if err != nil {
			t.Fatalf("MigrateLegacy: %v", err)
		}
		if !moved {
			t.Fatalf("moved = false, want true")
		}

		requireExists(t, targetDir)

		gotMain, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read target: %v", err)
		}
		if string(gotMain) != "main-db-contents" {
			t.Fatalf("target contents = %q, want %q", gotMain, "main-db-contents")
		}

		gotWAL, err := os.ReadFile(target + "-wal")
		if err != nil {
			t.Fatalf("read target wal: %v", err)
		}
		if string(gotWAL) != "wal-contents" {
			t.Fatalf("target -wal contents = %q, want %q", gotWAL, "wal-contents")
		}

		gotSHM, err := os.ReadFile(target + "-shm")
		if err != nil {
			t.Fatalf("read target shm: %v", err)
		}
		if string(gotSHM) != "shm-contents" {
			t.Fatalf("target -shm contents = %q, want %q", gotSHM, "shm-contents")
		}

		requireGone(t, legacy)
		requireGone(t, legacy+"-wal")
		requireGone(t, legacy+"-shm")
	})

	t.Run("failing sibling leaves the main file in place", func(t *testing.T) {
		srcDir := t.TempDir()
		legacy := filepath.Join(srcDir, "kana.db")
		writeFile(t, legacy, "main-db-contents")
		writeFile(t, legacy+"-shm", "shm-contents")

		targetDir := t.TempDir()
		target := filepath.Join(targetDir, "kana.db")
		// A directory where the -shm sibling should go makes both rename and
		// copy of that sibling fail.
		if err := os.Mkdir(target+"-shm", 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		moved, err := MigrateLegacy(legacy, target)
		if err == nil || moved {
			t.Fatalf("MigrateLegacy = %v, %v; want an error and moved=false", moved, err)
		}
		requireExists(t, legacy)
		requireGone(t, target)
	})

	t.Run("target present: nothing changes", func(t *testing.T) {
		dir := t.TempDir()
		legacy := filepath.Join(dir, "kana.db")
		writeFile(t, legacy, "legacy-contents")

		targetDir := t.TempDir()
		target := filepath.Join(targetDir, "kana.db")
		writeFile(t, target, "target-contents")

		moved, err := MigrateLegacy(legacy, target)
		if err != nil {
			t.Fatalf("MigrateLegacy: %v", err)
		}
		if moved {
			t.Fatalf("moved = true, want false")
		}

		gotLegacy, err := os.ReadFile(legacy)
		if err != nil {
			t.Fatalf("read legacy: %v", err)
		}
		if string(gotLegacy) != "legacy-contents" {
			t.Fatalf("legacy contents changed: %q", gotLegacy)
		}

		gotTarget, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read target: %v", err)
		}
		if string(gotTarget) != "target-contents" {
			t.Fatalf("target contents changed: %q", gotTarget)
		}
	})

	t.Run("legacy missing: no-op", func(t *testing.T) {
		dir := t.TempDir()
		legacy := filepath.Join(dir, "does-not-exist.db")

		targetDir := t.TempDir()
		target := filepath.Join(targetDir, "kana.db")

		moved, err := MigrateLegacy(legacy, target)
		if err != nil {
			t.Fatalf("MigrateLegacy: %v", err)
		}
		if moved {
			t.Fatalf("moved = true, want false")
		}
		requireGone(t, target)
	})
}
