# Vocabulary Trainer Stage 2: Word Data Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `vocab` package that loads the bundled JLPT word list and private word lists, merges them into one index per reading and filters playable words, plus the `cmd/vocabgen` tool that produces the bundled list from JMdict and the JLPT mapping.

**Architecture:** `vocab` depends only on `kanacore` (for `KanaUnits`). Loaders return `Word` values; `NewIndex` merges them into `Entry` values keyed by reading; `Playable` filters entries by a `known` callback and a source `Filter`. `cmd/vocabgen` streams the JMdict XML, joins it with the JLPT CSVs by JMdict sequence number and writes a deterministic gzip JSON file that `vocab` embeds. No UI in this stage.

**Tech Stack:** Go 1.25 standard library (`encoding/csv`, `encoding/json`, `encoding/xml`, `compress/gzip`, `embed`).

**Spec:** `docs/superpowers/specs/2026-10-12-vocab-trainer-design.md`, section "Stage 2" (including "Meaning lists").

**Data sources (verified 2026-10-13):**
- JMdict full multilingual: `http://ftp.edrdg.org/pub/Nihongo/JMdict.gz` (~22 MB), EDRDG, CC BY-SA 4.0.
- JLPT mapping: `https://github.com/stephenmk/yomitan-jlpt-vocab`, folder `original_data/`, files `n1.csv` … `n5.csv` with header `jmdict_seq,kana,kanji,waller_definition` (8,293 rows). Licence CC BY-SA 4.0; levels from Jonathan Waller (tanos.co.uk), CC BY. The same `jmdict_seq` can appear in several files and with several `kana` (e.g. アイデア/アイディア).

**Conventions:** `gofmt -l .` empty, `go vet ./...` clean, `go test -count=1 ./...` passing before each commit. Commit messages `type(scope): summary` plus a short prose paragraph; no Co-Authored-By, no AI/assistant/session mentions.

---

## Amendments after plan review (override the task code below)

The plan's code was built and run against the real data (8,293 JLPT rows → 7,031 words, 2 dropped: あっ, レクリェーション; output 1.07 MB; non-strict `encoding/xml` handles the real DTD). Apply these changes while implementing:

1. **Task 2 – BOM:** Go source must not contain a literal U+FEFF. Write it as the escape `\ufeff` (backslash-u) in both `csvlist_test.go` and `csvlist.go`.
2. **Task 2 – read errors:** in the `readCSV` loop, only a `*csv.ParseError` (`errors.As`) is a skipped row; any other error records a problem and `break`s (otherwise the loop spins on a failing reader). An empty `kana` cell is reported as `line N: empty kana`.
3. **Task 3 – no `mergeWithin`:** do not merge within a list in `LoadDir` (it would lose the second lesson of a repeated reading). `List.Words` keeps every row; `NewIndex` already merges meanings and collects lessons per list. In `TestLoadDir`, the `minna` list then has 2 words; add an index check that `NewIndex(nil, lists)` gives one せんせい entry with meanings `["Lehrer", "Lehrerin"]`. Delete `mergeWithin` and its call.
4. **Task 3 – names:** compare base names case-insensitively (`strings.ToLower`) when counting collisions and when checking the reserved `jmdict` (so `Jmdict.csv` → `Jmdict (csv)`, and `a.CSV` + `a.csv` both get suffixes). If two names still collide after suffixing (`a.CSV`, `a.csv`), append ` 2`, ` 3` … in directory order. In `addJSONMeaning`, iterate object keys in sorted order.
5. **Task 5 – test file:** delete the test's `outWord` type (it clashes with `build.go`'s; same package) and add `"sort"` to the test's import block. Ignore the paragraph after `build.go` that mentions "testWord".
6. **Task 5 – empty sequence numbers:** the real `n1.csv` has 14 rows with an empty `jmdict_seq` (single-kanji readings like い 依, お 於). Skip rows with `Seq == ""` in `run`, count them in a new `Stats.NoSeq`, and print it in `Stats.String`. Add such a row to the fixture `n5.csv` (`,ぬ,,test`) and assert `stats.NoSeq == 1` and that ぬ is not written.
7. **Task 5 – gloss cleanup:** some non-English glosses carry markup. In `readJMdict`, clean each gloss before use: strip a leading `^\d+\)\s*`; drop it if it then starts with `(`, `{` or `-`, contains any Hiragana, Katakana or CJK ideograph (U+3040–U+30FF, U+4E00–U+9FFF), or is shorter than 2 runes. Add fixture glosses `<gloss xml:lang="rus">1) учитель</gloss>` and `<gloss xml:lang="rus">{～をする} работать</gloss>` to せんせい and assert `ru == ["учитель"]`.
8. **Task 5 – merge cap:** `maxGlossesMerged` is per language (as implemented). The spec is corrected to say so.
9. **Task 6 – numbers:** expect about 7,000 words (not 8,000) and roughly 1 MB. Stop and report only if the file exceeds 5 MB or fewer than 5,000 words are written.
10. **Reading fallback:** a JLPT row with an empty `kana` is dropped (the real data has none). Rows always carry their reading, so the spec's "first prioritised reading" fallback is not implemented; the spec is corrected.

---

## File Structure

- Create `vocab/vocab.go`: `Meanings`, `Word`, `List`, `SplitMeanings`, `Meanings.Add/For/Display`, `parseLevel`.
- Create `vocab/csvlist.go`: CSV/TSV loader.
- Create `vocab/jsonlist.go`: JSON loader.
- Create `vocab/dir.go`: `LoadDir`, list naming, `mergeWithin`.
- Create `vocab/index.go`: `Entry`, `Index`, `NewIndex`, `Filter`, `Playable`.
- Create `vocab/bundled.go`: `go:embed` of `data/words.json.gz`, `Bundled()`.
- Create `cmd/vocabgen/main.go`, `cmd/vocabgen/jmdict.go`, `cmd/vocabgen/jlpt.go`, `cmd/vocabgen/build.go` and tests with small fixtures in `cmd/vocabgen/testdata/`.
- Create `vocab/data/words.json.gz` (generated), `vocab/data/ATTRIBUTION.md`.
- Modify `.gitignore` (`words/`), `README.md`, `PLAN.md`.

Tests live next to each file (`vocab/*_test.go`, `cmd/vocabgen/*_test.go`).

---

### Task 1: Model and meaning lists

**Files:** Create `vocab/vocab.go`, `vocab/vocab_test.go`

- [ ] **Step 1: Write the failing test** — `vocab/vocab_test.go`:

```go
package vocab

import (
	"reflect"
	"testing"
)

func TestSplitMeanings(t *testing.T) {
	cases := map[string][]string{
		"Lehrer; Lehrerin":    {"Lehrer", "Lehrerin"},
		" teacher ":           {"teacher"},
		"a;;a; b ;":           {"a", "b"},
		"":                    nil,
		" ; ":                 nil,
		"Brücke;Essstäbchen":  {"Brücke", "Essstäbchen"},
	}
	for in, want := range cases {
		if got := SplitMeanings(in); !reflect.DeepEqual(got, want) {
			t.Errorf("SplitMeanings(%q) = %#v, want %#v", in, got, want)
		}
	}
}

func TestMeaningsAddAndFor(t *testing.T) {
	m := Meanings{}
	m.Add("de", "Lehrer", "Lehrerin")
	m.Add("de", "Lehrer", "Meister")
	m.Add("en", "teacher")
	if got := m["de"]; !reflect.DeepEqual(got, []string{"Lehrer", "Lehrerin", "Meister"}) {
		t.Fatalf("de = %v", got)
	}
	if got := m.For("de"); got[0] != "Lehrer" {
		t.Fatalf("For(de) = %v", got)
	}
	if got := m.For("fr"); !reflect.DeepEqual(got, []string{"teacher"}) {
		t.Fatalf("For(fr) falls back to en, got %v", got)
	}
	g := Meanings{}
	g.Add("", "Lehrer")
	g.Add("en", "teacher")
	if got := g.For("de"); !reflect.DeepEqual(got, []string{"Lehrer"}) {
		t.Fatalf("generic meaning wins over en for an unlisted language, got %v", got)
	}
	only := Meanings{}
	only.Add("sv", "lärare")
	if got := only.For("de"); !reflect.DeepEqual(got, []string{"lärare"}) {
		t.Fatalf("any language as last resort, got %v", got)
	}
	if got := (Meanings{}).For("de"); got != nil {
		t.Fatalf("empty meanings = %v", got)
	}
	if got := m.Display("de"); got != "Lehrer; Lehrerin; Meister" {
		t.Fatalf("Display = %q", got)
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]int{"5": 5, "N5": 5, "n4": 4, " 3 ": 3, "": 0, "6": 0, "x": 0, "N0": 0}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %d, want %d", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 ./vocab/` — compile error (package/functions missing).

- [ ] **Step 3: Implement** `vocab/vocab.go`:

```go
// Package vocab loads word lists and selects words a learner can read.
package vocab

import (
	"sort"
	"strconv"
	"strings"
)

// Meanings maps an ISO 639-1 language to its meanings in first-seen order.
// The key "" holds meanings given without a language; they apply to every
// language that has none of its own.
type Meanings map[string][]string

// Word is one entry of a word list.
type Word struct {
	Kana     string   // reading, e.g. "せんせい"
	Meanings Meanings // language → alternatives
	JLPT     int      // 5…1, 0 = none
	Source   string   // "jmdict" or a list name
	Lesson   string   // optional, private lists only
}

// List is one private word list file.
type List struct {
	Name     string
	Words    []Word
	Problems []string // at most maxProblems messages
	Skipped  int      // rows skipped, including those without a message
}

// BundledSource is the Source of bundled words and a reserved list name.
const BundledSource = "jmdict"

const maxProblems = 10

// SplitMeanings splits a field on ";", trims items and drops empty and
// duplicate ones, keeping first-seen order.
func SplitMeanings(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ";") {
		p := strings.TrimSpace(part)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// Add appends items for lang, skipping ones already present.
func (m Meanings) Add(lang string, items ...string) {
	for _, it := range items {
		dup := false
		for _, have := range m[lang] {
			if have == it {
				dup = true
				break
			}
		}
		if !dup {
			m[lang] = append(m[lang], it)
		}
	}
}

// Merge adds every language of other to m.
func (m Meanings) Merge(other Meanings) {
	for lang, items := range other {
		m.Add(lang, items...)
	}
}

// For returns the meanings to show for lang: lang itself, then the
// language-less meanings, then English, then the first language in
// alphabetical order. Nil if there are none.
func (m Meanings) For(lang string) []string {
	for _, k := range []string{lang, "", "en"} {
		if len(m[k]) > 0 {
			return m[k]
		}
	}
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if len(v) > 0 {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	return m[keys[0]]
}

// Display joins For(lang) with "; ".
func (m Meanings) Display(lang string) string {
	return strings.Join(m.For(lang), "; ")
}

// parseLevel reads a JLPT level written as "5" or "N5"; anything else is 0.
func parseLevel(s string) int {
	s = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "N")
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 5 {
		return 0
	}
	return n
}
```

- [ ] **Step 4: Run** `go test -count=1 ./vocab/` — PASS. Then `gofmt -w vocab/` (test map alignment).

- [ ] **Step 5: Commit**

```bash
git add vocab/vocab.go vocab/vocab_test.go
git commit -m "feat(vocab): add the word model and meaning lists

Words carry per-language lists of meanings; fields separate alternatives with a semicolon, and lookup falls back from the chosen language to language-less meanings, English and any other language."
```

---

### Task 2: CSV/TSV loader

**Files:** Create `vocab/csvlist.go`, `vocab/csvlist_test.go`

- [ ] **Step 1: Write the failing test** — `vocab/csvlist_test.go`:

```go
package vocab

import (
	"reflect"
	"strings"
	"testing"
)

func TestReadCSVSeparators(t *testing.T) {
	inputs := map[string]string{
		"comma":     "kana,meaning_de,meaning_en,lesson\nせんせい,Lehrer; Lehrerin,teacher,1\n",
		"semicolon": "kana;meaning_de;meaning_en;lesson\nせんせい;\"Lehrer; Lehrerin\";teacher;1\n",
		"tab":       "kana\tmeaning_de\tmeaning_en\tlesson\nせんせい\tLehrer; Lehrerin\tteacher\t1\n",
		"bom":       "\ufeffkana,meaning_de,meaning_en,lesson\nせんせい,Lehrer; Lehrerin,teacher,1\n",
		"case":      "Kana,Meaning_DE,MEANING_EN,Lesson\nせんせい,Lehrer; Lehrerin,teacher,1\n",
	}
	for name, in := range inputs {
		l := readCSV("minna", strings.NewReader(in))
		if len(l.Words) != 1 || len(l.Problems) != 0 {
			t.Fatalf("%s: words=%d problems=%v", name, len(l.Words), l.Problems)
		}
		w := l.Words[0]
		if w.Kana != "せんせい" || w.Lesson != "1" || w.Source != "minna" {
			t.Fatalf("%s: word = %+v", name, w)
		}
		if !reflect.DeepEqual(w.Meanings["de"], []string{"Lehrer", "Lehrerin"}) || !reflect.DeepEqual(w.Meanings["en"], []string{"teacher"}) {
			t.Fatalf("%s: meanings = %v", name, w.Meanings)
		}
	}
}

func TestReadCSVGenericMeaningAndLevel(t *testing.T) {
	in := "kana,meaning,jlpt\nねこ,Katze,N5\nいぬ,Hund,4\n"
	l := readCSV("x", strings.NewReader(in))
	if len(l.Words) != 2 {
		t.Fatalf("words = %d, problems %v", len(l.Words), l.Problems)
	}
	if got := l.Words[0].Meanings[""]; !reflect.DeepEqual(got, []string{"Katze"}) {
		t.Fatalf("generic meaning = %v", got)
	}
	if l.Words[0].JLPT != 5 || l.Words[1].JLPT != 4 {
		t.Fatalf("levels = %d, %d", l.Words[0].JLPT, l.Words[1].JLPT)
	}
}

func TestReadCSVProblems(t *testing.T) {
	l := readCSV("x", strings.NewReader("word,meaning\nねこ,Katze\n"))
	if len(l.Words) != 0 || len(l.Problems) != 1 || !strings.Contains(l.Problems[0], "kana") {
		t.Fatalf("missing kana column: %+v", l)
	}
	l = readCSV("x", strings.NewReader("kana,meaning\n先生,Lehrer\nねこ,Katze\n,leer\n"))
	if len(l.Words) != 1 || l.Skipped != 2 {
		t.Fatalf("words=%d skipped=%d", len(l.Words), l.Skipped)
	}
	if !strings.Contains(l.Problems[0], "line 2") || !strings.Contains(l.Problems[0], "先生") {
		t.Fatalf("problem = %q", l.Problems[0])
	}
	var b strings.Builder
	b.WriteString("kana\n")
	for i := 0; i < 15; i++ {
		b.WriteString("x\n")
	}
	l = readCSV("x", strings.NewReader(b.String()))
	if len(l.Problems) != maxProblems || l.Skipped != 15 {
		t.Fatalf("problems=%d skipped=%d", len(l.Problems), l.Skipped)
	}
	l = readCSV("x", strings.NewReader(""))
	if len(l.Words) != 0 || len(l.Problems) != 1 {
		t.Fatalf("empty file: %+v", l)
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 -run TestReadCSV ./vocab/` — compile error.

- [ ] **Step 3: Implement** `vocab/csvlist.go`:

```go
package vocab

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"kana/kanacore"
)

// readCSV reads a word list with a header row. The separator (tab,
// semicolon or comma) is taken from the header line.
func readCSV(name string, r io.Reader) List {
	l := List{Name: name}
	br := bufio.NewReader(r)
	header, err := br.ReadString('\n')
	if err != nil && header == "" {
		l.problem("file is empty")
		return l
	}
	header = strings.TrimPrefix(header, "\ufeff")
	sep := ','
	switch {
	case strings.Contains(header, "\t"):
		sep = '\t'
	case strings.Contains(header, ";"):
		sep = ';'
	}
	cr := csv.NewReader(io.MultiReader(strings.NewReader(header), br))
	cr.Comma = sep
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cols, err := cr.Read()
	if err != nil {
		l.problem(fmt.Sprintf("header: %v", err))
		return l
	}
	idx := map[string]int{}
	for i, c := range cols {
		idx[strings.ToLower(strings.TrimSpace(c))] = i
	}
	kanaCol, ok := idx["kana"]
	if !ok {
		l.problem(`no "kana" column in the header`)
		return l
	}
	field := func(rec []string, col string) string {
		if i, ok := idx[col]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		line, _ := cr.FieldPos(0)
		if err != nil {
			l.skip(fmt.Sprintf("line %d: %v", line, err))
			continue
		}
		kana := ""
		if kanaCol < len(rec) {
			kana = strings.TrimSpace(rec[kanaCol])
		}
		if _, ok := kanacore.KanaUnits(kana); !ok {
			l.skip(fmt.Sprintf("line %d: %s is not kana", line, kana))
			continue
		}
		w := Word{Kana: kana, Meanings: Meanings{}, Source: name,
			Lesson: field(rec, "lesson"), JLPT: parseLevel(field(rec, "jlpt"))}
		for col, i := range idx {
			if i >= len(rec) {
				continue
			}
			switch {
			case col == "meaning":
				w.Meanings.Add("", SplitMeanings(rec[i])...)
			case strings.HasPrefix(col, "meaning_"):
				lang := strings.TrimPrefix(col, "meaning_")
				w.Meanings.Add(lang, SplitMeanings(rec[i])...)
			}
		}
		l.Words = append(l.Words, w)
	}
	return l
}

// problem records a message without counting a skipped row.
func (l *List) problem(msg string) {
	if len(l.Problems) < maxProblems {
		l.Problems = append(l.Problems, msg)
	}
}

// skip counts a skipped row and records its message.
func (l *List) skip(msg string) {
	l.Skipped++
	l.problem(msg)
}
```

Note: an empty `kana` cell fails `KanaUnits` and is reported as " is not kana"; that is acceptable, but if the message reads badly, special-case empty to "line N: empty kana". The test only checks counts for it.

- [ ] **Step 4: Run** `go test -count=1 ./vocab/` — PASS.

- [ ] **Step 5: Commit**

```bash
git add vocab/csvlist.go vocab/csvlist_test.go
git commit -m "feat(vocab): read CSV and TSV word lists

The separator is detected from the header, a BOM is ignored, meaning columns may hold several alternatives, and rows that are not kana are skipped and reported."
```

---

### Task 3: JSON loader and `LoadDir`

**Files:** Create `vocab/jsonlist.go`, `vocab/dir.go`, `vocab/dir_test.go`; modify `.gitignore`

- [ ] **Step 1: Write the failing tests** — `vocab/dir_test.go`:

```go
package vocab

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadJSON(t *testing.T) {
	in := `[
	 {"kana": "せんせい", "meaning": {"de": "Lehrer; Lehrerin", "en": ["teacher", "master"]}, "lesson": 1, "jlpt": "N5"},
	 {"kana": "ねこ", "meaning": "Katze"},
	 {"kana": "猫", "meaning": "cat"}
	]`
	l := readJSON("x", strings.NewReader(in))
	if len(l.Words) != 2 || l.Skipped != 1 {
		t.Fatalf("words=%d skipped=%d problems=%v", len(l.Words), l.Skipped, l.Problems)
	}
	w := l.Words[0]
	if !reflect.DeepEqual(w.Meanings["de"], []string{"Lehrer", "Lehrerin"}) || !reflect.DeepEqual(w.Meanings["en"], []string{"teacher", "master"}) {
		t.Fatalf("meanings = %v", w.Meanings)
	}
	if w.Lesson != "1" || w.JLPT != 5 {
		t.Fatalf("lesson=%q jlpt=%d", w.Lesson, w.JLPT)
	}
	if got := l.Words[1].Meanings[""]; !reflect.DeepEqual(got, []string{"Katze"}) {
		t.Fatalf("string meaning = %v", got)
	}
	bad := readJSON("x", strings.NewReader(`{"kana":`))
	if len(bad.Words) != 0 || len(bad.Problems) != 1 {
		t.Fatalf("broken JSON: %+v", bad)
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("minna.csv", "kana,meaning\nせんせい,Lehrer\nせんせい,Lehrerin\n")
	write("a.csv", "kana\nねこ\n")
	write("a.json", `[{"kana":"いぬ"}]`)
	write("jmdict.tsv", "kana\nとり\n")
	write("notes.txt", "ignored")
	lists, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	byName := map[string]List{}
	for _, l := range lists {
		names = append(names, l.Name)
		byName[l.Name] = l
	}
	want := []string{"a (csv)", "a (json)", "jmdict (tsv)", "minna"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	m := byName["minna"]
	if len(m.Words) != 1 || !reflect.DeepEqual(m.Words[0].Meanings[""], []string{"Lehrer", "Lehrerin"}) {
		t.Fatalf("duplicate readings merged within a list: %+v", m.Words)
	}
	if byName["a (json)"].Words[0].Source != "a (json)" {
		t.Fatal("Source is the list name")
	}
	lists, err = LoadDir(filepath.Join(dir, "missing"))
	if err != nil || len(lists) != 0 {
		t.Fatalf("missing dir: %v, %v", lists, err)
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 -run 'TestReadJSON|TestLoadDir' ./vocab/` — compile error.

- [ ] **Step 3: Implement** `vocab/jsonlist.go`:

```go
package vocab

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"kana/kanacore"
)

type jsonWord struct {
	Kana    string          `json:"kana"`
	Meaning json.RawMessage `json:"meaning"`
	Lesson  json.RawMessage `json:"lesson"`
	JLPT    json.RawMessage `json:"jlpt"`
}

// readJSON reads a word list: an array of objects with kana, meaning
// (string, array of strings, or an object of language → string or array),
// lesson and jlpt (number or string).
func readJSON(name string, r io.Reader) List {
	l := List{Name: name}
	var raw []jsonWord
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		l.problem(fmt.Sprintf("invalid JSON: %v", err))
		return l
	}
	for i, jw := range raw {
		kana := strings.TrimSpace(jw.Kana)
		if _, ok := kanacore.KanaUnits(kana); !ok {
			l.skip(fmt.Sprintf("entry %d: %s is not kana", i+1, kana))
			continue
		}
		w := Word{Kana: kana, Meanings: Meanings{}, Source: name,
			Lesson: scalar(jw.Lesson), JLPT: parseLevel(scalar(jw.JLPT))}
		addJSONMeaning(w.Meanings, "", jw.Meaning)
		l.Words = append(l.Words, w)
	}
	return l
}

// scalar renders a JSON string or number as text; anything else is "".
func scalar(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

func addJSONMeaning(m Meanings, lang string, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		m.Add(lang, SplitMeanings(s)...)
		return
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		for _, item := range list {
			m.Add(lang, SplitMeanings(item)...)
		}
		return
	}
	if lang != "" {
		return // no nested objects
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		for k, v := range obj {
			addJSONMeaning(m, strings.ToLower(k), v)
		}
	}
}
```

`vocab/dir.go`:

```go
package vocab

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadDir reads every *.csv, *.tsv and *.json file in dir as a list. A
// missing directory gives no lists and no error. Lists are sorted by name.
func LoadDir(dir string) ([]List, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	type file struct{ path, base, ext string }
	var files []file
	count := map[string]int{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".csv" && ext != ".tsv" && ext != ".json" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		files = append(files, file{filepath.Join(dir, e.Name()), base, ext})
		count[base]++
	}
	var lists []List
	for _, f := range files {
		name := f.base
		if count[f.base] > 1 || f.base == BundledSource {
			name = f.base + " (" + strings.TrimPrefix(f.ext, ".") + ")"
		}
		fh, err := os.Open(f.path)
		if err != nil {
			l := List{Name: name}
			l.problem(err.Error())
			lists = append(lists, l)
			continue
		}
		var l List
		if f.ext == ".json" {
			l = readJSON(name, fh)
		} else {
			l = readCSV(name, fh)
		}
		fh.Close()
		l.Words = mergeWithin(l.Words)
		lists = append(lists, l)
	}
	sort.Slice(lists, func(i, j int) bool { return lists[i].Name < lists[j].Name })
	return lists, nil
}

// mergeWithin merges words with the same reading, keeping the first word's
// position, lesson and level and merging meanings.
func mergeWithin(words []Word) []Word {
	pos := map[string]int{}
	var out []Word
	for _, w := range words {
		if i, ok := pos[w.Kana]; ok {
			out[i].Meanings.Merge(w.Meanings)
			continue
		}
		pos[w.Kana] = len(out)
		out = append(out, w)
	}
	return out
}
```

Append `words/` to `.gitignore`.

- [ ] **Step 4: Run** `go test -count=1 ./vocab/` — PASS.

- [ ] **Step 5: Commit**

```bash
git add vocab/jsonlist.go vocab/dir.go vocab/dir_test.go .gitignore
git commit -m "feat(vocab): load private word lists from a directory

JSON lists accept a meaning as text, array or per-language object. LoadDir reads every CSV, TSV and JSON file, keeps names unique, reserves jmdict for the bundled list and merges repeated readings within a list."
```

---

### Task 4: Index and playable filter

**Files:** Create `vocab/index.go`, `vocab/index_test.go`

- [ ] **Step 1: Write the failing test** — `vocab/index_test.go`:

```go
package vocab

import (
	"reflect"
	"testing"
)

func word(kana string, level int, source, lesson string, lang, meaning string) Word {
	m := Meanings{}
	m.Add(lang, meaning)
	return Word{Kana: kana, JLPT: level, Source: source, Lesson: lesson, Meanings: m}
}

func testIndex() *Index {
	bundled := []Word{
		word("ねこ", 5, BundledSource, "", "en", "cat"),
		word("はし", 5, BundledSource, "", "en", "bridge"),
		word("きって", 4, BundledSource, "", "en", "stamp"),
		word("かんきょう", 2, BundledSource, "", "en", "environment"),
	}
	lists := []List{
		{Name: "minna", Words: []Word{
			word("はし", 0, "minna", "3", "de", "Essstäbchen"),
			word("せんせい", 0, "minna", "1", "de", "Lehrer"),
			word("かんきょう", 0, "minna", "2", "de", "Umwelt"),
		}},
	}
	return NewIndex(bundled, lists)
}

func kanaOf(es []Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Kana)
	}
	return out
}

func TestNewIndexMerges(t *testing.T) {
	ix := testIndex()
	e, ok := ix.Lookup("はし")
	if !ok || !e.Bundled || e.JLPT != 5 {
		t.Fatalf("はし = %+v", e)
	}
	if !reflect.DeepEqual(e.Meanings["en"], []string{"bridge"}) || !reflect.DeepEqual(e.Meanings["de"], []string{"Essstäbchen"}) {
		t.Fatalf("meanings merged across sources: %v", e.Meanings)
	}
	if !reflect.DeepEqual(e.Lists["minna"], []string{"3"}) {
		t.Fatalf("lists = %v", e.Lists)
	}
	if !reflect.DeepEqual(e.Units, []string{"は", "し"}) {
		t.Fatalf("units = %v", e.Units)
	}
	s, _ := ix.Lookup("せんせい")
	if s.Bundled || s.JLPT != 0 {
		t.Fatalf("private-only entry = %+v", s)
	}
	if len(ix.Entries()) != 5 {
		t.Fatalf("entries = %v", kanaOf(ix.Entries()))
	}
}

func TestPlayable(t *testing.T) {
	ix := testIndex()
	all := func(string) bool { return true }
	cases := []struct {
		name   string
		filter Filter
		known  func(string) bool
		want   []string
	}{
		{"mixed N5", Filter{MaxLevel: 5}, all, []string{"かんきょう", "せんせい", "ねこ", "はし"}},
		{"mixed N4", Filter{MaxLevel: 4}, all, []string{"かんきょう", "きって", "せんせい", "ねこ", "はし"}},
		{"jmdict N5", Filter{MaxLevel: 5, List: BundledSource}, all, []string{"ねこ", "はし"}},
		{"jmdict N2", Filter{MaxLevel: 2, List: BundledSource}, all, []string{"かんきょう", "きって", "ねこ", "はし"}},
		{"minna", Filter{MaxLevel: 5, List: "minna"}, all, []string{"かんきょう", "せんせい", "はし"}},
		{"minna lesson 1", Filter{MaxLevel: 5, List: "minna", Lesson: "1"}, all, []string{"せんせい"}},
		{"unknown list", Filter{MaxLevel: 5, List: "nope"}, all, nil},
		{"known units", Filter{MaxLevel: 5}, func(u string) bool { return u == "ね" || u == "こ" }, []string{"ねこ"}},
	}
	for _, c := range cases {
		got := kanaOf(Playable(ix.Entries(), c.known, c.filter))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run** `go test -count=1 -run 'TestNewIndex|TestPlayable' ./vocab/` — compile error.

- [ ] **Step 3: Implement** `vocab/index.go`:

```go
package vocab

import (
	"sort"

	"kana/kanacore"
)

// Entry is one reading with everything known about it across sources.
type Entry struct {
	Kana     string
	Units    []string // kanacore.KanaUnits, computed once
	Meanings Meanings
	JLPT     int                 // easiest bundled level, 0 if not bundled
	Bundled  bool                // the bundled list contains it
	Lists    map[string][]string // private list name → lessons
}

// Index holds one entry per reading.
type Index struct {
	entries []Entry
	byKana  map[string]int
}

// NewIndex merges bundled words and private lists into one entry per
// reading. Words whose reading cannot be split into units are dropped.
func NewIndex(bundled []Word, lists []List) *Index {
	ix := &Index{byKana: map[string]int{}}
	add := func(w Word, list string) {
		i, ok := ix.byKana[w.Kana]
		if !ok {
			units, valid := kanacore.KanaUnits(w.Kana)
			if !valid {
				return
			}
			ix.byKana[w.Kana] = len(ix.entries)
			ix.entries = append(ix.entries, Entry{Kana: w.Kana, Units: units, Meanings: Meanings{}, Lists: map[string][]string{}})
			i = len(ix.entries) - 1
		}
		e := &ix.entries[i]
		e.Meanings.Merge(w.Meanings)
		if list == "" {
			if !e.Bundled || w.JLPT > e.JLPT {
				e.JLPT = w.JLPT
			}
			e.Bundled = true
			return
		}
		lessons := e.Lists[list]
		if w.Lesson != "" && !contains(lessons, w.Lesson) {
			lessons = append(lessons, w.Lesson)
		}
		e.Lists[list] = lessons
	}
	for _, w := range bundled {
		add(w, "")
	}
	for _, l := range lists {
		for _, w := range l.Words {
			add(w, l.Name)
		}
	}
	sort.Slice(ix.entries, func(a, b int) bool { return ix.entries[a].Kana < ix.entries[b].Kana })
	for i, e := range ix.entries {
		ix.byKana[e.Kana] = i
	}
	return ix
}

// Entries returns all entries sorted by reading.
func (ix *Index) Entries() []Entry { return ix.entries }

// Lookup returns the entry for a reading.
func (ix *Index) Lookup(kana string) (Entry, bool) {
	i, ok := ix.byKana[kana]
	if !ok {
		return Entry{}, false
	}
	return ix.entries[i], true
}

// Filter selects the source of playable entries.
type Filter struct {
	MaxLevel int    // bundled entries need JLPT >= MaxLevel (5 = N5 only)
	List     string // "" = bundled up to MaxLevel plus every private list;
	// BundledSource = bundled only, up to MaxLevel; else only this private list
	Lesson string // with a private List: only this lesson ("" = all)
}

// Playable returns the entries whose units all satisfy known and which pass
// the filter, in index order.
func Playable(entries []Entry, known func(unit string) bool, f Filter) []Entry {
	var out []Entry
	for _, e := range entries {
		if !f.passes(e) || !allKnown(e.Units, known) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (f Filter) passes(e Entry) bool {
	bundledOK := e.Bundled && e.JLPT >= f.MaxLevel
	switch f.List {
	case "":
		return bundledOK || len(e.Lists) > 0
	case BundledSource:
		return bundledOK
	default:
		lessons, ok := e.Lists[f.List]
		return ok && (f.Lesson == "" || contains(lessons, f.Lesson))
	}
}

func allKnown(units []string, known func(string) bool) bool {
	for _, u := range units {
		if !known(u) {
			return false
		}
	}
	return true
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
```

Note: the `Lists` value for a list is created even when the word has no lesson, so `len(e.Lists) > 0` marks "comes from a private list".

- [ ] **Step 4: Run** `go test -count=1 ./vocab/` — PASS.

- [ ] **Step 5: Commit**

```bash
git add vocab/index.go vocab/index_test.go
git commit -m "feat(vocab): merge sources into one index and filter playable words

Each reading becomes one entry with merged meanings, its easiest bundled level, the private lists and lessons it appears in, and its kana units. Playable filters by known units and by source, level and lesson."
```

---

### Task 5: `cmd/vocabgen`

**Files:** Create `cmd/vocabgen/main.go`, `jmdict.go`, `jlpt.go`, `build.go`, `vocabgen_test.go`, `testdata/JMdict_sample.xml`, `testdata/jlpt/n4.csv`, `testdata/jlpt/n5.csv`

The output format (shared with Task 6's `Bundled`) is a gzip of a JSON array of
`{"k": kana, "j": level, "m": {"en": ["…"], "de": ["…"]}}`, sorted by `k`, gzip header without name and with zero ModTime so the file is reproducible.

- [ ] **Step 1: Fixtures**

`cmd/vocabgen/testdata/JMdict_sample.xml` (structure of the real file: DTD with entities, `<ent_seq>`, `<r_ele><reb>`, `<sense>` with `<gloss>`; English glosses have no `xml:lang`, other languages have `xml:lang="ger"` etc. and sit in their own `<sense>`):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE JMdict [
<!ENTITY n "noun (common) (futsuumeishi)">
]>
<JMdict>
<entry>
<ent_seq>1</ent_seq>
<k_ele><keb>先生</keb></k_ele>
<r_ele><reb>せんせい</reb></r_ele>
<sense><pos>&n;</pos><gloss>teacher</gloss><gloss>instructor</gloss><gloss>master</gloss><gloss>doctor</gloss></sense>
<sense><gloss>second sense</gloss></sense>
<sense><gloss xml:lang="ger">Lehrer</gloss><gloss xml:lang="ger">Meister</gloss></sense>
<sense><gloss xml:lang="fre">professeur</gloss></sense>
</entry>
<entry>
<ent_seq>2</ent_seq>
<r_ele><reb>はし</reb></r_ele>
<sense><gloss>bridge</gloss></sense>
<sense><gloss xml:lang="ger">Brücke</gloss></sense>
</entry>
<entry>
<ent_seq>3</ent_seq>
<r_ele><reb>はし</reb></r_ele>
<sense><gloss>chopsticks</gloss></sense>
</entry>
<entry>
<ent_seq>4</ent_seq>
<r_ele><reb>アイデア</reb></r_ele>
<r_ele><reb>アイディア</reb></r_ele>
<sense><gloss>idea</gloss></sense>
</entry>
<entry>
<ent_seq>5</ent_seq>
<r_ele><reb>あっ</reb></r_ele>
<sense><gloss>ah</gloss></sense>
</entry>
<entry>
<ent_seq>99</ent_seq>
<r_ele><reb>いらない</reb></r_ele>
<sense><gloss>not in any list</gloss></sense>
</entry>
</JMdict>
```

`cmd/vocabgen/testdata/jlpt/n5.csv`:

```csv
jmdict_seq,kana,kanji,waller_definition
1,せんせい,先生,teacher
2,はし,橋,bridge
5,あっ,,ah
7,ねこ,猫,cat
```

`cmd/vocabgen/testdata/jlpt/n4.csv`:

```csv
jmdict_seq,kana,kanji,waller_definition
3,はし,箸,chopsticks
4,アイデア,,idea
4,アイディア,,idea
1,せんせい,先生,teacher
```

(seq 7 is missing from the JMdict sample: its English comes from `waller_definition`.)

- [ ] **Step 2: Write the failing test** — `cmd/vocabgen/vocabgen_test.go`:

```go
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type outWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	M map[string][]string `json:"m"`
}

func generate(t *testing.T) ([]outWord, Stats, []byte) {
	t.Helper()
	var buf bytes.Buffer
	stats, err := run("testdata/JMdict_sample.xml", "testdata/jlpt", &buf)
	if err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), buf.Bytes()...)
	zr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	var words []outWord
	if err := json.NewDecoder(zr).Decode(&words); err != nil {
		t.Fatal(err)
	}
	return words, stats, raw
}

func TestGenerate(t *testing.T) {
	words, stats, _ := generate(t)
	byK := map[string]outWord{}
	var order []string
	for _, w := range words {
		byK[w.K] = w
		order = append(order, w.K)
	}
	if want := []string{"アイディア", "アイデア", "せんせい", "ねこ", "はし"}; !reflect.DeepEqual(order, sortedCopy(want)) {
		t.Fatalf("words = %v", order)
	}
	s := byK["せんせい"]
	if s.J != 5 {
		t.Fatalf("easiest level wins: %d", s.J)
	}
	if !reflect.DeepEqual(s.M["en"], []string{"teacher", "instructor", "master"}) {
		t.Fatalf("en = %v (first sense, at most three)", s.M["en"])
	}
	if !reflect.DeepEqual(s.M["de"], []string{"Lehrer", "Meister"}) || !reflect.DeepEqual(s.M["fr"], []string{"professeur"}) {
		t.Fatalf("other languages = %v", s.M)
	}
	h := byK["はし"]
	if h.J != 5 || !reflect.DeepEqual(h.M["en"], []string{"bridge", "chopsticks"}) || !reflect.DeepEqual(h.M["de"], []string{"Brücke"}) {
		t.Fatalf("homophones merged: %+v", h)
	}
	if byK["アイディア"].J != 4 || !reflect.DeepEqual(byK["アイディア"].M["en"], []string{"idea"}) {
		t.Fatalf("second reading of a seq: %+v", byK["アイディア"])
	}
	if !reflect.DeepEqual(byK["ねこ"].M["en"], []string{"cat"}) {
		t.Fatalf("waller fallback: %+v", byK["ねこ"])
	}
	if stats.Dropped != 1 || stats.MissingSeq != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestGenerateIsReproducible(t *testing.T) {
	_, _, a := generate(t)
	_, _, b := generate(t)
	if !bytes.Equal(a, b) {
		t.Fatal("output differs between runs")
	}
}

func TestGenerateMissingInput(t *testing.T) {
	var buf bytes.Buffer
	if _, err := run("testdata/nope.xml", "testdata/jlpt", &buf); err == nil {
		t.Fatal("missing JMdict must fail")
	}
	if _, err := run("testdata/JMdict_sample.xml", os.TempDir()+"/no-such-jlpt", &buf); err == nil {
		t.Fatal("missing JLPT dir must fail")
	}
}
```

Add a small helper in the test file:

```go
func sortedCopy(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}
```

(import `sort`).

- [ ] **Step 3: Run** `go test -count=1 ./cmd/vocabgen/` — compile error.

- [ ] **Step 4: Implement**

`cmd/vocabgen/jlpt.go`:

```go
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// jlptRow is one row of the JLPT lists (yomitan-jlpt-vocab original_data).
type jlptRow struct {
	Seq    string
	Kana   string
	Level  int
	Waller string
}

// readJLPT reads every n1.csv … n5.csv in dir, easiest level (n5) first and
// each file in its own order, so the easiest level's meanings come first
// when words are merged. It fails if dir has no such file.
func readJLPT(dir string) ([]jlptRow, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(dir, "n[1-5].csv"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%s: no n1.csv … n5.csv", dir)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths))) // n5 … n1
	var rows []jlptRow
	for _, path := range paths {
		level := int(filepath.Base(path)[1] - '0')
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		recs, err := csv.NewReader(f).ReadAll()
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		for i, r := range recs {
			if i == 0 || len(r) < 4 {
				continue // header or short row
			}
			rows = append(rows, jlptRow{Seq: strings.TrimSpace(r[0]), Kana: strings.TrimSpace(r[1]), Level: level, Waller: r[3]})
		}
	}
	return rows, nil
}
```

`cmd/vocabgen/jmdict.go`:

```go
package main

import (
	"compress/gzip"
	"encoding/xml"
	"io"
	"os"
	"strings"
)

// langCodes maps JMdict's ISO 639-2 codes to ISO 639-1.
var langCodes = map[string]string{
	"eng": "en", "ger": "de", "fre": "fr", "rus": "ru", "spa": "es",
	"hun": "hu", "dut": "nl", "swe": "sv", "slv": "sl",
}

const maxGlossesPerLang = 3

type xmlGloss struct {
	Lang string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
	Text string `xml:",chardata"`
}

type xmlEntry struct {
	Seq    string `xml:"ent_seq"`
	Senses []struct {
		Glosses []xmlGloss `xml:"gloss"`
	} `xml:"sense"`
}

// readJMdict streams the JMdict XML (plain or .gz) and returns, for every
// entry whose sequence number is wanted, the glosses per language: those of
// the first sense that has glosses in that language, at most three.
func readJMdict(path string, wanted map[string]bool) (map[string]map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		r = zr
	}
	dec := xml.NewDecoder(r)
	dec.Strict = false // JMdict uses DTD entities (&n; …); their text is not needed
	out := map[string]map[string][]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "entry" {
			continue
		}
		var e xmlEntry
		if err := dec.DecodeElement(&e, &se); err != nil {
			return nil, err
		}
		if !wanted[e.Seq] {
			continue
		}
		glosses := map[string][]string{}
		for _, sense := range e.Senses {
			// A language is taken only from the first sense that has
			// glosses in it.
			fresh := map[string]bool{}
			for _, g := range sense.Glosses {
				lang := g.Lang
				if lang == "" {
					lang = "eng"
				}
				code, ok := langCodes[lang]
				text := strings.TrimSpace(g.Text)
				if !ok || text == "" {
					continue
				}
				if _, taken := glosses[code]; taken && !fresh[code] {
					continue
				}
				fresh[code] = true
				if len(glosses[code]) < maxGlossesPerLang {
					glosses[code] = append(glosses[code], text)
				}
			}
		}
		out[e.Seq] = glosses
	}
}
```

Check during implementation whether `encoding/xml` in non-strict mode handles the real file's `<!DOCTYPE … [ … ]>` and `&n;` entities: if unknown entities cause errors, set `dec.Entity` to a map built from the DTD (parse `<!ENTITY name "text">` from the `Directive` token) — the fixture includes one entity so the test covers it.

`cmd/vocabgen/build.go`:

```go
package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"kana/kanacore"
)

const maxGlossesMerged = 5

// Stats reports what the generator kept and dropped.
type Stats struct {
	Rows       int // JLPT rows read
	Words      int // words written
	Dropped    int // readings KanaUnits rejects
	MissingSeq int // JLPT sequence numbers not found in JMdict
}

type outWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	M map[string][]string `json:"m"`
}

// run builds the bundled word list from a JMdict file and a JLPT directory
// and writes it gzip-compressed to w.
func run(jmdictPath, jlptDir string, w io.Writer) (Stats, error) {
	var st Stats
	rows, err := readJLPT(jlptDir)
	if err != nil {
		return st, err
	}
	st.Rows = len(rows)
	wanted := map[string]bool{}
	for _, r := range rows {
		wanted[r.Seq] = true
	}
	glosses, err := readJMdict(jmdictPath, wanted)
	if err != nil {
		return st, err
	}
	missing := map[string]bool{}
	byKana := map[string]*outWord{}
	dropped := map[string]bool{}
	for _, r := range rows {
		if _, ok := kanacore.KanaUnits(r.Kana); !ok {
			dropped[r.Kana] = true
			continue
		}
		g, ok := glosses[r.Seq]
		if !ok {
			missing[r.Seq] = true
			g = map[string][]string{"en": wallerGlosses(r.Waller)}
		}
		ow, ok := byKana[r.Kana]
		if !ok {
			ow = &outWord{K: r.Kana, J: r.Level, M: map[string][]string{}}
			byKana[r.Kana] = ow
		}
		if r.Level > ow.J {
			ow.J = r.Level
		}
		for lang, items := range g {
			for _, it := range items {
				if len(ow.M[lang]) < maxGlossesMerged && !has(ow.M[lang], it) {
					ow.M[lang] = append(ow.M[lang], it)
				}
			}
		}
	}
	st.Dropped = len(dropped)
	st.MissingSeq = len(missing)
	out := make([]outWord, 0, len(byKana))
	for _, ow := range byKana {
		out = append(out, *ow)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].K < out[j].K })
	st.Words = len(out)
	zw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return st, err
	}
	// Zero header (no name, zero ModTime) keeps the output reproducible.
	if err := json.NewEncoder(zw).Encode(out); err != nil {
		return st, err
	}
	if err := zw.Close(); err != nil {
		return st, err
	}
	return st, nil
}

// wallerGlosses splits Waller's comma-separated definition.
func wallerGlosses(def string) []string {
	var out []string
	for _, p := range strings.Split(def, ",") {
		if p = strings.TrimSpace(p); p != "" && len(out) < 3 {
			out = append(out, p)
		}
	}
	return out
}

func has(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func (s Stats) String() string {
	return fmt.Sprintf("%d JLPT rows, %d words written, %d readings dropped (not typeable), %d sequence numbers missing in JMdict", s.Rows, s.Words, s.Dropped, s.MissingSeq)
}
```

Merge order matters for reproducibility and for "bridge" before "chopsticks": rows are processed in file order n1…n5 — so `はし` from n4 (chopsticks) would come before n5 (bridge). The test expects `["bridge", "chopsticks"]`. `readJLPT` returns rows from n5 down to n1, so the easiest level's meanings come first. Note `outWord` is declared in both the test and `build.go`; rename the test's type to `testWord`.

`json.Encoder` sorts map keys, so `m` is deterministic.

`cmd/vocabgen/main.go`:

```go
// Command vocabgen builds vocab/data/words.json.gz from JMdict and the JLPT
// lists. Run by hand; inputs are downloaded by the maintainer.
//
//	go run ./cmd/vocabgen -jmdict JMdict.gz -jlpt yomitan-jlpt-vocab/original_data -out vocab/data/words.json.gz
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	jmdict := flag.String("jmdict", "", "path to JMdict or JMdict.gz (full multilingual)")
	jlpt := flag.String("jlpt", "", "directory with n1.csv … n5.csv (yomitan-jlpt-vocab original_data)")
	out := flag.String("out", "vocab/data/words.json.gz", "output file")
	flag.Parse()
	if *jmdict == "" || *jlpt == "" {
		flag.Usage()
		os.Exit(2)
	}
	tmp := *out + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	st, err := run(*jmdict, *jlpt, f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Rename(tmp, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(st)
}
```

- [ ] **Step 5: Run** `go test -count=1 ./cmd/vocabgen/` — PASS; `go vet ./...` clean.

- [ ] **Step 6: Commit**

```bash
git add cmd/vocabgen
git commit -m "feat(vocabgen): generate the bundled word list from JMdict and JLPT lists

Joins the JLPT lists with JMdict by sequence number, keeps up to three glosses per language from the first sense that has them, merges homophones at their easiest level and writes a reproducible gzip JSON file."
```

---

### Task 6: Bundled data, attribution and docs

**Files:** Create `vocab/data/words.json.gz`, `vocab/data/ATTRIBUTION.md`, `vocab/bundled.go`, `vocab/bundled_test.go`; modify `README.md`, `PLAN.md`

- [ ] **Step 1: Generate the data** (network needed):

```bash
d=$(mktemp -d)
curl -fsSL -o "$d/JMdict.gz" http://ftp.edrdg.org/pub/Nihongo/JMdict.gz
git clone -q --depth 1 https://github.com/stephenmk/yomitan-jlpt-vocab "$d/jlpt"
git -C "$d/jlpt" rev-parse HEAD   # record in ATTRIBUTION.md
mkdir -p vocab/data
go run ./cmd/vocabgen -jmdict "$d/JMdict.gz" -jlpt "$d/jlpt/original_data" -out vocab/data/words.json.gz
ls -l vocab/data/words.json.gz
```

Expected: about 8,000 words written, a small number dropped (e.g. あっ), few or no missing sequence numbers. Report the printed stats and the file size. If the file exceeds ~5 MB, stop and report (the spec assumes a few MB). Record the JMdict file date (its `Last-Modified` or the date in the XML header comment) for `ATTRIBUTION.md`.

- [ ] **Step 2: Write the failing test** — `vocab/bundled_test.go`:

```go
package vocab

import (
	"testing"

	"kana/kanacore"
)

func TestBundled(t *testing.T) {
	words, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	if len(words) < 5000 {
		t.Fatalf("bundled words = %d, want > 5000", len(words))
	}
	levels := map[int]int{}
	for _, w := range words {
		if _, ok := kanacore.KanaUnits(w.Kana); !ok {
			t.Fatalf("%s is not typeable", w.Kana)
		}
		if w.Source != BundledSource || w.JLPT < 1 || w.JLPT > 5 {
			t.Fatalf("bad word %+v", w)
		}
		levels[w.JLPT]++
	}
	for l := 1; l <= 5; l++ {
		if levels[l] == 0 {
			t.Fatalf("no words for N%d", l)
		}
	}
	ix := NewIndex(words, nil)
	s, ok := ix.Lookup("せんせい")
	if !ok || s.JLPT != 5 || !contains(s.Meanings["en"], "teacher") {
		t.Fatalf("せんせい = %+v", s)
	}
	if len(s.Meanings["de"]) == 0 {
		t.Fatalf("せんせい has no German meaning: %v", s.Meanings)
	}
}
```

- [ ] **Step 3: Run** `go test -count=1 -run TestBundled ./vocab/` — compile error.

- [ ] **Step 4: Implement** `vocab/bundled.go`:

```go
package vocab

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/words.json.gz
var bundledData []byte

type bundledWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	M map[string][]string `json:"m"`
}

// Bundled returns the word list shipped with the app (JMdict meanings, JLPT
// levels from Jonathan Waller's lists; see data/ATTRIBUTION.md).
func Bundled() ([]Word, error) {
	zr, err := gzip.NewReader(bytes.NewReader(bundledData))
	if err != nil {
		return nil, fmt.Errorf("bundled words: %w", err)
	}
	defer zr.Close()
	var raw []bundledWord
	if err := json.NewDecoder(zr).Decode(&raw); err != nil {
		return nil, fmt.Errorf("bundled words: %w", err)
	}
	words := make([]Word, len(raw))
	for i, r := range raw {
		words[i] = Word{Kana: r.K, JLPT: r.J, Meanings: Meanings(r.M), Source: BundledSource}
	}
	return words, nil
}
```

- [ ] **Step 5: `vocab/data/ATTRIBUTION.md`** — state:
  - `words.json.gz` is generated by `cmd/vocabgen`; licensed **CC BY-SA 4.0** as an adaptation; the Kana code remains MIT.
  - JMdict: © Electronic Dictionary Research and Development Group, used under CC BY-SA 4.0, https://www.edrdg.org/edrdg/licence.html, file date of the version used.
  - JLPT levels: Jonathan Waller, JLPT resources, http://www.tanos.co.uk/jlpt/, CC BY. These lists are unofficial; no official JLPT vocabulary lists have been published since 2010.
  - JMdict sequence mapping: stephenmk/yomitan-jlpt-vocab, https://github.com/stephenmk/yomitan-jlpt-vocab, CC BY-SA 4.0, commit used.
  - How to regenerate (the Step 1 commands).

- [ ] **Step 6: Docs**
  - `README.md`: a "Word data" section (where the bundled words come from, the three attributions in short, the CC BY-SA licence of the data file, link to `vocab/data/ATTRIBUTION.md`); a "Private word lists" note (location `~/.local/share/kana/words/`, CSV/TSV/JSON columns, `;` for several meanings, quoting in semicolon files, never committed; used from stage 3 on). Add `vocab/` and `cmd/vocabgen/` to the architecture list.
  - `PLAN.md`: tick "Bundled word list derived from JMdict …" and "Import private word lists …" with a note that the game uses them from stage 3.

- [ ] **Step 7: Run** `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...` — all clean/PASS.

- [ ] **Step 8: Commit**

```bash
git add vocab/data vocab/bundled.go vocab/bundled_test.go README.md PLAN.md
git commit -m "feat(vocab): bundle the JLPT word list

Embeds the generated list of JLPT words with JMdict meanings in nine languages, with attribution for JMdict, Jonathan Waller's levels and the sequence mapping, and documents word data and private lists in the README."
```
