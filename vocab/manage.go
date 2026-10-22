package vocab

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// Why a word list cannot get a name. The texts are shown to the learner.
var (
	ErrNameEmpty     = errors.New("Enter a name.")
	ErrNameSeparator = errors.New(`The name must not contain "/" or "\".`)
	ErrNameDot       = errors.New(`The name must not start with ".".`)
	ErrNameChars     = errors.New(`The name must not contain < > : " | ? * or control characters.`)
	ErrNameReserved  = errors.New("The name jmdict is reserved for the bundled word list.")
	ErrNameDevice    = errors.New("This name is reserved by Windows.")
	ErrNameTaken     = errors.New("Another word list has this name.")
)

// windowsDevices are the names Windows keeps for devices, with or without
// an extension, in any case.
var windowsDevices = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
}

// isWindowsDevice reports whether name is a Windows device name: the part
// before the first "." is one, ignoring case.
func isWindowsDevice(name string) bool {
	stem, _, _ := strings.Cut(name, ".")
	return slices.ContainsFunc(windowsDevices, func(d string) bool { return strings.EqualFold(stem, d) })
}

// hasBadChar reports whether name has a character Windows does not allow in
// a file name (besides the separators) or a control character.
func hasBadChar(name string) bool {
	return strings.ContainsAny(name, `<>:"|?*`) || strings.ContainsFunc(name, unicode.IsControl)
}

// fileError is a failed file operation on a word list, told to the learner
// in a sentence with the system's reason but without the path.
type fileError struct {
	what string // "renamed", "deleted"
	err  error
}

func (e *fileError) Error() string {
	reason := e.err
	var pe *fs.PathError
	var le *os.LinkError
	switch {
	case errors.As(e.err, &pe):
		reason = pe.Err
	case errors.As(e.err, &le):
		reason = le.Err
	}
	return "The word list could not be " + e.what + ": " + reason.Error() + "."
}

func (e *fileError) Unwrap() error { return e.err }

// DeleteList removes a word list's file for good. A file that is already
// gone counts as deleted.
func DeleteList(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return &fileError{"deleted", err}
	}
	return nil
}

// CheckListName checks newName as the new name of the word list stored at
// path and returns it trimmed. It must not be empty, contain "/" or "\",
// start with "." (LoadDir would hide the file), contain a character Windows
// does not allow in file names or a control character, be jmdict (any case)
// or a Windows device name such as CON or com1.txt (on every system, so a
// list can move between them), and no other word list file in the folder
// may have it as its file name or its list name, ignoring case. The list's
// own name in another case is fine.
func CheckListName(path, newName string) (string, error) {
	name := strings.TrimSpace(newName)
	switch {
	case name == "":
		return "", ErrNameEmpty
	case strings.ContainsAny(name, `/\`):
		return "", ErrNameSeparator
	case strings.HasPrefix(name, "."):
		return "", ErrNameDot
	case hasBadChar(name):
		return "", ErrNameChars
	case strings.EqualFold(name, BundledSource):
		return "", ErrNameReserved
	case isWindowsDevice(name):
		return "", ErrNameDevice
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
// temporary dot file name, so case-insensitive file systems take it too;
// a leftover file of that name stops it. An existing file is never
// overwritten.
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
		if _, err := os.Lstat(tmp); err == nil {
			return "", &fileError{"renamed", fmt.Errorf("the leftover file %s is in the way", filepath.Base(tmp))}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", &fileError{"renamed", err}
		}
		if err := os.Rename(path, tmp); err != nil {
			return "", &fileError{"renamed", err}
		}
		if err := os.Rename(tmp, target); err != nil {
			_ = os.Rename(tmp, path) // put it back
			return "", &fileError{"renamed", err}
		}
		return name, nil
	}
	if _, err := os.Lstat(target); err == nil {
		return "", ErrNameTaken
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", &fileError{"renamed", err}
	}
	if err := os.Rename(path, target); err != nil {
		return "", &fileError{"renamed", err}
	}
	return name, nil
}
