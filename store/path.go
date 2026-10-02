package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// legacySuffixes are the database file and its SQLite WAL-mode siblings, in
// the order they are put in place: siblings first, the main file last.
var legacySuffixes = []string{"-wal", "-shm", ""}

// DefaultPath returns the per-user location for the database:
// $XDG_DATA_HOME/kana/kana.db when XDG_DATA_HOME is set to an absolute path,
// otherwise <home>/.local/share/kana/kana.db. It is the same on every OS; it
// does not use os.UserConfigDir or platform-specific application-data
// directories. It returns an error only if no home directory can be found.
func DefaultPath() (string, error) {
	if dataHome := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dataHome) {
		return filepath.Join(dataHome, "kana", "kana.db"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("store: determine home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "kana", "kana.db"), nil
}

// MigrateLegacy moves a pre-existing database at legacy (plus its -wal and
// -shm siblings, if present) to target, once. It creates target's directory
// as needed.
//
// If target already exists, legacy is left untouched and moved is false. If
// legacy does not exist, it is a no-op. Otherwise every file is first copied
// next to target under a temporary name, then renamed into place with the
// main file last, and only then are the legacy files removed. On any error
// before the main file is in place, legacy is left as it was and the caller
// should keep using it; the move is retried on the next start.
func MigrateLegacy(legacy, target string) (moved bool, err error) {
	if _, err := os.Stat(target); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("store: stat target: %w", err)
	}

	if _, err := os.Stat(legacy); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("store: stat legacy: %w", err)
	}

	if err := ensureDir(target); err != nil {
		return false, fmt.Errorf("store: ensure target directory: %w", err)
	}

	var present []string
	for _, suffix := range legacySuffixes {
		if _, err := os.Stat(legacy + suffix); err == nil {
			present = append(present, suffix)
		}
	}

	// Copy everything first, so a failure leaves legacy complete.
	for _, suffix := range present {
		if err := copyFile(legacy+suffix, target+suffix+".tmp"); err != nil {
			removeTemps(target, present)
			return false, fmt.Errorf("store: copy %s: %w", legacy+suffix, err)
		}
	}

	// Put the copies in place, main file last: a target without its main file
	// is ignored and replaced on the next attempt.
	for _, suffix := range present {
		if err := os.Rename(target+suffix+".tmp", target+suffix); err != nil {
			removeTemps(target, present)
			return false, fmt.Errorf("store: place %s: %w", target+suffix, err)
		}
	}

	// The data is safe at target; leftovers at legacy are harmless because
	// target now exists.
	for _, suffix := range present {
		_ = os.Remove(legacy + suffix)
	}
	return true, nil
}

func removeTemps(target string, suffixes []string) {
	for _, suffix := range suffixes {
		_ = os.Remove(target + suffix + ".tmp")
	}
}

// copyFile copies src to dst by content, fsyncing the destination so the
// data is durable before the caller trusts the migration. It never removes
// src.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, databaseFilePerm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
