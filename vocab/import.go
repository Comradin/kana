package vocab

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// MaxImportSize is the largest source file that can be imported.
const MaxImportSize = 5 << 20

// importExts maps the extensions that can be imported to the extension the
// word list is stored with: Excel saves "Unicode Text" as a tab-separated
// .txt.
var importExts = map[string]string{".csv": ".csv", ".tsv": ".tsv", ".txt": ".tsv", ".json": ".json"}

// spreadsheetExts are rejected with a hint to save the sheet as CSV.
var spreadsheetExts = map[string]bool{".xlsx": true, ".xls": true, ".ods": true}

const (
	hintUTF8        = `Save the file as "CSV UTF-8" in your spreadsheet program and import it again.`
	hintSpreadsheet = `Save the sheet as "CSV UTF-8" in your spreadsheet program and import the .csv file.`
	hintRename      = "Rename the file and import it again."
	hintHeader      = "The first line names the columns, for example: kana;meaning;lesson"
)

// ImportError is why a source file cannot be imported, with a hint on what
// to do about it (may be empty). For a file without words, List is what was
// read, with its skipped rows' messages.
type ImportError struct {
	Reason, Hint string
	List         List
}

func (e *ImportError) Error() string { return e.Reason }

// Candidate is a checked source file: the word list it becomes and the
// UTF-8 bytes to store.
type Candidate struct {
	Name string // the word list's name: the file name without extension
	Ext  string // the extension it is stored with
	List List
	Data []byte
}

// Inspect checks a source file for import: its name and extension, size,
// encoding (UTF-16 with BOM is converted to UTF-8) and content. It touches
// no files. The list keeps the first maxProblems messages.
func Inspect(fileName string, data []byte) (Candidate, error) {
	return inspect(fileName, data, maxProblems)
}

// InspectAll is Inspect keeping every skipped row's message, for the check
// command.
func InspectAll(fileName string, data []byte) (Candidate, error) {
	return inspect(fileName, data, allProblems)
}

// inspect is Inspect with the list's message limit (see List).
func inspect(fileName string, data []byte, limit int) (Candidate, error) {
	fileName = filepath.Base(fileName) // a path must not escape the folder
	ext := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if spreadsheetExts[ext] {
		return Candidate{}, &ImportError{Reason: "This is a spreadsheet file.", Hint: hintSpreadsheet}
	}
	stored, ok := importExts[ext]
	if !ok {
		return Candidate{}, &ImportError{Reason: "Only .csv, .tsv, .txt and .json files can be imported."}
	}
	if base == "" || strings.HasPrefix(base, ".") {
		return Candidate{}, &ImportError{Reason: `The file name must not start with ".".`, Hint: hintRename}
	}
	if strings.EqualFold(base, BundledSource) {
		return Candidate{}, &ImportError{Reason: "The name jmdict is reserved for the bundled word list.", Hint: hintRename}
	}
	if len(data) > MaxImportSize {
		return Candidate{}, &ImportError{Reason: "The file is larger than 5 MB."}
	}
	text, err := toUTF8(data)
	if err != nil {
		return Candidate{}, &ImportError{Reason: "The file is not saved as UTF-8.", Hint: hintUTF8}
	}
	var l List
	if stored == ".json" {
		l = readJSON(base, bytes.NewReader(text), limit)
	} else {
		l = readCSV(base, bytes.NewReader(text), limit)
	}
	if len(l.Words) == 0 {
		return Candidate{}, noWords(l, text)
	}
	return Candidate{Name: base, Ext: stored, List: l, Data: text}, nil
}

// noWords explains a list without words. Unless the file is empty or has no
// kana column, the error keeps the list for its skipped rows' messages.
func noWords(l List, text []byte) error {
	if len(bytes.TrimSpace(text)) == 0 {
		return &ImportError{Reason: "The file is empty."}
	}
	if len(l.Problems) > 0 && l.Problems[0] == problemNoKanaColumn {
		return &ImportError{Reason: `The file has no "kana" column.`, Hint: hintHeader}
	}
	if !hasKana(text) {
		return &ImportError{Reason: "The file contains no kana.", Hint: hintUTF8, List: l}
	}
	reason := "The file has no words."
	if len(l.Problems) > 0 {
		reason += " " + l.Problems[0]
	}
	return &ImportError{Reason: reason, List: l}
}

// toUTF8 returns data as UTF-8: UTF-16 with a byte order mark is converted
// (without the mark), anything else must already be valid UTF-8.
func toUTF8(data []byte) ([]byte, error) {
	var be bool
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}):
	case bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		be = true
	default:
		// A NUL byte is UTF-16 without a byte order mark.
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return nil, errors.New("not UTF-8")
		}
		return data, nil
	}
	data = data[2:]
	if len(data)%2 != 0 {
		return nil, errors.New("odd UTF-16 length")
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		if be {
			units[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
		} else {
			units[i] = uint16(data[2*i+1])<<8 | uint16(data[2*i])
		}
	}
	return []byte(string(utf16.Decode(units))), nil
}

// hasKana reports whether text contains a hiragana or katakana character.
func hasKana(text []byte) bool {
	return bytes.ContainsFunc(text, func(r rune) bool { return r >= 0x3040 && r <= 0x30FF })
}

// listFile reports whether name is a word list file of LoadDir and returns
// its base name.
func listFile(name string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if strings.HasPrefix(name, ".") || (ext != ".csv" && ext != ".tsv" && ext != ".json") {
		return "", false
	}
	return strings.TrimSuffix(name, filepath.Ext(name)), true
}

// Existing returns the names (with extension, sorted) of the word list files
// in dir named like name, ignoring case and extension.
func Existing(dir, name string) []string {
	entries, _ := os.ReadDir(dir)
	var files []string
	for _, e := range entries {
		if base, ok := listFile(e.Name()); ok && !e.IsDir() && strings.EqualFold(base, name) {
			files = append(files, e.Name())
		}
	}
	slices.Sort(files)
	return files
}

// Import stores c in dir, creating dir if needed, and returns the word
// list's name. A list of the same name (ignoring case) keeps its name, so
// the learner's selection stays valid; otherwise it is c.Name. The file is
// written under a temporary name and renamed into place; only then are other
// word list files of that name (another extension or case) removed. On an
// error the old list stays as it was.
func Import(dir string, c Candidate) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".import-*") // a dot file: LoadDir skips it
	if err != nil {
		return "", fmt.Errorf("import: %w", err)
	}
	defer os.Remove(tmp.Name()) // a no-op after the rename
	if _, err := tmp.Write(c.Data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("import: %w", err)
	}
	// CreateTemp makes the file 0600; a word list is an ordinary file.
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return "", fmt.Errorf("import: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", fmt.Errorf("import: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("import: %w", err)
	}
	name := c.Name
	if old := Existing(dir, c.Name); len(old) > 0 {
		name = strings.TrimSuffix(old[0], filepath.Ext(old[0]))
	}
	target := filepath.Join(dir, name+c.Ext)
	if err := os.Rename(tmp.Name(), target); err != nil {
		return "", fmt.Errorf("import: %w", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("import: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("import: %w", err)
	}
	for _, e := range entries {
		base, ok := listFile(e.Name())
		if !ok || e.IsDir() || !strings.EqualFold(base, c.Name) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		// On a case-insensitive file system the old name can be the new file.
		if other, err := os.Stat(path); err == nil && os.SameFile(other, info) {
			continue
		}
		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("import: remove %s: %w", e.Name(), err)
		}
	}
	return name, nil
}
