# Reading Practice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A saved checkbox "Reading practice" in the Words tab lets any word of the chosen source whose kana are all learned fall, introduced or not; a word read counts as a hit on each kana unit of the reading typed, never for the word, and the status line shows how many words can be read and the round's kana per minute.

**Architecture:** `store` gets a `reading_practice` flag; `vocab.Entry.MatchedReading` tells which of a word's readings the input was. In `fyne` a new file `reading.go` holds the readable-word pool (cached next to the playable cache and invalidated with it), the uniform pick, the kana credit and the kana/min figure. Every tile spawned in reading practice carries `KanaTile.practice`, so answers, hits, misses and dropping go by the tile, not by the current setting. `recordCorrect` is split into `creditKana` (stats, streak, memory, level) and `progressAfterHit` (auto-progression), so a word read credits each unit through the same path as a kana tile and raises the tempo once. `packageDue` returns nothing while reading practice is on. Play time is counted in ticks, which already stop during pauses and dialogs.

**Tech Stack:** Go 1.25 (toolchain 1.27), Fyne v2.7.3, SQLite (modernc.org/sqlite). No new dependencies.

Spec: `docs/superpowers/specs/2026-10-24-reading-practice-design.md`. Terms: `GLOSSARY.md` (reading practice, reading, learned, learning level, words-only mode, meaning mode, lesson filter, word list). The GLOSSARY already has every term; no GLOSSARY task.

**Conventions:** comments like the surrounding code (short doc comment on every function, "Must be called under lock." where it applies). Every `GameState` field is read and written under `gs.mu`; the `fyne` tests touch fields directly without the lock, as the existing tests do. **Test data is synthetic only** (invented words such as かさ/umbrella, あい/love); every new `fyne` test that calls `NewGameState` calls `isolateWordsDir(t)` first, and nothing may read anything under `~/.local/share/kana/` (private lists live there). Quick checks per task: `go build ./... && go vet ./...`, `gofmt -l .` (empty; run `gofmt -w` on the listed files when it is not, e.g. after adding fields to an aligned struct) and the named `-run` tests; `go test ./fyne/` takes several minutes, so the whole package runs only where a step says so. Commit messages: conventional subject of at most 72 characters plus one short body paragraph, written to a file and committed with `git commit -F <file>`; no Co-Authored-By line, no mention of AI, no issue tracker identifiers. Implementers may leave the commit to the controller, who reviews and commits with the given message. `CLAUDE.md` is not in the repository and is not touched.

---

## Decisions made in the plan

The spec leaves these open or calls them indicative; check them before Task 1.

1. **The setting:** key `reading_practice`, a flag like `words_only` (`ReadingPractice()`/`SaveReadingPractice()`), default off. `GameState.readingPractice`, `wordSettings.readingPractice`, saved by `applyWordSettings`.
2. **The readable pool** (`gs.readableWords()`): every entry of `gs.words.Entries()` with `src.Has(e)` for the source from `gs.source()` (JLPT up to `gs.wordLevel`, or the chosen list with its effective lesson filter) and `gs.unitsKnown(e.Units)` (every kana learned and in an active script; っ/ッ/ー need their script). None when the source is missing. It ignores the introduced words, words-only mode, "Words ask for" and whether the introduced words loaded — reading practice touches no word progress. Sorted by reading (the index order). It is cached like `playableWords` and invalidated by the same `invalidatePlayable`, whose triggers (a kana learned or unlearned, scripts, index, source, lesson filter, JLPT level) are exactly what the pool depends on.
3. **Which reading was typed** (`vocab.Entry.MatchedReading`): the entry's reading if the input matches it, else the first of its other readings (`Alts`, in their order) that does. The reading is the shown variant, so when two variants read the same (あめりか // アメリカ, both `amerika`) the shown one gets the credit. `ReadingMatches` becomes a thin wrapper.
4. **What counts as a kana unit:** the units of the typed reading (`kanacore.KanaUnits`) that are kana of a row (`kanacore.CharToRow`), yōon such as きゃ as one unit. っ, ッ and ー count for nothing, as they have no level of their own (`unitKnown` treats them the same way). Only units that pass `gs.unitKnown` (learned and in an active script) are credited: the pool guarantees the shown reading's kana are known, but another variant or merged reading may hold others (あした // あす, typed "ashita": し and た may be unlearned or in an unselected row).
5. **The credit path:** `recordCorrect(char)` becomes `creditKana(char)` + `raiseTempo()` + `progressAfterHit()`. A word read calls `creditKana` per unit, then `raiseTempo` once and `progressAfterHit` once; the result (rows announced) suppresses the katakana offer as for a kana hit. Auto-progression and the katakana offer stay off in words-only mode, as today.
6. **Tiles remember their rules:** `KanaTile.practice bool`, set at spawn when reading practice is on. A practice tile is a plain word tile asking for the reading (`ask == readingKind`, no marker), whatever "Words ask for" says. `answers`, `checkAnswer`, `wrongInput`, the miss in `tick` and `dropInactiveTiles` go by `tile.practice`; a word tile spawned before the change keeps counting for the word, and a practice tile keeps counting for its kana after reading practice is turned off.
7. **Answers on a practice tile:** only `ReadingMatches`; a meaning answers nothing. No "type the meaning" hint (practice tiles never ask for the meaning).
8. **A wrong input on a practice tile** (the only tile on the field) changes nothing — no kana and no word mistake — because the spec says a word in reading practice changes no word and only a reading counts for kana. The tile stays.
9. **A practice tile falling through:** counts towards the miss limit, lowers the tempo (`lowerTempo`), is listed in the game-over missed words and shows the miss label as today; no kana and no word is touched.
10. **Feedback after a hit:** as today for a reading hit, without romaji: `hitFeedbackText(e, lang, false)`, e.g. "かさ — umbrella" (decided with the user; the learner just typed the romaji).
11. **Selection:** `pickWord` delegates to `pickReadable` while reading practice is on: the word share is drawn as today from `wordShare(len(pool))` (mixed mode; words-only always a word), the word uniformly with `gs.rnd` over `wordCandidates(pool)` (falling words avoided unless that leaves none). No focus pick, no memory pick.
12. **Packages and level offer:** `packageDue` returns nothing while reading practice is on, which covers `maybeOfferPackage` (every re-check point, including `Reset`) and `offerPackageAtStart`. Turning reading practice off in the settings re-checks at once (`applySettings` already calls `maybeOfferPackage`). A package or level offer already pending when it is turned on is still shown.
13. **Dropping tiles on a settings change:** a practice tile stays while its word is in the readable pool (not the playable words), a word tile while its word is playable, as today. Turning reading practice on or off drops nothing (the pool does not depend on the flag).
14. **kana/min:** `GameState.readKana` counts the credited units this round, `GameState.playTicks` the ticks of play this round. `tick` returns early while paused (learner, settings, import, any dialog) or over, so counting there excludes pauses and dialogs without new bookkeeping and is testable by calling `tick()`. kana/min = `round(readKana / (playTicks / 600))`. `Reset` zeroes both. Both count the whole round, also while reading practice is off (the figure is shown only while it is on). The figure shows "–" while `readKana == 0`.
15. **Status line precedence** in reading practice: with readable words "Reading practice: N words to read · N kana/min" ("1 word"); with none, the existing messages when the bundled list failed to load or the chosen list is missing (they say what to do), else "No words to read yet — learn more kana or pick another list" — also in words-only mode, where kana fall. The " · N rows skipped" suffix stays. The Words tab summary in the settings is unchanged.
16. **Statistics panel:** "Correct" counts session kana hits plus word hits; a word read adds its kana units (they are kana hits in `sessionStats`) and no word hit, as the spec says "the statistics show them like other kana hits".
17. **Settings dialog:** the check "Reading practice" is its own row of the Words tab's top form, under "Words ask for", with an empty label cell. While it is checked, the "Words ask for" radio group is disabled; its selection stays and is saved unchanged.
18. **Task order:** storage (1–2), the setting in the game (3), the pool (4), the package gate (5, before anything spawns practice tiles, so tests need no `levelOfferDeclined`), spawning (6), answers and credit (7), misses and dropping (8), kana/min and the status line (9), the dialog (10), README (11).

---

## File structure

- Modify `store/store.go`, `store/store_test.go`: `readingPracticeKey`, `ReadingPractice`, `SaveReadingPractice`.
- Modify `vocab/index.go`, `vocab/index_test.go`: `Entry.MatchedReading`; `ReadingMatches` uses it.
- Create `fyne/reading.go`, `fyne/reading_test.go`: `readableWords`, `readingSet`, `pickReadable`, `recordReading`, `kanaPerMinute`.
- Modify `fyne/game.go`: fields `readingPractice`, `readable`, `readableOK`, `readKana`, `playTicks`; `NewGameState`, `Reset`, `tick`, `spawnKana`, `checkAnswer`, `wrongInput`, `answers`, `recordCorrect` → `creditKana` + `progressAfterHit`, `dropInactiveTiles`.
- Modify `fyne/wordgame.go`: `invalidatePlayable`, `pickWord`, `wordsStatus`.
- Modify `fyne/packages.go`: `wordSettings.readingPractice`, `packageDue`, `applyWordSettings`.
- Modify `fyne/tile.go`: `KanaTile.practice`.
- Modify `fyne/words.go`: `WordsStatus` fields, `noWordsToReadText`, `formatWordsLine`.
- Modify `fyne/settings.go`: `settingsState.readingPractice`, `settingsForm.readingCheck`, `newWordsTab`, `syncWordsAsk`, `words`.
- Modify tests `fyne/words_test.go`, `fyne/settings_test.go`.
- Modify `README.md`.

---

### Task 1: Store — the reading-practice setting

**Files:**
- Modify: `store/store.go`
- Test: `store/store_test.go`

- [ ] **Step 1: Write the failing test.** Append to `store/store_test.go`:

```go
func TestReadingPractice(t *testing.T) {
	st := openTemp(t)
	if on, err := st.ReadingPractice(); err != nil || on {
		t.Fatalf("unset ReadingPractice = %v, %v; want false", on, err)
	}
	if err := st.SaveReadingPractice(true); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.ReadingPractice(); !on {
		t.Fatal("ReadingPractice not saved")
	}
	if err := st.SaveReadingPractice(false); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.ReadingPractice(); on {
		t.Fatal("ReadingPractice off not saved")
	}
}
```

- [ ] **Step 2: Run** `go test ./store/ -run TestReadingPractice` → build failure: `st.ReadingPractice undefined`.

- [ ] **Step 3: Implement.** In `store/store.go`, in the key constants after `wordsAskKey        = "words_ask"`, add:

```go
	readingPracticeKey = "reading_practice"
```

After `SaveFocusWeak`, add:

```go
// ReadingPractice reports whether reading practice is on; false when unset.
func (s *Store) ReadingPractice() (bool, error) { return s.flag(readingPracticeKey) }

// SaveReadingPractice persists reading practice.
func (s *Store) SaveReadingPractice(on bool) error { return s.saveFlag(readingPracticeKey, on) }
```

- [ ] **Step 4: Run** `go test ./store/ && go build ./... && go vet ./... && gofmt -l .` → PASS, no gofmt output.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(store): store whether reading practice is on

Reading practice is a saved flag like words-only mode, off by default.
EOF
git add store/store.go store/store_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 2: `vocab.Entry.MatchedReading`

**Files:**
- Modify: `vocab/index.go`
- Test: `vocab/index_test.go`

- [ ] **Step 1: Write the failing test.** Append to `vocab/index_test.go`:

```go
func TestEntryMatchedReading(t *testing.T) {
	e := Entry{Kana: "どうぞよろしく", Alts: []string{"よろしく"}, Phrase: true}
	for in, want := range map[string]string{
		"douzoyoroshiku": "どうぞよろしく",
		"yoroshiku":      "よろしく",
		"douzo":          "",
		"":               "",
	} {
		got, ok := e.MatchedReading(in)
		if got != want || ok != (want != "") {
			t.Errorf("MatchedReading(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	v := Entry{Kana: "ローソク", Alts: []string{"ろうそく"}}
	if got, _ := v.MatchedReading("roosoku"); got != "ローソク" {
		t.Errorf("roosoku gives %q, want ローソク", got)
	}
	if got, _ := v.MatchedReading("rousoku"); got != "ろうそく" {
		t.Errorf("rousoku gives %q, want ろうそく", got)
	}
	same := Entry{Kana: "アメリカ", Alts: []string{"あめりか"}}
	if got, _ := same.MatchedReading("amerika"); got != "アメリカ" {
		t.Errorf("variants that read the same: %q, want the shown reading アメリカ", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./vocab/ -run TestEntryMatchedReading` → build failure: `e.MatchedReading undefined`.

- [ ] **Step 3: Implement.** In `vocab/index.go`, replace `ReadingMatches` with:

```go
// ReadingMatches reports whether input is a romaji reading of the entry's
// reading or of one of its other accepted readings (see MatchedReading).
func (e Entry) ReadingMatches(input string) bool {
	_, ok := e.MatchedReading(input)
	return ok
}

// MatchedReading returns the reading input is a romaji reading of (see
// kanacore.ReadingMatches; kanacore.PhraseReadingMatches for a phrase): the
// entry's reading if it matches, else the first of its other readings that
// does. ok is false when none matches.
func (e Entry) MatchedReading(input string) (reading string, ok bool) {
	matches := kanacore.ReadingMatches
	if e.Phrase {
		matches = kanacore.PhraseReadingMatches
	}
	if matches(e.Kana, input) {
		return e.Kana, true
	}
	for _, a := range e.Alts {
		if matches(a, input) {
			return a, true
		}
	}
	return "", false
}
```

- [ ] **Step 4: Run** `go test ./vocab/ && go build ./... && go vet ./... && gofmt -l .` → PASS (the existing `TestEntryReadingMatches*` tests pin the unchanged matching).

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(vocab): tell which reading of a word was typed

Reading practice credits the kana of the form typed, so a short form
counts only its own kana. The shown reading wins when two read the same.
EOF
git add vocab/index.go vocab/index_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 3: The setting in the game

**Files:**
- Modify: `fyne/game.go` (field, `NewGameState`)
- Modify: `fyne/packages.go` (`wordSettings`, `applyWordSettings`)
- Create: `fyne/reading_test.go`

- [ ] **Step 1: Write the failing test.** Create `fyne/reading_test.go`:

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestReadingPracticeIsStored(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	if gs.readingPractice {
		t.Fatal("reading practice must be off by default")
	}
	ws := wordSettings{source: gs.wordSource, level: gs.wordLevel, packageSize: gs.packageSize, readingPractice: true}
	gs.applyWordSettings(ws)
	if on, _ := st.ReadingPractice(); !on || !gs.readingPractice {
		t.Fatalf("stored %v, state %v", on, gs.readingPractice)
	}
	if !NewGameState(st).readingPractice {
		t.Fatal("reading practice not loaded")
	}
	ws.readingPractice = false
	gs.applyWordSettings(ws)
	if on, _ := st.ReadingPractice(); on || gs.readingPractice {
		t.Fatalf("off: stored %v, state %v", on, gs.readingPractice)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run TestReadingPracticeIsStored` → build failure: `gs.readingPractice undefined`.

- [ ] **Step 3: Implement.**

In `fyne/game.go`, in `GameState` after the `wordsAsk` field, add:

```go
	readingPractice      bool                // reading practice: every readable word may fall and counts for its kana (see reading.go)
```

In `NewGameState`, after the block that loads `WordsAsk`, add:

```go
		if on, err := st.ReadingPractice(); err == nil {
			gs.readingPractice = on
		}
```

In `fyne/packages.go`, add to `wordSettings` after `ask`:

```go
	readingPractice bool // reading practice is on
```

In `applyWordSettings`, after the `ws.ask` block, add:

```go
	if ws.readingPractice != gs.readingPractice {
		// Only new tiles follow; falling tiles keep their rules (see
		// KanaTile.practice), and the readable words do not depend on
		// it, so nothing is dropped.
		gs.readingPractice = ws.readingPractice
		if gs.store != nil {
			_ = gs.store.SaveReadingPractice(ws.readingPractice)
		}
	}
```

Extend the doc comment of `applyWordSettings`: after "A new \"Words ask for\" applies to tiles spawned afterwards." add " So does reading practice."

- [ ] **Step 4: Run** `gofmt -w fyne/game.go fyne/packages.go && go test ./fyne/ -run 'TestReadingPracticeIsStored|TestWordsAskIsStored' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): keep the reading practice setting in the game

The game loads the flag at start and the Words settings save it. It
changes nothing yet.
EOF
git add fyne/game.go fyne/packages.go fyne/reading_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 4: The readable-word pool

**Files:**
- Create: `fyne/reading.go`
- Modify: `fyne/game.go` (cache fields), `fyne/wordgame.go` (`invalidatePlayable`)
- Test: `fyne/reading_test.go`

- [ ] **Step 1: Write the failing tests.** In `fyne/reading_test.go`, extend the imports to:

```go
import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/vocab"
)
```

and append:

```go
// readableKana lists the readings of gs.readableWords().
func readableKana(gs *GameState) []string {
	var out []string
	for _, e := range gs.readableWords() {
		out = append(out, e.Kana)
	}
	return out
}

func TestReadablePoolTakesEveryWordWhoseKanaAreLearned(t *testing.T) {
	for _, wordsOnly := range []bool{false, true} {
		gs := newTestState() // hiragana only
		gs.wordsOnly = wordsOnly
		withWords(gs, bw("かさ", 5, "umbrella"), bw("かき", 5, "persimmon"), bw("さか", 5, "slope"), bw("カサ", 5, "umbrella"))
		gs.introduced = map[string]bool{"さか": true} // introduced or not makes no difference
		if got := readableKana(gs); len(got) != 0 {
			t.Fatalf("wordsOnly=%v: %v readable before any kana is learned", wordsOnly, got)
		}
		learnKana(gs, "か", "さ", "カ", "サ")
		if got := readableKana(gs); !equalIDs(got, []string{"かさ", "さか"}) {
			t.Fatalf("wordsOnly=%v: readable %v; かき has an unlearned kana, カサ an inactive script", wordsOnly, got)
		}
		gs.activeScripts[kanacore.ScriptKatakana] = true
		gs.invalidatePlayable()
		if got := readableKana(gs); !equalIDs(got, []string{"かさ", "さか", "カサ"}) {
			t.Fatalf("wordsOnly=%v: with katakana on: %v", wordsOnly, got)
		}
		gs.applyKanaMistake("さ")
		if got := readableKana(gs); !equalIDs(got, []string{"カサ"}) {
			t.Fatalf("wordsOnly=%v: さ no longer learned: %v", wordsOnly, got)
		}
	}
}

func TestReadablePoolFollowsTheLessonFilter(t *testing.T) {
	gs := newTestState()
	words := []vocab.Word{lw("minna", "あい", "1"), lw("minna", "あお", "2"), lw("minna", "いえ", "")}
	withList(gs, "minna", words...)
	gs.words = vocab.NewIndex([]vocab.Word{bw("うえ", 5, "above")}, []vocab.List{{Name: "minna", Words: words}})
	gs.introduced = map[string]bool{}
	learnKana(gs, "あ", "い", "う", "え", "お")
	if got := readableKana(gs); !equalIDs(got, []string{"あい", "あお", "いえ"}) {
		t.Fatalf("all lessons: %v (うえ is not in the list)", got)
	}
	gs.lessonFilters = map[string][]string{"minna": {"1"}}
	gs.invalidatePlayable()
	if got := readableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("lesson 1: %v", got)
	}
	gs.lessonFilters = map[string][]string{"minna": {"2", ""}}
	gs.invalidatePlayable()
	if got := readableKana(gs); !equalIDs(got, []string{"あお", "いえ"}) {
		t.Fatalf("lesson 2 and no lesson: %v", got)
	}
}

func TestReadablePoolFollowsTheJLPTLevel(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("いえ", 4, "house"), bw("うえ", 3, "above"))
	learnKana(gs, "あ", "い", "う", "え", "お")
	if got := readableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("N5: %v", got)
	}
	gs.wordLevel = 4
	gs.invalidatePlayable()
	if got := readableKana(gs); !equalIDs(got, []string{"あい", "いえ"}) {
		t.Fatalf("N4: %v", got)
	}
	gs.wordSource = listSourcePrefix + "gone"
	gs.invalidatePlayable()
	if got := readableKana(gs); len(got) != 0 {
		t.Fatalf("a missing list has no readable words, got %v", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run TestReadablePool` → build failure: `gs.readableWords undefined`.

- [ ] **Step 3: Implement.**

In `fyne/game.go`, in `GameState` after the `playableOK` field, add:

```go
	readable                  []vocab.Entry // cached readableWords result
	readableOK                bool          // readable is up to date
```

In `fyne/wordgame.go`, replace `invalidatePlayable` with:

```go
// invalidatePlayable marks the playable and the readable cache stale. Call
// it whenever a kana may have become learned or stopped being learned, or
// the active scripts, the introduced words, the index, the mode, the source
// or its lesson filter changed. Must be called under lock.
func (gs *GameState) invalidatePlayable() {
	gs.playableOK = false
	gs.readableOK = false
}
```

Create `fyne/reading.go`:

```go
package main

import "kana/vocab"

// Reading practice: every word of the source whose kana are all learned
// may fall, introduced or not, and a word read counts for its kana, not
// for the word.

// readableWords returns the words that may fall in reading practice: every
// entry of the source (see source: JLPT up to the chosen level, or the
// chosen list with its lesson filter) whose units are all known (see
// unitsKnown), introduced or not, sorted by reading. None while the source
// is missing. The result is cached until invalidatePlayable; do not modify
// it. Must be called under lock.
func (gs *GameState) readableWords() []vocab.Entry {
	if gs.words == nil {
		return nil
	}
	if !gs.readableOK {
		var out []vocab.Entry
		if src, found := gs.source(); found {
			for _, e := range gs.words.Entries() {
				if src.Has(e) && gs.unitsKnown(e.Units) {
					out = append(out, e)
				}
			}
		}
		gs.readable = out
		gs.readableOK = true
	}
	return gs.readable
}
```

- [ ] **Step 4: Run** `gofmt -w fyne/game.go && go test ./fyne/ -run 'TestReadablePool|TestPlayable|TestLessonFilter' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): find the words a learner can read

Every word of the chosen source whose kana are all learned and in an
active script, introduced or not, cached next to the playable words.
EOF
git add fyne/reading.go fyne/reading_test.go fyne/game.go fyne/wordgame.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 5: No packages or level offer in reading practice

**Files:**
- Modify: `fyne/packages.go` (`packageDue`)
- Test: `fyne/reading_test.go`

- [ ] **Step 1: Write the failing test.** Append to `fyne/reading_test.go`:

```go
func TestReadingPracticeHoldsPackagesAndTheLevelOffer(t *testing.T) {
	gs := packageReady(10)
	gs.readingPractice = true
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 || gs.paused || len(drainEvents(gs)) != 0 {
		t.Fatal("no package may come while reading practice is on")
	}
	gs.readingPractice = false
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) == 0 {
		t.Fatal("packages resume when reading practice is off")
	}

	lv := levelDone()
	lv.readingPractice = true
	lv.maybeOfferPackage()
	if lv.pendingLevelOffer || lv.paused {
		t.Fatal("no level offer while reading practice is on")
	}
	lv.readingPractice = false
	lv.maybeOfferPackage()
	if !lv.pendingLevelOffer {
		t.Fatal("the level offer resumes when reading practice is off")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run TestReadingPracticeHoldsPackages` → FAIL: "no package may come while reading practice is on".

- [ ] **Step 3: Implement.** In `fyne/packages.go`, in `packageDue`, change the first condition to:

```go
	if gs.over || gs.readingPractice || gs.words == nil || gs.introducedLoadFailed || len(gs.pendingPackage) > 0 || gs.pendingLevelOffer {
```

and in its doc comment change "Nothing is due when over," to "Nothing is due when over, in reading practice,".

- [ ] **Step 4: Run** `go test ./fyne/ -run 'TestReadingPracticeHoldsPackages|TestLevelOffer|TestPackage' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): hold word packages while reading practice is on

Reading practice needs no introductions, so neither packages nor the
JLPT level offer come; both resume when it is turned off.
EOF
git add fyne/packages.go fyne/reading_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 6: Spawning practice tiles

**Files:**
- Modify: `fyne/tile.go` (`KanaTile.practice`), `fyne/game.go` (`spawnKana`), `fyne/wordgame.go` (`pickWord`), `fyne/reading.go` (`pickReadable`)
- Test: `fyne/reading_test.go`

- [ ] **Step 1: Write the failing tests.** In `fyne/reading_test.go`, add `"math/rand"` to the imports and append:

```go
func TestReadingPracticeSpawnsReadableWords(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.readingPractice = true
	gs.wordsAsk = meaningKind // practice tiles ask for the reading all the same
	gs.rnd = rand.New(rand.NewSource(3)).Float64
	withWords(gs, bw("あい", 5, "love"), bw("あか", 5, "red"))
	gs.introduced = map[string]bool{}
	gs.invalidatePlayable()

	gs.spawnKana()
	if gs.tiles[0].word != nil {
		t.Fatal("no word to read yet: a kana falls")
	}
	learnKana(gs, "あ", "い", "う", "え", "お")
	for i := 0; i < 50; i++ {
		gs.tiles = nil
		gs.spawnKana()
		tile := gs.tiles[0]
		if tile.word == nil || !tile.practice || tile.ask != readingKind || tile.marker != nil {
			t.Fatalf("spawn %d: want a practice tile asking for the reading, got %+v", i, tile)
		}
		if tile.word.Kana != "あい" {
			t.Fatalf("spawn %d: %s has an unlearned kana", i, tile.word.Kana)
		}
	}
	gs.readingPractice = false
	gs.tiles = nil
	gs.spawnKana()
	if gs.tiles[0].word != nil {
		t.Fatal("reading practice off: あい is not introduced and must not fall")
	}
}

func TestReadingPracticePicksUniformlyAndAvoidsFallingWords(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.readingPractice = true
	gs.focusWeak = true // applies to kana tiles only
	gs.rnd = rand.New(rand.NewSource(5)).Float64
	words := vowelWords(5)
	withWords(gs, words...)
	gs.introduced = map[string]bool{}
	learnKana(gs, "あ", "い", "う", "え", "お")
	setWordLevel(gs, words[0].Kana, 2) // weak, yet not favoured

	seen := map[string]int{}
	for i := 0; i < 1000; i++ {
		gs.tiles = nil
		gs.spawnKana()
		seen[gs.tiles[0].word.Kana]++
	}
	for _, w := range words {
		if n := seen[w.Kana]; n < 150 || n > 250 {
			t.Fatalf("%s spawned %d/1000 times, want about 200 (uniform): %v", w.Kana, n, seen)
		}
	}

	var falling []*KanaTile
	for _, w := range words[:4] {
		falling = append(falling, practiceTileAt(gs, w.Kana, 100))
	}
	for i := 0; i < 20; i++ {
		gs.tiles = append([]*KanaTile(nil), falling...)
		gs.spawnKana()
		if got := gs.tiles[len(gs.tiles)-1].word.Kana; got != words[4].Kana {
			t.Fatalf("spawned %s, which is already falling", got)
		}
	}
}

func TestReadingPracticeKeepsTheWordShareInMixedMode(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.readingPractice = true
	gs.rnd = rand.New(rand.NewSource(9)).Float64
	withWords(gs, vowelWords(60)...)
	gs.introduced = map[string]bool{}
	learnKana(gs, "あ", "い", "う", "え", "お")
	if n := countWordSpawns(gs, 1000); n < 60 || n > 140 {
		t.Fatalf("%d/1000 word spawns, want about 10 %%", n)
	}
}

// practiceTileAt is a word tile spawned in reading practice at (10, y) for
// a reading in gs.words.
func practiceTileAt(gs *GameState, kana string, y float32) *KanaTile {
	tile := wordTileAt(gs, kana, y)
	tile.practice = true
	return tile
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestReadingPractice(Spawns|Picks|Keeps)'` → build failure: `tile.practice undefined`.

- [ ] **Step 3: Implement.**

In `fyne/tile.go`, add to `KanaTile` after `ask`:

```go
	practice    bool         // spawned in reading practice: answered by a reading only, counted for its kana
```

In `fyne/game.go`, in `spawnKana`, replace

```go
		if gs.wordsAsk == meaningKind {
			tile = newMeaningTile(entry, gs.canvasW, gs.kanaFont)
		} else {
			tile = newWordTile(entry, gs.canvasW, gs.kanaFont)
		}
```

with

```go
		switch {
		case gs.readingPractice:
			tile = newWordTile(entry, gs.canvasW, gs.kanaFont)
			tile.practice = true
		case gs.wordsAsk == meaningKind:
			tile = newMeaningTile(entry, gs.canvasW, gs.kanaFont)
		default:
			tile = newWordTile(entry, gs.canvasW, gs.kanaFont)
		}
```

and in its doc comment replace "A word tile asks for what gs.wordsAsk says." with "A word tile asks for what gs.wordsAsk says; in reading practice it is a practice tile asking for the reading (see pickReadable)."

In `fyne/wordgame.go`, at the top of `pickWord`'s body add:

```go
	if gs.readingPractice {
		return gs.pickReadable(always)
	}
```

and append to its doc comment: "In reading practice pickReadable decides instead."

In `fyne/reading.go`, append:

```go
// pickReadable is pickWord in reading practice: whether this spawn is a
// word is drawn with gs.rnd as in pickWord, over the readable words; the
// word is drawn uniformly with gs.rnd among those not already falling (see
// wordCandidates). The focus on weak items applies to kana tiles only.
// With no readable word it returns false without drawing. Must be called
// under lock.
func (gs *GameState) pickReadable(always bool) (vocab.Entry, bool) {
	pool := gs.readableWords()
	if len(pool) == 0 || (!always && gs.rnd() >= wordShare(len(pool))) {
		return vocab.Entry{}, false
	}
	candidates := gs.wordCandidates(pool)
	i := min(int(gs.rnd()*float64(len(candidates))), len(candidates)-1)
	return gs.words.Lookup(candidates[i])
}
```

- [ ] **Step 4: Run** `gofmt -w fyne/tile.go && go test ./fyne/ -run 'TestReadingPractice|TestWordShare|TestWordsOnly|TestMeaningModeSpawns' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): let readable words fall in reading practice

Any word whose kana are learned may fall, picked uniformly and not twice
at once; the tile remembers it was spawned in reading practice.
EOF
git add fyne/tile.go fyne/game.go fyne/wordgame.go fyne/reading.go fyne/reading_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 7: A word read counts for its kana

**Files:**
- Modify: `fyne/game.go` (`readKana` field, `answers`, `checkAnswer`, `wrongInput`, `recordCorrect` → `creditKana` + `progressAfterHit`), `fyne/reading.go` (`recordReading`)
- Test: `fyne/reading_test.go`

- [ ] **Step 1: Write the failing tests.** In `fyne/reading_test.go`, add `"kana/store"` to the imports and append:

```go
func TestReadingHitCreditsItsKanaNotTheWord(t *testing.T) {
	gs := newTestState()
	gs.readingPractice = true
	withWords(gs, bw("かさ", 5, "umbrella")) // introduced: its word progress stays all the same
	setWordLevel(gs, "かさ", 2)
	learnKana(gs, "か", "さ")
	gs.tiles = []*KanaTile{practiceTileAt(gs, "かさ", 100)}
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 0 {
		t.Fatal("the reading answers the tile")
	}
	if gs.wordProgress["かさ"] != (store.Progress{Level: 2}) || len(gs.wordSessionStats) != 0 || len(gs.wordMemory) != 0 ||
		len(gs.meaningProgress) != 0 || len(gs.meaningSessionStats) != 0 || len(gs.meaningMemory) != 0 {
		t.Fatalf("no word progress: reading %+v, stats %v, memory %v", gs.wordProgress["かさ"], gs.wordSessionStats, gs.wordMemory)
	}
	for _, c := range []string{"か", "さ"} {
		if gs.sessionStats[c].CorrectCount != 1 || gs.currentStreak[c] != 1 || gs.kanaProgress[c].Level != learnedLevel || !gs.memoryDirty[c] {
			t.Fatalf("%s: stats %+v, streak %d, progress %+v", c, gs.sessionStats[c], gs.currentStreak[c], gs.kanaProgress[c])
		}
	}
	if gs.answerStreak != 1 {
		t.Fatalf("answer streak %d: a word read raises the tempo once", gs.answerStreak)
	}
	if gs.score != 20*gs.pointsFactor() || gs.readKana != 2 {
		t.Fatalf("score %d, read kana %d", gs.score, gs.readKana)
	}
	if got := gs.feedback[0].text.Text; got != "かさ — umbrella" {
		t.Fatalf("feedback %q", got)
	}
}

func TestReadingHitCreditsTheUnitsOfTheTypedForm(t *testing.T) {
	okasa := vocab.Word{Kana: "おかさ", Display: "（お）かさ", Alts: []string{"かさ"}, Phrase: true, Source: "course", Meanings: vocab.Meanings{"en": {"umbrella"}}}
	koko := vocab.Word{Kana: "ここ", Source: "course", Meanings: vocab.Meanings{"en": {"here"}}}
	kitte := vocab.Word{Kana: "きって", Source: "course", Meanings: vocab.Meanings{"en": {"stamp"}}}
	gs := newTestState()
	gs.readingPractice = true
	withList(gs, "course", okasa, koko, kitte)
	learnKana(gs, "お", "か", "さ", "こ", "き", "て")
	read := func(kana, input string) {
		gs.tiles = []*KanaTile{practiceTileAt(gs, kana, 100)}
		gs.checkAnswer(input)
		if len(gs.tiles) != 0 {
			t.Fatalf("%q does not answer %s", input, kana)
		}
	}
	count := func(c string) int { return gs.sessionStats[c].CorrectCount }

	read("おかさ", "kasa")
	if count("お") != 0 || count("か") != 1 || count("さ") != 1 {
		t.Fatalf("short form: お %d, か %d, さ %d; only its own kana count", count("お"), count("か"), count("さ"))
	}
	read("おかさ", "okasa")
	if count("お") != 1 || count("か") != 2 || count("さ") != 2 {
		t.Fatalf("full form: お %d, か %d, さ %d", count("お"), count("か"), count("さ"))
	}
	read("ここ", "koko")
	if count("こ") != 2 || gs.currentStreak["こ"] != 2 {
		t.Fatalf("ここ: こ %d, streak %d; a doubled kana counts twice", count("こ"), gs.currentStreak["こ"])
	}
	read("きって", "kitte")
	if count("き") != 1 || count("て") != 1 {
		t.Fatalf("きって: き %d, て %d", count("き"), count("て"))
	}
	if _, ok := gs.sessionStats["っ"]; ok {
		t.Fatal("っ is no kana of its own")
	}
	if gs.readKana != 9 || gs.answerStreak != 4 {
		t.Fatalf("read kana %d (want 2+3+2+2), answer streak %d (want one per word)", gs.readKana, gs.answerStreak)
	}
}

func TestReadingPracticeTakesOnlyTheReading(t *testing.T) {
	gs := newTestState()
	gs.readingPractice = true
	gs.wordsAsk = meaningKind
	withWords(gs, kasa)
	learnKana(gs, "か", "さ")
	tile := practiceTileAt(gs, "かさ", 100)
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("umbrella")
	gs.checkAnswer("kaso")
	if len(gs.tiles) != 1 || gs.tiles[0] != tile {
		t.Fatal("a meaning or a wrong reading answers no practice tile")
	}
	if len(gs.sessionStats) != 0 || len(gs.wordSessionStats) != 0 || len(gs.meaningSessionStats) != 0 ||
		len(gs.wordProgress) != 0 || len(gs.meaningProgress) != 0 || gs.kanaProgress["か"] != (store.Progress{Level: learnedLevel}) ||
		gs.score != 0 || len(gs.feedback) != 0 {
		t.Fatal("neither is a hit, a mistake or a hint")
	}
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 0 || gs.sessionStats["か"].CorrectCount != 1 {
		t.Fatal("the reading answers it, whatever Words ask for says")
	}
}

func TestReadingHitDrivesTheKanaPath(t *testing.T) {
	gs := newUnlockReadyState()
	gs.readingPractice = true
	withWords(gs, bw("あい", 5, "love"))
	gs.tiles = []*KanaTile{practiceTileAt(gs, "あい", 0)}
	gs.checkAnswer("ai")
	if !equalIDs(gs.pendingIntro, []string{"k", "s"}) || !gs.paused {
		t.Fatalf("pendingIntro %v, paused %v: a word read counts towards auto-progression", gs.pendingIntro, gs.paused)
	}

	offer := newOfferReadyState()
	offer.readingPractice = true
	withWords(offer, bw("あい", 5, "love"))
	offer.tiles = []*KanaTile{practiceTileAt(offer, "あい", 0)}
	offer.checkAnswer("ai")
	if !offer.pendingOffer {
		t.Fatal("a word read counts towards the katakana offer")
	}
}

func TestATileKeepsTheRulesItWasSpawnedUnder(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("かさ", 5, "umbrella"))
	learnKana(gs, "か", "さ")
	gs.levelOfferDeclined = true // かさ is the whole source: no level offer may pause the game
	gs.readingPractice = true
	gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 100)} // spawned before reading practice was on
	gs.checkAnswer("kasa")
	if gs.wordProgress["かさ"].Level != 1 || len(gs.sessionStats) != 0 {
		t.Fatal("a word tile from before counts for the word")
	}
	gs.readingPractice = false
	gs.tiles = []*KanaTile{practiceTileAt(gs, "かさ", 100)}
	gs.checkAnswer("kasa")
	if gs.wordProgress["かさ"].Level != 1 || gs.sessionStats["か"].CorrectCount != 1 {
		t.Fatal("a practice tile keeps counting for its kana")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestReadingHit|TestReadingPracticeTakes|TestATileKeeps'` → build failure: `gs.readKana undefined`.

- [ ] **Step 3: Implement.**

In `fyne/game.go`, in `GameState` after `tempoStep`, add:

```go
	readKana      int       // kana units of words read in reading practice this round
```

Replace `recordCorrect` with:

```go
// recordCorrect records a hit on a character: session stats, memory, level
// and tempo, then auto-progression. Reports whether it announced newly
// unlocked rows. Must be called under lock.
func (gs *GameState) recordCorrect(char string) bool {
	gs.creditKana(char)
	gs.raiseTempo()
	return gs.progressAfterHit()
}

// creditKana counts a hit on a character: streak, session stats, memory
// and level. Must be called under lock.
func (gs *GameState) creditKana(char string) {
	streak := gs.currentStreak[char] + 1
	gs.currentStreak[char] = streak

	stat := gs.sessionStats[char]
	stat.Char = char
	stat.CorrectCount++
	stat.Streak = streak
	gs.sessionStats[char] = stat
	gs.sessionDirty = true

	prev, known := gs.memory[char]
	gs.memory[char] = memory.Correct(prev, known, gs.now())
	gs.memoryDirty[char] = true

	if gs.raiseKana(char) {
		gs.invalidatePlayable() // the kana just became learned
	}
}

// progressAfterHit runs auto-progression after a hit and reports whether it
// announced newly unlocked rows. Must be called under lock.
func (gs *GameState) progressAfterHit() bool {
	if unlocked := gs.checkAutoProgression(); len(unlocked) > 0 {
		gs.announceRows(unlocked)
		return true
	}
	return false
}
```

In `answers`, make the word branch:

```go
	if tile.word != nil {
		if tile.practice {
			return tile.word.ReadingMatches(input)
		}
		if tile.word.Meanings.MeaningMatches(gs.meaningLang, input) {
			return true
		}
		return tile.ask == readingKind && tile.word.ReadingMatches(input)
	}
```

and add to its doc comment: "A tile spawned in reading practice takes its readings only."

In `checkAnswer`, replace the word branch

```go
		if tile.word != nil {
			gs.score += 10 * len(tile.word.Units) * gs.pointsFactor()
			kind := answeredKind(tile, input)
			gs.recordWordCorrect(tile.word.Kana, kind)
			gs.addFeedback(feedbackHit, hitFeedbackText(*tile.word, gs.meaningLang, kind == meaningKind), tile.pos)
		} else {
```

with

```go
		if tile.word != nil {
			gs.score += 10 * len(tile.word.Units) * gs.pointsFactor()
			if tile.practice {
				announced = gs.recordReading(*tile.word, input)
				gs.addFeedback(feedbackHit, hitFeedbackText(*tile.word, gs.meaningLang, false), tile.pos)
			} else {
				kind := answeredKind(tile, input)
				gs.recordWordCorrect(tile.word.Kana, kind)
				gs.addFeedback(feedbackHit, hitFeedbackText(*tile.word, gs.meaningLang, kind == meaningKind), tile.pos)
			}
		} else {
```

and add to its doc comment after "a word hit counts for the reading or the meaning (see answeredKind)": ", a word read in reading practice for its kana (see recordReading)".

In `wrongInput`, replace

```go
	tile := gs.tiles[0]
	if tile.word != nil {
		gs.applyWordMistake(tile.word.Kana, tile.ask)
	} else {
		gs.applyKanaMistake(tile.kana.Char)
	}
```

with

```go
	tile := gs.tiles[0]
	switch {
	case tile.practice:
		// Reading practice changes no word, and only a reading counts
		// for kana: a wrong input costs nothing.
	case tile.word != nil:
		gs.applyWordMistake(tile.word.Kana, tile.ask)
	default:
		gs.applyKanaMistake(tile.kana.Char)
	}
```

and add to its doc comment: "On a tile spawned in reading practice nothing counts."

In `fyne/reading.go`, change the imports to

```go
import (
	"kana/kanacore"
	"kana/vocab"
)
```

and append:

```go
// recordReading counts a word read in reading practice: every kana unit
// of the reading input is a romaji reading of (see
// vocab.Entry.MatchedReading) gets a hit as on a kana tile (see
// creditKana) — a short form only its own kana, a kana that occurs twice
// two hits, っ, ッ and ー none — then the tempo rises once and
// auto-progression runs. The word's own progress is not touched. Reports
// whether rows were announced. Must be called under lock.
func (gs *GameState) recordReading(e vocab.Entry, input string) bool {
	reading, ok := e.MatchedReading(input)
	if !ok {
		return false
	}
	units, _ := kanacore.KanaUnits(reading)
	for _, u := range units {
		if _, isKana := kanacore.CharToRow[u]; isKana {
			gs.creditKana(u)
			gs.readKana++
		}
	}
	gs.raiseTempo()
	return gs.progressAfterHit()
}
```

In `Reset`, after the lines `gs.answerStreak = 0` / `gs.tempoStep = 0` (one tab deep; `lowerTempo` has another `gs.tempoStep = 0`), add `gs.readKana = 0`.

- [ ] **Step 4: Run** `gofmt -w fyne/game.go && go test ./fyne/ -run 'TestReading|TestATileKeeps|TestCheckAnswer|TestUnlock|TestOffer|TestTempo|TestMeaning|TestWrongInput' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): count a word read for its kana

In reading practice only a reading answers a tile; each kana of the form
typed counts as a kana hit, the tempo rises once and the word itself
gains nothing. A wrong input costs nothing.
EOF
git add fyne/game.go fyne/reading.go fyne/reading_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 8: Misses and dropped tiles in reading practice

**Files:**
- Modify: `fyne/game.go` (`tick`, `dropInactiveTiles`), `fyne/reading.go` (`readingSet`)
- Test: `fyne/reading_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `fyne/reading_test.go`:

```go
func TestReadingPracticeMissChangesNoKanaOrWord(t *testing.T) {
	gs := newTestState()
	gs.readingPractice = true
	withWords(gs, bw("かさ", 5, "umbrella"))
	learnKana(gs, "か", "さ")
	setWordLevel(gs, "かさ", 3)
	gs.tempoStep, gs.answerStreak = 3, 2
	gs.tiles = []*KanaTile{practiceTileAt(gs, "かさ", 599)}
	gs.tick()
	if len(gs.tiles) != 0 || gs.missed != 1 || len(gs.missedWords) != 1 {
		t.Fatalf("tiles %d, missed %d, missed words %d", len(gs.tiles), gs.missed, len(gs.missedWords))
	}
	if gs.tempoStep != 3-missTempoSteps || gs.answerStreak != 0 {
		t.Fatalf("tempo %d, streak %d: a miss lowers the tempo", gs.tempoStep, gs.answerStreak)
	}
	if gs.wordProgress["かさ"] != (store.Progress{Level: 3}) || len(gs.wordSessionStats) != 0 || len(gs.sessionStats) != 0 ||
		gs.kanaProgress["か"] != (store.Progress{Level: learnedLevel}) || gs.kanaProgress["さ"] != (store.Progress{Level: learnedLevel}) {
		t.Fatal("a miss changes no kana and no word")
	}
	if got := gs.feedback[0].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("miss feedback %q", got)
	}
	gs.missed = gs.level.MissLimit - 1
	gs.tiles = []*KanaTile{practiceTileAt(gs, "かさ", 599)}
	gs.tick()
	if !gs.over {
		t.Fatal("a missed word counts towards the miss limit")
	}
}

func TestPracticeTilesStayWhileReadable(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("かさ", 5, "umbrella"))
	gs.introduced = map[string]bool{} // not playable
	learnKana(gs, "か", "さ")
	gs.tiles = []*KanaTile{practiceTileAt(gs, "かさ", 100)}
	gs.dropInactiveTiles()
	if len(gs.tiles) != 1 {
		t.Fatal("a practice tile stays while its word is readable, introduced or not")
	}
	gs.wordSource = listSourcePrefix + "gone"
	gs.invalidatePlayable()
	gs.dropInactiveTiles()
	if len(gs.tiles) != 0 {
		t.Fatal("a practice tile whose word is no longer readable goes")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestReadingPracticeMiss|TestPracticeTilesStay'` → FAIL (the miss lowers the word level; the unintroduced practice tile is dropped).

- [ ] **Step 3: Implement.**

In `fyne/game.go`, in `tick`, replace

```go
			if tile.word != nil {
				gs.recordWordMiss(tile.word.Kana, tile.ask)
```

with

```go
			if tile.word != nil {
				if tile.practice {
					gs.lowerTempo() // reading practice: no kana and no word changes
				} else {
					gs.recordWordMiss(tile.word.Kana, tile.ask)
				}
```

Replace `dropInactiveTiles` with:

```go
// dropInactiveTiles removes kana tiles whose row is deselected or whose
// script is inactive, word tiles whose word is no longer playable, and
// tiles spawned in reading practice whose word is no longer readable. Must
// be called under lock.
func (gs *GameState) dropInactiveTiles() {
	var playable, readable map[string]bool // built on the first word tile that needs them
	kept := gs.tiles[:0]
	for _, t := range gs.tiles {
		if t.word != nil {
			var ok bool
			if t.practice {
				if readable == nil {
					readable = readingSet(gs.readableWords())
				}
				ok = readable[t.word.Kana]
			} else {
				if playable == nil {
					playable = readingSet(gs.playableWords())
				}
				ok = playable[t.word.Kana]
			}
			if ok {
				kept = append(kept, t)
			}
			continue
		}
		rowID, ok := kanacore.CharToRow[t.kana.Char]
		if ok && (!gs.selectedRows[rowID] || !gs.activeScripts[kanacore.ScriptOf(rowID)]) {
			continue
		}
		kept = append(kept, t)
	}
	gs.tiles = kept
}
```

In `fyne/reading.go`, append:

```go
// readingSet returns the readings of entries as a set.
func readingSet(entries []vocab.Entry) map[string]bool {
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		set[e.Kana] = true
	}
	return set
}
```

- [ ] **Step 4: Run** `go test ./fyne/ -run 'TestReadingPractice|TestPracticeTiles|TestDrop|TestMiss|TestTick' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): handle missed and dropped tiles in reading practice

A missed word counts towards the miss limit and lowers the tempo but
changes no kana or word; a falling practice tile stays while readable.
EOF
git add fyne/game.go fyne/reading.go fyne/reading_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 9: kana/min and the status line

**Files:**
- Modify: `fyne/game.go` (`playTicks` field, `tick`, `Reset`), `fyne/reading.go` (`kanaPerMinute`), `fyne/words.go` (`WordsStatus`, `noWordsToReadText`, `formatWordsLine`), `fyne/wordgame.go` (`wordsStatus`)
- Test: `fyne/words_test.go`, `fyne/reading_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `fyne/words_test.go`:

```go
func TestFormatWordsLineReadingPractice(t *testing.T) {
	cases := []struct {
		name string
		ws   WordsStatus
		want string
	}{
		{"no word read yet", WordsStatus{Reading: true, Readable: 12, Source: "JLPT N5", Level: 5}, "Reading practice: 12 words to read · – kana/min"},
		{"speed", WordsStatus{Reading: true, Readable: 1, ReadKana: 40, KanaPerMin: 35, Source: "JLPT N5", Level: 5}, "Reading practice: 1 word to read · 35 kana/min"},
		{"nothing readable", WordsStatus{Reading: true, Source: "JLPT N5", Level: 5}, noWordsToReadText},
		{"nothing readable, words only", WordsStatus{Reading: true, WordsOnly: true, KanaFallback: true, Source: "minna", Next: wordsNeedKana}, noWordsToReadText},
		{"list missing", WordsStatus{Reading: true, WordsOnly: true, KanaFallback: true, Missing: true, Dir: "~/words", Source: "minna"}, "Words only: add a list to ~/words or choose JLPT"},
		{"skipped rows", WordsStatus{Reading: true, Readable: 3, Skipped: 2, Source: "minna"}, "Reading practice: 3 words to read · – kana/min · 2 rows skipped"},
	}
	for _, c := range cases {
		if got := formatWordsLine(c.ws); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if noWordsToReadText != "No words to read yet — learn more kana or pick another list" {
		t.Fatalf("reason %q", noWordsToReadText)
	}
}
```

Append to `fyne/reading_test.go`:

```go
func TestReadingPracticeStatusLine(t *testing.T) {
	gs := newTestState()
	gs.readingPractice = true
	withWords(gs, bw("かさ", 5, "umbrella"), bw("さか", 5, "slope"), bw("かき", 5, "persimmon"))
	gs.introduced = map[string]bool{}
	gs.invalidatePlayable()
	if got := formatWordsLine(gs.wordsStatus()); got != noWordsToReadText {
		t.Fatalf("no kana learned: %q", got)
	}
	learnKana(gs, "か", "さ")
	if got := formatWordsLine(gs.wordsStatus()); got != "Reading practice: 2 words to read · – kana/min" {
		t.Fatalf("before reading: %q", got)
	}
	for i := 0; i < 60*ticksPerSecond; i++ { // one minute of play
		gs.tick()
	}
	gs.paused = true
	for i := 0; i < 60*ticksPerSecond; i++ { // a pause does not count
		gs.tick()
	}
	gs.paused = false
	gs.tiles = []*KanaTile{practiceTileAt(gs, "かさ", 100)}
	gs.checkAnswer("kasa")
	gs.tiles = []*KanaTile{practiceTileAt(gs, "さか", 100)}
	gs.checkAnswer("saka")
	if got := formatWordsLine(gs.wordsStatus()); got != "Reading practice: 2 words to read · 4 kana/min" {
		t.Fatalf("4 kana in one minute of play: %q", got)
	}
	gs.Reset()
	if gs.playTicks != 0 || gs.readKana != 0 {
		t.Fatalf("a new round starts at 0: ticks %d, kana %d", gs.playTicks, gs.readKana)
	}
	if got := formatWordsLine(gs.wordsStatus()); got != "Reading practice: 2 words to read · – kana/min" {
		t.Fatalf("after Reset: %q", got)
	}
}

func TestKanaPerMinute(t *testing.T) {
	gs := newTestState()
	if got := gs.kanaPerMinute(); got != 0 {
		t.Fatalf("no play: %d", got)
	}
	gs.playTicks, gs.readKana = 30*ticksPerSecond, 10
	if got := gs.kanaPerMinute(); got != 20 {
		t.Fatalf("10 kana in 30 s: %d, want 20", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestFormatWordsLineReadingPractice|TestReadingPracticeStatusLine|TestKanaPerMinute'` → build failure: `unknown field Reading in struct literal of type WordsStatus`.

- [ ] **Step 3: Implement.**

In `fyne/game.go`, in `GameState` after `readKana`, add:

```go
	playTicks     int       // ticks of play this round; tick does not run while paused or over
```

In `tick`, right after the early return for `gs.over || gs.paused`, add:

```go
	gs.playTicks++
```

In `Reset`, change `gs.readKana = 0` to `gs.readKana, gs.playTicks = 0, 0`.

In `fyne/reading.go`, change the imports to

```go
import (
	"math"

	"kana/kanacore"
	"kana/vocab"
)
```

and append:

```go
// kanaPerMinute is the kana units of words read per minute of play this
// round (see readKana and playTicks), rounded; 0 before any play. Must be
// called under lock.
func (gs *GameState) kanaPerMinute() int {
	if gs.playTicks == 0 {
		return 0
	}
	minutes := float64(gs.playTicks) / (ticksPerSecond * 60)
	return int(math.Round(float64(gs.readKana) / minutes))
}
```

In `fyne/words.go`, add at the end of `WordsStatus`:

```go

	Reading    bool // reading practice is on
	Readable   int  // words that may fall in reading practice
	ReadKana   int  // kana units of words read this round
	KanaPerMin int  // ReadKana per minute of play this round; shown once ReadKana > 0
```

After `maxWordsLineRunes`, add:

```go
// noWordsToReadText is the words line in reading practice when no word of
// the source can be read yet.
const noWordsToReadText = "No words to read yet — learn more kana or pick another list"
```

In `formatWordsLine`, insert at the top of the `switch` (before `case ws.KanaFallback:`):

```go
	case ws.Reading && ws.Readable > 0:
		rate := "–"
		if ws.ReadKana > 0 {
			rate = fmt.Sprint(ws.KanaPerMin)
		}
		s = fmt.Sprintf("Reading practice: %s to read · %s kana/min", plural(ws.Readable, "word"), rate)
	case ws.Reading && ws.Err == "" && !ws.Missing:
		s = noWordsToReadText
```

and add to its doc comment: "In reading practice it shows the words to read and the kana per minute, or why there is none (a broken bundled list or a missing list keep their own message)."

In `fyne/wordgame.go`, in `wordsStatus`, insert before `if gs.words == nil {`:

```go
	if gs.readingPractice {
		ws.Reading = true
		ws.Readable = len(gs.readableWords())
		ws.ReadKana, ws.KanaPerMin = gs.readKana, gs.kanaPerMinute()
	}
```

- [ ] **Step 4: Run** `gofmt -w fyne/game.go fyne/words.go && go test ./fyne/ -run 'TestFormatWordsLine|TestReadingPractice|TestKanaPerMinute|TestWordsStatus|TestStats' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): show words to read and kana per minute

In reading practice the words line shows how many words can fall and the
round's reading speed, counted over play time only, or why none can.
EOF
git add fyne/game.go fyne/reading.go fyne/words.go fyne/wordgame.go fyne/words_test.go fyne/reading_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 10: "Reading practice" in the settings dialog

**Files:**
- Modify: `fyne/settings.go`
- Test: `fyne/settings_test.go`

- [ ] **Step 1: Write the failing tests.** Append to `fyne/settings_test.go`:

```go
func TestReadingPracticeInTheWordsTab(t *testing.T) {
	test.NewApp()
	gs := settingsGame()
	gs.wordsAsk = meaningKind
	f := newSettingsForm(gs.settingsState(), nil)
	if f.readingCheck.Text != "Reading practice" || f.readingCheck.Checked || f.wordsAsk.Disabled() {
		t.Fatalf("check %q checked %v, words ask disabled %v", f.readingCheck.Text, f.readingCheck.Checked, f.wordsAsk.Disabled())
	}
	f.readingCheck.SetChecked(true)
	if !f.wordsAsk.Disabled() || f.wordsAsk.Selected != "Meaning" {
		t.Fatal("Words ask for is greyed out and keeps its choice")
	}
	f.apply(gs)
	if !gs.readingPractice || gs.wordsAsk != meaningKind {
		t.Fatalf("applied: reading practice %v, words ask %v", gs.readingPractice, gs.wordsAsk)
	}
	f = newSettingsForm(gs.settingsState(), nil)
	if !f.readingCheck.Checked || !f.wordsAsk.Disabled() {
		t.Fatal("reopened: checked and greyed out")
	}
	f.readingCheck.SetChecked(false)
	if f.wordsAsk.Disabled() {
		t.Fatal("unchecked: Words ask for is enabled again")
	}
	f.apply(gs)
	if gs.readingPractice || gs.wordsAsk != meaningKind {
		t.Fatalf("off: reading practice %v, words ask %v", gs.readingPractice, gs.wordsAsk)
	}
}

func TestTurningReadingPracticeOffOffersTheDuePackage(t *testing.T) {
	test.NewApp()
	gs := packageReady(10)
	gs.readingPractice = true
	f := newSettingsForm(gs.settingsState(), nil)
	f.apply(gs)
	if len(gs.pendingPackage) != 0 {
		t.Fatal("no package while reading practice stays on")
	}
	f.readingCheck.SetChecked(false)
	f.apply(gs)
	if len(gs.pendingPackage) == 0 {
		t.Fatal("the due package comes once reading practice is off")
	}
}
```

- [ ] **Step 2: Run** `go test ./fyne/ -run 'TestReadingPracticeInTheWordsTab|TestTurningReadingPracticeOff'` → build failure: `f.readingCheck undefined`.

- [ ] **Step 3: Implement.** In `fyne/settings.go`:

Add to `settingsState` after `focusWeak`:

```go
	readingPractice bool
```

and in `settingsState()`'s return literal after `focusWeak: gs.focusWeak,`:

```go
		readingPractice: gs.readingPractice,
```

Add to `settingsForm` after `wordsAsk`:

```go
	readingCheck  *widget.Check       // reading practice; "Words ask for" is disabled while it is checked
```

In `newWordsTab`, right after the `f.wordsAsk` block (after its `SetSelected(askMeaningLabel)` `if`), add:

```go
	f.readingCheck = widget.NewCheck("Reading practice", func(bool) { f.syncWordsAsk() })
	f.readingCheck.SetChecked(st.readingPractice)
	f.syncWordsAsk()
```

and change the top form to:

```go
		container.New(layout.NewFormLayout(),
			widget.NewLabelWithStyle("Tiles", fyne.TextAlignLeading, bold), f.tiles,
			widget.NewLabelWithStyle("Words ask for", fyne.TextAlignLeading, bold), f.wordsAsk,
			widget.NewLabel(""), f.readingCheck,
		),
```

In `newWordsTab`'s doc comment, change "what word tiles ask for (reading or meaning)," to "what word tiles ask for (reading or meaning), reading practice,".

After `askedKind`, add:

```go
// syncWordsAsk disables "Words ask for" while reading practice is checked:
// a word there counts by its reading only. Its choice stays and is saved
// as it is.
func (f *settingsForm) syncWordsAsk() {
	if f.readingCheck.Checked {
		f.wordsAsk.Disable()
	} else {
		f.wordsAsk.Enable()
	}
}
```

In `words()`, add `readingPractice: f.readingCheck.Checked` to the returned literal:

```go
	return wordSettings{source: f.wordSource(), level: level, packageSize: size,
		wordsOnly: f.tiles.Selected == tilesWordsOnly, lessons: f.selectedLessons(), meaningLang: f.meaningLang(),
		ask: f.askedKind(), readingPractice: f.readingCheck.Checked}
```

- [ ] **Step 4: Run** `gofmt -w fyne/settings.go && go test ./fyne/ -run 'TestReadingPractice|TestTurningReadingPracticeOff|TestWordsAsk|TestFitTabs|TestSettings' && go build ./... && go vet ./...` → PASS.

- [ ] **Step 5: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
feat(fyne): turn on reading practice in the settings

The Words tab gets a "Reading practice" check under "Words ask for",
which is greyed out while it is on and keeps its choice.
EOF
git add fyne/settings.go fyne/settings_test.go && git commit -F "$msg" && rm "$msg"
```

---

### Task 11: README — reading practice for players

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Core Gameplay.** In the "**Words**" bullet, change "Only introduced words fall." to "Only introduced words fall (except in reading practice, below)." After the "**Meaning mode**" bullet (ending "…still fall until their meaning is learned."), insert:

```markdown
- **Reading practice**: Settings → Words → "Reading practice" trains reading speed. Any word of
  your source (the JLPT words up to the chosen level, or the chosen list with its lessons)
  whose kana you have all learned may fall, whether or not a package introduced it, picked at
  random. Type its reading; a meaning does not count, and "Words ask for" is greyed out
  meanwhile. A word read counts for its kana, not for the word: each kana of the form you typed
  counts as a hit on that kana, for the statistics, the learning path and the katakana offer
  (the short form of an optional part counts only its own kana, ここ counts こ twice, っ and ー
  count for nothing), and the tempo rises once per word. Words gain no level, and a missed word
  costs a miss but changes no kana. No word packages and no JLPT level offer come while it is
  on. The status line shows how many words you can read and your reading speed this round in
  kana per minute, pauses excluded ("Reading practice: 214 words to read · 38 kana/min"). With
  "Words only" it is a pure reading drill; while no word can be read, kana fall and the status
  line says so. A change applies to new tiles; tiles already falling keep their rules.
```

- [ ] **Step 2: Customization.** In the "**Word packages**" bullet, change `what word tiles ask for ("Words ask for": Reading or Meaning) and the meaning` to `what word tiles ask for ("Words ask for": Reading or Meaning), reading practice and the meaning`.

- [ ] **Step 3: Architecture.** In the `fyne/` list, after the `wordgame.go` line, add:

```markdown
- `reading.go`: reading practice — the readable words, the uniform pick, the kana credit for a word read and the kana per minute
```

In the `settings.go` line, change `Words (tiles, words ask for, source, …` to `Words (tiles, words ask for, reading practice, source, …`. In the `store/store.go` line, add `` `reading_practice`, `` after `` `words_ask`, ``.

- [ ] **Step 4: Data Persistence.** After the bullet "What word tiles ask for (`words_ask`: …)", add:

```markdown
- Whether reading practice is on (`reading_practice`, default off); the kana per minute is not saved
```

- [ ] **Step 5: Check** `grep -n 'Reading practice\|reading_practice\|reading.go' README.md` shows the new lines, and the README has no issue identifiers.

- [ ] **Step 6: Final verification.** Run `gofmt -l .` (empty), `go vet ./...` (clean), `go build -o /tmp/kana-desktop-check ./fyne/ && rm /tmp/kana-desktop-check`, and `go test ./... -timeout 900s` → PASS.

- [ ] **Step 7: Commit.**

```bash
msg=$(mktemp) && cat > "$msg" <<'EOF'
docs: describe reading practice for players

The README explains which words fall, how a word read counts for its
kana, the status line with kana per minute and the stored setting.
EOF
git add README.md && git commit -F "$msg" && rm "$msg"
```
