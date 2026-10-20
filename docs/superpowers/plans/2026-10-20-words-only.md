# Words-only Mode, Lesson Filter, Meaning Language and Focus Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Learners practise only words (words-only mode), only the lessons they choose (lesson filter), with meanings in a language they choose, and can focus on what they keep getting wrong (weak items).

**Architecture:** `vocab.Source` carries the chosen lessons; `Has`, `NextPackage` and the package lesson follow them, and the index reports a list's lessons and the meaning languages. `store` keeps three new settings. `GameState` gets `wordsOnly`, `focusWeak`, `lessonFilters` and `meaningSetting`; a per-mode unit rule (`unitFits`) and a practice filter (`practised`) decide which words fall and which packages come; the kana path checks `wordsOnly` and rests; a focus pick (`pickFocus`) sits in front of `memory.Pick` for kana and words. The status line and the settings (Words and Game tabs) show and set it all.

**Tech Stack:** Go 1.25, Fyne v2.7.3 (`widget.RadioGroup`, `widget.Select`, `widget.Check`, `container.NewVScroll`). No new dependencies.

Spec: `docs/superpowers/specs/2026-10-20-words-only-design.md`. Terms: `GLOSSARY.md` (words-only mode, lesson filter, weak item, learning level, learned, slipped, waiting word).

**Conventions:** comments like the surrounding code (short doc comment on every function, "Must be called under lock." where it applies; prose uses the glossary terms — "slipped", not "flagged", although the field is `store.Progress.Flagged`). Every `GameState` field is read and written under `gs.mu`; exported methods lock, `…Locked`/"Must be called under lock" helpers do not. UI code runs on the Fyne thread (Fyne callbacks or `fyne.Do`). Tests: `go test ./... -timeout 300s`; the `fyne` tests touch fields directly without the lock, as the existing tests do. Implementers leave changes **uncommitted**; the controller reviews and commits.

---

## Decisions made in the plan

The spec leaves these open or calls them indicative; check them before Task 1.

1. **Lessons in `vocab.Source`:** `Lessons []string` — the chosen lessons, `""` standing for "No lesson" (words without a lesson); empty or nil = all lessons; ignored for JLPT. A slice makes `Source` non-comparable: checked, nothing compares a `vocab.Source` with `==` (only field reads and `%+v`). `wordSettings` **is** compared with `==` (`fyne/settings_test.go:227`); Task 6 adds a slice to it and switches that assertion to `reflect.DeepEqual`. `WordsStatus` is compared with `==` (`fyne/wordgame_test.go:947`); its new fields are bools, so it stays comparable.
2. **Storage keys:** `words_only` and `focus_weak` (`"1"`/`"0"`, unset = off), `lesson_filter:<list name>` (JSON array; absent = all lessons; saving an empty filter deletes the key). `meaning_lang` keeps its meaning: `""` = System.
3. **Stale lessons:** the stored filter is used as is. A lesson that no longer exists matches nothing, so it is ignored next to others; a filter whose lessons are all gone matches no word, which the words line reports as "no words in the chosen lessons" (both modes; in words-only mode kana fall). The settings show only existing lessons, and saving writes what the checks show.
4. **Checks to filter:** all lessons checked, or none checked, saves "all lessons" (the key is removed). "All lessons" checks/unchecks every lesson, like the row groups' "all" (shared helper `linkAllCheck`).
5. **What falls, per mode:** words-only — introduced words of the source (JLPT: bundled with JLPT ≥ level; a list: its words) passing the lesson filter, whose scripts are active. Mixed without a lesson filter — today's rule (every introduced word, every source, all kana learned). Mixed **with** a lesson filter — limited to the source and filter like words-only, still with the learned-kana rule. ("All lessons is today's behaviour.")
6. **Packages in words-only mode:** the spec's "`known` accepts every unit" is implemented as "every unit whose script is active" (`unitFits`), so a katakana word is not introduced while katakana is off — otherwise it would be introduced and could not fall.
7. **Waiting for kana** is counted only in mixed mode, among the words the practice filter lets fall; always 0 in words-only mode (a word held back by an inactive script is not counted).
8. **Package lesson under a filter:** a list package holds the earliest *chosen* lesson that has a word to introduce; a word in several lessons counts for its first chosen lesson (`Index.LessonIn`). The package title and the "already known" replacement use the same lesson. Within the lesson the list's order is kept.
9. **Kana fallback:** in words-only mode a spawn tries a word (`pickWord(true)`) and spawns a kana when none is playable — no extra "package could come" check at spawn time; a due package is offered at the usual re-check points and pauses the game.
10. **Pending kana intros in words-only mode** are invisible to `dialogPending`, `PendingDialogs` and `Reset` (`introPending`); switching back to mixed mode pauses the game so `showPendingDialogs` shows them.
11. **Words line texts** (words-only, problems only): "Words only: could not load progress", "Words only: words unavailable: <err>", "Words only: add a list to <dir> or choose JLPT" / "… add a word list or choose JLPT", "Words only: no words in the chosen lessons" / "Words only: no words in <source>", "Words only: no word fits the active scripts", and, with words falling but new ones held back by a script, "Words only: new words need a script that is off". Mixed mode gains "Words: no words in the chosen lessons".
12. **Meaning languages** are listed by language code (de, en, es, fr, hu, nl, ru, sl, sv — the spec's example order), each by its own name from a built-in table (an unknown code shows as itself), "System" first. The spec text says "sorted by display name", but its example is in code order and plain string sorting would put Русский last; the controller may flip this in `meaningLangOptions`. A stored language the data no longer has stays as an option.
13. **Focus pick:** focus off or no weak candidate → no `rnd` draw at all (the usual pick is untouched, seeded tests stay stable). Otherwise one draw against `focusShare = 0.5`, then a weighted draw. Slipped = `Flagged` at any level (weight 2); learning = level 1–4 and not slipped (weight 1).
14. **Focus setting** is applied with a separate `SetFocusWeak` (like `SetKanaFont`); `applySettings`' signature stays, so no fixture changes.
15. **Settings dialog:** only the selected list's lesson filter is saved; edits to other lists' checks are kept while the dialog is open (switching back and forth) but dropped on Save. The lesson box is hidden for JLPT and for a list without lessons; check labels are the lesson names, `""` is "No lesson".
16. **Filter change** counts like a source change: it clears the newest package (saved), invalidates the playable words and drops word tiles that may no longer fall.

---

## File structure

- Modify `vocab/package.go`: `Source.Lessons`, `Has`, `NextPackageWhere` (earliest chosen lesson), `LessonIn`, `firstLesson` taking a `Source`.
- Modify `vocab/index.go`: `Index.Lessons`, `Index.Languages`.
- Modify `vocab/package_test.go`, `vocab/index_test.go`.
- Modify `store/store.go`, `store/store_test.go`: `WordsOnly`, `FocusWeak`, `LessonFilter` (+ save), `flag`/`saveFlag`.
- Modify `fyne/game.go`: fields `wordsOnly`, `focusWeak`, `lessonFilters`, `meaningSetting`; loading; `spawnKana`; resting kana path (`checkAutoProgression`, `maybeOfferKatakana`, `introPending`, `dialogPending`, `Reset`, `PendingDialogs`); `SetFocusWeak`.
- Modify `fyne/wordgame.go`: `unitActive`, `unitFits`, `unitsFit`, `practised`, `playableWords`, `pickWord`, `source`, `introducedCounts`, `sourceSize`, `wordsStatus`, `loadLists` (lesson filters).
- Modify `fyne/packages.go`: `wordSettings`, `newestPackageDone`, `packageDue`, `ReplacementWord`, `samePackage`, `applyWordSettings`, `setLessonFilter`.
- Create `fyne/focus.go`: `focusShare`, `weakWeight`, `pickFocus`, `pickKana`, `pickReading`.
- Modify `fyne/words.go`: `WordsStatus` fields, `formatWordsLine`, `wordsOnlyReason`, `languageNames`, `languageName`, `meaningLangOptions`.
- Modify `fyne/packageintro.go`: `packageTitle` uses `LessonIn`.
- Modify `fyne/settings.go`: settings state, Words tab (tiles radio, lesson box, meaning select), Game tab (focus check), `linkAllCheck`, `words()`, `apply`.
- Modify `fyne/importlist.go`: refresh the lessons after an import.
- Create tests `fyne/wordsonly_test.go`, `fyne/focus_test.go`, `fyne/lessons_test.go`, `fyne/meaning_test.go`; modify `fyne/words_test.go`, `fyne/settings_test.go`.
- Modify `README.md`.

---

### Task 1: `vocab` — lessons in the source, package lesson, lessons and languages of the index

**Files:** Modify `vocab/package.go`, `vocab/index.go`, `vocab/package_test.go`, `vocab/index_test.go`.

- [ ] **Step 1: Failing tests** — append to `vocab/package_test.go`:

```go
func TestSourceHasChosenLessons(t *testing.T) {
	ix := NewIndex(nil, []List{lessonList(
		[2]string{"き", "1"}, [2]string{"ねこ", "2"}, [2]string{"あめ", ""}, [2]string{"き", "3"},
	)})
	cases := []struct {
		lessons []string
		kana    string
		want    bool
	}{
		{nil, "き", true},
		{nil, "あめ", true},
		{[]string{"3"}, "き", true}, // き is in lessons 1 and 3
		{[]string{"2"}, "き", false},
		{[]string{"2"}, "ねこ", true},
		{[]string{"2"}, "あめ", false},
		{[]string{""}, "あめ", true}, // "No lesson"
		{[]string{""}, "ねこ", false},
		{[]string{"9"}, "ねこ", false}, // a lesson that is gone matches nothing
	}
	for _, c := range cases {
		e, _ := ix.Lookup(c.kana)
		if got := (Source{List: "minna", Lessons: c.lessons}).Has(e); got != c.want {
			t.Errorf("lessons %q: Has(%s) = %v, want %v", c.lessons, c.kana, got, c.want)
		}
	}
	jlpt := testIndex()
	e, _ := jlpt.Lookup("ねこ")
	if !(Source{Level: 5, Lessons: []string{"2"}}).Has(e) {
		t.Fatal("JLPT ignores lessons")
	}
}

func TestNextPackageWithinChosenLessons(t *testing.T) {
	ix := NewIndex(nil, []List{lessonList(
		[2]string{"き", "1"}, [2]string{"ねこ", "2"}, [2]string{"いぬ", "1"},
		[2]string{"みず", "3"}, [2]string{"いぬ", "3"}, [2]string{"あめ", ""},
	)})
	pkg := func(lessons ...string) []string {
		return kanaOf(NextPackage(ix, Source{List: "minna", Lessons: lessons}, nil, 5, knowAll))
	}
	if got := pkg("3"); !reflect.DeepEqual(got, []string{"いぬ", "みず"}) {
		t.Fatalf("lesson 3 = %v, want [いぬ みず]: いぬ is in lessons 1 and 3", got)
	}
	for _, lessons := range [][]string{{"2", "3"}, {"3", "2"}} {
		if got := pkg(lessons...); !reflect.DeepEqual(got, []string{"ねこ"}) {
			t.Fatalf("lessons %v = %v, want [ねこ]: the earliest chosen lesson, and いぬ counts for lesson 3 here", lessons, got)
		}
	}
	if got := pkg(""); !reflect.DeepEqual(got, []string{"あめ"}) {
		t.Fatalf("No lesson = %v, want [あめ]", got)
	}
	if got := pkg("9"); got != nil {
		t.Fatalf("a gone lesson = %v, want none", got)
	}
	got := allPackages(ix, Source{List: "minna", Lessons: []string{"1", "3"}}, 10, knowAll)
	if !reflect.DeepEqual(got, []string{"き", "いぬ", "みず"}) {
		t.Fatalf("lessons 1 and 3 = %v, want [き いぬ] then [みず]", got)
	}
	e, _ := ix.Lookup("いぬ")
	if l := ix.LessonIn(e, Source{List: "minna", Lessons: []string{"3"}}); l != "3" {
		t.Fatalf("LessonIn(いぬ, lesson 3) = %q", l)
	}
	if l := ix.LessonIn(e, Source{List: "minna"}); l != "1" {
		t.Fatalf("LessonIn(いぬ, all) = %q, want its first lesson 1", l)
	}
}
```

Append to `vocab/index_test.go`:

```go
func TestIndexLessons(t *testing.T) {
	ix := NewIndex(nil, []List{
		lessonList([2]string{"いぬ", "10"}, [2]string{"ねこ", "2"}, [2]string{"あめ", ""}, [2]string{"き", "1"}),
		{Name: "named", Words: []Word{
			word("いぬ", 0, "named", "Tiere", "de", "Hund"),
			word("みず", 0, "named", "Wetter", "de", "Wasser"),
		}},
	})
	if lessons, none := ix.Lessons("minna"); !reflect.DeepEqual(lessons, []string{"1", "2", "10"}) || !none {
		t.Fatalf("minna: %v, no lesson %v; want [1 2 10] in number order and words without a lesson", lessons, none)
	}
	if lessons, none := ix.Lessons("named"); !reflect.DeepEqual(lessons, []string{"Tiere", "Wetter"}) || none {
		t.Fatalf("named: %v, no lesson %v", lessons, none)
	}
	if lessons, none := ix.Lessons("gone"); lessons != nil || none {
		t.Fatalf("unknown list: %v, %v", lessons, none)
	}
}

func TestIndexLanguages(t *testing.T) {
	if got := testIndex().Languages(); !reflect.DeepEqual(got, []string{"de", "en"}) {
		t.Fatalf("Languages = %v, want [de en]", got)
	}
	ix := NewIndex([]Word{word("ねこ", 5, BundledSource, "", "", "cat")}, nil)
	if got := ix.Languages(); len(got) != 0 {
		t.Fatalf("meanings without a language count as none, got %v", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./vocab/ -timeout 300s` → build failure (`Lessons` field, `LessonIn`, `Index.Lessons`, `Index.Languages`).

- [ ] **Step 3: Implement** in `vocab/package.go`. Replace `Source` and `Has`:

```go
// Source names where new words come from.
type Source struct {
	List    string   // private list name; "" = JLPT (the bundled list)
	Level   int      // JLPT: lowest level number allowed (5 = N5 only); ignored for lists
	Lessons []string // list: the chosen lessons, "" for words without a lesson; empty = all lessons. Ignored for JLPT
}

// Has reports whether e belongs to the source: for JLPT a bundled entry
// with JLPT >= Level, for a list an entry of that list in one of the chosen
// lessons (or without a lesson, if "" is chosen).
func (s Source) Has(e Entry) bool {
	if s.List == "" {
		return e.Bundled && e.JLPT >= s.Level
	}
	lessons, ok := e.Lists[s.List]
	if !ok {
		return false
	}
	if len(s.Lessons) == 0 {
		return true
	}
	if len(lessons) == 0 {
		return contains(s.Lessons, "")
	}
	for _, l := range lessons {
		if contains(s.Lessons, l) {
			return true
		}
	}
	return false
}
```

Update the `NextPackage` doc's last paragraph sentence "A list package holds words of a single lesson." to "A list package holds words of a single lesson: the earliest chosen lesson (src.Lessons, all when empty) with a word to introduce; an entry in several lessons counts for its first chosen one." Replace `NextPackageWhere`:

```go
// NextPackageWhere is NextPackage limited to the entries keep accepts (nil
// accepts all), e.g. to refill a package within its lesson or levels.
func NextPackageWhere(ix *Index, src Source, introduced map[string]bool, n int, known func(string) bool, keep func(Entry) bool) []Entry {
	if ix == nil || n <= 0 {
		return nil
	}
	eligible := func(e Entry) bool {
		return src.Has(e) && !introduced[e.Kana] && allKnown(e.Units, known) && (keep == nil || keep(e))
	}
	order := ix.order(src.List)
	var out []Entry
	if src.List == "" {
		for _, i := range order {
			if e := ix.entries[i]; eligible(e) {
				out = append(out, e)
				if len(out) == n {
					break
				}
			}
		}
		return out
	}
	// A list package holds words of one lesson only: the earliest (chosen)
	// lesson with an eligible word, so earlier lessons' unreadable words do
	// not hold it back. Words without a lesson come last.
	ranks := ix.lessonRanks(src.List)
	var lesson string
	var rank int
	found, hasLesson := false, false
	for _, i := range order {
		e := ix.entries[i]
		if !eligible(e) {
			continue
		}
		l, r, ok := firstLesson(e, src, ranks)
		if !found || (ok && (!hasLesson || r < rank)) {
			lesson, rank, hasLesson, found = l, r, ok, true
		}
	}
	if !found {
		return nil
	}
	for _, i := range order {
		e := ix.entries[i]
		if l, _, _ := firstLesson(e, src, ranks); l == lesson && eligible(e) {
			out = append(out, e)
			if len(out) == n {
				break
			}
		}
	}
	return out
}
```

Replace `Lesson` and `firstLesson`, add `LessonIn`:

```go
// Lesson returns e's first lesson in the list's lesson order, "" if it has
// none in that list.
func (ix *Index) Lesson(e Entry, list string) string {
	return ix.LessonIn(e, Source{List: list})
}

// LessonIn returns e's first lesson in the lesson order of src's list,
// among src.Lessons when set; "" if it has none there or src is JLPT.
func (ix *Index) LessonIn(e Entry, src Source) string {
	if src.List == "" {
		return ""
	}
	lesson, _, _ := firstLesson(e, src, ix.lessonRanks(src.List))
	return lesson
}

// firstLesson returns e's lesson in src's list that comes first by ranks,
// among src.Lessons when set, and its rank; ok is false when e has none
// there.
func firstLesson(e Entry, src Source, ranks map[string]int) (lesson string, rank int, ok bool) {
	for _, l := range e.Lists[src.List] {
		if len(src.Lessons) > 0 && !contains(src.Lessons, l) {
			continue
		}
		if r, known := ranks[l]; known && (!ok || r < rank) {
			lesson, rank, ok = l, r, true
		}
	}
	return lesson, rank, ok
}
```

In `order`, the call becomes `_, r, ok := firstLesson(e, Source{List: list}, ranks)`.

In `vocab/index.go` add (imports gain nothing new: `maps`, `slices`, `sort` are there):

```go
// Lessons returns a list's lessons in learning order (by number if every
// lesson is an integer, else by first appearance) and whether the list has
// words without a lesson. Nil and false for an unknown list.
func (ix *Index) Lessons(list string) (lessons []string, noLesson bool) {
	ranks := ix.lessonRanks(list)
	lessons = slices.Clone(ix.lessons[list])
	sort.SliceStable(lessons, func(a, b int) bool { return ranks[lessons[a]] < ranks[lessons[b]] })
	for _, i := range ix.order(list) {
		if len(ix.entries[i].Lists[list]) == 0 {
			noLesson = true
			break
		}
	}
	return lessons, noLesson
}

// Languages returns the meaning languages of the index's entries, sorted by
// code. Meanings without a language do not count.
func (ix *Index) Languages() []string {
	seen := map[string]bool{}
	for _, e := range ix.entries {
		for lang, items := range e.Meanings {
			if lang != "" && len(items) > 0 {
				seen[lang] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(seen))
}
```

`slices.Clone(nil)` is nil, so an unknown list gives `nil, false`.

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS (the existing package tests, `TestNextPackageListOrder` and `TestNextPackageSkipsAheadPastUnreadableWords` in particular, must stay green: without a filter the earliest-lesson rule picks the same lesson as before).

---

### Task 2: `store` — words-only, focus and lesson filter settings

**Files:** Modify `store/store.go` (setting keys near line 20, accessors after `SaveNewestPackage`), `store/store_test.go`.

- [ ] **Step 1: Failing test** (append to `store/store_test.go`):

```go
func TestWordsOnlyFocusAndLessonFilter(t *testing.T) {
	st := openTemp(t)
	if on, err := st.WordsOnly(); err != nil || on {
		t.Fatalf("unset WordsOnly = %v, %v; want false", on, err)
	}
	if on, err := st.FocusWeak(); err != nil || on {
		t.Fatalf("unset FocusWeak = %v, %v; want false", on, err)
	}
	if l, err := st.LessonFilter("minna"); err != nil || l != nil {
		t.Fatalf("unset LessonFilter = %v, %v; want nil (all lessons)", l, err)
	}
	if err := st.SaveWordsOnly(true); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFocusWeak(true); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveLessonFilter("minna", []string{"2", ""}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveLessonFilter("other", []string{"1"}); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.WordsOnly(); !on {
		t.Fatal("WordsOnly not saved")
	}
	if on, _ := st.FocusWeak(); !on {
		t.Fatal("FocusWeak not saved")
	}
	if l, _ := st.LessonFilter("minna"); !reflect.DeepEqual(l, []string{"2", ""}) {
		t.Fatalf("minna = %v, want [2 \"\"]", l)
	}
	if err := st.SaveLessonFilter("minna", nil); err != nil {
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("minna"); l != nil {
		t.Fatalf("cleared filter = %v, want nil", l)
	}
	if l, _ := st.LessonFilter("other"); !reflect.DeepEqual(l, []string{"1"}) {
		t.Fatalf("other = %v: filters are per list", l)
	}
	if err := st.SaveWordsOnly(false); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.WordsOnly(); on {
		t.Fatal("WordsOnly off not saved")
	}
}
```

- [ ] **Step 2: Run** `go test ./store/ -timeout 300s` → build failure.

- [ ] **Step 3: Implement.** Keys:

```go
	wordsOnlyKey       = "words_only"
	focusWeakKey       = "focus_weak"
	lessonFilterPrefix = "lesson_filter:" // followed by a word list's name
```

Accessors (after `SaveNewestPackage`):

```go
// WordsOnly reports whether only words fall (words-only mode); false when
// unset.
func (s *Store) WordsOnly() (bool, error) { return s.flag(wordsOnlyKey) }

// SaveWordsOnly persists words-only mode.
func (s *Store) SaveWordsOnly(on bool) error { return s.saveFlag(wordsOnlyKey, on) }

// FocusWeak reports whether practice focuses on weak items; false when
// unset.
func (s *Store) FocusWeak() (bool, error) { return s.flag(focusWeakKey) }

// SaveFocusWeak persists the focus on weak items.
func (s *Store) SaveFocusWeak(on bool) error { return s.saveFlag(focusWeakKey, on) }

// LessonFilter returns a word list's lesson filter: the chosen lessons, ""
// for words without a lesson. Nil when unset: all lessons.
func (s *Store) LessonFilter(list string) ([]string, error) {
	value, err := s.getSetting(lessonFilterPrefix + list)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var lessons []string
	if err := json.Unmarshal([]byte(value), &lessons); err != nil {
		return nil, fmt.Errorf("store: decode lesson filter: %w", err)
	}
	if len(lessons) == 0 {
		return nil, nil
	}
	return lessons, nil
}

// SaveLessonFilter persists a word list's lesson filter as a JSON array;
// nil or empty removes it (all lessons).
func (s *Store) SaveLessonFilter(list string, lessons []string) error {
	if len(lessons) == 0 {
		return s.deleteSetting(lessonFilterPrefix + list)
	}
	payload, err := json.Marshal(lessons)
	if err != nil {
		return fmt.Errorf("store: encode lesson filter: %w", err)
	}
	return s.setSetting(lessonFilterPrefix+list, string(payload))
}

// flag reads a setting stored as "1" or "0"; unset is false.
func (s *Store) flag(key string) (bool, error) {
	value, err := s.getSetting(key)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == "1", nil
}

// saveFlag stores a setting as "1" or "0".
func (s *Store) saveFlag(key string, on bool) error {
	if on {
		return s.setSetting(key, "1")
	}
	return s.setSetting(key, "0")
}
```

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS.

---

### Task 3: Words-only mode — what falls, what comes, what spawns

**Files:** Modify `fyne/game.go` (fields, `NewGameState`, `spawnKana`), `fyne/wordgame.go`, `fyne/packages.go`; create `fyne/wordsonly_test.go`.

- [ ] **Step 1: Failing tests** (`fyne/wordsonly_test.go`):

```go
package main

import (
	"math/rand"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/vocab"
)

func TestWordsOnlySpawnsOnlyWords(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.rnd = rand.New(rand.NewSource(1)).Float64
	withWords(gs, bw("あい", 5, "love"), bw("かお", 5, "face")) // no kana learned
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "かお"}) {
		t.Fatalf("playable = %v, want both words although no kana is learned", got)
	}
	if n := countWordSpawns(gs, 200); n != 200 {
		t.Fatalf("%d/200 word spawns, want every spawn a word", n)
	}
}

func TestWordsOnlyKatakanaWordNeedsKatakana(t *testing.T) {
	gs := newTestState() // hiragana only
	gs.wordsOnly = true
	withWords(gs, bw("アイ", 5, "eye"), bw("あい", 5, "love"))
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable = %v, want only あい while katakana is off", got)
	}
	gs.activeScripts[kanacore.ScriptKatakana] = true
	gs.invalidatePlayable()
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "アイ"}) {
		t.Fatalf("playable with katakana = %v", got)
	}
}

func TestWordsOnlyOnlyFromTheSource(t *testing.T) {
	gs := newTestState()
	gs.wordsOnly = true
	withWords(gs, bw("あい", 5, "love"), bw("あお", 4, "blue")) // the source is JLPT N5
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable = %v, want only the N5 word", got)
	}
}

func TestWordsOnlyPackagesIgnoreLearnedKana(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.packageSize = 10
	gs.words = vocab.NewIndex(append(vowelWords(3), bw("かお", 5, "face"), bw("アイ", 5, "eye")), nil)
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 {
		t.Fatalf("mixed mode without learned kana: pending %v", gs.pendingPackage)
	}
	gs.wordsOnly = true
	gs.maybeOfferPackage()
	got := strings.Join(gs.pendingPackage, " ")
	if len(gs.pendingPackage) != 4 || !strings.Contains(got, "かお") || strings.Contains(got, "アイ") {
		t.Fatalf("words-only package = %v, want the 3 vowel words and かお, not アイ (katakana off)", gs.pendingPackage)
	}
}

func TestWordsOnlyFallsBackToKana(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	withWords(gs, bw("あい", 5, "love"))
	gs.wordSource = listSourcePrefix + "gone"
	gs.invalidatePlayable()
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].word != nil {
		t.Fatal("no word can fall: a kana must fall")
	}
}

func TestWordsOnlyCountsNoWaitingWords(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	if _, _, waiting := gs.introducedCounts(); waiting != 1 {
		t.Fatalf("mixed: waiting = %d, want 1", waiting)
	}
	gs.wordsOnly = true
	gs.invalidatePlayable()
	if _, _, waiting := gs.introducedCounts(); waiting != 0 {
		t.Fatalf("words-only: waiting = %d, want 0", waiting)
	}
	if s := gs.wordsSummary(); strings.Contains(s, "waiting") {
		t.Fatalf("summary %q must not mention waiting words", s)
	}
}

func TestSwitchingWordsOnlyRestoresMixedRules(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	withWords(gs, bw("あい", 5, "love"))
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, wordsOnly: true})
	if !gs.wordsOnly || !equalIDs(playableKana(gs), []string{"あい"}) {
		t.Fatalf("words-only %v, playable %v", gs.wordsOnly, playableKana(gs))
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 0)}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5})
	if gs.wordsOnly || len(playableKana(gs)) != 0 || len(gs.tiles) != 0 {
		t.Fatalf("back to mixed: words-only %v, playable %v, tiles %d; あい waits for its kana again", gs.wordsOnly, playableKana(gs), len(gs.tiles))
	}
}

func TestWordsOnlyIsStored(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.applyWordSettings(wordSettings{source: gs.wordSource, level: 5, packageSize: 5, wordsOnly: true})
	if on, _ := st.WordsOnly(); !on {
		t.Fatal("words-only not saved")
	}
	if !NewGameState(st).wordsOnly {
		t.Fatal("words-only not loaded")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -timeout 300s` → build failure (`wordsOnly`).

- [ ] **Step 3: Implement.**

`fyne/game.go`, in the "Word packages" field block after `wordSource`:

```go
	wordsOnly            bool            // words-only mode: only words fall; the kana learning path rests
```

In `NewGameState`'s `st != nil` block, next to the other word settings (before `loadWords`, so `offerPackageAtStart` sees the mode):

```go
		if on, err := st.WordsOnly(); err == nil {
			gs.wordsOnly = on
		}
```

`spawnKana`: doc and the pick line:

```go
// spawnKana adds one falling tile. In mixed mode it is a word with
// probability wordShare when words are playable, otherwise a kana; with no
// kana available it spawns a word if one is playable, else nothing. In
// words-only mode every spawn is a word while one is playable, and a kana
// only when none is.
```

```go
	chars := gs.availableCharacters()
	if entry, ok := gs.pickWord(gs.wordsOnly || len(chars) == 0); ok {
```

`fyne/wordgame.go`: replace `unitKnown`/`unitsKnown` with:

```go
// unitActive reports whether a word unit's script is active: a kana's own
// script; っ and ッ need their script, ー either script. Must be called
// under lock.
func (gs *GameState) unitActive(unit string) bool {
	switch unit {
	case "っ":
		return gs.activeScripts[kanacore.ScriptHiragana]
	case "ッ":
		return gs.activeScripts[kanacore.ScriptKatakana]
	case "ー":
		return gs.activeScripts[kanacore.ScriptHiragana] || gs.activeScripts[kanacore.ScriptKatakana]
	}
	rowID, ok := kanacore.CharToRow[unit]
	return ok && gs.activeScripts[kanacore.ScriptOf(rowID)]
}

// unitKnown reports whether a word unit is known in mixed mode: its script
// is active and, for a kana, the kana is learned. Must be called under lock.
func (gs *GameState) unitKnown(unit string) bool {
	switch unit {
	case "っ", "ッ", "ー":
		return gs.unitActive(unit)
	}
	return gs.unitActive(unit) && learned(gs.kanaLevel(unit))
}

// unitsKnown reports whether every unit of a word is known. Must be called
// under lock.
func (gs *GameState) unitsKnown(units []string) bool {
	for _, u := range units {
		if !gs.unitKnown(u) {
			return false
		}
	}
	return true
}

// unitFits reports whether a word unit lets its word fall and be
// introduced in the current mode: known in mixed mode, of an active script
// in words-only mode. Must be called under lock.
func (gs *GameState) unitFits(unit string) bool {
	if gs.wordsOnly {
		return gs.unitActive(unit)
	}
	return gs.unitKnown(unit)
}

// unitsFit reports whether every unit of a word fits (see unitFits). Must
// be called under lock.
func (gs *GameState) unitsFit(units []string) bool {
	for _, u := range units {
		if !gs.unitFits(u) {
			return false
		}
	}
	return true
}

// practised reports whether an introduced word belongs to what is
// practised: in words-only mode the source with its lesson filter; in mixed
// mode every introduced word, unless a lesson filter is set, which limits
// it to the source as well. found is whether the source exists. Must be
// called under lock.
func (gs *GameState) practised(e vocab.Entry, src vocab.Source, found bool) bool {
	if !gs.wordsOnly && len(src.Lessons) == 0 {
		return true
	}
	return found && src.Has(e)
}
```

`playableWords` (doc and loop):

```go
// playableWords returns the entries that may fall: introduced words that
// are practised (see practised) and whose units fit the mode (see
// unitFits), sorted by reading. None while the introduced words could not
// be loaded. The result is cached until invalidatePlayable; do not modify
// it. Must be called under lock.
func (gs *GameState) playableWords() []vocab.Entry {
	if gs.words == nil || gs.introducedLoadFailed {
		return nil
	}
	if !gs.playableOK {
		src, found := gs.source()
		var out []vocab.Entry
		for kana := range gs.introduced {
			if e, ok := gs.words.Lookup(kana); ok && gs.practised(e, src, found) && gs.unitsFit(e.Units) {
				out = append(out, e)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Kana < out[j].Kana })
		gs.playable = out
		gs.playableOK = true
	}
	return gs.playable
}
```

`invalidatePlayable` doc: add "the mode, the source or its lesson filter" to the list of triggers.

`introducedCounts`:

```go
// introducedCounts counts the introduced readings in the word index, those
// of them that are learned and, in mixed mode, the practised ones that wait
// for a kana to be learned before they can fall; words-only mode does not
// wait for kana. Must be called under lock.
func (gs *GameState) introducedCounts() (introduced, learnedN, waiting int) {
	src, found := gs.source()
	for kana := range gs.introduced {
		e, ok := gs.words.Lookup(kana)
		if !ok {
			continue
		}
		introduced++
		if gs.wordLearned(kana) {
			learnedN++
		}
		if !gs.wordsOnly && gs.practised(e, src, found) && !gs.unitsKnown(e.Units) {
			waiting++
		}
	}
	return introduced, learnedN, waiting
}
```

`wordsStatus`: `vocab.NextPackage(gs.words, src, gs.introduced, 1, gs.unitFits)`.

`fyne/packages.go`:

- `wordSettings` gains `wordsOnly bool // only words fall`.
- `newestPackageDone`: `if !ok || !gs.unitsFit(e.Units) {`; doc "readings that can fall now".
- `packageDue`: `vocab.NextPackage(gs.words, src, gs.introduced, gs.packageSize, gs.unitFits)`.
- `ReplacementWord`: `vocab.NextPackageWhere(gs.words, src, skip, 1, gs.unitFits, samePackage(gs.words, src, pending))`.
- `applyWordSettings`, before `if changed {`:

```go
	if ws.wordsOnly != gs.wordsOnly {
		gs.wordsOnly = ws.wordsOnly
		changed = true
		if gs.store != nil {
			_ = gs.store.SaveWordsOnly(ws.wordsOnly)
		}
	}
```

and its doc: "… A new source clears the newest package …; switching words-only mode keeps it."

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS. The existing mixed-mode tests must not change: without a lesson filter `practised` is true and `unitFits` is `unitKnown`.

---

### Task 4: Words-only mode — the kana learning path rests

**Files:** Modify `fyne/game.go` (`checkAutoProgression`, `maybeOfferKatakana`, `dialogPending`, `Reset`, `PendingDialogs`), `fyne/packages.go` (`applyWordSettings`), `fyne/wordsonly_test.go`.

- [ ] **Step 1: Failing tests** (append to `fyne/wordsonly_test.go`):

```go
func TestWordsOnlyRestsTheKanaPath(t *testing.T) {
	gs := newUnlockReadyState()
	gs.wordsOnly = true
	gs.checkAnswer("a")
	if len(gs.pendingIntro) != 0 || gs.paused || gs.selectedRows["k"] {
		t.Fatalf("words-only: intro %v, paused %v, k selected %v; want no unlock", gs.pendingIntro, gs.paused, gs.selectedRows["k"])
	}
	if kinds := drainEvents(gs); len(kinds) != 0 {
		t.Fatalf("events = %v, want none", kinds)
	}
	gs = newOfferReadyState()
	gs.wordsOnly = true
	gs.checkAnswer("a")
	if gs.pendingOffer || gs.katakanaOffered {
		t.Fatal("words-only: no katakana offer")
	}
	gs.wordsOnly = false
	answerOne(gs)
	if !gs.pendingOffer {
		t.Fatal("mixed again: the offer comes as before")
	}
}

func TestWordsOnlyKeepsPendingIntros(t *testing.T) {
	gs := newTestState()
	gs.pendingIntro = []string{"k"}
	gs.paused = true
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, wordsOnly: true})
	gs.Resume()
	if gs.paused || len(gs.PendingDialogs().Intro) != 0 || !equalIDs(gs.pendingIntro, []string{"k"}) {
		t.Fatalf("words-only: paused %v, shown intro %v, kept %v; want running with the intro kept", gs.paused, gs.PendingDialogs().Intro, gs.pendingIntro)
	}
	gs.Reset()
	if gs.paused {
		t.Fatal("Reset in words-only mode must not wait for the kana intro")
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5})
	gs.Resume()
	if !gs.paused || !equalIDs(gs.PendingDialogs().Intro, []string{"k"}) {
		t.Fatalf("back to mixed: paused %v, intro %v; want the intro shown", gs.paused, gs.PendingDialogs().Intro)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'WordsOnly' -timeout 300s` → FAIL.

- [ ] **Step 3: Implement** in `fyne/game.go`:

- `checkAutoProgression`: first line `if !gs.autoProgress || gs.wordsOnly {`; doc adds "Nothing is unlocked in words-only mode, where the kana path rests."
- `maybeOfferKatakana`: `if gs.over || gs.wordsOnly || gs.katakanaOffered || …`; doc adds "or in words-only mode".
- New helper next to `dialogPending`:

```go
// introPending reports whether a kana intro waits to be shown: never in
// words-only mode, where the kana path rests and its intros wait until
// mixed mode is back. Must be called under lock.
func (gs *GameState) introPending() bool {
	return len(gs.pendingIntro) > 0 && !gs.wordsOnly
}
```

- `dialogPending`: `return gs.introPending() || gs.pendingOffer || …`; doc "a kana intro (not in words-only mode)".
- `Reset`: `gs.paused = gs.introPending()`.
- `PendingDialogs`: build `p` without `Intro`, then

```go
	if gs.introPending() {
		p.Intro = append([]string(nil), gs.pendingIntro...)
	}
```

In `fyne/packages.go`, `applyWordSettings`' words-only block gains:

```go
		if gs.introPending() {
			gs.paused = true // the kana intros that waited are shown now
		}
```

(after `gs.wordsOnly = ws.wordsOnly`). `showSettingsDialog`'s `closed` already calls `showPendingDialogs` after `apply`, which shows the intro.

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS.

---

### Task 5: Focus on weak items

**Files:** Create `fyne/focus.go`, `fyne/focus_test.go`; modify `fyne/game.go` (field, loading, `spawnKana`, `SetFocusWeak`), `fyne/wordgame.go` (`pickWord`).

- [ ] **Step 1: Failing tests** (`fyne/focus_test.go`):

```go
package main

import (
	"math/rand"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/store"
)

func TestWeakWeight(t *testing.T) {
	cases := []struct {
		p    store.Progress
		want float64
	}{
		{store.Progress{}, 0},                    // new
		{store.Progress{Level: 1}, 1},            // learning
		{store.Progress{Level: 4}, 1},            // learning
		{store.Progress{Level: learnedLevel}, 0}, // learned
		{store.Progress{Level: 0, Flagged: true}, 2},
		{store.Progress{Level: 3, Flagged: true}, 2},
	}
	for _, c := range cases {
		if got := weakWeight(c.p); got != c.want {
			t.Errorf("weakWeight(%+v) = %v, want %v", c.p, got, c.want)
		}
	}
}

func TestFocusPicksWeakItems(t *testing.T) {
	gs := newTestState()
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(3)).Float64
	gs.kanaProgress["あ"] = store.Progress{Level: 1, Flagged: true}
	gs.kanaProgress["い"] = store.Progress{Level: 3}
	gs.kanaProgress["う"] = store.Progress{Level: learnedLevel}
	cands := []string{"あ", "い", "う", "え", "お"}
	counts := map[string]int{}
	weak := 0
	for i := 0; i < 3000; i++ {
		if c, ok := gs.pickFocus(cands, gs.kanaProgress); ok {
			weak++
			counts[c]++
		}
	}
	if weak < 1350 || weak > 1650 {
		t.Fatalf("%d/3000 focus picks, want about half", weak)
	}
	if r := float64(counts["あ"]) / float64(counts["い"]); r < 1.6 || r > 2.5 {
		t.Fatalf("slipped/learning = %d/%d, want about 2", counts["あ"], counts["い"])
	}
	if counts["う"]+counts["え"]+counts["お"] != 0 {
		t.Fatalf("learned and new items are not weak: %v", counts)
	}
}

func TestFocusDrawsNothingWithoutWeakItemsOrWhenOff(t *testing.T) {
	gs := newTestState()
	draws := 0
	gs.rnd = func() float64 { draws++; return 0 }
	gs.kanaProgress["あ"] = store.Progress{Level: 2, Flagged: true}
	if _, ok := gs.pickFocus([]string{"あ"}, gs.kanaProgress); ok || draws != 0 {
		t.Fatalf("focus off: ok %v, %d draws; want the usual pick untouched", ok, draws)
	}
	gs.focusWeak = true
	if _, ok := gs.pickFocus([]string{"い", "う"}, gs.kanaProgress); ok || draws != 0 {
		t.Fatalf("no weak candidate: ok %v, %d draws", ok, draws)
	}
}

func TestFocusAppliesToSpawnedKanaAndWords(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(5)).Float64
	gs.kanaProgress["え"] = store.Progress{Level: 1, Flagged: true}
	n := 0
	for i := 0; i < 1000; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if gs.tiles[0].kana.Char == "え" {
			n++
		}
	}
	if n < 520 || n > 680 {
		t.Fatalf("slipped え in %d/1000 spawns, want about 600 (half, plus its usual share)", n)
	}

	gs = newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(5)).Float64
	words := vowelWords(10)
	withWords(gs, words...)
	weakWord := words[3].Kana
	gs.wordProgress[weakWord] = store.Progress{Level: 2, Flagged: true}
	n = 0
	for i := 0; i < 1000; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if gs.tiles[0].word.Kana == weakWord {
			n++
		}
	}
	if n < 480 || n > 640 {
		t.Fatalf("slipped word in %d/1000 spawns, want about 550", n)
	}
}

func TestFocusKeepsTheWordShare(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(7)).Float64
	withWords(gs, vowelWords(60)...)
	learnKana(gs, "あ", "い", "う", "え", "お")
	for kana := range gs.introduced {
		gs.wordProgress[kana] = store.Progress{Level: 1, Flagged: true}
	}
	if n := countWordSpawns(gs, 1000); n < 70 || n > 130 {
		t.Fatalf("%d/1000 word spawns with focus on, want about 100 as without", n)
	}
}

func TestFocusWeakIsStored(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.SetFocusWeak(true)
	if on, _ := st.FocusWeak(); !on || !NewGameState(st).focusWeak {
		t.Fatal("focus on weak items not saved or not loaded")
	}
}
```

The bounds are for the given seeds; if a seed lands just outside, widen the bounds a little rather than hunting for a lucky seed — the expected values are in the messages.

- [ ] **Step 2: Run** `go test ./fyne/ -run 'Focus|WeakWeight' -timeout 300s` → build failure.

- [ ] **Step 3: Implement** `fyne/focus.go`:

```go
package main

import (
	"kana/memory"
	"kana/store"
)

// focusShare is the share of picks that, with the focus on weak items, go
// to the weak candidates.
const focusShare = 0.5

// weakWeight is how strongly the focus pick favours an item with progress
// p: 2 if it slipped, 1 if it is learning (level 1–4), 0 if it is not weak
// (new or learned).
func weakWeight(p store.Progress) float64 {
	switch {
	case p.Flagged:
		return 2
	case p.Level >= 1 && !learned(p.Level):
		return 1
	}
	return 0
}

// pickFocus is the focus pick. With the focus on weak items and a weak
// candidate, it draws with gs.rnd whether this pick goes to the weak
// candidates (focusShare) and, if so, picks one weighted by weakWeight. It
// returns false without drawing when the focus is off or no candidate is
// weak, and false after the draw when the usual pick applies. Must be
// called under lock.
func (gs *GameState) pickFocus(candidates []string, progress map[string]store.Progress) (string, bool) {
	if !gs.focusWeak {
		return "", false
	}
	var weak []string
	var weights []float64
	total := 0.0
	for _, c := range candidates {
		if w := weakWeight(progress[c]); w > 0 {
			weak = append(weak, c)
			weights = append(weights, w)
			total += w
		}
	}
	if total == 0 || gs.rnd() >= focusShare {
		return "", false
	}
	x := gs.rnd() * total
	for i, w := range weights {
		if x < w {
			return weak[i], true
		}
		x -= w
	}
	return weak[len(weak)-1], true
}

// pickKana picks the next kana from candidates: the focus pick, else the
// spaced-repetition pick. Must be called under lock.
func (gs *GameState) pickKana(candidates []string) string {
	if c, ok := gs.pickFocus(candidates, gs.kanaProgress); ok {
		return c
	}
	return memory.Pick(candidates, gs.memory, gs.now(), gs.rnd)
}

// pickReading picks the next word reading from candidates, like pickKana
// over the word progress and memory. Must be called under lock.
func (gs *GameState) pickReading(candidates []string) string {
	if c, ok := gs.pickFocus(candidates, gs.wordProgress); ok {
		return c
	}
	return memory.Pick(candidates, gs.wordMemory, gs.now(), gs.rnd)
}
```

`fyne/game.go`:

- field, after `pointsSetting`: `focusWeak     bool      // focus on weak items: half the picks go to them`
- `NewGameState`, `st != nil` block: `if on, err := st.FocusWeak(); err == nil { gs.focusWeak = on }`
- `spawnKana`: `char := gs.pickKana(gs.spawnCandidates(chars))`
- next to `SetKanaFont`:

```go
// SetFocusWeak turns the focus on weak items on or off and saves it.
func (gs *GameState) SetFocusWeak(on bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.focusWeak = on
	if gs.store != nil {
		_ = gs.store.SaveFocusWeak(on)
	}
}
```

`fyne/wordgame.go`, `pickWord`: `kana := gs.pickReading(gs.wordCandidates(playable))`; doc: "… picks one with the focus pick or memory.Pick over the word memory." The word-share draw stays before the focus pick, so the focus does not change how often a word falls. If `memory` becomes unused in `wordgame.go`, drop the import there (it is still used by `recordWordCorrect`).

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS.

---

### Task 6: Lesson filter in the game

**Files:** Modify `fyne/game.go` (field), `fyne/wordgame.go` (`loadLists`, `source`), `fyne/packages.go` (`wordSettings`, `samePackage`, `applyWordSettings`, `setLessonFilter`), `fyne/packageintro.go` (`packageTitle`), `fyne/settings_test.go` (line 227); create `fyne/lessons_test.go`.

- [ ] **Step 1: Failing tests** (`fyne/lessons_test.go`):

```go
package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/vocab"
)

// lw is a word of a private list in a lesson ("" = none), with its reading
// as German meaning.
func lw(list, kana, lesson string) vocab.Word {
	return vocab.Word{Kana: kana, Source: list, Lesson: lesson, Meanings: vocab.Meanings{"de": {kana}}}
}

// withList makes the private list the word source, with all its words
// introduced.
func withList(gs *GameState, list string, words ...vocab.Word) {
	gs.words = vocab.NewIndex(nil, []vocab.List{{Name: list, Words: words}})
	gs.listNames = []string{list}
	gs.wordSource = listSourcePrefix + list
	gs.introduced = map[string]bool{}
	for _, w := range words {
		gs.introduced[w.Kana] = true
	}
	gs.invalidatePlayable()
}

func TestLessonFilterLimitsFallingWords(t *testing.T) {
	for _, wordsOnly := range []bool{false, true} {
		gs := newTestState()
		gs.wordsOnly = wordsOnly
		withList(gs, "minna", lw("minna", "あい", "1"), lw("minna", "あお", "2"), lw("minna", "いえ", ""))
		gs.words = vocab.NewIndex([]vocab.Word{bw("うえ", 5, "above")},
			[]vocab.List{{Name: "minna", Words: []vocab.Word{lw("minna", "あい", "1"), lw("minna", "あお", "2"), lw("minna", "いえ", "")}}})
		gs.introduced["うえ"] = true
		learnKana(gs, "あ", "い", "う", "え", "お")

		all := []string{"あい", "あお", "いえ"}
		if !wordsOnly {
			all = []string{"あい", "あお", "いえ", "うえ"} // mixed, all lessons: every introduced word
		}
		if got := playableKana(gs); !equalIDs(got, all) {
			t.Fatalf("words-only %v, all lessons: %v, want %v", wordsOnly, got, all)
		}
		gs.lessonFilters = map[string][]string{"minna": {"2", ""}}
		gs.invalidatePlayable()
		if got := playableKana(gs); !equalIDs(got, []string{"あお", "いえ"}) {
			t.Fatalf("words-only %v, lesson 2 and no lesson: %v, want [あお いえ]", wordsOnly, got)
		}
	}
}

func TestPackagesComeFromTheChosenLessons(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う", "え", "お")
	withList(gs, "minna", lw("minna", "あい", "1"), lw("minna", "あう", "2"), lw("minna", "あえ", "3"), lw("minna", "あお", "3"))
	gs.introduced = map[string]bool{}
	gs.lessonFilters = map[string][]string{"minna": {"3", "2"}}
	gs.maybeOfferPackage()
	if !equalIDs(gs.pendingPackage, []string{"あう"}) {
		t.Fatalf("first package = %v, want lesson 2: [あう]", gs.pendingPackage)
	}
	if got := gs.PackageIntro().Title; got != "New words · minna · lesson 2" {
		t.Fatalf("title = %q", got)
	}
	gs.FinishPackage(nil)
	setWordLevel(gs, "あう", learnedLevel)
	drainEvents(gs)
	gs.maybeOfferPackage()
	if !equalIDs(gs.pendingPackage, []string{"あえ", "あお"}) {
		t.Fatalf("second package = %v, want lesson 3", gs.pendingPackage)
	}
}

func TestLessonFilterChangeStartsThePackageRhythmAfresh(t *testing.T) {
	gs := newTestState()
	withList(gs, "minna", lw("minna", "あい", "1"), lw("minna", "あお", "2"))
	gs.newestPackage = []string{"あい"}
	gs.applyWordSettings(wordSettings{source: listSourcePrefix + "minna", level: 5, packageSize: 5})
	if !equalIDs(gs.newestPackage, []string{"あい"}) {
		t.Fatal("an unchanged filter keeps the newest package")
	}
	gs.applyWordSettings(wordSettings{source: listSourcePrefix + "minna", level: 5, packageSize: 5, lessons: []string{"2"}})
	if gs.newestPackage != nil || !slices.Equal(gs.lessonFilters["minna"], []string{"2"}) {
		t.Fatalf("newest %v, filter %v", gs.newestPackage, gs.lessonFilters["minna"])
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5})
	if !slices.Equal(gs.lessonFilters["minna"], []string{"2"}) {
		t.Fatal("choosing JLPT must not touch the list's filter")
	}
}

func TestLessonFilterIsStoredPerList(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"minna.csv": "kana;meaning;lesson\nあい;love;1\nあお;blue;2\n",
		"other.csv": "kana;meaning;lesson\nいえ;house;1\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordSource("list:minna")
	gs := NewGameState(st)
	gs.applyWordSettings(wordSettings{source: "list:minna", level: 5, packageSize: 5, lessons: []string{"2"}})
	if l, _ := st.LessonFilter("minna"); !slices.Equal(l, []string{"2"}) {
		t.Fatalf("stored minna = %v", l)
	}
	if l, _ := st.LessonFilter("other"); l != nil {
		t.Fatalf("stored other = %v, want none", l)
	}
	gs = NewGameState(st)
	gs.mu.Lock()
	src, _ := gs.source()
	gs.wordSource = "list:other"
	other, _ := gs.source()
	gs.mu.Unlock()
	if !slices.Equal(src.Lessons, []string{"2"}) || other.Lessons != nil {
		t.Fatalf("loaded: minna %v, other %v", src.Lessons, other.Lessons)
	}
}
```

In `fyne/settings_test.go` (`TestSettingsFormCarriesValues`, line 227) replace the `==` comparison, which no longer compiles once `wordSettings` holds a slice, and add `"reflect"` to the imports:

```go
	if got := f.words(); !reflect.DeepEqual(got, wordSettings{source: jlptSource, level: 4, packageSize: 10}) {
```

- [ ] **Step 2: Run** `go test ./fyne/ -timeout 300s` → build failure (`lessonFilters`, `wordSettings.lessons`).

- [ ] **Step 3: Implement.**

`fyne/game.go`, after `wordSource`:

```go
	lessonFilters        map[string][]string // lesson filter per list name; absent = all lessons
```

`fyne/wordgame.go`, `loadLists`, after `gs.listNames` is filled:

```go
	gs.lessonFilters = map[string][]string{}
	if gs.store != nil {
		for _, name := range gs.listNames {
			if lessons, err := gs.store.LessonFilter(name); err == nil && len(lessons) > 0 {
				gs.lessonFilters[name] = lessons
			}
		}
	}
```

(doc: "… and the stored lesson filters of the lists found"). `source`:

```go
// source returns where new words come from, with the list's lesson filter,
// and whether that source exists: a chosen list that was not found does
// not. Must be called under lock.
func (gs *GameState) source() (vocab.Source, bool) {
	if name, ok := strings.CutPrefix(gs.wordSource, listSourcePrefix); ok {
		if slices.Contains(gs.listNames, name) {
			return vocab.Source{List: name, Lessons: gs.lessonFilters[name]}, true
		}
		return vocab.Source{List: name}, false
	}
	return vocab.Source{Level: gs.wordLevel}, true
}
```

(add `"slices"` to the imports). `sourceLeft` already counts with `src.Has`, so "left in source" now counts within the chosen lessons.

`fyne/packages.go`:

- `wordSettings` gains `lessons []string // the lesson filter of a list source; nil = all lessons`.
- `samePackage` uses the lesson within the filter:

```go
	if src.List != "" {
		lesson := ix.LessonIn(pending[0], src)
		return func(e vocab.Entry) bool { return ix.LessonIn(e, src) == lesson }
	}
```

- `applyWordSettings`, right after the source block (add `"slices"`, `"strings"` to the imports):

```go
	if name, ok := strings.CutPrefix(ws.source, listSourcePrefix); ok && !slices.Equal(ws.lessons, gs.lessonFilters[name]) {
		gs.setLessonFilter(name, ws.lessons)
		gs.newestPackage = nil
		gs.saveNewestPackage()
		changed = true
	}
```

Doc: "A new source or lesson filter clears the newest package, so its first package can come at once; …".

- new helper:

```go
// setLessonFilter sets and saves a list's lesson filter; nil or empty means
// all lessons. Must be called under lock.
func (gs *GameState) setLessonFilter(list string, lessons []string) {
	if gs.lessonFilters == nil {
		gs.lessonFilters = map[string][]string{}
	}
	if len(lessons) == 0 {
		delete(gs.lessonFilters, list)
	} else {
		gs.lessonFilters[list] = slices.Clone(lessons)
	}
	if gs.store != nil {
		_ = gs.store.SaveLessonFilter(list, lessons)
	}
}
```

`fyne/packageintro.go`, `packageTitle`: `if l := ix.LessonIn(e, src); l != "" && …` (the source passed in carries the filter, so a word in several lessons is titled by its chosen lesson).

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS.

---

### Task 7: Meaning language in the game

**Files:** Modify `fyne/game.go` (field, loading), `fyne/packages.go` (`wordSettings`, `applyWordSettings`), `fyne/words.go` (names, options); create `fyne/meaning_test.go`.

- [ ] **Step 1: Failing tests** (`fyne/meaning_test.go`):

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/vocab"
)

func TestMeaningLangSettingDecidesShownAndAcceptedMeaning(t *testing.T) {
	orig := systemLanguage
	systemLanguage = func() string { return "de-DE" }
	t.Cleanup(func() { systemLanguage = orig })
	kasa := vocab.Word{Kana: "かさ", JLPT: 5, Meanings: vocab.Meanings{"de": {"Regenschirm"}, "en": {"umbrella"}, "fr": {"parapluie"}}}
	answer := func(gs *GameState, input string) bool {
		gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
		gs.checkAnswer(input)
		return len(gs.tiles) == 0
	}

	gs := newTestState()
	withWords(gs, kasa)
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, meaningLang: "fr"})
	if gs.meaningLang != "fr" || gs.meaningSetting != "fr" {
		t.Fatalf("meaningLang %q, setting %q", gs.meaningLang, gs.meaningSetting)
	}
	if !answer(gs, "parapluie") || gs.feedback[len(gs.feedback)-1].text.Text != "かさ · kasa — parapluie" {
		t.Fatal("the chosen language is accepted and shown")
	}
	if !answer(gs, "umbrella") {
		t.Fatal("English is always accepted")
	}
	if answer(gs, "Regenschirm") {
		t.Fatal("German is not the meaning language now")
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, meaningLang: ""})
	if gs.meaningLang != "de" || gs.meaningSetting != "" {
		t.Fatalf("System: meaningLang %q, setting %q; want today's rule (the system's de)", gs.meaningLang, gs.meaningSetting)
	}
}

func TestMeaningLangSettingIsStored(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.applyWordSettings(wordSettings{source: gs.wordSource, level: 5, packageSize: 5, meaningLang: "nl"})
	if l, _ := st.MeaningLang(); l != "nl" {
		t.Fatalf("stored %q", l)
	}
	if gs := NewGameState(st); gs.meaningSetting != "nl" || gs.meaningLang != "nl" {
		t.Fatalf("loaded: setting %q, lang %q", gs.meaningSetting, gs.meaningLang)
	}
}

func TestMeaningLangOptions(t *testing.T) {
	labels, values := meaningLangOptions([]string{"de", "en", "fr", "ru", "xx"}, "")
	if !equalIDs(labels, []string{"System", "Deutsch", "English", "Français", "Русский", "xx"}) ||
		!equalIDs(values, []string{"", "de", "en", "fr", "ru", "xx"}) {
		t.Fatalf("labels %v, values %v", labels, values)
	}
	labels, values = meaningLangOptions([]string{"de", "en"}, "it")
	if !equalIDs(labels, []string{"System", "Deutsch", "English", "Italiano"}) || values[3] != "it" {
		t.Fatalf("a stored language the words lack stays: %v %v", labels, values)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run MeaningLang -timeout 300s` → build failure.

- [ ] **Step 3: Implement.**

`fyne/game.go`, after `meaningLang`:

```go
	meaningSetting         string                    // the stored meaning language; "" = System (see resolveMeaningLang)
```

`NewGameState`: after `gs.meaningLang = resolveMeaningLang(storedLang, systemLanguage())` add `gs.meaningSetting = storedLang`.

`fyne/packages.go`: `wordSettings` gains `meaningLang string // "" = System`; in `applyWordSettings`, before `if changed {`:

```go
	if ws.meaningLang != gs.meaningSetting {
		gs.meaningSetting = ws.meaningLang
		gs.meaningLang = resolveMeaningLang(ws.meaningLang, systemLanguage())
		if gs.store != nil {
			_ = gs.store.SaveMeaningLang(ws.meaningLang)
		}
	}
```

The meaning language changes neither the playable words nor the tiles, so it does not set `changed`. Feedback (`tick`, `checkAnswer`), answers (`answers`) and the package intro read `gs.meaningLang`, so the change takes effect at once.

`fyne/words.go`, after `resolveMeaningLang` (add `"slices"` to the imports):

```go
// languageNames are meaning languages by their own names.
var languageNames = map[string]string{
	"cs": "Čeština", "da": "Dansk", "de": "Deutsch", "en": "English", "es": "Español",
	"fi": "Suomi", "fr": "Français", "hu": "Magyar", "it": "Italiano", "ja": "日本語",
	"ko": "한국어", "nl": "Nederlands", "no": "Norsk", "pl": "Polski", "pt": "Português",
	"ru": "Русский", "sl": "Slovenščina", "sv": "Svenska", "tr": "Türkçe", "uk": "Українська",
	"zh": "中文",
}

// languageName names a meaning language in itself; an unknown code is
// shown as is.
func languageName(code string) string {
	if n, ok := languageNames[code]; ok {
		return n
	}
	return code
}

// meaningLangOptions are the "Meanings in" select's labels and values:
// "System" (value "") first, then the languages by code, each by its own
// name. A stored language the words no longer have stays as an option.
func meaningLangOptions(langs []string, stored string) (labels, values []string) {
	all := slices.Clone(langs)
	if stored != "" && !slices.Contains(all, stored) {
		all = append(all, stored)
		slices.Sort(all)
	}
	labels, values = []string{"System"}, []string{""}
	for _, l := range all {
		labels = append(labels, languageName(l))
		values = append(values, l)
	}
	return labels, values
}
```

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS.

---

### Task 8: Status texts

**Files:** Modify `fyne/words.go` (`WordsStatus`, `formatWordsLine`), `fyne/wordgame.go` (`wordsStatus`, `sourceSize`), `fyne/words_test.go`, `fyne/wordsonly_test.go`.

- [ ] **Step 1: Failing tests.** In `fyne/words_test.go`, `TestFormatWordsLine`, add to the table:

```go
		{"words only, counts", WordsStatus{WordsOnly: true, Introduced: 3, Learned: 1, Source: "minna"}, "Words: 3 introduced · 1 learned · minna"},
		{"words only, load failed", WordsStatus{WordsOnly: true, KanaFallback: true, LoadFailed: true}, "Words only: could not load progress"},
		{"words only, bundled error", WordsStatus{WordsOnly: true, KanaFallback: true, Err: "broken data", Level: 5}, "Words only: words unavailable: broken data"},
		{"words only, missing list", WordsStatus{WordsOnly: true, KanaFallback: true, Missing: true, Dir: "~/w", Source: "minna"}, "Words only: add a list to ~/w or choose JLPT"},
		{"words only, missing list, no folder", WordsStatus{WordsOnly: true, KanaFallback: true, Missing: true, Source: "minna"}, "Words only: add a word list or choose JLPT"},
		{"words only, empty lessons", WordsStatus{WordsOnly: true, KanaFallback: true, Empty: true, Filtered: true, Source: "minna"}, "Words only: no words in the chosen lessons"},
		{"words only, empty source", WordsStatus{WordsOnly: true, KanaFallback: true, Empty: true, Source: "minna"}, "Words only: no words in minna"},
		{"words only, scripts", WordsStatus{WordsOnly: true, KanaFallback: true, Introduced: 2, Source: "minna"}, "Words only: no word fits the active scripts"},
		{"words only, new words need a script", WordsStatus{WordsOnly: true, Next: wordsNeedKana, Introduced: 3, Source: "minna"}, "Words only: new words need a script that is off"},
		{"mixed, empty lessons", WordsStatus{Empty: true, Filtered: true, Next: wordsFinished, Source: "minna"}, "Words: no words in the chosen lessons"},
```

In `fyne/wordsonly_test.go`:

```go
func TestWordsOnlyStatusSaysWhyKanaFall(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	withList(gs, "minna", lw("minna", "あい", "1"))
	gs.lessonFilters = map[string][]string{"minna": {"7"}}
	gs.invalidatePlayable()
	line := func() string { return formatWordsLine(gs.wordsStatus()) }
	if got := line(); got != "Words only: no words in the chosen lessons" {
		t.Fatalf("gone lesson: %q", got)
	}
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].word != nil {
		t.Fatal("no word can fall: a kana must fall")
	}
	gs.lessonFilters = nil
	gs.invalidatePlayable()
	if got := line(); got != "Words: minna finished" {
		t.Fatalf("all lessons: %q, want the usual line", got)
	}
	withList(gs, "minna", lw("minna", "アイ", "1"))
	if got := line(); got != "Words only: no word fits the active scripts" {
		t.Fatalf("katakana word, katakana off: %q", got)
	}
	gs.wordSource = listSourcePrefix + "gone"
	gs.wordsDir = "/data/kana/words"
	gs.invalidatePlayable()
	if got := line(); got != "Words only: add a list to /data/kana/words or choose JLPT" {
		t.Fatalf("missing list: %q", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'FormatWordsLine|WordsOnlyStatus' -timeout 300s` → build failure.

- [ ] **Step 3: Implement.** `WordsStatus` gains:

```go
	WordsOnly    bool // words-only mode is on
	KanaFallback bool // words-only mode, but no word can fall and no package can come: kana fall
	Filtered     bool // a lesson filter limits the source
	Empty        bool // the source with its lesson filter has no word at all
```

`formatWordsLine`:

```go
// formatWordsLine is the status panel's words line: counts, or a hint when
// nothing can be introduced. In words-only mode a problem starts with
// "Words only:".
func formatWordsLine(ws WordsStatus) string {
	var s string
	switch {
	case ws.KanaFallback:
		s = "Words only: " + wordsOnlyReason(ws)
	case ws.LoadFailed:
		s = "Words unavailable: could not load progress"
	case ws.Err != "":
		s = "Words unavailable: " + ws.Err
	case ws.Missing && ws.Dir != "":
		s = "Words: add a list to " + ws.Dir + " or choose JLPT"
	case ws.Missing:
		s = "Words: add a word list or choose JLPT"
	case ws.Empty && ws.Filtered:
		s = "Words: no words in the chosen lessons"
	case ws.Next == wordsNeedKana && ws.WordsOnly:
		s = "Words only: new words need a script that is off"
	case ws.Next == wordsNeedKana:
		s = "Words: learn more kana for new words"
	// … the remaining cases as before …
	}
	return truncateRunes(s, maxWordsLineRunes)
}

// wordsOnlyReason says why no word can fall in words-only mode.
func wordsOnlyReason(ws WordsStatus) string {
	switch {
	case ws.LoadFailed:
		return "could not load progress"
	case ws.Err != "":
		return "words unavailable: " + ws.Err
	case ws.Missing && ws.Dir != "":
		return "add a list to " + ws.Dir + " or choose JLPT"
	case ws.Missing:
		return "add a word list or choose JLPT"
	case ws.Empty && ws.Filtered:
		return "no words in the chosen lessons"
	case ws.Empty:
		return "no words in " + ws.Source
	}
	return "no word fits the active scripts"
}
```

`fyne/wordgame.go`:

```go
// sourceSize counts the source's entries, introduced or not. Must be called
// under lock.
func (gs *GameState) sourceSize(src vocab.Source) int {
	if gs.words == nil {
		return 0
	}
	n := 0
	for _, e := range gs.words.Entries() {
		if src.Has(e) {
			n++
		}
	}
	return n
}

// wordsStatus summarises the words for the status panel. Must be called
// under lock.
func (gs *GameState) wordsStatus() WordsStatus {
	src, found := gs.source()
	ws := WordsStatus{LoadFailed: gs.introducedLoadFailed, Source: sourceLabel(src), Missing: !found,
		WordsOnly: gs.wordsOnly, Filtered: len(src.Lessons) > 0}
	if src.List == "" {
		ws.Err, ws.Level = gs.wordsErr, src.Level
	}
	if gs.wordsDir != "" {
		home, _ := os.UserHomeDir()
		ws.Dir = displayDir(gs.wordsDir, home)
	}
	if gs.words == nil {
		ws.KanaFallback = gs.wordsOnly
		return ws
	}
	ws.Introduced, ws.Learned, ws.Waiting = gs.introducedCounts()
	canCome := false
	if found && !ws.LoadFailed {
		ws.Empty = gs.sourceSize(src) == 0
		canCome = len(vocab.NextPackage(gs.words, src, gs.introduced, 1, gs.unitFits)) > 0
		if !canCome {
			ws.Next = wordsFinished
			if gs.sourceLeft(src) > 0 {
				ws.Next = wordsNeedKana
			}
		}
	}
	ws.KanaFallback = gs.wordsOnly && len(gs.playableWords()) == 0 && !canCome
	return ws
}
```

`TestSnapshotWordsStatus` compares the whole struct; its game is mixed mode without a filter and its source has words, so the new fields stay false.

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS.

---

### Task 9: Settings — tiles, lessons, meaning language, focus

**Files:** Modify `fyne/settings.go`, `fyne/importlist.go` (line 145), `fyne/settings_test.go`.

- [ ] **Step 1: Failing tests** (append to `fyne/settings_test.go`):

```go
// lessonsGame is settingsGame with lessons in its list, a second list and
// every new setting away from its default.
func lessonsGame() *GameState {
	gs := settingsGame()
	gs.words = vocab.NewIndex(
		[]vocab.Word{bw("あい", 5, "love")},
		[]vocab.List{
			{Name: "minna-no-nihongo", Words: []vocab.Word{
				lw("minna-no-nihongo", "あい", "2"), lw("minna-no-nihongo", "うえ", "1"), lw("minna-no-nihongo", "いえ", ""),
			}},
			{Name: "other", Words: []vocab.Word{lw("other", "あお", "A")}},
		},
	)
	gs.listNames = []string{"minna-no-nihongo", "other"}
	gs.lessonFilters = map[string][]string{"minna-no-nihongo": {"2"}}
	gs.wordsOnly = true
	gs.focusWeak = true
	gs.meaningSetting = "de"
	return gs
}

func checkedLessons(f *settingsForm) []string {
	var out []string
	for _, l := range f.lessonOrder {
		if f.lessonChecks[l].Checked {
			out = append(out, l)
		}
	}
	return out
}

func TestSettingsFormCarriesTheNewSettings(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(lessonsGame().settingsState(), nil)
	if f.tiles.Selected != "Words only" || !f.focusCheck.Checked {
		t.Fatalf("tiles %q, focus %v", f.tiles.Selected, f.focusCheck.Checked)
	}
	if !equalIDs(f.meaningSelect.Options, []string{"System", "Deutsch", "English"}) || f.meaningSelect.Selected != "Deutsch" {
		t.Fatalf("meaning options %v, selected %q", f.meaningSelect.Options, f.meaningSelect.Selected)
	}
	if !equalIDs(f.lessonOrder, []string{"1", "2", ""}) || !equalIDs(checkedLessons(f), []string{"2"}) || f.allLessons.Checked {
		t.Fatalf("lessons %v, checked %v, all %v", f.lessonOrder, checkedLessons(f), f.allLessons.Checked)
	}
	if f.lessonChecks[""].Text != "No lesson" || !f.lessonScroll.Visible() {
		t.Fatalf("label %q, box visible %v", f.lessonChecks[""].Text, f.lessonScroll.Visible())
	}
	f.allLessons.SetChecked(true)
	if f.words().lessons != nil {
		t.Fatalf("all lessons checked: %v, want nil", f.words().lessons)
	}
	f.lessonChecks["1"].SetChecked(false)
	if got := f.words().lessons; !equalIDs(got, []string{"2", ""}) {
		t.Fatalf("lessons = %v", got)
	}

	f.sourceSelect.SetSelected("other")
	if !equalIDs(f.lessonOrder, []string{"A"}) || !equalIDs(checkedLessons(f), []string{"A"}) {
		t.Fatalf("the lesson list follows the source: %v, checked %v", f.lessonOrder, checkedLessons(f))
	}
	f.sourceSelect.SetSelected("minna no nihongo")
	if !equalIDs(checkedLessons(f), []string{"2", ""}) {
		t.Fatalf("switching back keeps the edited checks: %v", checkedLessons(f))
	}
	f.sourceSelect.SetSelected("JLPT")
	if f.lessonScroll.Visible() || f.words().lessons != nil {
		t.Fatal("JLPT has no lesson filter")
	}
}

func TestSettingsApplySavesTheNewSettings(t *testing.T) {
	test.NewApp()
	gs := settingsGame()
	gs.words = lessonsGame().words
	gs.listNames = []string{"minna-no-nihongo", "other"}
	f := newSettingsForm(gs.settingsState(), nil)
	f.tiles.SetSelected("Words only")
	f.focusCheck.SetChecked(true)
	f.meaningSelect.SetSelected("Deutsch")
	f.lessonChecks["1"].SetChecked(false)
	f.apply(gs)
	if !gs.wordsOnly || !gs.focusWeak || gs.meaningSetting != "de" || gs.meaningLang != "de" {
		t.Fatalf("words-only %v, focus %v, meaning %q/%q", gs.wordsOnly, gs.focusWeak, gs.meaningSetting, gs.meaningLang)
	}
	if got := gs.lessonFilters["minna-no-nihongo"]; !equalIDs(got, []string{"2", ""}) {
		t.Fatalf("lesson filter = %v", got)
	}
}

func TestLessonBoxHiddenForAListWithoutLessons(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(settingsGame().settingsState(), nil) // its list has no lessons
	if f.lessonScroll.Visible() || f.words().lessons != nil {
		t.Fatal("a list without lessons shows no lesson box")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'Settings|LessonBox' -timeout 300s` → build failure.

- [ ] **Step 3: Implement** in `fyne/settings.go`.

`settingsState` gains:

```go
	wordsOnly     bool
	lessons       map[string][]string // per list: its lessons in learning order, "" last when it has words without a lesson
	lessonFilters map[string][]string // per list: the stored lesson filter; absent = all lessons
	meaningLang   string              // the stored meaning language; "" = System
	languages     []string            // the meaning languages of the words, by code
	focusWeak     bool
```

and `settingsState()` fills them under the lock:

```go
	lessons := make(map[string][]string, len(gs.listNames))
	var langs []string
	if gs.words != nil {
		for _, name := range gs.listNames {
			ls, none := gs.words.Lessons(name)
			if none {
				ls = append(ls, "")
			}
			lessons[name] = ls
		}
		langs = gs.words.Languages()
	}
	filters := make(map[string][]string, len(gs.lessonFilters))
	for name, ls := range gs.lessonFilters {
		filters[name] = slices.Clone(ls)
	}
```

with `wordsOnly: gs.wordsOnly, lessons: lessons, lessonFilters: filters, meaningLang: gs.meaningSetting, languages: langs, focusWeak: gs.focusWeak` in the literal. (`Lessons` returns a fresh slice, so appending `""` is safe.)

Tiles choices, near the top:

```go
// The Words tab's "Tiles" choices.
const (
	tilesMixed     = "Kana and words"
	tilesWordsOnly = "Words only"
)
```

`settingsForm` gains:

```go
	tiles         *widget.RadioGroup  // tilesMixed or tilesWordsOnly
	listLessons   map[string][]string // per list: its lessons, "" last for words without one
	lessonFilters map[string][]string // per list: its lesson filter as edited in this dialog
	lessonBox     *fyne.Container     // "All lessons" and the selected list's lesson checks
	lessonScroll  *container.Scroll
	lessonList    string              // the list lessonBox shows; "" = none
	lessonOrder   []string            // its lessons, in order
	lessonChecks  map[string]*widget.Check
	allLessons    *widget.Check
	meaningSelect *widget.Select
	meaningValues []string // the meaning language of each meaningSelect option; "" = System
	focusCheck    *widget.Check
```

`newWordsTab`: build the tiles radio and the lesson box **before** the source select (its `OnChanged`, fired by `SetSelectedIndex`, calls `showLessons`):

```go
	f.tiles = widget.NewRadioGroup([]string{tilesMixed, tilesWordsOnly}, nil)
	f.tiles.Horizontal = true
	f.tiles.Required = true
	f.tiles.SetSelected(tilesMixed)
	if st.wordsOnly {
		f.tiles.SetSelected(tilesWordsOnly)
	}

	f.listLessons = st.lessons
	f.lessonFilters = maps.Clone(st.lessonFilters)
	if f.lessonFilters == nil {
		f.lessonFilters = map[string][]string{}
	}
	f.lessonBox = container.NewVBox()
	f.lessonScroll = container.NewVScroll(f.lessonBox)
	f.lessonScroll.SetMinSize(fyne.NewSize(460, 96))
```

The source select's callback becomes:

```go
	f.sourceSelect = widget.NewSelect(labels, func(string) {
		if f.wordSource() == jlptSource {
			f.wordLevel.Enable()
		} else {
			f.wordLevel.Disable()
		}
		f.showLessons()
	})
```

After the initial `SetSelectedIndex`, call `f.showLessons()` once more (harmless if the callback already ran; needed if it did not). The meaning select:

```go
	mLabels, mValues := meaningLangOptions(st.languages, st.meaningLang)
	f.meaningValues = mValues
	f.meaningSelect = widget.NewSelect(mLabels, nil)
	f.meaningSelect.SetSelectedIndex(slices.Index(mValues, st.meaningLang))
```

The tab:

```go
	return container.NewVBox(
		widget.NewLabelWithStyle("Tiles", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.tiles,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("New words from", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.sourceSelect,
		importButton,
		f.lessonScroll,
		widget.NewLabel("JLPT level"),
		f.wordLevel,
		widget.NewLabel("Words per package"),
		f.packageSize,
		container.NewHBox(widget.NewLabel("Meanings in"), f.meaningSelect),
		widget.NewSeparator(),
		f.summary,
		f.folder,
	)
```

Update `newWordsTab`'s doc: "the tiles (kana and words, or words only), where new words come from, the import button, the lessons of a list, the JLPT level (for JLPT only), the package size, the meaning language, …".

Lesson helpers:

```go
// showLessons fills the lesson box for the selected source: "All lessons"
// and a check per lesson of the list ("No lesson" for words without one),
// checked as the list's filter in this dialog says (all when it has none).
// JLPT and a list without lessons hide the box. The checks of the list
// shown so far are kept first. Must run on the Fyne thread.
func (f *settingsForm) showLessons() {
	f.keepLessons()
	f.lessonList, f.lessonOrder, f.lessonChecks, f.allLessons = "", nil, nil, nil
	f.lessonBox.RemoveAll()
	name, isList := strings.CutPrefix(f.wordSource(), listSourcePrefix)
	lessons := f.listLessons[name]
	if !isList || !slices.ContainsFunc(lessons, func(l string) bool { return l != "" }) {
		f.lessonScroll.Hide()
		return
	}
	chosen := f.lessonFilters[name]
	f.lessonList, f.lessonOrder = name, lessons
	f.lessonChecks = make(map[string]*widget.Check, len(lessons))
	checks := make([]*widget.Check, 0, len(lessons))
	grid := container.NewGridWithColumns(4)
	for _, l := range lessons {
		c := widget.NewCheck(lessonLabel(l), nil)
		c.SetChecked(len(chosen) == 0 || slices.Contains(chosen, l))
		f.lessonChecks[l] = c
		checks = append(checks, c)
		grid.Add(c)
	}
	f.allLessons = widget.NewCheck("All lessons", nil)
	f.allLessons.SetChecked(allChecked(checks))
	linkAllCheck(f.allLessons, checks)
	f.lessonBox.Add(f.allLessons)
	f.lessonBox.Add(grid)
	f.lessonScroll.Show()
}

// keepLessons keeps the lesson checks shown as that list's filter in this
// dialog, so switching sources back and forth keeps them.
func (f *settingsForm) keepLessons() {
	if f.lessonList != "" {
		f.lessonFilters[f.lessonList] = f.chosenLessons()
	}
}

// chosenLessons is the filter the lesson checks shown give: the checked
// lessons in list order; nil (all lessons) when all or none are checked.
func (f *settingsForm) chosenLessons() []string {
	var out []string
	for _, l := range f.lessonOrder {
		if f.lessonChecks[l].Checked {
			out = append(out, l)
		}
	}
	if len(out) == len(f.lessonOrder) {
		return nil
	}
	return out
}

// selectedLessons is the lesson filter to save for the selected source:
// the checks for a list that shows them, the list's filter as it is for
// one that does not (a missing list keeps its filter), nil for JLPT.
func (f *settingsForm) selectedLessons() []string {
	name, ok := strings.CutPrefix(f.wordSource(), listSourcePrefix)
	switch {
	case !ok:
		return nil
	case name == f.lessonList:
		return f.chosenLessons()
	}
	return f.lessonFilters[name]
}

// lessonLabel is a lesson's check label: its name, "No lesson" for words
// without one.
func lessonLabel(l string) string {
	if l == "" {
		return "No lesson"
	}
	return l
}

// meaningLang is the meaning language the Words tab has selected; "" =
// System.
func (f *settingsForm) meaningLang() string {
	if i := f.meaningSelect.SelectedIndex(); i >= 0 {
		return f.meaningValues[i]
	}
	return ""
}
```

Extract the sync logic of `newGroupChecks` into a shared helper and call it there (`linkAllCheck(all, checkList)` replaces the `syncing` block):

```go
// linkAllCheck keeps all and checks in sync: checking or unchecking all
// sets every check, and a check's change sets all to whether every check is
// on. A syncing flag guards against the handlers recursing into each other.
func linkAllCheck(all *widget.Check, checks []*widget.Check) {
	syncing := false
	all.OnChanged = func(on bool) {
		if syncing {
			return
		}
		syncing = true
		for _, c := range checks {
			c.SetChecked(on)
		}
		syncing = false
	}
	for _, c := range checks {
		c.OnChanged = func(bool) {
			if syncing {
				return
			}
			syncing = true
			all.SetChecked(allChecked(checks))
			syncing = false
		}
	}
}
```

`words()`:

```go
	return wordSettings{source: f.wordSource(), level: level, packageSize: size,
		wordsOnly: f.tiles.Selected == tilesWordsOnly, lessons: f.selectedLessons(), meaningLang: f.meaningLang()}
```

`newGameTab`: `f.focusCheck = widget.NewCheck("Focus on weak items", nil)`, `f.focusCheck.SetChecked(st.focusWeak)`, placed after the points row. `apply`: after `gs.SetKanaFont(f.kanaFontID())` add `gs.SetFocusWeak(f.focusCheck.Checked)`.

`setLists` ends with `f.showLessons()` (selecting the same index again fires no `OnChanged`). In `fyne/importlist.go` (around line 145) refresh the lessons with the lists:

```go
		st := gs.settingsState()
		f.listLessons = st.lessons
		f.setLists(st.lists, name)
		// The summary may have said the chosen list was missing.
		f.summary.SetText(st.summary)
```

The dialog must still fit the 1000 × 760 window: run `TestSettingsDialogSaveAndInvalidValue` and, if the Words tab overflows, wrap the Words tab's VBox in `container.NewVScroll` (as the kana rows are).

- [ ] **Step 4: Run** `go test ./... -timeout 300s` → PASS (the existing settings tests too: `TestSettingsFormCarriesValues` compares `words()` with `reflect.DeepEqual` since Task 6; with System selected and JLPT chosen the new fields are zero).

---

### Task 10: README

**Files:** Modify `README.md`.

- [ ] **Step 1:** In the features list (the word bullet ending "longer words fall more slowly (up to twice the time).", around line 34) add a bullet after it:

  - **Words only**: Settings → Words → Tiles "Words only" lets only words fall: every word you were introduced to (of the chosen source and lessons, in an active script) falls, whether or not you have learned its kana, and new word packages come the same way. The kana learning path waits meanwhile (no new rows, no katakana offer) and goes on when you switch back. If no word can fall (no word in the chosen lessons, list missing), kana fall and the status panel says why.

- [ ] **Step 2:** In "Spaced repetition" add a bullet: **Focus on weak items** (Settings → Game): half the tiles are drawn from what you are learning (levels 1–4) or slipped, slipped ones twice as often; the other half and all new items come as usual. It does not change how often a word falls.

- [ ] **Step 3:** In "Customization", extend the **Word packages** bullet: "…, the lessons of a list (checkboxes with "All lessons" and "No lesson", stored per list: only these lessons' words fall and new packages come from the earliest chosen lesson), the package size (3, 5 or 10) and the meaning language ("Meanings in": System, or any language of the word data; English answers are always accepted)".

- [ ] **Step 4:** In "Private word lists", after the paragraph on choosing a list, add: "Choose lessons below the source to practise only them (all lessons by default); renaming a list file forgets its lesson choice."

- [ ] **Step 5:** In the structure section: `settings.go` — "Words (tiles, source, lessons, JLPT level, package size, meaning language), Game (difficulty level, points per hit, focus on weak items, score limit, kana font)"; `store/store.go` — add `words_only`, `focus_weak`, `lesson_filter:<list>` to the word settings; add `focus.go` under the `fyne/` files ("the focus pick on weak items") if the section lists them.

---

### Task 11: Final check (controller)

- [ ] `go test ./... -race -timeout 600s`, `go vet ./...`, `gofmt -l .` (empty).
- [ ] Manual: `go run ./fyne/`. Settings → Words: switch to "Words only" — only words fall, also ones with unlearned kana; the status panel's kana line does not change. Choose a list, uncheck lessons — only the chosen lessons' words fall; the next package comes from the earliest chosen lesson. Choose "Meanings in: Français" — feedback shows French, English still answers. Settings → Game → "Focus on weak items" — slipped kana come back noticeably often. Switch back to "Kana and words" — a kana intro that waited is shown.
