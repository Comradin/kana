# Word List Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Learners import a word list (CSV/TSV/TXT/JSON) from the settings' Words tab instead of copying it into the words folder by hand.

**Architecture:** `vocab` gets a pure `Inspect` (decode, check, parse, return a `Candidate` with the UTF-8 bytes to write) and `Import`/`Existing` (write into the words folder atomically, replacing files of the same name). `fyne` reloads the lists at runtime (`ReloadLists`, `ImportList`), creates the words folder at startup, and adds the button, folder line, preview dialog and source-select refresh to the Words tab.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (`dialog.NewFileOpen`, `storage.NewExtensionFileFilter`), stdlib `unicode/utf16`, `unicode/utf8`. No new dependencies.

Spec: `docs/superpowers/specs/2026-10-20-word-list-import-design.md`. Terms: `GLOSSARY.md` (word list, source file, import).

**Conventions:** comments like the surrounding code (short doc comment on every function, "Must be called under lock." where it applies). Tests: `go test ./vocab/ ./fyne/ -timeout 300s`. Fyne UI code runs on the Fyne thread; tests use `test.NewApp()` / `test.NewWindow`. Implementers leave changes **uncommitted**; the controller reviews and commits.

---

## File structure

- Create `vocab/import.go`: `MaxImportSize`, `ImportError`, `Candidate`, `Inspect`, `Existing`, `Import`, `toUTF8`, `hasKana`.
- Create `vocab/import_test.go`.
- Modify `vocab/csvlist.go`: name the "no kana column" problem as a constant.
- Modify `fyne/wordgame.go`: split `loadLists` out of `loadWords`, create the folder at startup, add `ReloadLists`, `ImportList`, `HasList`.
- Create `fyne/importlist.go`: preview text, `importChosen`, `showImportDialog`.
- Create `fyne/importlist_test.go`.
- Modify `fyne/settings.go`: `settingsState.wordsDir`, `sourceOptions`, `setLists`, Words-tab button and folder line, `onImport` wiring in `showSettingsDialog`.
- Modify `fyne/settings_test.go`.
- Modify `README.md` (private word lists section).

---

### Task 1: `vocab.Inspect`

**Files:**
- Create: `vocab/import.go`, `vocab/import_test.go`
- Modify: `vocab/csvlist.go` (the `no "kana" column in the header` problem, around line 76)

- [ ] **Step 1: Constant for the missing kana column**

In `vocab/csvlist.go` add next to `headerAliases`:

```go
// problemNoKanaColumn is the problem of a header without a kana column.
const problemNoKanaColumn = `no "kana" column in the header`
```

and use it in `readCSV`: `l.problem(problemNoKanaColumn)`.

- [ ] **Step 2: Write the failing tests** (`vocab/import_test.go`)

```go
package vocab

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"unicode/utf16"
)

func utf16Bytes(s string, bigEndian bool) []byte {
	var b bytes.Buffer
	if bigEndian {
		b.Write([]byte{0xFE, 0xFF})
	} else {
		b.Write([]byte{0xFF, 0xFE})
	}
	for _, u := range utf16.Encode([]rune(s)) {
		if bigEndian {
			b.Write([]byte{byte(u >> 8), byte(u)})
		} else {
			b.Write([]byte{byte(u), byte(u >> 8)})
		}
	}
	return b.Bytes()
}

func importErr(t *testing.T, err error) *ImportError {
	t.Helper()
	var ie *ImportError
	if !errors.As(err, &ie) {
		t.Fatalf("err = %v, want an *ImportError", err)
	}
	return ie
}

func TestInspectAcceptsUTF8(t *testing.T) {
	for _, data := range []string{"kana;meaning;lesson\nせんせい;teacher;1\nがくせい;student;1\nえき;station;2\n",
		"\ufeffkana;meaning;lesson\nせんせい;teacher;1\nがくせい;student;1\nえき;station;2\n"} {
		c, err := Inspect("Minna no Nihongo.csv", []byte(data))
		if err != nil {
			t.Fatal(err)
		}
		if c.Name != "Minna no Nihongo" || c.Ext != ".csv" || len(c.List.Words) != 3 || c.List.Name != "Minna no Nihongo" {
			t.Fatalf("candidate %+v", c)
		}
		if !bytes.Equal(c.Data, []byte(data)) {
			t.Fatal("UTF-8 data must be written as is")
		}
	}
}

func TestInspectConvertsUTF16(t *testing.T) {
	text := "kana\tmeaning\nせんせい\tteacher\n"
	for _, be := range []bool{false, true} {
		c, err := Inspect("liste.txt", utf16Bytes(text, be))
		if err != nil {
			t.Fatal(err)
		}
		if c.Ext != ".tsv" || string(c.Data) != text || len(c.List.Words) != 1 {
			t.Fatalf("big endian %v: ext %q, data %q, words %d", be, c.Ext, c.Data, len(c.List.Words))
		}
	}
}

func TestInspectJSON(t *testing.T) {
	c, err := Inspect("a.JSON", []byte(`[{"kana":"せんせい","meaning":"teacher"}]`))
	if err != nil || c.Ext != ".json" || len(c.List.Words) != 1 {
		t.Fatalf("c %+v, err %v", c, err)
	}
}

func TestInspectRejects(t *testing.T) {
	big := "kana\n" + strings.Repeat("あ\n", MaxImportSize/4)
	cases := []struct {
		name, file string
		data       []byte
		reason     string // substring
		hint       string // substring, "" = no hint expected
	}{
		{"spreadsheet", "list.xlsx", []byte("PK"), "spreadsheet", "CSV UTF-8"},
		{"other extension", "list.pdf", []byte("x"), ".csv", ""},
		{"dot name", ".hidden.csv", []byte("kana\nあ\n"), `"."`, "Rename"},
		{"reserved", "JMdict.csv", []byte("kana\nあ\n"), "jmdict", "Rename"},
		{"too large", "big.csv", []byte(big), "5 MB", ""},
		{"not UTF-8", "ansi.csv", []byte("kana;meaning\n\x82\xa0;x\n"), "UTF-8", "CSV UTF-8"},
		{"question marks", "lost.csv", []byte("kana;meaning\n???;teacher\n"), "no kana", "CSV UTF-8"},
		{"no kana column", "cols.csv", []byte("word;meaning\nせんせい;teacher\n"), "kana", "kana;meaning;lesson"},
		{"empty", "empty.csv", nil, "empty", ""},
		{"odd UTF-16", "odd.txt", []byte{0xFF, 0xFE, 0x41}, "UTF-8", "CSV UTF-8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Inspect(tc.file, tc.data)
			ie := importErr(t, err)
			if !strings.Contains(ie.Reason, tc.reason) {
				t.Fatalf("reason %q, want it to contain %q", ie.Reason, tc.reason)
			}
			if (tc.hint == "") != (ie.Hint == "") || !strings.Contains(ie.Hint, tc.hint) {
				t.Fatalf("hint %q, want %q", ie.Hint, tc.hint)
			}
		})
	}
}
```

- [ ] **Step 3: Run, expect a build failure** (`Inspect` undefined): `go test ./vocab/ -run Inspect -timeout 300s`

- [ ] **Step 4: Implement** (`vocab/import.go`, first part)

```go
package vocab

import (
	"bytes"
	"errors"
	"path/filepath"
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

var spreadsheetExts = map[string]bool{".xlsx": true, ".xls": true, ".ods": true}

const (
	hintUTF8        = `Save the file as "CSV UTF-8" in your spreadsheet program and import it again.`
	hintSpreadsheet = `Save the sheet as "CSV UTF-8" in your spreadsheet program and import the .csv file.`
	hintRename      = "Rename the file and import it again."
	hintHeader      = "The first line names the columns, for example: kana;meaning;lesson"
)

// ImportError is why a source file cannot be imported, with a hint on what
// to do about it (may be empty).
type ImportError struct {
	Reason, Hint string
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
// no files.
func Inspect(fileName string, data []byte) (Candidate, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if spreadsheetExts[ext] {
		return Candidate{}, &ImportError{"This is a spreadsheet file.", hintSpreadsheet}
	}
	stored, ok := importExts[ext]
	if !ok {
		return Candidate{}, &ImportError{"Only .csv, .tsv, .txt and .json files can be imported.", ""}
	}
	if base == "" || strings.HasPrefix(base, ".") {
		return Candidate{}, &ImportError{`The file name must not start with ".".`, hintRename}
	}
	if strings.EqualFold(base, BundledSource) {
		return Candidate{}, &ImportError{"The name jmdict is reserved for the bundled word list.", hintRename}
	}
	if len(data) > MaxImportSize {
		return Candidate{}, &ImportError{"The file is larger than 5 MB.", ""}
	}
	text, err := toUTF8(data)
	if err != nil {
		return Candidate{}, &ImportError{"The file is not saved as UTF-8.", hintUTF8}
	}
	var l List
	if stored == ".json" {
		l = readJSON(base, bytes.NewReader(text))
	} else {
		l = readCSV(base, bytes.NewReader(text))
	}
	if len(l.Words) == 0 {
		return Candidate{}, noWords(l, text)
	}
	return Candidate{Name: base, Ext: stored, List: l, Data: text}, nil
}

// noWords explains a list without words.
func noWords(l List, text []byte) error {
	if len(bytes.TrimSpace(text)) == 0 {
		return &ImportError{"The file is empty.", ""}
	}
	if len(l.Problems) > 0 && l.Problems[0] == problemNoKanaColumn {
		return &ImportError{`The file has no "kana" column.`, hintHeader}
	}
	if !hasKana(text) {
		return &ImportError{"The file contains no kana.", hintUTF8}
	}
	reason := "The file has no words."
	if len(l.Problems) > 0 {
		reason += " " + l.Problems[0]
	}
	return &ImportError{reason, ""}
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
		if !utf8.Valid(data) {
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
```

Note the empty file: `readCSV` reports "file is empty" and `noWords` checks emptiness first. Check that "question marks" reaches `!hasKana` (its kana column exists, `???` rows are skipped as "not kana").

- [ ] **Step 5: Run** `go test ./vocab/ -timeout 300s` → PASS (all vocab tests).

---

### Task 2: `vocab.Existing` and `vocab.Import`

**Files:** Modify `vocab/import.go`, `vocab/import_test.go`

- [ ] **Step 1: Failing tests**

```go
func writeFile(t *testing.T, dir, name, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestImportNewList(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "words") // does not exist yet
	c, err := Inspect("mine.txt", []byte("kana\tmeaning\nえき\tstation\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := Existing(dir, "mine"); ok {
		t.Fatal("Existing before the import")
	}
	if name, err := Import(dir, c); err != nil || name != "mine" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); len(got) != 1 || got[0] != "mine.tsv" {
		t.Fatalf("files = %v", got)
	}
	if name, ok := Existing(dir, "MINE"); !ok || name != "mine" {
		t.Fatalf("Existing ignores case and keeps the stored one: %q %v", name, ok)
	}
	lists, err := LoadDir(dir)
	if err != nil || len(lists) != 1 || lists[0].Name != "mine" || len(lists[0].Words) != 1 {
		t.Fatalf("lists %+v, err %v", lists, err)
	}
}

func TestImportReplacesSameName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Mine.json", `[{"kana":"あお"}]`)
	writeFile(t, dir, "other.csv", "kana\nいえ\n")
	writeFile(t, dir, "notes.md", "kept")
	c, err := Inspect("mine.csv", []byte("kana\nえき\n"))
	if err != nil {
		t.Fatal(err)
	}
	if name, ok := Existing(dir, "mine"); !ok || name != "Mine" {
		t.Fatal("Mine.json exists under the same name")
	}
	if name, err := Import(dir, c); err != nil || name != "Mine" {
		t.Fatalf("name %q, err %v; the list keeps its name", name, err)
	}
	got := dirNames(t, dir)
	if len(got) != 3 || !slices.Contains(got, "Mine.csv") || !slices.Contains(got, "other.csv") || !slices.Contains(got, "notes.md") {
		t.Fatalf("files = %v; want Mine.json replaced by Mine.csv", got)
	}
}

func TestImportSameFileOtherCase(t *testing.T) {
	// On a case-insensitive file system (macOS) Mine.csv and mine.csv are
	// one file; the replaced list must not be deleted afterwards.
	dir := t.TempDir()
	writeFile(t, dir, "Mine.csv", "kana\nあお\n")
	c, _ := Inspect("mine.csv", []byte("kana\nえき\n"))
	if _, err := Import(dir, c); err != nil {
		t.Fatal(err)
	}
	lists, _ := LoadDir(dir)
	if len(lists) != 1 || len(lists[0].Words) != 1 || lists[0].Words[0].Kana != "えき" {
		t.Fatalf("lists %+v", lists)
	}
}

func TestImportFailureKeepsTheOldList(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "mine.json", `[{"kana":"あお"}]`)
	// A directory where the new file should go makes the rename fail.
	if err := os.Mkdir(filepath.Join(dir, "mine.csv"), 0o755); err != nil {
		t.Fatal(err)
	}
	c, _ := Inspect("mine.csv", []byte("kana\nえき\n"))
	if _, err := Import(dir, c); err == nil {
		t.Fatal("want an error")
	}
	got := dirNames(t, dir)
	if !slices.Contains(got, "mine.json") || len(got) != 2 {
		t.Fatalf("files = %v; want the old list kept and no temporary file", got)
	}
}
```

(Add `os`, `path/filepath`, `slices` to the test imports.)

- [ ] **Step 2: Run, expect a build failure.**

- [ ] **Step 3: Implement** (append to `vocab/import.go`, add `fmt`, `os` imports)

```go
// listFile reports whether name is a word list file of LoadDir and returns
// its base name.
func listFile(name string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if strings.HasPrefix(name, ".") || (ext != ".csv" && ext != ".tsv" && ext != ".json") {
		return "", false
	}
	return strings.TrimSuffix(name, filepath.Ext(name)), true
}

// Existing returns the name of the word list file in dir named like name
// (any extension, ignoring case), keeping its case.
func Existing(dir, name string) (string, bool) {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if base, ok := listFile(e.Name()); ok && !e.IsDir() && strings.EqualFold(base, name) {
			return base, true
		}
	}
	return "", false
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
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("import: %w", err)
	}
	name := c.Name
	if old, ok := Existing(dir, c.Name); ok {
		name = old
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
```

`LoadDir` (`vocab/dir.go`) may use `listFile` for its extension check if that reads cleaner; keep its behaviour unchanged.

- [ ] **Step 4: Run** `go test ./vocab/ -timeout 300s` → PASS.

---

### Task 3: Reloading lists in `GameState`

**Files:** Modify `fyne/wordgame.go` (`loadWords`, around line 97), test in `fyne/importlist_test.go` (create).

- [ ] **Step 1: Failing tests** (`fyne/importlist_test.go`)

```go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/vocab"
)

func TestStartupCreatesTheWordsFolder(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	NewGameState(openTestStore(t))
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("words folder: %v", err)
	}
}

func TestImportListReloadsAndKeepsProgress(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	c, err := vocab.Inspect("mine.csv", []byte("kana;meaning\nあお;blue\n"))
	if err != nil {
		t.Fatal(err)
	}
	if gs.HasList("mine") {
		t.Fatal("HasList before the import")
	}
	if name, err := gs.ImportList(c); err != nil || name != "mine" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if !equalIDs(gs.listNames, []string{"mine"}) || !gs.HasList("MINE") {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if _, ok := gs.words.Lookup("あお"); !ok {
		t.Fatal("the imported word is not in the index")
	}
	gs.mu.Lock()
	setWordLevel(gs, "あお", 3)
	gs.introduced["あお"] = true
	gs.wordSource = listSourcePrefix + "mine"
	gs.mu.Unlock()

	c, _ = vocab.Inspect("Mine.json", []byte(`[{"kana":"いえ","meaning":"house"}]`))
	if name, err := gs.ImportList(c); err != nil || name != "mine" {
		t.Fatalf("name %q, err %v; the list keeps its name", name, err)
	}
	if !equalIDs(gs.listNames, []string{"mine"}) {
		t.Fatalf("after replacing: lists = %v", gs.listNames)
	}
	gs.mu.Lock()
	_, found := gs.source()
	gs.mu.Unlock()
	if !found || !gs.introduced["あお"] {
		t.Fatal("the selected list must still be found and its introduced words kept")
	}
	if _, ok := gs.words.Lookup("いえ"); !ok {
		t.Fatal("the new word is not in the index")
	}
	if gs.wordProgress["あお"].Level != 3 {
		t.Fatal("replacing a list must keep the progress of its old words")
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.json")); err != nil {
		t.Fatal("the new file is stored under the list's name: mine.json")
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); !os.IsNotExist(err) {
		t.Fatal("the old file must be gone")
	}
}
```

Check `setWordLevel` in `fyne/learning_test.go` for its exact signature and how it keys `wordProgress` (reading or `word:` prefix); adapt the assertion to match. If `NewGameState` needs selected rows (see `TestWordSourceResolvedAndSavedOnFirstStart`), save `[]string{"vowels"}` first as it does.

- [ ] **Step 2: Run, expect a build failure.**

- [ ] **Step 3: Implement** in `fyne/wordgame.go`: replace the list part of `loadWords` with a call to `loadLists`, and create the folder first.

```go
// wordsDirPerm is the permission of the word list folder.
const wordsDirPerm = 0o755

// loadWords builds the word index: the private lists merged into the
// bundled index, which is built once per process. A bundled load error is
// kept for the status panel; kana play is unaffected. The word list folder
// is created if it is missing. Called from NewGameState before the game
// starts, so no lock is needed.
func (gs *GameState) loadWords() {
	if _, err := bundledWords(); err != nil {
		gs.wordsErr = err.Error()
	}
	if dir, err := wordsDir(); err == nil {
		_ = os.MkdirAll(dir, wordsDirPerm) // an error shows as listsLoadFailed
	}
	gs.loadLists()
}

// loadLists reads the private lists and rebuilds the word index on the
// bundled one (an empty index if that failed to load). An unreadable folder
// gives no private lists and sets listsLoadFailed. Must be called under lock
// (or before the game starts).
func (gs *GameState) loadLists() {
	base := vocab.NewIndex(nil, nil)
	if gs.wordsErr == "" {
		base = bundledIndex()
	}
	var lists []vocab.List
	gs.listsLoadFailed = true
	if dir, err := wordsDir(); err == nil {
		gs.wordsDir = dir
		var loadErr error
		lists, loadErr = vocab.LoadDir(dir)
		gs.listsLoadFailed = loadErr != nil
	}
	gs.listNames = nil
	for _, l := range lists {
		gs.listNames = append(gs.listNames, l.Name)
	}
	gs.words = base.WithLists(lists)
	gs.invalidatePlayable()
}

// ReloadLists reads the word list folder again. The word source is left as
// it is.
func (gs *GameState) ReloadLists() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.loadLists()
}

// HasList reports whether a word list of that name exists (ignoring case
// and extension).
func (gs *GameState) HasList(name string) bool {
	gs.mu.Lock()
	dir := gs.wordsDir
	gs.mu.Unlock()
	if dir == "" {
		return false
	}
	_, ok := vocab.Existing(dir, name)
	return ok
}

// ImportList stores c in the word list folder, reloads the lists and
// returns the word list's name (an existing list keeps its name).
func (gs *GameState) ImportList(c vocab.Candidate) (string, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.wordsDir == "" {
		return "", errors.New("no folder for word lists")
	}
	name, err := vocab.Import(gs.wordsDir, c)
	if err != nil {
		return "", err
	}
	gs.loadLists()
	return name, nil
}
```

Before relying on `gs.wordsErr == ""` meaning "bundled loaded", check what `loadWords` did before (it set `wordsErr` only on a `bundledWords()` error) and that nothing else sets `wordsErr`. `TestWordSourceNotSavedWhenTheListsFailToLoad` puts a file where the folder should be; `MkdirAll` then fails and `listsLoadFailed` stays true, as the test expects.

- [ ] **Step 4: Run** `go test ./fyne/ -timeout 300s` → PASS (the whole package: the startup change touches every `NewGameState` test).

---

### Task 4: Preview, Words tab and dialog

**Files:** Create `fyne/importlist.go`; modify `fyne/settings.go`, `fyne/importlist_test.go`, `fyne/settings_test.go`.

- [ ] **Step 1: Failing tests**

In `fyne/importlist_test.go`:

```go
func TestImportPreviewLines(t *testing.T) {
	c, err := vocab.Inspect("minna-no-nihongo.csv", []byte(
		"kana;meaning_de;meaning_en;lesson\nせんせい;Lehrer;teacher;1\nがくせい;Student;student;1\nえき;Bahnhof;station;2\nいえ;Haus;house;2\nabc;x;x;3\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := importPreviewLines(c, "de", true)
	want := []string{
		"Word list: minna no nihongo",
		"Replaces the word list of the same name; your progress is kept.",
		"4 words · 2 lessons · meanings: de, en",
		"1 row skipped:",
		"• line 6: abc is not kana", // the parser's message; copy the exact text from vocab/csvlist.go
		"First words:",
		"せんせい · Lehrer · lesson 1",
		"がくせい · Student · lesson 1",
		"えき · Bahnhof · lesson 2",
	}
	if !equalIDs(got, want) {
		t.Fatalf("lines:\n%q\nwant\n%q", got, want)
	}

	c, _ = vocab.Inspect("a.csv", []byte("kana;meaning\nせんせい;teacher\n"))
	if got := importPreviewLines(c, "en", false); !equalIDs(got, []string{
		"Word list: a", "1 word", "First words:", "せんせい · teacher",
	}) {
		t.Fatalf("plain list: %q", got)
	}
}

func TestImportChosenShowsPreviewAndImports(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	f := newSettingsForm(gs.settingsState(), nil)

	importChosen(gs, f, "mine.csv", []byte("kana;meaning\nあお;blue\n"), w)
	preview := w.Canvas().Overlays().Top()
	if preview == nil || buttonIn(preview, "Import") == nil {
		t.Fatal("no preview with an Import button")
	}
	test.Tap(buttonIn(preview, "Import"))
	if !equalIDs(gs.listNames, []string{"mine"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if f.wordSource() != "list:mine" || !slices.Contains(f.sourceSelect.Options, "mine") {
		t.Fatalf("source %q, options %v; want the new list preselected", f.wordSource(), f.sourceSelect.Options)
	}

	importChosen(gs, f, "MINE.csv", []byte("kana;meaning\nいえ;house\n"), w)
	if buttonIn(w.Canvas().Overlays().Top(), "Replace") == nil {
		t.Fatal("an existing name must offer Replace")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Replace"))
	if !equalIDs(gs.listNames, []string{"mine"}) || f.wordSource() != "list:mine" {
		t.Fatalf("lists %v, source %q; the replaced list keeps its name", gs.listNames, f.wordSource())
	}
}

func TestImportChosenRejectsWithHint(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	w := test.NewWindow(nil)
	defer w.Close()
	f := newSettingsForm(gs.settingsState(), nil)
	importChosen(gs, f, "list.xlsx", []byte("PK"), w)
	top := w.Canvas().Overlays().Top()
	if top == nil || !labelContains(top, "CSV UTF-8") {
		t.Fatal("a rejected file must show the reason and the hint")
	}
	if len(gs.listNames) != 0 {
		t.Fatal("nothing may be imported")
	}
}
```

Add a helper if none exists (check `grep -n "func labelContains\|func labelTexts" fyne/*_test.go` first and reuse what is there):

```go
// labelContains reports whether a label or rich text under o contains s.
func labelContains(o fyne.CanvasObject, s string) bool {
	for _, obj := range test.LaidOutObjects(o) {
		switch v := obj.(type) {
		case *widget.Label:
			if strings.Contains(v.Text, s) {
				return true
			}
		case *widget.RichText:
			if strings.Contains(v.String(), s) {
				return true
			}
		}
	}
	return false
}
```

In `fyne/settings_test.go`, `TestSettingsFormCarriesValues`: assert the Words tab has a button "Import word list…" and a label containing "Word lists are kept in". `settingsGame()` has no `wordsDir`; set `gs.wordsDir = filepath.Join(home, ".local", "share", "kana", "words")` with `home, _ := os.UserHomeDir()` in the test and expect "Word lists are kept in ~/.local/share/kana/words" (use `filepath.Join` for the expected separator). With no `wordsDir` the line is hidden (empty text is fine).

Add a test that an import does not change the word source by itself (the settings' Cancel does not call `apply`, which `TestSettingsDialogSaveAndInvalidValue` already relies on, so the list stays and only the preselection is dropped):

```go
func TestImportDoesNotChangeTheSource(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	before, _ := st.WordSource()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	f := newSettingsForm(gs.settingsState(), nil)
	importChosen(gs, f, "mine.csv", []byte("kana;meaning\nあお;blue\n"), w)
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if after, _ := st.WordSource(); gs.wordSource != before || after != before {
		t.Fatalf("source %q, stored %q; want %q until the settings are saved", gs.wordSource, after, before)
	}
	if !equalIDs(gs.listNames, []string{"mine"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
}
```

- [ ] **Step 2: Run, expect a build failure.**

- [ ] **Step 3: Implement `fyne/settings.go` changes**

1. `settingsState` gets `wordsDir string // the word list folder for display; "" if unknown`, filled in `settingsState()` as `displayDir(gs.wordsDir, home)` when `gs.wordsDir != ""` (`home, _ := os.UserHomeDir()`, as `wordsStatus` does).
2. `settingsForm` gets `storedSource string` (the stored word source, for `setLists`), `folder *widget.Label`, and `onImport func() // the import button was clicked; may be nil`.
3. Extract the option building from `newWordsTab`:

```go
// sourceOptions are the source select's labels and word sources: the lists
// by name, a missing stored list (it stays selected until the learner picks
// another), then JLPT.
func sourceOptions(lists []string, stored string) (labels, values []string) {
	for _, name := range lists {
		labels = append(labels, prettyListName(name))
		values = append(values, listSourcePrefix+name)
	}
	if name, ok := strings.CutPrefix(stored, listSourcePrefix); ok && !slices.Contains(lists, name) {
		labels = append(labels, prettyListName(name))
		values = append(values, stored)
	}
	return append(labels, "JLPT"), append(values, jlptSource)
}

// setLists replaces the source select's lists and selects the list named
// selected.
func (f *settingsForm) setLists(lists []string, selected string) {
	labels, values := sourceOptions(lists, f.storedSource)
	f.sourceValues = values
	f.sourceSelect.SetOptions(labels)
	if i := slices.Index(values, listSourcePrefix+selected); i >= 0 {
		f.sourceSelect.SetSelectedIndex(i)
	}
}
```

`newWordsTab` uses `sourceOptions(st.lists, st.wordSource)` and sets `f.storedSource = st.wordSource`. Check that `widget.Select.SetOptions` keeps or clears the selection in Fyne v2.7.3 (read `fyne.io/fyne/v2/widget/select.go` in the module cache) and select explicitly either way.

4. In the Words tab, below `f.sourceSelect`:

```go
	importButton := widget.NewButton("Import word list…", func() {
		if f.onImport != nil {
			f.onImport()
		}
	})
	f.folder = widget.NewLabel("")
	if st.wordsDir != "" {
		f.folder.SetText("Word lists are kept in " + st.wordsDir)
	}
```

and add `importButton` after `f.sourceSelect`, `f.folder` after `f.summary`, in the `VBox`.

5. In `showSettingsDialog`, after `form := newSettingsForm(...)`: `form.onImport = func() { showImportDialog(gs, form, win) }`.

- [ ] **Step 4: Implement `fyne/importlist.go`**

```go
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"kana/vocab"
)

// importPreviewWords is how many words the import preview shows.
const importPreviewWords = 3

// importFileExts are the extensions the file dialog shows; spreadsheets are
// shown so that their hint can be given.
var importFileExts = []string{".csv", ".tsv", ".txt", ".json", ".xlsx", ".xls", ".ods"}

// importPreviewLines describes a candidate: its name, whether it replaces a
// word list, counts, skipped rows and the first words with meanings in lang.
func importPreviewLines(c vocab.Candidate, lang string, replaces bool) []string {
	lines := []string{"Word list: " + prettyListName(c.Name)}
	if replaces {
		lines = append(lines, "Replaces the word list of the same name; your progress is kept.")
	}
	lessons := map[string]bool{}
	langs := map[string]bool{}
	for _, w := range c.List.Words {
		if w.Lesson != "" {
			lessons[w.Lesson] = true
		}
		for l := range w.Meanings {
			if l != "" {
				langs[l] = true
			}
		}
	}
	counts := []string{plural(len(c.List.Words), "word")}
	if len(lessons) > 0 {
		counts = append(counts, plural(len(lessons), "lesson"))
	}
	if len(langs) > 0 {
		counts = append(counts, "meanings: "+strings.Join(slices.Sorted(maps.Keys(langs)), ", "))
	}
	lines = append(lines, strings.Join(counts, " · "))
	if c.List.Skipped > 0 {
		lines = append(lines, plural(c.List.Skipped, "row")+" skipped:")
		for _, p := range c.List.Problems {
			lines = append(lines, "• "+p)
		}
	}
	lines = append(lines, "First words:")
	for _, w := range c.List.Words[:min(importPreviewWords, len(c.List.Words))] {
		parts := []string{w.Kana}
		if m := meaningSnippet(w.Meanings, lang); m != "" {
			parts = append(parts, m)
		}
		if w.Lesson != "" {
			parts = append(parts, "lesson "+w.Lesson)
		}
		lines = append(lines, strings.Join(parts, " · "))
	}
	return lines
}

// plural is "1 word" or "4 words".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// showImportDialog lets the learner pick a source file, starting in the
// home directory. Must run on the Fyne thread.
func showImportDialog(gs *GameState, f *settingsForm, win fyne.Window) {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if r == nil {
			return // cancelled
		}
		defer r.Close()
		data, err := io.ReadAll(io.LimitReader(r, vocab.MaxImportSize+1))
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		importChosen(gs, f, r.URI().Name(), data, win)
	}, win)
	d.SetFilter(storage.NewExtensionFileFilter(importFileExts))
	if home, err := os.UserHomeDir(); err == nil {
		if l, err := storage.ListerForURI(storage.NewFileURI(home)); err == nil {
			d.SetLocation(l)
		}
	}
	d.Resize(win.Canvas().Size())
	d.Show()
}

// importChosen checks a chosen source file and shows its preview, or why it
// cannot be imported. Import (Replace for an existing name) stores it,
// reloads the lists and preselects it in f. Must run on the Fyne thread.
func importChosen(gs *GameState, f *settingsForm, fileName string, data []byte, win fyne.Window) {
	c, err := vocab.Inspect(fileName, data)
	if err != nil {
		showImportError(err, win)
		return
	}
	replaces := gs.HasList(c.Name)
	body := container.NewVBox()
	for _, line := range importPreviewLines(c, gs.MeaningLang(), replaces) {
		body.Add(widget.NewLabel(line))
	}
	confirm := "Import"
	if replaces {
		confirm = "Replace"
	}
	dialog.ShowCustomConfirm("Import word list", confirm, "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		name, err := gs.ImportList(c)
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		f.setLists(gs.ListNames(), name)
		// The summary may have said the chosen list was missing.
		f.summary.SetText(gs.settingsState().summary)
	}, win)
}

// showImportError shows why a file cannot be imported and what to do.
func showImportError(err error, win fyne.Window) {
	text := err.Error()
	var ie *vocab.ImportError
	if errors.As(err, &ie) && ie.Hint != "" {
		text += "\n\n" + ie.Hint
	}
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom("Cannot import", "OK", label, win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}
```

Add `"maps"` to the imports. `gs.MeaningLang()` and `gs.ListNames()` may not exist: check (`grep -n "func (gs \*GameState) MeaningLang\|func (gs \*GameState) ListNames" fyne/*.go`); add locked getters next to `ReloadLists` if missing:

```go
// ListNames returns the private word lists' names, sorted.
func (gs *GameState) ListNames() []string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return slices.Clone(gs.listNames)
}
```

and `MeaningLang()` returning `gs.meaningLang` under the lock.

The Fyne file dialog in a 1000 × 760 window: check it opens at a usable size; `d.Resize(win.Canvas().Size())` is the common fix. In `showImportError`, a wrapping label inside a custom dialog needs a width; if `Resize(480, 0)` misbehaves, use `dialog.NewCustom` with a `container.NewGridWrap(fyne.NewSize(460, ...))` or simply `dialog.ShowInformation("Cannot import", text, win)` (it wraps) — pick what the test window shows correctly.

- [ ] **Step 5: Run** `go test ./fyne/ -timeout 300s` and `go vet ./...` → PASS.

---

### Task 5: README

**Files:** Modify `README.md` (section "Private word lists", around line 372)

- [ ] **Step 1:** Rewrite the section's first paragraph: lists are added with **Settings → Words → Import word list…**, which checks the file, shows a preview (words, lessons, meaning languages, skipped rows, first words) and copies it into the words folder (`$XDG_DATA_HOME/kana/words/` or `~/.local/share/kana/words/`, created at startup; the Words tab shows where). A file of the same name is replaced; progress is kept. Files can still be copied into the folder by hand; they are read at startup. Mention: files must be UTF-8 ("CSV UTF-8" in spreadsheet programs) or UTF-16 with BOM (Excel "Unicode Text", `.txt`); `.xlsx`/`.ods` must be saved as CSV first; at most 5 MB. Keep the format bullets below as they are.
- [ ] **Step 2:** If the README's structure section lists the `vocab/` files, add `import.go`: `Inspect` and `Import` for the settings' import.

---

### Task 6: Final check

- [ ] `go test ./... -race -timeout 600s`, `go vet ./...`, `gofmt -l .` (empty).
- [ ] Manual: `go run ./fyne/`, Settings → Words → Import word list…: the dialog starts in the home folder; importing a CSV preselects it; importing it again offers Replace; an `.xlsx` shows the hint.
