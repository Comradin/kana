# Complete Hiragana Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the Fyne desktop app to all 104 practical hiragana. New users follow a stepwise learning path, and every newly unlocked row is introduced in a dialog before it is asked. The terminal app is removed first.

**Architecture:** `kanacore` rows become the single source of truth: each row has a group and entries with canonical and alternative romaji. `CharacterSet`, `CharToRow`, `ProgressionSteps` and the UI are derived from the rows. `fyne/GameState` gains a pause state and a pending intro, and reports unlocks as events. The UI shows the intro dialog on the Fyne thread.

**Tech Stack:** Go 1.25, Fyne v2.7 (CGO), `modernc.org/sqlite`, the standard `testing` package with `fyne.io/fyne/v2/test`.

**Spec:** `docs/superpowers/specs/2026-09-30-complete-hiragana-design.md`

**Branch:** `feat/complete-hiragana` (already created; the spec is committed on it).

**Conventions:**
- Commit messages follow the repo: `type(scope): summary`, then a short paragraph. Never add `Co-Authored-By` or any AI/session trailer.
- The first `go test ./fyne/` builds Fyne through CGO and takes several minutes. Later runs are cached.
- Everything under `fyne/` is `package main`, so tests there can reach unexported fields.

---

## File Map

| File | Responsibility | Tasks |
|---|---|---|
| `main.go`, `game.go`, `ui.go`, `settings_form.go` (root) | Terminal app, **deleted** | 1 |
| `kanacore/kana.go` | `Kana`, `CharacterSet`, `NewCharacterSet`, `Matches` | 1, 3 |
| `kanacore/kana_rows.go` | Groups, entries, rows, lookups, `ProgressionSteps` | 3, 4, 5 |
| `kanacore/kana_test.go` | Data and matching tests | 3, 4, 5 |
| `fyne/game.go` | Game state: matching, progression, pause, intro, fresh start | 3, 6, 7, 8, 9, 10 |
| `fyne/tile.go` | Tile size and colours, text centring | 7 |
| `fyne/intro.go` (new) | Intro dialog | 11 |
| `fyne/app.go` | Window wiring, event watcher, game-over dialog | 11 |
| `fyne/input.go` | Enable/disable the entry | 11 |
| `fyne/settings.go` | Grouped row selection | 12 |
| `fyne/stats.go` | Progress table from PR #5, group label rows | 2, 3, 9, 13 |
| `fyne/*_test.go` | Tests | 3, 6–10, 12, 13 |
| `README.md`, `.gitignore`, `PLAN.md`, `CLAUDE.md` | Docs | 1, 14 |

---

### Task 1: Remove the terminal app

**Files:**
- Delete: `main.go`, `game.go`, `ui.go`, `settings_form.go`
- Modify: `kanacore/kana.go`, `go.mod`, `go.sum`, `.gitignore`, `README.md`, `PLAN.md`
- Modify (local only, gitignored): `CLAUDE.md`

- [ ] **Step 1: Delete the terminal app files**

```bash
git rm main.go game.go ui.go settings_form.go
```

- [ ] **Step 2: Remove the terminal-only fields `X`/`Y` from `kanacore.Kana`**

In `kanacore/kana.go`, replace the `Kana` struct with:

```go
// Kana represents a falling character in the game.
type Kana struct {
	Char   string
	Romaji string
	Speed  float32
}
```

- [ ] **Step 3: Tidy the modules**

Run: `go mod tidy`
Expected: `bubbletea`, `lipgloss`, `huh` and their Charm/muesli indirect dependencies disappear from `go.mod`. Check with `grep -c charmbracelet go.mod`, which should print `0`.

- [ ] **Step 4: Build, vet and test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all packages pass. There are no root-level tests any more.

- [ ] **Step 5: Ignore the desktop binary**

Append to `.gitignore`:

```
kana-desktop
```

- [ ] **Step 6: Drop the stale PLAN.md item**

Delete this line from `PLAN.md`:

```
- [ ] Sound effects via terminal bell
```

- [ ] **Step 7: Update README.md to describe the desktop app only**

Make these edits:
- In **Apps**: replace the two-row table and the paragraph after it with one sentence: “Kana is a Fyne v2 desktop app (`kana-desktop`).”
- Delete the **Terminal App (Bubble Tea)** subsection under Features.
- **Prerequisites**: merge “Both apps” and “Desktop app only” into one list.
- **Build from Source / Run directly / Development**: keep only the desktop commands (`go build -o kana-desktop ./fyne/`, `go run ./fyne/`).
- **How to Play / Controls**: drop the terminal variants and keep the desktop ones.
- **Architecture**: delete the `### Terminal App (``)` section, which contains the stale `kana.go` bullet. Move the `store/store.go` bullet under a new `### Persistence (store/)` heading.
- **Game Timing**: delete “Terminal fall speed”.
- **Data Persistence**: change “Both apps share” to “The app stores”.
- **Dependencies**: delete Bubble Tea, Lipgloss and Huh.

Verify: `grep -n -i "terminal\|bubble\|lipgloss\|huh" README.md` should print nothing.

- [ ] **Step 8: Update the local CLAUDE.md**

`CLAUDE.md` is ignored by the global gitignore, so it will not be committed. Replace its content with:

```markdown
# CLAUDE.md

Guidance for Claude Code in this repository.

## Project Overview

Kana is a Fyne v2 desktop typing game for learning Japanese hiragana. Kana tiles fall down the play field; the player types the romaji and presses Enter.

## Build and Run

    go build -o kana-desktop ./fyne/
    go run ./fyne/
    go test ./...

Fyne needs CGO and system GL/X11 headers; the first build is slow.

## Structure

- `kanacore/` — character data: rows (grouped, with canonical and alternative romaji), character sets, progression steps.
- `store/` — SQLite persistence (`kana.db` in the working directory): settings and per-character stats.
- `fyne/` — the desktop app (`package main`): `GameState` (game loop goroutines, answer matching, auto-progression, pause/intro), widgets (canvas, tiles, stats panel, input bar), dialogs (settings, intro, game over).

UI updates from goroutines must go through `fyne.Do`.
```

- [ ] **Step 9: Commit**

```bash
git add -A main.go game.go ui.go settings_form.go kanacore/kana.go go.mod go.sum .gitignore README.md PLAN.md
git commit -F - <<'EOF'
chore: remove the terminal app

Development concentrates on the Fyne desktop app. The Bubble Tea app, its
Charm dependencies and the terminal-only Kana position fields are removed,
and the README now documents the desktop app only.
EOF
```

---

### Task 2: Bring in PR #5 (aligned stats table)

**Files:**
- Modify: `fyne/stats.go` (via cherry-pick)

- [ ] **Step 1: Cherry-pick the table commit**

Run: `git cherry-pick 6bcea93`
Expected: it applies cleanly and touches only `fyne/stats.go`.

- [ ] **Step 2: Cherry-pick the follow-up commit without its dependency bumps**

Run: `git cherry-pick 9d95806`
Expected: conflicts in `go.mod`/`go.sum`, or it applies with the bumps. Either way, keep the branch's module files:

```bash
git checkout HEAD -- go.mod go.sum
go mod tidy
git diff HEAD --stat
```

Expected: `git diff HEAD --stat` lists only `fyne/stats.go`. If `go mod tidy` changed `go.mod`/`go.sum`, stop and report it, because the stats change should not need new modules.

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./fyne/`
Expected: PASS.

- [ ] **Step 4: Finish the cherry-pick**

If the cherry-pick stopped on the conflict:

```bash
git add fyne/stats.go go.mod go.sum
git cherry-pick --continue --no-edit
```

If it applied with the bumps (no conflict), amend it:

```bash
git add go.mod go.sum
git commit --amend --no-edit
```

Verify: `git show --stat HEAD` lists only `fyne/stats.go`.

---

### Task 3: Restructure kanacore around rows with entries

This task is behaviour-preserving for the 46 basic characters. It only adds the alternatives for し/ち/つ/ふ/ん.

**Files:**
- Modify: `kanacore/kana.go`, `kanacore/kana_rows.go`, `kanacore/kana_test.go`
- Modify (call sites): `fyne/game.go:549-567`, `fyne/stats.go` (the `row.Characters` loops), `fyne/game_test.go:110`

- [ ] **Step 1: Write the failing tests**

Append to `kanacore/kana_test.go`:

```go
func TestMatchesBasic(t *testing.T) {
	cs := Hiragana()
	cases := []struct {
		char, input string
		want        bool
	}{
		{"か", "ka", true},
		{"か", "  KA ", true},
		{"し", "shi", true},
		{"し", "si", true},
		{"ち", "ti", true},
		{"つ", "tu", true},
		{"ふ", "hu", true},
		{"ん", "n", true},
		{"ん", "nn", true},
		{"を", "wo", true},
		{"を", "o", false},
		{"か", "ki", false},
		{"か", "", false},
		{"x", "ka", false},
	}
	for _, c := range cases {
		if got := cs.Matches(c.char, c.input); got != c.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", c.char, c.input, got, c.want)
		}
	}
}

func TestRowByID(t *testing.T) {
	row, ok := RowByID("k")
	if !ok || row.Label != "K-row (か)" {
		t.Fatalf("RowByID(k) = %+v, %v", row, ok)
	}
	if _, ok := RowByID("nope"); ok {
		t.Fatal("RowByID(nope) should not exist")
	}
}

func TestRowCharactersFollowEntries(t *testing.T) {
	row, _ := RowByID("vowels")
	got := row.Characters()
	want := []string{"あ", "い", "う", "え", "お"}
	if len(got) != len(want) {
		t.Fatalf("Characters() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Characters() = %v, want %v", got, want)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./kanacore/`
Expected: FAIL to compile (`cs.Matches undefined`, `undefined: RowByID`, `cannot call non-function row.Characters`).

- [ ] **Step 3: Rewrite `kanacore/kana.go`**

```go
package kanacore

import "strings"

// Kana represents a falling character in the game.
type Kana struct {
	Char   string
	Romaji string
	Speed  float32
}

// CharacterSet represents a collection of kana characters with their romaji.
type CharacterSet struct {
	Name string
	Data map[string]string // char → canonical romaji
	alts map[string][]string
}

// NewCharacterSet builds a character set from the entries of the given rows.
func NewCharacterSet(name string, rows []KanaRow) CharacterSet {
	cs := CharacterSet{
		Name: name,
		Data: make(map[string]string),
		alts: make(map[string][]string),
	}
	for _, row := range rows {
		for _, e := range row.Entries {
			cs.Data[e.Char] = e.Romaji
			if len(e.Alt) > 0 {
				cs.alts[e.Char] = e.Alt
			}
		}
	}
	return cs
}

// Hiragana returns the character set of all hiragana rows.
func Hiragana() CharacterSet {
	return NewCharacterSet("Hiragana", AllKanaRows)
}

// GetCharacters returns a slice of all characters in the set.
func (cs CharacterSet) GetCharacters() []string {
	chars := make([]string, 0, len(cs.Data))
	for char := range cs.Data {
		chars = append(chars, char)
	}
	return chars
}

// GetRomaji returns the canonical romaji for a given character.
func (cs CharacterSet) GetRomaji(char string) (string, bool) {
	romaji, exists := cs.Data[char]
	return romaji, exists
}

// Matches reports whether input is the canonical romaji or an accepted
// alternative for char. Input is trimmed and lower-cased first.
func (cs CharacterSet) Matches(char, input string) bool {
	in := strings.ToLower(strings.TrimSpace(input))
	if in == "" {
		return false
	}
	romaji, ok := cs.Data[char]
	if !ok {
		return false
	}
	if in == romaji {
		return true
	}
	for _, alt := range cs.alts[char] {
		if in == alt {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Rewrite `kanacore/kana_rows.go`**

```go
package kanacore

// Group classifies rows by kind of kana; the order of Groups() is the
// progression order.
type Group string

const (
	GroupBasic      Group = "basic"
	GroupDakuon     Group = "dakuon"
	GroupHandakuon  Group = "handakuon"
	GroupYoon       Group = "yoon"
	GroupYoonDakuon Group = "yoon-dakuon"
)

// GroupInfo pairs a group with its display label.
type GroupInfo struct {
	Group Group
	Label string
}

// Groups returns all groups in display and progression order.
func Groups() []GroupInfo {
	return []GroupInfo{
		{GroupBasic, "Basic"},
		{GroupDakuon, "Dakuon"},
		{GroupHandakuon, "Handakuon"},
		{GroupYoon, "Yōon"},
		{GroupYoonDakuon, "Yōon + Dakuten"},
	}
}

// Entry is one kana with its canonical Hepburn romaji and accepted alternatives.
type Entry struct {
	Char   string
	Romaji string
	Alt    []string
}

// KanaRow groups related kana characters by their consonant row.
type KanaRow struct {
	ID      string
	Label   string
	Group   Group
	Entries []Entry
}

// Characters returns the row's kana in entry order.
func (r KanaRow) Characters() []string {
	chars := make([]string, len(r.Entries))
	for i, e := range r.Entries {
		chars[i] = e.Char
	}
	return chars
}

func entry(char, romaji string, alt ...string) Entry {
	return Entry{Char: char, Romaji: romaji, Alt: alt}
}

// AllKanaRows lists every row in progression order.
var AllKanaRows = []KanaRow{
	{ID: "vowels", Label: "Vowels (あ)", Group: GroupBasic, Entries: []Entry{entry("あ", "a"), entry("い", "i"), entry("う", "u"), entry("え", "e"), entry("お", "o")}},
	{ID: "k", Label: "K-row (か)", Group: GroupBasic, Entries: []Entry{entry("か", "ka"), entry("き", "ki"), entry("く", "ku"), entry("け", "ke"), entry("こ", "ko")}},
	{ID: "s", Label: "S-row (さ)", Group: GroupBasic, Entries: []Entry{entry("さ", "sa"), entry("し", "shi", "si"), entry("す", "su"), entry("せ", "se"), entry("そ", "so")}},
	{ID: "t", Label: "T-row (た)", Group: GroupBasic, Entries: []Entry{entry("た", "ta"), entry("ち", "chi", "ti"), entry("つ", "tsu", "tu"), entry("て", "te"), entry("と", "to")}},
	{ID: "n", Label: "N-row (な)", Group: GroupBasic, Entries: []Entry{entry("な", "na"), entry("に", "ni"), entry("ぬ", "nu"), entry("ね", "ne"), entry("の", "no")}},
	{ID: "h", Label: "H-row (は)", Group: GroupBasic, Entries: []Entry{entry("は", "ha"), entry("ひ", "hi"), entry("ふ", "fu", "hu"), entry("へ", "he"), entry("ほ", "ho")}},
	{ID: "m", Label: "M-row (ま)", Group: GroupBasic, Entries: []Entry{entry("ま", "ma"), entry("み", "mi"), entry("む", "mu"), entry("め", "me"), entry("も", "mo")}},
	{ID: "y", Label: "Y-row (や)", Group: GroupBasic, Entries: []Entry{entry("や", "ya"), entry("ゆ", "yu"), entry("よ", "yo")}},
	{ID: "r", Label: "R-row (ら)", Group: GroupBasic, Entries: []Entry{entry("ら", "ra"), entry("り", "ri"), entry("る", "ru"), entry("れ", "re"), entry("ろ", "ro")}},
	{ID: "w", Label: "W-row (わ)", Group: GroupBasic, Entries: []Entry{entry("わ", "wa"), entry("を", "wo")}},
	{ID: "n-only", Label: "N (ん)", Group: GroupBasic, Entries: []Entry{entry("ん", "n", "nn")}},
}

// CharToRow maps each kana character to its row ID.
var CharToRow map[string]string

func init() {
	CharToRow = make(map[string]string)
	for _, row := range AllKanaRows {
		for _, char := range row.Characters() {
			CharToRow[char] = row.ID
		}
	}
}

// RowsInGroup returns the rows of one group in progression order.
func RowsInGroup(g Group) []KanaRow {
	rows := make([]KanaRow, 0)
	for _, row := range AllKanaRows {
		if row.Group == g {
			rows = append(rows, row)
		}
	}
	return rows
}

// BasicRows returns the 46-character basic rows.
func BasicRows() []KanaRow {
	return RowsInGroup(GroupBasic)
}

// RowByID looks up a row by its ID.
func RowByID(id string) (KanaRow, bool) {
	for _, row := range AllKanaRows {
		if row.ID == id {
			return row, true
		}
	}
	return KanaRow{}, false
}

// DefaultRowIDs returns the IDs of the basic rows, used as the fallback selection.
func DefaultRowIDs() []string {
	rows := BasicRows()
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}
```

- [ ] **Step 5: Fix the call sites of `row.Characters`**

`fyne/game.go`, in `isRowMastered`: replace the body so the slice is computed once:

```go
func (gs *GameState) isRowMastered(row kanacore.KanaRow) bool {
	chars := row.Characters()
	if len(chars) == 0 {
		return true
	}

	masteredCount := 0
	for _, char := range chars {
		total := gs.overallStats[char].CorrectCount + gs.sessionStats[char].CorrectCount
		if total >= 3 {
			masteredCount++
		}
	}

	threshold := int(float64(len(chars)) * 0.8)
	if threshold == 0 {
		threshold = 1
	}
	return masteredCount >= threshold
}
```

`fyne/game_test.go:110`: change `row.Characters[:4]` to `row.Characters()[:4]`.

`fyne/stats.go` (PR #5 version): delete the line `cs := kanacore.Hiragana()`, then replace the character-placement loop

```go
		for _, char := range row.Characters {
			romaji, ok := cs.GetRomaji(char)
			if !ok {
				continue
			}
			col := vowelColIndex(romaji)
```

with

```go
		for _, e := range row.Entries {
			char := e.Char
			col := vowelColIndex(e.Romaji)
```

Leave the rest of that loop body unchanged. In the missed-label loop, change `for _, char := range row.Characters {` to `for _, char := range row.Characters() {`.

- [ ] **Step 6: Run all tests**

Run: `go vet ./... && go test ./kanacore/ ./fyne/`
Expected: PASS, including the existing `TestHiraganaGetCharacters` (46) and `TestAllKanaRowsCount` (11).

- [ ] **Step 7: Commit**

```bash
git add kanacore fyne/game.go fyne/game_test.go fyne/stats.go
git commit -F - <<'EOF'
refactor(kanacore): derive character data from grouped rows

Rows now carry a group and entries with canonical Hepburn romaji and
accepted alternatives, and the character set, row lookup and default
selection are derived from them. Matching accepts si/ti/tu/hu/nn.
EOF
```

---

### Task 4: Add Dakuon, Handakuon and Yōon rows

**Files:**
- Modify: `kanacore/kana_rows.go` (append to `AllKanaRows`)
- Modify: `kanacore/kana_test.go`

- [ ] **Step 1: Write the failing tests**

In `kanacore/kana_test.go`, change `TestHiraganaGetCharacters` to expect `104` and `TestAllKanaRowsCount` to expect `27`. Keep `TestDefaultRowIDs` at 11. Then append:

```go
func TestAllKanaRowsHaveNoDuplicateCharacters(t *testing.T) {
	seen := make(map[string]string)
	total := 0
	for _, row := range AllKanaRows {
		for _, char := range row.Characters() {
			if other, dup := seen[char]; dup {
				t.Fatalf("%s appears in rows %s and %s", char, other, row.ID)
			}
			seen[char] = row.ID
			total++
		}
	}
	if total != 104 {
		t.Fatalf("expected 104 characters, got %d", total)
	}
}

func TestRowsAreOrderedByGroup(t *testing.T) {
	order := make(map[Group]int)
	for i, g := range Groups() {
		order[g.Group] = i
	}
	last := 0
	for _, row := range AllKanaRows {
		idx, ok := order[row.Group]
		if !ok {
			t.Fatalf("row %s has unknown group %q", row.ID, row.Group)
		}
		if idx < last {
			t.Fatalf("row %s (group %s) appears after a later group", row.ID, row.Group)
		}
		last = idx
	}
}

func TestBasicRowsUnchanged(t *testing.T) {
	rows := BasicRows()
	wantIDs := []string{"vowels", "k", "s", "t", "n", "h", "m", "y", "r", "w", "n-only"}
	if len(rows) != len(wantIDs) {
		t.Fatalf("BasicRows() has %d rows, want %d", len(rows), len(wantIDs))
	}
	chars := 0
	for i, row := range rows {
		if row.ID != wantIDs[i] {
			t.Errorf("BasicRows()[%d] = %s, want %s", i, row.ID, wantIDs[i])
		}
		chars += len(row.Entries)
	}
	if chars != 46 {
		t.Errorf("BasicRows() has %d characters, want 46", chars)
	}
}

func TestMatchesExtended(t *testing.T) {
	cs := Hiragana()
	cases := []struct {
		char, input string
		want        bool
	}{
		{"が", "ga", true},
		{"じ", "ji", true},
		{"じ", "zi", true},
		{"じ", "di", false},
		{"ぢ", "ji", true},
		{"ぢ", "di", true},
		{"ず", "zu", true},
		{"ず", "du", false},
		{"づ", "zu", true},
		{"づ", "du", true},
		{"ぱ", "pa", true},
		{"きゃ", "kya", true},
		{"しゃ", "sha", true},
		{"しゃ", "sya", true},
		{"ちゃ", "cha", true},
		{"ちゃ", "tya", true},
		{"ちゃ", "cya", true},
		{"じゃ", "ja", true},
		{"じゃ", "jya", true},
		{"じゃ", "zya", true},
		{"ぴょ", "pyo", true},
		{"きゃ", "ka", false},
	}
	for _, c := range cases {
		if got := cs.Matches(c.char, c.input); got != c.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", c.char, c.input, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./kanacore/`
Expected: FAIL (`expected 104 characters, got 46`, `expected 27 rows, got 11`, `Matches("が", "ga") = false`).

- [ ] **Step 3: Append the new rows**

In `kanacore/kana_rows.go`, add these entries at the end of the `AllKanaRows` literal, after the `n-only` row:

```go
	// Dakuon
	{ID: "g", Label: "G-row (が)", Group: GroupDakuon, Entries: []Entry{entry("が", "ga"), entry("ぎ", "gi"), entry("ぐ", "gu"), entry("げ", "ge"), entry("ご", "go")}},
	{ID: "z", Label: "Z-row (ざ)", Group: GroupDakuon, Entries: []Entry{entry("ざ", "za"), entry("じ", "ji", "zi"), entry("ず", "zu"), entry("ぜ", "ze"), entry("ぞ", "zo")}},
	{ID: "d", Label: "D-row (だ)", Group: GroupDakuon, Entries: []Entry{entry("だ", "da"), entry("ぢ", "ji", "di"), entry("づ", "zu", "du"), entry("で", "de"), entry("ど", "do")}},
	{ID: "b", Label: "B-row (ば)", Group: GroupDakuon, Entries: []Entry{entry("ば", "ba"), entry("び", "bi"), entry("ぶ", "bu"), entry("べ", "be"), entry("ぼ", "bo")}},
	// Handakuon
	{ID: "p", Label: "P-row (ぱ)", Group: GroupHandakuon, Entries: []Entry{entry("ぱ", "pa"), entry("ぴ", "pi"), entry("ぷ", "pu"), entry("ぺ", "pe"), entry("ぽ", "po")}},
	// Yōon
	{ID: "ky", Label: "KY (きゃ)", Group: GroupYoon, Entries: []Entry{entry("きゃ", "kya"), entry("きゅ", "kyu"), entry("きょ", "kyo")}},
	{ID: "sy", Label: "SH (しゃ)", Group: GroupYoon, Entries: []Entry{entry("しゃ", "sha", "sya"), entry("しゅ", "shu", "syu"), entry("しょ", "sho", "syo")}},
	{ID: "ch", Label: "CH (ちゃ)", Group: GroupYoon, Entries: []Entry{entry("ちゃ", "cha", "tya", "cya"), entry("ちゅ", "chu", "tyu", "cyu"), entry("ちょ", "cho", "tyo", "cyo")}},
	{ID: "ny", Label: "NY (にゃ)", Group: GroupYoon, Entries: []Entry{entry("にゃ", "nya"), entry("にゅ", "nyu"), entry("にょ", "nyo")}},
	{ID: "hy", Label: "HY (ひゃ)", Group: GroupYoon, Entries: []Entry{entry("ひゃ", "hya"), entry("ひゅ", "hyu"), entry("ひょ", "hyo")}},
	{ID: "my", Label: "MY (みゃ)", Group: GroupYoon, Entries: []Entry{entry("みゃ", "mya"), entry("みゅ", "myu"), entry("みょ", "myo")}},
	{ID: "ry", Label: "RY (りゃ)", Group: GroupYoon, Entries: []Entry{entry("りゃ", "rya"), entry("りゅ", "ryu"), entry("りょ", "ryo")}},
	// Yōon with dakuten / handakuten
	{ID: "gy", Label: "GY (ぎゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("ぎゃ", "gya"), entry("ぎゅ", "gyu"), entry("ぎょ", "gyo")}},
	{ID: "j", Label: "J (じゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("じゃ", "ja", "zya", "jya"), entry("じゅ", "ju", "zyu", "jyu"), entry("じょ", "jo", "zyo", "jyo")}},
	{ID: "by", Label: "BY (びゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("びゃ", "bya"), entry("びゅ", "byu"), entry("びょ", "byo")}},
	{ID: "py", Label: "PY (ぴゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("ぴゃ", "pya"), entry("ぴゅ", "pyu"), entry("ぴょ", "pyo")}},
```

- [ ] **Step 4: Run all tests**

Run: `go test ./kanacore/ ./fyne/`
Expected: PASS. `fyne` still passes because the desktop app selects rows explicitly.

- [ ] **Step 5: Commit**

```bash
git add kanacore
git commit -F - <<'EOF'
feat(kanacore): add Dakuon, Handakuon and Yōon rows

Adds 58 characters in 16 rows with Hepburn romaji and common Kunrei/IME
alternatives, bringing the set to 104 hiragana. ぢ/づ share ji/zu with
じ/ず and can also be typed as di/du.
EOF
```

---

### Task 5: Progression steps

**Files:**
- Modify: `kanacore/kana_rows.go`, `kanacore/kana_test.go`

- [ ] **Step 1: Write the failing test**

Append to `kanacore/kana_test.go`:

```go
func TestProgressionStepsCoverRowsInOrder(t *testing.T) {
	var flat []string
	for i, step := range ProgressionSteps {
		if len(step) == 0 {
			t.Fatalf("step %d is empty", i)
		}
		first, _ := RowByID(step[0])
		for _, id := range step {
			row, ok := RowByID(id)
			if !ok {
				t.Fatalf("step %d has unknown row %q", i, id)
			}
			if row.Group != first.Group {
				t.Fatalf("step %d crosses groups: %v", i, step)
			}
		}
		flat = append(flat, step...)
	}
	if len(flat) != len(AllKanaRows) {
		t.Fatalf("steps cover %d rows, want %d", len(flat), len(AllKanaRows))
	}
	for i, row := range AllKanaRows {
		if flat[i] != row.ID {
			t.Fatalf("step order position %d = %s, want %s", i, flat[i], row.ID)
		}
	}
	if len(ProgressionSteps[0]) != 1 || ProgressionSteps[0][0] != "vowels" {
		t.Fatalf("first step = %v, want [vowels]", ProgressionSteps[0])
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./kanacore/ -run TestProgressionSteps`
Expected: FAIL to compile (`undefined: ProgressionSteps`).

- [ ] **Step 3: Add the steps**

Append to `kanacore/kana_rows.go`:

```go
// ProgressionSteps lists the row IDs that auto-progression unlocks together,
// in order. A step never crosses a group boundary.
var ProgressionSteps = [][]string{
	{"vowels"}, {"k", "s"}, {"t", "n"}, {"h", "m"}, {"y", "r"}, {"w", "n-only"},
	{"g", "z"}, {"d", "b"},
	{"p"},
	{"ky", "sy"}, {"ch", "ny"}, {"hy", "my"}, {"ry"},
	{"gy", "j"}, {"by", "py"},
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./kanacore/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kanacore
git commit -m "feat(kanacore): define learning path progression steps"
```

---

### Task 6: Answer matching with alternatives and lowest tile first

**Files:**
- Modify: `fyne/game.go` (`checkAnswer`)
- Test: `fyne/game_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `fyne/game_test.go` (add `"fyne.io/fyne/v2"` to the imports):

```go
func tileAt(char, romaji string, y float32) *KanaTile {
	tile := newKanaTile(kanacore.Kana{Char: char, Romaji: romaji, Speed: 5})
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestCheckAnswerAcceptsAlternative(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("し", "shi", 0)}
	gs.checkAnswer("si")
	if len(gs.tiles) != 0 {
		t.Fatalf("expected し removed by 'si', %d tiles left", len(gs.tiles))
	}
}

func TestCheckAnswerNormalisesInput(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("か", "ka", 0)}
	gs.checkAnswer("  KA ")
	if len(gs.tiles) != 0 {
		t.Fatalf("expected か removed by '  KA ', %d tiles left", len(gs.tiles))
	}
}

func TestCheckAnswerRemovesLowestMatchingTile(t *testing.T) {
	gs := newTestState()
	high := tileAt("じ", "ji", 100)
	low := tileAt("ぢ", "ji", 300)
	gs.tiles = []*KanaTile{high, low}
	gs.checkAnswer("ji")
	if len(gs.tiles) != 1 || gs.tiles[0] != high {
		t.Fatalf("expected only the upper じ tile to remain")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'TestCheckAnswer'`
Expected: `TestCheckAnswerAcceptsAlternative`, `TestCheckAnswerNormalisesInput` and `TestCheckAnswerRemovesLowestMatchingTile` FAIL.

- [ ] **Step 3: Implement**

Replace `checkAnswer` in `fyne/game.go`:

```go
// checkAnswer removes the lowest tile whose kana matches input (canonical or
// alternative romaji). Acquires the lock itself; releases before triggering
// canvas.Refresh() so the UI updates immediately on correct answers.
func (gs *GameState) checkAnswer(input string) {
	gs.mu.Lock()
	best := -1
	for i, tile := range gs.tiles {
		if !gs.charSet.Matches(tile.kana.Char, input) {
			continue
		}
		if best < 0 || tile.pos.Y > gs.tiles[best].pos.Y {
			best = i
		}
	}
	matched := best >= 0
	if matched {
		tile := gs.tiles[best]
		gs.tiles = append(gs.tiles[:best], gs.tiles[best+1:]...)
		gs.score += 10
		gs.recordCorrect(tile.kana.Char)
		if gs.scoreLimit > 0 && gs.score >= gs.scoreLimit {
			gs.endGame("score")
		}
		gs.buildSnapshot()
	}
	canvas := gs.canvas
	gs.mu.Unlock()

	if matched && canvas != nil {
		canvas.Refresh()
	}
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./fyne/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/game_test.go
git commit -F - <<'EOF'
feat(fyne): match alternative romaji and prefer the lowest tile

Input is trimmed and lower-cased and matched against canonical and
alternative romaji. When several tiles match, the one closest to the
bottom is removed, which makes shared answers like ji and zu predictable.
EOF
```

---

### Task 7: Tiles sized for two-glyph kana

**Files:**
- Modify: `fyne/tile.go`, `fyne/game.go` (`spawnKana`)
- Test: `fyne/tile_test.go`, `fyne/game_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `fyne/tile_test.go`:

```go
func TestWideTileForYoon(t *testing.T) {
	test.NewApp()
	narrow := newKanaTile(kanacore.Kana{Char: "き", Romaji: "ki"})
	wide := newKanaTile(kanacore.Kana{Char: "きゃ", Romaji: "kya"})
	if wide.Width() <= narrow.Width() {
		t.Fatalf("wide tile %.0f should be wider than narrow tile %.0f", wide.Width(), narrow.Width())
	}
}

func TestTileTextCentredInsideFace(t *testing.T) {
	test.NewApp()
	for _, char := range []string{"か", "きゃ"} {
		tile := newKanaTile(kanacore.Kana{Char: char})
		tile.Move(fyne.NewPos(40, 20))
		textX := tile.text.Position().X
		textW := tile.text.Size().Width
		left := textX - 40
		right := 40 + tile.Width() - (textX + textW)
		if left < 0 || right < 0 {
			t.Fatalf("%s: text outside face (left %.1f, right %.1f)", char, left, right)
		}
		if diff := left - right; diff > 1 || diff < -1 {
			t.Fatalf("%s: text not centred (left %.1f, right %.1f)", char, left, right)
		}
	}
}
```

Append to `fyne/game_test.go`:

```go
func TestSpawnKeepsWideTilesInsideCanvas(t *testing.T) {
	gs := newTestState()
	gs.selectedRows = map[string]bool{"ky": true}
	gs.canvasW = 100
	for i := 0; i < 50; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if tile.pos.X < 0 || tile.pos.X+tile.Width() > gs.canvasW {
			t.Fatalf("tile %s at x=%.1f width %.1f exceeds canvas width %.1f",
				tile.kana.Char, tile.pos.X, tile.Width(), gs.canvasW)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'Tile|Spawn'`
Expected: FAIL to compile (`tile.Width undefined`).

- [ ] **Step 3: Rewrite `fyne/tile.go`**

```go
package main

import (
	"image/color"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"kana/kanacore"
)

const (
	tileW     float32 = 52 // single glyph
	tileWideW float32 = 84 // two glyphs (yōon)
	tileH     float32 = 60
)

var (
	tileFaceColor   = color.RGBA{R: 0xee, G: 0xdf, B: 0xc0, A: 0xff}
	tileShadowColor = color.RGBA{R: 0xb8, G: 0x95, B: 0x6a, A: 0xff}
	kanaTextColor   = color.RGBA{R: 0x2c, G: 0x1a, B: 0x0e, A: 0xff}
)

// tileWidthFor returns the tile width for a kana, wider for two-glyph kana.
func tileWidthFor(char string) float32 {
	if utf8.RuneCountInString(char) > 1 {
		return tileWideW
	}
	return tileW
}

// KanaTile is a falling kana card rendered as three canvas objects.
type KanaTile struct {
	kana   kanacore.Kana
	pos    fyne.Position
	width  float32
	shadow *canvas.Rectangle
	face   *canvas.Rectangle
	text   *canvas.Text
}

func newKanaTile(k kanacore.Kana) *KanaTile {
	w := tileWidthFor(k.Char)

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(tileFaceColor)
	face.Resize(fyne.NewSize(w, tileH))

	text := canvas.NewText(k.Char, kanaTextColor)
	text.TextSize = 32
	text.Resize(fyne.MeasureText(text.Text, text.TextSize, text.TextStyle))

	t := &KanaTile{kana: k, width: w, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// Width returns the tile's width in device-independent pixels.
func (t *KanaTile) Width() float32 {
	return t.width
}

// Move updates the positions of all three canvas objects atomically.
func (t *KanaTile) Move(pos fyne.Position) {
	t.pos = pos
	t.shadow.Move(fyne.NewPos(pos.X+3, pos.Y+3))
	t.face.Move(pos)
	ts := t.text.Size()
	t.text.Move(fyne.NewPos(pos.X+(t.width-ts.Width)/2, pos.Y+(tileH-ts.Height)/2))
}

// Objects returns the canvas objects for the renderer, shadow first so face renders on top.
func (t *KanaTile) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{t.shadow, t.face, t.text}
}
```

- [ ] **Step 4: Use the real width when spawning**

In `spawnKana` in `fyne/game.go`, replace

```go
	maxX := gs.canvasW - tileW
```

with

```go
	maxX := gs.canvasW - tileWidthFor(char)
```

- [ ] **Step 5: Run the tests**

Run: `go test ./fyne/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add fyne/tile.go fyne/tile_test.go fyne/game.go fyne/game_test.go
git commit -F - <<'EOF'
feat(fyne): widen tiles for two-glyph kana

Yōon tiles are 84px wide instead of 52px. The glyph is centred from its
measured size instead of a fixed estimate, and spawning accounts for the
real tile width so wide tiles stay on the canvas.
EOF
```

---

### Task 8: Step-based auto-progression

**Files:**
- Modify: `fyne/game.go` (`checkAutoProgression` plus helpers)
- Test: `fyne/game_lifecycle_test.go` (rewrite `TestCheckAutoProgressionUnlocksNextRow`), `fyne/game_test.go`

- [ ] **Step 1: Write the failing tests**

In `fyne/game_lifecycle_test.go`, replace `TestCheckAutoProgressionUnlocksNextRow` with:

```go
// TestCheckAutoProgressionUnlocksNextStep verifies that when all selected rows
// are mastered the whole next progression step is unlocked.
func TestCheckAutoProgressionUnlocksNextStep(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	gs.selectedRows = map[string]bool{"vowels": true}
	masterRows(gs, "vowels")

	unlocked := gs.checkAutoProgression()
	if !equalIDs(unlocked, []string{"k", "s"}) {
		t.Fatalf("unlocked %v, want [k s]", unlocked)
	}
	if !gs.selectedRows["k"] || !gs.selectedRows["s"] {
		t.Error("expected k and s selected after unlock")
	}
}
```

Append to `fyne/game_test.go`:

```go
func masterRows(gs *GameState, ids ...string) {
	for _, id := range ids {
		row, _ := kanacore.RowByID(id)
		for _, char := range row.Characters() {
			gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
		}
	}
}

func selectRows(gs *GameState, ids ...string) {
	gs.selectedRows = make(map[string]bool)
	for _, id := range ids {
		gs.selectedRows[id] = true
	}
}

func equalIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestProgressionCrossesIntoDakuon(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	basic := kanacore.DefaultRowIDs()
	selectRows(gs, basic...)
	masterRows(gs, basic...)
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"g", "z"}) {
		t.Fatalf("unlocked %v, want [g z]", got)
	}
}

func TestProgressionAfterHandakuonUnlocksYoon(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	ids := append(kanacore.DefaultRowIDs(), "g", "z", "d", "b", "p")
	selectRows(gs, ids...)
	masterRows(gs, ids...)
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"ky", "sy"}) {
		t.Fatalf("unlocked %v, want [ky sy]", got)
	}
}

func TestProgressionCompletesPartialStep(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels", "k")
	masterRows(gs, "vowels", "k")
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"s"}) {
		t.Fatalf("unlocked %v, want [s]", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'Progression'`
Expected: FAIL (for example `unlocked [k], want [k s]`). `TestProgressionCompletesPartialStep` already passes with the old rule; that is expected.

- [ ] **Step 3: Implement**

Replace `checkAutoProgression` in `fyne/game.go` and add the helpers after it:

```go
// checkAutoProgression unlocks the rest of the next progression step when all
// selected rows are mastered. Must be called under lock. Returns the IDs unlocked.
func (gs *GameState) checkAutoProgression() []string {
	if !gs.autoProgress {
		return nil
	}
	next := gs.nextStepRows()
	if len(next) == 0 || !gs.selectedRowsMastered() {
		return nil
	}
	for _, id := range next {
		gs.selectedRows[id] = true
	}
	gs.saveSelectedRows()
	return next
}

// nextStepRows returns the unselected rows of the first incomplete progression
// step. Must be called under lock.
func (gs *GameState) nextStepRows() []string {
	for _, step := range kanacore.ProgressionSteps {
		var missing []string
		for _, id := range step {
			if !gs.selectedRows[id] {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return missing
		}
	}
	return nil
}

// selectedRowsMastered reports whether every selected row is mastered. Must be
// called under lock.
func (gs *GameState) selectedRowsMastered() bool {
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] && !gs.isRowMastered(row) {
			return false
		}
	}
	return true
}

// saveSelectedRows persists the selection in row order. Must be called under lock.
func (gs *GameState) saveSelectedRows() {
	if gs.store == nil {
		return
	}
	ids := make([]string, 0, len(gs.selectedRows))
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] {
			ids = append(ids, row.ID)
		}
	}
	_ = gs.store.SaveSelectedRows(ids)
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./fyne/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/game_test.go fyne/game_lifecycle_test.go
git commit -F - <<'EOF'
feat(fyne): unlock rows in learning path steps

Auto-progression now unlocks the remaining rows of the next progression
step instead of a single row, so learners move through the basic rows in
pairs and never mix groups within one step.
EOF
```

---

### Task 9: Pause, unlock events and pending intro

**Files:**
- Modify: `fyne/game.go`, `fyne/stats.go`
- Test: `fyne/game_test.go`, `fyne/stats_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `fyne/game_test.go`:

```go
// newUnlockReadyState returns a state where one correct あ unlocks [k s].
func newUnlockReadyState() *GameState {
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels")
	masterRows(gs, "vowels")
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	return gs
}

func TestUnlockPausesAndQueuesEvent(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a")
	if !gs.paused {
		t.Fatal("expected game paused after unlock")
	}
	if !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("pendingIntro = %v, want [k s]", gs.pendingIntro)
	}
	select {
	case ev := <-gs.eventCh:
		if ev.kind != rowsUnlockedEvent || !equalIDs(ev.rows, []string{"k", "s"}) {
			t.Fatalf("unexpected event %+v", ev)
		}
	default:
		t.Fatal("expected rowsUnlockedEvent on eventCh")
	}
}

func TestUnlockWithFullEventChannelDoesNotPause(t *testing.T) {
	gs := newUnlockReadyState()
	for i := 0; i < cap(gs.eventCh); i++ {
		gs.eventCh <- gameEvent{kind: gameOverEvent}
	}
	gs.checkAnswer("a")
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("expected no pause without a delivered event (paused=%v, pending=%v)", gs.paused, gs.pendingIntro)
	}
	if !gs.selectedRows["k"] || !gs.selectedRows["s"] {
		t.Fatal("rows should still be unlocked")
	}
}

func TestPausedGameDoesNotMoveSpawnOrScore(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.Pause()

	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 0 {
		t.Fatalf("tile moved while paused: y=%.1f", y)
	}
	gs.spawnKana()
	if len(gs.tiles) != 1 {
		t.Fatalf("spawned while paused: %d tiles", len(gs.tiles))
	}
	gs.checkAnswer("a")
	if len(gs.tiles) != 1 || gs.score != 0 {
		t.Fatal("answer accepted while paused")
	}

	gs.Resume()
	gs.tick()
	if y := gs.tiles[0].pos.Y; y == 0 {
		t.Fatal("tile did not move after Resume")
	}
}

func TestFinishIntroClearsPendingAndResumes(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a")
	gs.FinishIntro()
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("after FinishIntro paused=%v pending=%v", gs.paused, gs.pendingIntro)
	}
}

func TestUnlockAndScoreLimitKeepIntroForPlayAgain(t *testing.T) {
	gs := newUnlockReadyState()
	gs.scoreLimit = 10
	gs.checkAnswer("a")
	if !gs.over {
		t.Fatal("expected game over at score limit")
	}
	if !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("pendingIntro = %v, want [k s]", gs.pendingIntro)
	}
	first := <-gs.eventCh
	second := <-gs.eventCh
	if first.kind != rowsUnlockedEvent || second.kind != gameOverEvent {
		t.Fatalf("event order = %v, %v", first.kind, second.kind)
	}

	gs.Reset()
	if !gs.paused || !equalIDs(gs.PendingIntro(), []string{"k", "s"}) {
		t.Fatalf("after Reset paused=%v pending=%v, want paused with [k s]", gs.paused, gs.pendingIntro)
	}
}

func TestResetWithoutPendingIntroUnpauses(t *testing.T) {
	gs := newTestState()
	gs.Pause()
	gs.Reset()
	if gs.paused {
		t.Fatal("expected Reset to clear pause without a pending intro")
	}
}
```

In `fyne/stats_test.go`, delete `TestStatsPanelUnlockMessageClears` and remove the now-unused `"time"` import.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/`
Expected: FAIL to compile (`gs.paused undefined`, `rowsUnlockedEvent undefined`, `gs.Pause undefined`).

- [ ] **Step 3: Extend the event type and state**

In `fyne/game.go`, replace the event declarations:

```go
type gameEventType int

const (
	gameOverEvent gameEventType = iota
	rowsUnlockedEvent
)

type gameEvent struct {
	kind gameEventType
	rows []string // rowsUnlockedEvent: IDs to introduce
}
```

In the `GameState` struct, replace

```go
	selectedRows  map[string]bool
	autoProgress  bool
	newlyUnlocked []string
	unlockMessage string
	unlockAt      time.Time
```

with

```go
	selectedRows map[string]bool
	autoProgress bool

	paused       bool     // tick, spawn and answers are suspended
	pendingIntro []string // row IDs waiting to be introduced
```

- [ ] **Step 4: Announce unlocks, pause and resume**

Replace the unlock part of `recordCorrect`:

```go
	if unlocked := gs.checkAutoProgression(); len(unlocked) > 0 {
		gs.announceRows(unlocked)
	}
```

Delete `showUnlockMessage` and add:

```go
// announceRows queues an intro for newly unlocked rows and pauses the game
// until it is dismissed. If the event cannot be delivered the game keeps
// running. Must be called under lock.
func (gs *GameState) announceRows(ids []string) {
	rows := append([]string(nil), ids...)
	select {
	case gs.eventCh <- gameEvent{kind: rowsUnlockedEvent, rows: rows}:
		gs.paused = true
		gs.pendingIntro = rows
	default:
	}
}

// Pause suspends ticking, spawning and answer checking.
func (gs *GameState) Pause() {
	gs.mu.Lock()
	gs.paused = true
	gs.mu.Unlock()
}

// Resume continues a paused game.
func (gs *GameState) Resume() {
	gs.mu.Lock()
	gs.paused = false
	gs.mu.Unlock()
}

// FinishIntro clears the pending intro and resumes the game.
func (gs *GameState) FinishIntro() {
	gs.mu.Lock()
	gs.pendingIntro = nil
	gs.paused = false
	gs.mu.Unlock()
}

// PendingIntro returns a copy of the row IDs waiting to be introduced.
func (gs *GameState) PendingIntro() []string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return append([]string(nil), gs.pendingIntro...)
}
```

- [ ] **Step 5: Respect the pause**

In `tick` and `spawnKana`, change `if gs.over {` to `if gs.over || gs.paused {`. At the top of `checkAnswer`, right after `gs.mu.Lock()`, add:

```go
	if gs.paused {
		gs.mu.Unlock()
		return
	}
```

- [ ] **Step 6: Reset keeps a pending intro and clears the pause otherwise**

In `Reset`, replace

```go
	gs.newlyUnlocked = nil
	gs.unlockMessage = ""
	gs.unlockAt = time.Time{}
```

with

```go
	// An intro dropped by game over is shown before the new game starts.
	gs.paused = len(gs.pendingIntro) > 0
```

- [ ] **Step 7: Remove the unlock message from the snapshot and panel**

In `snapshot()`, delete the `UnlockMessage` and `UnlockAt` fields. In `fyne/stats.go`:
- delete `UnlockMessage string` and `UnlockAt time.Time` from `StatsSnapshot`;
- delete the `unlockLabel` field, its initialisation `unlockLabel: widget.NewLabel(""),` and `p.unlockLabel,` in the VBox;
- delete the `if snap.UnlockMessage != ""` block in `Update`;
- remove the `"time"` import.

Remove `"time"` from `fyne/game.go`'s imports only if the compiler reports it unused (the tick and spawn loops still use it).

- [ ] **Step 8: Run the tests**

Run: `go vet ./fyne/ && go test ./fyne/`
Expected: PASS. `TestMergeSessionStatsRoundTrip` still passes because three correct か answers don't master the selected rows.

- [ ] **Step 9: Commit**

```bash
git add fyne/game.go fyne/game_test.go fyne/stats.go fyne/stats_test.go
git commit -F - <<'EOF'
feat(fyne): pause the game while unlocked rows are introduced

An unlock now queues a rowsUnlockedEvent and pauses ticking, spawning and
answer checking until the intro is finished. If the same answer ends the
game, the intro stays pending and is shown after Play Again. The timed
unlock label in the stats panel is removed.
EOF
```

---

### Task 10: Fresh start with learning path and catch-up

**Files:**
- Modify: `fyne/game.go` (`NewGameState`, new helpers)
- Test: `fyne/game_lifecycle_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `fyne/game_lifecycle_test.go` (add `"kana/kanacore"` to the imports):

```go
func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestFreshStartBeginsLearningPath(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)

	gs := NewGameState(st)

	if len(gs.selectedRows) != 1 || !gs.selectedRows["vowels"] {
		t.Fatalf("selected rows = %v, want only vowels", gs.selectedRows)
	}
	if !gs.autoProgress || !gs.paused || !equalIDs(gs.pendingIntro, []string{"vowels"}) {
		t.Fatalf("auto=%v paused=%v pending=%v", gs.autoProgress, gs.paused, gs.pendingIntro)
	}
	rows, _ := st.SelectedRows()
	auto, _ := st.AutoProgress()
	if !equalIDs(rows, []string{"vowels"}) || !auto {
		t.Fatalf("persisted rows=%v auto=%v", rows, auto)
	}
}

func TestFreshStartOverridesStoredAutoProgressOff(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveAutoProgress(false)

	gs := NewGameState(st)
	if !gs.autoProgress {
		t.Fatal("fresh start should turn auto-progression on")
	}
}

func TestFreshStartCatchesUpMasteredSteps(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	for _, id := range []string{"vowels", "k", "s", "t", "n"} {
		row, _ := kanacore.RowByID(id)
		for _, char := range row.Characters() {
			_ = st.SaveKanaStats(char, 3, 0, 3)
		}
	}

	gs := NewGameState(st)

	for _, id := range []string{"vowels", "k", "s", "t", "n"} {
		if !gs.selectedRows[id] {
			t.Errorf("expected %s selected by catch-up", id)
		}
	}
	if gs.selectedRows["h"] {
		t.Error("catch-up must stop at the first unmastered step")
	}
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("returning user should get no intro (paused=%v pending=%v)", gs.paused, gs.pendingIntro)
	}
}

func TestReturningUserWithUnmasteredVowelsGetsNoIntro(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveKanaStats("あ", 1, 0, 1)

	gs := NewGameState(st)
	if len(gs.selectedRows) != 1 || !gs.selectedRows["vowels"] {
		t.Fatalf("selected rows = %v, want only vowels", gs.selectedRows)
	}
	if gs.paused || gs.pendingIntro != nil {
		t.Fatal("user with stats should start without intro")
	}
}

func TestSavedSelectionIsKept(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels", "k"})
	_ = st.SaveAutoProgress(false)

	gs := NewGameState(st)
	if len(gs.selectedRows) != 2 || !gs.selectedRows["k"] || gs.autoProgress || gs.paused {
		t.Fatalf("rows=%v auto=%v paused=%v", gs.selectedRows, gs.autoProgress, gs.paused)
	}
}

func TestStoreErrorDoesNotTriggerFreshStart(t *testing.T) {
	test.NewApp()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_ = st.Close() // every query now fails

	gs := NewGameState(st)
	if len(gs.selectedRows) != len(kanacore.DefaultRowIDs()) || gs.paused {
		t.Fatalf("rows=%v paused=%v, want basic defaults without pause", gs.selectedRows, gs.paused)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'FreshStart|Returning|SavedSelection|StoreError'`
Expected: `TestFreshStartBeginsLearningPath`, `TestFreshStartOverridesStoredAutoProgressOff`, `TestFreshStartCatchesUpMasteredSteps` and `TestReturningUserWithUnmasteredVowelsGetsNoIntro` FAIL, because today the app selects all basic rows.

- [ ] **Step 3: Implement**

In `NewGameState`, replace the whole `if st != nil { ... }` block with:

```go
	if st != nil {
		if stats, err := st.KanaStatistics(); err == nil {
			for _, stat := range stats {
				copied := stat
				gs.overallStats[stat.Char] = copied
				gs.currentStreak[stat.Char] = stat.Streak
			}
		}
		if auto, err := st.AutoProgress(); err == nil {
			gs.autoProgress = auto
		}
		if limit, err := st.ScoreLimit(); err == nil {
			if limit < 0 {
				limit = 0
			}
			gs.scoreLimit = limit
		}
		rows, err := st.SelectedRows()
		switch {
		case err != nil:
			// Keep the basic defaults; never overwrite what we could not read.
		case len(rows) > 0:
			gs.applySelectedRows(rows)
		default:
			gs.startLearningPath()
		}
	}
```

Add after `NewGameState`:

```go
// startLearningPath sets up a first launch: the first progression step plus
// every following step the learner has already mastered, with
// auto-progression on. Only a learner without any stats gets the intro for
// the first step. Must be called before the game starts (no lock needed).
func (gs *GameState) startLearningPath() {
	gs.applySelectedRows(kanacore.ProgressionSteps[0])
	for _, step := range kanacore.ProgressionSteps[1:] {
		if !gs.rowsMastered(step) {
			break
		}
		for _, id := range step {
			gs.selectedRows[id] = true
		}
	}
	gs.autoProgress = true
	if gs.store != nil {
		_ = gs.store.SaveAutoProgress(true)
	}
	gs.saveSelectedRows()

	if !gs.hasStats() {
		gs.pendingIntro = append([]string(nil), kanacore.ProgressionSteps[0]...)
		gs.paused = true
	}
}

// rowsMastered reports whether every given row is mastered. Must be called under lock.
func (gs *GameState) rowsMastered(ids []string) bool {
	for _, id := range ids {
		row, ok := kanacore.RowByID(id)
		if !ok || !gs.isRowMastered(row) {
			return false
		}
	}
	return true
}

// hasStats reports whether the learner has answered or missed anything before.
func (gs *GameState) hasStats() bool {
	for _, stat := range gs.overallStats {
		if stat.CorrectCount > 0 || stat.MissCount > 0 {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./fyne/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add fyne/game.go fyne/game_lifecycle_test.go
git commit -F - <<'EOF'
feat(fyne): start new learners on the vowel row

Without a saved selection the app starts the learning path: the vowel row
with auto-progression on, introduced before the first tile. Returning
learners with stats get their mastered steps back without intros, and a
store read error never overwrites the stored selection.
EOF
```

---

### Task 11: Intro dialog and event wiring

**Files:**
- Create: `fyne/intro.go`
- Modify: `fyne/app.go`, `fyne/input.go`

The dialog is UI-only. Its logic (pause, pending, FinishIntro) is tested in Tasks 9–10; this task is verified manually in Step 6.

- [ ] **Step 1: Let the input bar be disabled**

Append to `fyne/input.go`:

```go
// SetEnabled enables or disables romaji entry; focus returns to the entry when enabled.
func (ib *InputBar) SetEnabled(enabled bool, win fyne.Window) {
	if enabled {
		ib.entry.Enable()
		win.Canvas().Focus(ib.entry)
		return
	}
	ib.entry.Disable()
}
```

- [ ] **Step 2: Create `fyne/intro.go`**

```go
package main

import (
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

const (
	introCardW     float32 = 104
	introCardWideW float32 = 168
	introCardH     float32 = 120
)

// showIntroDialog presents newly unlocked rows as large tiles with their
// romaji. The game stays paused and input disabled until the dialog closes.
// Must run on the Fyne thread.
func showIntroDialog(gs *GameState, rowIDs []string, inputBar *InputBar, w fyne.Window) {
	labels := make([]string, 0, len(rowIDs))
	body := container.NewVBox()
	for _, id := range rowIDs {
		row, ok := kanacore.RowByID(id)
		if !ok {
			continue
		}
		labels = append(labels, row.Label)
		cards := container.NewHBox()
		for _, e := range row.Entries {
			cards.Add(newIntroCard(e))
		}
		body.Add(widget.NewLabelWithStyle(row.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		body.Add(cards)
	}

	inputBar.SetEnabled(false, w)
	d := dialog.NewCustom("New: "+strings.Join(labels, ", "), "Let's go", body, w)
	d.SetOnClosed(func() {
		gs.FinishIntro()
		inputBar.SetEnabled(true, w)
	})
	d.Show()
}

// newIntroCard renders one kana as a large paper tile with its romaji below.
func newIntroCard(e kanacore.Entry) fyne.CanvasObject {
	width := introCardW
	if utf8.RuneCountInString(e.Char) > 1 {
		width = introCardWideW
	}
	face := canvas.NewRectangle(tileFaceColor)
	face.StrokeColor = tileShadowColor
	face.StrokeWidth = 3
	face.SetMinSize(fyne.NewSize(width, introCardH))

	glyph := canvas.NewText(e.Char, kanaTextColor)
	glyph.TextSize = 64

	romaji := widget.NewLabelWithStyle(e.Romaji, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	return container.NewVBox(container.NewStack(face, container.NewCenter(glyph)), romaji)
}
```

- [ ] **Step 3: Route events through `fyne.Do`**

In `fyne/app.go`, replace `watchEvents`:

```go
func watchEvents(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, w fyne.Window) {
	for event := range gs.eventCh {
		switch event.kind {
		case rowsUnlockedEvent:
			gs.mu.Lock()
			over := gs.over
			gs.mu.Unlock()
			if over {
				// Game over wins; the intro stays pending for Play Again.
				continue
			}
			rows := event.rows
			fyne.Do(func() { showIntroDialog(gs, rows, inputBar, w) })
		case gameOverEvent:
			gs.mu.Lock()
			snap := gs.snapshot()
			reason := gs.overReason
			gs.mu.Unlock()
			fyne.Do(func() {
				showGameOverDialog(gs, snap, reason, statsPanel, gameCanvas, inputBar, w)
			})
		}
	}
}
```

- [ ] **Step 4: Show pending intros on start and after Play Again**

In `buildWindow`, after the line `go watchEvents(gs, statsPanel, gameCanvas, inputBar, w)`, add:

```go
	// First launch: introduce the first row once the window is up.
	a.Lifecycle().SetOnStarted(func() {
		if rows := gs.PendingIntro(); len(rows) > 0 {
			showIntroDialog(gs, rows, inputBar, w)
		}
	})
```

In `showGameOverDialog`'s Play Again branch, after `gameCanvas.Refresh()`, add:

```go
		if rows := gs.PendingIntro(); len(rows) > 0 {
			showIntroDialog(gs, rows, inputBar, w)
		}
```

- [ ] **Step 5: Build and run the tests**

Run: `go vet ./fyne/ && go test ./fyne/`
Expected: PASS.

- [ ] **Step 6: Verify manually**

Run from an empty directory, so a fresh `kana.db` is created:

```bash
go build -o /tmp/kana-desktop ./fyne/ && mkdir -p /tmp/kana-fresh && cd /tmp/kana-fresh && rm -f kana.db && /tmp/kana-desktop
```

Check:
1. The window opens with a dialog “New: Vowels (あ)” showing あいうえお as large tiles with romaji. No tiles fall behind it and the entry is disabled.
2. After “Let's go”, tiles start falling, only vowels spawn, and the entry has focus.
3. Answer vowels until each of at least four has been answered correctly three times. The game pauses and “New: K-row (か), S-row (さ)” appears. After closing it, か/さ-row tiles spawn.
4. The settings gear still works while the game runs.

If the start dialog does not appear in step 1, replace the `SetOnStarted` block with a direct call queued through `fyne.Do(func() { ... })` in `buildWindow` (the spec's fallback) and check again.

- [ ] **Step 7: Commit**

```bash
git add fyne/intro.go fyne/app.go fyne/input.go
git commit -F - <<'EOF'
feat(fyne): introduce unlocked rows in a dialog

Newly unlocked rows are shown as large paper tiles with their romaji
before they are asked; the game stays paused and input disabled until the
learner continues. The first launch introduces the vowel row, and the
game-over dialog now opens on the Fyne thread as well.
EOF
```

---

### Task 12: Grouped settings dialog

**Files:**
- Modify: `fyne/settings.go`
- Test: `fyne/settings_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `fyne/settings_test.go`:

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestCheckedRowIDsFollowsRowOrder(t *testing.T) {
	test.NewApp()
	checks := map[string]*widget.Check{
		"ky":     widget.NewCheck("", nil),
		"vowels": widget.NewCheck("", nil),
		"g":      widget.NewCheck("", nil),
		"k":      widget.NewCheck("", nil),
	}
	checks["ky"].SetChecked(true)
	checks["vowels"].SetChecked(true)
	checks["g"].SetChecked(true)

	got := checkedRowIDs(checks)
	if !equalIDs(got, []string{"vowels", "g", "ky"}) {
		t.Fatalf("checkedRowIDs = %v, want [vowels g ky]", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./fyne/ -run TestCheckedRowIDs`
Expected: FAIL to compile (`undefined: checkedRowIDs`).

- [ ] **Step 3: Rewrite the row selection in `fyne/settings.go`**

Replace the part of `showSettingsDialog` from `// Build options list (labels)` down to `rowCheck.SetSelected(selectedLabels)` with:

```go
	rowChecks := make(map[string]*widget.Check)
	sections := container.NewVBox()
	for _, g := range kanacore.Groups() {
		groupRows := kanacore.RowsInGroup(g.Group)
		checks := make([]*widget.Check, 0, len(groupRows))
		grid := container.NewGridWithColumns(2)
		for _, row := range groupRows {
			c := widget.NewCheck(row.Label, nil)
			c.SetChecked(selected[row.ID])
			rowChecks[row.ID] = c
			checks = append(checks, c)
			grid.Add(c)
		}
		all := widget.NewCheck("all", nil)
		all.SetChecked(allChecked(checks))
		all.OnChanged = func(on bool) {
			for _, c := range checks {
				c.SetChecked(on)
			}
		}
		sections.Add(container.NewHBox(
			widget.NewLabelWithStyle(g.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			all,
		))
		sections.Add(grid)
	}
	rowScroll := container.NewVScroll(sections)
	rowScroll.SetMinSize(fyne.NewSize(460, 320))
```

At the top of the function, replace the `currentRowIDs` construction with a copy of the selection:

```go
	gs.mu.Lock()
	selected := make(map[string]bool, len(gs.selectedRows))
	for id, ok := range gs.selectedRows {
		selected[id] = ok
	}
	currentAuto := gs.autoProgress
	currentLimit := gs.scoreLimit
	gs.mu.Unlock()
```

In the form, replace `rowCheck,` with `rowScroll,`. In the save callback, replace the label-to-ID mapping block (from `// Map selected labels back to IDs` to the closing brace of the `for _, lbl := range rowCheck.Selected` loop) with:

```go
		newRows := checkedRowIDs(rowChecks)
```

Keep the existing `if len(newRows) == 0 { newRows = kanacore.DefaultRowIDs() }`.

Append the helpers to `fyne/settings.go`:

```go
// checkedRowIDs returns the IDs of checked rows in progression order.
func checkedRowIDs(checks map[string]*widget.Check) []string {
	ids := make([]string, 0, len(checks))
	for _, row := range kanacore.AllKanaRows {
		if c, ok := checks[row.ID]; ok && c.Checked {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

func allChecked(checks []*widget.Check) bool {
	for _, c := range checks {
		if !c.Checked {
			return false
		}
	}
	return len(checks) > 0
}
```

- [ ] **Step 4: Run the tests and check visually**

Run: `go vet ./fyne/ && go test ./fyne/`
Expected: PASS.

Manual check: run the app and open the gear. There should be five groups, each with a bold heading and an “all” check. The row list scrolls, and the dialog fits into the 900×620 window. Ticking “all” for Dakuon and saving makes が–ぼ tiles spawn.

- [ ] **Step 5: Commit**

```bash
git add fyne/settings.go fyne/settings_test.go
git commit -F - <<'EOF'
feat(fyne): group row selection in the settings dialog

Rows are listed per group with an "all" toggle in a scrollable section,
so learners with prior knowledge can add whole groups at once.
EOF
```

---

### Task 13: Stats table group labels

**Files:**
- Modify: `fyne/stats.go`
- Test: `fyne/stats_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `fyne/stats_test.go`:

```go
func TestStatsPanelShowsGroupLabelForVisibleRows(t *testing.T) {
	test.NewApp()
	panel := newStatsPanel()
	panel.Update(StatsSnapshot{
		SessionStats: map[string]store.KanaStats{},
		SelectedRows: map[string]bool{"vowels": true, "g": true},
	})
	if !panel.groupCells[kanacore.GroupDakuon][0].Visible() {
		t.Error("expected Dakuon group label visible")
	}
	if panel.groupCells[kanacore.GroupYoon][0].Visible() {
		t.Error("expected Yōon group label hidden")
	}
	if _, ok := panel.groupCells[kanacore.GroupBasic]; ok {
		t.Error("basic group should have no label row")
	}
}

func TestRowShortLabelForSH(t *testing.T) {
	if got := rowShortLabel("sy"); got != "sh" {
		t.Fatalf("rowShortLabel(sy) = %q, want sh", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./fyne/ -run 'GroupLabel|ShortLabel'`
Expected: FAIL to compile (`panel.groupCells undefined`).

- [ ] **Step 3: Implement**

In `fyne/stats.go`:

Add the field to `StatsPanel`:

```go
	// groupCells holds the label row shown above each non-basic group.
	groupCells map[kanacore.Group][6]*widget.Label
```

Initialise it in `newStatsPanel` next to `rowCells`: `groupCells: make(map[kanacore.Group][6]*widget.Label),`.

Add `case "sy": return "sh"` to `rowShortLabel`.

In the table-building loop `for _, row := range kanacore.AllKanaRows {`, insert this at the top of the loop body, before `rowLbl := ...`:

```go
		if _, done := p.groupCells[row.Group]; !done && row.Group != kanacore.GroupBasic {
			var g6 [6]*widget.Label
			g6[0] = widget.NewLabelWithStyle(groupLabel(row.Group), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
			for i := 1; i < 6; i++ {
				g6[i] = widget.NewLabel("")
			}
			for _, lbl := range g6 {
				lbl.Hide()
				gridItems = append(gridItems, lbl)
			}
			p.groupCells[row.Group] = g6
		}
```

Add the helper:

```go
func groupLabel(g kanacore.Group) string {
	for _, info := range kanacore.Groups() {
		if info.Group == g {
			return info.Label
		}
	}
	return string(g)
}
```

In `Update`, after the loop that shows and hides `rowCells`, add:

```go
	for group, g6 := range p.groupCells {
		visible := false
		for _, row := range kanacore.RowsInGroup(group) {
			if snap.SelectedRows[row.ID] {
				visible = true
				break
			}
		}
		for _, lbl := range g6 {
			if visible {
				lbl.Show()
			} else {
				lbl.Hide()
			}
		}
	}
```

- [ ] **Step 4: Run the tests and check visually**

Run: `go vet ./fyne/ && go test ./fyne/`
Expected: PASS.

Manual check: with Dakuon and Yōon rows selected in the settings, the progress table shows the group label rows. If “Yōon + Dakuten” visibly widens all six columns, change `groupLabel` to use short labels (`Daku`, `Handaku`, `Yōon`, `Yōon+D`) and note it in the commit message. This fallback is allowed by the spec.

- [ ] **Step 5: Commit**

```bash
git add fyne/stats.go fyne/stats_test.go
git commit -F - <<'EOF'
feat(fyne): label kana groups in the progress table

The progress table shows a group label row above the visible Dakuon,
Handakuon and Yōon rows, and the しゃ row is labelled sh.
EOF
```

---

### Task 14: Final verification and docs

**Files:**
- Modify: `README.md`, `PLAN.md`

- [ ] **Step 1: Full test run with the race detector**

Run: `go vet ./... && go test -race ./...`
Expected: PASS with no race reports.

- [ ] **Step 2: Update the README character set**

In `README.md`:
- In **Core Gameplay**, replace “46 Basic Hiragana Characters” with “104 Hiragana: basic, Dakuon, Handakuon and Yōon”.
- Add a bullet “Learning path: start with あいうえお, unlock two rows at a time, each new row introduced before it is asked”.
- Under **Character Set**, add tables for Dakuon/Handakuon and Yōon, copied from the spec's *Character Data* section, and mention the accepted alternatives (si, ti, tu, hu, zi, di, du, sya, cya, jya, nn).
- In **Customization**, change the Auto-Progression bullet to “unlocks the next step (two rows) once 80% of each active row is mastered”.

- [ ] **Step 3: Tick PLAN.md**

In `PLAN.md`, change these to `[x]`: “Add Dakuten characters”, “Add Handakuten characters”, “Add Yoon combinations”.

- [ ] **Step 4: Commit**

```bash
git add README.md PLAN.md
git commit -m "docs: describe the complete hiragana set and learning path"
```

- [ ] **Step 5: Hand back**

Report: the task list, the final `go test -race ./...` output, and the manual checks from Tasks 11–13. Do **not** push or open a PR: the user decides that, and closing PR #5 is part of that decision.
