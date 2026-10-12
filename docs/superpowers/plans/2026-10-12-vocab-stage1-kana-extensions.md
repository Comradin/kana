# Vocabulary Trainer Stage 1: Kana Extensions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the extended katakana learning-path group and the word-reading functions (`KanaUnits`, `ReadingMatches`, `Romaji`) that later stages of the vocabulary trainer build on.

**Architecture:** All logic lives in `kanacore`. The extended rows are hand-written katakana rows appended after the generated katakana rows, with three extra katakana-only progression steps. A new file `kanacore/words.go` splits words into units and matches romaji input with a memoised recursion. The Fyne app only changes where it lists groups (`GroupsFor`) and how the status panel shows the extended rows and a longer path line.

**Tech Stack:** Go 1.25, Fyne v2.7.3, standard `testing`.

**Spec:** `docs/superpowers/specs/2026-10-12-vocab-trainer-design.md`, section "Stage 1".

**Conventions:** Run tests with `go test -count=1 ./...` from the repository root. Commit messages: `type(scope): summary` plus a short prose paragraph; no Co-Authored-By or any AI/assistant trailers. `gofmt -l .` must print nothing before each commit.

---

## File Structure

- Modify `kanacore/kana_rows.go`: `GroupExtended`, extended rows, `KatakanaRows` append, extra steps, `GroupsFor`.
- Modify `kanacore/kana_test.go`: updated counts, mirror test limited to generated rows, new tests.
- Create `kanacore/words.go`: unit table, `KanaUnits`, `ReadingMatches`, `Romaji`.
- Create `kanacore/words_test.go`.
- Modify `fyne/settings.go`: `GroupsFor` instead of `Groups()`.
- Modify `fyne/stats.go`: paired grid via `GroupsFor(ScriptHiragana)`, extended block, segment width fitted to step count, short labels for extended rows.
- Modify `fyne/stats_test.go`.
- Modify `README.md`, `PLAN.md`.

---

### Task 1: Extended katakana rows, steps and `GroupsFor`

**Files:**
- Modify: `kanacore/kana_rows.go`
- Test: `kanacore/kana_test.go`

- [ ] **Step 1: Update existing tests and add new ones**

In `kanacore/kana_test.go`:

1. `TestAllKanaRowsCount`: expect `60` rows (54 + 6).
2. `TestAllKanaRowsHaveNoDuplicateCharacters`: expect `231` characters (208 + 23).
3. `TestKatakanaTable`: the final count check expects `127` characters; add cases
   `{"ティ", "ti", []string{"thi"}}`, `{"ヴ", "vu", nil}`, `{"ジェ", "je", []string{"zye", "jye"}}`,
   `{"ツォ", "tso", nil}`.
4. `TestAllKanaHasBothScripts`: expect `231` characters.
5. `TestKatakanaRowsMirrorHiragana`: compare only the generated rows. Replace the length check
   and loop header with:

```go
	generated := KatakanaRows[:len(HiraganaRows)]
	for i, k := range generated {
```

and add after the loop:

```go
	for _, row := range KatakanaRows[len(HiraganaRows):] {
		if row.Group != GroupExtended || row.Script != ScriptKatakana {
			t.Fatalf("row %s after the generated rows must be extended katakana", row.ID)
		}
	}
```

Append new tests:

```go
func TestExtendedKatakanaRows(t *testing.T) {
	rows := RowsInGroup(ScriptKatakana, GroupExtended)
	wantIDs := []string{"kata:ext-t", "kata:ext-f", "kata:ext-w", "kata:ext-sh", "kata:ext-v", "kata:ext-ts"}
	if len(rows) != len(wantIDs) {
		t.Fatalf("extended rows = %d, want %d", len(rows), len(wantIDs))
	}
	total := 0
	for i, row := range rows {
		if row.ID != wantIDs[i] {
			t.Fatalf("extended row %d = %s, want %s", i, row.ID, wantIDs[i])
		}
		if ScriptOf(row.ID) != ScriptKatakana {
			t.Fatalf("%s must be a katakana row", row.ID)
		}
		total += len(row.Entries)
	}
	if total != 23 {
		t.Fatalf("extended entries = %d, want 23", total)
	}
	if len(RowsInGroup(ScriptHiragana, GroupExtended)) != 0 {
		t.Fatal("hiragana has no extended rows")
	}
	if CharToRow["ファ"] != "kata:ext-f" {
		t.Fatalf("CharToRow[ファ] = %q", CharToRow["ファ"])
	}
}

func TestExtendedSteps(t *testing.T) {
	kata := ProgressionStepsFor(ScriptKatakana)
	hira := ProgressionStepsFor(ScriptHiragana)
	if len(kata) != len(hira)+3 {
		t.Fatalf("katakana steps = %d, want %d", len(kata), len(hira)+3)
	}
	want := [][]string{{"kata:ext-t", "kata:ext-f"}, {"kata:ext-w", "kata:ext-sh"}, {"kata:ext-v", "kata:ext-ts"}}
	for i, step := range kata[len(hira):] {
		if len(step) != 2 || step[0] != want[i][0] || step[1] != want[i][1] {
			t.Fatalf("extended step %d = %v, want %v", i, step, want[i])
		}
	}
	for _, step := range hira {
		for _, id := range step {
			if row, _ := RowByID(id); row.Group == GroupExtended {
				t.Fatal("hiragana steps must not contain extended rows")
			}
		}
	}
}

func TestGroupsFor(t *testing.T) {
	has := func(gs []GroupInfo, g Group) bool {
		for _, info := range gs {
			if info.Group == g {
				return true
			}
		}
		return false
	}
	if has(GroupsFor(ScriptHiragana), GroupExtended) {
		t.Fatal("hiragana groups must not list Extended")
	}
	k := GroupsFor(ScriptKatakana)
	if !has(k, GroupExtended) || k[len(k)-1].Group != GroupExtended {
		t.Fatal("katakana groups end with Extended")
	}
	if len(GroupsFor(ScriptHiragana)) != 5 {
		t.Fatalf("hiragana groups = %d, want 5", len(GroupsFor(ScriptHiragana)))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 ./kanacore/`
Expected: compile errors (`GroupExtended`, `GroupsFor` undefined).

- [ ] **Step 3: Implement**

In `kanacore/kana_rows.go`:

Add the constant after `GroupYoonDakuon`:

```go
	GroupExtended   Group = "extended"
```

Append to the `Groups()` slice: `{GroupExtended, "Extended katakana"},`.

Add after `Groups()`:

```go
// GroupsFor returns the groups that have rows in the given script, in
// progression order.
func GroupsFor(s Script) []GroupInfo {
	out := make([]GroupInfo, 0, len(Groups()))
	for _, g := range Groups() {
		if len(RowsInGroup(s, g.Group)) > 0 {
			out = append(out, g)
		}
	}
	return out
}
```

Replace `var KatakanaRows = toKatakanaRows(HiraganaRows)` with:

```go
// extendedKatakanaRows are katakana-only combinations used mainly in
// loanwords. They have no hiragana counterpart.
var extendedKatakanaRows = withScript(ScriptKatakana, []KanaRow{
	{ID: katakanaPrefix + "ext-t", Label: "T/D (ティ)", Group: GroupExtended, Entries: []Entry{entry("ティ", "ti", "thi"), entry("ディ", "di", "dhi"), entry("トゥ", "tu", "twu"), entry("ドゥ", "du", "dwu")}},
	{ID: katakanaPrefix + "ext-f", Label: "F (ファ)", Group: GroupExtended, Entries: []Entry{entry("ファ", "fa"), entry("フィ", "fi"), entry("フェ", "fe"), entry("フォ", "fo")}},
	{ID: katakanaPrefix + "ext-w", Label: "W (ウィ)", Group: GroupExtended, Entries: []Entry{entry("ウィ", "wi"), entry("ウェ", "we"), entry("ウォ", "wo")}},
	{ID: katakanaPrefix + "ext-sh", Label: "SH/J/CH (シェ)", Group: GroupExtended, Entries: []Entry{entry("シェ", "she", "sye"), entry("ジェ", "je", "zye", "jye"), entry("チェ", "che", "tye", "cye")}},
	{ID: katakanaPrefix + "ext-v", Label: "V (ヴ)", Group: GroupExtended, Entries: []Entry{entry("ヴァ", "va"), entry("ヴィ", "vi"), entry("ヴ", "vu"), entry("ヴェ", "ve"), entry("ヴォ", "vo")}},
	{ID: katakanaPrefix + "ext-ts", Label: "TS (ツァ)", Group: GroupExtended, Entries: []Entry{entry("ツァ", "tsa"), entry("ツィ", "tsi"), entry("ツェ", "tse"), entry("ツォ", "tso")}},
})

// KatakanaRows mirrors HiraganaRows in katakana, followed by the extended
// katakana rows.
var KatakanaRows = append(toKatakanaRows(HiraganaRows), extendedKatakanaRows...)
```

Add after `hiraganaSteps`:

```go
// extendedSteps follow the generated katakana steps.
var extendedSteps = [][]string{
	{"kata:ext-t", "kata:ext-f"}, {"kata:ext-w", "kata:ext-sh"}, {"kata:ext-v", "kata:ext-ts"},
}
```

In `ProgressionStepsFor`, before `return steps`:

```go
	if s == ScriptKatakana {
		for _, step := range extendedSteps {
			steps = append(steps, append([]string(nil), step...))
		}
	}
```

and update its doc comment: "katakana steps are the hiragana steps with the katakana prefix, followed by the extended katakana steps."

- [ ] **Step 4: Run tests**

Run: `go test -count=1 ./kanacore/`
Expected: PASS (`TestRowsAreOrderedByGroup` and `TestProgressionStepsCoverRowsInOrder` pass unchanged).

Then fix the one fyne test that breaks: in `fyne/stats_test.go`, `TestPanelPathLineRendersSegmentsAndNext` expects `"2/15"` for katakana; change both `2/15` to `2/18` (the katakana path now has 18 steps).

Also move the settings change here so the hiragana tab never shows an empty heading: in `fyne/settings.go`, replace `for _, g := range kanacore.Groups() {` in the per-script loop with `for _, g := range kanacore.GroupsFor(info.Script) {`.

Run `go test -count=1 ./...`. Any other failure: stop and report.

- [ ] **Step 5: Commit**

```bash
gofmt -l .
git add kanacore/kana_rows.go kanacore/kana_test.go fyne/stats_test.go fyne/settings.go
git commit -m "feat(kanacore): add extended katakana learning-path group

Adds the ティ/ファ/ウィ/シェ/ヴ/ツァ rows as a katakana-only group with three extra steps after the generated katakana path, and GroupsFor to list only the groups a script has."
```


---

### Task 2: `KanaUnits`

**Files:**
- Create: `kanacore/words.go`
- Test: `kanacore/words_test.go`

- [ ] **Step 1: Write the failing test**

```go
package kanacore

import (
	"reflect"
	"testing"
)

func TestKanaUnits(t *testing.T) {
	ok := []struct {
		word string
		want []string
	}{
		{"きゃく", []string{"きゃ", "く"}},
		{"パーティー", []string{"パ", "ー", "ティ", "ー"}},
		{"ヴァイオリン", []string{"ヴァ", "イ", "オ", "リ", "ン"}},
		{"きって", []string{"き", "っ", "て"}},
		{"まっちゃ", []string{"ま", "っ", "ちゃ"}},
		{"こーー", []string{"こ", "ー", "ー"}},
		{"すげー", []string{"す", "げ", "ー"}},
	}
	for _, c := range ok {
		got, valid := KanaUnits(c.word)
		if !valid || !reflect.DeepEqual(got, c.want) {
			t.Errorf("KanaUnits(%s) = %v, %v; want %v", c.word, got, valid, c.want)
		}
	}
	bad := []string{
		"",     // empty
		"先生",   // kanji
		"ゐ",    // unsupported kana
		"ァ",    // lone small kana
		"ーあ",   // leading long mark
		"あっ",   // trailing sokuon
		"あっあ",  // sokuon before a vowel
		"あっん",  // sokuon before ん
		"あっー",  // sokuon before ー (also covers ー after っ)
		"っか",   // leading sokuon
		"かんー",  // long mark after ん
		"kana", // latin
	}
	for _, w := range bad {
		if _, valid := KanaUnits(w); valid {
			t.Errorf("KanaUnits(%q) should fail", w)
		}
	}
}
```

Run `gofmt -w kanacore/words_test.go` after writing the file (comment alignment).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -count=1 -run TestKanaUnits ./kanacore/`
Expected: compile error, `KanaUnits` undefined.

- [ ] **Step 3: Implement**

Create `kanacore/words.go`:

```go
package kanacore

import "strings"

// Units that only occur inside words.
const (
	sokuonHira = "っ"
	sokuonKata = "ッ"
	longMark   = "ー"
)

// unitSpellings maps every kana known to a row to its canonical romaji
// followed by its alternatives.
var unitSpellings map[string][]string

func init() {
	unitSpellings = make(map[string][]string)
	for _, row := range AllKanaRows {
		for _, e := range row.Entries {
			unitSpellings[e.Char] = append([]string{e.Romaji}, e.Alt...)
		}
	}
}

func isSokuon(u string) bool { return u == sokuonHira || u == sokuonKata }
func isN(u string) bool      { return u == "ん" || u == "ン" }

func isVowel(b byte) bool { return strings.IndexByte("aiueo", b) >= 0 }

// KanaUnits splits a word into units: two-character combinations known to a
// row (きゃ, ティ), single kana, っ/ッ and ー. It reports false when a
// character cannot be placed or the word contains a sequence that cannot be
// typed (a leading, trailing or misplaced っ, a ー with no vowel before it).
func KanaUnits(word string) ([]string, bool) {
	runes := []rune(word)
	if len(runes) == 0 {
		return nil, false
	}
	var units []string
	for i := 0; i < len(runes); {
		if i+1 < len(runes) {
			if pair := string(runes[i : i+2]); unitSpellings[pair] != nil {
				units = append(units, pair)
				i += 2
				continue
			}
		}
		single := string(runes[i])
		if unitSpellings[single] == nil && !isSokuon(single) && single != longMark {
			return nil, false
		}
		units = append(units, single)
		i++
	}
	for i, u := range units {
		switch {
		case isSokuon(u):
			if i == 0 || i+1 >= len(units) {
				return nil, false
			}
			next := units[i+1]
			sp := unitSpellings[next]
			if sp == nil || isN(next) || isVowel(sp[0][0]) {
				return nil, false
			}
		case u == longMark:
			// ー after っ is already rejected by the っ rule above.
			if i == 0 || isN(units[i-1]) {
				return nil, false
			}
		}
	}
	return units, true
}
```

Note: `sp == nil` covers っ before っ or ー (neither has row spellings).

- [ ] **Step 4: Run tests**

Run: `go test -count=1 ./kanacore/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -l .
git add kanacore/words.go kanacore/words_test.go
git commit -m "feat(kanacore): split words into kana units

KanaUnits breaks a kana word into the units the reading match works on, including っ and ー, and rejects words that cannot be typed."
```

---

### Task 3: `ReadingMatches` and `Romaji`

**Files:**
- Modify: `kanacore/words.go`
- Test: `kanacore/words_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `kanacore/words_test.go`:

```go
func TestReadingMatches(t *testing.T) {
	yes := map[string][]string{
		"きって":    {"kitte", " KITTE "},
		"まっちゃ":   {"matcha", "maccha", "mattya"},
		"コーヒー":   {"koohii", "ko-hi-", "ko-hii"},
		"でんしゃ":   {"densha", "densya", "dennsha", "den'sha"},
		"きんえん":   {"kinen", "kin'en", "kinnen"},
		"パーティー":  {"paatii", "paathii", "pa-ti-"},
		"とうきょう":  {"toukyou"},
		"ふじさん":   {"fujisan", "huzisan"},
		"こーー":    {"kooo", "ko--", "koo-"},
		"ヴァイオリン": {"vaiorin"},
		"はい":     {"hai"},
	}
	for word, inputs := range yes {
		for _, in := range inputs {
			if !ReadingMatches(word, in) {
				t.Errorf("ReadingMatches(%s, %q) = false, want true", word, in)
			}
		}
	}
	no := map[string][]string{
		"とうきょう": {"tokyo", "toukyo"},
		"きって":   {"kite", "kitt", "kittee", ""},
		"はい":    {"ha", "haii", "hi"},
		"コーヒー":  {"kohi", "kouhii"},
		"先生":    {"sensei"},
	}
	for word, inputs := range no {
		for _, in := range inputs {
			if ReadingMatches(word, in) {
				t.Errorf("ReadingMatches(%s, %q) = true, want false", word, in)
			}
		}
	}
}

func TestRomaji(t *testing.T) {
	cases := map[string]string{
		"きって":    "kitte",
		"まっちゃ":   "matcha",
		"コーヒー":   "koohii",
		"きんえん":   "kin'en",
		"きんよう":   "kin'you",
		"でんしゃ":   "densha",
		"パーティー":  "paatii",
		"とうきょう":  "toukyou",
		"ヴァイオリン": "vaiorin",
		"先生":     "",
	}
	for word, want := range cases {
		if got := Romaji(word); got != want {
			t.Errorf("Romaji(%s) = %q, want %q", word, got, want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestReadingMatches|TestRomaji' ./kanacore/`
Expected: compile error, `ReadingMatches` and `Romaji` undefined.

- [ ] **Step 3: Implement**

Append to `kanacore/words.go`:

```go
// spellingsOf returns the accepted spellings of a normal unit; ん also
// accepts n'.
func spellingsOf(u string) []string {
	sp := unitSpellings[u]
	if isN(u) {
		return append(append([]string(nil), sp...), "n'")
	}
	return sp
}

// lastVowel returns the final vowel of a spelling, or 0 if it ends in a
// consonant or apostrophe.
func lastVowel(s string) byte {
	if s == "" || !isVowel(s[len(s)-1]) {
		return 0
	}
	return s[len(s)-1]
}

// ReadingMatches reports whether input is a valid romaji reading of word:
// each unit by one of its spellings, っ as a doubled first letter (or t
// before ch), ー as the previous vowel or "-". Input is trimmed and
// lower-cased first.
func ReadingMatches(word, input string) bool {
	units, ok := KanaUnits(word)
	if !ok {
		return false
	}
	in := strings.ToLower(strings.TrimSpace(input))
	if in == "" {
		return false
	}
	type state struct {
		unit, pos int
		vowel     byte
	}
	failed := make(map[state]bool)
	var match func(i, pos int, vowel byte) bool
	match = func(i, pos int, vowel byte) bool {
		if i == len(units) {
			return pos == len(in)
		}
		st := state{i, pos, vowel}
		if failed[st] {
			return false
		}
		rest := in[pos:]
		u := units[i]
		switch {
		case u == longMark:
			if strings.HasPrefix(rest, "-") && match(i+1, pos+1, vowel) {
				return true
			}
			if vowel != 0 && rest != "" && rest[0] == vowel && match(i+1, pos+1, vowel) {
				return true
			}
		case isSokuon(u):
			for _, s := range spellingsOf(units[i+1]) {
				prefixes := []string{s[:1]}
				if strings.HasPrefix(s, "ch") {
					prefixes = append(prefixes, "t")
				}
				for _, p := range prefixes {
					if strings.HasPrefix(rest, p+s) && match(i+2, pos+len(p)+len(s), lastVowel(s)) {
						return true
					}
				}
			}
		default:
			for _, s := range spellingsOf(u) {
				if strings.HasPrefix(rest, s) && match(i+1, pos+len(s), lastVowel(s)) {
					return true
				}
			}
		}
		failed[st] = true
		return false
	}
	return match(0, 0, 0)
}

// Romaji returns the canonical romaji of word, or "" if it cannot be split
// into units: っ doubles the next consonant (t before ch), ー repeats the
// previous vowel, ん is n' before a vowel or y.
func Romaji(word string) string {
	units, ok := KanaUnits(word)
	if !ok {
		return ""
	}
	var b strings.Builder
	var vowel byte
	for i, u := range units {
		switch {
		case u == longMark:
			b.WriteByte(vowel)
		case isSokuon(u):
			next := unitSpellings[units[i+1]][0]
			if strings.HasPrefix(next, "ch") {
				b.WriteByte('t')
			} else {
				b.WriteByte(next[0])
			}
		case isN(u):
			b.WriteString("n")
			if i+1 < len(units) {
				if sp := unitSpellings[units[i+1]]; sp != nil && (isVowel(sp[0][0]) || sp[0][0] == 'y') {
					b.WriteString("'")
				}
			}
			vowel = 0
		default:
			s := unitSpellings[u][0]
			b.WriteString(s)
			vowel = lastVowel(s)
		}
	}
	return b.String()
}
```

Note on ー after a unit typed with `-`: `vowel` is passed through unchanged for ー, so the nearest preceding non-ー unit's vowel applies (`koo-` and `ko--` both match こーー).

- [ ] **Step 4: Run tests**

Run: `go test -count=1 ./kanacore/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -l .
git add kanacore/words.go kanacore/words_test.go
git commit -m "feat(kanacore): match romaji readings of kana words

ReadingMatches accepts every spelling a row allows plus doubled consonants for っ, a repeated vowel or dash for ー and n, nn or n' for ん. Romaji gives the canonical spelling shown as a solution."
```

---

### Task 4: Settings and status panel

**Files:**
- Modify: `fyne/settings.go:44`
- Modify: `fyne/stats.go` (path line ~140-165, grid ~170-195, `Update` group loop ~311-330, `rowShortLabel` ~62)
- Test: `fyne/stats_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `fyne/stats_test.go`:

```go
func TestPanelExtendedBlock(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.Update(panelSnap([]string{"kata:ext-t"}, both))
	if !p.cells["ティ"].Visible() || p.cells["ファ"].Visible() {
		t.Fatal("selected extended row visible, unselected one hidden")
	}
	if !p.groupGrids[kanacore.GroupExtended].Visible() {
		t.Fatal("extended grid visible when an extended row is selected")
	}
	if w := p.groupGrids[kanacore.GroupExtended].MinSize().Width; w != 288 {
		t.Fatalf("extended grid width = %v, want 288", w)
	}
	if h := p.groupGrids[kanacore.GroupExtended].MinSize().Height; h != cellH {
		t.Fatalf("extended grid height = %v, want one row (%v)", h, cellH)
	}
	p.Update(panelSnap([]string{"kata:ext-t"}, []kanacore.Script{kanacore.ScriptHiragana}))
	if p.groupGrids[kanacore.GroupExtended].Visible() {
		t.Fatal("extended grid hidden while katakana is inactive")
	}
}

func TestPanelExtendedCellsInEntryOrder(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	cells := p.rowCells["kata:ext-t"]
	want := []string{"ティ", "ディ", "トゥ", "ドゥ"}
	if len(cells) != len(want) {
		t.Fatalf("ext-t cells = %d, want %d", len(cells), len(want))
	}
	for i, c := range cells {
		if c != p.cells[want[i]] {
			t.Fatalf("ext-t cell %d is not %s", i, want[i])
		}
	}
}

func TestPathLineFitsWithExtendedSteps(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	if w := p.pathLines[kanacore.ScriptKatakana].MinSize().Width; w > 288 {
		t.Fatalf("katakana path line width = %v, must fit 288", w)
	}
}

func TestRowShortLabelExtended(t *testing.T) {
	cases := map[string]string{"kata:ext-t": "t", "kata:ext-sh": "sh", "kata:ext-ts": "ts", "kata:ext-v": "v"}
	for id, want := range cases {
		if got := rowShortLabel(id); got != want {
			t.Errorf("rowShortLabel(%s) = %q, want %q", id, got, want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestPanelExtended|TestPathLineFits|TestRowShortLabelExtended' ./fyne/`
Expected: FAIL (no extended grid / cells, path line too wide, label "ext-t").

- [ ] **Step 3: Implement**

(The settings change was done in Task 1; `TestGroupsFor` covers that the hiragana tab gets no extended heading.)

`fyne/stats.go`:

1. `rowShortLabel`: before `default`, add

```go
	case "ext-t", "ext-f", "ext-w", "ext-sh", "ext-v", "ext-ts":
		return strings.TrimPrefix(kanacore.BaseRowID(rowID), "ext-")
```

(`strings` is already imported.)

2. Path line: compute the segment width from the step count so the line fits 288 px. Replace
`widths = append(widths, 10)` with `widths = append(widths, segW)` and before the segment
loop add:

```go
		// Fit label (62), count (30) and the gaps into the 288px grid width.
		segW := float32(math.Floor(float64((288 - 62 - 30 - 2*float32(steps+1)) / float32(steps))))
		if segW > 10 {
			segW = 10
		}
```

(`math` is already imported.) 15 steps keep 10 px; 18 steps get 8 px.

3. Paired grid: replace `for _, g := range kanacore.Groups() {` in the constructor with
`for _, g := range kanacore.GroupsFor(kanacore.ScriptHiragana) {`. After that loop, add the
extended block, laid out with the same `gridColumns()` so it lines up under "カタカナ"; the
hiragana half and the spacer are transparent fillers and the katakana half holds the entries
in order:

```go
	// Extended katakana: no hiragana half; entries in order, not by vowel.
	var ext []fyne.CanvasObject
	for _, row := range kanacore.RowsInGroup(kanacore.ScriptKatakana, kanacore.GroupExtended) {
		lbl := smallText(rowShortLabel(row.ID), 11)
		lbl.Alignment = fyne.TextAlignTrailing
		lbl.Color = panelSubColor
		p.rowLabels[row.ID] = lbl
		ext = append(ext, lbl)
		filler := func() fyne.CanvasObject {
			f := canvas.NewRectangle(color.Transparent)
			p.rowFillers[row.ID] = append(p.rowFillers[row.ID], f)
			return f
		}
		for i := 0; i < 6; i++ { // empty hiragana half and spacer
			ext = append(ext, filler())
		}
		for i := 0; i < 5; i++ {
			if i < len(row.Entries) {
				cell := newKanaCell(row.Entries[i].Char)
				p.cells[row.Entries[i].Char] = cell
				p.rowCells[row.ID] = append(p.rowCells[row.ID], cell)
				ext = append(ext, cell)
			} else {
				ext = append(ext, filler())
			}
		}
	}
	extGrid := container.New(gridColumns(), ext...)
	p.groupGrids[kanacore.GroupExtended] = extGrid
	grids.Add(extGrid)
```

4. `Update`: replace `for _, g := range kanacore.Groups() {` with
`for _, g := range kanacore.GroupsFor(kanacore.ScriptHiragana) {`, and after that loop add:

```go
	extVisible := false
	for _, row := range kanacore.RowsInGroup(kanacore.ScriptKatakana, kanacore.GroupExtended) {
		on := snap.ActiveScripts[kanacore.ScriptKatakana] && snap.SelectedRows[row.ID]
		extVisible = extVisible || on
		setVisible(p.rowLabels[row.ID], on)
		for _, f := range p.rowFillers[row.ID] {
			setVisible(f, on)
		}
		for _, c := range p.rowCells[row.ID] {
			setVisible(c, on)
		}
	}
	setVisible(p.groupGrids[kanacore.GroupExtended], extVisible)
```

`fixedColumns` skips a row only when all of its objects are hidden, so the fillers (keyed by the full `kata:ext-…` ID, which cannot collide with base IDs) are toggled with the row. Also update the stale comment above the path-line loop ("15 segments") to "one segment per step".

- [ ] **Step 4: Run tests**

Run: `go test -count=1 ./...`
Expected: PASS, including the existing panel tests (`TestPanelHidesEmptyGroupsAndKeepsWidth`).

- [ ] **Step 5: Commit**

```bash
gofmt -l .
git add fyne/stats.go fyne/stats_test.go
git commit -m "feat(fyne): show extended katakana in settings and status panel

The settings tabs list only the groups a script has, the status panel shows the extended rows as a katakana-only block in entry order, and the path line narrows its segments to fit the longer katakana path."
```

---

### Task 5: Docs

**Files:**
- Modify: `README.md`, `PLAN.md`

- [ ] **Step 1: Update docs**

- `README.md`: line ~16 "104 Hiragana and 104 Katakana" becomes 104 hiragana and 127 katakana (including 23 extended); the group list (~line 68) adds "Extended katakana". Where the character sets / learning path are described, add the extended katakana group (ティ, ファ, ウィ, シェ, ヴ, ツァ rows, three extra katakana steps after Yōon + Dakuten). In the architecture list, note `kanacore` also splits and matches kana words (`KanaUnits`, `ReadingMatches`, `Romaji`) for the upcoming vocabulary trainer.
- `PLAN.md`: tick "Extended katakana for loanwords" (`[x]`) and change the small っ/ー item to "- [x] Small っ/ッ (sokuon) and the long vowel mark ー in word readings (`kanacore.ReadingMatches`)".

- [ ] **Step 2: Commit**

```bash
git add README.md PLAN.md
git commit -m "docs: describe extended katakana and word readings"
```

- [ ] **Step 3: Final check**

Run: `gofmt -l . && go vet ./... && go test -count=1 ./...`
Expected: no gofmt output, vet clean, all packages PASS.

Manual check (for the user): with katakana active and the katakana path complete, the extended steps unlock with the intro dialog; ティ tiles accept "ti" and "thi"; the panel shows the extended block under カタカナ; the hiragana settings tab has no "Extended katakana" heading.
