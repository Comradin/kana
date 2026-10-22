# Word List Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Learners manage their word lists from the game: see each list with its counts and skipped rows, rename and delete lists, open the word list folder, pick up hand-copied files without a restart, and import by dropping files on the window.

**Architecture:** `vocab.List` remembers its file (`Path`); `vocab` gains `DeleteList`, `CheckListName` and `RenameList` on that path, sharing the naming rules of `LoadDir` through a new `scanDir`. `store` can move a lesson filter. `GameState` keeps a `listInfo` per list (counts, skipped rows, path) and gets `ReloadLists`, `DeleteList`, `RenameList`, `CheckListName`, which keep the lesson filter and the source of new words in step. The settings form gets one refresh path for import, delete and rename; a new word list dialog sits on top of it. A `dropImporter` on the window feeds dropped files one by one into the existing import preview, pausing the game with a new `importOpen` flag.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (`dialog.NewForm`, `dialog.NewConfirm`, `widget.Label.Selectable`, `Window.SetOnDropped`, `App.OpenURL`). No new dependencies.

Spec: `docs/superpowers/specs/2026-10-22-list-management-design.md`. Terms: `GLOSSARY.md` (word list, source file, import, skipped row, delete a word list, rename a word list, lesson filter). The UI never says "problem"; the field `vocab.List.Problems` keeps its name.

**Conventions:** comments like the surrounding code (short doc comment on every function, "Must be called under lock." where it applies). Every `GameState` field is read and written under `gs.mu`; exported methods lock, "Must be called under lock" helpers do not. UI code runs on the Fyne thread (Fyne callbacks or `fyne.Do`); package-level UI state is marked "Fyne-thread-only" like `dialogShowing`. Tests: `go test ./... -timeout 300s`; the `fyne` tests touch fields directly without the lock, as the existing tests do. **Test data is synthetic only**: every test that reaches the word list folder calls `isolateWordsDir(t)` first; nothing may read the real `~/.local/share/kana/words/` (private lists such as `minna-no-nihongo.csv` live there and must never enter the repository or a test). Implementers leave changes **uncommitted**; the controller reviews and commits.

---

## Decisions made in the plan

The spec leaves these open or calls them indicative; check them before Task 1.

1. **`vocab` signatures take the path:** `DeleteList(path)`, `CheckListName(path, newName) (string, error)`, `RenameList(path, newName) (string, error)`. The folder is `filepath.Dir(path)`; the spec's `RenameList(dir, l, newName)` carried the same information twice.
2. **Name errors are exported sentinel errors with whole sentences** (`ErrNameEmpty` "Enter a name.", `ErrNameSeparator` `The name must not contain "/" or "\".`, `ErrNameDot` `The name must not start with ".".`, `ErrNameReserved` "The name jmdict is reserved for the bundled word list.", `ErrNameTaken` "Another word list has this name."), not `ImportError`: they need no hint, `errors.Is` tests them, and they show directly as the rename entry's validation message.
3. **A name starting with "." is rejected** (not in the spec): `LoadDir` hides dot files, so the list would vanish.
4. **"Another list's name"** is checked against every other list file's file name *and* its list name, ignoring case, so neither `minna` (beside `minna.csv`) nor a suffixed name like `a (csv)` or `a (csv) 2` can be taken. In addition the target file must not exist (a folder `ours.csv` blocks `ours`): an existing file is never overwritten. Characters other than `/` and `\` that an OS forbids (`:` on Windows) are left to the OS; its error is shown.
5. **Case-only rename** goes through a hidden temporary name `.<file>.rename` (skipped by `LoadDir`); if the second step fails the file is renamed back. Renaming to the identical name changes nothing and succeeds.
6. **Deleting a file that is already gone counts as deleted** (no error; filter and source follow as for a delete). The spec lists "file gone" among errors to show; for delete that would leave the source pointing at a missing list for no gain. Renaming a file that is gone shows the OS error, and the lists are reloaded. (Deviation — confirm.)
7. **Delete of the chosen source** switches it to JLPT *and clears the newest package*, exactly like a source change in `applyWordSettings`, so JLPT's first package can come at once. **Rename** keeps the newest package: it holds readings, not list names, so "the newest package follows" needs no code — it is simply not cleared.
8. **Store:** one new method `RenameLessonFilter(from, to)` in a transaction; a stale filter under the new name is removed even when the old list had none. Dropping a filter uses the existing `SaveLessonFilter(list, nil)` (via `gs.setLessonFilter(name, nil)`); no extra delete method.
9. **`GameState.lists []listInfo`** (name, path, words, lessons, skipped, messages) is kept next to `listNames`, which stays the source of truth for the rest of the game (many tests set it by hand). Word count = rows read as words (`len(l.Words)`), lessons = distinct non-empty lessons, as in the import preview.
10. **Reload never touches a game without a folder:** `reloadLists` returns early when `gs.wordsDir == ""`. Hand-built test games (`newTestState`, `settingsGame`) have none, so opening their settings cannot read the real folder. Reload, delete and rename also drop falling word tiles that may no longer fall (`dropInactiveTiles`).
11. **Words line:** `WordsStatus.Skipped` is set only for a list source that is found; `formatWordsLine` appends " · N rows skipped" (`plural`) to whatever line it produced, including the words-only reasons.
12. **Dialog layout:** title "Word lists", dismiss "Close". Per row: the pretty list name in bold, below it "4 words · 2 lessons" (lessons left out at 0) plus " · 1 row skipped"; on the right "Details" (only with skipped rows), "Rename", "Delete". Rows scroll (min 560 × 280). Footer: "Import…" and "Open folder" (both disabled without a folder). Rows show pretty names, like the source select; the rename entry is prefilled with the **file name without extension**, because that is what is renamed (for "minna (json)" it is `minna`, which is taken while `minna.csv` exists, so Rename starts disabled).
13. **Rename dialog** is `dialog.NewForm` ("Rename <name>", "Rename"/"Cancel") with the entry's `Validator` calling `GameState.CheckListName`: Rename is disabled while the name is invalid and the reason shows under the entry; the rename re-checks. Hint text: "The file keeps its extension (.csv)."
14. **Details** dialog, title "<name>: N rows skipped": one "• message" line per stored message, then "Correct these lines in your source file and import it again; the rest of the list plays meanwhile." No "and N more" line.
15. **Open folder:** the URL is `file://` plus the slash path with a leading `/` (`file:///C:/…` on Windows), built by `folderURL`. `OpenURL` runs synchronously (desktop drivers return at once: `Start` on Linux, `open`/`rundll32` on macOS/Windows). The folder is created if missing. On any error a dialog "Word list folder" shows the path in a selectable label. `openURL` is a package variable so tests replace it.
16. **Drop pause:** a new `GameState.importOpen` with `SetImportOpen`, treated like `settingsOpen` in `settle` and `buildSnapshot` (tiles hidden). A separate flag, so closing a preview shown over the settings does not release the settings' pause. `Reset` does not touch it (the preview is modal; nothing restarts the round under it).
17. **Drop queue:** `dropImporter` (built in `buildWindow`) queues URIs; a drop while a preview shows is appended. After the last file it refreshes the stats panel and, when no settings dialog is open, no game dialog is showing (`dialogShowing`) and the game is not over, calls `showPendingDialogs` (which resumes). Otherwise the dialog underneath is simply there again and resumes the game when it closes. The `SetOnDropped` callback goes through `fyne.Do`.
18. **The open settings dialog** is tracked in a Fyne-thread-only package variable `openSettings *settingsForm`, set by `showSettingsDialog` and cleared by the dialog's `SetOnClosed` (covers Save, Cancel and the reset path). A dropped file imported while it is open is preselected there, exactly as with "Import word list…".
19. **A dropped folder** shows the import's "Cannot import" dialog with "This is a folder." and the hint "Drop the word list files themselves." Unsuitable files get `vocab.Inspect`'s messages as usual.
20. **`importChosen`** gains a `then func()` (may be nil) called once its preview or error dialog is closed, whatever the answer, and accepts a nil form. `showImportError` gains the same `then`.
21. **One refresh path in the form:** `listsChanged(st, selected)` (lists, lessons, meaning languages, summary, stored source, open word list dialog), with `listImported`, `listDeleted`, `listRenamed` in front. `setLists` now takes a word source value; one that is not offered selects JLPT. After a rename, the form's lesson checks (also unsaved edits) and its selection follow the new name; after a delete, a selection of the deleted list becomes JLPT and its checks are dropped. A failed delete or rename still refreshes (the folder may have changed).
22. **GLOSSARY** already has the terms (spec commit); no GLOSSARY task. README gets one task.
23. **Accepted gap — duplicate-name siblings:** deleting or renaming one list of a pair such as "minna (csv)" / "minna (json)" makes the other reload as "minna"; its stored lesson filter and its place as the source (keyed by the old name) are not carried over, so the source shows as missing until chosen again. Rare; not handled.
24. **gofmt:** Task 6 adds longer `settingsForm` fields; run `gofmt -w fyne/settings.go` after editing so `onReset` and its neighbours stay aligned.

---

## File structure

- Modify `vocab/vocab.go`: `List.Path`.
- Modify `vocab/dir.go`: `listEntry`, `scanDir`; `LoadDir` and `loadFile` use it and set `Path`.
- Create `vocab/manage.go`: name errors, `DeleteList`, `CheckListName`, `RenameList`.
- Modify `vocab/dir_test.go`; create `vocab/manage_test.go`.
- Modify `store/store.go`, `store/store_test.go`: `RenameLessonFilter`.
- Modify `fyne/game.go`: fields `lists`, `importOpen`; `settle`, `buildSnapshot`; `SetImportOpen`.
- Modify `fyne/wordgame.go`: `loadLists` fills `gs.lists`; `wordsStatus` sets `Skipped`.
- Modify `fyne/words.go`: `WordsStatus.Skipped`, `formatWordsLine`.
- Create `fyne/lists.go`: `listInfo`, `newListInfo`, `fileBase`, `listByName`, `errListGone`, `ListInfos`, `WordsDir`, `ReloadLists`, `reloadLists`, `DeleteList`, `RenameList`, `CheckListName`, `moveLessonFilter`.
- Modify `fyne/settings.go`: `openSettings`, `onManage`, `onListsChanged`, "Manage word lists…" button, `setLists`, `listsChanged`, `listImported`, `listDeleted`, `listRenamed`, reload in `showSettingsDialog`.
- Modify `fyne/importlist.go`: `importChosen(…, then)`, nil form, `showImportError(…, then)`, `showErrorThen`.
- Create `fyne/listsdialog.go`: `listsDialog`, `showListsDialog`, `listCounts`, `skippedRowsLines`, `showSkippedRows`, rename and delete flows, `openURL`, `folderURL`, `openWordsFolder`, `showFolderPath`.
- Create `fyne/drop.go`: `dropImporter`, `readDropped`, `errFolder`.
- Modify `fyne/app.go`: `SetOnDropped`.
- Create tests `fyne/lists_test.go`, `fyne/listsdialog_test.go`, `fyne/drop_test.go`; modify `fyne/words_test.go`, `fyne/importlist_test.go`, `fyne/settings_test.go`.
- Modify `README.md`.

---

### Task 1: `vocab` — lists remember their file

**Files:** Modify `vocab/vocab.go` (`List`, line 26), `vocab/dir.go` (whole file), `vocab/dir_test.go` (`TestLoadDir`).

- [ ] **Step 1: Failing test.** In `vocab/dir_test.go`, `TestLoadDir`, after `m := byName["minna"]` and its word-count check, add:

```go
	if m.Path != filepath.Join(dir, "minna.csv") {
		t.Fatalf("Path = %q, want the list's file", m.Path)
	}
	if a := byName["a (json)"]; a.Path != filepath.Join(dir, "a.json") {
		t.Fatalf("a (json) Path = %q: each list knows its own file", a.Path)
	}
```

- [ ] **Step 2: Run** `go test ./vocab/ -run TestLoadDir -timeout 300s` → build failure (`m.Path undefined`).

- [ ] **Step 3: Implement.** In `vocab/vocab.go`, `List`:

```go
// List is one private word list file.
type List struct {
	Name     string
	Path     string // the list's file; "" for a list not read from a folder
	Words    []Word
	Problems []string // at most maxProblems messages
	Skipped  int      // rows skipped, including those without a message
}
```

Replace `LoadDir` and `loadFile` in `vocab/dir.go` (keep the package clause and imports; the doc comment of `LoadDir` stays as it is):

```go
func LoadDir(dir string) ([]List, error) {
	files, err := scanDir(dir)
	if err != nil {
		return nil, err
	}
	var lists []List
	for _, f := range files {
		lists = append(lists, loadFile(f.name, f.path, f.ext))
	}
	sort.Slice(lists, func(i, j int) bool { return lists[i].Name < lists[j].Name })
	return lists, nil
}

// listEntry is a word list file found by scanDir: the name LoadDir gives its
// list, its path, its file name without extension and its lower-case
// extension.
type listEntry struct{ name, path, base, ext string }

// scanDir finds the word list files in dir and names their lists as LoadDir
// does, in directory order, without reading them. A missing directory gives
// none and no error.
func scanDir(dir string) ([]listEntry, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []listEntry
	count := map[string]int{}
	for _, e := range entries {
		// Dot files are hidden: ".csv" would be a list without a name and
		// macOS writes "._x.csv" resource forks next to copied files.
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".csv" && ext != ".tsv" && ext != ".json" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		files = append(files, listEntry{path: filepath.Join(dir, e.Name()), base: base, ext: ext})
		count[strings.ToLower(base)]++
	}
	used := map[string]bool{}
	for i, f := range files {
		key := strings.ToLower(f.base)
		name := f.base
		if count[key] > 1 || key == BundledSource {
			name = f.base + " (" + strings.TrimPrefix(f.ext, ".") + ")"
		}
		if used[name] {
			n := 2
			for used[name+" "+strconv.Itoa(n)] {
				n++
			}
			name += " " + strconv.Itoa(n)
		}
		used[name] = true
		files[i].name = name
	}
	return files, nil
}

// loadFile reads one list file; an open error becomes the list's problem.
// The list remembers its path.
func loadFile(name, path, ext string) List {
	var l List
	fh, err := os.Open(path)
	switch {
	case err != nil:
		l = List{Name: name}
		l.problem(err.Error())
	case ext == ".json":
		l = readJSON(name, fh)
	default:
		l = readCSV(name, fh)
	}
	if fh != nil {
		fh.Close()
	}
	l.Path = path
	return l
}
```

- [ ] **Step 4: Run** `go test ./vocab/ -timeout 300s` → PASS (`TestLoadDirNames`, `TestLoadDirSkipsDotFiles` unchanged).

---

### Task 2: `vocab` — delete, check and rename a word list

**Files:** Create `vocab/manage.go`, `vocab/manage_test.go`.

- [ ] **Step 1: Failing tests** (`vocab/manage_test.go`; `dirNames` is in `import_test.go`):

```go
package vocab

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// writeLists writes a one-word list file per name into dir.
func writeLists(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, name := range names {
		content := "kana\nねこ\n"
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			content = `[{"kana":"ねこ"}]`
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeleteList(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv", "other.csv")
	path := filepath.Join(dir, "mine.csv")
	if err := DeleteList(path); err != nil {
		t.Fatal(err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"other.csv"}) {
		t.Fatalf("files = %v, want only other.csv", got)
	}
	if err := DeleteList(path); err != nil {
		t.Fatalf("a file already gone counts as deleted: %v", err)
	}
}

func TestCheckListName(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv", "other.csv", "dup.csv", "dup.json")
	path := filepath.Join(dir, "mine.csv")
	for _, tc := range []struct {
		in   string
		want error
	}{
		{"", ErrNameEmpty},
		{"   ", ErrNameEmpty},
		{"a/b", ErrNameSeparator},
		{`a\b`, ErrNameSeparator},
		{".hidden", ErrNameDot},
		{"JMdict", ErrNameReserved},
		{"other", ErrNameTaken},
		{"OTHER", ErrNameTaken},
		{"Dup", ErrNameTaken},       // the file name of two other lists
		{"dup (CSV)", ErrNameTaken}, // the list name of another list
	} {
		if _, err := CheckListName(path, tc.in); !errors.Is(err, tc.want) {
			t.Errorf("CheckListName(%q) = %v, want %v", tc.in, err, tc.want)
		}
	}
	for in, want := range map[string]string{"mine": "mine", "Mine": "Mine", "  ours ": "ours"} {
		if got, err := CheckListName(path, in); err != nil || got != want {
			t.Errorf("CheckListName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestRenameListKeepsExtension(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.TSV")
	name, err := RenameList(filepath.Join(dir, "mine.TSV"), " ours ")
	if err != nil || name != "ours" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"ours.TSV"}) {
		t.Fatalf("files = %v, want ours.TSV", got)
	}
	lists, err := LoadDir(dir)
	if err != nil || len(lists) != 1 || lists[0].Name != "ours" || lists[0].Path != filepath.Join(dir, "ours.TSV") {
		t.Fatalf("lists %+v, err %v", lists, err)
	}
}

func TestRenameListRejectsWithoutTouching(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv", "other.json")
	if _, err := RenameList(filepath.Join(dir, "mine.csv"), "Other"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("err = %v, want ErrNameTaken", err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"mine.csv", "other.json"}) {
		t.Fatalf("files = %v, want both unchanged", got)
	}
}

func TestRenameListCaseOnly(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv")
	name, err := RenameList(filepath.Join(dir, "mine.csv"), "Mine")
	if err != nil || name != "Mine" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"Mine.csv"}) {
		t.Fatalf("files = %v, want Mine.csv only", got)
	}
	if name, err := RenameList(filepath.Join(dir, "Mine.csv"), "Mine"); err != nil || name != "Mine" {
		t.Fatalf("the same name: %q, %v; want no change", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"Mine.csv"}) {
		t.Fatalf("files = %v after renaming to the same name", got)
	}
}

func TestRenameListOfDuplicateName(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "dup.csv", "dup.json")
	lists, _ := LoadDir(dir)
	var js List
	for _, l := range lists {
		if l.Name == "dup (json)" {
			js = l
		}
	}
	name, err := RenameList(js.Path, "dup2")
	if err != nil || name != "dup2" {
		t.Fatalf("name %q, err %v", name, err)
	}
	if got := dirNames(t, dir); !slices.Equal(got, []string{"dup.csv", "dup2.json"}) {
		t.Fatalf("files = %v; only the list's own file is renamed", got)
	}
	lists, _ = LoadDir(dir)
	if len(lists) != 2 || lists[0].Name != "dup" || lists[1].Name != "dup2" {
		t.Fatalf("lists = %+v, want dup and dup2", lists)
	}
}

func TestRenameListNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	writeLists(t, dir, "mine.csv")
	if err := os.Mkdir(filepath.Join(dir, "ours.csv"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := RenameList(filepath.Join(dir, "mine.csv"), "ours"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("err = %v, want ErrNameTaken for an existing file", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); err != nil {
		t.Fatal("the list must be untouched")
	}
}

func TestRenameListGone(t *testing.T) {
	dir := t.TempDir()
	if _, err := RenameList(filepath.Join(dir, "gone.csv"), "ours"); err == nil {
		t.Fatal("renaming a missing file must fail")
	}
}
```

- [ ] **Step 2: Run** `go test ./vocab/ -timeout 300s` → build failure.

- [ ] **Step 3: Implement** `vocab/manage.go`:

```go
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
```

- [ ] **Step 4: Run** `go test ./vocab/ -timeout 300s` and `go vet ./vocab/` → PASS.

---

### Task 3: `store` — move a lesson filter

**Files:** Modify `store/store.go` (after `SaveLessonFilter`, ~line 446), `store/store_test.go`.

- [ ] **Step 1: Failing test** (append to `store/store_test.go`):

```go
func TestRenameLessonFilter(t *testing.T) {
	st := openTemp(t)
	_ = st.SaveLessonFilter("minna", []string{"2", ""})
	_ = st.SaveLessonFilter("ours", []string{"9"}) // stale, from an earlier list of that name
	if err := st.RenameLessonFilter("minna", "ours"); err != nil {
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("ours"); !reflect.DeepEqual(l, []string{"2", ""}) {
		t.Fatalf("ours = %v, want the moved filter", l)
	}
	if l, _ := st.LessonFilter("minna"); l != nil {
		t.Fatalf("minna = %v, want none left", l)
	}
	if err := st.RenameLessonFilter("ours", "ours"); err != nil {
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("ours"); !reflect.DeepEqual(l, []string{"2", ""}) {
		t.Fatalf("the same name must keep the filter: %v", l)
	}
	if err := st.RenameLessonFilter("Ours", "ours"); err != nil { // no filter under "Ours"
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("ours"); l != nil {
		t.Fatalf("a list without a filter must not inherit a stale one: %v", l)
	}
}
```

- [ ] **Step 2: Run** `go test ./store/ -timeout 300s` → build failure.

- [ ] **Step 3: Implement** (after `SaveLessonFilter`):

```go
// RenameLessonFilter moves a word list's lesson filter to its new name; a
// filter stored under the new name is replaced, also when the list has
// none (all lessons). The same name changes nothing.
func (s *Store) RenameLessonFilter(from, to string) error {
	if from == to {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: rename lesson filter: %w", err)
	}
	defer tx.Rollback() // a no-op after Commit
	if _, err := tx.Exec(`DELETE FROM settings WHERE key = ?`, lessonFilterPrefix+to); err != nil {
		return fmt.Errorf("store: rename lesson filter: %w", err)
	}
	if _, err := tx.Exec(`UPDATE settings SET key = ? WHERE key = ?`, lessonFilterPrefix+to, lessonFilterPrefix+from); err != nil {
		return fmt.Errorf("store: rename lesson filter: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: rename lesson filter: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run** `go test ./store/ -timeout 300s` → PASS.

---

### Task 4: `GameState` — list infos, reload, skipped rows on the words line

**Files:** Create `fyne/lists.go`, `fyne/lists_test.go`. Modify `fyne/game.go` (field after `listNames`, line 79), `fyne/wordgame.go` (`loadLists`, `wordsStatus`), `fyne/words.go` (`WordsStatus`, `formatWordsLine`), `fyne/words_test.go`.

- [ ] **Step 1: Failing tests.** `fyne/lists_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

// writeList writes a synthetic word list file into dir, creating dir, and
// returns its path.
func writeList(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestListInfos(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	path := writeList(t, dir, "mine.csv", "kana;meaning;lesson\nあお;blue;1\nいえ;house;2\nabc;x;3\n")
	writeList(t, dir, "plain.csv", "kana;meaning\nうえ;up\n")
	gs := NewGameState(openTestStore(t))
	infos := gs.ListInfos()
	if len(infos) != 2 || infos[0].name != "mine" || infos[1].name != "plain" {
		t.Fatalf("infos = %+v, want mine and plain by name", infos)
	}
	m := infos[0]
	if m.path != path || m.fileBase() != "mine" || m.words != 2 || m.lessons != 2 || m.skipped != 1 ||
		!slices.Equal(m.problems, []string{"line 4: abc is not kana"}) {
		t.Fatalf("mine = %+v", m)
	}
	if p := infos[1]; p.lessons != 0 || p.skipped != 0 || p.words != 1 {
		t.Fatalf("plain = %+v", p)
	}
	if gs.WordsDir() != dir {
		t.Fatalf("WordsDir = %q, want %q", gs.WordsDir(), dir)
	}
}

func TestReloadListsPicksUpHandCopiedList(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	if len(gs.listNames) != 0 {
		t.Fatalf("lists = %v, want none yet", gs.listNames)
	}
	writeList(t, dir, "copied.csv", "kana;meaning\nあお;blue\n")
	gs.ReloadLists()
	if !equalIDs(gs.listNames, []string{"copied"}) || len(gs.ListInfos()) != 1 {
		t.Fatalf("lists = %v after reload", gs.listNames)
	}
	if _, ok := gs.words.Lookup("あお"); !ok {
		t.Fatal("the copied list's word is not in the index")
	}
}

func TestReloadWithoutFolderReadsNothing(t *testing.T) {
	gs := newTestState() // no word list folder: must never read the real one
	gs.listNames = []string{"hand-made"}
	gs.ReloadLists()
	if !equalIDs(gs.listNames, []string{"hand-made"}) {
		t.Fatalf("lists = %v; a game without a folder has nothing to reload", gs.listNames)
	}
}

func TestWordsStatusSkippedRowsOfChosenSourceOnly(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	writeList(t, dir, "mine.csv", "kana;meaning\nあお;blue\nabc;x\n")
	writeList(t, dir, "other.csv", "kana;meaning\nいえ;house\n")
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordSource(listSourcePrefix + "mine")
	gs := NewGameState(st)
	gs.mu.Lock()
	mine := gs.wordsStatus()
	gs.wordSource = listSourcePrefix + "other"
	other := gs.wordsStatus()
	gs.wordSource = jlptSource
	jlpt := gs.wordsStatus()
	gs.mu.Unlock()
	if mine.Skipped != 1 || !strings.HasSuffix(formatWordsLine(mine), " · 1 row skipped") {
		t.Fatalf("mine: skipped %d, line %q", mine.Skipped, formatWordsLine(mine))
	}
	if other.Skipped != 0 || jlpt.Skipped != 0 {
		t.Fatalf("other %d, JLPT %d; only the chosen source's skipped rows count", other.Skipped, jlpt.Skipped)
	}
}
```

In `fyne/words_test.go`, add to the `formatWordsLine` table (after "list finished"):

```go
		{"skipped rows", WordsStatus{Introduced: 3, Learned: 1, Source: "minna", Skipped: 2}, "Words: 3 introduced · 1 learned · minna · 2 rows skipped"},
		{"skipped rows, finished", WordsStatus{Next: wordsFinished, Source: "minna", Skipped: 1}, "Words: minna finished · 1 row skipped"},
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestListInfos|TestReload|TestWordsStatusSkipped|TestFormatWordsLine' -timeout 300s` → build failure. (Use the table test's real name if it differs: `grep -n "func Test" fyne/words_test.go`.)

- [ ] **Step 3: Implement.** `fyne/game.go`, after `listNames`:

```go
	lists                  []listInfo                // the private lists found, sorted by name: counts, skipped rows, file
```

`fyne/wordgame.go`, `loadLists`: replace

```go
	gs.listNames = nil
	for _, l := range lists {
		gs.listNames = append(gs.listNames, l.Name)
	}
```

with

```go
	gs.listNames, gs.lists = nil, nil
	for _, l := range lists {
		gs.listNames = append(gs.listNames, l.Name)
		gs.lists = append(gs.lists, newListInfo(l))
	}
```

`fyne/wordgame.go`, `wordsStatus`, after the `if src.List == "" { … }` block:

```go
	if found && src.List != "" {
		if li, ok := gs.listByName(src.List); ok {
			ws.Skipped = li.skipped
		}
	}
```

`fyne/words.go`, `WordsStatus`, after `Next`:

```go
	Skipped    int       // rows skipped in the chosen list; 0 for JLPT or a missing list
```

and in `formatWordsLine`, just before `return truncateRunes(s, maxWordsLineRunes)`:

```go
	if ws.Skipped > 0 {
		s += " · " + plural(ws.Skipped, "row") + " skipped"
	}
```

Create `fyne/lists.go` (Task 5 adds delete and rename to it):

```go
package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"kana/vocab"
)

// listInfo describes a private word list for the word list dialog and the
// status panel.
type listInfo struct {
	name     string   // the list's name, as in the source select
	path     string   // its file in the word list folder
	words    int      // rows read as words
	lessons  int      // distinct lessons; words without a lesson do not count
	skipped  int      // skipped rows
	problems []string // messages on skipped rows, at most 10
}

// newListInfo summarises a loaded list.
func newListInfo(l vocab.List) listInfo {
	lessons := map[string]bool{}
	for _, w := range l.Words {
		if w.Lesson != "" {
			lessons[w.Lesson] = true
		}
	}
	return listInfo{name: l.Name, path: l.Path, words: len(l.Words), lessons: len(lessons),
		skipped: l.Skipped, problems: slices.Clone(l.Problems)}
}

// fileBase is the list's file name without its extension: what a rename
// changes.
func (li listInfo) fileBase() string {
	file := filepath.Base(li.path)
	return strings.TrimSuffix(file, filepath.Ext(file))
}

// listByName returns the list named name. Must be called under lock.
func (gs *GameState) listByName(name string) (listInfo, bool) {
	for _, li := range gs.lists {
		if li.name == name {
			return li, true
		}
	}
	return listInfo{}, false
}

// errListGone says that a list to rename or delete is no longer there.
func errListGone(name string) error {
	return fmt.Errorf("The word list %s is no longer there.", prettyListName(name))
}

// ListInfos returns the private word lists, sorted by name. The messages
// are shared; do not modify them.
func (gs *GameState) ListInfos() []listInfo {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return slices.Clone(gs.lists)
}

// WordsDir returns the word list folder; "" if unknown.
func (gs *GameState) WordsDir() string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.wordsDir
}

// ReloadLists reads the word list folder again, so lists copied, edited or
// removed by hand come in (see reloadLists).
func (gs *GameState) ReloadLists() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.reloadLists()
}

// reloadLists reloads the lists after they changed, which invalidates the
// playable words, and drops word tiles that may no longer fall. A game
// without a word list folder (the tests' hand-built games) has nothing to
// reload and never reads the real folder. Must be called under lock.
func (gs *GameState) reloadLists() {
	if gs.wordsDir == "" {
		return
	}
	gs.loadLists()
	gs.dropInactiveTiles()
	gs.buildSnapshot()
}
```

- [ ] **Step 4: Run** `go test ./fyne/ -timeout 300s` → PASS (`TestSnapshotWordsStatus` still compares equal: `Skipped` is 0 there).

---

### Task 5: `GameState` — delete and rename a word list

**Files:** Modify `fyne/lists.go`, `fyne/lists_test.go`.

- [ ] **Step 1: Failing tests** (append to `fyne/lists_test.go`; add `"errors"` and `"kana/vocab"` to its imports):

```go
// listGame is a game on a fresh store with the given lists in its isolated
// word list folder, the vowels selected (no intro) and the given source.
func listGame(t *testing.T, source string, files map[string]string) (*GameState, string) {
	t.Helper()
	test.NewApp()
	dir := isolateWordsDir(t)
	for name, content := range files {
		writeList(t, dir, name, content)
	}
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordSource(source)
	return NewGameState(st), dir
}

const lessonList = "kana;meaning;lesson\nあお;blue;1\nいえ;house;2\n"

func TestDeleteListKeepsProgressAndFallsBackToJLPT(t *testing.T) {
	gs, dir := listGame(t, listSourcePrefix+"mine", map[string]string{"mine.csv": lessonList})
	_ = gs.store.SaveLessonFilter("mine", []string{"2"})
	gs.ReloadLists() // pick up the stored filter
	gs.mu.Lock()
	setWordLevel(gs, "あお", 3)
	gs.introduced["あお"] = true
	gs.newestPackage = []string{"あお"}
	gs.mu.Unlock()

	if err := gs.DeleteList("mine"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); !os.IsNotExist(err) {
		t.Fatal("the file must be gone")
	}
	if len(gs.listNames) != 0 || len(gs.ListInfos()) != 0 {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if stored, _ := gs.store.WordSource(); gs.wordSource != jlptSource || stored != jlptSource {
		t.Fatalf("source %q, stored %q; want JLPT", gs.wordSource, stored)
	}
	if l, _ := gs.store.LessonFilter("mine"); l != nil || gs.lessonFilters["mine"] != nil {
		t.Fatalf("the lesson filter must be removed: stored %v, game %v", l, gs.lessonFilters["mine"])
	}
	if gs.wordProgress["あお"].Level != 3 || !gs.introduced["あお"] {
		t.Fatal("progress on the list's words must be kept")
	}
	if gs.newestPackage != nil {
		t.Fatalf("newest package %v; a source change clears it", gs.newestPackage)
	}
}

func TestDeleteListThatIsNotTheSource(t *testing.T) {
	gs, _ := listGame(t, listSourcePrefix+"a", map[string]string{"a.csv": lessonList, "b.csv": lessonList})
	gs.newestPackage = []string{"あお"}
	if err := gs.DeleteList("b"); err != nil {
		t.Fatal(err)
	}
	if gs.wordSource != listSourcePrefix+"a" || !equalIDs(gs.listNames, []string{"a"}) || len(gs.newestPackage) != 1 {
		t.Fatalf("source %q, lists %v, newest %v", gs.wordSource, gs.listNames, gs.newestPackage)
	}
}

func TestRenameListCarriesFilterSourceAndNewestPackage(t *testing.T) {
	gs, dir := listGame(t, listSourcePrefix+"mine", map[string]string{"mine.csv": lessonList})
	_ = gs.store.SaveLessonFilter("mine", []string{"2"})
	gs.ReloadLists()
	gs.newestPackage = []string{"いえ"}

	to, err := gs.RenameList("mine", " ours ")
	if err != nil || to != "ours" {
		t.Fatalf("to %q, err %v", to, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ours.csv")); err != nil {
		t.Fatal("the file must be ours.csv")
	}
	if !equalIDs(gs.listNames, []string{"ours"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if stored, _ := gs.store.WordSource(); gs.wordSource != listSourcePrefix+"ours" || stored != gs.wordSource {
		t.Fatalf("source %q, stored %q; want list:ours", gs.wordSource, stored)
	}
	if l, _ := gs.store.LessonFilter("ours"); !equalIDs(l, []string{"2"}) || !equalIDs(gs.lessonFilters["ours"], []string{"2"}) {
		t.Fatalf("filter stored %v, game %v; want [2] under the new name", l, gs.lessonFilters["ours"])
	}
	if l, _ := gs.store.LessonFilter("mine"); l != nil {
		t.Fatalf("old filter %v left behind", l)
	}
	if !equalIDs(gs.newestPackage, []string{"いえ"}) {
		t.Fatalf("newest package %v; a rename keeps it", gs.newestPackage)
	}
	gs.mu.Lock()
	src, found := gs.source()
	gs.mu.Unlock()
	if !found || !equalIDs(src.Lessons, []string{"2"}) {
		t.Fatalf("source %+v, found %v", src, found)
	}
}

func TestRenameListRejected(t *testing.T) {
	gs, _ := listGame(t, jlptSource, map[string]string{"mine.csv": lessonList, "other.csv": lessonList})
	if _, err := gs.RenameList("mine", "OTHER"); !errors.Is(err, vocab.ErrNameTaken) {
		t.Fatalf("err = %v, want ErrNameTaken", err)
	}
	if err := gs.CheckListName("mine", "jmdict"); !errors.Is(err, vocab.ErrNameReserved) {
		t.Fatalf("CheckListName = %v, want ErrNameReserved", err)
	}
	if err := gs.CheckListName("mine", "Mine"); err != nil {
		t.Fatalf("a change of case only: %v", err)
	}
	if !equalIDs(gs.listNames, []string{"mine", "other"}) {
		t.Fatalf("lists = %v, want unchanged", gs.listNames)
	}
	if _, err := gs.RenameList("gone", "x"); err == nil {
		t.Fatal("renaming an unknown list must fail")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestDeleteList|TestRenameList' -timeout 300s` → build failure.

- [ ] **Step 3: Implement** (append to `fyne/lists.go`):

```go
// DeleteList removes the list's file for good and reloads the lists.
// Progress on its words is kept; its lesson filter is removed. When it was
// the source of new words, JLPT becomes the source and the newest package
// is cleared, as for any source change. A file already gone counts as
// deleted. After an error the lists are reloaded too.
func (gs *GameState) DeleteList(name string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	li, ok := gs.listByName(name)
	if !ok {
		return errListGone(name)
	}
	if err := vocab.DeleteList(li.path); err != nil {
		gs.reloadLists()
		return err
	}
	gs.setLessonFilter(name, nil)
	if gs.wordSource == listSourcePrefix+name {
		gs.wordSource = jlptSource
		gs.newestPackage = nil
		if gs.store != nil {
			_ = gs.store.SaveWordSource(jlptSource)
		}
		gs.saveNewestPackage()
	}
	gs.reloadLists()
	return nil
}

// RenameList renames the list's file (see vocab.RenameList), reloads the
// lists and returns the list's new name. Its lesson filter and its place as
// the source of new words follow it; the newest package holds readings and
// stays. After an error the lists are reloaded too.
func (gs *GameState) RenameList(name, newName string) (string, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	li, ok := gs.listByName(name)
	if !ok {
		return "", errListGone(name)
	}
	renamed, err := vocab.RenameList(li.path, newName)
	if err != nil {
		gs.reloadLists()
		return "", err
	}
	if renamed != name {
		gs.moveLessonFilter(name, renamed)
		if gs.wordSource == listSourcePrefix+name {
			gs.wordSource = listSourcePrefix + renamed
			if gs.store != nil {
				_ = gs.store.SaveWordSource(gs.wordSource)
			}
		}
	}
	gs.reloadLists()
	return renamed, nil
}

// CheckListName reports why the list name cannot be renamed to newName
// (see vocab.CheckListName); nil if it can.
func (gs *GameState) CheckListName(name, newName string) error {
	gs.mu.Lock()
	li, ok := gs.listByName(name)
	gs.mu.Unlock()
	if !ok {
		return errListGone(name)
	}
	_, err := vocab.CheckListName(li.path, newName)
	return err
}

// moveLessonFilter moves a list's lesson filter to its new name, in the game
// and in the store. Must be called under lock.
func (gs *GameState) moveLessonFilter(from, to string) {
	if gs.lessonFilters == nil {
		gs.lessonFilters = map[string][]string{}
	}
	if ls, ok := gs.lessonFilters[from]; ok {
		delete(gs.lessonFilters, from)
		gs.lessonFilters[to] = ls
	} else {
		delete(gs.lessonFilters, to)
	}
	if gs.store != nil {
		_ = gs.store.RenameLessonFilter(from, to)
	}
}
```

- [ ] **Step 4: Run** `go test ./fyne/ -timeout 300s` → PASS.

---

### Task 6: Settings — one refresh path, reload on open, import callback

**Files:** Modify `fyne/settings.go`, `fyne/importlist.go`, `fyne/importlist_test.go`, `fyne/settings_test.go`.

- [ ] **Step 1: Failing tests.** In `fyne/importlist_test.go` add `, nil` as the last argument of the five existing `importChosen(…)` calls (lines ~126, 139, 158, 180, 211). Then append:

```go
func TestImportChosenWithoutFormCallsThen(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	calls := 0
	then := func() { calls++ }

	importChosen(gs, nil, "mine.csv", []byte("kana;meaning\nあお;blue\n"), w, then)
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Cancel"))
	if calls != 1 || len(gs.listNames) != 0 {
		t.Fatalf("after Cancel: then called %d times, lists %v", calls, gs.listNames)
	}
	importChosen(gs, nil, "list.xlsx", []byte("PK"), w, then)
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "OK"))
	if calls != 2 {
		t.Fatalf("after the error: then called %d times", calls)
	}
	importChosen(gs, nil, "mine.csv", []byte("kana;meaning\nあお;blue\n"), w, then)
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if calls != 3 || !equalIDs(gs.listNames, []string{"mine"}) {
		t.Fatalf("after Import: then called %d times, lists %v", calls, gs.listNames)
	}
}
```

Append to `fyne/settings_test.go`:

```go
func TestWordsTabHasManageButton(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(settingsGame().settingsState(), nil)
	b := buttonIn(f.wordsTab.Content, "Manage word lists…")
	if b == nil {
		t.Fatal("no Manage word lists… button")
	}
	test.Tap(b) // a nil handler must not panic
	called := false
	f.onManage = func() { called = true }
	test.Tap(b)
	if !called {
		t.Fatal("onManage not called")
	}
}

func TestSettingsOpenReloadsLists(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	t.Cleanup(func() { openSettings = nil })
	dir := isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	writeList(t, dir, "copied.csv", "kana;meaning\nあお;blue\n") // by hand, after the start
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)

	showSettingsDialog(gs, statsPanel, gameCanvas, ib, w)
	if openSettings == nil || !slices.Contains(openSettings.sourceSelect.Options, "copied") {
		t.Fatal("the settings must show the hand-copied list")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Cancel"))
	if openSettings != nil {
		t.Fatal("openSettings must be cleared when the dialog closes")
	}
}
```

(Add `"slices"` to `settings_test.go`'s imports if it is not there.)

- [ ] **Step 2: Run** `go test ./fyne/ -timeout 300s` → build failure.

- [ ] **Step 3: Implement `fyne/settings.go`.**

After the `settingsForm` type, add:

```go
// openSettings is the form of the settings dialog while it is open, so a
// dropped file's import can show its list there; nil otherwise.
// Fyne-thread-only, like dialogShowing.
var openSettings *settingsForm
```

In `settingsForm`, replace the `onImport` line with:

```go
	onImport       func()                // the import button was clicked; may be nil
	onManage       func()                // the word lists button was clicked; may be nil
	onListsChanged func()                // the lists shown changed; set while the word list dialog is open
```

In `newWordsTab`, after `importButton`:

```go
	manageButton := widget.NewButton("Manage word lists…", func() {
		if f.onManage != nil {
			f.onManage()
		}
	})
```

and in the returned VBox replace `importButton,` with `container.NewHBox(importButton, manageButton),`.

Replace `setLists`:

```go
// setLists replaces the source select's lists and selects the word source
// selected; JLPT when it is not among them. Must run on the Fyne thread.
func (f *settingsForm) setLists(lists []string, selected string) {
	labels, values := sourceOptions(lists, f.storedSource)
	f.sourceValues = values
	f.sourceSelect.SetOptions(labels)
	i := slices.Index(values, selected)
	if i < 0 {
		i = len(values) - 1 // JLPT
	}
	f.sourceSelect.SetSelectedIndex(i)
	f.showLessons()
}

// listsChanged shows the word lists of st after an import, delete or rename:
// the source select with selected (a word source) selected, the lessons, the
// meaning languages and the summary. An open word list dialog refreshes
// too. Must run on the Fyne thread.
func (f *settingsForm) listsChanged(st settingsState, selected string) {
	f.storedSource = st.wordSource
	f.listLessons = st.lessons
	f.setLists(st.lists, selected)
	f.setMeaningLangs(st.languages)
	f.summary.SetText(st.summary)
	if f.onListsChanged != nil {
		f.onListsChanged()
	}
}

// listImported shows an imported list, selected. Its lesson checks start
// from its stored filter: checks shown so far belonged to the list it
// replaced. Must run on the Fyne thread.
func (f *settingsForm) listImported(st settingsState, name string) {
	if f.lessonList == name {
		f.lessonList = ""
	}
	if ls, ok := st.lessonFilters[name]; ok {
		f.lessonFilters[name] = ls
	} else {
		delete(f.lessonFilters, name)
	}
	f.listsChanged(st, listSourcePrefix+name)
}

// listDeleted drops a deleted list: its lesson checks go, and JLPT is
// selected if it was selected. Must run on the Fyne thread.
func (f *settingsForm) listDeleted(st settingsState, name string) {
	if f.lessonList == name {
		f.lessonList = ""
	}
	delete(f.lessonFilters, name)
	selected := f.wordSource()
	if selected == listSourcePrefix+name {
		selected = jlptSource
	}
	f.listsChanged(st, selected)
}

// listRenamed follows a renamed list: its lesson checks as edited in this
// dialog and its selection move to the new name. Must run on the Fyne
// thread.
func (f *settingsForm) listRenamed(st settingsState, from, to string) {
	f.keepLessons()
	if f.lessonList == from {
		f.lessonList = ""
	}
	if ls, ok := f.lessonFilters[from]; ok {
		delete(f.lessonFilters, from)
		f.lessonFilters[to] = ls
	}
	selected := f.wordSource()
	if selected == listSourcePrefix+from {
		selected = listSourcePrefix + to
	}
	f.listsChanged(st, selected)
}
```

In `showSettingsDialog` replace

```go
	form := newSettingsForm(gs.settingsState(), onReset)
	form.onImport = func() { showImportDialog(gs, form, win) }
```

with

```go
	gs.ReloadLists() // lists copied, edited or removed by hand come in
	form := newSettingsForm(gs.settingsState(), onReset)
	form.onImport = func() { showImportDialog(gs, form, win) }
```

and after `d = dialog.NewCustomWithoutButtons("Settings", form.tabs, win)`:

```go
	openSettings = form
	// Every way out (Save, Cancel, a reset) hides the dialog.
	d.SetOnClosed(func() { openSettings = nil })
```

- [ ] **Step 4: Implement `fyne/importlist.go`.** In `showImportDialog` call `importChosen(gs, f, r.URI().Name(), data, win, nil)`. Replace `importChosen` and `showImportError`:

```go
// importChosen checks a chosen source file and shows its preview, or why it
// cannot be imported. Import (Replace for an existing name) stores it and
// reloads the lists; a settings form f (may be nil) preselects it. then (may
// be nil) is called once the preview or error is closed, whatever the
// answer. Must run on the Fyne thread.
func importChosen(gs *GameState, f *settingsForm, fileName string, data []byte, win fyne.Window, then func()) {
	done := func() {
		if then != nil {
			then()
		}
	}
	c, err := vocab.Inspect(fileName, data)
	if err != nil {
		showImportError(err, win, then)
		return
	}
	files := gs.ListFiles(c.Name)
	body := container.NewVBox()
	for _, line := range importPreviewLines(c, gs.MeaningLang(), files) {
		label := widget.NewLabel(line)
		label.Wrapping = fyne.TextWrapWord
		body.Add(label)
	}
	confirm := "Import"
	if len(files) > 0 {
		confirm = "Replace"
	}
	d := dialog.NewCustomConfirm("Import word list", confirm, "Cancel", body, func(ok bool) {
		if !ok {
			done()
			return
		}
		name, err := gs.ImportList(c)
		if err != nil {
			showErrorThen(err, win, then)
			return
		}
		if f != nil {
			f.listImported(gs.settingsState(), name)
		}
		done()
	}, win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}

// showImportError shows why a file cannot be imported and what to do; then
// (may be nil) is called once it is closed.
func showImportError(err error, win fyne.Window, then func()) {
	text := err.Error()
	var ie *vocab.ImportError
	if errors.As(err, &ie) && ie.Hint != "" {
		text += "\n\n" + ie.Hint
	}
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom("Cannot import", "OK", label, win)
	if then != nil {
		d.SetOnClosed(then)
	}
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}

// showErrorThen shows err; then (may be nil) is called once it is closed.
func showErrorThen(err error, win fyne.Window, then func()) {
	d := dialog.NewError(err, win)
	if then != nil {
		d.SetOnClosed(then)
	}
	d.Show()
}
```

- [ ] **Step 5: Run** `go test ./fyne/ -timeout 300s` → PASS. The existing import tests (`TestImportChosenShowsPreviewAndImports`, `TestImportRefreshesMeaningLanguagesAndLessonFilter`) cover `listImported`. Check that the Words tab still fits the 1000 × 760 window (`TestSettingsDialogSaveAndInvalidValue`); the buttons share one row, so it should.

---

### Task 7: The word list dialog and "Open folder"

**Files:** Create `fyne/listsdialog.go`, `fyne/listsdialog_test.go`. Modify `fyne/settings.go` (`showSettingsDialog`: `onManage`).

- [ ] **Step 1: Failing tests** (`fyne/listsdialog_test.go`):

```go
package main

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/vocab"
)

// listsDialogGame opens the word list dialog over a settings form for a game
// with the given lists; source as in listGame.
func listsDialogGame(t *testing.T, source string, files map[string]string) (*GameState, *settingsForm, fyne.Window, string) {
	t.Helper()
	gs, dir := listGame(t, source, files)
	t.Cleanup(func() { test.NewApp() })
	w := test.NewWindow(nil)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(1000, 760))
	f := newSettingsForm(gs.settingsState(), nil)
	showListsDialog(gs, f, w)
	return gs, f, w, dir
}

// entryIn finds the first entry in a dialog's objects.
func entryIn(o fyne.CanvasObject) *widget.Entry {
	for _, obj := range test.LaidOutObjects(o) {
		if e, ok := obj.(*widget.Entry); ok {
			return e
		}
	}
	return nil
}

func TestListCountsAndSkippedLines(t *testing.T) {
	li := listInfo{name: "minna", words: 4, lessons: 2, skipped: 1, problems: []string{"line 6: abc is not kana"}}
	if got := listCounts(li); got != "4 words · 2 lessons · 1 row skipped" {
		t.Fatalf("listCounts = %q", got)
	}
	if got := listCounts(listInfo{words: 1}); got != "1 word" {
		t.Fatalf("listCounts without lessons = %q", got)
	}
	if got := skippedRowsLines(li); !equalIDs(got, []string{"• line 6: abc is not kana", skippedRowsHint}) {
		t.Fatalf("skippedRowsLines = %q", got)
	}
}

func TestListsDialogRows(t *testing.T) {
	_, _, w, _ := listsDialogGame(t, jlptSource, map[string]string{
		"minna-no-test.csv": "kana;meaning;lesson\nあお;blue;1\nいえ;house;2\nabc;x;3\n",
		"plain.csv":         "kana;meaning\nうえ;up\n",
	})
	top := w.Canvas().Overlays().Top()
	for _, s := range []string{"minna no test", "2 words · 2 lessons · 1 row skipped", "plain", "1 word"} {
		if !labelContains(top, s) {
			t.Errorf("no %q in the dialog", s)
		}
	}
	details := 0
	for _, o := range test.LaidOutObjects(top) {
		if b, ok := o.(*widget.Button); ok && b.Text == "Details" {
			details++
		}
	}
	if details != 1 {
		t.Fatalf("%d Details buttons, want one for the list with skipped rows", details)
	}
	for _, b := range []string{"Rename", "Delete", "Import…", "Open folder", "Close"} {
		if buttonIn(top, b) == nil {
			t.Errorf("no %q button", b)
		}
	}
	test.Tap(buttonIn(top, "Details"))
	if d := w.Canvas().Overlays().Top(); !labelContains(d, "line 4: abc is not kana") || !labelContains(d, "source file") {
		t.Fatal("Details must show the messages and how to fix the file")
	}
}

func TestListsDialogEmpty(t *testing.T) {
	_, _, w, _ := listsDialogGame(t, jlptSource, nil)
	if !labelContains(w.Canvas().Overlays().Top(), "No word lists yet") {
		t.Fatal("want No word lists yet")
	}
}

func TestListsDialogDelete(t *testing.T) {
	gs, f, w, dir := listsDialogGame(t, listSourcePrefix+"mine", map[string]string{"mine.csv": lessonList})
	lists := w.Canvas().Overlays().Top()
	test.Tap(buttonIn(lists, "Delete"))
	confirm := w.Canvas().Overlays().Top()
	if !labelContains(confirm, "Delete mine? Its file is removed; your progress on its words is kept.") {
		t.Fatal("no confirmation text")
	}
	test.Tap(buttonIn(confirm, "Cancel"))
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); err != nil {
		t.Fatal("Cancel must keep the file")
	}
	test.Tap(buttonIn(lists, "Delete"))
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Delete"))
	if _, err := os.Stat(filepath.Join(dir, "mine.csv")); !os.IsNotExist(err) {
		t.Fatal("the file must be gone")
	}
	if gs.wordSource != jlptSource || f.wordSource() != jlptSource || !equalIDs(f.sourceSelect.Options, []string{"JLPT"}) {
		t.Fatalf("game source %q, form source %q, options %v", gs.wordSource, f.wordSource(), f.sourceSelect.Options)
	}
	if w.Canvas().Overlays().Top() != lists || !labelContains(lists, "No word lists yet") {
		t.Fatal("the dialog must stay open and show no lists")
	}
}

func TestListsDialogRename(t *testing.T) {
	gs, f, w, dir := listsDialogGame(t, listSourcePrefix+"mine", map[string]string{
		"mine.csv": lessonList, "other.csv": lessonList,
	})
	_ = gs.store.SaveLessonFilter("mine", []string{"2"})
	gs.ReloadLists()
	f.listsChanged(gs.settingsState(), f.wordSource())
	f.lessonChecks["1"].SetChecked(false) // an unsaved edit: only lesson 2
	lists := w.Canvas().Overlays().Top()

	var rename *widget.Button // the first row's: mine
	for _, o := range test.LaidOutObjects(lists) {
		if b, ok := o.(*widget.Button); ok && b.Text == "Rename" && rename == nil {
			rename = b
		}
	}
	test.Tap(rename)
	form := w.Canvas().Overlays().Top()
	entry := entryIn(form)
	if entry == nil || entry.Text != "mine" {
		t.Fatal("the entry must hold the file name")
	}
	entry.SetText("jmdict")
	if err := entry.Validate(); !errors.Is(err, vocab.ErrNameReserved) {
		t.Fatalf("Validate = %v, want ErrNameReserved", err)
	}
	if b := buttonIn(form, "Rename"); b == nil || !b.Disabled() {
		t.Fatal("Rename must be disabled while the name is invalid")
	}
	entry.SetText("ours")
	test.Tap(buttonIn(form, "Rename"))
	if _, err := os.Stat(filepath.Join(dir, "ours.csv")); err != nil {
		t.Fatal("the file must be ours.csv")
	}
	if f.wordSource() != listSourcePrefix+"ours" || f.storedSource != listSourcePrefix+"ours" {
		t.Fatalf("form source %q, stored %q; want list:ours", f.wordSource(), f.storedSource)
	}
	if !equalIDs(checkedLessons(f), []string{"2"}) || !equalIDs(f.words().lessons, []string{"2"}) {
		t.Fatalf("checked %v; the lesson checks must follow the list", checkedLessons(f))
	}
	if !labelContains(lists, "ours") {
		t.Fatal("the dialog must show the new name")
	}
}

func TestFolderURL(t *testing.T) {
	if got := folderURL("/home/x/my words").String(); got != "file:///home/x/my%20words" {
		t.Fatalf("folderURL = %q", got)
	}
}

func TestOpenFolderShowsPathWhenItFails(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	orig := openURL
	t.Cleanup(func() { openURL = orig })
	dir := filepath.Join(t.TempDir(), "words")

	var opened *url.URL
	openURL = func(u *url.URL) error { opened = u; return nil }
	openWordsFolder(dir, w)
	if opened == nil || opened.Path != filepath.ToSlash(dir) || w.Canvas().Overlays().Top() != nil {
		t.Fatalf("opened %v; want the folder opened without a dialog", opened)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatal("a missing folder is created first")
	}

	openURL = func(*url.URL) error { return errors.New("no file manager") }
	openWordsFolder(dir, w)
	if top := w.Canvas().Overlays().Top(); top == nil || !labelContains(top, dir) {
		t.Fatal("a failure must show the path")
	}
}
```

Note on `Disabled()`: `dialog.NewForm` disables its confirm button through the entry's validation callback. If `SetText` in the test does not trigger it (the form only revalidates on a typed change in some Fyne versions), replace the `SetText` calls with `test.Type(entry, …)` after `entry.SetText("")`; keep the `Validate` assertion either way.

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestList|TestFolderURL|TestOpenFolder' -timeout 300s` → build failure.

- [ ] **Step 3: Implement** `fyne/listsdialog.go`:

```go
package main

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// skippedRowsHint ends the skipped rows' details.
const skippedRowsHint = "Correct these lines in your source file and import it again; the rest of the list plays meanwhile."

// listCounts is a list's line in the word list dialog: "4 words · 2
// lessons · 1 row skipped", without the parts that are 0.
func listCounts(li listInfo) string {
	parts := []string{plural(li.words, "word")}
	if li.lessons > 0 {
		parts = append(parts, plural(li.lessons, "lesson"))
	}
	if li.skipped > 0 {
		parts = append(parts, plural(li.skipped, "row")+" skipped")
	}
	return strings.Join(parts, " · ")
}

// skippedRowsLines are the skipped rows' details: a line per message, then
// how to fix the file.
func skippedRowsLines(li listInfo) []string {
	lines := make([]string, 0, len(li.problems)+1)
	for _, p := range li.problems {
		lines = append(lines, "• "+p)
	}
	return append(lines, skippedRowsHint)
}

// listsDialog is the word list dialog over the settings dialog's form f.
type listsDialog struct {
	gs   *GameState
	f    *settingsForm
	win  fyne.Window
	rows *fyne.Container
}

// showListsDialog shows the word lists, each with its counts and skipped
// rows, to rename or delete; another can be imported and the folder opened.
// Changes act at once and show in f. Must run on the Fyne thread.
func showListsDialog(gs *GameState, f *settingsForm, win fyne.Window) {
	ld := &listsDialog{gs: gs, f: f, win: win, rows: container.NewVBox()}
	ld.fill()
	scroll := container.NewVScroll(ld.rows)
	scroll.SetMinSize(fyne.NewSize(560, 280))
	importButton := widget.NewButton("Import…", func() {
		if f.onImport != nil {
			f.onImport()
		}
	})
	dir := gs.WordsDir()
	folderButton := widget.NewButton("Open folder", func() { openWordsFolder(dir, win) })
	if dir == "" {
		importButton.Disable()
		folderButton.Disable()
	}
	content := container.NewBorder(nil, container.NewHBox(importButton, folderButton), nil, nil, scroll)
	d := dialog.NewCustom("Word lists", "Close", content, win)
	f.onListsChanged = ld.fill
	d.SetOnClosed(func() { f.onListsChanged = nil })
	d.Show()
}

// fill shows a row per list, or that there are none. Must run on the Fyne
// thread.
func (ld *listsDialog) fill() {
	ld.rows.RemoveAll()
	infos := ld.gs.ListInfos()
	if len(infos) == 0 {
		ld.rows.Add(widget.NewLabel("No word lists yet"))
		return
	}
	for _, li := range infos {
		ld.rows.Add(ld.row(li))
	}
}

// row is a list's row: its name, counts and skipped rows, and its buttons.
func (ld *listsDialog) row(li listInfo) fyne.CanvasObject {
	buttons := container.NewHBox()
	if li.skipped > 0 {
		buttons.Add(widget.NewButton("Details", func() { showSkippedRows(li, ld.win) }))
	}
	buttons.Add(widget.NewButton("Rename", func() { ld.rename(li) }))
	buttons.Add(widget.NewButton("Delete", func() { ld.confirmDelete(li) }))
	text := container.NewVBox(
		widget.NewLabelWithStyle(prettyListName(li.name), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(listCounts(li)),
	)
	return container.NewBorder(nil, nil, nil, buttons, text)
}

// showSkippedRows shows a list's skipped rows and how to fix the file.
func showSkippedRows(li listInfo, win fyne.Window) {
	body := container.NewVBox()
	for _, line := range skippedRowsLines(li) {
		label := widget.NewLabel(line)
		label.Wrapping = fyne.TextWrapWord
		body.Add(label)
	}
	d := dialog.NewCustom(prettyListName(li.name)+": "+plural(li.skipped, "row")+" skipped", "OK", body, win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}

// confirmDelete asks before deleting a list, then deletes it and refreshes
// the settings (and so this dialog). Must run on the Fyne thread.
func (ld *listsDialog) confirmDelete(li listInfo) {
	c := dialog.NewConfirm("Delete word list",
		"Delete "+prettyListName(li.name)+"? Its file is removed; your progress on its words is kept.",
		func(ok bool) {
			if !ok {
				return
			}
			if err := ld.gs.DeleteList(li.name); err != nil {
				ld.f.listsChanged(ld.gs.settingsState(), ld.f.wordSource())
				dialog.ShowError(err, ld.win)
				return
			}
			ld.f.listDeleted(ld.gs.settingsState(), li.name)
		}, ld.win)
	c.SetConfirmText("Delete")
	c.SetDismissText("Cancel")
	c.Show()
}

// rename asks for a list's new file name, checked while typing, then renames
// it and refreshes the settings (and so this dialog). Must run on the Fyne
// thread.
func (ld *listsDialog) rename(li listInfo) {
	entry := widget.NewEntry()
	entry.SetText(li.fileBase())
	entry.Validator = func(s string) error { return ld.gs.CheckListName(li.name, s) }
	item := widget.NewFormItem("Name", entry)
	item.HintText = "The file keeps its extension (" + filepath.Ext(li.path) + ")."
	d := dialog.NewForm("Rename "+prettyListName(li.name), "Rename", "Cancel", []*widget.FormItem{item}, func(ok bool) {
		if !ok {
			return
		}
		to, err := ld.gs.RenameList(li.name, entry.Text)
		if err != nil {
			ld.f.listsChanged(ld.gs.settingsState(), ld.f.wordSource())
			dialog.ShowError(err, ld.win)
			return
		}
		ld.f.listRenamed(ld.gs.settingsState(), li.name, to)
	}, ld.win)
	d.Resize(fyne.NewSize(420, 0))
	d.Show()
}

// openURL opens a URL with the system's handler. Tests replace it.
var openURL = func(u *url.URL) error { return fyne.CurrentApp().OpenURL(u) }

// folderURL is the file URL of a folder: file:///home/… or, on Windows,
// file:///C:/….
func folderURL(dir string) *url.URL {
	p := filepath.ToSlash(dir)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return &url.URL{Scheme: "file", Path: p}
}

// openWordsFolder opens the word list folder in the system file manager,
// creating it first if needed; if that fails it shows the path to copy.
// Must run on the Fyne thread.
func openWordsFolder(dir string, win fyne.Window) {
	if err := os.MkdirAll(dir, wordsDirPerm); err == nil {
		if err := openURL(folderURL(dir)); err == nil {
			return
		}
	}
	showFolderPath(dir, win)
}

// showFolderPath shows the word list folder's path, selectable for copying.
func showFolderPath(dir string, win fyne.Window) {
	path := widget.NewLabel(dir)
	path.Selectable = true
	d := dialog.NewCustom("Word list folder", "OK", container.NewVBox(
		widget.NewLabel("The folder could not be opened. Open it in your file manager:"),
		path,
	), win)
	d.Show()
}
```

In `fyne/settings.go`, `showSettingsDialog`, after `form.onImport = …`:

```go
	form.onManage = func() { showListsDialog(gs, form, win) }
```

- [ ] **Step 4: Run** `go test ./fyne/ -timeout 300s` and `go vet ./...` → PASS. If `labelContains` does not find the confirmation text (a `ConfirmDialog` may render its message as a `widget.Label` with wrapping — it should), fall back to checking a shorter substring such as "Delete mine?".

---

### Task 8: Drag & drop onto the window

**Files:** Create `fyne/drop.go`, `fyne/drop_test.go`. Modify `fyne/game.go` (field after `settingsOpen` ~line 119, `settle` ~1224, `SetSettingsOpen` neighbourhood ~1265, `buildSnapshot` ~1434), `fyne/app.go` (`buildWindow`).

- [ ] **Step 1: Failing tests** (`fyne/drop_test.go`):

```go
package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
)

func TestImportOpenPausesLikeTheSettings(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("あ", "a", 10)}
	gs.SetSettingsOpen(true)
	gs.SetImportOpen(true)
	if !gs.paused || len(tileObjects(gs)) != 0 {
		t.Fatal("an import preview pauses and hides the tiles")
	}
	gs.SetImportOpen(false)
	if !gs.paused {
		t.Fatal("closing the preview must not release the settings' pause")
	}
	gs.SetSettingsOpen(false)
	if gs.paused {
		t.Fatal("with nothing open the game goes on")
	}
}

// dropGame is a running window setup for drops: a fresh game with the vowels
// selected, a test window and a drop importer; src is a folder for source
// files outside the word list folder.
func dropGame(t *testing.T) (gs *GameState, d *dropImporter, w fyne.Window, src string) {
	t.Helper()
	test.NewApp()
	t.Cleanup(func() { test.NewApp() })
	openSettings, dialogShowing = nil, false
	t.Cleanup(func() { openSettings, dialogShowing = nil, false })
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs = NewGameState(st)
	w = test.NewWindow(nil)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(1000, 760))
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)
	d = &dropImporter{gs: gs, statsPanel: statsPanel, gameCanvas: gameCanvas, inputBar: ib, win: w}
	return gs, d, w, t.TempDir()
}

func TestDropShowsPreviewAndPauses(t *testing.T) {
	gs, d, w, src := dropGame(t)
	a := writeList(t, src, "a.csv", "kana;meaning\nあお;blue\n")
	d.drop([]fyne.URI{storage.NewFileURI(a)})
	if !gs.importOpen || !gs.paused {
		t.Fatal("the game must pause while the preview shows")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if !equalIDs(gs.listNames, []string{"a"}) {
		t.Fatalf("lists = %v", gs.listNames)
	}
	if gs.importOpen || gs.paused || w.Canvas().Overlays().Top() != nil {
		t.Fatalf("after the last file: importOpen %v, paused %v", gs.importOpen, gs.paused)
	}
}

func TestDropSeveralFilesInTurn(t *testing.T) {
	gs, d, w, src := dropGame(t)
	a := writeList(t, src, "a.csv", "kana;meaning\nあお;blue\n")
	b := writeList(t, src, "b.csv", "kana;meaning\nいえ;house\n")
	d.drop([]fyne.URI{storage.NewFileURI(a), storage.NewFileURI(b)})
	if !labelContains(w.Canvas().Overlays().Top(), "Word list: a") {
		t.Fatal("first the preview of a")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if !labelContains(w.Canvas().Overlays().Top(), "Word list: b") || !gs.importOpen {
		t.Fatal("then the preview of b, still paused")
	}
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Cancel"))
	if !equalIDs(gs.listNames, []string{"a"}) || gs.importOpen {
		t.Fatalf("lists %v, importOpen %v", gs.listNames, gs.importOpen)
	}
}

func TestDropFolderShowsTheImportError(t *testing.T) {
	gs, d, w, src := dropGame(t)
	folder := filepath.Join(src, "lists.csv")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	d.drop([]fyne.URI{storage.NewFileURI(folder)})
	top := w.Canvas().Overlays().Top()
	if top == nil || !labelContains(top, "This is a folder.") {
		t.Fatal("a folder must get the import's error")
	}
	test.Tap(buttonIn(top, "OK"))
	if gs.importOpen || len(gs.listNames) != 0 {
		t.Fatalf("importOpen %v, lists %v", gs.importOpen, gs.listNames)
	}
}

func TestDropOnOpenSettingsShowsTheNewList(t *testing.T) {
	gs, d, w, src := dropGame(t)
	showSettingsDialog(gs, d.statsPanel, d.gameCanvas, d.inputBar, w)
	settings := w.Canvas().Overlays().Top()
	a := writeList(t, src, "a.csv", "kana;meaning\nあお;blue\n")
	d.drop([]fyne.URI{storage.NewFileURI(a)})
	test.Tap(buttonIn(w.Canvas().Overlays().Top(), "Import"))
	if w.Canvas().Overlays().Top() != settings || !gs.settingsOpen || !gs.paused {
		t.Fatal("the settings must be there again, the game still paused")
	}
	if openSettings == nil || !slices.Contains(openSettings.sourceSelect.Options, "a") || openSettings.wordSource() != listSourcePrefix+"a" {
		t.Fatal("the open settings must show and preselect the new list")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestImportOpen|TestDrop' -timeout 300s` → build failure.

- [ ] **Step 3: Implement the pause.** `fyne/game.go`, after `settingsOpen`:

```go
	importOpen   bool // a dropped file's import preview is open
```

`settle`:

```go
// settle enforces the pause invariant: the game is paused only while a dialog
// is pending, the learner has paused, the settings dialog or a dropped
// file's import is open. It is the only place that unpauses. Must be called
// under lock.
func (gs *GameState) settle() {
	if gs.paused && !gs.dialogPending() && !gs.userPaused && !gs.settingsOpen && !gs.importOpen {
		gs.paused = false
	}
}
```

After `SetSettingsOpen`:

```go
// SetImportOpen pauses while dropped files are imported and releases that
// pause afterwards; another reason (the settings) may still hold it. Reset
// leaves it alone: the preview is modal, nothing restarts the round under
// it.
func (gs *GameState) SetImportOpen(open bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.importOpen = open
	if open {
		gs.paused = true
	} else {
		gs.settle()
	}
	gs.buildSnapshot()
}
```

`buildSnapshot`: `hide := gs.userPaused || gs.settingsOpen || gs.importOpen` and extend its comment: "While the learner has paused, or the settings dialog or a dropped file's import is open, nothing is published …".

- [ ] **Step 4: Implement** `fyne/drop.go`:

```go
package main

import (
	"io"
	"os"

	"fyne.io/fyne/v2"
	"kana/vocab"
)

// errFolder is why a dropped folder cannot be imported.
var errFolder = &vocab.ImportError{Reason: "This is a folder.", Hint: "Drop the word list files themselves."}

// dropImporter imports files dropped on the game window, one import preview
// after another, with the game paused as for the settings. Its fields are
// Fyne-thread-only.
type dropImporter struct {
	gs         *GameState
	statsPanel *StatsPanel
	gameCanvas *GameCanvas
	inputBar   *InputBar
	win        fyne.Window

	queue []fyne.URI // dropped files waiting for their preview
	busy  bool       // a preview or its error is showing
}

// drop queues dropped files and, unless a preview is showing already,
// pauses the game and shows the first. Must run on the Fyne thread.
func (d *dropImporter) drop(uris []fyne.URI) {
	if len(uris) == 0 {
		return
	}
	d.queue = append(d.queue, uris...)
	if d.busy {
		return
	}
	d.busy = true
	d.gs.SetImportOpen(true)
	d.gameCanvas.Refresh()
	d.next()
}

// next shows the next queued file's import preview, or why it cannot be
// imported, on top of any open dialog; after the last file it finishes. An
// open settings dialog shows and preselects an imported list. Must run on
// the Fyne thread.
func (d *dropImporter) next() {
	if len(d.queue) == 0 {
		d.finish()
		return
	}
	uri := d.queue[0]
	d.queue = d.queue[1:]
	data, err := readDropped(uri)
	if err != nil {
		showImportError(err, d.win, d.next)
		return
	}
	importChosen(d.gs, openSettings, uri.Name(), data, d.win, d.next)
}

// finish releases the import's pause and refreshes the status panel. A
// dialog that was open underneath is there again and goes on itself when
// it closes; with none, the game goes on. Must run on the Fyne thread.
func (d *dropImporter) finish() {
	d.busy = false
	d.gs.SetImportOpen(false)
	d.gameCanvas.Refresh()
	d.gs.mu.Lock()
	snap := d.gs.snapshot()
	d.gs.mu.Unlock()
	d.statsPanel.Update(snap)
	if openSettings == nil && !dialogShowing && !d.gs.IsOver() {
		showPendingDialogs(d.gs, d.statsPanel, d.inputBar, d.win)
	}
}

// readDropped reads a dropped file for import, at most one byte more than
// vocab.MaxImportSize so that Inspect rejects a larger one. A folder gives
// errFolder.
func readDropped(uri fyne.URI) ([]byte, error) {
	path := uri.Path()
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, errFolder
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return io.ReadAll(io.LimitReader(fh, vocab.MaxImportSize+1))
}
```

`fyne/app.go`, `buildWindow`, after `w.Canvas().SetOnTypedKey(…)`:

```go
	// A file dropped anywhere on the window opens the import preview.
	// GLFW delivers drops on the main thread; fyne.Do keeps that true for
	// any driver.
	drops := &dropImporter{gs: gs, statsPanel: statsPanel, gameCanvas: gameCanvas, inputBar: inputBar, win: w}
	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		fyne.Do(func() { drops.drop(uris) })
	})
```

- [ ] **Step 5: Run** `go test ./fyne/ -timeout 300s` → PASS. If a preview's `then` runs before Fyne has removed the closed dialog's overlay, so that the next preview lands *under* it in the test, wrap the call in `next()` as `fyne.Do(d.next)` in `importChosen`'s and `showImportError`'s `then` — but first check that the tests above see the expected `Top()`; the intro dialogs already chain from `SetOnClosed` the same way.

---

### Task 9: README

**Files:** Modify `README.md` (section "Private word lists", ~line 390; structure sections ~lines 280 and 330).

- [ ] **Step 1:** In "Private word lists", first paragraph: after "…the Words tab shows where)." add "You can also drop files anywhere on the game window: each gets the same preview, one after another, and the game pauses meanwhile." Replace "Files can also be copied into the folder by hand; they are read when the app starts." with "Files can also be copied into the folder or edited there by hand; they are read each time the settings open."

- [ ] **Step 2:** After that paragraph add:

  **Settings → Words → Manage word lists…** lists every word list with its number of words and lessons. A list with rows the game could not read says "N rows skipped"; **Details** shows the lines and why (at most ten), so you can correct your source file and import it again. **Rename** renames the list's file (the extension stays); its lesson choice and its place as the source of new words go with it. **Delete** removes the file for good after asking; your progress on its words is kept, and if it was the source of new words, JLPT takes its place. Both act at once, not on Save. **Open folder** opens the words folder in your file manager. The status panel's words line adds "· N rows skipped" when the chosen list has skipped rows.

- [ ] **Step 3:** Replace "renaming a list file forgets its lesson choice." with "renaming a list with Manage word lists keeps its lesson choice; renaming its file by hand forgets it."

- [ ] **Step 4:** Structure: in the `vocab/` list, after the `import.go` line add "- `manage.go`: `DeleteList`, `CheckListName` and `RenameList` for the word list dialog". In the `fyne/` list (near `settings.go`) add "- `lists.go`, `listsdialog.go`: the word lists in the game (reload, delete, rename) and the Manage word lists dialog" and "- `drop.go`: importing files dropped on the window". In the `store/store.go` line nothing changes (`lesson_filter:<list>` is listed already).

---

### Task 10: Final check (controller)

- [ ] `go test ./... -race -timeout 600s`, `go vet ./...`, `gofmt -l .` (empty).
- [ ] `git status --short`: no file from `~/.local/share/kana/words/` and no non-synthetic word list anywhere in the tree; `grep -rn "minna-no-nihongo" --include=*_test.go .` shows only the existing synthetic names.
- [ ] Manual: `go run ./fyne/` (with a throw-away list in the words folder, not a real one).
  - Copy a list into the words folder by hand, open the settings: it is in the source select.
  - Settings → Words → Manage word lists…: rows with counts; a list with a bad row shows "1 row skipped" and Details; the status panel's words line ends in "· 1 row skipped" while that list is the source.
  - Rename a list to `jmdict`, to another list's name, to a name with `/` (Rename disabled, reason shown); rename it in case only and to a new name: file renamed, extension kept, the source select and lessons follow; Cancel the settings afterwards — the rename stays.
  - Delete the chosen list: confirmation, file gone, source JLPT, introduced words of the list no longer fall unless JLPT has them.
  - Open folder: the file manager opens the folder (Linux, macOS, Windows if available).
  - Drop one file, then three files, then a folder onto the game window: previews in turn, the game paused and the tiles hidden; afterwards the game goes on. Drop a file while the settings (and the word list dialog) are open: the new list shows and is preselected, the settings stay open. Drop during a kana intro and on the game-over dialog: the dialog is there again afterwards.
