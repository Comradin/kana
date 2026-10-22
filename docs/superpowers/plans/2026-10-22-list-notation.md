# Word List Notation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Word list rows with a placeholder (`～さん`), optional parts (`（どうぞ）よろしく（おねがいします）`), punctuation (`おなまえは？`) or variants (`あめりか // アメリカ`) become playable words, shown as written and answered by their kana; a check command lists every skipped row of a file.

**Architecture:** First, on its own, `cmd/listcheck` reads files through a new `vocab.InspectAll` (Inspect without the message limit; the limit itself rises from 10 to 100). Then a pure `vocab.parseNotation` turns a kana field into a reading, a shown form and other accepted readings, or a reason to skip the row. The CSV and JSON readers use it, `vocab.Word`/`vocab.Entry` carry `Display` and `Alts`, the index merges them, and `Entry.ReadingMatches` accepts any of the readings. In `fyne` every place that shows a word uses `Entry.Shown()`, answers go through `Entry.ReadingMatches`, and long runs of skipped-row messages scroll.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (`container.NewVScroll`). No new dependencies.

Spec: `docs/superpowers/specs/2026-10-22-list-notation-design.md`. Terms: `GLOSSARY.md` (word list, source file, skipped row, reading, placeholder, optional part, variant). The UI never says "problem"; the field `vocab.List.Problems` keeps its name.

**Conventions:** comments like the surrounding code (short doc comment on every function, "Must be called under lock." / "Must run on the Fyne thread." where it applies). Every `GameState` field is read and written under `gs.mu`; the `fyne` tests touch fields directly without the lock, as the existing tests do. Tests: `go test ./... -timeout 300s`. **Test data is synthetic only** (invented rows such as `～さん`, `（どうぞ）よろしく`, `あめりか // アメリカ`, `ろうそく // ローソク`); every `fyne` test that reaches the word list folder calls `isolateWordsDir(t)` first, and nothing may read anything under `~/.local/share/kana/` (private, copyrighted lists live there). Every task ends with `gofmt -l .` (empty) and `go vet ./...` (clean). Commit steps name the message; commits carry no Co-Authored-By line and do not mention AI. Implementers may leave the commit to the controller, who reviews and commits with the given message.

**Task 1 is implemented now; Tasks 2–10 wait for the go-ahead.** Task 1 is complete on its own: nothing in it depends on the notation.

---

## Decisions made in the plan

The spec leaves these open or calls them indicative; check them before Task 2.

1. **Task order.** The check command comes first (Task 1), with the vocab change it needs: `maxProblems` 10 → 100 and `InspectAll`. Raising the limit in Task 1 means the Details dialog and the import preview can show up to 100 message lines without a scroll area until Task 7. That only lasts on this branch; if Task 1 is released alone, move Task 7 forward.
2. **How the limit is lifted:** `vocab.List` gets an unexported `limit` (0 = `maxProblems`, `allProblems` = −1 = every message). `readCSV` and `readJSON` take the limit as a third parameter; `loadFile` and `Inspect` pass `maxProblems`, the new exported `InspectAll(fileName, data)` passes `allProblems`. A zero `List{}` (an open error in `loadFile`, test literals) still keeps 100.
3. **Check command output** (exact; tests pin it):
   ```
   <path>: word list "<name>", 3 words, 2 rows skipped
   Words:                      (only with -words)
     せんせい · sensei · lesson 1
   Skipped rows:               (only if there are messages)
     line 4: abc is not kana
   ```
   A file that cannot be imported prints `<path>: cannot import: <reason>` and the hint indented on the next line; a file that cannot be read prints `<path>: cannot read: <os error>`. Files are separated by a blank line. Exit status 0, 1 when a file cannot be read or imported (the other files are still checked), 2 for a usage error (no file, unknown flag); `-h` exits 0. The file is read with the import's size limit (`MaxImportSize+1` bytes), so a too-large file gets the import's message. When a file cannot be imported, its skipped rows are not listed (`Inspect` returns no candidate; the reason quotes the first message, as in the game).
4. **`cmd/listcheck` is testable through `run(args, stdout, stderr) int`** and `checkFile(w, path, words) bool`; `main` only calls `os.Exit(run(…))`.
5. **README in Task 1 (deviation):** Task 1 already adds a short README paragraph on the check command, the architecture line and "(at most 100)" instead of "(at most ten)", so that Task 1 ships documented. Task 9 adds the notation and extends the paragraph for `-words`.
6. **The parser** is `parseNotation(field string) (notation, string)` in a new `vocab/notation.go`: a pure function (no I/O) that also runs the kana check (`kanacore.KanaUnits`) on every accepted reading. The second result is the skip reason, which the readers prefix with `line N: ` / `entry N: `. `notation` is unexported; the readers copy it into `Word`.
7. **Brackets:** either closing bracket closes either opening one (`（どうぞ)よろしく` is fine: learners mix IME and ASCII brackets; with no nesting there is no ambiguity). A second opening bracket before a close is "nested" and gives the same message as unpaired: `… has unpaired brackets`.
8. **A variant that is all brackets** (`（～さん）`): its short form is empty and does not count; only the full form is accepted.
9. **An empty variant** (`あめりか //`, `// アメリカ`) skips the row with `… has an empty variant` (a message the spec does not name; "empty kana" would be misleading for a field with text). A field that is empty as a whole after the notation stays "empty kana".
10. **Order of other readings (`Alts`):** variants in field order, for each its full form then its short form, without the reading itself, unique. `（どうぞ）よろしく // よろしく` has the reading `よろしく` and the other reading `どうぞよろしく`.
11. **`Display` is "" when the shown form equals the reading** (`あめりか // アメリカ` has `Display` ""). `Word.Shown()` and `Entry.Shown()` return `Display`, else `Kana`; all display sites use `Shown()`.
12. **Shown form across sources:** the first private-list row of a reading decides, in list order and row order, even when it is plain: `さん` in list `a` and `～さん` in list `b` shows `さん`. The index detects "first list row" as "the entry has no `ListPos` yet". Bundled words never set it.
13. **Reading match lives in `vocab`** as `Entry.ReadingMatches(input)`: it tries `kanacore.ReadingMatches` on `Kana`, then on each of `Alts`. `kanacore` stays a per-word romaji matcher that knows nothing of entries, lists or alternative readings; the set of accepted readings is word list data, so it belongs where `Entry` is.
14. **Missed words** become `[]vocab.Entry` (in `GameState.missedWords` and `StatsSnapshot.MissedWords`), so the game-over list can show `～さん (san)`; they are still unique by reading and sorted by reading.
15. **Scrolling skipped rows:** one helper `linesBody(lines)` builds both the Details dialog and the import preview from their line lists; a run of more than 8 lines starting with `• ` goes into a vertical scroll area 200 px high, the lines around it (counts, hint, first words) stay in view. `importPreviewLines` and `skippedRowsLines` keep returning plain strings (their tests stay).
16. **`-words` after the notation:** `<shown> · <romaji of the reading> · also <alt> (<romaji>), … · lesson N`. Other readings are printed in kana with their romaji, because two variants often share the romaji (`あめりか`/`アメリカ` are both `amerika`).
17. **Particles (user decision, option a).** Superseded the earlier "は is not read as wa": a word whose field uses a placeholder, an optional part, punctuation or a space is a *phrase* (`notation.phrase`, `Word.Phrase`, `Entry.Phrase`, OR-merged across sources); in a phrase the hiragana units は and へ also take `wa` and `e`. Variants alone do not make a phrase. を takes `o` as an alternative spelling everywhere (kana row entry), canonical romaji stays `wo`. See Task 6b. Tests written in Tasks 2–6 that expect `kochirahasandesu` stay valid (both are accepted).
18. **GLOSSARY** already has all terms (spec commit); no GLOSSARY task. `CLAUDE.md` is not in the repository and is not touched (you may want to add `cmd/listcheck/` to its structure list).
19. **The bundled list and `cmd/vocabgen` are unchanged**; bundled entries have no `Alts` and no `Display`.
20. In a phrase every は/へ, not only particles, takes wa/e (the game cannot tell particles apart); as the reading key ignores punctuation, a phrase row can make a bundled word with the same reading lenient too. Accepted.
21. The miss romaji stays the reading's (`ha`): showing `wa` would also turn はじめまして into wajimemashite.
22. **A short form that is no word on its own** (`いっ（て）`, `（コ）ーヒー`, `（き）ゃ`: っ, ー or a small kana at its edge) does not skip the row: the full form is checked first, and only a full form that fails `kanacore.KanaUnits` gives "… is not kana". A short form that fails is dropped as a reading, like an empty short form, and only the full form counts.

---

## File structure

- Modify `vocab/vocab.go`: `List.limit`, `maxProblems` 100, `allProblems`; `Word.Display`, `Word.Alts`, `Word.Shown`.
- Modify `vocab/csvlist.go`, `vocab/jsonlist.go`: limit parameter; `parseNotation` instead of the kana checks.
- Modify `vocab/dir.go`: `loadFile` passes `maxProblems`.
- Modify `vocab/import.go`: `inspect` with a limit, `Inspect`, `InspectAll`.
- Create `vocab/notation.go`, `vocab/notation_test.go`: `notation`, `parseNotation`, `variantForms`.
- Modify `vocab/index.go`: `Entry.Display`, `Entry.Alts`, merging in `add`, `clone`, `Entry.Shown`, `Entry.ReadingMatches`.
- Modify tests `vocab/csvlist_test.go`, `vocab/dir_test.go`, `vocab/import_test.go`, `vocab/index_test.go`, `vocab/vocab_test.go`.
- Create `cmd/listcheck/main.go`, `cmd/listcheck/main_test.go`.
- Modify `fyne/tile.go` (`newWordTile`), `fyne/words.go` (`hitFeedbackText`, `missFeedbackText`, `missedWordsText`), `fyne/app.go` (`missedLines`), `fyne/stats.go` (`StatsSnapshot.MissedWords`), `fyne/game.go` (`missedWords`, `tick`, `checkAnswer`, `answers`, `buildSnapshot`), `fyne/packageintro.go` (`addRow`), `fyne/importlist.go` (`importPreviewLines`, `importChosen`), `fyne/listsdialog.go` (`skippedRowBullet`, `linesBody`, `wrappedLabel`, `skippedRowsLines`, `showSkippedRows`), `fyne/lists.go` (comment).
- Modify tests `fyne/tile_test.go`, `fyne/words_test.go`, `fyne/app_test.go`, `fyne/wordgame_test.go`, `fyne/packageintro_test.go`, `fyne/importlist_test.go`, `fyne/listsdialog_test.go`.
- Modify `README.md`.

---

### Task 1: The check command `cmd/listcheck`

**Files:**
- Modify: `vocab/vocab.go` (`List`, `maxProblems`), `vocab/csvlist.go` (`readCSV`, `problem`), `vocab/jsonlist.go` (`readJSON`), `vocab/dir.go` (`loadFile`), `vocab/import.go` (`Inspect`), `fyne/lists.go` (comment), `README.md`
- Create: `cmd/listcheck/main.go`, `cmd/listcheck/main_test.go`
- Test: `vocab/csvlist_test.go`, `vocab/dir_test.go`, `vocab/import_test.go`

- [ ] **Step 1: Failing vocab tests.** In `vocab/csvlist_test.go`, `TestReadCSVProblems`, replace the block from `var b strings.Builder` to the `problems=%d skipped=%d` check with:

```go
	var b strings.Builder
	b.WriteString("kana\n")
	for i := 0; i < 150; i++ {
		b.WriteString("x\n")
	}
	l = readCSV("x", strings.NewReader(b.String()), maxProblems)
	if len(l.Problems) != maxProblems || l.Skipped != 150 {
		t.Fatalf("problems=%d skipped=%d", len(l.Problems), l.Skipped)
	}
	l = readCSV("x", strings.NewReader(b.String()), allProblems)
	if len(l.Problems) != 150 || l.Skipped != 150 {
		t.Fatalf("all problems: problems=%d skipped=%d", len(l.Problems), l.Skipped)
	}
```

Then append `, maxProblems` as the third argument to every other `readCSV(` call in `vocab/csvlist_test.go` and every `readJSON(` call in `vocab/dir_test.go` (13 + 4 calls; the compiler lists any you miss). Add to `vocab/dir_test.go`:

```go
func TestReadJSONProblemLimit(t *testing.T) {
	in := "[" + strings.TrimSuffix(strings.Repeat(`{"kana":"x"},`, 150), ",") + "]"
	if l := readJSON("x", strings.NewReader(in), maxProblems); len(l.Problems) != maxProblems || l.Skipped != 150 {
		t.Fatalf("problems=%d skipped=%d", len(l.Problems), l.Skipped)
	}
	if l := readJSON("x", strings.NewReader(in), allProblems); len(l.Problems) != 150 {
		t.Fatalf("all problems: problems=%d", len(l.Problems))
	}
}
```

Add to `vocab/import_test.go`:

```go
func TestInspectAllKeepsEveryMessage(t *testing.T) {
	data := []byte("kana;meaning\nあお;blue\n" + strings.Repeat("x;y\n", 150))
	c, err := Inspect("many.csv", data)
	if err != nil || len(c.List.Problems) != 100 || c.List.Skipped != 150 {
		t.Fatalf("Inspect: %d messages, %d skipped, err %v; want the first 100 of 150", len(c.List.Problems), c.List.Skipped, err)
	}
	all, err := InspectAll("many.csv", data)
	if err != nil || len(all.List.Problems) != 150 || all.List.Skipped != 150 {
		t.Fatalf("InspectAll: %d messages, %d skipped, err %v; want all 150", len(all.List.Problems), all.List.Skipped, err)
	}
	if got := all.List.Problems[149]; got != "line 152: x is not kana" {
		t.Fatalf("last message %q", got)
	}
	if all.Name != "many" || len(all.List.Words) != 1 {
		t.Fatalf("candidate %+v", all)
	}
}
```

- [ ] **Step 2: Run** `go test ./vocab/ -timeout 300s` → build failure: `too many arguments in call to readCSV`, `undefined: allProblems`, `undefined: InspectAll`.

- [ ] **Step 3: Implement the limit.** In `vocab/vocab.go` replace `List` and `maxProblems`:

```go
// List is one private word list file.
type List struct {
	Name     string
	Path     string // the list's file; "" for a list not read from a folder
	Words    []Word
	Problems []string // messages in file order, at most the list's limit
	Skipped  int      // rows skipped, including those without a message

	limit int // messages kept: 0 = maxProblems, allProblems = every one
}

// BundledSource is the Source of bundled words and a reserved list name.
const BundledSource = "jmdict"

// maxProblems is how many messages a list keeps unless it is read with
// allProblems (see InspectAll).
const maxProblems = 100

// allProblems is the message limit that keeps every message.
const allProblems = -1
```

In `vocab/csvlist.go` change the reader's signature, first lines and doc comment:

```go
// readCSV reads a word list with a header row. The separator (tab,
// semicolon or comma) is taken from the header line. limit is how many
// messages the list keeps (see List).
func readCSV(name string, r io.Reader, limit int) List {
	l := List{Name: name, limit: limit}
```

and replace `problem`:

```go
// problem records a message without counting a skipped row; beyond the
// list's limit it is dropped.
func (l *List) problem(msg string) {
	limit := l.limit
	if limit == 0 {
		limit = maxProblems
	}
	if limit < 0 || len(l.Problems) < limit {
		l.Problems = append(l.Problems, msg)
	}
}
```

In `vocab/jsonlist.go`:

```go
// readJSON reads a word list (an optional UTF-8 BOM is skipped): an array of objects with kana, meaning
// (string, array of strings, or an object of language → string or array),
// lesson and jlpt (number or string). limit is how many messages the list
// keeps (see List).
func readJSON(name string, r io.Reader, limit int) List {
	l := List{Name: name, limit: limit}
```

In `vocab/dir.go`, `loadFile`: `l = readJSON(name, fh, maxProblems)` and `l = readCSV(name, fh, maxProblems)`.

In `vocab/import.go`, rename `func Inspect(fileName string, data []byte) (Candidate, error)` to `func inspect(fileName string, data []byte, limit int) (Candidate, error)`, pass `limit` to its `readJSON`/`readCSV` calls, and put in front of it:

```go
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
```

(The old doc comment of `Inspect` above `inspect` is replaced by the line above.)

In `fyne/lists.go`, `listInfo.problems`: change the comment to `// messages on skipped rows, at most vocab's 100`.

- [ ] **Step 4: Run** `go test ./vocab/ ./fyne/ -timeout 300s` → PASS.

- [ ] **Step 5: Commit** `vocab/` and `fyne/lists.go`:

```bash
git add vocab/vocab.go vocab/csvlist.go vocab/jsonlist.go vocab/dir.go vocab/import.go vocab/csvlist_test.go vocab/dir_test.go vocab/import_test.go fyne/lists.go
git commit -m "feat(vocab): keep 100 skipped-row messages, all with InspectAll"
```

- [ ] **Step 6: Failing command tests.** Create `cmd/listcheck/main_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes a synthetic word list into a fresh temp dir and returns
// its path.
func writeFile(t *testing.T, name, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckGoodFile(t *testing.T) {
	path := writeFile(t, "good.csv", "kana;meaning;lesson\nせんせい;teacher;1\nがくせい;student;1\n")
	var out bytes.Buffer
	if !checkFile(&out, path, false) {
		t.Fatalf("a good file must pass:\n%s", out.String())
	}
	if want := path + ": word list \"good\", 2 words, 0 rows skipped\n"; out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
}

func TestCheckListsEverySkippedRow(t *testing.T) {
	path := writeFile(t, "many.csv", "kana;meaning\nあお;blue\n"+strings.Repeat("x;y\n", 150))
	var out bytes.Buffer
	if !checkFile(&out, path, false) {
		t.Fatalf("a file with words must pass:\n%s", out.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if want := path + ": word list \"many\", 1 word, 150 rows skipped"; lines[0] != want {
		t.Fatalf("first line %q, want %q", lines[0], want)
	}
	if len(lines) != 152 || lines[1] != "Skipped rows:" {
		t.Fatalf("%d lines, second %q; want a heading and all 150 messages", len(lines), lines[1])
	}
	if lines[2] != "  line 3: x is not kana" || lines[151] != "  line 152: x is not kana" {
		t.Fatalf("messages %q … %q", lines[2], lines[151])
	}
}

func TestCheckFileThatCannotBeImported(t *testing.T) {
	path := writeFile(t, "sheet.xlsx", "PK")
	var out bytes.Buffer
	if checkFile(&out, path, false) {
		t.Fatal("a spreadsheet cannot be imported")
	}
	want := path + ": cannot import: This is a spreadsheet file.\n" +
		"  Save the sheet as \"CSV UTF-8\" in your spreadsheet program and import the .csv file.\n"
	if out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}

	missing := filepath.Join(t.TempDir(), "gone.csv")
	out.Reset()
	if checkFile(&out, missing, false) || !strings.HasPrefix(out.String(), missing+": cannot read: ") {
		t.Fatalf("missing file: %q", out.String())
	}
}

func TestCheckWords(t *testing.T) {
	path := writeFile(t, "words.csv", "kana;meaning;lesson\nせんせい;teacher;1\nえき;station;\n")
	var out bytes.Buffer
	checkFile(&out, path, true)
	want := path + ": word list \"words\", 2 words, 0 rows skipped\n" +
		"Words:\n  せんせい · sensei · lesson 1\n  えき · eki\n"
	if out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
}

func TestRunExitStatus(t *testing.T) {
	good := writeFile(t, "good.csv", "kana\nえき\n")
	bad := writeFile(t, "bad.json", "[]")
	var out, errOut bytes.Buffer
	if got := run([]string{good}, &out, &errOut); got != 0 {
		t.Fatalf("good file: status %d", got)
	}
	out.Reset()
	if got := run([]string{bad, good}, &out, &errOut); got != 1 {
		t.Fatalf("one bad file: status %d, want 1", got)
	}
	if s := out.String(); !strings.Contains(s, bad+": cannot import: ") || !strings.Contains(s, "\n\n"+good+": word list") {
		t.Fatalf("both files must be reported, separated by a blank line:\n%s", s)
	}
	out.Reset()
	if got := run([]string{"-words", good}, &out, &errOut); got != 0 || !strings.Contains(out.String(), "  えき · eki\n") {
		t.Fatalf("-words: status %d, output %q", got, out.String())
	}
	if got := run(nil, &out, &errOut); got != 2 || !strings.Contains(errOut.String(), "usage: listcheck [-words] FILE...") {
		t.Fatalf("no file: status %d, stderr %q", got, errOut.String())
	}
	if got := run([]string{"-nope", good}, &out, &errOut); got != 2 {
		t.Fatalf("unknown flag: status %d", got)
	}
}
```

- [ ] **Step 7: Run** `go test ./cmd/listcheck/ -timeout 300s` → build failure: `undefined: checkFile`, `undefined: run`.

- [ ] **Step 8: Implement.** Create `cmd/listcheck/main.go`:

```go
// Command listcheck checks word list files the way the game's import does,
// without importing them, and prints every skipped row.
//
//	go run ./cmd/listcheck [-words] FILE...
//
// For each file it prints the word list's name with its number of words and
// skipped rows, with -words every word, then every skipped row's message
// (all of them; the game keeps the first 100), or why the file cannot be
// imported and what to do about it. The exit status is 1 when a file cannot
// be read or imported, 2 for a usage error.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"kana/kanacore"
	"kana/vocab"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run checks the files named in args, reporting to stdout, and returns the
// exit status. Usage errors go to stderr.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("listcheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	words := flags.Bool("words", false, "also print every word with its romaji and lesson")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: listcheck [-words] FILE...")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() == 0 {
		flags.Usage()
		return 2
	}
	status := 0
	for i, path := range flags.Args() {
		if i > 0 {
			fmt.Fprintln(stdout)
		}
		if !checkFile(stdout, path, *words) {
			status = 1
		}
	}
	return status
}

// checkFile checks one file as the import does and reports on it to w: the
// word list's name and counts, with words every word, then every message.
// It returns false when the file cannot be read or imported.
func checkFile(w io.Writer, path string, words bool) bool {
	data, err := readFile(path)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err // the path is printed already
		}
		fmt.Fprintf(w, "%s: cannot read: %v\n", path, err)
		return false
	}
	c, err := vocab.InspectAll(path, data)
	if err != nil {
		fmt.Fprintf(w, "%s: cannot import: %v\n", path, err)
		var ie *vocab.ImportError
		if errors.As(err, &ie) && ie.Hint != "" {
			fmt.Fprintf(w, "  %s\n", ie.Hint)
		}
		return false
	}
	l := c.List
	fmt.Fprintf(w, "%s: word list %q, %s, %s skipped\n", path, c.Name,
		plural(len(l.Words), "word"), plural(l.Skipped, "row"))
	if words {
		fmt.Fprintln(w, "Words:")
		for _, word := range l.Words {
			fmt.Fprintf(w, "  %s\n", wordLine(word))
		}
	}
	if len(l.Problems) > 0 {
		fmt.Fprintln(w, "Skipped rows:")
		for _, p := range l.Problems {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
	return true
}

// readFile reads at most one byte more than an import accepts, so that
// Inspect can tell a file is too large.
func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, vocab.MaxImportSize+1))
}

// wordLine describes a word for -words: "せんせい · sensei · lesson 1".
func wordLine(w vocab.Word) string {
	parts := []string{w.Kana, kanacore.Romaji(w.Kana)}
	if w.Lesson != "" {
		parts = append(parts, "lesson "+w.Lesson)
	}
	return strings.Join(parts, " · ")
}

// plural is "1 word" or "4 words".
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
```

- [ ] **Step 9: Run** `go test ./cmd/listcheck/ ./vocab/ -timeout 300s` → PASS. Try it by hand on a synthetic file only (never on a file under `~/.local/share/kana/`):

```bash
printf 'kana;meaning;lesson\nせんせい;teacher;1\n先生;teacher;1\n' > "$TMPDIR/try.csv"
go run ./cmd/listcheck -words "$TMPDIR/try.csv"; echo "status $?"
```

Expected: `…/try.csv: word list "try", 1 word, 1 row skipped`, `Words:`, `  せんせい · sensei · lesson 1`, `Skipped rows:`, `  line 3: 先生 is not kana`, `status 0`.

- [ ] **Step 10: README.** In "Private word lists", in the "Manage word lists…" paragraph, replace "(at most ten)" with "(at most 100)". After that paragraph add:

  **Checking a file without importing it:** in a clone of this repository, `go run ./cmd/listcheck FILE...` reads each file as the import does and changes nothing. For each file it prints the word list's name, its number of words and skipped rows, then the message of every skipped row (all of them, where the game keeps the first 100), or why the file cannot be imported and what to do about it. With `-words` it also lists every word with its romaji and lesson. The output can be passed on as it is, for example to whoever keeps the list; the exit status is 1 when a file cannot be imported.

  In "Architecture", after the "Word Data Generator (`cmd/vocabgen/`)" section add:

  ### List Checker (`cmd/listcheck/`)

  A command that checks word list files as the import does, without importing them, and prints every skipped row (see [Private word lists](#private-word-lists)).

  In the `vocab/` list, change the `import.go` line to: "- `import.go`: `Inspect` and `Import`, checking and storing a word list for the settings' import; `InspectAll` keeps every skipped row's message for the check command".

- [ ] **Step 11: Check** `gofmt -l .` (empty), `go vet ./...` (clean), `go test ./... -timeout 300s` (PASS).

- [ ] **Step 12: Commit**

```bash
git add cmd/listcheck/main.go cmd/listcheck/main_test.go README.md
git commit -m "feat(listcheck): add a command that lists every skipped row of a list"
```

**Stop here until the go-ahead for Tasks 2–10.**

---

### Task 2: `vocab` — the notation parser

**Files:**
- Create: `vocab/notation.go`, `vocab/notation_test.go`

- [ ] **Step 1: Failing tests.** Create `vocab/notation_test.go`:

```go
package vocab

import (
	"reflect"
	"testing"
)

func TestParseNotation(t *testing.T) {
	cases := []struct {
		field string
		want  notation
	}{
		{"せんせい", notation{reading: "せんせい"}},
		// placeholders
		{"～さん", notation{reading: "さん", display: "～さん"}},
		{"〜さん", notation{reading: "さん", display: "〜さん"}},
		{"~さん", notation{reading: "さん", display: "~さん"}},
		{"こちらは～さんです", notation{reading: "こちらはさんです", display: "こちらは～さんです"}},
		// punctuation and spaces
		{"おなまえは？", notation{reading: "おなまえは", display: "おなまえは？"}},
		{"はい、そうです。", notation{reading: "はいそうです", display: "はい、そうです。"}},
		{"どうぞ よろしく", notation{reading: "どうぞよろしく", display: "どうぞ よろしく"}},
		{"どうぞ　よろしく", notation{reading: "どうぞよろしく", display: "どうぞ　よろしく"}},
		// optional parts: the full and the short form, nothing in between
		{"（どうぞ）よろしく（おねがいします）", notation{reading: "どうぞよろしくおねがいします",
			display: "（どうぞ）よろしく（おねがいします）", alts: []string{"よろしく"}}},
		{"(どうぞ)よろしく", notation{reading: "どうぞよろしく", display: "(どうぞ)よろしく", alts: []string{"よろしく"}}},
		{"（どうぞ)よろしく", notation{reading: "どうぞよろしく", display: "（どうぞ)よろしく", alts: []string{"よろしく"}}},
		// a placeholder with an optional part; a placeholder in brackets
		{"（お）～さん", notation{reading: "おさん", display: "（お）～さん", alts: []string{"さん"}}},
		{"（～さん）", notation{reading: "さん", display: "（～さん）"}},
		// variants: the last one is shown and is the reading
		{"あめりか // アメリカ", notation{reading: "アメリカ", alts: []string{"あめりか"}}},
		{"あめりか//アメリカ", notation{reading: "アメリカ", alts: []string{"あめりか"}}},
		{"ろうそく // ローソク", notation{reading: "ローソク", alts: []string{"ろうそく"}}},
		{"～くん // ～クン", notation{reading: "クン", display: "～クン", alts: []string{"くん"}}},
		{"（どうぞ）よろしく // よろしく", notation{reading: "よろしく", alts: []string{"どうぞよろしく"}}},
		{"ねこ // ねこ", notation{reading: "ねこ"}},
	}
	for _, c := range cases {
		got, reason := parseNotation(c.field)
		if reason != "" || !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseNotation(%q) = %+v, %q; want %+v", c.field, got, reason, c.want)
		}
	}
}

func TestParseNotationSkips(t *testing.T) {
	cases := map[string]string{
		"":              "empty kana",
		"～":             "empty kana",
		"？ 。":           "empty kana",
		"（）":            "empty kana",
		"（どうぞよろしく":      "（どうぞよろしく has unpaired brackets",
		"どうぞ）よろしく":      "どうぞ）よろしく has unpaired brackets",
		"（どう（ぞ））":       "（どう（ぞ）） has unpaired brackets",
		"先生":            "先生 is not kana",
		"sensei":        "sensei is not kana",
		"せんせい // 先生":    "せんせい // 先生 is not kana",
		"先生 // せんせい":    "先生 // せんせい is not kana",
		"ん（い）ー":         "ん（い）ー is not kana", // the short form んー cannot be typed
		"あめりか //":       "あめりか // has an empty variant",
		"// アメリカ":       "// アメリカ has an empty variant",
	}
	for field, want := range cases {
		if got, reason := parseNotation(field); reason != want {
			t.Errorf("parseNotation(%q) = %+v, %q; want reason %q", field, got, reason, want)
		}
	}
}
```

(`gofmt -w vocab/notation_test.go` aligns the map.)

- [ ] **Step 2: Run** `go test ./vocab/ -run TestParseNotation -timeout 300s` → build failure: `undefined: notation`, `undefined: parseNotation`.

- [ ] **Step 3: Implement.** Create `vocab/notation.go`:

```go
package vocab

import (
	"strings"
	"unicode"

	"kana/kanacore"
)

// variantSep separates the variants of a word in a kana field.
const variantSep = "//"

// notation is what a word list's kana field says: the reading, how the word
// is shown, and the other readings that count as answers.
type notation struct {
	reading string   // the last variant's full form
	display string   // the last variant as written; "" when it is the reading
	alts    []string // the other variants' and forms' readings, unique, in field order
}

// isPlaceholder reports whether r stands for something the learner fills
// in: shown, not typed.
func isPlaceholder(r rune) bool { return r == '～' || r == '〜' || r == '~' }

// isShownOnly reports whether r is punctuation or a space: shown, not typed.
func isShownOnly(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune("？！。、・?!.,", r)
}

// isOpenBracket and isCloseBracket report the brackets of an optional part;
// either closing bracket closes either opening one.
func isOpenBracket(r rune) bool  { return r == '（' || r == '(' }
func isCloseBracket(r rune) bool { return r == '）' || r == ')' }

// parseNotation reads a trimmed kana field: placeholders, punctuation and
// spaces are shown but not typed, text in brackets is optional (the full and
// the short form count), and "//" separates variants, of which the last is
// shown and gives the reading. It returns the notation, or why the row is
// skipped: "empty kana", or the field as written followed by "is not kana",
// "has unpaired brackets" or "has an empty variant". Every accepted reading
// must pass kanacore.KanaUnits.
func parseNotation(field string) (notation, string) {
	var n notation
	var readings []string
	variants := strings.Split(field, variantSep)
	for i, v := range variants {
		v = strings.TrimSpace(v)
		full, short, ok := variantForms(v)
		if !ok {
			return notation{}, field + " has unpaired brackets"
		}
		if full == "" {
			if len(variants) == 1 {
				return notation{}, "empty kana"
			}
			return notation{}, field + " has an empty variant"
		}
		for _, r := range []string{full, short} {
			if r == "" {
				continue // all of the variant is optional: only the full form counts
			}
			if _, ok := kanacore.KanaUnits(r); !ok {
				return notation{}, field + " is not kana"
			}
			if !contains(readings, r) {
				readings = append(readings, r)
			}
		}
		if i == len(variants)-1 {
			n.reading = full
			if v != full {
				n.display = v
			}
		}
	}
	for _, r := range readings {
		if r != n.reading {
			n.alts = append(n.alts, r)
		}
	}
	return n, ""
}

// variantForms returns a variant's full form (bracket characters removed,
// their contents kept) and its short form (bracketed parts removed), both
// without placeholders, punctuation and spaces. It reports false for an
// unpaired or nested bracket.
func variantForms(v string) (full, short string, ok bool) {
	var f, s strings.Builder
	open := false
	for _, r := range v {
		switch {
		case isOpenBracket(r):
			if open {
				return "", "", false
			}
			open = true
		case isCloseBracket(r):
			if !open {
				return "", "", false
			}
			open = false
		case isPlaceholder(r) || isShownOnly(r):
		default:
			f.WriteRune(r)
			if !open {
				s.WriteRune(r)
			}
		}
	}
	if open {
		return "", "", false
	}
	return f.String(), s.String(), true
}
```

(`contains` is the existing helper in `vocab/index.go`.)

- [ ] **Step 4: Run** `go test ./vocab/ -run TestParseNotation -timeout 300s` → PASS; then `gofmt -l .`, `go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add vocab/notation.go vocab/notation_test.go
git commit -m "feat(vocab): parse placeholders, optional parts and variants"
```

---

### Task 3: `vocab` — the readers use the notation

**Files:**
- Modify: `vocab/vocab.go` (`Word`), `vocab/csvlist.go` (row loop, imports), `vocab/jsonlist.go` (entry loop, imports)
- Test: `vocab/vocab_test.go`, `vocab/csvlist_test.go`, `vocab/dir_test.go`, `vocab/import_test.go`

- [ ] **Step 1: Failing tests.** Add to `vocab/csvlist_test.go`:

```go
func TestReadCSVNotation(t *testing.T) {
	in := "kana;meaning;lesson\n" +
		"～さん;Mr./Ms.;1\n" +
		"（どうぞ）よろしく（おねがいします）;nice to meet you;1\n" +
		"あめりか // アメリカ;America;2\n" +
		"（どうぞ;x;2\n" +
		"～;x;2\n" +
		"あめりか // 米国;x;2\n"
	l := readCSV("course", strings.NewReader(in), maxProblems)
	want := []Word{
		{Kana: "さん", Display: "～さん", Meanings: Meanings{"": {"Mr./Ms."}}, Source: "course", Lesson: "1"},
		{Kana: "どうぞよろしくおねがいします", Display: "（どうぞ）よろしく（おねがいします）", Alts: []string{"よろしく"},
			Meanings: Meanings{"": {"nice to meet you"}}, Source: "course", Lesson: "1"},
		{Kana: "アメリカ", Alts: []string{"あめりか"}, Meanings: Meanings{"": {"America"}}, Source: "course", Lesson: "2"},
	}
	if !reflect.DeepEqual(l.Words, want) {
		t.Fatalf("words:\n%+v\nwant\n%+v", l.Words, want)
	}
	wantProblems := []string{
		"line 5: （どうぞ has unpaired brackets",
		"line 6: empty kana",
		"line 7: あめりか // 米国 is not kana",
	}
	if l.Skipped != 3 || !reflect.DeepEqual(l.Problems, wantProblems) {
		t.Fatalf("skipped %d, problems %q; want %q", l.Skipped, l.Problems, wantProblems)
	}
}
```

Add to `vocab/dir_test.go`:

```go
func TestReadJSONNotation(t *testing.T) {
	in := `[{"kana":"～さん","meaning":"Mr./Ms."},{"kana":"(どうぞ)よろしく","meaning":"x"},{"kana":"（どう（ぞ））","meaning":"x"}]`
	l := readJSON("x", strings.NewReader(in), maxProblems)
	if len(l.Words) != 2 || l.Words[0].Kana != "さん" || l.Words[0].Display != "～さん" {
		t.Fatalf("words = %+v", l.Words)
	}
	if w := l.Words[1]; w.Kana != "どうぞよろしく" || !reflect.DeepEqual(w.Alts, []string{"よろしく"}) {
		t.Fatalf("optional part = %+v", w)
	}
	if want := []string{"entry 3: （どう（ぞ）） has unpaired brackets"}; !reflect.DeepEqual(l.Problems, want) {
		t.Fatalf("problems %q, want %q", l.Problems, want)
	}
}
```

Add to `vocab/import_test.go`:

```go
func TestInspectReadsNotation(t *testing.T) {
	c, err := Inspect("course.csv", []byte("kana;meaning\n～さん;Mr./Ms.\nあめりか // アメリカ;America\n"))
	if err != nil || len(c.List.Words) != 2 || c.List.Skipped != 0 {
		t.Fatalf("c %+v, err %v; rows in notation are words, not skipped rows", c.List, err)
	}
}
```

Add to `vocab/vocab_test.go`:

```go
func TestWordShown(t *testing.T) {
	if got := (Word{Kana: "さん"}).Shown(); got != "さん" {
		t.Fatalf("plain word shown as %q", got)
	}
	if got := (Word{Kana: "さん", Display: "～さん"}).Shown(); got != "～さん" {
		t.Fatalf("word with notation shown as %q", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./vocab/ -timeout 300s` → build failure: `unknown field Display in struct literal`, `w.Shown undefined`.

- [ ] **Step 3: Implement.** In `vocab/vocab.go` replace `Word` and add `Shown`:

```go
// Word is one entry of a word list.
type Word struct {
	Kana     string   // reading, e.g. "せんせい"
	Display  string   // the shown form as the list writes it, e.g. "～さん"; "" = Kana
	Alts     []string // other accepted readings, e.g. an optional part left out; private lists only
	Meanings Meanings // language → alternatives
	JLPT     int      // 5…1, 0 = none
	Rank     int      // bundled words: frequency rank, lower = more common; 0 = unknown
	Source   string   // "jmdict" or a list name
	Lesson   string   // optional, private lists only
}

// Shown returns how the word is shown to the learner: its Display, else its
// reading.
func (w Word) Shown() string {
	if w.Display != "" {
		return w.Display
	}
	return w.Kana
}
```

In `vocab/csvlist.go`, in the row loop, replace everything from `if kana == "" {` up to and including the `w := Word{…}` statement with:

```go
		n, reason := parseNotation(kana)
		if reason != "" {
			l.skip(fmt.Sprintf("line %d: %s", line, reason))
			continue
		}
		w := Word{Kana: n.reading, Display: n.display, Alts: n.alts, Meanings: Meanings{}, Source: name,
			Lesson: field(rec, "lesson"), JLPT: parseLevel(field(rec, "jlpt"))}
```

and remove the now unused `"kana/kanacore"` import.

In `vocab/jsonlist.go`, in the entry loop, replace everything from `kana := strings.TrimSpace(jw.Kana)` up to and including the `w := Word{…}` statement with:

```go
		n, reason := parseNotation(strings.TrimSpace(jw.Kana))
		if reason != "" {
			l.skip(fmt.Sprintf("entry %d: %s", i+1, reason))
			continue
		}
		w := Word{Kana: n.reading, Display: n.display, Alts: n.alts, Meanings: Meanings{}, Source: name,
			Lesson: scalar(jw.Lesson), JLPT: parseLevel(scalar(jw.JLPT))}
```

and remove the now unused `"kana/kanacore"` import. Update the doc comment of `readJSON`'s first sentence to end "…, lesson and jlpt (number or string); the kana field may use the notation (see parseNotation)." and add the same remark to `readCSV`'s doc comment: "The kana column may use the notation (see parseNotation)."

- [ ] **Step 4: Run** `go test ./vocab/ -timeout 300s` → PASS (the existing tests still hold: `先生` gives "line 2: 先生 is not kana", an empty field "empty kana", and `???` is now punctuation, so the "question marks" import still says "The file contains no kana."). Then `gofmt -l .`, `go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add vocab/vocab.go vocab/csvlist.go vocab/jsonlist.go vocab/vocab_test.go vocab/csvlist_test.go vocab/dir_test.go vocab/import_test.go
git commit -m "feat(vocab): read the kana notation in CSV and JSON word lists"
```

---

### Task 4: `vocab` — entries merge shown forms and readings

**Files:**
- Modify: `vocab/index.go` (`Entry`, `add`, `clone`, `Entries`/`Lookup` comments, new `Shown`, `ReadingMatches`)
- Test: `vocab/index_test.go`

- [ ] **Step 1: Failing tests.** Add to `vocab/index_test.go`:

```go
func TestIndexShownForm(t *testing.T) {
	bundled := []Word{word("さん", 5, BundledSource, "", "en", "Mr./Ms.")}
	if e, _ := NewIndex(bundled, nil).Lookup("さん"); e.Shown() != "さん" {
		t.Fatalf("a bundled word alone is shown as %q, want its reading", e.Shown())
	}
	san := Word{Kana: "さん", Display: "～さん", Meanings: Meanings{}}
	plain := Word{Kana: "さん", Meanings: Meanings{}}
	cases := []struct {
		name  string
		lists []List
		want  string
	}{
		{"a list over the bundled word", []List{{Name: "a", Words: []Word{san}}}, "～さん"},
		{"the first list wins", []List{{Name: "a", Words: []Word{san}}, {Name: "b", Words: []Word{plain}}}, "～さん"},
		{"a plain first list wins too", []List{{Name: "a", Words: []Word{plain}}, {Name: "b", Words: []Word{san}}}, "さん"},
		{"the first row of a list wins", []List{{Name: "a", Words: []Word{plain, san}}}, "さん"},
	}
	for _, c := range cases {
		for how, ix := range map[string]*Index{
			"NewIndex":  NewIndex(bundled, c.lists),
			"WithLists": NewIndex(bundled, nil).WithLists(c.lists),
		} {
			e, ok := ix.Lookup("さん")
			if !ok || !e.Bundled || e.Shown() != c.want {
				t.Errorf("%s, %s: shown %q, bundled %v; want %q, one entry", c.name, how, e.Shown(), e.Bundled, c.want)
			}
		}
	}
}

func TestIndexMergesAlts(t *testing.T) {
	a := List{Name: "a", Words: []Word{{Kana: "アメリカ", Alts: []string{"あめりか"}, Meanings: Meanings{}}}}
	b := List{Name: "b", Words: []Word{{Kana: "アメリカ", Alts: []string{"めりけん", "あめりか"}, Meanings: Meanings{}}}}
	want := []string{"あめりか", "めりけん"}
	if e, _ := NewIndex(nil, []List{a, b}).Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, want) {
		t.Fatalf("NewIndex alts = %v, want %v", e.Alts, want)
	}
	base := NewIndex(nil, []List{a})
	if e, _ := base.WithLists([]List{b}).Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, want) {
		t.Fatalf("WithLists alts = %v, want %v", e.Alts, want)
	}
	if e, _ := base.Lookup("アメリカ"); !reflect.DeepEqual(e.Alts, []string{"あめりか"}) {
		t.Fatalf("base alts = %v; WithLists must not change the base index", e.Alts)
	}
}

func TestEntryReadingMatches(t *testing.T) {
	e := Entry{Kana: "どうぞよろしくおねがいします", Alts: []string{"よろしく"}}
	for in, want := range map[string]bool{
		"douzoyoroshikuonegaishimasu": true,
		"yoroshiku":                   true,
		"douzoyoroshiku":              false, // no mix of the full and the short form
		"douzo yoroshiku":             false,
		"":                            false,
	} {
		if got := e.ReadingMatches(in); got != want {
			t.Errorf("ReadingMatches(%q) = %v, want %v", in, got, want)
		}
	}
	v := Entry{Kana: "ローソク", Alts: []string{"ろうそく"}}
	if !v.ReadingMatches("roosoku") || !v.ReadingMatches("rousoku") {
		t.Fatal("every variant's reading must match")
	}
}
```

- [ ] **Step 2: Run** `go test ./vocab/ -run 'TestIndexShownForm|TestIndexMergesAlts|TestEntryReadingMatches' -timeout 300s` → build failure: `e.Shown undefined`, `e.Alts undefined`, `e.ReadingMatches undefined`.

- [ ] **Step 3: Implement.** In `vocab/index.go` replace the `Entry` type:

```go
// Entry is one reading with everything known about it across sources.
type Entry struct {
	Kana     string
	Display  string   // shown form from the first private list row of this reading; "" = Kana
	Alts     []string // other accepted readings across sources, unique, in first-seen order
	Units    []string // kanacore.KanaUnits of Kana, computed once
	Meanings Meanings
	JLPT     int                 // easiest bundled level, 0 if not bundled
	Rank     int                 // best bundled frequency rank, lower = more common; 0 = unknown
	Bundled  bool                // the bundled list contains it
	Lists    map[string][]string // private list name → lessons
	ListPos  map[string]int      // private list name → index of its first row in that list
}

// Shown returns how the entry is shown to the learner: its Display, else
// its reading.
func (e Entry) Shown() string {
	if e.Display != "" {
		return e.Display
	}
	return e.Kana
}

// ReadingMatches reports whether input is a romaji reading of the entry's
// reading or of one of its other accepted readings (see
// kanacore.ReadingMatches).
func (e Entry) ReadingMatches(input string) bool {
	if kanacore.ReadingMatches(e.Kana, input) {
		return true
	}
	for _, a := range e.Alts {
		if kanacore.ReadingMatches(a, input) {
			return true
		}
	}
	return false
}
```

In `add`, replace `e.Meanings.Merge(w.Meanings)` with:

```go
	e.Meanings.Merge(w.Meanings)
	for _, a := range w.Alts {
		if !contains(e.Alts, a) {
			e.Alts = append(e.Alts, a)
		}
	}
```

and, after the bundled branch's closing brace (before `if _, ok := e.ListPos[list]; !ok {`), insert:

```go
	// The first private list row of this reading decides how it is shown,
	// even when it has no notation.
	if len(e.ListPos) == 0 {
		e.Display = w.Display
	}
```

Update `add`'s doc comment: append "The first private list row gives the shown form; other readings merge like meanings."

Replace `clone`:

```go
// clone copies the entry's Meanings, Alts, Lists and ListPos, so merging
// into the copy leaves the original untouched. Units is never modified and
// stays shared.
func (e Entry) clone() Entry {
	m := make(Meanings, len(e.Meanings))
	for lang, items := range e.Meanings {
		m[lang] = slices.Clone(items)
	}
	l := make(map[string][]string, len(e.Lists))
	for name, lessons := range e.Lists {
		l[name] = slices.Clone(lessons)
	}
	e.Meanings, e.Lists, e.ListPos, e.Alts = m, l, maps.Clone(e.ListPos), slices.Clone(e.Alts)
	return e
}
```

In the doc comments of `Entries` and `Lookup`, change "Units, Meanings and Lists" to "Units, Alts, Meanings and Lists".

- [ ] **Step 4: Run** `go test ./vocab/ -timeout 300s` → PASS (including `TestWithListsMatchesNewIndex`); then `gofmt -l .`, `go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add vocab/index.go vocab/index_test.go
git commit -m "feat(vocab): merge shown forms and accepted readings per entry"
```

---

### Task 5: `fyne` — words are shown as their list writes them

**Files:**
- Modify: `fyne/tile.go` (`newWordTile`), `fyne/words.go` (`missedWordsText`, `hitFeedbackText`, `missFeedbackText`), `fyne/app.go` (`missedLines`, imports), `fyne/stats.go` (`StatsSnapshot.MissedWords`, imports), `fyne/game.go` (`missedWords`, `tick`, `buildSnapshot`), `fyne/packageintro.go` (`addRow`), `fyne/importlist.go` (`importPreviewLines`)
- Test: `fyne/tile_test.go`, `fyne/words_test.go`, `fyne/app_test.go`, `fyne/wordgame_test.go`, `fyne/packageintro_test.go`, `fyne/importlist_test.go`

- [ ] **Step 1: Failing tests.**

`fyne/tile_test.go`:

```go
func TestWordTileShowsTheShownForm(t *testing.T) {
	test.NewApp()
	tile := newWordTile(vocab.Entry{Kana: "さん", Display: "～さん"}, 400, fontChoice{})
	if tile.text.Text != "～さん" || tile.word.Kana != "さん" {
		t.Fatalf("text %q, word %q; want ～さん on the tile, さん as its reading", tile.text.Text, tile.word.Kana)
	}
}
```

`fyne/words_test.go`: replace `TestMissedWordsText` and extend `TestFeedbackTexts`:

```go
func TestMissedWordsText(t *testing.T) {
	if got := missedWordsText(nil); got != "" {
		t.Fatalf("no words = %q", got)
	}
	var words []vocab.Entry
	for _, k := range []string{"せんせい", "きって", "せんせい", "コーヒー", "まっちゃ"} {
		words = append(words, vocab.Entry{Kana: k})
	}
	want := "きって (kitte), せんせい (sensei), まっちゃ (matcha)\nコーヒー (koohii)"
	if got := missedWordsText(words); got != want {
		t.Fatalf("missedWordsText = %q, want %q", got, want)
	}
	san := vocab.Entry{Kana: "さん", Display: "～さん"}
	if got := missedWordsText([]vocab.Entry{san, san}); got != "～さん (san)" {
		t.Fatalf("shown form = %q, want ～さん (san)", got)
	}
}
```

At the end of `TestFeedbackTexts` add:

```go
	san := vocab.Entry{Kana: "さん", Display: "～さん", Meanings: vocab.Meanings{"en": {"Mr./Ms."}}}
	if got := hitFeedbackText(san, "en", false); got != "～さん — Mr./Ms." {
		t.Errorf("hit with notation = %q", got)
	}
	if got := hitFeedbackText(san, "en", true); got != "～さん · san — Mr./Ms." {
		t.Errorf("hit by meaning with notation = %q", got)
	}
	if got := missFeedbackText(san, "en"); got != "～さん · san — Mr./Ms." {
		t.Errorf("miss with notation = %q", got)
	}
```

`fyne/app_test.go`: add `"kana/vocab"` to the imports; in `TestMissedLines` change the `words` field to `[]vocab.Entry`, the cases to `[]vocab.Entry{{Kana: "あい"}}` where they had `[]string{"あい"}`, and add the case:

```go
		{"shown form", nil, []vocab.Entry{{Kana: "さん", Display: "～さん"}}, []string{"Words missed:", "～さん (san)"}},
```

`fyne/wordgame_test.go`: add the helper

```go
// readingsOf returns the entries' readings.
func readingsOf(es []vocab.Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Kana)
	}
	return out
}
```

In `TestWordMiss` change `equalIDs(gs.missedWords, []string{"あい"})` to `equalIDs(readingsOf(gs.missedWords), []string{"あい"})`. In `TestSnapshotCarriesMissedWordsAndResetClears` use `gs.missedWords = []vocab.Entry{{Kana: "あい"}}` and `equalIDs(readingsOf(snap.MissedWords), []string{"あい"})`. Add:

```go
func TestPlaceholderWordIsShownAsWritten(t *testing.T) {
	san := vocab.Word{Kana: "さん", Display: "～さん", Source: "course", Meanings: vocab.Meanings{"en": {"Mr./Ms."}}}

	gs := newTestState()
	withList(gs, "course", san)
	tile := wordTileAt(gs, "さん", 0)
	if tile.text.Text != "～さん" {
		t.Fatalf("tile shows %q, want ～さん", tile.text.Text)
	}
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("san")
	if len(gs.tiles) != 0 || gs.feedback[0].text.Text != "～さん — Mr./Ms." {
		t.Fatalf("tiles %d, feedback %q; want ～さん answered by san", len(gs.tiles), gs.feedback[0].text.Text)
	}

	gs = newTestState()
	gs.canvasH = 600
	withList(gs, "course", san)
	gs.tiles = []*KanaTile{wordTileAt(gs, "さん", 599)}
	gs.tick()
	if got := gs.feedback[0].text.Text; got != "～さん · san — Mr./Ms." {
		t.Fatalf("miss feedback %q", got)
	}
	if got := missedWordsText(gs.snapshot().MissedWords); got != "～さん (san)" {
		t.Fatalf("game-over list %q, want the shown form", got)
	}
}
```

`fyne/packageintro_test.go`:

```go
func TestPackageViewShowsTheShownForm(t *testing.T) {
	test.NewApp()
	san := vocab.Entry{Kana: "さん", Display: "～さん", Meanings: vocab.Meanings{"en": {"Mr./Ms."}}}
	v := newPackageView([]vocab.Entry{san}, "en", fontChoice{}, func() (vocab.Entry, bool) { return vocab.Entry{}, false })
	if got := v.rows[0].kana.Text; got != "～さん" {
		t.Fatalf("kana column %q, want ～さん", got)
	}
	if !labelContains(v.list, "san") {
		t.Fatal("the romaji is that of the reading: san")
	}
	if got := v.known(); len(got) != 0 {
		t.Fatalf("known = %v", got)
	}
}
```

`fyne/importlist_test.go`:

```go
func TestImportPreviewShowsTheShownForm(t *testing.T) {
	c, err := vocab.Inspect("course.csv", []byte("kana;meaning\n～さん;Mr./Ms.\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Word list: course", "1 word", "First words:", "～さん · Mr./Ms."}
	if got := importPreviewLines(c, "en", nil); !equalIDs(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -timeout 300s` → build failure: `cannot use words (variable of type []vocab.Entry) as []string value in argument to missedWordsText` (and the same for `missedLines`, `gs.missedWords`).

- [ ] **Step 3: Implement.**

`fyne/tile.go`, `newWordTile`: `text := canvas.NewText(e.Shown(), kanaTextColor)`; doc comment: "…as wide as the word's shown form plus padding…".

`fyne/words.go`:

```go
// missedWordsText lists each missed word once, sorted by reading, in its
// shown form with the romaji of its reading, as "～さん (san)", three per
// line so the game-over dialog stays narrow. "" when no word was missed.
func missedWordsText(words []vocab.Entry) string {
	seen := map[string]bool{}
	var uniq []vocab.Entry
	for _, e := range words {
		if !seen[e.Kana] {
			seen[e.Kana] = true
			uniq = append(uniq, e)
		}
	}
	sort.Slice(uniq, func(i, j int) bool { return uniq[i].Kana < uniq[j].Kana })
	var lines []string
	for i := 0; i < len(uniq); i += 3 {
		end := min(i+3, len(uniq))
		parts := make([]string, 0, 3)
		for _, e := range uniq[i:end] {
			parts = append(parts, fmt.Sprintf("%s (%s)", e.Shown(), kanacore.Romaji(e.Kana)))
		}
		lines = append(lines, strings.Join(parts, ", "))
	}
	return strings.Join(lines, "\n")
}
```

```go
// hitFeedbackText is shown after a word hit, the word in its shown form:
// "はい — yes", or with the romaji of its reading, "はい · hai — yes", when
// the word was answered by its meaning.
func hitFeedbackText(e vocab.Entry, lang string, byMeaning bool) string {
	s := e.Shown()
	if byMeaning {
		s += " · " + kanacore.Romaji(e.Kana)
	}
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		s += " — " + m
	}
	return s
}

// missFeedbackText is shown after a missed word, with the solution: the
// shown form and the romaji of the reading, "～さん · san — Mr./Ms.".
func missFeedbackText(e vocab.Entry, lang string) string {
	s := e.Shown() + " · " + kanacore.Romaji(e.Kana)
	if m := meaningSnippet(e.Meanings, lang); m != "" {
		s += " — " + m
	}
	return s
}
```

`fyne/app.go`: add `"kana/vocab"` to the imports; `func missedLines(kanas []kanacore.Kana, words []vocab.Entry) []string` (body unchanged).

`fyne/stats.go`: add `"kana/vocab"` to the imports; `MissedWords   []vocab.Entry // missed words, in order`.

`fyne/game.go`: the field `missedWords  []vocab.Entry // missed words, in order`; in `tick` `gs.missedWords = append(gs.missedWords, *tile.word)`; in `buildSnapshot`'s `StatsSnapshot` literal `MissedWords:   append([]vocab.Entry(nil), gs.missedWords...),`. Run `gofmt -w fyne/game.go fyne/stats.go` (the field comments realign).

`fyne/packageintro.go`, `addRow`: `row.kana = canvas.NewText(e.Shown(), kanaTextColor)` (the romaji label keeps `kanacore.Romaji(e.Kana)`). Doc comment of `packageView`: "…one row per word with its shown form, the romaji of its reading, …".

`fyne/importlist.go`, `importPreviewLines`: `parts := []string{w.Shown()}`.

- [ ] **Step 4: Run** `go test ./fyne/ -timeout 300s` → PASS; then `gofmt -l .`, `go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/tile.go fyne/words.go fyne/app.go fyne/stats.go fyne/game.go fyne/packageintro.go fyne/importlist.go fyne/tile_test.go fyne/words_test.go fyne/app_test.go fyne/wordgame_test.go fyne/packageintro_test.go fyne/importlist_test.go
git commit -m "feat(fyne): show words as their word list writes them"
```

---

### Task 6: `fyne` — answers through every accepted reading

**Files:**
- Modify: `fyne/game.go` (`checkAnswer`, `answers`)
- Test: `fyne/wordgame_test.go`

- [ ] **Step 1: Failing tests.** Add to `fyne/wordgame_test.go`:

```go
func TestOptionalPartsTakeBothReadings(t *testing.T) {
	yoroshiku := vocab.Word{Kana: "どうぞよろしくおねがいします", Display: "（どうぞ）よろしく（おねがいします）",
		Alts: []string{"よろしく"}, Source: "course", Meanings: vocab.Meanings{"en": {"nice to meet you"}}}
	for _, input := range []string{"douzoyoroshikuonegaishimasu", "yoroshiku"} {
		gs := newTestState()
		gs.canvasW = 1000
		withList(gs, "course", yoroshiku)
		gs.tiles = []*KanaTile{wordTileAt(gs, yoroshiku.Kana, 0)}
		gs.checkAnswer(input)
		if len(gs.tiles) != 0 || gs.wordSessionStats[yoroshiku.Kana].CorrectCount != 1 {
			t.Fatalf("%q must answer the word", input)
		}
		if got := gs.feedback[0].text.Text; strings.Contains(got, "·") {
			t.Fatalf("%q: feedback %q; a reading is not an answer by meaning", input, got)
		}
	}
	gs := newTestState()
	withList(gs, "course", yoroshiku)
	gs.tiles = []*KanaTile{wordTileAt(gs, yoroshiku.Kana, 0)}
	gs.checkAnswer("douzoyoroshiku")
	if len(gs.tiles) != 1 {
		t.Fatal("a mix of the full and the short form is no reading")
	}
}

func TestVariantWordNeedsItsScriptAndTakesEveryReading(t *testing.T) {
	candle := vocab.Word{Kana: "ローソク", Alts: []string{"ろうそく"}, Source: "course", Meanings: vocab.Meanings{"en": {"candle"}}}
	gs := newTestState() // hiragana only
	withList(gs, "course", candle)
	learnKana(gs, "ろ", "う", "そ", "く", "ロ", "ソ", "ク")
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable without katakana = %v; the shown variant ローソク needs katakana", got)
	}
	gs.activeScripts[kanacore.ScriptKatakana] = true
	gs.invalidatePlayable()
	if got := playableKana(gs); !equalIDs(got, []string{"ローソク"}) {
		t.Fatalf("playable with katakana = %v", got)
	}
	for _, input := range []string{"roosoku", "rousoku"} { // the reading, the other variant's reading
		gs.tiles = []*KanaTile{wordTileAt(gs, "ローソク", 0)}
		gs.checkAnswer(input)
		if len(gs.tiles) != 0 {
			t.Fatalf("%q must answer ローソク", input)
		}
		if got := gs.feedback[len(gs.feedback)-1].text.Text; strings.Contains(got, "·") {
			t.Fatalf("%q: feedback %q; a variant's reading is not an answer by meaning", input, got)
		}
	}
}
```

(`strings` and `kanacore` are already imported in `fyne/wordgame_test.go`; add them if the compiler says otherwise.)

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestOptionalPartsTakeBothReadings|TestVariantWordNeedsItsScript' -timeout 300s` → FAIL: `"yoroshiku" must answer the word`, `"rousoku" must answer ローソク`.

- [ ] **Step 3: Implement.** In `fyne/game.go`, `checkAnswer`: `byMeaning := !tile.word.ReadingMatches(input)`; doc comment: "…a word tile by any valid reading of the whole word or of another accepted reading (an optional part left out, another variant), or by its meaning…". In `answers`:

```go
// answers reports whether input answers the tile: a word by one of its
// readings (see vocab.Entry.ReadingMatches) or its meaning. Must be called
// under lock.
func (gs *GameState) answers(tile *KanaTile, input string) bool {
	if tile.word != nil {
		return tile.word.ReadingMatches(input) || tile.word.Meanings.MeaningMatches(gs.meaningLang, input)
	}
	return gs.charSet.Matches(tile.kana.Char, input)
}
```

- [ ] **Step 4: Run** `go test ./fyne/ -timeout 300s` → PASS; then `gofmt -l .`, `go vet ./...` (if `kanacore` is now unused in `game.go`, the build says so; it is still used for `kanacore.Kana`).

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/wordgame_test.go
git commit -m "feat(fyne): accept every reading of a word with optional parts or variants"
```

---

### Task 6b: particles in phrases, を as `o`

**Files:**
- Modify: `kanacore/kana_rows.go` (を entry), `kanacore/words.go` (`ReadingMatches` gains a particle variant), `vocab/notation.go` (`phrase`), `vocab/vocab.go` (`Word.Phrase`), `vocab/csvlist.go`, `vocab/jsonlist.go` (copy it), `vocab/index.go` (`Entry.Phrase`, OR-merge in `add`, `ReadingMatches` passes it on)
- Test: `kanacore/words_test.go`, `kanacore/kana_test.go`, `vocab/notation_test.go`, `vocab/index_test.go`, `fyne/wordgame_test.go`

- [ ] **Step 1: Failing tests.**
  - kanacore: `PhraseReadingMatches("こちらはさんです", "kochirawasandesu")` and `"kochirahasandesu"` true; `PhraseReadingMatches("にほんへ", "nihone")` true; `ReadingMatches("はな", "wana")` false; `PhraseReadingMatches("ハナ", "wana")` false (katakana ハ/ヘ are not particles); `ReadingMatches("をかし", "okashi")` and `"wokashi"` true; the を kana-row test expects alternatives `[]string{"o"}`.
  - vocab: `parseNotation` sets `phrase` for placeholder, brackets, punctuation and space cases, not for `せんせい`, `あめりか // アメリカ` or `ねこ // ねこ`; a variant with a placeholder (`～くん // ～クン`) is a phrase. Update the expected `notation` values in `TestParseNotation` accordingly. Index: an entry is a phrase if any source row is; `Entry.ReadingMatches` accepts `wa` for は only when `Phrase`.
  - fyne: `こちらは～さんです` from a list is answered by `kochirawasandesu` (not by meaning).
- [ ] **Step 2: Run** the new tests and see them fail.
- [ ] **Step 3: Implement.**
  - `kana_rows.go`: `entry("を", "wo", "o")`.
  - `words.go`: rename the matcher body into `readingMatches(word, input string, particles bool) bool`; `ReadingMatches` calls it with false, new exported `PhraseReadingMatches` (doc: "…like ReadingMatches; は may also be wa and へ e, as particles in a phrase") with true. In the default case, when `particles` and `u == "は"` also try `"wa"`, when `u == "へ"` also try `"e"`.
  - `notation.go`: `phrase bool` in `notation`, true when the field (any variant) had a placeholder, a bracket, punctuation or a space. `vocab.go`: `Word.Phrase bool` with a doc comment. Readers copy it. `index.go`: `Entry.Phrase bool`; in `add`, `e.Phrase = e.Phrase || w.Phrase`; `ReadingMatches` uses `kanacore.PhraseReadingMatches` for every reading when `e.Phrase`.
- [ ] **Step 4: Run** `go test ./kanacore/ ./vocab/ ./fyne/ -timeout 600s`, `gofmt -l .`, `go vet ./...`.
- [ ] **Step 5: Commit** `feat: accept particle readings in phrases and o for を`

---

### Task 7: `fyne` — long runs of skipped rows scroll

**Files:**
- Modify: `fyne/listsdialog.go` (constants, `skippedRowsLines`, `showSkippedRows`, new `linesBody`, `wrappedLabel`), `fyne/importlist.go` (`importPreviewLines`, `importChosen`)
- Test: `fyne/listsdialog_test.go`, `fyne/importlist_test.go`

- [ ] **Step 1: Failing tests.** Add to `fyne/listsdialog_test.go` (imports: add `"fmt"`, `"fyne.io/fyne/v2/container"`):

```go
// scrollIn finds the first scroll area in a dialog's objects.
func scrollIn(o fyne.CanvasObject) *container.Scroll {
	for _, obj := range test.LaidOutObjects(o) {
		if s, ok := obj.(*container.Scroll); ok {
			return s
		}
	}
	return nil
}

// skippedLines returns n skipped-row lines as the dialogs show them.
func skippedLines(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("%sline %d: x is not kana", skippedRowBullet, i+2))
	}
	return out
}

func TestLinesBodyScrollsLongRunsOfSkippedRows(t *testing.T) {
	test.NewApp()
	lines := append(append([]string{"Word list: a", "9 rows skipped:"}, skippedLines(9)...), "First words:")
	body := linesBody(lines)
	if len(body.Objects) != 4 {
		t.Fatalf("%d objects, want two labels, a scroll area, a label", len(body.Objects))
	}
	scroll, ok := body.Objects[2].(*container.Scroll)
	if !ok || len(scroll.Content.(*fyne.Container).Objects) != 9 {
		t.Fatalf("object 2 is %T; want a scroll area with the 9 messages", body.Objects[2])
	}
	short := linesBody(append([]string{"8 rows skipped:"}, skippedLines(8)...))
	for _, o := range short.Objects {
		if _, ok := o.(*widget.Label); !ok {
			t.Fatalf("8 messages need no scroll area, got a %T", o)
		}
	}
}

func TestDetailsScrollWithManySkippedRows(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	li := listInfo{name: "many", words: 1, skipped: 150}
	for i := 0; i < 100; i++ {
		li.problems = append(li.problems, fmt.Sprintf("line %d: x is not kana", i+2))
	}
	showSkippedRows(li, w)
	top := w.Canvas().Overlays().Top()
	if scrollIn(top) == nil {
		t.Fatal("100 messages must scroll")
	}
	if !labelContains(top, skippedRowsHint) {
		t.Fatal("the hint stays below the scroll area")
	}
}
```

Add to `fyne/importlist_test.go`:

```go
func TestImportPreviewScrollsManySkippedRows(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	isolateWordsDir(t)
	gs := NewGameState(openTestStore(t))
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	data := "kana;meaning\nあお;blue\n" + strings.Repeat("x;y\n", 30)
	importChosen(gs, nil, "many.csv", []byte(data), w, nil)
	preview := w.Canvas().Overlays().Top()
	if preview == nil || scrollIn(preview) == nil {
		t.Fatal("30 skipped rows must scroll in the preview")
	}
	if !labelContains(preview, "30 rows skipped:") || !labelContains(preview, "First words:") {
		t.Fatal("the count and the first words stay in view")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestLinesBody|TestDetailsScroll|TestImportPreviewScrolls' -timeout 300s` → build failure: `undefined: skippedRowBullet`, `undefined: linesBody`.

- [ ] **Step 3: Implement.** In `fyne/listsdialog.go`, after `skippedRowsHint`:

```go
// skippedRowBullet starts each skipped row's message in the details and the
// import preview.
const skippedRowBullet = "• "

// A run of more skipped-row messages than skippedRowsScrollAfter scrolls in
// an area skippedRowsScrollH high, so a dialog with 100 of them stays on
// screen.
const (
	skippedRowsScrollAfter         = 8
	skippedRowsScrollH     float32 = 200
)
```

In `skippedRowsLines`: `lines = append(lines, skippedRowBullet+p)`. Add:

```go
// linesBody shows lines as wrapped labels. A run of more than
// skippedRowsScrollAfter skipped-row messages (lines starting with
// skippedRowBullet) sits in a scroll area of its own; the lines around it
// stay in view.
func linesBody(lines []string) *fyne.Container {
	body := container.NewVBox()
	for i := 0; i < len(lines); {
		end := i
		for end < len(lines) && strings.HasPrefix(lines[end], skippedRowBullet) {
			end++
		}
		if end-i > skippedRowsScrollAfter {
			rows := container.NewVBox()
			for _, line := range lines[i:end] {
				rows.Add(wrappedLabel(line))
			}
			scroll := container.NewVScroll(rows)
			scroll.SetMinSize(fyne.NewSize(0, skippedRowsScrollH))
			body.Add(scroll)
			i = end
			continue
		}
		end = max(end, i+1)
		for _, line := range lines[i:end] {
			body.Add(wrappedLabel(line))
		}
		i = end
	}
	return body
}

// wrappedLabel is a label that wraps at word boundaries.
func wrappedLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}
```

Replace `showSkippedRows`'s body construction:

```go
// showSkippedRows shows a list's skipped rows, scrolling when there are
// many, and how to fix the file. Must run on the Fyne thread.
func showSkippedRows(li listInfo, win fyne.Window) {
	d := dialog.NewCustom(prettyListName(li.name)+": "+plural(li.skipped, "row")+" skipped", "OK",
		linesBody(skippedRowsLines(li)), win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}
```

In `fyne/importlist.go`: in `importPreviewLines` use `skippedRowBullet+p` instead of `"• "+p`; in `importChosen` replace the `body := container.NewVBox()` loop with `body := linesBody(importPreviewLines(c, gs.MeaningLang(), files))`. Remove imports the compiler reports as unused (`container` in `importlist.go` if nothing else uses it).

- [ ] **Step 4: Run** `go test ./fyne/ -timeout 300s` → PASS (the existing Details and preview tests still find their labels); then `gofmt -l .`, `go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add fyne/listsdialog.go fyne/importlist.go fyne/listsdialog_test.go fyne/importlist_test.go
git commit -m "feat(fyne): scroll long lists of skipped rows"
```

---

### Task 8: `cmd/listcheck` — `-words` with shown forms and other readings

**Files:**
- Modify: `cmd/listcheck/main.go` (`wordLine`, the `-words` flag text)
- Test: `cmd/listcheck/main_test.go`

- [ ] **Step 1: Failing test.** Add to `cmd/listcheck/main_test.go`:

```go
func TestCheckWordsWithNotation(t *testing.T) {
	path := writeFile(t, "course.csv", "kana;meaning;lesson\n"+
		"～さん;Mr./Ms.;1\n"+
		"（どうぞ）よろしく（おねがいします）;nice to meet you;1\n"+
		"あめりか // アメリカ;America;2\n")
	var out bytes.Buffer
	checkFile(&out, path, true)
	want := path + ": word list \"course\", 3 words, 0 rows skipped\nWords:\n" +
		"  ～さん · san · lesson 1\n" +
		"  （どうぞ）よろしく（おねがいします） · douzoyoroshikuonegaishimasu · also よろしく (yoroshiku) · lesson 1\n" +
		"  アメリカ · amerika · also あめりか (amerika) · lesson 2\n"
	if out.String() != want {
		t.Fatalf("output:\n%s\nwant:\n%s", out.String(), want)
	}
}
```

- [ ] **Step 2: Run** `go test ./cmd/listcheck/ -run TestCheckWordsWithNotation -timeout 300s` → FAIL (`さん · san · lesson 1` without the placeholder, no "also").

- [ ] **Step 3: Implement.** Replace `wordLine`:

```go
// wordLine describes a word for -words: its shown form, the romaji of its
// reading, its other accepted readings and its lesson, as
// "（どうぞ）よろしく · douzoyoroshiku · also よろしく (yoroshiku) · lesson 1".
// Other readings are given in kana, since variants often share the romaji.
func wordLine(w vocab.Word) string {
	parts := []string{w.Shown(), kanacore.Romaji(w.Kana)}
	if len(w.Alts) > 0 {
		alts := make([]string, len(w.Alts))
		for i, a := range w.Alts {
			alts[i] = fmt.Sprintf("%s (%s)", a, kanacore.Romaji(a))
		}
		parts = append(parts, "also "+strings.Join(alts, ", "))
	}
	if w.Lesson != "" {
		parts = append(parts, "lesson "+w.Lesson)
	}
	return strings.Join(parts, " · ")
}
```

Change the flag text to `"also print every word: shown form, romaji, other readings, lesson"` and in the package comment "with -words every word" to "with -words every word (shown form, romaji, other accepted readings, lesson)".

- [ ] **Step 4: Run** `go test ./cmd/listcheck/ -timeout 300s` → PASS; then `gofmt -l .`, `go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add cmd/listcheck/main.go cmd/listcheck/main_test.go
git commit -m "feat(listcheck): print shown forms and other readings with -words"
```

---

### Task 9: README

**Files:** Modify `README.md` (section "Private word lists", ~line 393; "Word Lists (`vocab/`)" ~line 275).

- [ ] **Step 1:** In "Private word lists", after the CSV/JSON bullet list, replace "Rows whose reading is not kana are skipped and reported." with:

  **Notation in the kana column** (CSV and JSON alike), for course lists that write more than plain kana:

  - `～` (also `〜` or `~`) stands for something you fill in, such as a name: `～さん`, `こちらは～さんです`. The tile shows it; you type only the kana: `san`.
  - Text in brackets, `（…）` or `(…)`, is optional: `（どうぞ）よろしく（おねがいします）` is answered with the full form `douzoyoroshikuonegaishimasu` or the short form `yoroshiku`, nothing in between. Brackets must pair and must not nest.
  - Punctuation (`？ ！ 。 、 ・ ? ! . ,`) and spaces are shown, not typed: `おげんきですか？` is `ogenkidesuka`.
  - `//` separates two writings of one word: `あめりか // アメリカ`. The tile shows the last one, which also decides the script (`アメリカ` needs katakana switched on); every writing's reading counts.

  - In an entry that uses any of the above except `//` alone (a phrase), は may also be typed `wa` and へ `e`, as particles are spoken: `こちらは～さんです` takes `kochirawasandesu`. Everywhere, を takes `o` as well as `wo`.

  The tile, the feedback, the word package intro, the import preview and the game-over list show a word as the list writes it, with the romaji of its full form. Progress belongs to the reading, the kana without the notation: `～さん` in two lists, or `～さん` in a list and `さん` in the JLPT list, is one word, shown as the first list writes it.

  Rows are skipped and reported when their kana field is empty after the notation, contains anything other than kana (kanji, Latin letters; a variant that is not kana skips the whole row), or has unpaired or nested brackets.

- [ ] **Step 2:** In the check command paragraph (added in Task 1), replace "With `-words` it also lists every word with its romaji and lesson." with "With `-words` it also lists every word as shown, with the romaji of its reading, its other accepted readings (an optional part left out, another variant) and its lesson."

- [ ] **Step 3:** Architecture, `vocab/` list: after the `csvlist.go, jsonlist.go, dir.go` line add "- `notation.go`: the notation of a kana field (placeholders, optional parts, punctuation, variants): reading, shown form and other accepted readings, or why the row is skipped". Extend the `index.go` line after "(one entry per reading across all sources, with each list entry's first row" with ", its shown form and its other accepted readings; `Entry.ReadingMatches`".

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: describe the word list notation and the check command"
```

---

### Task 10: Final check (controller)

- [ ] `go test ./... -race -timeout 600s`, `go vet ./...`, `gofmt -l .` (empty).
- [ ] `git status --short`: no file from `~/.local/share/kana/` and no non-synthetic word list anywhere in the tree.
- [ ] `go run ./cmd/listcheck -words` on a throw-away file in a temp directory with the spec's rows (`～さん`, `（どうぞ）よろしく（おねがいします）`, `おなまえは？`, `あめりか // アメリカ`, one bad row `（どうぞ`): four words with shown forms and "also" readings, one skipped row, status 0; the same with a `.xlsx` name: the reason and hint, status 1.
- [ ] Manual: `go run ./fyne/` with that throw-away file imported (not a real list).
  - The import preview shows the first words as written; a file with 30 bad rows shows the messages in a scroll area with the count and first words in view.
  - Choose the list as the source: the package intro shows `～さん` with romaji `san`; the tile shows `～さん` and takes `san`; `（どうぞ）よろしく（おねがいします）` takes the full and the short form, not `douzoyoroshiku`, and the hit feedback has no " · romaji" part; `アメリカ` falls only with katakana on.
  - Let a word fall through: the miss feedback and the game-over list show the shown form.
  - Manage word lists → Details on a list with many skipped rows: the messages scroll, the hint stays below.
