# Vocabulary Trainer Stage 4a: Word Packages, Settings Tabs, Kana Font Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Words enter the game the way kana rows do: in small packages, each shown first in an intro dialog (kana, romaji, meaning, "Already known"). Only introduced words fall, which brings the bundled JLPT words back. New words come from one private list or from JLPT (N5 first, common words first); a fully introduced JLPT level leads to an offer of the next one. The settings dialog gets the tabs Kana, Words, Game and Sources (the data attribution moves there), and the learner can pick the kana font (Noto Sans JP, M PLUS Rounded 1c, System), both fonts shipped with the app.

**Architecture:** `cmd/vocabgen` reads JMdict's priority tags and writes a frequency rank `"r"` per bundled word. `vocab` carries `Rank` and each list entry's first row (`ListPos`), and `NextPackage` walks a per-source learning order that is computed once per index and cached. `store` gains the `introduced_words` table (seeded once from existing word progress) and the settings `package_size`, `word_source`, `kana_font`, `newest_package`; `bundled_words` goes. In `fyne`, the playable set becomes introduced ∩ known; `maybeOfferPackage` decides at fixed re-check points whether the next package (80 % rule) or the level offer is due and pauses the game like a kana intro; `FinishPackage` / `ReplacementWord` / `AcceptLevelOffer` / `DeclineLevelOffer` are driven by two new dialogs through the existing `showPendingDialogs`, which now reads a `Pending` struct. The settings dialog is rebuilt as a `settingsForm` with four tabs and its own Save/Cancel buttons. Fonts are embedded `fyne.Resource`s; a `fontChoice` sets `canvas.Text.FontSource` only when the font has every rune of the text (checked with `go-text/typesetting`).

**Tech Stack:** Go 1.25, Fyne v2.7.3 (`canvas.Text.FontSource`, `container.AppTabs`, `dialog.NewCustomWithoutButtons`), `github.com/go-text/typesetting` v0.3.3 (`font.ParseTTF`, `Font.NominalGlyph`; becomes a direct dependency), modernc SQLite, standard `testing`.

**Spec:** `docs/superpowers/specs/2026-10-15-vocab-stage4a-word-packages-design.md` (the section "Review resolutions" is authoritative), building on `docs/superpowers/specs/2026-10-12-vocab-trainer-design.md`.

**Conventions:** `gofmt -l .` prints nothing, `go vet ./...` is clean and `go test -count=1 ./...` passes before each commit. Commit messages: `type(scope): summary` plus a short prose paragraph; no Co-Authored-By lines, no mention of AI, assistants or sessions. Commit only the files the task lists (`git add` them by name, never `git add -A`). Never commit anything from `~/.local/share/kana` (the database or private word lists). Run every command from the repository root. Do not run `go build ./cmd/vocabgen` without `-o`: it leaves a `vocabgen` binary in the root.

**Verification record:** this plan was applied task by task in a throwaway worktree of `feat/word-packages` (at `ac2c6de`). After every task `gofmt -l .` was empty, `go vet ./...` clean and `go test -count=1 ./...` passed, also for each task's tree exported on its own; at the end `go test -race -count=1 ./fyne/` passed and `go build -o kana-desktop ./fyne/` linked. The code blocks below are taken from those trees. The "Expected" failures in each task are the real compiler output of that task's tests against the previous task's code.

---

## Notes on the spec

Places where the spec leaves room or does not match the code, and how this plan resolves them:

1. **`bundled_words` removal moves to Task 4.** The spec lists it with the store changes, but `fyne` still calls `Store.BundledWords` until Task 4 replaces the stage 3 playable rule. Task 3 only adds; Task 4 removes `BundledWords`/`SaveBundledWords`, `useBundled`, `WordsStatus.Bundled` and `Filter.NoBundled` together, so every commit builds.
2. **"First lesson" of an entry in several lessons** is the lesson that comes first in the list's lesson order (the lowest number for numeric lessons, else the earliest first appearance in the file), not the lesson of the entry's first row. `vocab.(*Index).Lesson` exposes it for the intro title. The list's lesson order is recorded per list while the index is built (`Index.lessons`).
3. **Re-check points after a hit.** "After a word hit", "after a kana hit that makes a kana secure" and "at the end of `checkAnswer`" are one call: `maybeOfferPackage()` at the end of `checkAnswer`, after the score/end check and `maybeOfferKatakana`, which runs after every hit.
4. **Dialog order.** `showPendingDialogs` now follows the spec's order (kana intro, katakana offer, word package, level offer); the old code showed the katakana offer before an intro. The two are never pending together (`EnableKatakana` queues the intro only after clearing the offer), so nothing changes for the learner.
5. **`PendingDialogs` struct** is `Pending{Intro, KatakanaOffer, Package, LevelOffer}`; `LevelOffer` is the finished JLPT level (0 = none), which the dialog text needs.
6. **Level offer needs the bundled list.** With a broken bundled list there are no bundled entries, so "every bundled entry of the level is introduced" would be vacuously true; the offer also requires `wordsErr == ""`.
7. **Start path.** `NewGameState` ends with `offerPackageAtStart()`, which sets a due package (or level offer) pending directly, like `startLearningPath` sets the first intro. `TestFreshStartCatchesUpMasteredSteps` is adjusted: a learner with secure kana now gets the first package at start, and only that may hold the pause.
8. **"Already known" save.** `markKnown` writes `word_stats` at once with `max(3, current)` and updates `wordOverallStats` (not the session stats). A failed `SaveWordStats` is not retried (the in-memory value still counts for this run); the memory state is saved by the next merge like any dirty word state. Readings in `known` that are not in the pending package are ignored. If every word is ticked, the newest package is empty and the next package is offered at once.
9. **Replacement word** is `NextPackage(…, 1)` over introduced ∪ pending readings; a missing list source gives none.
10. **Status line.** "secure" counts introduced readings (in the index) with ≥ 3 correct; stage 3 counted playable ones. The hint replaces the counts when `NextPackage(…, 1)` is empty, independent of the 80 % rule: "learn more kana" if the source still has words that are not introduced, else "<source> finished" ("JLPT finished" at N1). A missing list shows the add-a-list hint; a failed introduced load shows "Words unavailable: could not load progress".
11. **Settings dialog buttons.** `dialog.ShowCustomConfirm` always closes on Save, so the dialog is built with `dialog.NewCustomWithoutButtons` and its own Cancel/Save buttons: an invalid value keeps it open, switches to the tab with the value and shows the error.
12. **Words tab.** A missing list is kept as an option (its pretty name) so it stays selected; the JLPT level radio is disabled unless JLPT is selected. The summary line describes the stored source, not the selection being edited.
13. **Fonts.** The Noto files are the repository files of noto-cjk at tag `Sans2.004` (`Sans/SubsetOTF/JP/`), byte-identical to those in that release's `16_NotoSansJP.zip` (checked), so the script needs no unzip. google/fonts has no `OFL.txt` in `ofl/mplusrounded1c`; the script takes it from `ofl/roundedmplus1c` (the same family, same copyright line "The Rounded M+ Project Authors"). Noto's licence file is the noto-cjk `LICENSE` at that tag.
14. **Coverage facts (pinned files):** neither font has Thai ก (U+0E01) — used by the coverage tests; both have hiragana, katakana, ー, っ/ッ, ヴ, "…", "—", "·", Latin and Cyrillic (M PLUS Rounded 1c does have Cyrillic).
15. **Font plumbing.** Tile constructors take the font (`newKanaTile(k, kf)`, `newWordTile(e, maxW, kf)`); tests pass `fontChoice{}` (theme font), so the existing tile tests measure as before. Text that may be cut ("…") goes through `fitKanaText`, which checks the font again on the final text. Intro cards of the katakana offer also use the kana font. `fontFaces` parses each font once for the coverage check, separately from Fyne's own font cache.
16. **Test helper `withWords`** now introduces every word it adds (most stage 3 tests are about spawning and answering introduced words). `TestWordHitShowsFeedbackAtTile` sets `levelOfferDeclined`, because with the whole N5 test index introduced the first hit would queue the level offer and pause the game.
17. **Regenerated data.** Regenerating on 2026-10-15 gave 7,029 words (JMdict `Last-Modified: Thu, 15 Oct 2026 03:30:22 GMT`, mapping commit `b062d4e38c4bdd0950ae1d4ec55f04b176182e03`); a later run may differ slightly, so the tests only check ranges (every rank in 1–99, at least half with a priority tag, せんせい common).
18. **`Playable` and `Filter` stay** in `vocab` (stage 4b's words-only mode needs them), without `NoBundled`.

## Lock discipline

- Every new `GameState` method that reads or writes state is "Must be called under lock" (`gs.mu`), except the exported ones (`FinishPackage`, `ReplacementWord`, `AcceptLevelOffer`, `DeclineLevelOffer`, `PackageIntro`, `PendingDialogs`, `KanaFont`, `SetKanaFont`, `applyWordSettings`, `settingsState`), which take the lock themselves, and `offerPackageAtStart`, which runs inside `NewGameState` before the game starts (like `startLearningPath`).
- `maybeOfferPackage` only sends on the buffered `eventCh` with `select … default`, never blocks, and sets `pendingPackage`/`pendingLevelOffer` and `paused = true` only when the send succeeded. `settle()` stays the only place that unpauses.
- Dialogs run on the Fyne thread. The package dialog calls `gs.ReplacementWord` (locks) from a check's `OnChanged` and `gs.FinishPackage` from `SetOnClosed`; the level offer calls `AcceptLevelOffer`/`DeclineLevelOffer` from its response callback. They never hold `gs.mu` while touching widgets.
- No new goroutines. `watchEvents` handles the two new event kinds exactly like `rowsUnlockedEvent` (dropped when over, else `fyne.Do(showPendingDialogs)`); `dialogShowing` keeps dialogs from stacking.
- Tests call lock-requiring methods without the lock where no goroutine runs (as the existing tests do), and take the lock around a `NewGameState` with a store, as `TestMergePersistsMemory` does.

---

## File Structure

- Modify `cmd/vocabgen/jmdict.go` (priority tags, `rankOf`, `jmEntry`), `cmd/vocabgen/build.go` (`"r"`), `cmd/vocabgen/testdata/JMdict_sample.xml`, `cmd/vocabgen/vocabgen_test.go`.
- Regenerate `vocab/data/words.json.gz` and `vocab/data/SOURCES`.
- Modify `vocab/vocab.go` (`Word.Rank`), `vocab/bundled.go`, `vocab/index.go` (`Entry.Rank`, `Entry.ListPos`, lesson order, order cache; `Filter.NoBundled` removed), `vocab/index_test.go`, `vocab/bundled_test.go`.
- Create `vocab/package.go` (`Source`, `NextPackage`, `Lesson`), `vocab/package_test.go`.
- Modify `store/store.go` (`introduced_words`, new settings; `bundled_words` removed), `store/store_test.go`.
- Modify `fyne/game.go` (fields, load, re-check points, `Pending`, font), `fyne/wordgame.go` (playable rule, source, status, summary), `fyne/words.go` (sources, `WordsStatus`, line texts; attribution removed), `fyne/stats.go` (attribution removed, `SetKanaFont`), `fyne/intro.go` (dialog order, kana font), `fyne/app.go` (`watchEvents`, initial grid font), `fyne/settings.go` (rewritten: `settingsForm` with four tabs), `fyne/version.go` (Sources tab), `fyne/tile.go`, `fyne/feedback.go`.
- Create `fyne/packages.go` (package check, finishing, replacement, level offer, Words tab settings), `fyne/packageintro.go` (package intro and level offer dialogs), `fyne/fonts.go`, `fyne/fonts/*` (four font files, two licences), `scripts/fetch-fonts.sh`.
- Create `fyne/packages_test.go`, `fyne/packageintro_test.go`, `fyne/fonts_test.go`; modify `fyne/game_test.go`, `fyne/game_lifecycle_test.go`, `fyne/wordgame_test.go`, `fyne/words_test.go`, `fyne/stats_test.go`, `fyne/settings_test.go`, `fyne/tile_test.go`.
- Modify `go.mod` (`go-text/typesetting` direct), `README.md`, `PLAN.md`, `vocab/data/ATTRIBUTION.md`.

Code is located by its text (the "Replace" blocks are unique in their file at that point), not by line numbers.

---

### Task 1: Frequency rank in the word data

**Files:** Modify `cmd/vocabgen/jmdict.go`, `cmd/vocabgen/build.go`, `cmd/vocabgen/testdata/JMdict_sample.xml`, `cmd/vocabgen/vocabgen_test.go`, `vocab/vocab.go`, `vocab/bundled.go`, `vocab/index.go`, `vocab/index_test.go`, `vocab/bundled_test.go`; regenerate `vocab/data/words.json.gz`, `vocab/data/SOURCES`.

- [ ] **Step 1: Extend the fixture and write the failing tests**

In `cmd/vocabgen/testdata/JMdict_sample.xml`:

Replace:

```xml
<JMdict>
<entry>
<ent_seq>1</ent_seq>
<k_ele><keb>先生</keb></k_ele>
<r_ele><reb>せんせい</reb></r_ele>
<sense><pos>&n;</pos><gloss>teacher</gloss><gloss>instructor</gloss><gloss>master</gloss><gloss>doctor</gloss></sense>
<sense><gloss>second sense</gloss></sense>
<sense><gloss xml:lang="ger">Lehrer</gloss><gloss xml:lang="ger">Meister</gloss></sense>
```

with:

```xml
<JMdict>
<entry>
<ent_seq>1</ent_seq>
<k_ele><keb>先生</keb><ke_pri>news1</ke_pri><ke_pri>nf01</ke_pri></k_ele>
<r_ele><reb>せんせい</reb><re_pri>ichi1</re_pri><re_pri>news1</re_pri></r_ele>
<sense><pos>&n;</pos><gloss>teacher</gloss><gloss>instructor</gloss><gloss>master</gloss><gloss>doctor</gloss></sense>
<sense><gloss>second sense</gloss></sense>
<sense><gloss xml:lang="ger">Lehrer</gloss><gloss xml:lang="ger">Meister</gloss></sense>
```

Replace:

```xml
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
<sense><gloss>idea</gloss><gloss>(new) concept</gloss></sense>
</entry>
```

with:

```xml
</entry>
<entry>
<ent_seq>2</ent_seq>
<r_ele><reb>はし</reb><re_pri>nf05</re_pri></r_ele>
<sense><gloss>bridge</gloss></sense>
<sense><gloss xml:lang="ger">Brücke</gloss></sense>
</entry>
<entry>
<ent_seq>3</ent_seq>
<r_ele><reb>はし</reb><re_pri>ichi1</re_pri></r_ele>
<sense><gloss>chopsticks</gloss></sense>
</entry>
<entry>
<ent_seq>4</ent_seq>
<r_ele><reb>アイデア</reb><re_pri>gai2</re_pri></r_ele>
<r_ele><reb>アイディア</reb></r_ele>
<sense><gloss>idea</gloss><gloss>(new) concept</gloss></sense>
</entry>
```

In `cmd/vocabgen/vocabgen_test.go`, append the rank tests at the end of the file:

Replace:

```go
	sort.Strings(out)
	return out
}
```

with:

```go
	sort.Strings(out)
	return out
}

func TestRankOf(t *testing.T) {
	tests := []struct {
		tags []string
		want int
	}{
		{nil, 99},
		{[]string{"nf01"}, 1},
		{[]string{"nf48"}, 48},
		{[]string{"news1"}, 49},
		{[]string{"ichi1"}, 49},
		{[]string{"spec1"}, 49},
		{[]string{"gai1"}, 49},
		{[]string{"news2"}, 60},
		{[]string{"ichi2"}, 60},
		{[]string{"spec2"}, 60},
		{[]string{"gai2"}, 60},
		{[]string{"news2", "ichi1", "nf17"}, 17},
		{[]string{"news2", "spec1"}, 49},
		{[]string{"nf00", "nf49", "nfx", "other"}, 99},
	}
	for _, tt := range tests {
		if got := rankOf(tt.tags); got != tt.want {
			t.Errorf("rankOf(%v) = %d, want %d", tt.tags, got, tt.want)
		}
	}
}

func TestEntryRank(t *testing.T) {
	e := newJMEntry(xmlEntry{
		Kanji: []xmlKanji{{Pri: []string{"nf02"}}},
		Readings: []xmlReading{
			{Reb: "あ", Pri: []string{"news2"}},
			{Reb: "い"},
		},
	})
	if got := e.rank("あ"); got != 60 {
		t.Errorf("rank(あ) = %d, want 60 from its own tags, not the kanji's nf02", got)
	}
	if got := e.rank("い"); got != 2 {
		t.Errorf("rank(い) = %d, want 2, the best tag of the entry", got)
	}
	if got := e.rank("う"); got != 2 {
		t.Errorf("rank(う) = %d, want 2 for a reading the entry does not list", got)
	}
	if got := newJMEntry(xmlEntry{Readings: []xmlReading{{Reb: "あ"}}}).rank("あ"); got != 99 {
		t.Errorf("rank without any tag = %d, want 99", got)
	}
}

func TestGenerateRanks(t *testing.T) {
	words, _, _ := generate(t)
	want := map[string]int{
		"せんせい":  49, // its reading's ichi1/news1, not the kanji's nf01
		"はし":    5,  // homophones merged: nf05 beats ichi1
		"アイデア":  60, // gai2
		"アイディア": 60, // no tags of its own: the entry's best
		"ふく":    99, // entry without tags
		"ねこ":    99, // Waller fallback, no JMdict entry
	}
	for _, w := range words {
		if w.R != want[w.K] {
			t.Errorf("%s: rank %d, want %d", w.K, w.R, want[w.K])
		}
	}
}
```

In `vocab/index_test.go` (a `ListPos` leak check in `TestWithListsDoesNotShareMeanings`, then two new tests):

Replace:

```go
	if _, ok := ea.Lists["b"]; ok {
		t.Fatal("list b leaked into a's entry")
	}
}
```

with:

```go
	if _, ok := ea.Lists["b"]; ok {
		t.Fatal("list b leaked into a's entry")
	}
	if _, ok := ea.ListPos["b"]; ok {
		t.Fatal("list b's position leaked into a's entry")
	}
	if e, _ := base.Lookup("いぬ"); len(e.ListPos) != 0 {
		t.Fatalf("base list positions changed: %v", e.ListPos)
	}
}

func TestEntryListPosIsTheFirstRow(t *testing.T) {
	lists := []List{
		{Name: "minna", Words: []Word{
			word("せんせい", 0, "minna", "1", "de", "Lehrer"),
			word("ねこ", 0, "minna", "1", "de", "Katze"),
			word("せんせい", 0, "minna", "2", "de", "Meister"),
		}},
		{Name: "extra", Words: []Word{
			word("ねこ", 0, "extra", "", "en", "cat"),
		}},
	}
	bundled := []Word{word("ねこ", 5, BundledSource, "", "en", "cat")}
	for name, ix := range map[string]*Index{
		"NewIndex":  NewIndex(bundled, lists),
		"WithLists": NewIndex(bundled, nil).WithLists(lists),
	} {
		s, _ := ix.Lookup("せんせい")
		n, _ := ix.Lookup("ねこ")
		if !reflect.DeepEqual(s.ListPos, map[string]int{"minna": 0}) {
			t.Errorf("%s: せんせい ListPos = %v, want minna 0 (its first row)", name, s.ListPos)
		}
		if !reflect.DeepEqual(n.ListPos, map[string]int{"minna": 1, "extra": 0}) {
			t.Errorf("%s: ねこ ListPos = %v", name, n.ListPos)
		}
	}
}

func TestEntryRankFromBundledWordsOnly(t *testing.T) {
	a := word("はし", 5, BundledSource, "", "en", "bridge")
	a.Rank = 49
	b := word("はし", 4, BundledSource, "", "en", "chopsticks")
	b.Rank = 5
	c := word("ねこ", 5, BundledSource, "", "en", "cat") // no rank
	d := word("ねこ", 5, BundledSource, "", "en", "kitty")
	d.Rank = 60
	own := word("はし", 0, "mine", "", "en", "edge")
	own.Rank = 1 // a private word's rank is ignored
	ix := NewIndex([]Word{a, b, c, d}, []List{{Name: "mine", Words: []Word{own, word("いぬ", 0, "mine", "", "en", "dog")}}})
	if e, _ := ix.Lookup("はし"); e.Rank != 5 {
		t.Errorf("はし rank = %d, want the best bundled rank 5", e.Rank)
	}
	if e, _ := ix.Lookup("ねこ"); e.Rank != 60 {
		t.Errorf("ねこ rank = %d, want 60 (a known rank beats an unknown one)", e.Rank)
	}
	if e, _ := ix.Lookup("いぬ"); e.Rank != 0 {
		t.Errorf("private-only いぬ rank = %d, want 0", e.Rank)
	}
}
```

In `vocab/bundled_test.go`:

Replace:

```go
	if len(s.Meanings["de"]) == 0 {
		t.Fatalf("せんせい has no German meaning: %v", s.Meanings)
	}
}

func TestDecodeBundledMissingMeanings(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte(`[{"k":"ねこ","j":5},{"k":"いぬ","j":4,"m":{"en":["dog"]}}]`))
	zw.Close()
	words, err := decodeBundled(buf.Bytes())
	if err != nil {
```

with:

```go
	if len(s.Meanings["de"]) == 0 {
		t.Fatalf("せんせい has no German meaning: %v", s.Meanings)
	}
	if s.Rank < 1 || s.Rank > 49 {
		t.Fatalf("せんせい rank = %d, want a common word's rank (1–49)", s.Rank)
	}
	ranked := 0
	for _, w := range words {
		if w.Rank < 1 || w.Rank > 99 {
			t.Fatalf("%s: rank %d outside 1–99", w.Kana, w.Rank)
		}
		if w.Rank < 99 {
			ranked++
		}
	}
	if ranked < len(words)/2 {
		t.Fatalf("only %d of %d words have a priority rank", ranked, len(words))
	}
}

func TestDecodeBundledMissingMeanings(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte(`[{"k":"ねこ","j":5},{"k":"いぬ","j":4,"r":12,"m":{"en":["dog"]}}]`))
	zw.Close()
	words, err := decodeBundled(buf.Bytes())
	if err != nil {
```

Replace:

```go
		t.Fatalf("absent m must give empty, non-nil Meanings: %+v", words)
	}
	words[0].Meanings.Add("en", "cat") // must not panic on a nil map
	if words[1].Meanings["en"][0] != "dog" || words[1].Source != BundledSource {
		t.Fatalf("second word = %+v", words[1])
	}
```

with:

```go
		t.Fatalf("absent m must give empty, non-nil Meanings: %+v", words)
	}
	words[0].Meanings.Add("en", "cat") // must not panic on a nil map
	if words[0].Rank != 0 || words[1].Rank != 12 {
		t.Fatalf("ranks = %d, %d; want 0 (absent) and 12", words[0].Rank, words[1].Rank)
	}
	if words[1].Meanings["en"][0] != "dog" || words[1].Source != BundledSource {
		t.Fatalf("second word = %+v", words[1])
	}
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test -count=1 ./cmd/vocabgen/ ./vocab/`
Expected: FAIL — build errors such as `undefined: rankOf`, `undefined: newJMEntry`, `unknown field Kanji in struct literal of type xmlEntry`, `w.R undefined (type outWord has no field or method R)`; in `vocab`, `a.Rank undefined (type Word has no field or method Rank)` and `s.ListPos undefined`.

- [ ] **Step 3: Parse the priority tags in `cmd/vocabgen/jmdict.go`**

Replace:

```go
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
```

with:

```go
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
```

Replace:

```go
	Text string `xml:",chardata"`
}

type xmlEntry struct {
	Seq    string `xml:"ent_seq"`
	Senses []struct {
		Glosses []xmlGloss `xml:"gloss"`
	} `xml:"sense"`
}

var (
	// glossNumber is a leading "1) ", "1. ", "а) " (Cyrillic) or ": ".
	glossNumber = regexp.MustCompile(`^(?:\d+[.)]|[а-я]\))?\s*:?\s*`)
```

with:

```go
	Text string `xml:",chardata"`
}

// xmlKanji is a kanji element; only its priority tags are used.
type xmlKanji struct {
	Pri []string `xml:"ke_pri"`
}

// xmlReading is a reading element with its priority tags.
type xmlReading struct {
	Reb string   `xml:"reb"`
	Pri []string `xml:"re_pri"`
}

type xmlEntry struct {
	Seq      string       `xml:"ent_seq"`
	Kanji    []xmlKanji   `xml:"k_ele"`
	Readings []xmlReading `xml:"r_ele"`
	Senses   []struct {
		Glosses []xmlGloss `xml:"gloss"`
	} `xml:"sense"`
}

// Frequency ranks: nfXX gives XX (1–48), a "1" tag 49, a "2" tag 60.
const (
	rankFirst  = 49 // news1, ichi1, spec1, gai1
	rankSecond = 60 // news2, ichi2, spec2, gai2
	rankNone   = 99 // no priority tag
)

// tagRank is the rank one priority tag gives.
func tagRank(t string) int {
	switch t {
	case "news1", "ichi1", "spec1", "gai1":
		return rankFirst
	case "news2", "ichi2", "spec2", "gai2":
		return rankSecond
	}
	if s, ok := strings.CutPrefix(t, "nf"); ok {
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= 48 {
			return n
		}
	}
	return rankNone
}

// rankOf is the best (lowest) rank the priority tags give; rankNone
// without a usable tag.
func rankOf(tags []string) int {
	best := rankNone
	for _, t := range tags {
		best = min(best, tagRank(t))
	}
	return best
}

// jmEntry is what the generator keeps of one JMdict entry.
type jmEntry struct {
	glosses map[string][]string // language → meaning items
	ranks   map[string]int      // reading → rank of its own priority tags, if it has any
	best    int                 // best rank over every reading and kanji element
}

// rank is the frequency rank of reading: from the priority tags of the
// matching reading element, or, if that has none, the best tag of the
// whole entry.
func (e jmEntry) rank(reading string) int {
	if r, ok := e.ranks[reading]; ok {
		return r
	}
	return e.best
}

// newJMEntry collects the ranks of an entry; glosses are added by the caller.
func newJMEntry(x xmlEntry) jmEntry {
	e := jmEntry{glosses: map[string][]string{}, ranks: map[string]int{}}
	var all []string
	for _, r := range x.Readings {
		if len(r.Pri) > 0 {
			e.ranks[r.Reb] = rankOf(r.Pri)
		}
		all = append(all, r.Pri...)
	}
	for _, k := range x.Kanji {
		all = append(all, k.Pri...)
	}
	e.best = rankOf(all)
	return e
}

var (
	// glossNumber is a leading "1) ", "1. ", "а) " (Cyrillic) or ": ".
	glossNumber = regexp.MustCompile(`^(?:\d+[.)]|[а-я]\))?\s*:?\s*`)
```

Replace:

```go
}

// readJMdict streams the JMdict XML (plain or .gz) and returns, for every
// entry whose sequence number is wanted, the glosses per language: those of
// the first sense that has usable glosses in that language, split on ";",
// at most three items.
func readJMdict(path string, wanted map[string]bool) (map[string]map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
```

with:

```go
}

// readJMdict streams the JMdict XML (plain or .gz) and returns, for every
// entry whose sequence number is wanted, the glosses per language (those of
// the first sense that has usable glosses in that language, split on ";",
// at most three items) and the frequency ranks of its readings.
func readJMdict(path string, wanted map[string]bool) (map[string]jmEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
```

Replace:

```go
	}
	dec := xml.NewDecoder(r)
	dec.Strict = false // JMdict uses DTD entities (&n; …); their text is not needed
	out := map[string]map[string][]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
```

with:

```go
	}
	dec := xml.NewDecoder(r)
	dec.Strict = false // JMdict uses DTD entities (&n; …); their text is not needed
	out := map[string]jmEntry{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
```

Replace:

```go
		if !wanted[e.Seq] {
			continue
		}
		glosses := map[string][]string{}
		for _, sense := range e.Senses {
			// A language is taken only from the first sense that has
			// usable glosses in it.
```

with:

```go
		if !wanted[e.Seq] {
			continue
		}
		entry := newJMEntry(e)
		glosses := entry.glosses
		for _, sense := range e.Senses {
			// A language is taken only from the first sense that has
			// usable glosses in it.
```

Replace:

```go
				}
			}
		}
		out[e.Seq] = glosses
	}
}
```

with:

```go
				}
			}
		}
		out[e.Seq] = entry
	}
}
```

- [ ] **Step 4: Write the rank in `cmd/vocabgen/build.go`**

Replace:

```go
type outWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	M map[string][]string `json:"m"`
}
```

with:

```go
type outWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	R int                 `json:"r"` // frequency rank, lower = more common
	M map[string][]string `json:"m"`
}
```

Replace:

```go
			wanted[r.Seq] = true
		}
	}
	glosses, err := readJMdict(jmdictPath, wanted)
	if err != nil {
		return st, err
	}
```

with:

```go
			wanted[r.Seq] = true
		}
	}
	entries, err := readJMdict(jmdictPath, wanted)
	if err != nil {
		return st, err
	}
```

Replace:

```go
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
		for lang, glosses := range g {
			for _, gl := range glosses {
				// Already split for JMdict; Waller's fallback may need it.
```

with:

```go
			dropped[r.Kana] = true
			continue
		}
		var g map[string][]string
		rank := rankNone
		if e, ok := entries[r.Seq]; ok {
			g, rank = e.glosses, e.rank(r.Kana)
		} else {
			missing[r.Seq] = true
			g = map[string][]string{"en": wallerGlosses(r.Waller)}
		}
		ow, ok := byKana[r.Kana]
		if !ok {
			ow = &outWord{K: r.Kana, J: r.Level, R: rank, M: map[string][]string{}}
			byKana[r.Kana] = ow
		}
		if r.Level > ow.J {
			ow.J = r.Level
		}
		ow.R = min(ow.R, rank) // merged homophones take the best rank
		for lang, glosses := range g {
			for _, gl := range glosses {
				// Already split for JMdict; Waller's fallback may need it.
```

Run: `go test -count=1 ./cmd/vocabgen/` — expected: PASS.

- [ ] **Step 5: Carry the rank and list positions in `vocab`**

`vocab/vocab.go`:

Replace:

```go
	Kana     string   // reading, e.g. "せんせい"
	Meanings Meanings // language → alternatives
	JLPT     int      // 5…1, 0 = none
	Source   string   // "jmdict" or a list name
	Lesson   string   // optional, private lists only
}
```

with:

```go
	Kana     string   // reading, e.g. "せんせい"
	Meanings Meanings // language → alternatives
	JLPT     int      // 5…1, 0 = none
	Rank     int      // bundled words: frequency rank, lower = more common; 0 = unknown
	Source   string   // "jmdict" or a list name
	Lesson   string   // optional, private lists only
}
```

`vocab/bundled.go`:

Replace:

```go
type bundledWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	M map[string][]string `json:"m"`
}
```

with:

```go
type bundledWord struct {
	K string              `json:"k"`
	J int                 `json:"j"`
	R int                 `json:"r"`
	M map[string][]string `json:"m"`
}
```

Replace:

```go
		if m == nil {
			m = Meanings{}
		}
		words[i] = Word{Kana: r.K, JLPT: r.J, Meanings: m, Source: BundledSource}
	}
	return words, nil
}
```

with:

```go
		if m == nil {
			m = Meanings{}
		}
		words[i] = Word{Kana: r.K, JLPT: r.J, Rank: r.R, Meanings: m, Source: BundledSource}
	}
	return words, nil
}
```

`vocab/index.go`:

Replace:

```go
	Units    []string // kanacore.KanaUnits, computed once
	Meanings Meanings
	JLPT     int                 // easiest bundled level, 0 if not bundled
	Bundled  bool                // the bundled list contains it
	Lists    map[string][]string // private list name → lessons
}

// Index holds one entry per reading.
```

with:

```go
	Units    []string // kanacore.KanaUnits, computed once
	Meanings Meanings
	JLPT     int                 // easiest bundled level, 0 if not bundled
	Rank     int                 // best bundled frequency rank, lower = more common; 0 = unknown
	Bundled  bool                // the bundled list contains it
	Lists    map[string][]string // private list name → lessons
	ListPos  map[string]int      // private list name → index of its first row in that list
}

// Index holds one entry per reading.
```

Replace:

```go
func NewIndex(bundled []Word, lists []List) *Index {
	ix := &Index{byKana: map[string]int{}}
	for _, w := range bundled {
		ix.add(w, "")
	}
	ix.addLists(lists)
	ix.sort()
```

with:

```go
func NewIndex(bundled []Word, lists []List) *Index {
	ix := &Index{byKana: map[string]int{}}
	for _, w := range bundled {
		ix.add(w, "", 0)
	}
	ix.addLists(lists)
	ix.sort()
```

Replace:

```go

func (ix *Index) addLists(lists []List) {
	for _, l := range lists {
		for _, w := range l.Words {
			ix.add(w, l.Name)
		}
	}
}

// add merges one word into its entry; list is "" for the bundled list.
func (ix *Index) add(w Word, list string) {
	i, ok := ix.byKana[w.Kana]
	if !ok {
		units, valid := kanacore.KanaUnits(w.Kana)
```

with:

```go

func (ix *Index) addLists(lists []List) {
	for _, l := range lists {
		for i, w := range l.Words {
			ix.add(w, l.Name, i)
		}
	}
}

// add merges one word into its entry; list is "" for the bundled list, and
// pos is the word's row index in its list.
func (ix *Index) add(w Word, list string, pos int) {
	i, ok := ix.byKana[w.Kana]
	if !ok {
		units, valid := kanacore.KanaUnits(w.Kana)
```

Replace:

```go
			return
		}
		ix.byKana[w.Kana] = len(ix.entries)
		ix.entries = append(ix.entries, Entry{Kana: w.Kana, Units: units, Meanings: Meanings{}, Lists: map[string][]string{}})
		i = len(ix.entries) - 1
	} else if i < ix.shared && !ix.owned[i] {
		ix.entries[i] = ix.entries[i].clone()
```

with:

```go
			return
		}
		ix.byKana[w.Kana] = len(ix.entries)
		ix.entries = append(ix.entries, Entry{Kana: w.Kana, Units: units, Meanings: Meanings{}, Lists: map[string][]string{}, ListPos: map[string]int{}})
		i = len(ix.entries) - 1
	} else if i < ix.shared && !ix.owned[i] {
		ix.entries[i] = ix.entries[i].clone()
```

Replace:

```go
		if !e.Bundled || w.JLPT > e.JLPT {
			e.JLPT = w.JLPT
		}
		e.Bundled = true
		return
	}
	lessons := e.Lists[list]
	if w.Lesson != "" && !contains(lessons, w.Lesson) {
		lessons = append(lessons, w.Lesson)
```

with:

```go
		if !e.Bundled || w.JLPT > e.JLPT {
			e.JLPT = w.JLPT
		}
		if !e.Bundled || rankKey(w.Rank) < rankKey(e.Rank) {
			e.Rank = w.Rank
		}
		e.Bundled = true
		return
	}
	if _, ok := e.ListPos[list]; !ok {
		e.ListPos[list] = pos
	}
	lessons := e.Lists[list]
	if w.Lesson != "" && !contains(lessons, w.Lesson) {
		lessons = append(lessons, w.Lesson)
```

Replace:

```go
	ix.owned = nil
}

// clone copies the entry's Meanings and Lists, so merging into the copy
// leaves the original untouched. Units is never modified and stays shared.
func (e Entry) clone() Entry {
	m := make(Meanings, len(e.Meanings))
	for lang, items := range e.Meanings {
```

with:

```go
	ix.owned = nil
}

// rankKey orders frequency ranks: an unknown rank (0) sorts as 99, after
// every word with a priority tag.
func rankKey(r int) int {
	if r <= 0 {
		return 99
	}
	return r
}

// clone copies the entry's Meanings, Lists and ListPos, so merging into the
// copy leaves the original untouched. Units is never modified and stays
// shared.
func (e Entry) clone() Entry {
	m := make(Meanings, len(e.Meanings))
	for lang, items := range e.Meanings {
```

Replace:

```go
	for name, lessons := range e.Lists {
		l[name] = slices.Clone(lessons)
	}
	e.Meanings, e.Lists = m, l
	return e
}
```

with:

```go
	for name, lessons := range e.Lists {
		l[name] = slices.Clone(lessons)
	}
	e.Meanings, e.Lists, e.ListPos = m, l, maps.Clone(e.ListPos)
	return e
}
```

Run: `go test -count=1 ./vocab/`
Expected: only `TestBundled` fails (`せんせい rank = 0, want a common word's rank (1–49)`): the committed data has no ranks yet.

- [ ] **Step 6: Regenerate the word data**

```bash
d=$(mktemp -d)
curl -fsSL -D "$d/headers.txt" -o "$d/JMdict.gz" http://ftp.edrdg.org/pub/Nihongo/JMdict.gz
git clone -q --depth 1 https://github.com/stephenmk/yomitan-jlpt-vocab "$d/jlpt"
modified=$(grep -i '^last-modified:' "$d/headers.txt" | cut -d' ' -f2- | tr -d '\r')
commit=$(git -C "$d/jlpt" rev-parse HEAD)
go run ./cmd/vocabgen -jmdict "$d/JMdict.gz" -jlpt "$d/jlpt/original_data" -out vocab/data/words.json.gz
printf 'jmdict-last-modified: %s\njlpt-mapping-commit: %s\n' "$modified" "$commit" > vocab/data/SOURCES
cat vocab/data/SOURCES
```

Expected output of vocabgen (on 2026-10-15): `8293 JLPT rows, 7029 words written, 2 readings dropped (not typeable), 0 sequence numbers missing in JMdict, 14 rows without sequence number`; `SOURCES` then reads (dates and commit may be newer):

```text
jmdict-last-modified: Thu, 15 Oct 2026 03:30:22 GMT
jlpt-mapping-commit: b062d4e38c4bdd0950ae1d4ec55f04b176182e03
```

`.github/workflows/update-words.yml` needs no change: it runs the same tool and writes the same two lines.

- [ ] **Step 7: Run everything**

Run: `go test -count=1 ./vocab/ ./cmd/vocabgen/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...` — all clean.

- [ ] **Step 8: Commit**

```bash
git add cmd/vocabgen/jmdict.go cmd/vocabgen/build.go cmd/vocabgen/testdata/JMdict_sample.xml cmd/vocabgen/vocabgen_test.go \
	vocab/vocab.go vocab/bundled.go vocab/index.go vocab/index_test.go vocab/bundled_test.go \
	vocab/data/words.json.gz vocab/data/SOURCES
git commit -m "feat(vocab): add a frequency rank to the word data

vocabgen reads JMdict's priority tags (nfXX, news/ichi/spec/gai) of the
reading that matches each row, or the best tag of the entry, and writes
a rank per word; merged homophones keep the best one. vocab carries the
rank and, for list entries, the index of their first row, and the data
is regenerated from the current JMdict."
```

---

### Task 2: `vocab.NextPackage`

**Files:** Modify `vocab/index.go`. Create `vocab/package.go`, `vocab/package_test.go`.

- [ ] **Step 1: Write the failing tests** — create `vocab/package_test.go`:

```go
package vocab

import (
	"reflect"
	"testing"
)

func ranked(kana string, level, rank int) Word {
	w := word(kana, level, BundledSource, "", "en", kana)
	w.Rank = rank
	return w
}

func knowAll(string) bool { return true }

func TestNextPackageJLPTOrder(t *testing.T) {
	ix := NewIndex([]Word{
		ranked("みず", 5, 1),
		ranked("あめ", 5, 49),
		ranked("いぬ", 5, 49),
		ranked("いぬごや", 5, 49), // more units than いぬ
		ranked("き", 5, 0),     // unknown rank sorts as 99, after 60
		ranked("ねこ", 5, 60),
		ranked("あう", 4, 1), // N4 comes after every N5 word
		ranked("かんきょう", 2, 1),
	}, nil)
	got := kanaOf(NextPackage(ix, Source{Level: 4}, nil, 10, knowAll))
	want := []string{"みず", "あめ", "いぬ", "いぬごや", "ねこ", "き", "あう"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if got := kanaOf(NextPackage(ix, Source{Level: 5}, nil, 10, knowAll)); len(got) != 6 {
		t.Fatalf("N5 only = %v, want the six N5 words", got)
	}
	if got := kanaOf(NextPackage(ix, Source{Level: 4}, nil, 3, knowAll)); !reflect.DeepEqual(got, want[:3]) {
		t.Fatalf("n = 3: %v", got)
	}
}

func TestNextPackageSkipsIntroducedAndUnplayable(t *testing.T) {
	ix := NewIndex([]Word{ranked("あい", 5, 1), ranked("かお", 5, 2), ranked("あお", 5, 3), ranked("いえ", 5, 4)}, nil)
	introduced := map[string]bool{"あい": true}
	known := func(u string) bool { return u != "か" } // かお is not playable yet
	got := kanaOf(NextPackage(ix, Source{Level: 5}, introduced, 5, known))
	if !reflect.DeepEqual(got, []string{"あお", "いえ"}) {
		t.Fatalf("package = %v, want [あお いえ]: introduced and unplayable words skipped, a partial package", got)
	}
	if got := NextPackage(ix, Source{Level: 5}, map[string]bool{"あい": true, "かお": true, "あお": true, "いえ": true}, 5, knowAll); got != nil {
		t.Fatalf("everything introduced: %v, want none", kanaOf(got))
	}
	if got := NextPackage(ix, Source{Level: 5}, nil, 0, knowAll); got != nil {
		t.Fatalf("n = 0: %v", kanaOf(got))
	}
	if got := NextPackage(ix, Source{List: "nope"}, nil, 5, knowAll); got != nil {
		t.Fatalf("unknown list: %v", kanaOf(got))
	}
}

func TestNextPackageListOrder(t *testing.T) {
	list := func(rows ...[2]string) List {
		l := List{Name: "minna"}
		for _, r := range rows {
			l.Words = append(l.Words, word(r[0], 0, "minna", r[1], "de", r[0]))
		}
		return l
	}
	cases := []struct {
		name string
		list List
		want []string
	}{
		{"numeric lessons", list(
			[2]string{"いぬ", "10"}, [2]string{"ねこ", "2"}, [2]string{"あめ", ""},
			[2]string{"みず", "2"}, [2]string{"き", "1"},
		), []string{"き", "ねこ", "みず", "いぬ", "あめ"}},
		{"named lessons by first appearance", list(
			[2]string{"いぬ", "Tiere"}, [2]string{"あめ", ""}, [2]string{"みず", "Wetter"},
			[2]string{"ねこ", "Tiere"}, [2]string{"き", "1"},
		), []string{"いぬ", "ねこ", "みず", "き", "あめ"}},
		{"entry in two lessons counts for the first", list(
			[2]string{"いぬ", "3"}, [2]string{"ねこ", "2"}, [2]string{"いぬ", "1"},
		), []string{"いぬ", "ねこ"}},
		{"no lessons: file order", list(
			[2]string{"みず", ""}, [2]string{"あめ", ""}, [2]string{"いぬ", ""},
		), []string{"みず", "あめ", "いぬ"}},
	}
	for _, c := range cases {
		for name, ix := range map[string]*Index{
			"NewIndex":  NewIndex([]Word{ranked("いぬ", 5, 1)}, []List{c.list}),
			"WithLists": NewIndex([]Word{ranked("いぬ", 5, 1)}, nil).WithLists([]List{c.list}),
		} {
			got := kanaOf(NextPackage(ix, Source{List: "minna"}, nil, 10, knowAll))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s (%s): %v, want %v", c.name, name, got, c.want)
			}
		}
	}
}

func TestLessonIsTheFirstInListOrder(t *testing.T) {
	ix := NewIndex(nil, []List{{Name: "minna", Words: []Word{
		word("いぬ", 0, "minna", "3", "de", "Hund"),
		word("ねこ", 0, "minna", "", "de", "Katze"),
		word("いぬ", 0, "minna", "1", "de", "Hund"),
	}}})
	e, _ := ix.Lookup("いぬ")
	if got := ix.Lesson(e, "minna"); got != "1" {
		t.Fatalf("Lesson(いぬ) = %q, want 1", got)
	}
	n, _ := ix.Lookup("ねこ")
	if got := ix.Lesson(n, "minna"); got != "" {
		t.Fatalf("Lesson(ねこ) = %q, want none", got)
	}
}

func TestNextPackageOrderIsCachedPerSource(t *testing.T) {
	ix := NewIndex([]Word{ranked("あい", 5, 1)}, []List{{Name: "a", Words: []Word{word("いえ", 0, "a", "", "en", "house")}}})
	NextPackage(ix, Source{Level: 5}, nil, 1, knowAll)
	NextPackage(ix, Source{Level: 3}, nil, 1, knowAll)
	NextPackage(ix, Source{List: "a"}, nil, 1, knowAll)
	if len(ix.orders) != 2 || ix.orders[""] == nil || ix.orders["a"] == nil {
		t.Fatalf("orders = %v, want one per source (JLPT for every level, and list a)", ix.orders)
	}
}

func TestSourceHas(t *testing.T) {
	ix := testIndex()
	cases := []struct {
		src  Source
		kana string
		want bool
	}{
		{Source{Level: 5}, "ねこ", true},
		{Source{Level: 5}, "きって", false}, // N4
		{Source{Level: 4}, "きって", true},
		{Source{Level: 5}, "せんせい", false}, // private only
		{Source{List: "minna"}, "せんせい", true},
		{Source{List: "minna"}, "ねこ", false},
	}
	for _, c := range cases {
		e, _ := ix.Lookup(c.kana)
		if got := c.src.Has(e); got != c.want {
			t.Errorf("%+v.Has(%s) = %v, want %v", c.src, c.kana, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test -count=1 ./vocab/`
Expected: FAIL — `undefined: NextPackage`, `undefined: Source`.

- [ ] **Step 3: Record each list's lessons and keep the order cache in `vocab/index.go`**

`add` now reports whether the word was added, so `addLists` records only the lessons of words that made it into the index:

Replace:

```go
	"maps"
	"slices"
	"sort"

	"kana/kanacore"
)
```

with:

```go
	"maps"
	"slices"
	"sort"
	"sync"

	"kana/kanacore"
)
```

Replace:

```go
	// base index unless owned marks them as copied.
	shared int
	owned  map[int]bool
}

// NewIndex merges bundled words and private lists into one entry per
// reading. Words whose reading cannot be split into units are dropped.
func NewIndex(bundled []Word, lists []List) *Index {
	ix := &Index{byKana: map[string]int{}}
	for _, w := range bundled {
		ix.add(w, "", 0)
	}
```

with:

```go
	// base index unless owned marks them as copied.
	shared int
	owned  map[int]bool

	lessons map[string][]string // private list name → its lessons in first-appearance order

	mu     sync.Mutex
	orders map[string][]int // learning order per source (list name, "" = JLPT), see order
}

// NewIndex merges bundled words and private lists into one entry per
// reading. Words whose reading cannot be split into units are dropped.
func NewIndex(bundled []Word, lists []List) *Index {
	ix := &Index{byKana: map[string]int{}, lessons: map[string][]string{}}
	for _, w := range bundled {
		ix.add(w, "", 0)
	}
```

Replace:

```go
		entries: slices.Clone(ix.entries),
		byKana:  maps.Clone(ix.byKana),
		shared:  len(ix.entries),
	}
	out.addLists(lists)
	out.shared = 0
```

with:

```go
		entries: slices.Clone(ix.entries),
		byKana:  maps.Clone(ix.byKana),
		shared:  len(ix.entries),
		lessons: maps.Clone(ix.lessons),
	}
	out.addLists(lists)
	out.shared = 0
```

Replace:

```go
func (ix *Index) addLists(lists []List) {
	for _, l := range lists {
		for i, w := range l.Words {
			ix.add(w, l.Name, i)
		}
	}
}

// add merges one word into its entry; list is "" for the bundled list, and
// pos is the word's row index in its list.
func (ix *Index) add(w Word, list string, pos int) {
	i, ok := ix.byKana[w.Kana]
	if !ok {
		units, valid := kanacore.KanaUnits(w.Kana)
		if !valid {
			return
		}
		ix.byKana[w.Kana] = len(ix.entries)
		ix.entries = append(ix.entries, Entry{Kana: w.Kana, Units: units, Meanings: Meanings{}, Lists: map[string][]string{}, ListPos: map[string]int{}})
```

with:

```go
func (ix *Index) addLists(lists []List) {
	for _, l := range lists {
		for i, w := range l.Words {
			if ix.add(w, l.Name, i) && w.Lesson != "" && !contains(ix.lessons[l.Name], w.Lesson) {
				ix.lessons[l.Name] = append(ix.lessons[l.Name], w.Lesson)
			}
		}
	}
}

// add merges one word into its entry; list is "" for the bundled list, and
// pos is the word's row index in its list. It reports whether the word was
// added: a reading that cannot be split into units is dropped.
func (ix *Index) add(w Word, list string, pos int) bool {
	i, ok := ix.byKana[w.Kana]
	if !ok {
		units, valid := kanacore.KanaUnits(w.Kana)
		if !valid {
			return false
		}
		ix.byKana[w.Kana] = len(ix.entries)
		ix.entries = append(ix.entries, Entry{Kana: w.Kana, Units: units, Meanings: Meanings{}, Lists: map[string][]string{}, ListPos: map[string]int{}})
```

Replace:

```go
			e.Rank = w.Rank
		}
		e.Bundled = true
		return
	}
	if _, ok := e.ListPos[list]; !ok {
		e.ListPos[list] = pos
```

with:

```go
			e.Rank = w.Rank
		}
		e.Bundled = true
		return true
	}
	if _, ok := e.ListPos[list]; !ok {
		e.ListPos[list] = pos
```

Replace:

```go
		lessons = append(lessons, w.Lesson)
	}
	e.Lists[list] = lessons
}

// sort orders the entries by reading and rebuilds byKana.
```

with:

```go
		lessons = append(lessons, w.Lesson)
	}
	e.Lists[list] = lessons
	return true
}

// sort orders the entries by reading and rebuilds byKana.
```

- [ ] **Step 4: Create `vocab/package.go`**

```go
package vocab

import (
	"sort"
	"strconv"
	"strings"
)

// Source names where new words come from.
type Source struct {
	List  string // private list name; "" = JLPT (the bundled list)
	Level int    // JLPT: lowest level number allowed (5 = N5 only); ignored for lists
}

// Has reports whether e belongs to the source: for JLPT a bundled entry
// with JLPT >= Level, for a list an entry of that list.
func (s Source) Has(e Entry) bool {
	if s.List == "" {
		return e.Bundled && e.JLPT >= s.Level
	}
	return hasList(e, s.List)
}

// NextPackage returns up to n entries from src that are not introduced and
// whose units are all known, in learning order. Entries that are not
// playable yet are skipped, not waited for: they come once their kana are
// known.
//
// JLPT order: level from N5 down to src.Level, then frequency rank, then
// fewer units, then the reading. List order: by lesson (numeric if every
// lesson of the list is an integer, else by first appearance; an entry in
// several lessons counts for its first), then file order; entries without
// a lesson last, in file order.
func NextPackage(ix *Index, src Source, introduced map[string]bool, n int, known func(string) bool) []Entry {
	if ix == nil || n <= 0 {
		return nil
	}
	var out []Entry
	for _, i := range ix.order(src.List) {
		e := ix.entries[i]
		if !src.Has(e) || introduced[e.Kana] || !allKnown(e.Units, known) {
			continue
		}
		out = append(out, e)
		if len(out) == n {
			break
		}
	}
	return out
}

// Lesson returns e's first lesson in the list's lesson order, "" if it has
// none in that list.
func (ix *Index) Lesson(e Entry, list string) string {
	lesson, _, _ := firstLesson(e, list, ix.lessonRanks(list))
	return lesson
}

// order returns the indices of a source's entries in learning order (list
// name, "" for JLPT; levels are filtered by the caller). It is computed
// once per index and source, then cached; the entries never change after
// the index is built.
func (ix *Index) order(list string) []int {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if o, ok := ix.orders[list]; ok {
		return o
	}
	var idx []int
	for i, e := range ix.entries {
		if (list == "" && e.Bundled) || (list != "" && hasList(e, list)) {
			idx = append(idx, i)
		}
	}
	if list == "" {
		sort.SliceStable(idx, func(a, b int) bool {
			x, y := ix.entries[idx[a]], ix.entries[idx[b]]
			switch {
			case x.JLPT != y.JLPT:
				return x.JLPT > y.JLPT
			case rankKey(x.Rank) != rankKey(y.Rank):
				return rankKey(x.Rank) < rankKey(y.Rank)
			case len(x.Units) != len(y.Units):
				return len(x.Units) < len(y.Units)
			}
			return x.Kana < y.Kana
		})
	} else {
		ranks := ix.lessonRanks(list)
		type key struct {
			hasLesson bool
			lesson    int
			pos       int
		}
		keys := make(map[int]key, len(idx))
		for _, i := range idx {
			e := ix.entries[i]
			_, r, ok := firstLesson(e, list, ranks)
			keys[i] = key{hasLesson: ok, lesson: r, pos: e.ListPos[list]}
		}
		sort.SliceStable(idx, func(a, b int) bool {
			x, y := keys[idx[a]], keys[idx[b]]
			switch {
			case x.hasLesson != y.hasLesson:
				return x.hasLesson
			case x.lesson != y.lesson:
				return x.lesson < y.lesson
			}
			return x.pos < y.pos
		})
	}
	if ix.orders == nil {
		ix.orders = map[string][]int{}
	}
	ix.orders[list] = idx
	return idx
}

// lessonRanks ranks a list's lessons: by their number if every lesson is
// an integer, else by first appearance in the list.
func (ix *Index) lessonRanks(list string) map[string]int {
	lessons := ix.lessons[list]
	numbers := make([]int, len(lessons))
	numeric := true
	for i, l := range lessons {
		n, err := strconv.Atoi(strings.TrimSpace(l))
		if err != nil {
			numeric = false
			break
		}
		numbers[i] = n
	}
	ranks := make(map[string]int, len(lessons))
	for i, l := range lessons {
		if numeric {
			ranks[l] = numbers[i]
		} else {
			ranks[l] = i
		}
	}
	return ranks
}

// firstLesson returns e's lesson in list that comes first by ranks and its
// rank; ok is false when e has no lesson there.
func firstLesson(e Entry, list string, ranks map[string]int) (lesson string, rank int, ok bool) {
	for _, l := range e.Lists[list] {
		if r, known := ranks[l]; known && (!ok || r < rank) {
			lesson, rank, ok = l, r, true
		}
	}
	return lesson, rank, ok
}

func hasList(e Entry, list string) bool {
	_, ok := e.Lists[list]
	return ok
}
```

- [ ] **Step 5: Run**

Run: `go test -count=1 ./vocab/` — expected: PASS. Then `gofmt -l .`, `go vet ./...` (the `sync.Mutex` in `Index` is never copied: indexes are only used as `*Index`), `go test -count=1 ./...`.

- [ ] **Step 6: Commit**

```bash
git add vocab/index.go vocab/package.go vocab/package_test.go
git commit -m "feat(vocab): pick the next word package in learning order

NextPackage returns the next words of a source that are not introduced
and can be read: JLPT level by level from N5, then by frequency rank,
fewer kana and reading; a private list by lesson, then file order. The
order is computed once per index and source and cached."
```

---

### Task 3: Store — introduced words and the new settings

**Files:** Modify `store/store.go`, `store/store_test.go`.

- [ ] **Step 1: Write the failing tests** — in `store/store_test.go`, add `"reflect"` to the imports and append the tests:

Replace:

```go
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"
```

with:

```go
	"database/sql"
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"
```

Replace:

```go
		t.Fatal("BundledWords = true after saving false")
	}
}
```

with:

```go
		t.Fatal("BundledWords = true after saving false")
	}
}

func TestIntroducedWordsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.IntroducedWords()
	if err != nil || len(got) != 0 {
		t.Fatalf("fresh IntroducedWords = %v, %v", got, err)
	}
	at := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	if err := st.SaveIntroduced("ねこ", at); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveIntroduced("ねこ", at.Add(time.Hour)); err != nil {
		t.Fatalf("saving an introduced word again: %v", err)
	}
	if err := st.SaveIntroduced("", at); err == nil {
		t.Fatal("an empty reading must be rejected")
	}
	got, err = st.IntroducedWords()
	if err != nil || !reflect.DeepEqual(got, map[string]bool{"ねこ": true}) {
		t.Fatalf("IntroducedWords = %v, %v", got, err)
	}
	var stamp int64
	if err := st.db.QueryRow(`SELECT introduced_at FROM introduced_words WHERE kana = 'ねこ'`).Scan(&stamp); err != nil || stamp != at.Unix() {
		t.Fatalf("introduced_at = %d, %v; want the first time %d", stamp, err, at.Unix())
	}
}

func TestIntroducedSeededOnceFromWordProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// A stage 3 database: word progress, but no introduced_words table yet.
	_ = st.SaveWordStats("ねこ", 2, 1)
	_ = st.SaveWordStats("いぬ", 0, 1)
	_ = st.SaveMemoryState("word:あめ", memory.State{Stability: 1, LastSeen: time.Now()})
	_ = st.SaveMemoryState("word:ねこ", memory.State{Stability: 1, LastSeen: time.Now()})
	_ = st.SaveMemoryState("kana:か", memory.State{Stability: 1, LastSeen: time.Now()})
	if _, err := st.db.Exec(`DROP TABLE introduced_words`); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.IntroducedWords()
	if want := map[string]bool{"ねこ": true, "いぬ": true, "あめ": true}; err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("seeded = %v, %v; want %v", got, err, want)
	}
	_ = st.SaveWordStats("みず", 1, 0)
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if got, _ := st.IntroducedWords(); got["みず"] {
		t.Fatal("re-open must not seed again: word stats alone do not introduce a word any more")
	}
}

func TestWordPackageSettings(t *testing.T) {
	st := openTemp(t)
	if n, err := st.PackageSize(); err != nil || n != DefaultPackageSize || DefaultPackageSize != 5 {
		t.Fatalf("unset PackageSize = %d, %v; want 5", n, err)
	}
	if s, err := st.WordSource(); err != nil || s != "" {
		t.Fatalf("unset WordSource = %q, %v; want \"\" (auto)", s, err)
	}
	if f, err := st.KanaFont(); err != nil || f != "noto" {
		t.Fatalf("unset KanaFont = %q, %v; want noto", f, err)
	}
	if p, err := st.NewestPackage(); err != nil || p != nil {
		t.Fatalf("unset NewestPackage = %v, %v; want nil", p, err)
	}

	for _, n := range []int{3, 10} {
		if err := st.SavePackageSize(n); err != nil {
			t.Fatal(err)
		}
		if got, _ := st.PackageSize(); got != n {
			t.Fatalf("PackageSize = %d, want %d", got, n)
		}
	}
	if err := st.SavePackageSize(4); err == nil {
		t.Fatal("package size 4 must be rejected")
	}
	if err := st.setSetting(packageSizeKey, "7"); err != nil {
		t.Fatal(err)
	}
	if n, _ := st.PackageSize(); n != DefaultPackageSize {
		t.Fatalf("invalid stored size = %d, want the default", n)
	}

	if err := st.SaveWordSource("list:minna-no-nihongo"); err != nil {
		t.Fatal(err)
	}
	if s, _ := st.WordSource(); s != "list:minna-no-nihongo" {
		t.Fatalf("WordSource = %q", s)
	}
	if err := st.SaveKanaFont("mplus"); err != nil {
		t.Fatal(err)
	}
	if f, _ := st.KanaFont(); f != "mplus" {
		t.Fatalf("KanaFont = %q", f)
	}
	if err := st.SaveNewestPackage([]string{"ねこ", "いぬ"}); err != nil {
		t.Fatal(err)
	}
	if p, _ := st.NewestPackage(); !reflect.DeepEqual(p, []string{"ねこ", "いぬ"}) {
		t.Fatalf("NewestPackage = %v", p)
	}
	if err := st.SaveNewestPackage(nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := st.NewestPackage(); p != nil {
		t.Fatalf("NewestPackage after saving nil = %v, want nil", p)
	}
}
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test -count=1 ./store/`
Expected: FAIL — `st.IntroducedWords undefined`, `st.SaveIntroduced undefined`, `st.PackageSize undefined`, `undefined: DefaultPackageSize`, `st.WordSource undefined`.

- [ ] **Step 3: Implement** — in `store/store.go`:

Replace:

```go
	wordLevelKey       = "word_level"
	meaningLangKey     = "meaning_lang"
	bundledWordsKey    = "bundled_words"
	databaseFilePerm   = 0o644
	databaseDirPerm    = 0o755
	defaultOpenTimout  = 5 * time.Second
```

with:

```go
	wordLevelKey       = "word_level"
	meaningLangKey     = "meaning_lang"
	bundledWordsKey    = "bundled_words"
	packageSizeKey     = "package_size"
	wordSourceKey      = "word_source"
	kanaFontKey        = "kana_font"
	newestPackageKey   = "newest_package"
	databaseFilePerm   = 0o644
	databaseDirPerm    = 0o755
	defaultOpenTimout  = 5 * time.Second
```

Replace:

```go
// DefaultWordLevel is the JLPT level words are played up to when unset (N5).
const DefaultWordLevel = 5

// Store provides persisted access to user settings and kana statistics.
type Store struct {
	db *sql.DB
```

with:

```go
// DefaultWordLevel is the JLPT level words are played up to when unset (N5).
const DefaultWordLevel = 5

// DefaultPackageSize is the number of words in a word package when unset.
const DefaultPackageSize = 5

// DefaultKanaFont is the kana font when unset (Noto Sans JP).
const DefaultKanaFont = "noto"

// Store provides persisted access to user settings and kana statistics.
type Store struct {
	db *sql.DB
```

Replace:

```go
	return s.setSetting(meaningLangKey, lang)
}

// SaveKanaStats upserts the aggregated statistics for the provided kana.
func (s *Store) SaveKanaStats(char string, correctCount, missCount, streak int) error {
	if char == "" {
```

with:

```go
	return s.setSetting(meaningLangKey, lang)
}

// validPackageSize reports whether n is a package size the app offers.
func validPackageSize(n int) bool {
	return n == 3 || n == 5 || n == 10
}

// PackageSize returns the number of words per word package: 3, 5 or 10.
// Unset or invalid values give DefaultPackageSize.
func (s *Store) PackageSize() (int, error) {
	value, err := s.getSetting(packageSizeKey)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultPackageSize, nil
	}
	if err != nil {
		return DefaultPackageSize, err
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(value))
	if convErr != nil || !validPackageSize(n) {
		return DefaultPackageSize, nil
	}
	return n, nil
}

// SavePackageSize persists the package size (3, 5 or 10).
func (s *Store) SavePackageSize(n int) error {
	if !validPackageSize(n) {
		return fmt.Errorf("store: package size %d is not 3, 5 or 10", n)
	}
	return s.setSetting(packageSizeKey, strconv.Itoa(n))
}

// WordSource returns where new words come from ("jlpt" or "list:<name>"),
// or "" when unset: the app then picks a default and saves it.
func (s *Store) WordSource() (string, error) {
	value, err := s.getSetting(wordSourceKey)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SaveWordSource persists where new words come from.
func (s *Store) SaveWordSource(source string) error {
	return s.setSetting(wordSourceKey, source)
}

// KanaFont returns the kana font setting ("noto", "mplus" or "system"), or
// DefaultKanaFont when unset. The app checks the value.
func (s *Store) KanaFont() (string, error) {
	value, err := s.getSetting(kanaFontKey)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultKanaFont, nil
	}
	if err != nil {
		return DefaultKanaFont, err
	}
	return value, nil
}

// SaveKanaFont persists the kana font setting.
func (s *Store) SaveKanaFont(name string) error {
	return s.setSetting(kanaFontKey, name)
}

// NewestPackage returns the readings of the newest word package, nil when
// unset or empty.
func (s *Store) NewestPackage() ([]string, error) {
	value, err := s.getSetting(newestPackageKey)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && value == "") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Split(value, "\n"), nil
}

// SaveNewestPackage persists the readings of the newest word package,
// joined with "\n"; nil or empty stores an empty package.
func (s *Store) SaveNewestPackage(readings []string) error {
	return s.setSetting(newestPackageKey, strings.Join(readings, "\n"))
}

// IntroducedWords returns the readings that were shown in a word package
// intro (or counted as introduced when the table was created).
func (s *Store) IntroducedWords() (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT kana FROM introduced_words`)
	if err != nil {
		return nil, fmt.Errorf("store: query introduced words: %w", err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var kana string
		if err := rows.Scan(&kana); err != nil {
			return nil, fmt.Errorf("store: scan introduced words: %w", err)
		}
		out[kana] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate introduced words: %w", err)
	}
	return out, nil
}

// SaveIntroduced records a reading as introduced at the given time. A
// reading that is already introduced keeps its first time.
func (s *Store) SaveIntroduced(kana string, at time.Time) error {
	if kana == "" {
		return errors.New("store: word reading is required")
	}
	_, err := s.db.Exec(`
		INSERT INTO introduced_words (kana, introduced_at) VALUES (?, ?)
		ON CONFLICT(kana) DO NOTHING
	`, kana, at.Unix())
	if err != nil {
		return fmt.Errorf("store: save introduced word %s: %w", kana, err)
	}
	return nil
}

// SaveKanaStats upserts the aggregated statistics for the provided kana.
func (s *Store) SaveKanaStats(char string, correctCount, missCount, streak int) error {
	if char == "" {
```

Replace:

```go
			return fmt.Errorf("store: migrate statement failed: %w", err)
		}
	}
	return ensureMemory(ctx, db, time.Now())
}

// ensureMemory creates the memory table if it is missing and, in the same
```

with:

```go
			return fmt.Errorf("store: migrate statement failed: %w", err)
		}
	}
	now := time.Now()
	if err := ensureMemory(ctx, db, now); err != nil {
		return err
	}
	return ensureIntroduced(ctx, db, now)
}

// ensureIntroduced creates the introduced_words table if it is missing and,
// in the same transaction, seeds it with every reading that has word stats
// or a word memory state, so words learned before word packages keep
// falling. If anything fails nothing is committed and the next open tries
// again.
func ensureIntroduced(ctx context.Context, db *sql.DB, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin introduced migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='introduced_words'`).Scan(&name)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("store: check introduced_words table: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `CREATE TABLE introduced_words (
		kana TEXT PRIMARY KEY,
		introduced_at INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("store: create introduced_words table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO introduced_words (kana, introduced_at)
		SELECT kana, ? FROM word_stats
		UNION
		SELECT substr(item, 6), ? FROM memory WHERE substr(item, 1, 5) = 'word:'
	`, now.Unix(), now.Unix()); err != nil {
		return fmt.Errorf("store: seed introduced words: %w", err)
	}
	return tx.Commit()
}

// ensureMemory creates the memory table if it is missing and, in the same
```

- [ ] **Step 4: Run**

Run: `go test -count=1 ./store/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 5: Commit**

```bash
git add store/store.go store/store_test.go
git commit -m "feat(store): add introduced words and word package settings

The introduced_words table is created once and, in the same transaction,
seeded with every reading that has word stats or a word memory state, so
words learned before packages keep falling. New settings: package_size,
word_source, kana_font and newest_package."
```

---

### Task 4: Only introduced words fall

**Files:** Modify `fyne/game.go`, `fyne/wordgame.go`, `fyne/words.go`, `fyne/stats.go`, `store/store.go`, `vocab/index.go`, and the tests `fyne/game_test.go`, `fyne/wordgame_test.go`, `fyne/words_test.go`, `fyne/stats_test.go`, `store/store_test.go`, `vocab/index_test.go`.

The stage 3 rule (bundled words up to the level, behind `bundled_words`, plus every private word) is replaced by introduced ∩ known. The word source is resolved and saved on the first start, and the status line gets the stage 4a texts. Packages themselves come in Task 5.

- [ ] **Step 1: Update and write the tests**

`fyne/game_test.go` (`newTestState`):

Replace:

```go
		wordMemory:       map[string]memory.State{},
		wordMemoryDirty:  map[string]bool{},
		wordLevel:        5,
		useBundled:       true,
		meaningLang:      "en",
	}
	gs.charSet = kanacore.AllKana()
	for _, id := range kanacore.DefaultRowIDs() {
```

with:

```go
		wordMemory:       map[string]memory.State{},
		wordMemoryDirty:  map[string]bool{},
		wordLevel:        5,
		meaningLang:      "en",
		introduced:       map[string]bool{},
		introducedDirty:  map[string]bool{},
		packageSize:      5,
		wordSource:       jlptSource,
	}
	gs.charSet = kanacore.AllKana()
	for _, id := range kanacore.DefaultRowIDs() {
```

`fyne/wordgame_test.go` — `withWords` introduces its words; the level-gate and `bundled_words` tests are replaced:

Replace:

```go
	return vocab.Word{Kana: kana, JLPT: level, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {en}}}
}

// withWords replaces the word index with the given bundled words.
func withWords(gs *GameState, words ...vocab.Word) {
	gs.words = vocab.NewIndex(words, nil)
	gs.invalidatePlayable()
}
```

with:

```go
	return vocab.Word{Kana: kana, JLPT: level, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {en}}}
}

// withWords replaces the word index with the given bundled words, all of
// them introduced.
func withWords(gs *GameState, words ...vocab.Word) {
	gs.words = vocab.NewIndex(words, nil)
	gs.introduced = map[string]bool{}
	for _, w := range words {
		gs.introduced[w.Kana] = true
	}
	gs.invalidatePlayable()
}
```

Replace:

```go
	}
}

func TestPlayableWordsFollowLevelAndKnownKana(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("あお", 4, "blue"), bw("いえ", 5, "house"))
	secure(gs, "あ", "い", "お")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable at N5 = %v, want [あい] (あお is N4, いえ needs え)", got)
	}
	gs.wordLevel = 4
	gs.invalidatePlayable()
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("playable at N4 = %v", got)
	}
}

func TestPrivateListWordsIgnoreLevel(t *testing.T) {
	gs := newTestState()
	gs.words = vocab.NewIndex(nil, []vocab.List{{Name: "mine", Words: []vocab.Word{{Kana: "あお", Source: "mine", Meanings: vocab.Meanings{}}}}})
	secure(gs, "あ", "お")
	if got := playableKana(gs); !equalIDs(got, []string{"あお"}) {
		t.Fatalf("playable = %v, want [あお]", got)
	}
}

func TestNoWordIndexMeansNoWords(t *testing.T) {
	gs := newTestState()
	if got := gs.playableWords(); got != nil {
```

with:

```go
	}
}

func TestPlayableWordsAreIntroducedAndKnown(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("あお", 4, "blue"), bw("いえ", 5, "house"), bw("うえ", 5, "above"))
	delete(gs.introduced, "うえ")
	secure(gs, "あ", "い", "う", "お")
	gs.overallStats["え"] = store.KanaStats{Char: "え", CorrectCount: 2}
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("playable = %v, want [あい あお]: いえ needs え, うえ is not introduced, the level does not gate", got)
	}
}

func TestIntroducedWordsKeepFallingAfterLevelOrSourceChange(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("あお", 3, "blue"))
	secure(gs, "あ", "い", "お")
	gs.wordLevel = 2
	gs.wordSource = listSourcePrefix + "mine"
	gs.invalidatePlayable()
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("playable after the source and level changed = %v", got)
	}
}

func TestPrivateListWordsNeedIntroduction(t *testing.T) {
	gs := newTestState()
	gs.words = vocab.NewIndex(nil, []vocab.List{{Name: "mine", Words: []vocab.Word{{Kana: "あお", Source: "mine", Meanings: vocab.Meanings{}}}}})
	secure(gs, "あ", "お")
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable before the intro = %v", got)
	}
	gs.introduced["あお"] = true
	gs.invalidatePlayable()
	if got := playableKana(gs); !equalIDs(got, []string{"あお"}) {
		t.Fatalf("playable = %v, want [あお]", got)
	}
}

func TestIntroducedLoadFailureMeansNoWords(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	secure(gs, "あ", "い")
	gs.introducedLoadFailed = true
	gs.invalidatePlayable()
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable = %v, want none while introduced words are unknown", got)
	}
	if got := formatWordsLine(gs.wordsStatus()); got != "Words unavailable: could not load progress" {
		t.Fatalf("line = %q", got)
	}
}

func TestNewGameStateLoadsWordPackageState(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveIntroduced("あい", time.Now())
	_ = st.SaveNewestPackage([]string{"あい"})
	_ = st.SavePackageSize(10)
	gs := NewGameState(st)
	if !gs.introduced["あい"] || len(gs.introduced) != 1 || gs.introducedLoadFailed {
		t.Fatalf("introduced = %v, failed %v", gs.introduced, gs.introducedLoadFailed)
	}
	if !equalIDs(gs.newestPackage, []string{"あい"}) || gs.packageSize != 10 {
		t.Fatalf("newestPackage = %v, packageSize = %d", gs.newestPackage, gs.packageSize)
	}
}

func TestIntroducedLoadFailureIsDetected(t *testing.T) {
	test.NewApp()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	_ = st.SaveSelectedRows([]string{"vowels"})
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("DROP TABLE introduced_words"); err != nil {
		t.Fatal(err)
	}
	if gs := NewGameState(st); !gs.introducedLoadFailed {
		t.Fatal("an unreadable introduced_words table must set introducedLoadFailed")
	}
}

func TestWordSourceResolvedAndSavedOnFirstStart(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if gs := NewGameState(st); gs.wordSource != jlptSource {
		t.Fatalf("no lists: source = %q, want jlpt", gs.wordSource)
	}
	if s, _ := st.WordSource(); s != jlptSource {
		t.Fatalf("saved source = %q, want jlpt", s)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"b-list.csv", "a-list.csv"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("kana,meaning\nあお,blue\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if gs := NewGameState(st); gs.wordSource != jlptSource {
		t.Fatalf("a list added later switched the source to %q", gs.wordSource)
	}

	fresh := openTestStore(t)
	_ = fresh.SaveSelectedRows([]string{"vowels"})
	if gs := NewGameState(fresh); gs.wordSource != "list:a-list" {
		t.Fatalf("first start with lists: source = %q, want the first list by name", gs.wordSource)
	}
	if s, _ := fresh.WordSource(); s != "list:a-list" {
		t.Fatalf("saved source = %q", s)
	}
}

func TestWordsStatusLine(t *testing.T) {
	gs := newTestState()
	gs.words = vocab.NewIndex(
		[]vocab.Word{bw("あい", 5, "love"), bw("あお", 5, "blue"), bw("かお", 5, "face"), bw("いえ", 4, "house")},
		[]vocab.List{{Name: "minna-no-nihongo", Words: []vocab.Word{{Kana: "あい", Source: "minna-no-nihongo", Meanings: vocab.Meanings{}}}}},
	)
	gs.listNames = []string{"minna-no-nihongo"}
	secure(gs, "あ", "い", "お")
	gs.introduced = map[string]bool{"あい": true, "gone": true}
	gs.wordOverallStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 3}
	line := func() string { return formatWordsLine(gs.wordsStatus()) }

	if got := line(); got != "Words: 1 introduced · 1 secure · JLPT N5" {
		t.Fatalf("counts = %q (a reading missing from the index does not count)", got)
	}
	gs.introduced["あお"] = true
	if got := line(); got != "Words: learn more kana for new words" {
		t.Fatalf("only かお left, not playable: %q", got)
	}
	gs.introduced["かお"] = true
	if got := line(); got != "Words: JLPT N5 finished" {
		t.Fatalf("N5 introduced: %q", got)
	}
	gs.wordLevel = 1
	gs.introduced["いえ"] = true
	if got := line(); got != "Words: JLPT finished" {
		t.Fatalf("every level introduced: %q", got)
	}
	gs.wordSource = listSourcePrefix + "minna-no-nihongo"
	if got := line(); got != "Words: minna no nihongo finished" {
		t.Fatalf("list introduced: %q", got)
	}
	gs.wordSource = listSourcePrefix + "gone"
	gs.wordsDir = "/data/kana/words"
	if got := line(); got != "Words: add a list to /data/kana/words or choose JLPT" {
		t.Fatalf("missing list: %q", got)
	}
}

func TestNoWordIndexMeansNoWords(t *testing.T) {
	gs := newTestState()
	if got := gs.playableWords(); got != nil {
```

Replace:

```go
	gs.wordOverallStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 2}
	gs.wordSessionStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 1}
	snap := gs.snapshot()
	if snap.Words != (WordsStatus{Bundled: true, Playable: 2, Secure: 1, Level: 5}) {
		t.Fatalf("Words = %+v, want 2 playable, 1 secure, N5", snap.Words)
	}
	if snap.WordCorrect != 1 {
		t.Fatalf("WordCorrect = %d, want the session's 1", snap.WordCorrect)
```

with:

```go
	gs.wordOverallStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 2}
	gs.wordSessionStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 1}
	snap := gs.snapshot()
	if snap.Words != (WordsStatus{Introduced: 3, Secure: 1, Source: "JLPT N5", Level: 5, Next: wordsFinished}) {
		t.Fatalf("Words = %+v, want 3 introduced, 1 secure, JLPT N5 finished", snap.Words)
	}
	if snap.WordCorrect != 1 {
		t.Fatalf("WordCorrect = %d, want the session's 1", snap.WordCorrect)
```

Replace:

```go
	t.Cleanup(func() { bundledWords = orig })
	gs := NewGameState(nil)
	gs.mu.Lock()
	gs.useBundled = true
	snap := gs.snapshot()
	gs.mu.Unlock()
	if got := formatWordsLine(snap.Words); got != "Words unavailable: broken data" {
```

with:

```go
	t.Cleanup(func() { bundledWords = orig })
	gs := NewGameState(nil)
	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	if got := formatWordsLine(snap.Words); got != "Words unavailable: broken data" {
```

Replace:

```go
	t.Setenv("XDG_DATA_HOME", data)
	return filepath.Join(data, "kana", "words")
}

func TestBundledWordsSettingIsLoaded(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if NewGameState(st).useBundled {
		t.Fatal("bundled words must be off by default")
	}
	_ = st.SaveBundledWords(true)
	if !NewGameState(st).useBundled {
		t.Fatal("stored bundled_words=1 not loaded")
	}
}

func TestBundledOffPlaysOnlyPrivateWords(t *testing.T) {
	gs := newTestState()
	gs.useBundled = false
	gs.words = vocab.NewIndex(
		[]vocab.Word{bw("あい", 5, "love"), bw("あお", 4, "blue")},
		[]vocab.List{{Name: "mine", Words: []vocab.Word{
			{Kana: "あお", Source: "mine", Meanings: vocab.Meanings{}},
			{Kana: "いえ", Source: "mine", Meanings: vocab.Meanings{}},
		}}},
	)
	secure(gs, "あ", "い", "う", "え", "お")
	// あい is bundled only; あお is N4 but also private, so the level gate
	// does not apply to it.
	if got := playableKana(gs); !equalIDs(got, []string{"あお", "いえ"}) {
		t.Fatalf("playable with bundled off = %v, want [あお いえ]", got)
	}
	gs.useBundled = true
	gs.invalidatePlayable()
	if got := playableKana(gs); !equalIDs(got, []string{"あい", "あお", "いえ"}) {
		t.Fatalf("playable with bundled on = %v", got)
	}
}

func TestWordsLineWithoutBundledWords(t *testing.T) {
	test.NewApp()
	dir := isolateWordsDir(t)
	home, _ := os.UserHomeDir()

	gs := NewGameState(nil)
	gs.mu.Lock()
	ws := gs.wordsStatus()
	gs.mu.Unlock()
	if want := "Words: add a list to " + displayDir(dir, home); formatWordsLine(ws) != want {
		t.Fatalf("no lists: line = %q, want %q", formatWordsLine(ws), want)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mine.csv"), []byte("kana,meaning\nあお,blue\nねこ,cat\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gs = NewGameState(nil)
	gs.mu.Lock()
	secure(gs, "あ", "お")
	gs.wordOverallStats["あお"] = store.WordStats{Kana: "あお", CorrectCount: 3}
	ws = gs.wordsStatus()
	gs.mu.Unlock()
	if got := formatWordsLine(ws); got != "Words: 1 playable · 1 secure · own lists" {
		t.Fatalf("own lists: line = %q", got)
	}
}
```

with:

```go
	t.Setenv("XDG_DATA_HOME", data)
	return filepath.Join(data, "kana", "words")
}
```

`fyne/words_test.go`:

Replace:

```go
		ws   WordsStatus
		want string
	}{
		{"bundled", WordsStatus{Bundled: true, Playable: 42, Secure: 17, Level: 5}, "Words: 42 playable · 17 secure · up to N5"},
		{"bundled error", WordsStatus{Bundled: true, Err: "broken data", Playable: 3, Level: 5}, "Words unavailable: broken data"},
		{"own lists", WordsStatus{HasLists: true, Playable: 23, Secure: 5, Level: 5}, "Words: 23 playable · 5 secure · own lists"},
		{"own lists, bundled error", WordsStatus{HasLists: true, Err: "broken data", Playable: 23, Secure: 5}, "Words: 23 playable · 5 secure · own lists"},
		{"no lists", WordsStatus{Dir: "~/.local/share/kana/words"}, "Words: add a list to ~/.local/share/kana/words"},
		{"no lists, no folder", WordsStatus{}, "Words: add a word list"},
	}
	for _, c := range cases {
		if got := formatWordsLine(c.ws); got != c.want {
			t.Errorf("%s: line = %q, want %q", c.name, got, c.want)
		}
	}
	long := formatWordsLine(WordsStatus{Bundled: true, Err: strings.Repeat("x", 500)})
	if r := []rune(long); len(r) != maxWordsLineRunes || !strings.HasSuffix(long, "…") {
		t.Errorf("long error = %q (%d runes), want the %d-rune safety cap", long, len(r), maxWordsLineRunes)
	}
}

func TestDisplayDir(t *testing.T) {
	p := filepath.FromSlash
	home := p("/home/me")
```

with:

```go
		ws   WordsStatus
		want string
	}{
		{"JLPT counts", WordsStatus{Introduced: 25, Secure: 18, Source: "JLPT N5", Level: 5}, "Words: 25 introduced · 18 secure · JLPT N5"},
		{"list counts", WordsStatus{Introduced: 3, Secure: 1, Source: "minna no nihongo"}, "Words: 3 introduced · 1 secure · minna no nihongo"},
		{"load failed", WordsStatus{LoadFailed: true, Err: "broken data", Introduced: 3}, "Words unavailable: could not load progress"},
		{"bundled error", WordsStatus{Err: "broken data", Introduced: 3, Level: 5}, "Words unavailable: broken data"},
		{"missing list", WordsStatus{Missing: true, Dir: "~/.local/share/kana/words", Source: "minna"}, "Words: add a list to ~/.local/share/kana/words or choose JLPT"},
		{"missing list, no folder", WordsStatus{Missing: true, Source: "minna"}, "Words: add a word list or choose JLPT"},
		{"needs kana", WordsStatus{Next: wordsNeedKana, Introduced: 5, Source: "JLPT N5", Level: 5}, "Words: learn more kana for new words"},
		{"list finished", WordsStatus{Next: wordsFinished, Introduced: 40, Source: "minna no nihongo"}, "Words: minna no nihongo finished"},
		{"level finished", WordsStatus{Next: wordsFinished, Source: "JLPT N5", Level: 5}, "Words: JLPT N5 finished"},
		{"JLPT finished", WordsStatus{Next: wordsFinished, Source: "JLPT N1", Level: 1}, "Words: JLPT finished"},
	}
	for _, c := range cases {
		if got := formatWordsLine(c.ws); got != c.want {
			t.Errorf("%s: line = %q, want %q", c.name, got, c.want)
		}
	}
	long := formatWordsLine(WordsStatus{Err: strings.Repeat("x", 500)})
	if r := []rune(long); len(r) != maxWordsLineRunes || !strings.HasSuffix(long, "…") {
		t.Errorf("long error = %q (%d runes), want the %d-rune safety cap", long, len(r), maxWordsLineRunes)
	}
}

func TestResolveWordSource(t *testing.T) {
	cases := []struct {
		stored string
		lists  []string
		want   string
	}{
		{"", nil, "jlpt"},
		{"", []string{"a-list", "b-list"}, "list:a-list"},
		{"jlpt", []string{"a-list"}, "jlpt"},
		{"list:gone", nil, "list:gone"}, // a missing list stays selected
		{"list:", []string{"a"}, "list:a"},
		{"bogus", nil, "jlpt"},
	}
	for _, c := range cases {
		if got := resolveWordSource(c.stored, c.lists); got != c.want {
			t.Errorf("resolveWordSource(%q, %v) = %q, want %q", c.stored, c.lists, got, c.want)
		}
	}
}

func TestPrettyListName(t *testing.T) {
	if got := prettyListName("minna-no_nihongo"); got != "minna no nihongo" {
		t.Fatalf("prettyListName = %q", got)
	}
}

func TestDisplayDir(t *testing.T) {
	p := filepath.FromSlash
	home := p("/home/me")
```

`fyne/stats_test.go`:

Replace:

```go
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	snap.Words = WordsStatus{Bundled: true, Playable: 42, Secure: 17, Level: 5}
	p.Update(snap)
	if p.wordsText.Text != "Words: 42 playable · 17 secure · up to N5" {
		t.Fatalf("words line = %q", p.wordsText.Text)
	}
	var lines []string
```

with:

```go
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	snap.Words = WordsStatus{Introduced: 42, Secure: 17, Source: "JLPT N5", Level: 5}
	p.Update(snap)
	if p.wordsText.Text != "Words: 42 introduced · 17 secure · JLPT N5" {
		t.Fatalf("words line = %q", p.wordsText.Text)
	}
	var lines []string
```

Replace:

```go
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	snap.Words = WordsStatus{Bundled: true, Err: strings.Repeat("W", 48)}
	p.Update(snap)
	if w := p.wordsText.MinSize().Width; w > statsTextW {
		t.Fatalf("words line is %v px wide, want at most %v px", w, statsTextW)
```

with:

```go
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
	snap.Words = WordsStatus{Err: strings.Repeat("W", 48), Level: 5}
	p.Update(snap)
	if w := p.wordsText.MinSize().Width; w > statsTextW {
		t.Fatalf("words line is %v px wide, want at most %v px", w, statsTextW)
```

Replace:

```go
		t.Fatalf("words line = %q, want the start kept and an ellipsis", p.wordsText.Text)
	}

	snap.Words = WordsStatus{Bundled: true, Playable: 42, Secure: 17, Level: 5}
	p.Update(snap)
	if p.wordsText.TextSize != statsWordsTextSize || p.wordsText.Text != "Words: 42 playable · 17 secure · up to N5" {
		t.Fatalf("short line = %q at %v pt, want it whole at %v pt", p.wordsText.Text, p.wordsText.TextSize, statsWordsTextSize)
	}

	snap.Words = WordsStatus{Dir: "~/" + strings.Repeat("folder/", 12) + "words"}
	p.Update(snap)
	if w := p.wordsText.MinSize().Width; w > statsTextW {
		t.Fatalf("add-a-list line is %v px wide, want at most %v px", w, statsTextW)
```

with:

```go
		t.Fatalf("words line = %q, want the start kept and an ellipsis", p.wordsText.Text)
	}

	snap.Words = WordsStatus{Introduced: 42, Secure: 17, Source: "JLPT N5", Level: 5}
	p.Update(snap)
	if p.wordsText.TextSize != statsWordsTextSize || p.wordsText.Text != "Words: 42 introduced · 17 secure · JLPT N5" {
		t.Fatalf("short line = %q at %v pt, want it whole at %v pt", p.wordsText.Text, p.wordsText.TextSize, statsWordsTextSize)
	}

	snap.Words = WordsStatus{Missing: true, Dir: "~/" + strings.Repeat("folder/", 12) + "words"}
	p.Update(snap)
	if w := p.wordsText.MinSize().Width; w > statsTextW {
		t.Fatalf("add-a-list line is %v px wide, want at most %v px", w, statsTextW)
```

`store/store_test.go` — remove `TestBundledWordsRoundTrip`:

Replace:

```go
	}
}

func TestBundledWordsRoundTrip(t *testing.T) {
	st := openTemp(t)
	if on, err := st.BundledWords(); err != nil || on {
		t.Fatalf("unset BundledWords = %v, %v; want false", on, err)
	}
	if err := st.SaveBundledWords(true); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.BundledWords(); !on {
		t.Fatal("BundledWords = false after saving true")
	}
	if err := st.SaveBundledWords(false); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.BundledWords(); on {
		t.Fatal("BundledWords = true after saving false")
	}
}

func TestIntroducedWordsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.IntroducedWords()
```

with:

```go
	}
}

func TestIntroducedWordsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.IntroducedWords()
```

`vocab/index_test.go` — remove the `NoBundled` cases:

Replace:

```go
		{"minna lesson 1", Filter{MaxLevel: 5, List: "minna", Lesson: "1"}, all, []string{"せんせい"}},
		{"unknown list", Filter{MaxLevel: 5, List: "nope"}, all, nil},
		{"known units", Filter{MaxLevel: 5}, func(u string) bool { return u == "ね" || u == "こ" }, []string{"ねこ"}},
		// かんきょう and はし are bundled and in minna, so they stay playable
		// without bundled words, at any level.
		{"mixed without bundled", Filter{MaxLevel: 5, NoBundled: true}, all, []string{"かんきょう", "せんせい", "はし"}},
		{"jmdict without bundled", Filter{MaxLevel: 5, List: BundledSource, NoBundled: true}, all, nil},
		{"minna without bundled", Filter{MaxLevel: 5, List: "minna", NoBundled: true}, all, []string{"かんきょう", "せんせい", "はし"}},
	}
	for _, c := range cases {
		got := kanaOf(Playable(ix.Entries(), c.known, c.filter))
```

with:

```go
		{"minna lesson 1", Filter{MaxLevel: 5, List: "minna", Lesson: "1"}, all, []string{"せんせい"}},
		{"unknown list", Filter{MaxLevel: 5, List: "nope"}, all, nil},
		{"known units", Filter{MaxLevel: 5}, func(u string) bool { return u == "ね" || u == "こ" }, []string{"ねこ"}},
	}
	for _, c := range cases {
		got := kanaOf(Playable(ix.Entries(), c.known, c.filter))
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test -count=1 ./fyne/ ./store/ ./vocab/`
Expected: FAIL — `unknown field introduced in struct literal of type GameState`, `undefined: jlptSource`, `unknown field Introduced in struct literal of type WordsStatus`, `unknown field Missing …`.

- [ ] **Step 3: Remove `bundled_words` and `Filter.NoBundled`**

`store/store.go`:

Replace:

```go
	pointsFactorKey    = "points_factor"
	wordLevelKey       = "word_level"
	meaningLangKey     = "meaning_lang"
	bundledWordsKey    = "bundled_words"
	packageSizeKey     = "package_size"
	wordSourceKey      = "word_source"
	kanaFontKey        = "kana_font"
```

with:

```go
	pointsFactorKey    = "points_factor"
	wordLevelKey       = "word_level"
	meaningLangKey     = "meaning_lang"
	packageSizeKey     = "package_size"
	wordSourceKey      = "word_source"
	kanaFontKey        = "kana_font"
```

Replace:

```go
	return s.setSetting(autoProgressKey, "0")
}

// BundledWords reports whether the bundled word list is played in mixed
// mode. It is off unless saved as on.
func (s *Store) BundledWords() (bool, error) {
	value, err := s.getSetting(bundledWordsKey)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == "1", nil
}

// SaveBundledWords turns the bundled word list on or off.
func (s *Store) SaveBundledWords(enabled bool) error {
	if enabled {
		return s.setSetting(bundledWordsKey, "1")
	}
	return s.setSetting(bundledWordsKey, "0")
}

// ScoreLimit returns the configured score threshold for ending a session.
func (s *Store) ScoreLimit() (int, error) {
	value, err := s.getSetting(scoreLimitKey)
```

with:

```go
	return s.setSetting(autoProgressKey, "0")
}

// ScoreLimit returns the configured score threshold for ending a session.
func (s *Store) ScoreLimit() (int, error) {
	value, err := s.getSetting(scoreLimitKey)
```

`vocab/index.go`:

Replace:

```go
	List     string // "" = bundled up to MaxLevel plus every private list;
	// BundledSource = bundled only, up to MaxLevel; else only this private list
	Lesson string // with a private List: only this lesson ("" = all)
	// NoBundled drops entries that come only from the bundled list. An entry
	// that is also in a private list stays, without the level gate.
	NoBundled bool
}

// Playable returns the entries whose units all satisfy known and which pass
```

with:

```go
	List     string // "" = bundled up to MaxLevel plus every private list;
	// BundledSource = bundled only, up to MaxLevel; else only this private list
	Lesson string // with a private List: only this lesson ("" = all)
}

// Playable returns the entries whose units all satisfy known and which pass
```

Replace:

```go
}

func (f Filter) passes(e Entry) bool {
	bundledOK := !f.NoBundled && e.Bundled && e.JLPT >= f.MaxLevel
	switch f.List {
	case "":
		return bundledOK || len(e.Lists) > 0
```

with:

```go
}

func (f Filter) passes(e Entry) bool {
	bundledOK := e.Bundled && e.JLPT >= f.MaxLevel
	switch f.List {
	case "":
		return bundledOK || len(e.Lists) > 0
```

- [ ] **Step 4: Sources and the status line in `fyne/words.go`**

Replace:

```go
// status panel. Stage 4 moves it to the settings' Sources tab.
var wordDataSource = []string{"Word data: JMdict (EDRDG, CC BY-SA 4.0),", "JLPT levels by J. Waller"}

// WordsStatus is the status panel's words line.
type WordsStatus struct {
	Err      string // the bundled list failed to load
	Playable int    // words playable now
	Secure   int    // playable words with 3 or more correct answers
	Level    int    // words are played up to this JLPT level
	Bundled  bool   // bundled words are played
	HasLists bool   // at least one private list was found
	Dir      string // the private list folder for display; "" if unknown
}

// formatWordsLine is the status panel's words line. With bundled words off,
// a bundled load error does not matter and is not shown.
func formatWordsLine(ws WordsStatus) string {
	switch {
	case ws.Bundled && ws.Err != "":
		return truncateRunes("Words unavailable: "+ws.Err, maxWordsLineRunes)
	case ws.Bundled:
		return fmt.Sprintf("Words: %d playable · %d secure · up to N%d", ws.Playable, ws.Secure, ws.Level)
	case ws.HasLists:
		return fmt.Sprintf("Words: %d playable · %d secure · own lists", ws.Playable, ws.Secure)
	case ws.Dir != "":
		return truncateRunes("Words: add a list to "+ws.Dir, maxWordsLineRunes)
	default:
		return "Words: add a word list"
	}
}

// displayDir shortens a folder inside home to "~/…" for display.
```

with:

```go
// status panel. Stage 4 moves it to the settings' Sources tab.
var wordDataSource = []string{"Word data: JMdict (EDRDG, CC BY-SA 4.0),", "JLPT levels by J. Waller"}

// Word sources as stored in the word_source setting.
const (
	jlptSource       = "jlpt"  // the bundled JLPT words
	listSourcePrefix = "list:" // followed by a private list's name
)

// resolveWordSource returns the stored word source, or, when none (or no
// valid one) is stored, the first private list by name, else JLPT.
func resolveWordSource(stored string, lists []string) string {
	if stored == jlptSource || (strings.HasPrefix(stored, listSourcePrefix) && stored != listSourcePrefix) {
		return stored
	}
	if len(lists) > 0 {
		return listSourcePrefix + lists[0]
	}
	return jlptSource
}

// prettyListName shows a list name with spaces: "minna-no-nihongo" →
// "minna no nihongo".
func prettyListName(name string) string {
	return strings.NewReplacer("-", " ", "_", " ").Replace(name)
}

// sourceLabel names a source for display: "JLPT N5" or a pretty list name.
func sourceLabel(src vocab.Source) string {
	if src.List != "" {
		return prettyListName(src.List)
	}
	return fmt.Sprintf("JLPT N%d", src.Level)
}

// wordsNext says whether new words can be introduced from the source.
type wordsNext int

const (
	wordsMore     wordsNext = iota // the source has a word that can be introduced now
	wordsNeedKana                  // words are left, but none is playable yet
	wordsFinished                  // every word of the source is introduced
)

// WordsStatus is the status panel's words line.
type WordsStatus struct {
	LoadFailed bool      // the introduced words could not be loaded
	Err        string    // the bundled list failed to load (JLPT source only)
	Missing    bool      // the chosen list was not found
	Dir        string    // the private list folder for display; "" if unknown
	Source     string    // "JLPT N5" or a pretty list name
	Level      int       // the JLPT level of a JLPT source; 0 for a list
	Introduced int       // introduced readings in the word index
	Secure     int       // introduced readings with 3 or more correct answers
	Next       wordsNext // whether new words can come from the source
}

// formatWordsLine is the status panel's words line: counts, or a hint when
// nothing can be introduced.
func formatWordsLine(ws WordsStatus) string {
	var s string
	switch {
	case ws.LoadFailed:
		s = "Words unavailable: could not load progress"
	case ws.Err != "":
		s = "Words unavailable: " + ws.Err
	case ws.Missing && ws.Dir != "":
		s = "Words: add a list to " + ws.Dir + " or choose JLPT"
	case ws.Missing:
		s = "Words: add a word list or choose JLPT"
	case ws.Next == wordsNeedKana:
		s = "Words: learn more kana for new words"
	case ws.Next == wordsFinished && ws.Level == 1:
		s = "Words: JLPT finished"
	case ws.Next == wordsFinished:
		s = "Words: " + ws.Source + " finished"
	default:
		s = fmt.Sprintf("Words: %d introduced · %d secure · %s", ws.Introduced, ws.Secure, ws.Source)
	}
	return truncateRunes(s, maxWordsLineRunes)
}

// displayDir shortens a folder inside home to "~/…" for display.
```

- [ ] **Step 5: The playable rule, the source and the status in `fyne/wordgame.go`**

Replace:

```go

import (
	"os"

	"kana/kanacore"
	"kana/memory"
```

with:

```go

import (
	"os"
	"sort"
	"strings"

	"kana/kanacore"
	"kana/memory"
```

Replace:

```go
		gs.wordsDir = dir
		lists, _ = vocab.LoadDir(dir)
	}
	gs.hasLists = len(lists) > 0
	gs.words = base.WithLists(lists)
	gs.invalidatePlayable()
}
```

with:

```go
		gs.wordsDir = dir
		lists, _ = vocab.LoadDir(dir)
	}
	gs.listNames = nil
	for _, l := range lists {
		gs.listNames = append(gs.listNames, l.Name)
	}
	gs.words = base.WithLists(lists)
	gs.invalidatePlayable()
}
```

Replace:

```go
	return gs.overallStats[unit].CorrectCount+gs.sessionStats[unit].CorrectCount >= secureCorrect
}

// playableWords returns the entries playable in mixed mode: bundled words up
// to wordLevel (only with useBundled) plus every private word, all of whose
// units are known. The
// result is cached until invalidatePlayable; do not modify it. Must be
// called under lock.
func (gs *GameState) playableWords() []vocab.Entry {
	if gs.words == nil {
		return nil
	}
	if !gs.playableOK {
		gs.playable = vocab.Playable(gs.words.Entries(), gs.unitKnown, vocab.Filter{MaxLevel: gs.wordLevel, NoBundled: !gs.useBundled})
		gs.playableOK = true
	}
	return gs.playable
}

// invalidatePlayable marks the playable cache stale. Call it whenever a kana
// may have become secure, the active scripts, the level or the index
// changed. Must be called under lock.
func (gs *GameState) invalidatePlayable() {
	gs.playableOK = false
}
```

with:

```go
	return gs.overallStats[unit].CorrectCount+gs.sessionStats[unit].CorrectCount >= secureCorrect
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

// playableWords returns the entries that may fall: introduced words all of
// whose units are known, from every source and level, sorted by reading.
// None while the introduced words could not be loaded. The result is
// cached until invalidatePlayable; do not modify it. Must be called under
// lock.
func (gs *GameState) playableWords() []vocab.Entry {
	if gs.words == nil || gs.introducedLoadFailed {
		return nil
	}
	if !gs.playableOK {
		var out []vocab.Entry
		for kana := range gs.introduced {
			if e, ok := gs.words.Lookup(kana); ok && gs.unitsKnown(e.Units) {
				out = append(out, e)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Kana < out[j].Kana })
		gs.playable = out
		gs.playableOK = true
	}
	return gs.playable
}

// invalidatePlayable marks the playable cache stale. Call it whenever a kana
// may have become secure, or the active scripts, the introduced words or
// the index changed. Must be called under lock.
func (gs *GameState) invalidatePlayable() {
	gs.playableOK = false
}
```

Replace:

```go
	return gs.words.Lookup(kana)
}

// wordsStatus summarises the words for the status panel. Must be called
// under lock.
func (gs *GameState) wordsStatus() WordsStatus {
	ws := WordsStatus{Err: gs.wordsErr, Level: gs.wordLevel, Bundled: gs.useBundled, HasLists: gs.hasLists}
	if gs.wordsDir != "" {
		home, _ := os.UserHomeDir()
		ws.Dir = displayDir(gs.wordsDir, home)
	}
	for _, e := range gs.playableWords() {
		ws.Playable++
		if gs.wordCorrect(e.Kana) >= secureCorrect {
			ws.Secure++
		}
	}
	return ws
}
```

with:

```go
	return gs.words.Lookup(kana)
}

// source returns where new words come from and whether that source exists:
// a chosen list that was not found does not. Must be called under lock.
func (gs *GameState) source() (vocab.Source, bool) {
	if name, ok := strings.CutPrefix(gs.wordSource, listSourcePrefix); ok {
		for _, l := range gs.listNames {
			if l == name {
				return vocab.Source{List: name}, true
			}
		}
		return vocab.Source{List: name}, false
	}
	return vocab.Source{Level: gs.wordLevel}, true
}

// sourceLeft counts the source's entries that are not introduced yet (for
// JLPT: bundled entries with JLPT >= Level). Must be called under lock.
func (gs *GameState) sourceLeft(src vocab.Source) int {
	if gs.words == nil {
		return 0
	}
	n := 0
	for _, e := range gs.words.Entries() {
		if src.Has(e) && !gs.introduced[e.Kana] {
			n++
		}
	}
	return n
}

// wordsStatus summarises the words for the status panel. Must be called
// under lock.
func (gs *GameState) wordsStatus() WordsStatus {
	src, found := gs.source()
	ws := WordsStatus{LoadFailed: gs.introducedLoadFailed, Source: sourceLabel(src), Missing: !found}
	if src.List == "" {
		ws.Err, ws.Level = gs.wordsErr, src.Level
	}
	if gs.wordsDir != "" {
		home, _ := os.UserHomeDir()
		ws.Dir = displayDir(gs.wordsDir, home)
	}
	if gs.words == nil {
		return ws
	}
	for kana := range gs.introduced {
		if _, ok := gs.words.Lookup(kana); !ok {
			continue
		}
		ws.Introduced++
		if gs.wordCorrect(kana) >= secureCorrect {
			ws.Secure++
		}
	}
	if found && !ws.LoadFailed && len(vocab.NextPackage(gs.words, src, gs.introduced, 1, gs.unitKnown)) == 0 {
		ws.Next = wordsFinished
		if gs.sourceLeft(src) > 0 {
			ws.Next = wordsNeedKana
		}
	}
	return ws
}
```

- [ ] **Step 6: Fields and loading in `fyne/game.go`**

Replace:

```go
	wordMemory           map[string]memory.State // spaced-repetition state per reading
	wordMemoryDirty      map[string]bool         // readings whose state is not saved yet
	wordMemoryLoadFailed bool                    // stored word states unreadable: never overwrite them
	wordLevel            int                     // words are played up to this JLPT level, 5 = N5
	meaningLang          string                  // language meanings are shown in
	words                *vocab.Index            // every word reading; nil = no words
	wordsErr             string                  // bundled list failed to load
	useBundled           bool                    // bundled words are played (setting bundled_words)
	hasLists             bool                    // at least one private list was found
	wordsDir             string                  // the private list folder; "" if unknown
	playable             []vocab.Entry           // cached playableWords result
	playableOK           bool                    // playable is up to date

	selectedRows  map[string]bool
	autoProgress  bool
	activeScripts map[kanacore.Script]bool
```

with:

```go
	wordMemory           map[string]memory.State // spaced-repetition state per reading
	wordMemoryDirty      map[string]bool         // readings whose state is not saved yet
	wordMemoryLoadFailed bool                    // stored word states unreadable: never overwrite them
	wordLevel            int                     // JLPT packages come from N5 down to this level
	meaningLang          string                  // language meanings are shown in
	words                *vocab.Index            // every word reading; nil = no words
	wordsErr             string                  // bundled list failed to load
	listNames            []string                // the private lists found, sorted by name
	wordsDir             string                  // the private list folder; "" if unknown
	playable             []vocab.Entry           // cached playableWords result
	playableOK           bool                    // playable is up to date

	// Word packages: only introduced words fall.
	introduced           map[string]bool // readings shown in a package intro
	introducedDirty      map[string]bool // introduced readings whose save failed; mergeSessionStats retries
	introducedLoadFailed bool            // introduced words unreadable: no words, no packages, no writes
	newestPackage        []string        // readings of the newest package, without "already known" ones
	packageSize          int             // words per package: 3, 5 or 10
	wordSource           string          // where new words come from: jlptSource or listSourcePrefix + name

	selectedRows  map[string]bool
	autoProgress  bool
	activeScripts map[kanacore.Script]bool
```

Replace:

```go
		wordMemory:       map[string]memory.State{},
		wordMemoryDirty:  map[string]bool{},
		wordLevel:        store.DefaultWordLevel,
	}

	for _, id := range kanacore.DefaultRowIDs() {
```

with:

```go
		wordMemory:       map[string]memory.State{},
		wordMemoryDirty:  map[string]bool{},
		wordLevel:        store.DefaultWordLevel,
		introduced:       map[string]bool{},
		introducedDirty:  map[string]bool{},
		packageSize:      store.DefaultPackageSize,
		wordSource:       jlptSource,
	}

	for _, id := range kanacore.DefaultRowIDs() {
```

Replace:

```go
	}

	storedLang := ""
	if st != nil {
		if names, err := st.ActiveScripts(); err == nil && len(names) > 0 {
			gs.activeScripts = scriptSet(parseScripts(names))
```

with:

```go
	}

	storedLang := ""
	storedSource, sourceLoaded := "", false
	if st != nil {
		if names, err := st.ActiveScripts(); err == nil && len(names) > 0 {
			gs.activeScripts = scriptSet(parseScripts(names))
```

Replace:

```go
		if level, err := st.WordLevel(); err == nil {
			gs.wordLevel = level
		}
		if on, err := st.BundledWords(); err == nil {
			gs.useBundled = on
		}
		if l, err := st.MeaningLang(); err == nil {
			storedLang = l
```

with:

```go
		if level, err := st.WordLevel(); err == nil {
			gs.wordLevel = level
		}
		if in, err := st.IntroducedWords(); err == nil {
			gs.introduced = in
		} else {
			gs.introducedLoadFailed = true
		}
		if pkg, err := st.NewestPackage(); err == nil {
			gs.newestPackage = pkg
		}
		if n, err := st.PackageSize(); err == nil {
			gs.packageSize = n
		}
		if src, err := st.WordSource(); err == nil {
			storedSource, sourceLoaded = src, true
		}
		if l, err := st.MeaningLang(); err == nil {
			storedLang = l
```

Replace:

```go

	gs.meaningLang = resolveMeaningLang(storedLang, systemLanguage())
	gs.loadWords()
	return gs
}
```

with:

```go

	gs.meaningLang = resolveMeaningLang(storedLang, systemLanguage())
	gs.loadWords()
	// The automatic source is resolved once and saved, so a list added
	// later does not switch the source silently. A source that could not
	// be read is never overwritten.
	gs.wordSource = resolveWordSource(storedSource, gs.listNames)
	if sourceLoaded && gs.wordSource != storedSource {
		_ = st.SaveWordSource(gs.wordSource)
	}
	return gs
}
```

`fyne/stats.go` (comment only):

Replace:

```go

	correctText, missedText, accuracyText *canvas.Text

	wordsText   *canvas.Text   // "Words: … playable · … secure · up to N5" (or "own lists")
	sourceTexts []*canvas.Text // word data attribution

	pathLines map[kanacore.Script]*fyne.Container
```

with:

```go

	correctText, missedText, accuracyText *canvas.Text

	wordsText   *canvas.Text   // "Words: … introduced · … secure · JLPT N5", or a hint
	sourceTexts []*canvas.Text // word data attribution

	pathLines map[kanacore.Script]*fyne.Container
```

- [ ] **Step 7: Run**

Run: `go test -count=1 ./fyne/ ./store/ ./vocab/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 8: Commit**

```bash
git add fyne/game.go fyne/wordgame.go fyne/words.go fyne/stats.go store/store.go vocab/index.go \
	fyne/game_test.go fyne/wordgame_test.go fyne/words_test.go fyne/stats_test.go store/store_test.go vocab/index_test.go
git commit -m "feat(fyne): play only introduced words

A word falls once it was introduced and its kana are known, from every
source and level; bundled_words and the stage 3 level gate are gone. The
word source is resolved once (first list, else JLPT) and saved, and the
status line shows introduced and secure words or what to do next."
```

---

### Task 5: Word package offers in the game state

**Files:** Modify `fyne/game.go`, `fyne/game_lifecycle_test.go`, `fyne/wordgame_test.go`. Create `fyne/packages.go`, `fyne/packages_test.go`.

This task adds the state only; the dialogs that show a pending package or level offer come in Task 6. Between the two commits a due package keeps the game paused with nothing on screen, so do not build a release from Task 5 alone.

- [ ] **Step 1: Write the failing tests** — create `fyne/packages_test.go`:

```go
package main

import (
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/memory"
	"kana/store"
	"kana/vocab"
)

// packageReady returns a game with secure vowels and n vowel words (N5),
// none of them introduced.
func packageReady(n int) *GameState {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.words = vocab.NewIndex(vowelWords(n), nil)
	secure(gs, "あ", "い", "う", "え", "お")
	return gs
}

// introduceFirst introduces the first n words of the JLPT order and makes
// them the newest package.
func introduceFirst(gs *GameState, n int) []string {
	pkg := readingsOf(vocab.NextPackage(gs.words, vocab.Source{Level: 5}, gs.introduced, n, gs.unitKnown))
	for _, kana := range pkg {
		gs.introduced[kana] = true
	}
	gs.newestPackage = pkg
	gs.invalidatePlayable()
	return pkg
}

// drainEvents empties the event channel and returns the kinds it held.
func drainEvents(gs *GameState) []gameEventType {
	var kinds []gameEventType
	for {
		select {
		case e := <-gs.eventCh:
			kinds = append(kinds, e.kind)
		default:
			return kinds
		}
	}
}

func TestFirstPackageWhenASourceWordIsPlayable(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.words = vocab.NewIndex(vowelWords(12), nil)
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 || gs.paused {
		t.Fatalf("no secure kana: pending %v, paused %v", gs.pendingPackage, gs.paused)
	}
	secure(gs, "あ", "い", "う", "え", "お")
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 5 || !gs.paused {
		t.Fatalf("pending %v, paused %v; want a package of 5 and a paused game", gs.pendingPackage, gs.paused)
	}
	if kinds := drainEvents(gs); len(kinds) != 1 || kinds[0] != wordPackageEvent {
		t.Fatalf("events = %v, want one wordPackageEvent", kinds)
	}
	gs.maybeOfferPackage()
	if kinds := drainEvents(gs); len(kinds) != 0 {
		t.Fatalf("a pending package must not be offered twice: %v", kinds)
	}
}

func TestNextPackageAt80PercentSecure(t *testing.T) {
	for size, need := range map[int]int{5: 4, 3: 3, 10: 8} {
		gs := packageReady(40)
		gs.packageSize = size
		pkg := introduceFirst(gs, size)
		for _, kana := range pkg[:need-1] {
			gs.wordOverallStats[kana] = store.WordStats{Kana: kana, CorrectCount: 3}
		}
		gs.maybeOfferPackage()
		if len(gs.pendingPackage) != 0 {
			t.Fatalf("size %d: offered with %d of %d secure", size, need-1, size)
		}
		gs.wordSessionStats[pkg[need-1]] = store.WordStats{Kana: pkg[need-1], CorrectCount: 3}
		gs.maybeOfferPackage()
		if len(gs.pendingPackage) != size {
			t.Fatalf("size %d: pending %v with %d secure, want the next %d words", size, gs.pendingPackage, need, size)
		}
	}
}

func TestPackageRuleCountsOnlyPlayableWords(t *testing.T) {
	gs := packageReady(10)
	gs.words = vocab.NewIndex(append(vowelWords(10), bw("アイ", 5, "eye"), bw("アオ", 5, "blue"), bw("イエ", 5, "house")), nil)
	gs.activeScripts[kanacore.ScriptKatakana] = true
	secure(gs, "ア", "イ", "エ", "オ")
	gs.newestPackage = []string{"アイ", "アオ", "イエ"}
	for _, kana := range gs.newestPackage {
		gs.introduced[kana] = true
	}
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 {
		t.Fatal("offered while no katakana word of the newest package is secure")
	}
	gs.activeScripts = map[kanacore.Script]bool{kanacore.ScriptHiragana: true}
	gs.invalidatePlayable()
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 5 {
		t.Fatalf("katakana off: no package word is playable, so the package counts as done; pending %v", gs.pendingPackage)
	}

	gs = packageReady(20) // the first five: ああ あい あう あえ あああ
	pkg := introduceFirst(gs, 5)
	gs.overallStats["え"] = store.KanaStats{Char: "え", CorrectCount: 0} // words with え stop being playable
	gs.invalidatePlayable()
	playable := 0
	for _, kana := range pkg {
		if e, _ := gs.words.Lookup(kana); gs.unitsKnown(e.Units) {
			playable++
			gs.wordOverallStats[kana] = store.WordStats{Kana: kana, CorrectCount: 3}
		}
	}
	if playable == 0 || playable == len(pkg) {
		t.Fatalf("precondition: some but not all package words need え (%d of %d playable)", playable, len(pkg))
	}
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) == 0 {
		t.Fatal("every playable word of the newest package is secure, so the next package is due")
	}
}

func TestNoPackageWhileOverMissingListOrLoadFailed(t *testing.T) {
	gs := packageReady(10)
	gs.over = true
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 {
		t.Fatal("no package after game over")
	}
	gs = packageReady(10)
	gs.wordSource = listSourcePrefix + "gone"
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 || gs.pendingLevelOffer {
		t.Fatal("no package or offer from a missing list")
	}
	gs = packageReady(10)
	gs.introducedLoadFailed = true
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 || gs.pendingLevelOffer {
		t.Fatal("no package or offer while the introduced words are unknown")
	}
}

func TestFullEventChannelLeavesNothingPending(t *testing.T) {
	gs := packageReady(10)
	for i := 0; i < cap(gs.eventCh); i++ {
		gs.eventCh <- gameEvent{kind: rowsUnlockedEvent}
	}
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 || gs.paused {
		t.Fatalf("pending %v, paused %v; want neither when the event cannot be sent", gs.pendingPackage, gs.paused)
	}
}

func TestStartSetsPendingPackageWithoutEvent(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	for _, c := range []string{"あ", "い", "う", "え", "お"} {
		_ = st.SaveKanaStats(c, 3, 0, 3)
	}
	gs := NewGameState(st)
	if len(gs.pendingPackage) != 5 || !gs.paused {
		t.Fatalf("pending %v, paused %v; want the first package of 5", gs.pendingPackage, gs.paused)
	}
	if n := len(gs.eventCh); n != 0 {
		t.Fatalf("%d events queued; the start path sets the package directly", n)
	}
}

func TestFinishPackage(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordStats("あお", 5, 2)
	gs := NewGameState(st)
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	gs.mu.Lock()
	gs.now = func() time.Time { return now }
	gs.words = vocab.NewIndex(vowelWords(30), nil)
	gs.wordOverallStats["あお"] = store.WordStats{Kana: "あお", CorrectCount: 5, MissCount: 2}
	gs.pendingPackage = []string{"あい", "あう", "あえ", "あお"}
	gs.paused = true
	gs.mu.Unlock()

	gs.FinishPackage([]string{"あう", "あお"})

	gs.mu.Lock()
	defer gs.mu.Unlock()
	for _, kana := range []string{"あい", "あう", "あえ", "あお"} {
		if !gs.introduced[kana] {
			t.Errorf("%s not introduced", kana)
		}
	}
	if !equalIDs(gs.newestPackage, []string{"あい", "あえ"}) {
		t.Fatalf("newestPackage = %v, want the words not ticked as known", gs.newestPackage)
	}
	if len(gs.pendingPackage) != 0 || gs.paused {
		t.Fatalf("pending %v, paused %v; want the game resumed", gs.pendingPackage, gs.paused)
	}
	if in, _ := st.IntroducedWords(); len(in) != 4 {
		t.Fatalf("stored introduced = %v, want all four at once", in)
	}
	if np, _ := st.NewestPackage(); !equalIDs(np, []string{"あい", "あえ"}) {
		t.Fatalf("stored newest_package = %v", np)
	}
	words, _ := st.WordStatistics()
	if w := words["あう"]; w.CorrectCount != 3 || w.MissCount != 0 {
		t.Fatalf("stored あう = %+v, want 3 correct", w)
	}
	if w := words["あお"]; w.CorrectCount != 5 || w.MissCount != 2 {
		t.Fatalf("stored あお = %+v, want its 5/2 kept (max(3, current))", w)
	}
	if gs.wordCorrect("あう") != 3 {
		t.Fatalf("wordCorrect(あう) = %d, want 3", gs.wordCorrect("あう"))
	}
	want := memory.Seed(3, now)
	if s, ok := gs.wordMemory["あう"]; !ok || math.Abs(s.Stability-want.Stability) > 1e-12 || !s.LastSeen.Equal(now) || !gs.wordMemoryDirty["あう"] {
		t.Fatalf("memory = %+v, %v; want Seed(3, now) and dirty", s, ok)
	}
	if len(gs.wordSessionStats) != 0 || gs.score != 0 || gs.answerStreak != 0 || gs.tempoStep != 0 {
		t.Fatal("already known words give no session hits, points or tempo")
	}
}

func TestFinishPackageWithEveryWordKnownOffersTheNext(t *testing.T) {
	gs := packageReady(20)
	gs.pendingPackage = readingsOf(vocab.NextPackage(gs.words, vocab.Source{Level: 5}, nil, 5, gs.unitKnown))
	first := append([]string(nil), gs.pendingPackage...)
	gs.FinishPackage(first)
	if len(gs.newestPackage) != 0 || len(gs.pendingPackage) != 5 || !gs.paused {
		t.Fatalf("newest %v, pending %v; want the next package at once", gs.newestPackage, gs.pendingPackage)
	}
}

func TestFailedIntroducedSaveIsRetriedByTheMerge(t *testing.T) {
	test.NewApp()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("ALTER TABLE introduced_words RENAME TO hidden"); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	gs.words = vocab.NewIndex(vowelWords(5), nil)
	gs.pendingPackage = []string{"あい"}
	gs.mu.Unlock()
	gs.FinishPackage(nil)
	gs.mu.Lock()
	dirty := gs.introducedDirty["あい"]
	gs.mu.Unlock()
	if !dirty {
		t.Fatal("a failed save must keep the reading in introducedDirty")
	}
	if _, err := db.Exec("ALTER TABLE hidden RENAME TO introduced_words"); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	gs.mergeSessionStats()
	left := len(gs.introducedDirty)
	gs.mu.Unlock()
	if in, _ := st.IntroducedWords(); !in["あい"] || left != 0 {
		t.Fatalf("after the merge: stored %v, still dirty %d", in, left)
	}
}

func TestReplacementWord(t *testing.T) {
	gs := packageReady(7)
	gs.introduced["あい"] = true
	if _, ok := gs.ReplacementWord(); ok {
		t.Fatal("no replacement without a pending package")
	}
	gs.pendingPackage = readingsOf(vocab.NextPackage(gs.words, vocab.Source{Level: 5}, gs.introduced, 5, gs.unitKnown))
	seen := map[string]bool{"あい": true}
	for _, kana := range gs.pendingPackage {
		seen[kana] = true
	}
	e, ok := gs.ReplacementWord()
	if !ok || seen[e.Kana] || len(gs.pendingPackage) != 6 || gs.pendingPackage[5] != e.Kana {
		t.Fatalf("replacement %q (%v), pending %v; want a new word appended", e.Kana, ok, gs.pendingPackage)
	}
	if _, ok := gs.ReplacementWord(); ok {
		t.Fatal("all seven words are introduced or pending: no replacement left")
	}
	if len(gs.pendingPackage) != 6 {
		t.Fatalf("pending %v changed without a replacement", gs.pendingPackage)
	}
}

func TestPendingPackageHoldsThePause(t *testing.T) {
	gs := packageReady(10)
	gs.maybeOfferPackage()
	if !gs.paused {
		t.Fatal("a pending package must pause the game")
	}
	gs.TogglePause()
	gs.TogglePause()
	gs.SetSettingsOpen(true)
	gs.SetSettingsOpen(false)
	gs.Resume()
	if !gs.paused {
		t.Fatal("the pause must hold while the package is pending")
	}
	gs.FinishPackage(nil)
	if gs.paused {
		t.Fatal("finishing the package must resume the game")
	}
}

func TestResetClearsPendingAndOffersAgain(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	for _, c := range []string{"あ", "い", "う", "え", "お"} {
		_ = st.SaveKanaStats(c, 3, 0, 3)
	}
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.introduced["かさ"] = true
	gs.newestPackage = []string{"かさ"}
	gs.pendingLevelOffer = true
	gs.levelOfferDeclined = true
	gs.mu.Unlock()

	gs.Reset()

	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.pendingLevelOffer {
		t.Fatal("Reset must clear the level offer")
	}
	if len(gs.pendingPackage) != 5 || !gs.paused {
		t.Fatalf("pending %v, paused %v; Reset must offer the due package again and pause the new game", gs.pendingPackage, gs.paused)
	}
	if !gs.introduced["かさ"] || !equalIDs(gs.newestPackage, []string{"かさ"}) || !gs.levelOfferDeclined {
		t.Fatal("Reset must keep introduced words, the newest package and the Later of the level offer")
	}
}

func TestRecheckWhenAKanaBecomesSecure(t *testing.T) {
	gs := packageReady(10)
	gs.overallStats["あ"] = store.KanaStats{Char: "あ", CorrectCount: 2}
	gs.overallStats["い"] = store.KanaStats{Char: "い", CorrectCount: 0}
	gs.overallStats["う"] = store.KanaStats{Char: "う", CorrectCount: 0}
	gs.overallStats["え"] = store.KanaStats{Char: "え", CorrectCount: 0}
	gs.invalidatePlayable()
	gs.tiles = []*KanaTile{tileAt("あ", "a", 100)}
	gs.checkAnswer("a")
	if len(gs.pendingPackage) == 0 || !gs.paused {
		t.Fatalf("あ became secure: pending %v, want a package of あ/お words", gs.pendingPackage)
	}
}

func TestApplySettingsRechecksPackages(t *testing.T) {
	gs := packageReady(10)
	gs.wordSource = listSourcePrefix + "gone"
	gs.maybeOfferPackage()
	if len(gs.pendingPackage) != 0 {
		t.Fatal("precondition: no package from a missing list")
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 3})
	gs.applySettings([]string{"vowels"}, []kanacore.Script{kanacore.ScriptHiragana}, false, 0, LevelNormal, pointsAuto)
	if len(gs.pendingPackage) != 3 || !gs.paused {
		t.Fatalf("after Save: pending %v, want a package of 3 from JLPT", gs.pendingPackage)
	}
}

func TestSourceChangeClearsNewestPackage(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.newestPackage = []string{"あい"}
	gs.mu.Unlock()
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 4, packageSize: 10})
	if !equalIDs(gs.newestPackage, []string{"あい"}) || gs.wordLevel != 4 || gs.packageSize != 10 {
		t.Fatalf("same source: newest %v, level %d, size %d", gs.newestPackage, gs.wordLevel, gs.packageSize)
	}
	gs.applyWordSettings(wordSettings{source: listSourcePrefix + "minna", level: 4, packageSize: 10})
	if gs.newestPackage != nil || gs.wordSource != "list:minna" {
		t.Fatalf("new source: newest %v, source %q", gs.newestPackage, gs.wordSource)
	}
	if s, _ := st.WordSource(); s != "list:minna" {
		t.Fatalf("stored source = %q", s)
	}
	if np, _ := st.NewestPackage(); np != nil {
		t.Fatalf("stored newest_package = %v, want cleared", np)
	}
	if l, _ := st.WordLevel(); l != 4 {
		t.Fatalf("stored level = %d", l)
	}
	if n, _ := st.PackageSize(); n != 10 {
		t.Fatalf("stored package size = %d", n)
	}
}

// levelDone returns a game whose N5 words are all introduced and secure,
// with N4 words left.
func levelDone() *GameState {
	gs := newTestState()
	selectRows(gs, "vowels")
	n5 := vowelWords(6)
	n4 := []vocab.Word{bw("ええ", 4, "yes"), bw("おお", 4, "big")}
	gs.words = vocab.NewIndex(append(n5, n4...), nil)
	secure(gs, "あ", "い", "う", "え", "お")
	for _, w := range n5 {
		gs.introduced[w.Kana] = true
		gs.wordOverallStats[w.Kana] = store.WordStats{Kana: w.Kana, CorrectCount: 3}
	}
	gs.newestPackage = []string{n5[5].Kana}
	return gs
}

func TestLevelOfferWhenTheLevelIsExhausted(t *testing.T) {
	gs := levelDone()
	gs.maybeOfferPackage()
	if !gs.pendingLevelOffer || !gs.paused || len(gs.pendingPackage) != 0 {
		t.Fatalf("offer %v, paused %v, package %v", gs.pendingLevelOffer, gs.paused, gs.pendingPackage)
	}
	if kinds := drainEvents(gs); len(kinds) != 1 || kinds[0] != levelOfferEvent {
		t.Fatalf("events = %v, want one levelOfferEvent", kinds)
	}
	gs.TogglePause()
	gs.TogglePause()
	gs.Resume()
	if !gs.paused {
		t.Fatal("the pause must hold while the level offer is pending")
	}

	gs.AcceptLevelOffer()
	if gs.wordLevel != 4 || gs.pendingLevelOffer {
		t.Fatalf("level %d, offer %v; want N4 and the offer gone", gs.wordLevel, gs.pendingLevelOffer)
	}
	if !equalIDs(gs.pendingPackage, []string{"ええ", "おお"}) || !gs.paused {
		t.Fatalf("pending %v; want the N4 words as the next package", gs.pendingPackage)
	}
}

func TestLevelOfferLaterHoldsForTheSession(t *testing.T) {
	gs := levelDone()
	gs.maybeOfferPackage()
	gs.DeclineLevelOffer()
	if gs.pendingLevelOffer || gs.paused || gs.wordLevel != 5 {
		t.Fatalf("offer %v, paused %v, level %d after Later", gs.pendingLevelOffer, gs.paused, gs.wordLevel)
	}
	gs.maybeOfferPackage()
	gs.Reset()
	if gs.pendingLevelOffer {
		t.Fatal("the offer must not come again in the same app run")
	}
	if got := formatWordsLine(gs.wordsStatus()); got != "Words: JLPT N5 finished" {
		t.Fatalf("line = %q", got)
	}
}

func TestNoLevelOfferAtN1OrForAList(t *testing.T) {
	gs := levelDone()
	gs.wordLevel = 1
	for _, w := range []string{"ええ", "おお"} {
		gs.introduced[w] = true
	}
	gs.maybeOfferPackage()
	if gs.pendingLevelOffer {
		t.Fatal("no level below N1")
	}
	gs = levelDone()
	gs.listNames = []string{"mine"}
	gs.wordSource = listSourcePrefix + "mine"
	gs.maybeOfferPackage()
	if gs.pendingLevelOffer {
		t.Fatal("a list source has no level offer")
	}
}
```

Adjust two existing tests. `fyne/game_lifecycle_test.go`:

Replace:

```go
	if gs.selectedRows["h"] {
		t.Error("catch-up must stop at the first unmastered step")
	}
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("returning user should get no intro (paused=%v pending=%v)", gs.paused, gs.pendingIntro)
	}
}
```

with:

```go
	if gs.selectedRows["h"] {
		t.Error("catch-up must stop at the first unmastered step")
	}
	if gs.pendingIntro != nil {
		t.Fatalf("returning user should get no intro (pending=%v)", gs.pendingIntro)
	}
	// With these kana secure the first word package is due; only it may
	// hold the pause.
	if gs.paused != (len(gs.pendingPackage) > 0) {
		t.Fatalf("paused=%v with pending package %v", gs.paused, gs.pendingPackage)
	}
}
```

`fyne/wordgame_test.go`:

Replace:

```go
func TestWordHitShowsFeedbackAtTile(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 200), tileAt("か", "ka", 0)}
	gs.checkAnswer("ka")
	if len(gs.feedback) != 0 {
```

with:

```go
func TestWordHitShowsFeedbackAtTile(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	gs.levelOfferDeclined = true // N5 is all introduced; the offer would pause the game
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 200), tileAt("か", "ka", 0)}
	gs.checkAnswer("ka")
	if len(gs.feedback) != 0 {
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test -count=1 ./fyne/`
Expected: FAIL — `gs.pendingPackage undefined`, `undefined: readingsOf`, `gs.maybeOfferPackage undefined`, `undefined: wordPackageEvent`.

- [ ] **Step 3: Create `fyne/packages.go`**

```go
package main

import (
	"time"

	"kana/memory"
	"kana/vocab"
)

// packageSizes are the package sizes the settings offer.
var packageSizes = []int{3, 5, 10}

func validPackageSize(n int) bool {
	for _, s := range packageSizes {
		if s == n {
			return true
		}
	}
	return false
}

// wordSettings are the settings dialog's Words tab values.
type wordSettings struct {
	source      string // jlptSource or listSourcePrefix + a list name
	level       int    // JLPT level packages come down to, 1–5
	packageSize int    // 3, 5 or 10
}

// readingsOf returns the readings of entries, in order.
func readingsOf(entries []vocab.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Kana
	}
	return out
}

// newestPackageDone reports whether the newest package is learned well
// enough for the next one: at least ceil(0.8 × p) of its p readings that
// are playable now are secure. With none playable (a script switched off,
// a list gone) it counts as done. Must be called under lock.
func (gs *GameState) newestPackageDone() bool {
	playable, secured := 0, 0
	for _, kana := range gs.newestPackage {
		e, ok := gs.words.Lookup(kana)
		if !ok || !gs.unitsKnown(e.Units) {
			continue
		}
		playable++
		if gs.wordCorrect(kana) >= secureCorrect {
			secured++
		}
	}
	return playable == 0 || secured*5 >= playable*4
}

// packageDue returns the next word package if one is due, or reports that
// the level offer is due instead: when the JLPT source has no word left at
// the current level and an easier level remains. Nothing is due when over,
// while a package or the level offer is pending, without words or without
// the introduced words, while the newest package is not learned yet, or
// when the chosen list is missing. Must be called under lock.
func (gs *GameState) packageDue() (pkg []vocab.Entry, levelOffer bool) {
	if gs.over || gs.words == nil || gs.introducedLoadFailed || len(gs.pendingPackage) > 0 || gs.pendingLevelOffer {
		return nil, false
	}
	if !gs.newestPackageDone() {
		return nil, false
	}
	src, found := gs.source()
	if !found {
		return nil, false
	}
	if pkg := vocab.NextPackage(gs.words, src, gs.introduced, gs.packageSize, gs.unitKnown); len(pkg) > 0 {
		return pkg, false
	}
	offer := src.List == "" && gs.wordsErr == "" && gs.wordLevel > 1 && !gs.levelOfferDeclined && gs.sourceLeft(src) == 0
	return nil, offer
}

// maybeOfferPackage queues the next word package or the level offer if one
// is due, and pauses the game until its dialog is done. Pending is set only
// when the event is delivered; otherwise the next re-check point tries
// again. Must be called under lock.
func (gs *GameState) maybeOfferPackage() {
	pkg, level := gs.packageDue()
	switch {
	case len(pkg) > 0:
		select {
		case gs.eventCh <- gameEvent{kind: wordPackageEvent}:
			gs.pendingPackage = readingsOf(pkg)
			gs.paused = true
		default:
		}
	case level:
		select {
		case gs.eventCh <- gameEvent{kind: levelOfferEvent}:
			gs.pendingLevelOffer = true
			gs.paused = true
		default:
		}
	}
}

// offerPackageAtStart sets a due package or level offer pending without an
// event, like startLearningPath sets the first intro: the first
// showPendingDialogs shows it. Called from NewGameState before the game
// starts, so no lock is needed.
func (gs *GameState) offerPackageAtStart() {
	pkg, level := gs.packageDue()
	switch {
	case len(pkg) > 0:
		gs.pendingPackage = readingsOf(pkg)
		gs.paused = true
	case level:
		gs.pendingLevelOffer = true
		gs.paused = true
	}
}

// introduce marks a reading as introduced and saves it at once; a failed
// save is kept in introducedDirty, which mergeSessionStats retries. Must be
// called under lock.
func (gs *GameState) introduce(kana string, now time.Time) {
	gs.introduced[kana] = true
	if gs.store == nil || gs.introducedLoadFailed {
		return
	}
	if err := gs.store.SaveIntroduced(kana, now); err != nil {
		gs.introducedDirty[kana] = true
	}
}

// saveIntroducedDirty retries introduced readings whose save failed. Must
// be called under lock.
func (gs *GameState) saveIntroducedDirty() {
	if gs.store == nil || gs.introducedLoadFailed {
		return
	}
	for kana := range gs.introducedDirty {
		if err := gs.store.SaveIntroduced(kana, gs.now()); err == nil {
			delete(gs.introducedDirty, kana)
		}
	}
}

// markKnown records a word the learner already knows as secure: its stored
// correct count becomes at least secureCorrect, written at once and kept
// out of the session stats so the session tiles stay honest, and it gets
// the memory state of three correct answers. No points, no tempo change.
// Must be called under lock.
func (gs *GameState) markKnown(kana string, now time.Time) {
	ws := gs.wordOverallStats[kana]
	ws.Kana = kana
	ws.CorrectCount = max(ws.CorrectCount, secureCorrect)
	if gs.store != nil {
		_ = gs.store.SaveWordStats(kana, ws.CorrectCount, ws.MissCount)
	}
	gs.wordOverallStats[kana] = ws
	gs.wordMemory[kana] = memory.Seed(secureCorrect, now)
	gs.wordMemoryDirty[kana] = true
}

// FinishPackage closes the word package intro: every pending reading
// becomes introduced, the ones in known count as secure, and the others
// form the newest package for the 80 % rule. Both are saved at once. Then
// the game resumes unless another dialog is due.
func (gs *GameState) FinishPackage(known []string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	now := gs.now()
	isKnown := make(map[string]bool, len(known))
	for _, kana := range known {
		isKnown[kana] = true
	}
	var newest []string
	for _, kana := range gs.pendingPackage {
		gs.introduce(kana, now)
		if isKnown[kana] {
			gs.markKnown(kana, now)
		} else {
			newest = append(newest, kana)
		}
	}
	gs.newestPackage = newest
	if gs.store != nil {
		_ = gs.store.SaveNewestPackage(newest)
	}
	gs.pendingPackage = nil
	gs.invalidatePlayable()
	gs.settle()
	gs.maybeOfferPackage()
}

// ReplacementWord adds the next word of the source to the pending package,
// for a word the learner ticked as already known, and returns it; false
// when the source has none left.
func (gs *GameState) ReplacementWord() (vocab.Entry, bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.words == nil || len(gs.pendingPackage) == 0 {
		return vocab.Entry{}, false
	}
	src, found := gs.source()
	if !found {
		return vocab.Entry{}, false
	}
	skip := make(map[string]bool, len(gs.introduced)+len(gs.pendingPackage))
	for kana := range gs.introduced {
		skip[kana] = true
	}
	for _, kana := range gs.pendingPackage {
		skip[kana] = true
	}
	next := vocab.NextPackage(gs.words, src, skip, 1, gs.unitKnown)
	if len(next) == 0 {
		return vocab.Entry{}, false
	}
	gs.pendingPackage = append(gs.pendingPackage, next[0].Kana)
	return next[0], true
}

// AcceptLevelOffer lowers the JLPT level by one (N5 → N4) and offers the
// first package of the new level.
func (gs *GameState) AcceptLevelOffer() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.pendingLevelOffer = false
	if gs.wordLevel > 1 {
		gs.wordLevel--
		if gs.store != nil {
			_ = gs.store.SaveWordLevel(gs.wordLevel)
		}
	}
	gs.invalidatePlayable()
	gs.settle()
	gs.maybeOfferPackage()
}

// DeclineLevelOffer dismisses the level offer for the rest of the app run.
func (gs *GameState) DeclineLevelOffer() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.pendingLevelOffer = false
	gs.levelOfferDeclined = true
	gs.settle()
}

// applyWordSettings applies the settings dialog's Words tab. A new source
// clears the newest package, so its first package can come at once; words
// already introduced keep falling. The caller re-checks packages
// (applySettings does).
func (gs *GameState) applyWordSettings(ws wordSettings) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	changed := false
	if ws.source != "" && ws.source != gs.wordSource {
		gs.wordSource = ws.source
		gs.newestPackage = nil
		changed = true
		if gs.store != nil {
			_ = gs.store.SaveWordSource(ws.source)
			_ = gs.store.SaveNewestPackage(nil)
		}
	}
	if ws.level >= 1 && ws.level <= 5 && ws.level != gs.wordLevel {
		gs.wordLevel = ws.level
		changed = true
		if gs.store != nil {
			_ = gs.store.SaveWordLevel(ws.level)
		}
	}
	if validPackageSize(ws.packageSize) && ws.packageSize != gs.packageSize {
		gs.packageSize = ws.packageSize
		changed = true
		if gs.store != nil {
			_ = gs.store.SavePackageSize(ws.packageSize)
		}
	}
	if changed {
		gs.invalidatePlayable()
		gs.dropInactiveTiles()
	}
}
```

- [ ] **Step 4: Events, fields, pause invariant and re-check points in `fyne/game.go`**

Replace:

```go
	gameOverEvent gameEventType = iota
	rowsUnlockedEvent
	katakanaOfferEvent
)

type gameEvent struct {
```

with:

```go
	gameOverEvent gameEventType = iota
	rowsUnlockedEvent
	katakanaOfferEvent
	wordPackageEvent // a word package waits for its intro
	levelOfferEvent  // the next JLPT level waits to be offered
)

type gameEvent struct {
```

Replace:

```go
	pendingOffer    bool // the katakana offer dialog is waiting
	katakanaOffered bool // the offer was already made once

	userPaused   bool // the learner paused with the button or Esc
	settingsOpen bool // the settings dialog is open
```

with:

```go
	pendingOffer    bool // the katakana offer dialog is waiting
	katakanaOffered bool // the offer was already made once

	pendingPackage     []string // readings waiting in the word package intro
	pendingLevelOffer  bool     // the JLPT level offer is waiting
	levelOfferDeclined bool     // "Later" on the level offer; for the app run, survives Reset

	userPaused   bool // the learner paused with the button or Esc
	settingsOpen bool // the settings dialog is open
```

Replace:

```go
	if sourceLoaded && gs.wordSource != storedSource {
		_ = st.SaveWordSource(gs.wordSource)
	}
	return gs
}
```

with:

```go
	if sourceLoaded && gs.wordSource != storedSource {
		_ = st.SaveWordSource(gs.wordSource)
	}
	gs.offerPackageAtStart()
	return gs
}
```

Replace:

```go
	// matching settle()'s invariant (settle itself only ever unpauses, so it
	// cannot express this one case where Reset must pause).
	gs.pendingOffer = false
	gs.userPaused = false
	gs.settingsOpen = false
	gs.paused = len(gs.pendingIntro) > 0
```

with:

```go
	// matching settle()'s invariant (settle itself only ever unpauses, so it
	// cannot express this one case where Reset must pause).
	gs.pendingOffer = false
	gs.pendingPackage = nil
	gs.pendingLevelOffer = false
	gs.userPaused = false
	gs.settingsOpen = false
	gs.paused = len(gs.pendingIntro) > 0
```

Replace:

```go
		}
	}
	gs.invalidatePlayable()

	gs.buildSnapshot()
	gs.mu.Unlock()
```

with:

```go
		}
	}
	gs.invalidatePlayable()
	// A package due now is offered before the new game starts; the offer
	// pauses it (settle cannot, it only ever unpauses).
	gs.maybeOfferPackage()

	gs.buildSnapshot()
	gs.mu.Unlock()
```

Replace:

```go
		if !announced {
			gs.maybeOfferKatakana()
		}
		gs.buildSnapshot()
	}
	canvas := gs.canvas
```

with:

```go
		if !announced {
			gs.maybeOfferKatakana()
		}
		gs.maybeOfferPackage()
		gs.buildSnapshot()
	}
	canvas := gs.canvas
```

Replace:

```go
// stays dirty for a later retry, but Reset discards it, for kana and words
// alike. Must be called under lock.
func (gs *GameState) mergeSessionStats() {
	gs.saveMemory()
	gs.mergeWordStats()
	gs.mergeKanaStats()
```

with:

```go
// stays dirty for a later retry, but Reset discards it, for kana and words
// alike. Must be called under lock.
func (gs *GameState) mergeSessionStats() {
	gs.saveIntroducedDirty()
	gs.saveMemory()
	gs.mergeWordStats()
	gs.mergeKanaStats()
```

Replace:

```go
	gs.dropInactiveTiles()
	gs.buildSnapshot()
	gs.settle()
}

// dropInactiveTiles removes kana tiles whose row is deselected or whose
```

with:

```go
	gs.dropInactiveTiles()
	gs.buildSnapshot()
	gs.settle()
	gs.maybeOfferPackage()
}

// dropInactiveTiles removes kana tiles whose row is deselected or whose
```

Replace:

```go
// is pending, the learner has paused or the settings dialog is open. It is the
// only place that unpauses. Must be called under lock.
func (gs *GameState) settle() {
	if gs.paused && len(gs.pendingIntro) == 0 && !gs.pendingOffer && !gs.userPaused && !gs.settingsOpen {
		gs.paused = false
	}
}
```

with:

```go
// is pending, the learner has paused or the settings dialog is open. It is the
// only place that unpauses. Must be called under lock.
func (gs *GameState) settle() {
	if gs.paused && len(gs.pendingIntro) == 0 && !gs.pendingOffer && len(gs.pendingPackage) == 0 && !gs.pendingLevelOffer &&
		!gs.userPaused && !gs.settingsOpen {
		gs.paused = false
	}
}
```

Replace:

```go
	gs.pendingOffer = false
	gs.enableKatakanaLocked()
	gs.settle()
}

// DeclineKatakana dismisses the offer.
```

with:

```go
	gs.pendingOffer = false
	gs.enableKatakanaLocked()
	gs.settle()
	gs.maybeOfferPackage()
}

// DeclineKatakana dismisses the offer.
```

Replace:

```go
	defer gs.mu.Unlock()
	gs.pendingOffer = false
	gs.settle()
}

// PendingDialogs returns whether the offer is pending and a copy of the rows
```

with:

```go
	defer gs.mu.Unlock()
	gs.pendingOffer = false
	gs.settle()
	gs.maybeOfferPackage()
}

// PendingDialogs returns whether the offer is pending and a copy of the rows
```

Replace:

```go
	gs.mu.Unlock()
}

// FinishIntro clears the pending intro and resumes the game.
func (gs *GameState) FinishIntro() {
	gs.mu.Lock()
	gs.pendingIntro = nil
	gs.settle()
	gs.mu.Unlock()
}
```

with:

```go
	gs.mu.Unlock()
}

// FinishIntro clears the pending intro and resumes the game unless a word
// package is due.
func (gs *GameState) FinishIntro() {
	gs.mu.Lock()
	gs.pendingIntro = nil
	gs.settle()
	gs.maybeOfferPackage()
	gs.mu.Unlock()
}
```

The re-check points are now: the end of `checkAnswer`, the end of `Reset` (after `paused` is recomputed), `FinishIntro`, `EnableKatakana`, `DeclineKatakana`, the end of `applySettings` (settings Save), and inside `FinishPackage` and `AcceptLevelOffer`; `NewGameState` uses `offerPackageAtStart`. `mergeSessionStats` retries introduced readings whose immediate save failed.

- [ ] **Step 5: Run**

Run: `go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 6: Commit**

```bash
git add fyne/game.go fyne/packages.go fyne/packages_test.go fyne/game_lifecycle_test.go fyne/wordgame_test.go
git commit -m "feat(fyne): offer word packages and the next JLPT level

When 80 % of the newest package's playable words are secure, the next
package of the source is queued and pauses the game like a kana intro;
an exhausted JLPT level queues the offer of the next one instead.
Finishing a package introduces its words at once and counts the ones
ticked as already known as secure."
```

---

### Task 6: Word package and level offer dialogs

**Files:** Modify `fyne/game.go`, `fyne/intro.go`, `fyne/app.go`. Create `fyne/packageintro.go`, `fyne/packageintro_test.go`.

- [ ] **Step 1: Write the failing tests** — create `fyne/packageintro_test.go`:

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/vocab"
)

func TestPendingDialogsReportsEveryKind(t *testing.T) {
	gs := newTestState()
	if p := gs.PendingDialogs(); len(p.Intro) != 0 || p.KatakanaOffer || len(p.Package) != 0 || p.LevelOffer != 0 {
		t.Fatalf("nothing pending: %+v", p)
	}
	gs.pendingIntro = []string{"k"}
	gs.pendingOffer = true
	gs.pendingPackage = []string{"あい"}
	gs.pendingLevelOffer = true
	gs.wordLevel = 4
	p := gs.PendingDialogs()
	if !equalIDs(p.Intro, []string{"k"}) || !p.KatakanaOffer || !equalIDs(p.Package, []string{"あい"}) || p.LevelOffer != 4 {
		t.Fatalf("pending = %+v", p)
	}
	p.Package[0] = "x"
	if gs.pendingPackage[0] != "あい" {
		t.Fatal("PendingDialogs must return copies")
	}
}

func TestPackageTitle(t *testing.T) {
	list := vocab.List{Name: "minna-no-nihongo", Words: []vocab.Word{
		{Kana: "あい", Lesson: "2", Meanings: vocab.Meanings{}},
		{Kana: "あお", Lesson: "3", Meanings: vocab.Meanings{}},
		{Kana: "いえ", Meanings: vocab.Meanings{}},
	}}
	ix := vocab.NewIndex([]vocab.Word{bw("あい", 5, "love"), bw("あお", 4, "blue")}, []vocab.List{list})
	e := func(kana string) vocab.Entry { x, _ := ix.Lookup(kana); return x }
	minna := vocab.Source{List: "minna-no-nihongo"}
	cases := []struct {
		name    string
		src     vocab.Source
		entries []vocab.Entry
		want    string
	}{
		{"JLPT", vocab.Source{Level: 5}, []vocab.Entry{e("あい")}, "New words · JLPT N5"},
		{"JLPT across levels", vocab.Source{Level: 4}, []vocab.Entry{e("あい"), e("あお")}, "New words · JLPT N5–N4"},
		{"one lesson", minna, []vocab.Entry{e("あい")}, "New words · minna no nihongo · lesson 2"},
		{"two lessons", minna, []vocab.Entry{e("あい"), e("あお"), e("いえ")}, "New words · minna no nihongo · lessons 2–3"},
		{"no lessons", minna, []vocab.Entry{e("いえ")}, "New words · minna no nihongo"},
		{"empty", vocab.Source{Level: 5}, nil, "New words"},
	}
	for _, c := range cases {
		if got := packageTitle(ix, c.src, c.entries); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPackageViewAlreadyKnown(t *testing.T) {
	test.NewApp()
	entries := []vocab.Entry{
		{Kana: "あい", Meanings: vocab.Meanings{"en": {"love"}}},
		{Kana: "あお", Meanings: vocab.Meanings{"en": {"blue"}}},
	}
	extra := []vocab.Entry{{Kana: "いえ", Meanings: vocab.Meanings{"en": {"house"}}}}
	calls := 0
	v := newPackageView(entries, "en", func() (vocab.Entry, bool) {
		calls++
		if len(extra) == 0 {
			return vocab.Entry{}, false
		}
		e := extra[0]
		extra = extra[1:]
		return e, true
	})
	if len(v.rows) != 2 || len(v.list.Objects) != 2 {
		t.Fatalf("rows = %d", len(v.rows))
	}
	v.rows[0].known.SetChecked(true)
	if calls != 1 || len(v.rows) != 3 || v.rows[2].entry.Kana != "いえ" || len(v.list.Objects) != 3 {
		t.Fatalf("ticking must add the replacement いえ: calls %d, rows %d", calls, len(v.rows))
	}
	if !v.rows[0].strike.Visible() {
		t.Fatal("a ticked word is struck through")
	}
	v.rows[0].known.SetChecked(false)
	if len(v.rows) != 3 || v.rows[0].strike.Visible() {
		t.Fatal("unticking keeps the replacement and removes the strike")
	}
	v.rows[0].known.SetChecked(true)
	if calls != 1 {
		t.Fatal("ticking the same word again must not ask for another replacement")
	}
	v.rows[1].known.SetChecked(true)
	if calls != 2 || len(v.rows) != 3 || !v.rows[1].known.Checked {
		t.Fatalf("no replacement left: the box stays ticked and nothing is added (calls %d, rows %d)", calls, len(v.rows))
	}
	if got := v.known(); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("known = %v", got)
	}
}

// buttonIn finds the button with the given text in a dialog's objects.
func buttonIn(o fyne.CanvasObject, text string) *widget.Button {
	for _, obj := range test.LaidOutObjects(o) {
		if b, ok := obj.(*widget.Button); ok && b.Text == text {
			return b
		}
	}
	return nil
}

func TestPackageDialogFinishesThePackage(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	gs := packageReady(20)
	statsPanel := newStatsPanel()
	ib := newInputBar(gs, statsPanel, newGameCanvas(gs), w)
	gs.maybeOfferPackage()
	drainEvents(gs)
	pkg := append([]string(nil), gs.pendingPackage...)

	showPendingDialogs(gs, statsPanel, ib, w)
	top := w.Canvas().Overlays().Top()
	if top == nil || !dialogShowing {
		t.Fatal("the word package dialog is not showing")
	}
	if !ib.entry.Disabled() {
		t.Fatal("input must be disabled while the dialog shows")
	}
	if top.Size().Height > 760 {
		t.Fatalf("dialog is %v px high, more than the window", top.Size().Height)
	}
	b := buttonIn(top, "Let's go")
	if b == nil {
		t.Fatal("no Let's go button")
	}
	test.Tap(b)
	if dialogShowing || len(gs.pendingPackage) != 0 || gs.paused {
		t.Fatalf("after Let's go: showing %v, pending %v, paused %v", dialogShowing, gs.pendingPackage, gs.paused)
	}
	for _, kana := range pkg {
		if !gs.introduced[kana] {
			t.Fatalf("%s not introduced", kana)
		}
	}
	if ib.entry.Disabled() {
		t.Fatal("input must be enabled again")
	}
}

func TestLevelOfferDialog(t *testing.T) {
	for _, accept := range []bool{true, false} {
		test.NewApp()
		w := test.NewWindow(nil)
		gs := levelDone()
		statsPanel := newStatsPanel()
		ib := newInputBar(gs, statsPanel, newGameCanvas(gs), w)
		gs.maybeOfferPackage()
		drainEvents(gs)
		showPendingDialogs(gs, statsPanel, ib, w)
		top := w.Canvas().Overlays().Top()
		label := "Later"
		if accept {
			label = "Continue with N4"
		}
		b := buttonIn(top, label)
		if b == nil {
			t.Fatalf("no %q button", label)
		}
		test.Tap(b)
		if accept && (gs.wordLevel != 4 || len(gs.pendingPackage) == 0 || !dialogShowing) {
			t.Fatalf("accept: level %d, pending %v; want N4 and its package dialog", gs.wordLevel, gs.pendingPackage)
		}
		if !accept && (gs.wordLevel != 5 || !gs.levelOfferDeclined || gs.paused || dialogShowing) {
			t.Fatalf("later: level %d, declined %v, paused %v", gs.wordLevel, gs.levelOfferDeclined, gs.paused)
		}
		dialogShowing = false
		w.Close()
	}
	test.NewApp()
	if got := levelOfferText(5); got != "N5 finished — continue with N4?" {
		t.Fatalf("text = %q", got)
	}
}
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test -count=1 ./fyne/`
Expected: FAIL — `assignment mismatch: 1 variable but gs.PendingDialogs returns 2 values`, `undefined: packageTitle`, `undefined: newPackageView`, `undefined: levelOfferText`.

- [ ] **Step 3: `Pending` in `fyne/game.go`**

Replace:

```go
	gs.maybeOfferPackage()
}

// PendingDialogs returns whether the offer is pending and a copy of the rows
// waiting to be introduced.
func (gs *GameState) PendingDialogs() (offer bool, intro []string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.pendingOffer, append([]string(nil), gs.pendingIntro...)
}

// IsOver reports whether the current game has ended.
```

with:

```go
	gs.maybeOfferPackage()
}

// Pending lists the dialogs the game is waiting for.
type Pending struct {
	Intro         []string // kana row IDs to introduce
	KatakanaOffer bool     // the one-time katakana offer
	Package       []string // readings of the word package to introduce
	LevelOffer    int      // the finished JLPT level whose successor is offered; 0 = none
}

// PendingDialogs returns copies of everything the game is waiting for.
func (gs *GameState) PendingDialogs() Pending {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	p := Pending{
		Intro:         append([]string(nil), gs.pendingIntro...),
		KatakanaOffer: gs.pendingOffer,
		Package:       append([]string(nil), gs.pendingPackage...),
	}
	if gs.pendingLevelOffer {
		p.LevelOffer = gs.wordLevel
	}
	return p
}

// IsOver reports whether the current game has ended.
```

- [ ] **Step 4: Create `fyne/packageintro.go`**

```go
package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/vocab"
)

// Word package intro layout: the kana column, the row height and the
// scroll area, sized so ten words fit the 1000 × 760 window.
const (
	packageKanaSize    float32 = 32
	packageKanaMinSize float32 = 16
	packageKanaW       float32 = 240
	packageRowH        float32 = 56
	packageViewW       float32 = 680
	packageViewMaxH    float32 = 480
)

// packageIntro is what the word package dialog shows.
type packageIntro struct {
	Title   string
	Entries []vocab.Entry
	Lang    string // meaning language
}

// PackageIntro returns the pending package's entries, in order, with the
// dialog title and the meaning language.
func (gs *GameState) PackageIntro() packageIntro {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	in := packageIntro{Title: "New words", Lang: gs.meaningLang}
	if gs.words == nil {
		return in
	}
	for _, kana := range gs.pendingPackage {
		if e, ok := gs.words.Lookup(kana); ok {
			in.Entries = append(in.Entries, e)
		}
	}
	src, _ := gs.source()
	in.Title = packageTitle(gs.words, src, in.Entries)
	return in
}

// packageTitle names a package: "New words · JLPT N5" (or "N5–N4" when it
// spans levels), "New words · minna no nihongo · lesson 2" (or "lessons
// 2–3"); a list without lessons gives just its name.
func packageTitle(ix *vocab.Index, src vocab.Source, entries []vocab.Entry) string {
	var parts []string
	if src.List == "" {
		for _, e := range entries {
			if l := fmt.Sprintf("N%d", e.JLPT); !slices.Contains(parts, l) {
				parts = append(parts, l)
			}
		}
		switch len(parts) {
		case 0:
			return "New words"
		case 1:
			return "New words · JLPT " + parts[0]
		}
		return "New words · JLPT " + parts[0] + "–" + parts[len(parts)-1]
	}
	title := "New words · " + prettyListName(src.List)
	for _, e := range entries {
		if l := ix.Lesson(e, src.List); l != "" && !slices.Contains(parts, l) {
			parts = append(parts, l)
		}
	}
	switch len(parts) {
	case 0:
		return title
	case 1:
		return title + " · lesson " + parts[0]
	}
	return title + " · lessons " + parts[0] + "–" + parts[len(parts)-1]
}

// packageView is the word package intro's content: one row per word with
// its kana, romaji, meaning and an "Already known" check. Ticking a word
// strikes it through and asks once for a replacement, which gets a row of
// its own; unticking keeps the replacement.
type packageView struct {
	list    *fyne.Container
	rows    []*packageRow
	lang    string
	replace func() (vocab.Entry, bool)
}

type packageRow struct {
	entry    vocab.Entry
	kana     *canvas.Text
	strike   *canvas.Rectangle
	known    *widget.Check
	replaced bool // a replacement was already asked for
}

func newPackageView(entries []vocab.Entry, lang string, replace func() (vocab.Entry, bool)) *packageView {
	v := &packageView{list: container.NewVBox(), lang: lang, replace: replace}
	for _, e := range entries {
		v.addRow(e)
	}
	return v
}

func (v *packageView) addRow(e vocab.Entry) {
	row := &packageRow{entry: e}
	row.kana = canvas.NewText(e.Kana, kanaTextColor)
	row.kana.TextSize = packageKanaSize
	fitText(row.kana, packageKanaW, packageKanaMinSize)
	row.strike = canvas.NewRectangle(kanaTextColor)
	row.strike.Hide()
	word := container.NewGridWrap(fyne.NewSize(packageKanaW, packageRowH), container.New(strikeLayout{}, row.kana, row.strike))
	romaji := widget.NewLabelWithStyle(kanacore.Romaji(e.Kana), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	meaning := widget.NewLabel(meaningSnippet(e.Meanings, v.lang))
	meaning.Truncation = fyne.TextTruncateEllipsis
	row.known = widget.NewCheck("Already known", func(on bool) { v.toggle(row, on) })
	v.rows = append(v.rows, row)
	v.list.Add(container.NewBorder(nil, nil, word, row.known, container.NewVBox(romaji, meaning)))
}

func (v *packageView) toggle(row *packageRow, on bool) {
	if on {
		row.strike.Show()
	} else {
		row.strike.Hide()
	}
	if !on || row.replaced {
		return
	}
	row.replaced = true
	if e, ok := v.replace(); ok {
		v.addRow(e)
		v.list.Refresh()
	}
}

// known returns the readings ticked as already known.
func (v *packageView) known() []string {
	var out []string
	for _, r := range v.rows {
		if r.known.Checked {
			out = append(out, r.entry.Kana)
		}
	}
	return out
}

// strikeLayout shows a text at the left, centred vertically, with a thin
// bar (the second object) across its middle.
type strikeLayout struct{}

func (strikeLayout) MinSize(objs []fyne.CanvasObject) fyne.Size { return objs[0].MinSize() }

func (strikeLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	text := objs[0].MinSize()
	y := (size.Height - text.Height) / 2
	objs[0].Resize(text)
	objs[0].Move(fyne.NewPos(0, y))
	objs[1].Resize(fyne.NewSize(text.Width, 2))
	objs[1].Move(fyne.NewPos(0, y+text.Height/2-1))
}

// showWordPackageDialog introduces the pending word package. "Let's go"
// finishes it. Call it through showPendingDialogs. Must run on the Fyne
// thread.
func showWordPackageDialog(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	intro := gs.PackageIntro()
	view := newPackageView(intro.Entries, intro.Lang, gs.ReplacementWord)
	scroll := container.NewVScroll(view.list)
	scroll.SetMinSize(fyne.NewSize(packageViewW, min(float32(len(intro.Entries))*packageRowH, packageViewMaxH)))

	inputBar.SetEnabled(false, w)
	d := dialog.NewCustom(intro.Title, "Let's go", scroll, w)
	d.SetOnClosed(func() {
		dialogShowing = false
		gs.FinishPackage(view.known())
		continueAfterDialog(gs, statsPanel, inputBar, w)
	})
	dialogShowing = true
	d.Show()
}

// levelOfferText asks whether to continue with the next JLPT level.
func levelOfferText(level int) string {
	return fmt.Sprintf("N%d finished — continue with N%d?", level, level-1)
}

// showLevelOffer offers the next JLPT level once the current one is fully
// introduced. Call it through showPendingDialogs. Must run on the Fyne
// thread.
func showLevelOffer(gs *GameState, level int, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	body := widget.NewLabel(levelOfferText(level))
	inputBar.SetEnabled(false, w)
	d := dialog.NewCustomConfirm("More words", fmt.Sprintf("Continue with N%d", level-1), "Later", body, func(accept bool) {
		// The response callback runs before SetOnClosed; clear the guard
		// first so the chained package dialog is not swallowed.
		dialogShowing = false
		if accept {
			gs.AcceptLevelOffer()
		} else {
			gs.DeclineLevelOffer()
		}
		continueAfterDialog(gs, statsPanel, inputBar, w)
	}, w)
	dialogShowing = true
	d.Show()
}
```

- [ ] **Step 5: Show the new dialogs**

`fyne/intro.go`:

Replace:

```go
	introCardH     float32 = 120
)

// dialogShowing guards against stacking the offer and intro dialogs.
// Fyne-thread-only: read and written exclusively from code that runs via
// fyne.Do or a Fyne callback, so it needs no lock.
var dialogShowing bool

// showPendingDialogs shows whatever the game is waiting for: the katakana
// offer first, then an intro. With nothing pending it resumes the game and
// re-enables input, so "paused" never outlives its dialog. It also refreshes
// the stats panel, so a change that became visible only by accepting or
// declining a dialog (e.g. a newly active script's path line) shows up
// without waiting for the next Enter. Must run on the Fyne thread.
func showPendingDialogs(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	if dialogShowing {
		return
```

with:

```go
	introCardH     float32 = 120
)

// dialogShowing guards against stacking the offer, intro and package
// dialogs.
// Fyne-thread-only: read and written exclusively from code that runs via
// fyne.Do or a Fyne callback, so it needs no lock.
var dialogShowing bool

// showPendingDialogs shows whatever the game is waiting for, one dialog at a
// time: a kana intro, the katakana offer, a word package, the level offer.
// With nothing pending it resumes the game and re-enables input, so
// "paused" never outlives its dialog. It also refreshes the stats panel, so
// a change that became visible only by accepting or declining a dialog
// (e.g. a newly active script's path line) shows up without waiting for the
// next Enter. Must run on the Fyne thread.
func showPendingDialogs(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	if dialogShowing {
		return
```

Replace:

```go
	gs.mu.Unlock()
	statsPanel.Update(snap)

	offer, intro := gs.PendingDialogs()
	switch {
	case offer:
		showKatakanaOffer(gs, statsPanel, inputBar, w)
	case len(intro) > 0:
		showIntroDialog(gs, intro, statsPanel, inputBar, w)
	default:
		gs.Resume()
		if !gs.IsOver() {
```

with:

```go
	gs.mu.Unlock()
	statsPanel.Update(snap)

	p := gs.PendingDialogs()
	switch {
	case len(p.Intro) > 0:
		showIntroDialog(gs, p.Intro, statsPanel, inputBar, w)
	case p.KatakanaOffer:
		showKatakanaOffer(gs, statsPanel, inputBar, w)
	case len(p.Package) > 0:
		showWordPackageDialog(gs, statsPanel, inputBar, w)
	case p.LevelOffer > 0:
		showLevelOffer(gs, p.LevelOffer, statsPanel, inputBar, w)
	default:
		gs.Resume()
		if !gs.IsOver() {
```

`fyne/app.go`:

Replace:

```go
		}

		switch event.kind {
		case rowsUnlockedEvent, katakanaOfferEvent:
			if over {
				// Game over wins; an intro stays pending for Play Again.
				// A pending offer cannot occur at game over: Reset drops it
				// (see GameState.Reset), so dropping the event here is safe
				// for both kinds.
				continue
			}
			fyne.Do(func() { showPendingDialogs(gs, statsPanel, inputBar, w) })
```

with:

```go
		}

		switch event.kind {
		case rowsUnlockedEvent, katakanaOfferEvent, wordPackageEvent, levelOfferEvent:
			if over {
				// Game over wins; an intro stays pending for Play Again.
				// The offers and word packages cannot be pending at game
				// over (the game is paused while they are, and Reset clears
				// them and checks for a package again), so dropping the
				// event here is safe for every kind.
				continue
			}
			fyne.Do(func() { showPendingDialogs(gs, statsPanel, inputBar, w) })
```

- [ ] **Step 6: Run**

Run: `go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 7: Commit**

```bash
git add fyne/game.go fyne/intro.go fyne/app.go fyne/packageintro.go fyne/packageintro_test.go
git commit -m "feat(fyne): introduce word packages in a dialog

The intro lists each word with its romaji and meaning and an \"Already
known\" box; ticking one strikes it through and brings in another word.
\"Let's go\" finishes the package. A finished JLPT level is followed by
the offer of the next one, which \"Later\" puts off for this run."
```

---

### Task 7: Settings in tabs, the Sources tab

**Files:** Modify `fyne/settings.go` (rewritten), `fyne/version.go`, `fyne/wordgame.go`, `fyne/stats.go`, `fyne/words.go`, `fyne/settings_test.go`, `fyne/stats_test.go`.

- [ ] **Step 1: Write the failing tests** — in `fyne/settings_test.go` (imports, then the new tests at the end):

Replace:

```go
package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

func testGroupRows() []kanacore.KanaRow {
```

with:

```go
package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/store"
	"kana/vocab"
)

func testGroupRows() []kanacore.KanaRow {
```

Replace:

```go
		t.Fatalf("checkedScripts = %v", got)
	}
}
```

with:

```go
		t.Fatalf("checkedScripts = %v", got)
	}
}

// settingsGame is a game with a private list as the word source and every
// setting away from its default.
func settingsGame() *GameState {
	gs := newTestState()
	gs.words = vocab.NewIndex(
		[]vocab.Word{bw("あい", 5, "love"), bw("あお", 5, "blue"), bw("いえ", 4, "house")},
		[]vocab.List{{Name: "minna-no-nihongo", Words: []vocab.Word{
			{Kana: "あい", Meanings: vocab.Meanings{}}, {Kana: "うえ", Meanings: vocab.Meanings{}},
		}}},
	)
	gs.listNames = []string{"minna-no-nihongo"}
	gs.wordSource = listSourcePrefix + "minna-no-nihongo"
	gs.introduced = map[string]bool{"あい": true}
	gs.wordOverallStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 3}
	gs.wordLevel = 4
	gs.packageSize = 10
	gs.scoreLimit = 500
	gs.level = LevelByName("fast")
	gs.pointsSetting = 2
	gs.autoProgress = true
	return gs
}

func tabTexts(f *settingsForm) []string {
	var out []string
	for _, item := range f.tabs.Items {
		out = append(out, item.Text)
	}
	return out
}

func TestSettingsFormCarriesValues(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(settingsGame().settingsState())
	if got := tabTexts(f); !equalIDs(got, []string{"Kana", "Words", "Game", "Sources"}) {
		t.Fatalf("tabs = %v", got)
	}
	if !equalIDs(f.sourceSelect.Options, []string{"minna no nihongo", "JLPT"}) || f.sourceSelect.Selected != "minna no nihongo" {
		t.Fatalf("source options %v, selected %q", f.sourceSelect.Options, f.sourceSelect.Selected)
	}
	if f.wordLevel.Selected != "N4" || !f.wordLevel.Disabled() {
		t.Fatalf("level %q, disabled %v; want N4, disabled for a list", f.wordLevel.Selected, f.wordLevel.Disabled())
	}
	if f.packageSize.Selected != "10" {
		t.Fatalf("package size = %q", f.packageSize.Selected)
	}
	if f.summary.Text != "Introduced: 1 · secure: 1 · left in source: 1" {
		t.Fatalf("summary = %q", f.summary.Text)
	}
	if f.limitEntry.Text != "500" || f.difficulty.Selected != "Fast" || f.points.Selected != pointsLabel(2) || !f.autoCheck.Checked {
		t.Fatalf("game tab: limit %q, difficulty %q, points %q, auto %v", f.limitEntry.Text, f.difficulty.Selected, f.points.Selected, f.autoCheck.Checked)
	}
	f.sourceSelect.SetSelected("JLPT")
	if f.wordLevel.Disabled() {
		t.Fatal("the level must be enabled for JLPT")
	}
	if got := f.words(); got != (wordSettings{source: jlptSource, level: 4, packageSize: 10}) {
		t.Fatalf("words() = %+v", got)
	}
}

func TestSettingsFormMissingList(t *testing.T) {
	test.NewApp()
	gs := settingsGame()
	gs.listNames = nil
	f := newSettingsForm(gs.settingsState())
	if !equalIDs(f.sourceSelect.Options, []string{"minna no nihongo", "JLPT"}) || f.wordSource() != "list:minna-no-nihongo" {
		t.Fatalf("a missing list stays selected: options %v, source %q", f.sourceSelect.Options, f.wordSource())
	}
	if f.summary.Text != "List minna no nihongo not found" {
		t.Fatalf("summary = %q", f.summary.Text)
	}
}

func TestSettingsValidationSwitchesToTheTab(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(settingsGame().settingsState())
	if err := f.validate(); err != nil {
		t.Fatalf("valid form: %v", err)
	}
	f.limitEntry.SetText("lots")
	f.tabs.Select(f.kanaTab)
	if err := f.validate(); err == nil {
		t.Fatal("an invalid score limit must fail")
	}
	if f.tabs.Selected() != f.gameTab {
		t.Fatalf("selected tab = %q, want Game", f.tabs.Selected().Text)
	}
}

func TestSettingsApplySavesEveryTab(t *testing.T) {
	test.NewApp()
	gs := settingsGame()
	gs.newestPackage = []string{"あい"}
	f := newSettingsForm(gs.settingsState())
	f.sourceSelect.SetSelected("JLPT")
	f.wordLevel.SetSelected("N3")
	f.packageSize.SetSelected("3")
	f.limitEntry.SetText("200")
	f.difficulty.SetSelected("Relaxed")
	f.apply(gs)
	if gs.wordSource != jlptSource || gs.wordLevel != 3 || gs.packageSize != 3 || gs.newestPackage != nil {
		t.Fatalf("words: source %q, level %d, size %d, newest %v", gs.wordSource, gs.wordLevel, gs.packageSize, gs.newestPackage)
	}
	if gs.scoreLimit != 200 || gs.level.Level != LevelRelaxed {
		t.Fatalf("game: limit %d, level %v", gs.scoreLimit, gs.level.Level)
	}
}

func TestSourcesTabCreditsEverySource(t *testing.T) {
	test.NewApp()
	var texts []string
	for _, o := range test.LaidOutObjects(newSourcesTab()) {
		switch w := o.(type) {
		case *widget.Label:
			texts = append(texts, w.Text)
		case *widget.Hyperlink:
			texts = append(texts, w.Text+" <"+w.URL.String()+">")
		}
	}
	all := strings.Join(texts, "\n")
	for _, want := range []string{
		"Version dev ·", "Releases <" + releasesURL + ">",
		"JMdict", "EDRDG", "CC BY-SA 4.0", "Licence <https://www.edrdg.org/edrdg/licence.html>",
		"JMdict <https://www.edrdg.org/jmdict/j_jmdict.html>",
		"JLPT levels", "Jonathan Waller, CC BY", "Unofficial",
		"JMdict mapping", "stephenmk/yomitan-jlpt-vocab",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("Sources tab lacks %q:\n%s", want, all)
		}
	}
}

func TestSettingsDialogSaveAndInvalidValue(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	gs := settingsGame()
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)

	showSettingsDialog(gs, statsPanel, gameCanvas, ib, w)
	settings := w.Canvas().Overlays().Top()
	if settings == nil || !gs.settingsOpen {
		t.Fatal("settings dialog not open")
	}
	var tabs *container.AppTabs
	for _, o := range test.LaidOutObjects(settings) {
		if at, ok := o.(*container.AppTabs); ok && len(at.Items) == 4 {
			tabs = at
		}
	}
	if tabs == nil {
		t.Fatal("no settings tabs")
	}
	tabs.SelectIndex(2)
	var limit *widget.Entry
	for _, o := range test.LaidOutObjects(settings) {
		if e, ok := o.(*widget.Entry); ok {
			limit = e
		}
	}
	limit.SetText("-1")
	tabs.SelectIndex(0)
	test.Tap(buttonIn(settings, "Save"))
	if !gs.settingsOpen || tabs.SelectedIndex() != 2 || len(w.Canvas().Overlays().List()) != 2 {
		t.Fatalf("invalid value: open %v, tab %d, overlays %d; want the dialog open on Game with an error",
			gs.settingsOpen, tabs.SelectedIndex(), len(w.Canvas().Overlays().List()))
	}
	w.Canvas().Overlays().Remove(w.Canvas().Overlays().Top()) // dismiss the error
	limit.SetText("300")
	test.Tap(buttonIn(settings, "Save"))
	if gs.settingsOpen || gs.scoreLimit != 300 || w.Canvas().Overlays().Top() != nil {
		t.Fatalf("after Save: open %v, limit %d", gs.settingsOpen, gs.scoreLimit)
	}
}
```

`fyne/stats_test.go` (the panel no longer shows the attribution):

Replace:

```go
	}
}

func TestPanelShowsWordsLineAndSource(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
```

with:

```go
	}
}

func TestPanelShowsWordsLine(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	snap := panelSnap([]string{"vowels"}, both)
```

Replace:

```go
	if p.wordsText.Text != "Words: 42 introduced · 17 secure · JLPT N5" {
		t.Fatalf("words line = %q", p.wordsText.Text)
	}
	var lines []string
	for _, txt := range p.sourceTexts {
		lines = append(lines, txt.Text)
	}
	if got := strings.Join(lines, " "); got != "Word data: JMdict (EDRDG, CC BY-SA 4.0), JLPT levels by J. Waller" {
		t.Fatalf("attribution = %q", got)
	}
}

func TestPanelCountsWordHits(t *testing.T) {
```

with:

```go
	if p.wordsText.Text != "Words: 42 introduced · 17 secure · JLPT N5" {
		t.Fatalf("words line = %q", p.wordsText.Text)
	}
}

func TestPanelCountsWordHits(t *testing.T) {
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test -count=1 ./fyne/`
Expected: FAIL — `undefined: settingsForm`, `undefined: newSettingsForm`, `gs.settingsState undefined`, `undefined: newSourcesTab`.

- [ ] **Step 3: Replace `fyne/settings.go` with:**

```go
package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

// settingsState is what the settings dialog starts from, copied under the
// lock.
type settingsState struct {
	selected map[string]bool
	active   map[kanacore.Script]bool
	auto     bool
	limit    int
	level    Level
	points   int

	wordSource  string   // the stored word source
	lists       []string // the private lists found
	wordLevel   int
	packageSize int
	summary     string // the Words tab's summary line
}

// settingsState copies the current settings for the dialog.
func (gs *GameState) settingsState() settingsState {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return settingsState{
		selected:    maps.Clone(gs.selectedRows),
		active:      maps.Clone(gs.activeScripts),
		auto:        gs.autoProgress,
		limit:       gs.scoreLimit,
		level:       gs.level.Level,
		points:      gs.pointsSetting,
		wordSource:  gs.wordSource,
		lists:       append([]string(nil), gs.listNames...),
		wordLevel:   gs.wordLevel,
		packageSize: gs.packageSize,
		summary:     gs.wordsSummary(),
	}
}

// settingsForm is the settings dialog's content: the tabs Kana, Words, Game
// and Sources, whose widgets carry the settings.
type settingsForm struct {
	tabs                                   *container.AppTabs
	kanaTab, wordsTab, gameTab, sourcesTab *container.TabItem

	scriptChecks map[kanacore.Script]*widget.Check
	rowChecks    map[string]*widget.Check
	autoCheck    *widget.Check

	sourceSelect *widget.Select
	sourceValues []string // the word source of each sourceSelect option
	wordLevel    *widget.RadioGroup
	packageSize  *widget.RadioGroup
	summary      *widget.Label

	difficulty *widget.RadioGroup
	points     *widget.Select
	limitEntry *widget.Entry
	limit      int // the score limit the dialog started with
}

func newSettingsForm(st settingsState) *settingsForm {
	f := &settingsForm{limit: st.limit}
	f.kanaTab = container.NewTabItem("Kana", f.newKanaTab(st))
	f.wordsTab = container.NewTabItem("Words", f.newWordsTab(st))
	f.gameTab = container.NewTabItem("Game", f.newGameTab(st))
	f.sourcesTab = container.NewTabItem("Sources", newSourcesTab())
	f.tabs = container.NewAppTabs(f.kanaTab, f.wordsTab, f.gameTab, f.sourcesTab)
	return f
}

// newKanaTab holds the scripts, one row tab per script and auto-progression.
func (f *settingsForm) newKanaTab(st settingsState) fyne.CanvasObject {
	f.scriptChecks = make(map[kanacore.Script]*widget.Check)
	scriptRow := container.NewHBox(widget.NewLabelWithStyle("Scripts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, info := range kanacore.Scripts() {
		c := widget.NewCheck(info.Label, nil)
		c.SetChecked(st.active[info.Script])
		f.scriptChecks[info.Script] = c
		scriptRow.Add(c)
	}

	f.rowChecks = make(map[string]*widget.Check)
	rowTabs := container.NewAppTabs()
	for _, info := range kanacore.Scripts() {
		sections := container.NewVBox()
		for _, g := range kanacore.GroupsFor(info.Script) {
			all, checks, grid := newGroupChecks(kanacore.RowsInGroup(info.Script, g.Group), st.selected)
			for id, c := range checks {
				f.rowChecks[id] = c
			}
			sections.Add(container.NewHBox(
				widget.NewLabelWithStyle(g.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				all,
			))
			sections.Add(grid)
		}
		scroll := container.NewVScroll(sections)
		scroll.SetMinSize(fyne.NewSize(460, 220))
		rowTabs.Append(container.NewTabItem(info.Label, scroll))
	}

	f.autoCheck = widget.NewCheck("Enable auto-progression", nil)
	f.autoCheck.SetChecked(st.auto)
	return container.NewVBox(scriptRow, rowTabs, widget.NewSeparator(), f.autoCheck)
}

// newWordsTab holds where new words come from, the JLPT level (for JLPT
// only), the package size and a summary of the current source.
func (f *settingsForm) newWordsTab(st settingsState) fyne.CanvasObject {
	var labels []string
	for _, name := range st.lists {
		labels = append(labels, prettyListName(name))
		f.sourceValues = append(f.sourceValues, listSourcePrefix+name)
	}
	if name, ok := strings.CutPrefix(st.wordSource, listSourcePrefix); ok && !slices.Contains(st.lists, name) {
		// A missing list stays selected until the learner picks another.
		labels = append(labels, prettyListName(name))
		f.sourceValues = append(f.sourceValues, st.wordSource)
	}
	labels = append(labels, "JLPT")
	f.sourceValues = append(f.sourceValues, jlptSource)

	f.wordLevel = widget.NewRadioGroup([]string{"N5", "N4", "N3", "N2", "N1"}, nil)
	f.wordLevel.Horizontal = true
	f.wordLevel.Required = true
	f.wordLevel.SetSelected(fmt.Sprintf("N%d", st.wordLevel))

	f.sourceSelect = widget.NewSelect(labels, func(string) {
		if f.wordSource() == jlptSource {
			f.wordLevel.Enable()
		} else {
			f.wordLevel.Disable()
		}
	})
	if i := slices.Index(f.sourceValues, st.wordSource); i >= 0 {
		f.sourceSelect.SetSelectedIndex(i)
	} else {
		f.sourceSelect.SetSelectedIndex(len(labels) - 1)
	}

	sizes := make([]string, len(packageSizes))
	for i, n := range packageSizes {
		sizes[i] = strconv.Itoa(n)
	}
	f.packageSize = widget.NewRadioGroup(sizes, nil)
	f.packageSize.Horizontal = true
	f.packageSize.Required = true
	f.packageSize.SetSelected(strconv.Itoa(st.packageSize))

	f.summary = widget.NewLabel(st.summary)
	return container.NewVBox(
		widget.NewLabelWithStyle("New words from", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.sourceSelect,
		widget.NewLabel("JLPT level"),
		f.wordLevel,
		widget.NewLabel("Words per package"),
		f.packageSize,
		widget.NewSeparator(),
		f.summary,
	)
}

// newGameTab holds the difficulty, points per hit and the score limit.
func (f *settingsForm) newGameTab(st settingsState) fyne.CanvasObject {
	f.limitEntry = widget.NewEntry()
	f.limitEntry.SetText(strconv.Itoa(st.limit))
	f.limitEntry.Validator = func(s string) error {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 0 {
			return fmt.Errorf("enter a whole number ≥ 0")
		}
		return nil
	}

	levelLabels := make([]string, 0, len(Levels()))
	for _, l := range Levels() {
		levelLabels = append(levelLabels, l.Label)
	}
	f.difficulty = widget.NewRadioGroup(levelLabels, nil)
	f.difficulty.Horizontal = true
	f.difficulty.Required = true
	f.difficulty.SetSelected(LevelByName(string(st.level)).Label)

	pointsLabels := make([]string, 0, len(pointsChoices))
	for _, p := range pointsChoices {
		pointsLabels = append(pointsLabels, pointsLabel(p))
	}
	f.points = widget.NewSelect(pointsLabels, nil)
	f.points.SetSelected(pointsLabel(st.points))

	return container.NewVBox(
		widget.NewLabelWithStyle("Difficulty", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.difficulty,
		container.NewHBox(widget.NewLabel("Points per hit"), f.points),
		widget.NewSeparator(),
		widget.NewLabel("Score limit (0 = endless)"),
		f.limitEntry,
	)
}

// wordSource is the word source the Words tab has selected.
func (f *settingsForm) wordSource() string {
	if i := f.sourceSelect.SelectedIndex(); i >= 0 {
		return f.sourceValues[i]
	}
	return jlptSource
}

// words returns the Words tab's values.
func (f *settingsForm) words() wordSettings {
	level, _ := strconv.Atoi(strings.TrimPrefix(f.wordLevel.Selected, "N"))
	size, _ := strconv.Atoi(f.packageSize.Selected)
	return wordSettings{source: f.wordSource(), level: level, packageSize: size}
}

// validate checks every tab; on an error it switches to the tab that has it.
func (f *settingsForm) validate() error {
	if err := f.limitEntry.Validate(); err != nil {
		f.tabs.Select(f.gameTab)
		return err
	}
	return nil
}

// apply saves every tab: the Words tab first, then the rest, which ends by
// checking whether a word package is due.
func (f *settingsForm) apply(gs *GameState) {
	limit := f.limit
	if n, err := strconv.Atoi(strings.TrimSpace(f.limitEntry.Text)); err == nil && n >= 0 {
		limit = n
	}
	gs.applyWordSettings(f.words())
	gs.applySettings(checkedRowIDs(f.rowChecks), checkedScripts(f.scriptChecks), f.autoCheck.Checked, limit,
		levelFromLabel(f.difficulty.Selected), pointsFromLabel(f.points.Selected))
}

func showSettingsDialog(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, win fyne.Window) {
	form := newSettingsForm(gs.settingsState())
	gs.SetSettingsOpen(true)
	gameCanvas.Refresh()

	d := dialog.NewCustomWithoutButtons("Settings", form.tabs, win)
	// closed releases the settings pause on every path; showPendingDialogs
	// then resumes the game or shows whatever is pending.
	closed := func(save bool) {
		d.Hide()
		gs.SetSettingsOpen(false)
		if save {
			form.apply(gs)
			gs.mu.Lock()
			snap := gs.snapshot()
			gs.mu.Unlock()
			statsPanel.Update(snap)
			inputBar.Update(snap)
		}
		gameCanvas.Refresh()
		showPendingDialogs(gs, statsPanel, inputBar, win)
	}
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { closed(false) })
	save := widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
		// An invalid value keeps the dialog open on its tab.
		if err := form.validate(); err != nil {
			dialog.ShowError(err, win)
			return
		}
		closed(true)
	})
	save.Importance = widget.HighImportance
	d.SetButtons([]fyne.CanvasObject{cancel, save})
	d.Show()
}

// newGroupChecks builds the per-row checks and the "all" check for one kana
// group, keeping them in sync: checking/unchecking "all" sets every row, and
// checking/unchecking a row updates "all" to match. A per-group syncing flag
// guards against the two handlers recursing into each other.
func newGroupChecks(rows []kanacore.KanaRow, selected map[string]bool) (all *widget.Check, checks map[string]*widget.Check, grid *fyne.Container) {
	checks = make(map[string]*widget.Check, len(rows))
	checkList := make([]*widget.Check, 0, len(rows))
	grid = container.NewGridWithColumns(2)

	for _, row := range rows {
		c := widget.NewCheck(row.Label, nil)
		c.SetChecked(selected[row.ID])
		checks[row.ID] = c
		checkList = append(checkList, c)
		grid.Add(c)
	}

	all = widget.NewCheck("all", nil)
	all.SetChecked(allChecked(checkList))

	syncing := false

	all.OnChanged = func(on bool) {
		if syncing {
			return
		}
		syncing = true
		for _, c := range checkList {
			c.SetChecked(on)
		}
		syncing = false
	}

	for _, c := range checkList {
		c.OnChanged = func(bool) {
			if syncing {
				return
			}
			syncing = true
			all.SetChecked(allChecked(checkList))
			syncing = false
		}
	}

	return all, checks, grid
}

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

// checkedScripts returns the checked scripts in display order.
func checkedScripts(checks map[kanacore.Script]*widget.Check) []kanacore.Script {
	var scripts []kanacore.Script
	for _, info := range kanacore.Scripts() {
		if c, ok := checks[info.Script]; ok && c.Checked {
			scripts = append(scripts, info.Script)
		}
	}
	return scripts
}

func allChecked(checks []*widget.Check) bool {
	for _, c := range checks {
		if !c.Checked {
			return false
		}
	}
	return len(checks) > 0
}

// levelFromLabel maps a difficulty label back to its level; unknown → Normal.
func levelFromLabel(label string) Level {
	for _, l := range Levels() {
		if l.Label == label {
			return l.Level
		}
	}
	return LevelNormal
}

// pointsFromLabel maps a "Points per hit" label back to its setting.
func pointsFromLabel(label string) int {
	for _, p := range pointsChoices {
		if pointsLabel(p) == label {
			return p
		}
	}
	return pointsAuto
}
```

- [ ] **Step 4: The Sources tab in `fyne/version.go`**

Replace:

```go
	return "Kana " + versionLabel(meta)
}

// versionRow is the settings dialog's "Version v0.2.0 · Releases" line.
func versionRow() fyne.CanvasObject {
	u, _ := url.Parse(releasesURL)
	return container.NewHBox(
```

with:

```go
	return "Kana " + versionLabel(meta)
}

// versionRow is the Sources tab's "Version v0.2.0 · Releases" line.
func versionRow() fyne.CanvasObject {
	u, _ := url.Parse(releasesURL)
	return container.NewHBox(
```

Replace:

```go
		widget.NewHyperlink("Releases", u),
	)
}
```

with:

```go
		widget.NewHyperlink("Releases", u),
	)
}

// sourceLink is a link on the Sources tab.
type sourceLink struct{ label, url string }

// dataSource is one source the app uses, with its licence and links.
type dataSource struct {
	name   string
	detail string
	links  []sourceLink
}

// dataSources are credited on the Sources tab.
var dataSources = []dataSource{
	{"JMdict", "© Electronic Dictionary Research and Development Group (EDRDG), CC BY-SA 4.0. Word meanings.", []sourceLink{
		{"Licence", "https://www.edrdg.org/edrdg/licence.html"},
		{"JMdict", "https://www.edrdg.org/jmdict/j_jmdict.html"},
	}},
	{"JLPT levels", "Jonathan Waller, CC BY. Unofficial: there are no official JLPT word lists since 2010.", []sourceLink{
		{"tanos.co.uk", "http://www.tanos.co.uk/jlpt/"},
	}},
	{"JMdict mapping", "stephenmk/yomitan-jlpt-vocab, CC BY-SA 4.0. Matches the JLPT lists to JMdict.", []sourceLink{
		{"GitHub", "https://github.com/stephenmk/yomitan-jlpt-vocab"},
	}},
}

// newSourcesTab shows the version and credits every data source.
func newSourcesTab() fyne.CanvasObject {
	box := container.NewVBox(versionRow(), widget.NewSeparator())
	for _, s := range dataSources {
		detail := widget.NewLabel(s.detail)
		detail.Wrapping = fyne.TextWrapWord
		links := container.NewHBox()
		for _, l := range s.links {
			u, _ := url.Parse(l.url)
			links.Add(widget.NewHyperlink(l.label, u))
		}
		box.Add(widget.NewLabelWithStyle(s.name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		box.Add(detail)
		box.Add(links)
	}
	box.Add(widget.NewLabel("The word data is licensed under CC BY-SA 4.0; the app's code under MIT."))
	scroll := container.NewVScroll(box)
	scroll.SetMinSize(fyne.NewSize(460, 300))
	return scroll
}
```

- [ ] **Step 5: Counts for the Words tab, attribution out of the panel**

`fyne/wordgame.go`:

Replace:

```go
package main

import (
	"os"
	"sort"
	"strings"
```

with:

```go
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
```

Replace:

```go
	return n
}

// wordsStatus summarises the words for the status panel. Must be called
// under lock.
func (gs *GameState) wordsStatus() WordsStatus {
```

with:

```go
	return n
}

// introducedCounts counts the introduced readings in the word index and
// those of them that are secure. Must be called under lock.
func (gs *GameState) introducedCounts() (introduced, secure int) {
	for kana := range gs.introduced {
		if _, ok := gs.words.Lookup(kana); !ok {
			continue
		}
		introduced++
		if gs.wordCorrect(kana) >= secureCorrect {
			secure++
		}
	}
	return introduced, secure
}

// wordsSummary is the settings' Words tab line: "Introduced: 25 · secure:
// 18 · left in source: 101", or that the chosen list is missing. Must be
// called under lock.
func (gs *GameState) wordsSummary() string {
	src, found := gs.source()
	if !found {
		return fmt.Sprintf("List %s not found", prettyListName(src.List))
	}
	if gs.words == nil {
		return "No words available"
	}
	introduced, secure := gs.introducedCounts()
	return fmt.Sprintf("Introduced: %d · secure: %d · left in source: %d", introduced, secure, gs.sourceLeft(src))
}

// wordsStatus summarises the words for the status panel. Must be called
// under lock.
func (gs *GameState) wordsStatus() WordsStatus {
```

Replace:

```go
	if gs.words == nil {
		return ws
	}
	for kana := range gs.introduced {
		if _, ok := gs.words.Lookup(kana); !ok {
			continue
		}
		ws.Introduced++
		if gs.wordCorrect(kana) >= secureCorrect {
			ws.Secure++
		}
	}
	if found && !ws.LoadFailed && len(vocab.NextPackage(gs.words, src, gs.introduced, 1, gs.unitKnown)) == 0 {
		ws.Next = wordsFinished
		if gs.sourceLeft(src) > 0 {
```

with:

```go
	if gs.words == nil {
		return ws
	}
	ws.Introduced, ws.Secure = gs.introducedCounts()
	if found && !ws.LoadFailed && len(vocab.NextPackage(gs.words, src, gs.introduced, 1, gs.unitKnown)) == 0 {
		ws.Next = wordsFinished
		if gs.sourceLeft(src) > 0 {
```

`fyne/stats.go`:

Replace:

```go

	correctText, missedText, accuracyText *canvas.Text

	wordsText   *canvas.Text   // "Words: … introduced · … secure · JLPT N5", or a hint
	sourceTexts []*canvas.Text // word data attribution

	pathLines map[kanacore.Script]*fyne.Container
	pathSegs  map[kanacore.Script][]*canvas.Rectangle
```

with:

```go

	correctText, missedText, accuracyText *canvas.Text

	wordsText *canvas.Text // "Words: … introduced · … secure · JLPT N5", or a hint

	pathLines map[kanacore.Script]*fyne.Container
	pathSegs  map[kanacore.Script][]*canvas.Rectangle
```

Replace:

```go
	// the grid is 288px wide, so the padded content is 292px, exactly
	// matching the panel's 300-2*theme.Padding() width.
	padded := container.New(layout.NewCustomPaddedLayout(0, 0, 0, 4), grids)
	source := container.NewVBox()
	for _, line := range wordDataSource {
		txt := smallText(line, 10)
		txt.Color = panelSubColor
		p.sourceTexts = append(p.sourceTexts, txt)
		source.Add(txt)
	}
	p.content = container.NewBorder(top, container.NewVBox(legend, source), nil, nil, container.NewVScroll(padded))
	p.ExtendBaseWidget(p)
	return p
}
```

with:

```go
	// the grid is 288px wide, so the padded content is 292px, exactly
	// matching the panel's 300-2*theme.Padding() width.
	padded := container.New(layout.NewCustomPaddedLayout(0, 0, 0, 4), grids)
	p.content = container.NewBorder(top, legend, nil, nil, container.NewVScroll(padded))
	p.ExtendBaseWidget(p)
	return p
}
```

`fyne/words.go`:

Replace:

```go
// fits the line to its width (see StatsPanel.Update).
const maxWordsLineRunes = 200

// wordDataSource attributes the bundled word data; two lines so it fits the
// status panel. Stage 4 moves it to the settings' Sources tab.
var wordDataSource = []string{"Word data: JMdict (EDRDG, CC BY-SA 4.0),", "JLPT levels by J. Waller"}

// Word sources as stored in the word_source setting.
const (
	jlptSource       = "jlpt"  // the bundled JLPT words
```

with:

```go
// fits the line to its width (see StatsPanel.Update).
const maxWordsLineRunes = 200

// Word sources as stored in the word_source setting.
const (
	jlptSource       = "jlpt"  // the bundled JLPT words
```

- [ ] **Step 6: Run**

Run: `go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`.

- [ ] **Step 7: Commit**

```bash
git add fyne/settings.go fyne/version.go fyne/wordgame.go fyne/stats.go fyne/words.go fyne/settings_test.go fyne/stats_test.go
git commit -m "feat(fyne): split the settings into tabs and add a Sources tab

The settings dialog has the tabs Kana, Words, Game and Sources. The
Words tab chooses where new words come from, the JLPT level and the
package size; an invalid value keeps the dialog open on its tab. The
data attribution moves from the status panel to the Sources tab."
```

---

### Task 8: Kana font

**Files:** Create `scripts/fetch-fonts.sh`, `fyne/fonts/` (six files), `fyne/fonts.go`, `fyne/fonts_test.go`. Modify `fyne/tile.go`, `fyne/feedback.go`, `fyne/intro.go`, `fyne/packageintro.go`, `fyne/stats.go`, `fyne/app.go`, `fyne/game.go`, `fyne/settings.go`, `fyne/version.go`, `go.mod`, and the tests `fyne/game_test.go`, `fyne/tile_test.go`, `fyne/wordgame_test.go`, `fyne/packageintro_test.go`, `fyne/settings_test.go`.

- [ ] **Step 1: Create `scripts/fetch-fonts.sh` and fetch the fonts**

```sh
#!/bin/sh
# Downloads the kana fonts that ship with the app into fyne/fonts/ and checks
# each file against the SHA-256 sum recorded here. Run from the repository
# root. The files are committed; run this only to restore or verify them, or
# after changing a pinned version (then update the sums below).
#
#   Noto Sans JP:      notofonts/noto-cjk, tag Sans2.004, Sans/SubsetOTF/JP
#   M PLUS Rounded 1c: google/fonts, commit 84efd8ad78c3, ofl/mplusrounded1c
#                      (its OFL.txt lives in ofl/roundedmplus1c)
# Both are licensed under the SIL Open Font License 1.1.
set -eu

NOTO=https://raw.githubusercontent.com/notofonts/noto-cjk/Sans2.004
GFONTS=https://raw.githubusercontent.com/google/fonts/84efd8ad78c3710ad14bd909e3bc407151885628/ofl
DIR=fyne/fonts

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	else
		shasum -a 256 "$1" | cut -d ' ' -f 1
	fi
}

# fetch URL FILE SHA256 downloads URL to $DIR/FILE if the sum matches.
fetch() {
	tmp="$DIR/$2.tmp"
	curl -fsSL -o "$tmp" "$1"
	got=$(sha256 "$tmp")
	if [ "$got" != "$3" ]; then
		rm -f "$tmp"
		echo "$2: SHA-256 $got, expected $3" >&2
		exit 1
	fi
	mv "$tmp" "$DIR/$2"
	echo "$2 ok"
}

mkdir -p "$DIR"
fetch "$NOTO/Sans/SubsetOTF/JP/NotoSansJP-Regular.otf" NotoSansJP-Regular.otf \
	dff723ba59d57d136764a04b9b2d03205544f7cd785a711442d6d2d085ac5073
fetch "$NOTO/Sans/SubsetOTF/JP/NotoSansJP-Bold.otf" NotoSansJP-Bold.otf \
	1b0edfb500b73a4fa8a4fcaae1bbbd403994e08e73e3e0da37e70d3853f42c5f
fetch "$NOTO/LICENSE" OFL-NotoSansCJK.txt \
	6a73f9541c2de74158c0e7cf6b0a58ef774f5a780bf191f2d7ec9cc53efe2bf2
fetch "$GFONTS/mplusrounded1c/MPLUSRounded1c-Regular.ttf" MPLUSRounded1c-Regular.ttf \
	b75708b53e45b06d17d470aeeca5b766e3d1b3999f03f13ec4eb863ca846c14c
fetch "$GFONTS/mplusrounded1c/MPLUSRounded1c-Bold.ttf" MPLUSRounded1c-Bold.ttf \
	c358630584e8e2d8fbd6121d0f4693255ffef6d1e6d4f3441fd6e5a963a11f9e
fetch "$GFONTS/roundedmplus1c/OFL.txt" OFL-MPLUSRounded1c.txt \
	67f64c5509e5151796599e3ad47c3131cbe0c80c4f9430b90236a1249c2eacc9
```

```bash
chmod +x scripts/fetch-fonts.sh
scripts/fetch-fonts.sh
ls -l fyne/fonts
```

Expected: six `… ok` lines; `fyne/fonts` holds `NotoSansJP-Regular.otf` (4,533,028 bytes), `NotoSansJP-Bold.otf` (4,656,448), `MPLUSRounded1c-Regular.ttf` (3,389,792), `MPLUSRounded1c-Bold.ttf` (3,542,592), `OFL-NotoSansCJK.txt` (4,301) and `OFL-MPLUSRounded1c.txt` (4,397), about 16 MB in total.

| File | URL | SHA-256 |
|---|---|---|
| `NotoSansJP-Regular.otf` | `https://raw.githubusercontent.com/notofonts/noto-cjk/Sans2.004/Sans/SubsetOTF/JP/NotoSansJP-Regular.otf` | `dff723ba59d57d136764a04b9b2d03205544f7cd785a711442d6d2d085ac5073` |
| `NotoSansJP-Bold.otf` | `https://raw.githubusercontent.com/notofonts/noto-cjk/Sans2.004/Sans/SubsetOTF/JP/NotoSansJP-Bold.otf` | `1b0edfb500b73a4fa8a4fcaae1bbbd403994e08e73e3e0da37e70d3853f42c5f` |
| `OFL-NotoSansCJK.txt` | `https://raw.githubusercontent.com/notofonts/noto-cjk/Sans2.004/LICENSE` | `6a73f9541c2de74158c0e7cf6b0a58ef774f5a780bf191f2d7ec9cc53efe2bf2` |
| `MPLUSRounded1c-Regular.ttf` | `https://raw.githubusercontent.com/google/fonts/84efd8ad78c3710ad14bd909e3bc407151885628/ofl/mplusrounded1c/MPLUSRounded1c-Regular.ttf` | `b75708b53e45b06d17d470aeeca5b766e3d1b3999f03f13ec4eb863ca846c14c` |
| `MPLUSRounded1c-Bold.ttf` | `https://raw.githubusercontent.com/google/fonts/84efd8ad78c3710ad14bd909e3bc407151885628/ofl/mplusrounded1c/MPLUSRounded1c-Bold.ttf` | `c358630584e8e2d8fbd6121d0f4693255ffef6d1e6d4f3441fd6e5a963a11f9e` |
| `OFL-MPLUSRounded1c.txt` | `https://raw.githubusercontent.com/google/fonts/84efd8ad78c3710ad14bd909e3bc407151885628/ofl/roundedmplus1c/OFL.txt` | `67f64c5509e5151796599e3ad47c3131cbe0c80c4f9430b90236a1249c2eacc9` |

- [ ] **Step 2: Write the failing tests** — create `fyne/fonts_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/vocab"
)

// thai is a character neither bundled font has (checked against the pinned
// files), so a text with it falls back to the theme font.
const thai = "ก"

func TestFontResourcesLoad(t *testing.T) {
	names := map[string]bool{}
	for _, res := range []fyne.Resource{notoRegular, notoBold, mplusRegular, mplusBold} {
		if len(res.Content()) < 1_000_000 {
			t.Fatalf("%s: %d bytes, want the full font file", res.Name(), len(res.Content()))
		}
		if fontFaces()[res] == nil {
			t.Fatalf("%s does not parse", res.Name())
		}
		if !covers(res, "あいうえおアイウエオきゃキャっッーヴ…") {
			t.Fatalf("%s lacks kana", res.Name())
		}
		names[res.Name()] = true
	}
	if len(names) != 4 {
		t.Fatalf("resource names are not distinct: %v", names)
	}
}

func TestFontChoiceCoverage(t *testing.T) {
	noto, mplus, system := fontByID("noto"), fontByID("mplus"), fontByID("system")
	if noto.source("あ", false) != notoRegular || noto.source("あ", true) != notoBold {
		t.Fatal("Noto Sans JP: regular and bold files by style")
	}
	if mplus.source("ア — blue", false) != mplusRegular || mplus.source("ア", true) != mplusBold {
		t.Fatal("M PLUS Rounded 1c: regular and bold files by style")
	}
	for _, c := range []fontChoice{noto, mplus} {
		if src := c.source("あ"+thai, false); src != nil {
			t.Fatalf("%s: a text with a missing rune must use the theme font, got %v", c.id, src.Name())
		}
	}
	if system.source("あ", false) != nil || system.source("あ", true) != nil {
		t.Fatal("System uses the theme font")
	}
	if fontByID("bogus").id != "noto" || fontByID("").id != "noto" {
		t.Fatal("an unknown setting gives Noto Sans JP")
	}
}

func TestTilesUseTheKanaFontAndAreSizedWithIt(t *testing.T) {
	test.NewApp()
	noto := fontByID("noto")
	k := newKanaTile(kanacore.Kana{Char: "き", Romaji: "ki"}, noto)
	if k.text.FontSource != notoRegular || k.text.Size() != k.text.MinSize() {
		t.Fatalf("kana tile: source %v, size %v, min size with the font %v", k.text.FontSource, k.text.Size(), k.text.MinSize())
	}
	w := newWordTile(vocab.Entry{Kana: "コンピューター"}, 150, noto)
	if w.text.FontSource != notoRegular {
		t.Fatal("word tile not in the kana font")
	}
	if got, want := w.Width(), w.text.MinSize().Width+2*wordTilePad; got != want || w.Width() > 150 {
		t.Fatalf("word tile width %v, want %v (text measured in its font) and at most 150", got, want)
	}
	if s := newKanaTile(kanacore.Kana{Char: "き"}, fontByID("system")); s.text.FontSource != nil {
		t.Fatal("System: no font source")
	}
}

func TestGameUsesTheChosenFont(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.SetKanaFont("mplus")
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].text.FontSource != mplusRegular {
		t.Fatal("a new tile must use the chosen font")
	}
	gs.SetKanaFont("noto")
	if gs.tiles[0].text.FontSource != mplusRegular {
		t.Fatal("a falling tile keeps its font")
	}
	gs.addFeedback(feedbackHit, "あい — love", fyne.NewPos(10, 10))
	if gs.feedback[0].text.FontSource != notoBold {
		t.Fatalf("feedback label: %v, want the bold kana font", gs.feedback[0].text.FontSource)
	}
	gs.addFeedback(feedbackHit, "あい — "+thai, fyne.NewPos(10, 10))
	if gs.feedback[1].text.FontSource != nil {
		t.Fatal("a label with a missing rune keeps the theme font")
	}
}

func TestFeedbackFontCheckedOnTheFinalText(t *testing.T) {
	gs := newTestState()
	gs.canvasW = 300
	gs.kanaFont = fontByID("noto")
	gs.addFeedback(feedbackMiss, "あいうえお · aiueo — "+strings.Repeat("a long meaning; ", 6)+thai, fyne.NewPos(0, 0))
	f := gs.feedback[0].text
	if !strings.HasSuffix(f.Text, "…") || strings.Contains(f.Text, thai) {
		t.Fatalf("precondition: the label is cut before the Thai rune: %q", f.Text)
	}
	if f.FontSource != notoBold {
		t.Fatal("the cut text has no missing rune, so it uses the kana font")
	}
	if w := f.MinSize().Width; w > 300 {
		t.Fatalf("label %v px wide, more than the field", w)
	}
}

func TestIntroAndPackageUseTheKanaFont(t *testing.T) {
	test.NewApp()
	mplus := fontByID("mplus")
	card := newIntroCard(kanacore.Entry{Char: "か", Romaji: "ka"}, mplus)
	found := false
	for _, o := range test.LaidOutObjects(card) {
		if txt, ok := o.(*canvas.Text); ok && txt.Text == "か" {
			found = txt.FontSource == mplusRegular
		}
	}
	if !found {
		t.Fatal("the intro card's kana must use the kana font")
	}
	v := newPackageView([]vocab.Entry{{Kana: "かさ", Meanings: vocab.Meanings{}}}, "en", mplus, func() (vocab.Entry, bool) { return vocab.Entry{}, false })
	if v.rows[0].kana.FontSource != mplusRegular {
		t.Fatal("the package intro's kana must use the kana font")
	}
}

func TestStatsPanelKanaFont(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.SetKanaFont(fontByID("noto"))
	if p.cells["あ"].text.FontSource != notoRegular || p.cells["キャ"].text.FontSource != notoRegular {
		t.Fatal("grid cells must use the kana font")
	}
	p.SetKanaFont(fontByID("system"))
	if p.cells["あ"].text.FontSource != nil {
		t.Fatal("System: no font source in the grid")
	}
}

func TestKanaFontSetting(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if id := NewGameState(st).KanaFont().id; id != "noto" {
		t.Fatalf("default font = %q, want noto", id)
	}
	gs := NewGameState(st)
	f := newSettingsForm(gs.settingsState())
	if f.kanaFont.Selected != "Noto Sans JP" || len(f.kanaFont.Options) != 3 {
		t.Fatalf("font options %v, selected %q", f.kanaFont.Options, f.kanaFont.Selected)
	}
	f.kanaFont.SetSelected("M PLUS Rounded 1c")
	f.apply(gs)
	if gs.KanaFont().id != "mplus" {
		t.Fatalf("font after Save = %q", gs.KanaFont().id)
	}
	if id, _ := st.KanaFont(); id != "mplus" {
		t.Fatalf("stored font = %q", id)
	}
	if id := NewGameState(st).KanaFont().id; id != "mplus" {
		t.Fatalf("reloaded font = %q", id)
	}
}

func TestGameTabPreviewsEachFont(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(newTestState().settingsState())
	f.tabs.Select(f.gameTab)
	sources := map[fyne.Resource]bool{}
	previews := 0
	for _, o := range test.LaidOutObjects(f.gameTab.Content) {
		if txt, ok := o.(*canvas.Text); ok && txt.Text == fontPreview {
			previews++
			sources[txt.FontSource] = true
		}
	}
	if previews != 3 || !sources[notoRegular] || !sources[mplusRegular] || !sources[nil] {
		t.Fatalf("%d previews with fonts %v; want one per choice", previews, sources)
	}
}
```

The Sources tab also credits the fonts — in `fyne/settings_test.go`:

Replace:

```go
		"JMdict <https://www.edrdg.org/jmdict/j_jmdict.html>",
		"JLPT levels", "Jonathan Waller, CC BY", "Unofficial",
		"JMdict mapping", "stephenmk/yomitan-jlpt-vocab",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("Sources tab lacks %q:\n%s", want, all)
```

with:

```go
		"JMdict <https://www.edrdg.org/jmdict/j_jmdict.html>",
		"JLPT levels", "Jonathan Waller, CC BY", "Unofficial",
		"JMdict mapping", "stephenmk/yomitan-jlpt-vocab",
		"Noto Sans JP", "M PLUS Rounded 1c", "SIL Open Font License 1.1",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("Sources tab lacks %q:\n%s", want, all)
```

The tile constructors get a font parameter; the existing tests pass the theme font (`fontChoice{}`):

```bash
perl -pi -e 's/\bnewKanaTile\(((?:[^()]|\([^()]*\))*)\)/newKanaTile($1, fontChoice{})/g; s/\bnewWordTile\(((?:[^()]|\([^()]*\))*)\)/newWordTile($1, fontChoice{})/g; s/newPackageView\(entries, "en", func\(\)/newPackageView(entries, "en", fontChoice{}, func()/' \
	fyne/game_test.go fyne/tile_test.go fyne/wordgame_test.go fyne/packageintro_test.go
git diff --stat fyne/game_test.go fyne/tile_test.go fyne/wordgame_test.go fyne/packageintro_test.go
```

Expected: 5, 10, 1 and 1 changed lines; for example `tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"}, fontChoice{})`, `word := newWordTile(vocab.Entry{Kana: "か", Units: []string{"か"}}, 400, fontChoice{})`, `tile := newWordTile(e, gs.canvasW, fontChoice{})` and `v := newPackageView(entries, "en", fontChoice{}, func() (vocab.Entry, bool) {`.

- [ ] **Step 3: Run the tests to see them fail**

Run: `go test -count=1 ./fyne/`
Expected: FAIL — `undefined: notoRegular`, `undefined: fontFaces`, `undefined: covers`, `undefined: fontByID`, `undefined: fontChoice`.

- [ ] **Step 4: Create `fyne/fonts.go` and make the typesetting module direct**

```go
package main

import (
	"bytes"
	_ "embed"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/go-text/typesetting/font"
)

// The kana fonts ship with the app (SIL OFL 1.1, see fonts/OFL-*.txt);
// scripts/fetch-fonts.sh downloads and checks them.

//go:embed fonts/NotoSansJP-Regular.otf
var notoRegularData []byte

//go:embed fonts/NotoSansJP-Bold.otf
var notoBoldData []byte

//go:embed fonts/MPLUSRounded1c-Regular.ttf
var mplusRegularData []byte

//go:embed fonts/MPLUSRounded1c-Bold.ttf
var mplusBoldData []byte

// The font resources are created once, with distinct names: Fyne caches a
// custom font by its resource and never evicts it.
var (
	notoRegular  fyne.Resource = fyne.NewStaticResource("NotoSansJP-Regular.otf", notoRegularData)
	notoBold     fyne.Resource = fyne.NewStaticResource("NotoSansJP-Bold.otf", notoBoldData)
	mplusRegular fyne.Resource = fyne.NewStaticResource("MPLUSRounded1c-Regular.ttf", mplusRegularData)
	mplusBold    fyne.Resource = fyne.NewStaticResource("MPLUSRounded1c-Bold.ttf", mplusBoldData)
)

// fontChoice is a kana font the learner can pick. Without resources it is
// the theme font with the system fallback.
type fontChoice struct {
	id      string // the kana_font setting
	label   string
	regular fyne.Resource
	bold    fyne.Resource
}

// fontChoices are the kana fonts in the order the settings show them.
var fontChoices = []fontChoice{
	{id: "noto", label: "Noto Sans JP", regular: notoRegular, bold: notoBold},
	{id: "mplus", label: "M PLUS Rounded 1c", regular: mplusRegular, bold: mplusBold},
	{id: "system", label: "System"},
}

// fontPreview is the text each choice is previewed with.
const fontPreview = "あ ア き キ ー"

// fontByID returns the choice for a kana_font setting; an unknown value
// gives Noto Sans JP, the default.
func fontByID(id string) fontChoice {
	for _, c := range fontChoices {
		if c.id == id {
			return c
		}
	}
	return fontChoices[0]
}

// fontFaces parses each bundled font once, for the coverage check.
var fontFaces = sync.OnceValue(func() map[fyne.Resource]*font.Font {
	out := map[fyne.Resource]*font.Font{}
	for _, res := range []fyne.Resource{notoRegular, notoBold, mplusRegular, mplusBold} {
		if face, err := font.ParseTTF(bytes.NewReader(res.Content())); err == nil {
			out[res] = face.Font
		}
	}
	return out
})

// covers reports whether res has a glyph for every rune of text.
func covers(res fyne.Resource, text string) bool {
	f := fontFaces()[res]
	if f == nil {
		return false
	}
	for _, r := range text {
		if _, ok := f.NominalGlyph(r); !ok {
			return false
		}
	}
	return true
}

// source returns the font file for text: the bold or regular one if it has
// every rune of text, else nil (the theme font, which falls back to system
// fonts rune by rune). So a text never mixes fonts.
func (c fontChoice) source(text string, bold bool) fyne.Resource {
	res := c.regular
	if bold {
		res = c.bold
	}
	if res == nil || !covers(res, text) {
		return nil
	}
	return res
}

// setKanaFont sets txt's font source for its text and style. Call it
// before measuring txt: MinSize uses the font source, fyne.MeasureText
// does not.
func setKanaFont(txt *canvas.Text, c fontChoice) {
	txt.FontSource = c.source(txt.Text, txt.TextStyle.Bold)
}

// fitKanaText sets txt's kana font and fits it into maxW (see fitText). The
// font is checked again on the final text, which may end in "…".
func fitKanaText(txt *canvas.Text, c fontChoice, maxW, minSize float32) {
	setKanaFont(txt, c)
	fitText(txt, maxW, minSize)
	if src := c.source(txt.Text, txt.TextStyle.Bold); src != txt.FontSource {
		txt.FontSource = src
		fitText(txt, maxW, minSize)
	}
}
```

```bash
go mod tidy
git diff go.mod
```

Expected: `github.com/go-text/typesetting v0.3.3` moves from the indirect block into the first `require` block; `go.sum` is unchanged.

- [ ] **Step 5: Tiles and feedback labels**

`fyne/tile.go`:

Replace:

```go
	fallSeconds float32 // time to cross the canvas; see fallStep
}

func newKanaTile(k kanacore.Kana) *KanaTile {
	w := tileWidthFor(k.Char)

	shadow := canvas.NewRectangle(tileShadowColor)
```

with:

```go
	fallSeconds float32 // time to cross the canvas; see fallStep
}

// newKanaTile builds a kana tile in the kana font kf.
func newKanaTile(k kanacore.Kana, kf fontChoice) *KanaTile {
	w := tileWidthFor(k.Char)

	shadow := canvas.NewRectangle(tileShadowColor)
```

Replace:

```go

	text := canvas.NewText(k.Char, kanaTextColor)
	text.TextSize = 32
	text.Resize(fyne.MeasureText(text.Text, text.TextSize, text.TextStyle))

	t := &KanaTile{kana: k, width: w, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// newWordTile builds a word tile: a sage face with a border, as wide as the
// word plus padding, so even a one-unit word (か) does not look like a kana
// tile. A word wider than maxW (the field width) gets a smaller font, down
// to minWordTextSize, so the tile fits.
func newWordTile(e vocab.Entry, maxW float32) *KanaTile {
	text := canvas.NewText(e.Kana, kanaTextColor)
	text.TextSize = wordTextSize
	size := fyne.MeasureText(text.Text, text.TextSize, text.TextStyle)
	avail := maxW - 2*wordTilePad
	for size.Width > avail && text.TextSize > minWordTextSize {
		text.TextSize--
		size = fyne.MeasureText(text.Text, text.TextSize, text.TextStyle)
	}
	text.Resize(size)
	w := size.Width + 2*wordTilePad
```

with:

```go

	text := canvas.NewText(k.Char, kanaTextColor)
	text.TextSize = 32
	setKanaFont(text, kf)
	text.Resize(text.MinSize())

	t := &KanaTile{kana: k, width: w, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// newWordTile builds a word tile in the kana font kf: a sage face with a
// border, as wide as the word plus padding, so even a one-unit word (か)
// does not look like a kana tile. A word wider than maxW (the field width)
// gets a smaller font, down to minWordTextSize, so the tile fits.
func newWordTile(e vocab.Entry, maxW float32, kf fontChoice) *KanaTile {
	text := canvas.NewText(e.Kana, kanaTextColor)
	text.TextSize = wordTextSize
	setKanaFont(text, kf)
	size := text.MinSize()
	avail := maxW - 2*wordTilePad
	for size.Width > avail && text.TextSize > minWordTextSize {
		text.TextSize--
		size = text.MinSize()
	}
	text.Resize(size)
	w := size.Width + 2*wordTilePad
```

`fyne/feedback.go`:

Replace:

```go
// addFeedback shows msg for feedbackTicks ticks. A hit label sits at pos
// (the tile's last position), a miss label at the bottom of the field in the
// error colour; both are kept inside the field. A label wider than the field
// shrinks down to feedbackMinTextSize and is then cut with "…". Must be
// called under lock.
func (gs *GameState) addFeedback(kind feedbackKind, msg string, pos fyne.Position) {
	var c color.Color = feedbackHitColor
	if kind == feedbackMiss {
```

with:

```go
// addFeedback shows msg for feedbackTicks ticks. A hit label sits at pos
// (the tile's last position), a miss label at the bottom of the field in the
// error colour; both are kept inside the field. A label wider than the field
// shrinks down to feedbackMinTextSize and is then cut with "…". It is shown
// in the kana font if that has every character. Must be called under lock.
func (gs *GameState) addFeedback(kind feedbackKind, msg string, pos fyne.Position) {
	var c color.Color = feedbackHitColor
	if kind == feedbackMiss {
```

Replace:

```go
	txt := canvas.NewText(msg, c)
	txt.TextSize = feedbackTextSize
	txt.TextStyle = fyne.TextStyle{Bold: true}
	fitText(txt, gs.canvasW, feedbackMinTextSize)
	size := txt.MinSize()
	txt.Resize(size)
	y := pos.Y
```

with:

```go
	txt := canvas.NewText(msg, c)
	txt.TextSize = feedbackTextSize
	txt.TextStyle = fyne.TextStyle{Bold: true}
	fitKanaText(txt, gs.kanaFont, gs.canvasW, feedbackMinTextSize)
	size := txt.MinSize()
	txt.Resize(size)
	y := pos.Y
```

- [ ] **Step 6: The font setting in `fyne/game.go`**

Replace:

```go
	autoProgress  bool
	activeScripts map[kanacore.Script]bool

	level         LevelInfo // chosen difficulty
	pointsSetting int       // pointsAuto or a fixed factor
	answerStreak  int       // correct answers in a row this game
```

with:

```go
	autoProgress  bool
	activeScripts map[kanacore.Script]bool

	kanaFont fontChoice // the font kana are shown in

	level         LevelInfo // chosen difficulty
	pointsSetting int       // pointsAuto or a fixed factor
	answerStreak  int       // correct answers in a row this game
```

Replace:

```go
		introducedDirty:  map[string]bool{},
		packageSize:      store.DefaultPackageSize,
		wordSource:       jlptSource,
	}

	for _, id := range kanacore.DefaultRowIDs() {
```

with:

```go
		introducedDirty:  map[string]bool{},
		packageSize:      store.DefaultPackageSize,
		wordSource:       jlptSource,
		kanaFont:         fontByID(store.DefaultKanaFont),
	}

	for _, id := range kanacore.DefaultRowIDs() {
```

Replace:

```go
		if src, err := st.WordSource(); err == nil {
			storedSource, sourceLoaded = src, true
		}
		if l, err := st.MeaningLang(); err == nil {
			storedLang = l
		}
```

with:

```go
		if src, err := st.WordSource(); err == nil {
			storedSource, sourceLoaded = src, true
		}
		if id, err := st.KanaFont(); err == nil {
			gs.kanaFont = fontByID(id)
		}
		if l, err := st.MeaningLang(); err == nil {
			storedLang = l
		}
```

Replace:

```go
	base := gs.level.MinFall + rand.Float32()*(gs.level.MaxFall-gs.level.MinFall)
	chars := gs.availableCharacters()
	if entry, ok := gs.pickWord(len(chars) == 0); ok {
		tile = newWordTile(entry, gs.canvasW)
		tile.fallSeconds = base * wordFallFactor(len(entry.Units)) * tempoFactor(gs.tempoStep)
	} else if len(chars) == 0 {
		gs.mu.Unlock()
```

with:

```go
	base := gs.level.MinFall + rand.Float32()*(gs.level.MaxFall-gs.level.MinFall)
	chars := gs.availableCharacters()
	if entry, ok := gs.pickWord(len(chars) == 0); ok {
		tile = newWordTile(entry, gs.canvasW, gs.kanaFont)
		tile.fallSeconds = base * wordFallFactor(len(entry.Units)) * tempoFactor(gs.tempoStep)
	} else if len(chars) == 0 {
		gs.mu.Unlock()
```

Replace:

```go
	} else {
		char := memory.Pick(gs.spawnCandidates(chars), gs.memory, gs.now(), gs.rnd)
		romaji, _ := gs.charSet.GetRomaji(char)
		tile = newKanaTile(kanacore.Kana{Char: char, Romaji: romaji})
		correct := gs.overallStats[char].CorrectCount + gs.sessionStats[char].CorrectCount
		tile.fallSeconds = base * charFallFactor(correct) * tempoFactor(gs.tempoStep)
	}
```

with:

```go
	} else {
		char := memory.Pick(gs.spawnCandidates(chars), gs.memory, gs.now(), gs.rnd)
		romaji, _ := gs.charSet.GetRomaji(char)
		tile = newKanaTile(kanacore.Kana{Char: char, Romaji: romaji}, gs.kanaFont)
		correct := gs.overallStats[char].CorrectCount + gs.sessionStats[char].CorrectCount
		tile.fallSeconds = base * charFallFactor(correct) * tempoFactor(gs.tempoStep)
	}
```

Replace:

```go
	gs.buildSnapshot()
}

// IsUserPaused reports whether the learner has paused the game.
func (gs *GameState) IsUserPaused() bool {
	gs.mu.Lock()
```

with:

```go
	gs.buildSnapshot()
}

// KanaFont returns the kana font in use.
func (gs *GameState) KanaFont() fontChoice {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.kanaFont
}

// SetKanaFont switches the kana font and saves it. New tiles, labels and
// dialogs use it; falling tiles keep theirs.
func (gs *GameState) SetKanaFont(id string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.kanaFont = fontByID(id)
	if gs.store != nil {
		_ = gs.store.SaveKanaFont(gs.kanaFont.id)
	}
}

// IsUserPaused reports whether the learner has paused the game.
func (gs *GameState) IsUserPaused() bool {
	gs.mu.Lock()
```

- [ ] **Step 7: Intros, the package dialog and the kana grid**

`fyne/intro.go`:

Replace:

```go
// showIntroDialog presents newly unlocked rows as large tiles with their
// romaji. Call it through showPendingDialogs. Must run on the Fyne thread.
func showIntroDialog(gs *GameState, rowIDs []string, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	labels := make([]string, 0, len(rowIDs))
	body := container.NewVBox()
	for _, id := range rowIDs {
```

with:

```go
// showIntroDialog presents newly unlocked rows as large tiles with their
// romaji. Call it through showPendingDialogs. Must run on the Fyne thread.
func showIntroDialog(gs *GameState, rowIDs []string, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	kf := gs.KanaFont()
	labels := make([]string, 0, len(rowIDs))
	body := container.NewVBox()
	for _, id := range rowIDs {
```

Replace:

```go
		}
		labels = append(labels, row.Label)
		body.Add(widget.NewLabelWithStyle(row.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		body.Add(introCards(row))
	}

	inputBar.SetEnabled(false, w)
```

with:

```go
		}
		labels = append(labels, row.Label)
		body.Add(widget.NewLabelWithStyle(row.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		body.Add(introCards(row, kf))
	}

	inputBar.SetEnabled(false, w)
```

Replace:

```go
		widget.NewLabel("You know the basic hiragana. Katakana use the same sounds;\nstart with ア イ ウ エ オ?"),
	)
	if row, ok := kanacore.RowByID("kata:vowels"); ok {
		body.Add(introCards(row))
	}

	inputBar.SetEnabled(false, w)
```

with:

```go
		widget.NewLabel("You know the basic hiragana. Katakana use the same sounds;\nstart with ア イ ウ エ オ?"),
	)
	if row, ok := kanacore.RowByID("kata:vowels"); ok {
		body.Add(introCards(row, gs.KanaFont()))
	}

	inputBar.SetEnabled(false, w)
```

Replace:

```go
	d.Show()
}

// introCards lays out a row's kana as large cards.
func introCards(row kanacore.KanaRow) fyne.CanvasObject {
	cards := container.NewHBox()
	for _, e := range row.Entries {
		cards.Add(newIntroCard(e))
	}
	return cards
}

// newIntroCard renders one kana as a large paper tile with its romaji below.
func newIntroCard(e kanacore.Entry) fyne.CanvasObject {
	width := introCardW
	if utf8.RuneCountInString(e.Char) > 1 {
		width = introCardWideW
```

with:

```go
	d.Show()
}

// introCards lays out a row's kana as large cards in the kana font kf.
func introCards(row kanacore.KanaRow, kf fontChoice) fyne.CanvasObject {
	cards := container.NewHBox()
	for _, e := range row.Entries {
		cards.Add(newIntroCard(e, kf))
	}
	return cards
}

// newIntroCard renders one kana as a large paper tile with its romaji below.
func newIntroCard(e kanacore.Entry, kf fontChoice) fyne.CanvasObject {
	width := introCardW
	if utf8.RuneCountInString(e.Char) > 1 {
		width = introCardWideW
```

Replace:

```go

	glyph := canvas.NewText(e.Char, kanaTextColor)
	glyph.TextSize = 64

	romaji := widget.NewLabelWithStyle(e.Romaji, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	return container.NewVBox(container.NewStack(face, container.NewCenter(glyph)), romaji)
```

with:

```go

	glyph := canvas.NewText(e.Char, kanaTextColor)
	glyph.TextSize = 64
	setKanaFont(glyph, kf)

	romaji := widget.NewLabelWithStyle(e.Romaji, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	return container.NewVBox(container.NewStack(face, container.NewCenter(glyph)), romaji)
```

`fyne/packageintro.go`:

Replace:

```go
	list    *fyne.Container
	rows    []*packageRow
	lang    string
	replace func() (vocab.Entry, bool)
}
```

with:

```go
	list    *fyne.Container
	rows    []*packageRow
	lang    string
	font    fontChoice
	replace func() (vocab.Entry, bool)
}
```

Replace:

```go
	replaced bool // a replacement was already asked for
}

func newPackageView(entries []vocab.Entry, lang string, replace func() (vocab.Entry, bool)) *packageView {
	v := &packageView{list: container.NewVBox(), lang: lang, replace: replace}
	for _, e := range entries {
		v.addRow(e)
	}
```

with:

```go
	replaced bool // a replacement was already asked for
}

func newPackageView(entries []vocab.Entry, lang string, kf fontChoice, replace func() (vocab.Entry, bool)) *packageView {
	v := &packageView{list: container.NewVBox(), lang: lang, font: kf, replace: replace}
	for _, e := range entries {
		v.addRow(e)
	}
```

Replace:

```go
	row := &packageRow{entry: e}
	row.kana = canvas.NewText(e.Kana, kanaTextColor)
	row.kana.TextSize = packageKanaSize
	fitText(row.kana, packageKanaW, packageKanaMinSize)
	row.strike = canvas.NewRectangle(kanaTextColor)
	row.strike.Hide()
	word := container.NewGridWrap(fyne.NewSize(packageKanaW, packageRowH), container.New(strikeLayout{}, row.kana, row.strike))
```

with:

```go
	row := &packageRow{entry: e}
	row.kana = canvas.NewText(e.Kana, kanaTextColor)
	row.kana.TextSize = packageKanaSize
	fitKanaText(row.kana, v.font, packageKanaW, packageKanaMinSize)
	row.strike = canvas.NewRectangle(kanaTextColor)
	row.strike.Hide()
	word := container.NewGridWrap(fyne.NewSize(packageKanaW, packageRowH), container.New(strikeLayout{}, row.kana, row.strike))
```

Replace:

```go
// thread.
func showWordPackageDialog(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	intro := gs.PackageIntro()
	view := newPackageView(intro.Entries, intro.Lang, gs.ReplacementWord)
	scroll := container.NewVScroll(view.list)
	scroll.SetMinSize(fyne.NewSize(packageViewW, min(float32(len(intro.Entries))*packageRowH, packageViewMaxH)))
```

with:

```go
// thread.
func showWordPackageDialog(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	intro := gs.PackageIntro()
	view := newPackageView(intro.Entries, intro.Lang, gs.KanaFont(), gs.ReplacementWord)
	scroll := container.NewVScroll(view.list)
	scroll.SetMinSize(fyne.NewSize(packageViewW, min(float32(len(intro.Entries))*packageRowH, packageViewMaxH)))
```

`fyne/stats.go`:

Replace:

```go
	p.content.Refresh()
}

func setVisible(o fyne.CanvasObject, on bool) {
	if on {
		o.Show()
```

with:

```go
	p.content.Refresh()
}

// SetKanaFont shows the kana grid in the kana font kf.
func (p *StatsPanel) SetKanaFont(kf fontChoice) {
	for _, c := range p.cells {
		setKanaFont(c.text, kf)
		c.text.Refresh()
	}
}

func setVisible(o fyne.CanvasObject, on bool) {
	if on {
		o.Show()
```

`fyne/app.go`:

Replace:

```go
	gs := NewGameState(st)

	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)

	inputBar := newInputBar(gs, statsPanel, gameCanvas, w)
```

with:

```go
	gs := NewGameState(st)

	statsPanel := newStatsPanel()
	statsPanel.SetKanaFont(gs.KanaFont())
	gameCanvas := newGameCanvas(gs)

	inputBar := newInputBar(gs, statsPanel, gameCanvas, w)
```

- [ ] **Step 8: The font selector and the font credits**

`fyne/settings.go`:

Replace:

```go
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
```

with:

```go
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
```

Replace:

```go
	wordLevel   int
	packageSize int
	summary     string // the Words tab's summary line
}

// settingsState copies the current settings for the dialog.
```

with:

```go
	wordLevel   int
	packageSize int
	summary     string // the Words tab's summary line
	kanaFont    string // the kana font's id
}

// settingsState copies the current settings for the dialog.
```

Replace:

```go
		wordLevel:   gs.wordLevel,
		packageSize: gs.packageSize,
		summary:     gs.wordsSummary(),
	}
}
```

with:

```go
		wordLevel:   gs.wordLevel,
		packageSize: gs.packageSize,
		summary:     gs.wordsSummary(),
		kanaFont:    gs.kanaFont.id,
	}
}
```

Replace:

```go
	difficulty *widget.RadioGroup
	points     *widget.Select
	limitEntry *widget.Entry
	limit      int // the score limit the dialog started with
}
```

with:

```go
	difficulty *widget.RadioGroup
	points     *widget.Select
	limitEntry *widget.Entry
	kanaFont   *widget.RadioGroup
	limit      int // the score limit the dialog started with
}
```

Replace:

```go
	)
}

// newGameTab holds the difficulty, points per hit and the score limit.
func (f *settingsForm) newGameTab(st settingsState) fyne.CanvasObject {
	f.limitEntry = widget.NewEntry()
	f.limitEntry.SetText(strconv.Itoa(st.limit))
```

with:

```go
	)
}

// newGameTab holds the difficulty, points per hit, the score limit and the
// kana font, each font previewed in itself.
func (f *settingsForm) newGameTab(st settingsState) fyne.CanvasObject {
	f.limitEntry = widget.NewEntry()
	f.limitEntry.SetText(strconv.Itoa(st.limit))
```

Replace:

```go
	f.points = widget.NewSelect(pointsLabels, nil)
	f.points.SetSelected(pointsLabel(st.points))

	return container.NewVBox(
		widget.NewLabelWithStyle("Difficulty", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.difficulty,
```

with:

```go
	f.points = widget.NewSelect(pointsLabels, nil)
	f.points.SetSelected(pointsLabel(st.points))

	fontLabels := make([]string, len(fontChoices))
	previews := container.NewGridWithColumns(2)
	for i, c := range fontChoices {
		fontLabels[i] = c.label
		preview := canvas.NewText(fontPreview, kanaTextColor)
		preview.TextSize = 20
		setKanaFont(preview, c)
		previews.Add(widget.NewLabel(c.label))
		previews.Add(preview)
	}
	f.kanaFont = widget.NewRadioGroup(fontLabels, nil)
	f.kanaFont.Horizontal = true
	f.kanaFont.Required = true
	f.kanaFont.SetSelected(fontByID(st.kanaFont).label)

	return container.NewVBox(
		widget.NewLabelWithStyle("Difficulty", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.difficulty,
```

Replace:

```go
		widget.NewSeparator(),
		widget.NewLabel("Score limit (0 = endless)"),
		f.limitEntry,
	)
}

// wordSource is the word source the Words tab has selected.
func (f *settingsForm) wordSource() string {
	if i := f.sourceSelect.SelectedIndex(); i >= 0 {
```

with:

```go
		widget.NewSeparator(),
		widget.NewLabel("Score limit (0 = endless)"),
		f.limitEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Kana font", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		f.kanaFont,
		previews,
	)
}

// kanaFontID is the id of the kana font the Game tab has selected.
func (f *settingsForm) kanaFontID() string {
	for _, c := range fontChoices {
		if c.label == f.kanaFont.Selected {
			return c.id
		}
	}
	return fontChoices[0].id
}

// wordSource is the word source the Words tab has selected.
func (f *settingsForm) wordSource() string {
	if i := f.sourceSelect.SelectedIndex(); i >= 0 {
```

Replace:

```go
		limit = n
	}
	gs.applyWordSettings(f.words())
	gs.applySettings(checkedRowIDs(f.rowChecks), checkedScripts(f.scriptChecks), f.autoCheck.Checked, limit,
		levelFromLabel(f.difficulty.Selected), pointsFromLabel(f.points.Selected))
}
```

with:

```go
		limit = n
	}
	gs.applyWordSettings(f.words())
	gs.SetKanaFont(f.kanaFontID())
	gs.applySettings(checkedRowIDs(f.rowChecks), checkedScripts(f.scriptChecks), f.autoCheck.Checked, limit,
		levelFromLabel(f.difficulty.Selected), pointsFromLabel(f.points.Selected))
}
```

Replace:

```go
		gs.SetSettingsOpen(false)
		if save {
			form.apply(gs)
			gs.mu.Lock()
			snap := gs.snapshot()
			gs.mu.Unlock()
```

with:

```go
		gs.SetSettingsOpen(false)
		if save {
			form.apply(gs)
			statsPanel.SetKanaFont(gs.KanaFont())
			gs.mu.Lock()
			snap := gs.snapshot()
			gs.mu.Unlock()
```

`fyne/version.go`:

Replace:

```go
	{"JMdict mapping", "stephenmk/yomitan-jlpt-vocab, CC BY-SA 4.0. Matches the JLPT lists to JMdict.", []sourceLink{
		{"GitHub", "https://github.com/stephenmk/yomitan-jlpt-vocab"},
	}},
}

// newSourcesTab shows the version and credits every data source.
```

with:

```go
	{"JMdict mapping", "stephenmk/yomitan-jlpt-vocab, CC BY-SA 4.0. Matches the JLPT lists to JMdict.", []sourceLink{
		{"GitHub", "https://github.com/stephenmk/yomitan-jlpt-vocab"},
	}},
	{"Noto Sans JP", "© 2014-2021 Adobe, SIL Open Font License 1.1. Kana font.", []sourceLink{
		{"noto-cjk", "https://github.com/notofonts/noto-cjk"},
	}},
	{"M PLUS Rounded 1c", "© 2016 The Rounded M+ Project Authors, SIL Open Font License 1.1. Kana font.", []sourceLink{
		{"Google Fonts", "https://fonts.google.com/specimen/M+PLUS+Rounded+1c"},
	}},
}

// newSourcesTab shows the version and credits every data source.
```

- [ ] **Step 9: Run**

Run: `go test -count=1 ./fyne/` — expected: PASS. Then `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...`, and once `go test -race -count=1 ./fyne/` (takes about two minutes) and `go build -o kana-desktop ./fyne/ && rm kana-desktop`.

- [ ] **Step 10: Commit** (the font files are about 16 MB)

```bash
git add scripts/fetch-fonts.sh fyne/fonts/NotoSansJP-Regular.otf fyne/fonts/NotoSansJP-Bold.otf \
	fyne/fonts/MPLUSRounded1c-Regular.ttf fyne/fonts/MPLUSRounded1c-Bold.ttf \
	fyne/fonts/OFL-NotoSansCJK.txt fyne/fonts/OFL-MPLUSRounded1c.txt \
	fyne/fonts.go fyne/fonts_test.go fyne/tile.go fyne/feedback.go fyne/intro.go fyne/packageintro.go \
	fyne/stats.go fyne/app.go fyne/game.go fyne/settings.go fyne/version.go go.mod \
	fyne/game_test.go fyne/tile_test.go fyne/wordgame_test.go fyne/packageintro_test.go fyne/settings_test.go
git commit -m "feat(fyne): let the learner choose the kana font

Noto Sans JP (the default) and M PLUS Rounded 1c ship with the app
under the SIL OFL; scripts/fetch-fonts.sh fetches the pinned files and
checks their sums. Tiles, intros, feedback labels and the kana grid use
the chosen font whenever it has every character of a text, so a text
never mixes fonts. The Game tab previews each font."
```

---

### Task 9: Docs

**Files:** Modify `README.md`, `PLAN.md`, `vocab/data/ATTRIBUTION.md`.

- [ ] **Step 1: `README.md`**

Replace:

````markdown
  on the chosen difficulty level, a per-character factor and the session tempo (see
  Difficulty below), independent of window height
- **Romaji Input**: Type the romanized equivalent and press Enter to score
- **Words**: once you can read them, Japanese words fall among the kana tiles (green border).
  A word appears when every kana in it is secure (answered correctly three times) in an
  active script: every word from your own lists (see [Private word lists](#private-word-lists)).
  The bundled JLPT words are off for now, until they arrive in small packages with an
  introduction; with the `bundled_words` setting on they are played up to your level (N5 by
  default). Type the whole reading (`sensei`, `kitte`, `koohii`, `ko-hi-`; spaces are
  ignored, so `shitsurei desu ga` works); a hit shows the meaning, a missed word shows its
  romaji and meaning. One spawn in a hundred is a word per playable word, at most 10 %;
  longer words fall more slowly (up to twice the time).
- **Learning path**: start with あいうえお, unlock two rows at a time; each automatically unlocked row is introduced before it is asked, and the game pauses while new rows are introduced
- **Two scripts**: hiragana and katakana progress independently; switch scripts in the settings. After the basic hiragana you are offered katakana once.
- **Score Limit Mode**: Set a target score or 0 for endless practice
````

with:

````markdown
  on the chosen difficulty level, a per-character factor and the session tempo (see
  Difficulty below), independent of window height
- **Romaji Input**: Type the romanized equivalent and press Enter to score
- **Words**: Japanese words fall among the kana tiles (green border), introduced in small
  packages like kana rows. As soon as a word of your source can be read (every kana in it is
  secure, answered correctly three times, in an active script), a package of 3, 5 or 10 words
  is shown first: each word with its romaji and meaning, and an "Already known" box that
  counts the word as learned and brings in another one. Only introduced words fall. The next
  package comes when 80 % of the newest one is secure. New words come from one of your own
  lists (by lesson, then file order; see [Private word lists](#private-word-lists)) or from the
  bundled JLPT words (N5 first, the most common words first); once a JLPT level is fully
  introduced, the game offers the next one. Type the whole reading (`sensei`, `kitte`,
  `koohii`, `ko-hi-`; spaces are ignored, so `shitsurei desu ga` works); a hit shows the
  meaning, a missed word shows its romaji and meaning. One spawn in a hundred is a word per
  playable word, at most 10 %; longer words fall more slowly (up to twice the time).
- **Learning path**: start with あいうえお, unlock two rows at a time; each automatically unlocked row is introduced before it is asked, and the game pauses while new rows are introduced
- **Two scripts**: hiragana and katakana progress independently; switch scripts in the settings. After the basic hiragana you are offered katakana once.
- **Score Limit Mode**: Set a target score or 0 for endless practice
````

Replace:

````markdown
- **Row Selection**: Choose which hiragana and katakana rows to practice, grouped by Basic, Dakuon, Handakuon, Yōon, Yōon + Dakuten and (katakana only) Extended katakana
- **Auto-Progression**: per script, unlocks the next learning-path step (one or two rows) once every active row of that script is mastered: at least 80% of its kana (rounded down, at least one) answered correctly three times
- **Configurable Score Limit**: Set a target or play endlessly

### Desktop App (Fyne)
- Warm paper tile aesthetic — stamp-style kana tiles on a parchment background
- Split layout: game canvas on the left, stats panel on the right; window starts at 1000 × 760
- Stats panel: a session line, one learning-path line per active script, a side-by-side hiragana/katakana kana grid (new/learning/mastered, with a red border on a character missed this session), a words line (playable, secure, level or "own lists", or where to add a list) and the word data attribution
- In-game settings via gear icon (no pre-game setup form)
- Game-over dialog with missed character and word review and Play Again

## Download
````

with:

````markdown
- **Row Selection**: Choose which hiragana and katakana rows to practice, grouped by Basic, Dakuon, Handakuon, Yōon, Yōon + Dakuten and (katakana only) Extended katakana
- **Auto-Progression**: per script, unlocks the next learning-path step (one or two rows) once every active row of that script is mastered: at least 80% of its kana (rounded down, at least one) answered correctly three times
- **Configurable Score Limit**: Set a target or play endlessly
- **Word packages**: where new words come from (a private list or JLPT), the JLPT level and
  the package size (3, 5 or 10)
- **Kana font**: Noto Sans JP (default), M PLUS Rounded 1c or the system font, for tiles,
  intros, the kana grid and the word labels; both fonts ship with the app

### Desktop App (Fyne)
- Warm paper tile aesthetic — stamp-style kana tiles on a parchment background
- Split layout: game canvas on the left, stats panel on the right; window starts at 1000 × 760
- Stats panel: a session line, one learning-path line per active script, a side-by-side hiragana/katakana kana grid (new/learning/mastered, with a red border on a character missed this session), a words line (introduced and secure words and their source, or what to do for new words)
- In-game settings via gear icon (no pre-game setup form), in the tabs Kana, Words, Game and
  Sources (version and the credits for the word data and fonts)
- Game-over dialog with missed character and word review and Play Again

## Download
````

Replace:

````markdown
## How to Play

1. Launch the app (`./kana-desktop`)
2. Jump straight into the game; open the gear icon to adjust rows, difficulty level, points per hit, auto-progression, and score limit
3. As characters fall, type their romaji equivalent and press Enter
4. Each correct kana scores 10 points (a word 10 per kana unit), scaled by the points setting (Auto by level, or a fixed ×1/×2/×3/×5)
5. The game ends when you reach your score target, miss as many tiles as the level allows (5–10), or quit
````

with:

````markdown
## How to Play

1. Launch the app (`./kana-desktop`)
2. Jump straight into the game; open the gear icon to adjust rows, auto-progression, where new words come from, difficulty level, points per hit, score limit and the kana font
3. As characters fall, type their romaji equivalent and press Enter
4. Each correct kana scores 10 points (a word 10 per kana unit), scaled by the points setting (Auto by level, or a fixed ×1/×2/×3/×5)
5. The game ends when you reach your score target, miss as many tiles as the level allows (5–10), or quit
````

Replace:

````markdown

- `vocab.go`: `Word`, `Meanings` (per language, with fallback for display), `List`
- `csvlist.go`, `jsonlist.go`, `dir.go`: private word lists (CSV/TSV/JSON, with header aliases) and `LoadDir`
- `index.go`: `NewIndex` (one entry per reading across all sources) and `Playable` (words made
  only of known kana, filtered by JLPT level, list and lesson, optionally without bundled-only words)
- `bundled.go`: `Bundled`, the embedded JLPT word list (`data/words.json.gz`)

### Word Data Generator (`cmd/vocabgen/`)
````

with:

````markdown

- `vocab.go`: `Word`, `Meanings` (per language, with fallback for display), `List`
- `csvlist.go`, `jsonlist.go`, `dir.go`: private word lists (CSV/TSV/JSON, with header aliases) and `LoadDir`
- `index.go`: `NewIndex` (one entry per reading across all sources, with each list entry's first
  row) and `Playable` (words made only of known kana, filtered by JLPT level, list and lesson)
- `package.go`: `Source` and `NextPackage` (the next words to introduce from a list or JLPT, in
  learning order)
- `bundled.go`: `Bundled`, the embedded JLPT word list (`data/words.json.gz`, with a frequency
  rank per word)

### Word Data Generator (`cmd/vocabgen/`)
````

Replace:

````markdown
- `words.go`: word helpers — meaning language, missed-word and feedback texts, the words line
- `wordgame.go`: word progress (stats, memory, merge), the playable-word rule and cache, word spawning
- `feedback.go`: the labels shown after a word hit or miss
- `intro.go`: Pending dialogs (katakana offer and row intros) through one entry point, `showPendingDialogs`
- `stats.go`: `StatsPanel` widget — a session line, the words line (playable and secure words, or why words are unavailable), one learning-path line per active script, a side-by-side hiragana/katakana kana grid of each character's learning state, and the word data attribution lines
- `progress.go`: computes each script's learning-path status (`PathStatus`) and total correct counts per character
- `statswidgets.go`: the fixed-column grid layout and the kana cell widget used by the stats panel
- `input.go`: `InputBar` — score, level, tempo, miss count, text entry
- `settings.go`: In-game settings dialog with scripts and row selection (one tab per script, per-group "all" toggle), difficulty level and points per hit
- `theme.go`: `KanaTheme` — warm paper colour palette

### Persistence (`store/`)

- `store/store.go`: SQLite persistence, including active scripts, the katakana offer flag, the chosen difficulty level, the points setting and per-character spaced-repetition memory states (seeded once from past correct counts), per-word statistics and the word settings (`word_level`, `bundled_words`, `meaning_lang`)

### Game Timing
- Tick loop: 100ms
````

with:

````markdown
- `words.go`: word helpers — meaning language, missed-word and feedback texts, the words line
- `wordgame.go`: word progress (stats, memory, merge), the playable-word rule and cache, word spawning
- `feedback.go`: the labels shown after a word hit or miss
- `packages.go`: word packages — when the next one is due, finishing it ("already known" words
  count as secure), replacement words, the JLPT level offer
- `packageintro.go`: the word package intro and the level offer dialogs
- `fonts.go`: the bundled kana fonts (`fonts/`), the font choice and the glyph coverage check
- `intro.go`: Pending dialogs (row intros, katakana offer, word packages, level offer) through one entry point, `showPendingDialogs`
- `stats.go`: `StatsPanel` widget — a session line, the words line (introduced and secure words, or a hint), one learning-path line per active script and a side-by-side hiragana/katakana kana grid of each character's learning state
- `progress.go`: computes each script's learning-path status (`PathStatus`) and total correct counts per character
- `statswidgets.go`: the fixed-column grid layout and the kana cell widget used by the stats panel
- `input.go`: `InputBar` — score, level, tempo, miss count, text entry
- `settings.go`: In-game settings dialog with the tabs Kana (scripts and row selection, one tab per script, per-group "all" toggle, auto-progression), Words (source, JLPT level, package size), Game (difficulty level, points per hit, score limit, kana font) and Sources
- `version.go`: the version label and the Sources tab
- `theme.go`: `KanaTheme` — warm paper colour palette

### Persistence (`store/`)

- `store/store.go`: SQLite persistence, including active scripts, the katakana offer flag, the chosen difficulty level, the points setting and per-character spaced-repetition memory states (seeded once from past correct counts), per-word statistics, the introduced words and the word settings (`word_level`, `word_source`, `package_size`, `newest_package`, `meaning_lang`) and the kana font (`kana_font`)

### Game Timing
- Tick loop: 100ms
````

Replace:

````markdown
  spawn next
- Per-word statistics (`word_stats`: correct and miss count per reading) and per-word
  spaced-repetition memory (`word:` items in the `memory` table)
- The JLPT level words are played up to (`word_level`, default N5), whether bundled words are
  played (`bundled_words`, default off; no UI yet) and the meaning language
  (`meaning_lang`; unset means the system language if the word list has it, else English)

The database is created automatically on first run. If a `kana.db` from an older version is
found in the working directory, it is moved to the new location once on startup.
````

with:

````markdown
  spawn next
- Per-word statistics (`word_stats`: correct and miss count per reading) and per-word
  spaced-repetition memory (`word:` items in the `memory` table)
- The words shown in a package intro (`introduced_words`; when the table is created, every
  word with stats or memory counts as introduced) and the newest package (`newest_package`)
- Where new words come from (`word_source`: `jlpt` or `list:<name>`; chosen once on the first
  start: your first list, else JLPT), the JLPT level packages come down to (`word_level`,
  default N5), the package size (`package_size`, default 5) and the meaning language
  (`meaning_lang`; unset means the system language if the word list has it, else English)
- The kana font (`kana_font`: `noto`, `mplus` or `system`)

The database is created automatically on first run. If a `kana.db` from an older version is
found in the working directory, it is moved to the new location once on startup.
````

Replace:

````markdown
The data file is licensed under CC BY-SA 4.0; the code stays MIT. Details, versions and how to
regenerate it: [`vocab/data/ATTRIBUTION.md`](vocab/data/ATTRIBUTION.md).

The game names these sources in the status panel.

### Private word lists

Your own lists (for example from a textbook) go in the `words` folder next to `kana.db`:
`$XDG_DATA_HOME/kana/words/`, or `~/.local/share/kana/words/`. They are read from there only
and never belong in the repository. Their words fall in the game at any JLPT level; the
folder is read when the app starts. While bundled words are off, these lists are the only
source of words, and without one the status panel says where to add it.

- **CSV/TSV** (`*.csv`, `*.tsv`): a header row; the separator (tab, semicolon or comma) is
  taken from it. Columns: `kana` (required), `meaning` (any language), `meaning_de`,
````

with:

````markdown
The data file is licensed under CC BY-SA 4.0; the code stays MIT. Details, versions and how to
regenerate it: [`vocab/data/ATTRIBUTION.md`](vocab/data/ATTRIBUTION.md).

The game credits these sources on the settings' Sources tab.

### Private word lists

Your own lists (for example from a textbook) go in the `words` folder next to `kana.db`:
`$XDG_DATA_HOME/kana/words/`, or `~/.local/share/kana/words/`. They are read from there only
and never belong in the repository. The folder is read when the app starts. Choose a list as
the source of new words in the settings (Words tab); its words are introduced lesson by lesson
(numeric lessons in number order, others in the order they first appear), then in file order,
with words without a lesson last. If the chosen list is missing, the status panel says where
to add it.

- **CSV/TSV** (`*.csv`, `*.tsv`): a header row; the separator (tab, semicolon or comma) is
  taken from it. Columns: `kana` (required), `meaning` (any language), `meaning_de`,
````

Replace:

````markdown

Rows whose reading is not kana are skipped and reported.

## Dependencies

- [Fyne v2](https://fyne.io) — Desktop GUI framework
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite driver

## Development
````

with:

````markdown

Rows whose reading is not kana are skipped and reported.

## Fonts

The kana font can be Noto Sans JP (default) or M PLUS Rounded 1c; both ship with the app in
`fyne/fonts/` (regular and bold, about 16 MB) under the SIL Open Font License 1.1
(`fyne/fonts/OFL-*.txt`). `scripts/fetch-fonts.sh` downloads the pinned files (Noto CJK release
`Sans2.004`, google/fonts at a fixed commit) and checks their SHA-256 sums. A text is shown in
the chosen font only if the font has every character of it; otherwise it uses the system font,
so one text never mixes two fonts.

## Dependencies

- [Fyne v2](https://fyne.io) — Desktop GUI framework
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite driver
- [go-text/typesetting](https://github.com/go-text/typesetting) — font parsing for the kana
  font's coverage check (also used by Fyne)

## Development
````

- [ ] **Step 2: `PLAN.md`**

Replace:

````markdown
- [x] Bundled word list derived from JMdict (EDRDG, CC BY-SA 4.0, attribution required), JLPT words with Waller's levels (`vocab/data/`); used by the game
- [x] Import private word lists (CSV/TSV/JSON: kana, meaning, lesson, jlpt) from a user directory, never from the repository (`vocab.LoadDir`); used by the game
- [x] Throttle words after play-testing: at most 10 % word spawns, bundled words off by default (`bundled_words`), spaces in answers ignored, German/English header aliases in CSV/TSV lists
- [ ] Word packages with an intro dialog, which re-enable the bundled words (stage 4)

## Learning Progress System
````

with:

````markdown
- [x] Bundled word list derived from JMdict (EDRDG, CC BY-SA 4.0, attribution required), JLPT words with Waller's levels (`vocab/data/`); used by the game
- [x] Import private word lists (CSV/TSV/JSON: kana, meaning, lesson, jlpt) from a user directory, never from the repository (`vocab.LoadDir`); used by the game
- [x] Throttle words after play-testing: at most 10 % word spawns, bundled words off by default (`bundled_words`), spaces in answers ignored, German/English header aliases in CSV/TSV lists
- [x] Word packages with an intro dialog, which re-enable the bundled words (stage 4a): source (private list or JLPT), package size, level offer, "Already known"
- [x] Settings in tabs (Kana, Words, Game, Sources); the data attribution moves to the Sources tab (stage 4a)
- [x] Kana font choice: Noto Sans JP, M PLUS Rounded 1c or System, both fonts bundled (stage 4a)
- [ ] Reading practice, words-only mode with list/lesson, list management and the meaning language setting (stage 4b)

## Learning Progress System
````

- [ ] **Step 3: `vocab/data/ATTRIBUTION.md`**

Replace:

````markdown
  [stephenmk/yomitan-jlpt-vocab](https://github.com/stephenmk/yomitan-jlpt-vocab),
  CC BY-SA 4.0.

The app credits these sources in-app as well: in the vocabulary status panel (from stage 3
of the vocabulary trainer) and on a Sources tab (from stage 4).

The versions used for the committed file are recorded in [`SOURCES`](SOURCES): the
`Last-Modified` date of `JMdict.gz` and the commit of the sequence mapping.
````

with:

````markdown
  [stephenmk/yomitan-jlpt-vocab](https://github.com/stephenmk/yomitan-jlpt-vocab),
  CC BY-SA 4.0.

The app credits these sources in-app as well, on the settings' Sources tab.

The versions used for the committed file are recorded in [`SOURCES`](SOURCES): the
`Last-Modified` date of `JMdict.gz` and the commit of the sequence mapping.
````

Replace:

````markdown
## What the file contains

Every JLPT word whose reading can be typed, keyed by its kana reading, with its easiest
JLPT level and up to five meanings per language. For each JMdict entry, a language's meanings
come from the first sense that has meanings in that language (at most three), and homophones
are merged. Numbering, labels and source-language notes are removed from non-English meanings;
meanings of derived phrases and ones that cannot be cleaned (unbalanced parentheses, kana) are
````

with:

````markdown
## What the file contains

Every JLPT word whose reading can be typed, keyed by its kana reading, with its easiest
JLPT level, a frequency rank and up to five meanings per language. The rank comes from
JMdict's priority tags of the matching reading (or, without tags there, the best tag of the
entry): `nf01`–`nf48` give 1–48, `news1`/`ichi1`/`spec1`/`gai1` 49, the `…2` tags 60, none 99;
homophones keep the best rank. The game introduces common words first. For each JMdict entry, a language's meanings
come from the first sense that has meanings in that language (at most three), and homophones
are merged. Numbering, labels and source-language notes are removed from non-English meanings;
meanings of derived phrases and ones that cannot be cleaned (unbalanced parentheses, kana) are
````

- [ ] **Step 4: Run** `gofmt -l .`, `go vet ./...`, `go test -count=1 ./...` — all clean.

- [ ] **Step 5: Commit**

```bash
git add README.md PLAN.md vocab/data/ATTRIBUTION.md
git commit -m "docs: describe word packages, settings tabs and the kana font

Documents how words are introduced in packages, the word source and
level offer, the settings tabs, the bundled fonts with their licences
and the fetch script, the frequency rank in the word data and the new
database contents."
```

---

## Spec coverage (stage 4a testing list)

| Spec test | Where |
|---|---|
| vocabgen: rank from `nfXX`, `news1`/`ichi1`, `*2`, none → 99, merged homophones best; fixture; reproducible | Task 1 `TestRankOf`, `TestEntryRank`, `TestGenerateRanks`; `TestGenerateIsReproducible` (unchanged) |
| vocab: JLPT order (level, rank, units, kana); skips introduced and unplayable; partial; empty | Task 2 `TestNextPackageJLPTOrder`, `TestNextPackageSkipsIntroducedAndUnplayable` |
| vocab: list order numeric / first appearance / no lesson last; two lessons use the first; `ListPos` through `WithLists` | Task 2 `TestNextPackageListOrder`, `TestLessonIsTheFirstInListOrder`; Task 1 `TestEntryListPosIsTheFirstRow` |
| vocab: per-source cached order; rank only from bundled words, 0 sorts as 99; `clone` copies `ListPos` | Task 2 `TestNextPackageOrderIsCachedPerSource`; Task 1 `TestEntryRankFromBundledWordsOnly`, `TestWithListsDoesNotShareMeanings` |
| store: seeding from `word_stats` and `word:` memory once; settings round-trip and defaults | Task 3 `TestIntroducedSeededOnceFromWordProgress`, `TestIntroducedWordsRoundTrip`, `TestWordPackageSettings` |
| fyne: only introduced words spawn | Task 4 `TestPlayableWordsAreIntroducedAndKnown`, `TestPrivateListWordsNeedIntroduction` |
| first package when a source word is playable; next at ≥ 80 % (5 → 4, 3 → 3, 10 → 8) | Task 5 `TestFirstPackageWhenASourceWordIsPlayable`, `TestNextPackageAt80PercentSecure` |
| `FinishPackage` with known words: stats ≥ 3, `Seed(3)`, session untouched, persisted at once, newest excludes known | Task 5 `TestFinishPackage`, `TestFailedIntroducedSaveIsRetriedByTheMerge` |
| refill: `ReplacementWord` appends a new entry; none left → false | Task 5 `TestReplacementWord`; Task 6 `TestPackageViewAlreadyKnown` |
| 80 % rule counts only playable words; none playable → complete | Task 5 `TestPackageRuleCountsOnlyPlayableWords` |
| re-check after `Reset` offers again and pauses | Task 5 `TestResetClearsPendingAndOffersAgain` |
| `introducedLoadFailed`: no words, no packages, hint | Task 4 `TestIntroducedLoadFailureMeansNoWords`, `TestIntroducedLoadFailureIsDetected`; Task 5 `TestNoPackageWhileOverMissingListOrLoadFailed` |
| pause invariant with pending package / level offer; `Reset` clears pending, keeps progress | Task 5 `TestPendingPackageHoldsThePause`, `TestLevelOfferWhenTheLevelIsExhausted`, `TestResetClearsPendingAndOffersAgain` |
| level offer when exhausted, not after Later, accept lowers the level and offers a package | Task 5 `TestLevelOfferWhenTheLevelIsExhausted`, `TestLevelOfferLaterHoldsForTheSession`, `TestNoLevelOfferAtN1OrForAList`; Task 6 `TestLevelOfferDialog` |
| source change clears `newestPackage`; missing list → no packages, hint, Words tab "List … not found"; auto source saved on first start | Task 5 `TestSourceChangeClearsNewestPackage`, `TestNoPackageWhileOverMissingListOrLoadFailed`; Task 4 `TestWordsStatusLine`, `TestWordSourceResolvedAndSavedOnFirstStart`; Task 7 `TestSettingsFormMissingList` |
| introduced words keep falling after level or source change | Task 4 `TestIntroducedWordsKeepFallingAfterLevelOrSourceChange` |
| status line texts | Task 4 `TestFormatWordsLine`, `TestWordsStatusLine`; Task 5 `TestLevelOfferLaterHoldsForTheSession` |
| settings tabs carry the values; Sources tab lists all five sources; validation switches tab | Task 7 `TestSettingsFormCarriesValues`, `TestSettingsApplySavesEveryTab`, `TestSettingsValidationSwitchesToTheTab`, `TestSettingsDialogSaveAndInvalidValue`, `TestSourcesTabCreditsEverySource` (fonts added in Task 8) |
| fonts: resources load; Thai ก → nil; tiles use and are sized with the font; `system` → nil | Task 8 `TestFontResourcesLoad`, `TestFontChoiceCoverage`, `TestTilesUseTheKanaFontAndAreSizedWithIt`, `TestGameUsesTheChosenFont`, `TestFeedbackFontCheckedOnTheFinalText`, `TestIntroAndPackageUseTheKanaFont`, `TestStatsPanelKanaFont`, `TestKanaFontSetting`, `TestGameTabPreviewsEachFont` |
| Manual: kana render in both fonts on macOS, Windows, Linux; a first package for a beginner with secure vowels and k-row | after Task 8, with the release packages |
