package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// legacySuffixes are the SQLite WAL-mode sibling files that travel with the
// main database file.
var legacySuffixes = []string{"", "-wal", "-shm"}

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
// legacy does not exist, it is a no-op. Otherwise it tries os.Rename for the
// main file and each sibling; if renaming fails (for example across
// filesystems), it falls back to copying the bytes and leaves the originals
// in place rather than risk losing user data.
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

	var copyErr error
	for _, suffix := range legacySuffixes {
		src, dst := legacy+suffix, target+suffix
		if _, err := os.Stat(src); err != nil {
			continue // siblings are optional
		}
		if err := os.Rename(src, dst); err == nil {
			continue
		}
		if err := copyFile(src, dst); err != nil {
			copyErr = errors.Join(copyErr, fmt.Errorf("store: copy %s: %w", src, err))
		}
	}
	if copyErr != nil {
		return false, copyErr
	}

	return true, nil
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
