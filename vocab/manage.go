package vocab

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Why a word list cannot get a name. The texts are shown to the learner.
var (
	ErrNameEmpty     = errors.New("Enter a name.")
	ErrNameSeparator = errors.New(`The name must not contain "/" or "\".`)
	ErrNameDot       = errors.New(`The name must not start with ".".`)
	ErrNameReserved  = errors.New("The name jmdict is reserved for the bundled word list.")
	ErrNameTaken     = errors.New("Another word list has this name.")
)

// DeleteList removes a word list's file for good. A file that is already
// gone counts as deleted.
func DeleteList(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// CheckListName checks newName as the new name of the word list stored at
// path and returns it trimmed. It must not be empty, contain "/" or "\",
// start with "." (LoadDir would hide the file) or be jmdict (any case), and
// no other word list file in the folder may have it as its file name or its
// list name, ignoring case. The list's own name in another case is fine.
func CheckListName(path, newName string) (string, error) {
	name := strings.TrimSpace(newName)
	switch {
	case name == "":
		return "", ErrNameEmpty
	case strings.ContainsAny(name, `/\`):
		return "", ErrNameSeparator
	case strings.HasPrefix(name, "."):
		return "", ErrNameDot
	case strings.EqualFold(name, BundledSource):
		return "", ErrNameReserved
	}
	files, err := scanDir(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if f.path == path {
			continue
		}
		if strings.EqualFold(f.base, name) || strings.EqualFold(f.name, name) {
			return "", ErrNameTaken
		}
	}
	return name, nil
}

// RenameList renames the word list file at path to newName (see
// CheckListName) with its extension kept, and returns the list's new name.
// The same name changes nothing. A change of case only goes through a
// temporary dot file name, so case-insensitive file systems take it too. An
// existing file is never overwritten.
func RenameList(path, newName string) (string, error) {
	name, err := CheckListName(path, newName)
	if err != nil {
		return "", err
	}
	dir, file := filepath.Dir(path), filepath.Base(path)
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	if name == base {
		return name, nil
	}
	target := filepath.Join(dir, name+ext)
	if strings.EqualFold(name, base) {
		tmp := filepath.Join(dir, "."+file+".rename") // a dot file: LoadDir skips it
		if err := os.Rename(path, tmp); err != nil {
			return "", fmt.Errorf("rename: %w", err)
		}
		if err := os.Rename(tmp, target); err != nil {
			_ = os.Rename(tmp, path) // put it back
			return "", fmt.Errorf("rename: %w", err)
		}
		return name, nil
	}
	if _, err := os.Lstat(target); err == nil {
		return "", ErrNameTaken
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("rename: %w", err)
	}
	if err := os.Rename(path, target); err != nil {
		return "", fmt.Errorf("rename: %w", err)
	}
	return name, nil
}
