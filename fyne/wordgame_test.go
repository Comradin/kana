package main

import (
	"database/sql"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/memory"
	"kana/store"
	"kana/vocab"
)

func TestRecordWordCorrectTouchesOnlyWordProgress(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("あい")
	if got := gs.wordSessionStats["あい"]; got.CorrectCount != 1 || got.Kana != "あい" || !gs.wordSessionDirty {
		t.Fatalf("word session = %+v, dirty %v", got, gs.wordSessionDirty)
	}
	if s, ok := gs.wordMemory["あい"]; !ok || s.Stability != memory.MinStability || !gs.wordMemoryDirty["あい"] {
		t.Fatalf("word memory = %+v, %v", s, ok)
	}
	if len(gs.sessionStats) != 0 || len(gs.memory) != 0 || gs.sessionDirty {
		t.Fatal("a word hit must not touch kana stats or kana memory")
	}
	if gs.answerStreak != 1 {
		t.Fatalf("answerStreak = %d, want 1", gs.answerStreak)
	}
}

func TestRecordWordMiss(t *testing.T) {
	gs := newTestState()
	gs.tempoStep, gs.answerStreak = 3, 4
	gs.recordWordMiss("あい")
	if got := gs.wordSessionStats["あい"]; got.MissCount != 1 || !gs.wordSessionDirty {
		t.Fatalf("word session = %+v", got)
	}
	if _, ok := gs.wordMemory["あい"]; ok {
		t.Fatal("a miss on a new word must not create memory")
	}
	if gs.tempoStep != 1 || gs.answerStreak != 0 {
		t.Fatalf("tempoStep=%d streak=%d, want 1/0", gs.tempoStep, gs.answerStreak)
	}
	gs.wordMemory["あお"] = memory.State{Stability: 1, LastSeen: time.Now()}
	gs.recordWordMiss("あお")
	if s := gs.wordMemory["あお"].Stability; s < 0.29 || s > 0.31 || !gs.wordMemoryDirty["あお"] {
		t.Fatalf("miss on known word → %v, want 0.3 and dirty", s)
	}
}

func TestMergePersistsWordProgress(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)

	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.recordWordCorrect("あい")
	gs.recordWordMiss("あい")
	gs.mergeSessionStats() // only words changed: the kana merge returns early
	sessionLeft, dirty, memDirty := len(gs.wordSessionStats), gs.wordSessionDirty, len(gs.wordMemoryDirty)
	overall := gs.wordOverallStats["あい"]
	gs.mu.Unlock()

	if sessionLeft != 0 || dirty || memDirty != 0 {
		t.Fatalf("after merge: session %d, dirty %v, memory dirty %d", sessionLeft, dirty, memDirty)
	}
	if overall.CorrectCount != 2 || overall.MissCount != 1 {
		t.Fatalf("overall = %+v, want 2/1", overall)
	}
	if words, _ := st.WordStatistics(); words["あい"].CorrectCount != 2 || words["あい"].MissCount != 1 {
		t.Fatalf("persisted = %+v", words["あい"])
	}
	if mem, _ := st.MemoryStates("word:"); len(mem) != 1 {
		t.Fatalf("word memory = %v, want あい only", mem)
	}
	if kana, _ := st.MemoryStates("kana:"); len(kana) != 0 {
		t.Fatalf("word memory leaked into kana memory: %v", kana)
	}

	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if words, _ := st.WordStatistics(); words["あい"].CorrectCount != 3 {
		t.Fatalf("after second merge = %+v, want 3 (double-count?)", words["あい"])
	}

	reloaded := NewGameState(st)
	if reloaded.wordOverallStats["あい"].CorrectCount != 3 {
		t.Fatalf("reloaded overall = %+v", reloaded.wordOverallStats["あい"])
	}
	if _, ok := reloaded.wordMemory["あい"]; !ok {
		t.Fatal("word memory must load on start")
	}
}

// TestMergesAreIndependent breaks one stats table through a second
// connection: the other kind must still be merged.
func TestMergesAreIndependent(t *testing.T) {
	for _, broken := range []string{"kana_stats", "word_stats"} {
		t.Run(broken, func(t *testing.T) {
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
			if _, err := db.Exec("DROP TABLE " + broken); err != nil {
				t.Fatal(err)
			}

			gs.mu.Lock()
			gs.recordCorrect("か")
			gs.recordWordCorrect("あい")
			gs.mergeSessionStats()
			kanaDirty, wordDirty := gs.sessionDirty, gs.wordSessionDirty
			gs.mu.Unlock()

			if broken == "kana_stats" {
				if !kanaDirty || wordDirty {
					t.Fatalf("kanaDirty=%v wordDirty=%v, want true/false", kanaDirty, wordDirty)
				}
				if words, err := st.WordStatistics(); err != nil || words["あい"].CorrectCount != 1 {
					t.Fatalf("words = %v, %v", words, err)
				}
			} else {
				if kanaDirty || !wordDirty {
					t.Fatalf("kanaDirty=%v wordDirty=%v, want false/true", kanaDirty, wordDirty)
				}
				if kana, err := st.KanaStatistics(); err != nil || kana["か"].CorrectCount != 1 {
					t.Fatalf("kana = %v, %v", kana, err)
				}
			}
		})
	}
}

func TestResetReloadsWordStatsAndKeepsMemory(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordStats("あい", 2, 0)
	gs := NewGameState(st)
	if gs.wordOverallStats["あい"].CorrectCount != 2 {
		t.Fatalf("loaded = %+v", gs.wordOverallStats["あい"])
	}
	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.mu.Unlock()
	gs.Reset()
	if len(gs.wordSessionStats) != 0 || gs.wordSessionDirty {
		t.Fatal("Reset must clear the word session")
	}
	if gs.wordOverallStats["あい"].CorrectCount != 3 {
		t.Fatalf("after Reset overall = %+v, want 3", gs.wordOverallStats["あい"])
	}
	if _, ok := gs.wordMemory["あい"]; !ok {
		t.Fatal("Reset must keep word memory")
	}
}

func TestFailedWordMemoryLoadSkipsSave(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.wordMemoryLoadFailed = true
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if got, _ := st.MemoryStates("word:"); len(got) != 0 {
		t.Fatalf("failed load must not write word memory, got %v", got)
	}
}

func TestStoredWordLevelIsLoaded(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if l := NewGameState(st).wordLevel; l != 5 {
		t.Fatalf("default wordLevel = %d, want 5", l)
	}
	_ = st.SaveWordLevel(3)
	if l := NewGameState(st).wordLevel; l != 3 {
		t.Fatalf("wordLevel = %d, want 3", l)
	}
}

// bw is a bundled word with one English meaning.
func bw(kana string, level int, en string) vocab.Word {
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

// secure gives each character three stored correct answers.
func secure(gs *GameState, chars ...string) {
	for _, c := range chars {
		gs.overallStats[c] = store.KanaStats{Char: c, CorrectCount: 3}
	}
	gs.invalidatePlayable()
}

func playableKana(gs *GameState) []string {
	var out []string
	for _, e := range gs.playableWords() {
		out = append(out, e.Kana)
	}
	return out
}

func TestUnitKnown(t *testing.T) {
	gs := newTestState() // hiragana only
	secure(gs, "か", "カ")
	gs.overallStats["き"] = store.KanaStats{Char: "き", CorrectCount: 2}
	gs.overallStats["く"] = store.KanaStats{Char: "く", CorrectCount: 2}
	gs.sessionStats["く"] = store.KanaStats{Char: "く", CorrectCount: 1}
	cases := map[string]bool{"か": true, "き": false, "く": true, "カ": false, "っ": true, "ッ": false, "ー": true, "け": false}
	for unit, want := range cases {
		if got := gs.unitKnown(unit); got != want {
			t.Errorf("hiragana only: unitKnown(%s) = %v, want %v", unit, got, want)
		}
	}
	gs.activeScripts = map[kanacore.Script]bool{kanacore.ScriptKatakana: true}
	cases = map[string]bool{"か": false, "カ": true, "っ": false, "ッ": true, "ー": true}
	for unit, want := range cases {
		if got := gs.unitKnown(unit); got != want {
			t.Errorf("katakana only: unitKnown(%s) = %v, want %v", unit, got, want)
		}
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
		t.Fatalf("playable without an index = %v", got)
	}
}

func TestPlayableRefreshesWhenKanaBecomesSecure(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	secure(gs, "あ")
	gs.overallStats["い"] = store.KanaStats{Char: "い", CorrectCount: 2}
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable before い is secure = %v", got)
	}
	gs.recordCorrect("い")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable after い is secure = %v", got)
	}
}

func TestKatakanaWordStopsWhenKatakanaTurnsOff(t *testing.T) {
	gs := newTestState()
	gs.activeScripts[kanacore.ScriptKatakana] = true
	withWords(gs, bw("アイ", 5, "eye"))
	secure(gs, "ア", "イ")
	if got := playableKana(gs); !equalIDs(got, []string{"アイ"}) {
		t.Fatalf("playable with katakana = %v", got)
	}
	gs.applySettings([]string{"vowels", "kata:vowels"}, []kanacore.Script{kanacore.ScriptHiragana}, false, 0, LevelNormal, pointsAuto)
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable after katakana off = %v", got)
	}
}

func TestNewGameStateLoadsBundledWords(t *testing.T) {
	test.NewApp()
	gs := NewGameState(openTestStore(t))
	if gs.words == nil || gs.wordsErr != "" || len(gs.words.Entries()) < 5000 {
		t.Fatalf("words = %v entries, err %q", gs.words != nil, gs.wordsErr)
	}
}

func TestPrivateListsLoadWhenBundledFails(t *testing.T) {
	test.NewApp()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	dir := filepath.Join(data, "kana", "words")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mine.csv"), []byte("kana,meaning\nあお,blue\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := bundledWords
	bundledWords = func() ([]vocab.Word, error) { return nil, errors.New("broken data") }
	t.Cleanup(func() { bundledWords = orig })

	gs := NewGameState(nil)
	if gs.wordsErr != "broken data" {
		t.Fatalf("wordsErr = %q", gs.wordsErr)
	}
	if entries := gs.words.Entries(); len(entries) != 1 || entries[0].Kana != "あお" {
		t.Fatalf("entries = %v, want the private あお only", entries)
	}
}

func TestMeaningLangFollowsSystemUnlessStored(t *testing.T) {
	test.NewApp()
	orig := systemLanguage
	systemLanguage = func() string { return "de-AT" }
	t.Cleanup(func() { systemLanguage = orig })
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if l := NewGameState(st).meaningLang; l != "de" {
		t.Fatalf("meaningLang = %q, want de from the system", l)
	}
	_ = st.SaveMeaningLang("sv")
	if l := NewGameState(st).meaningLang; l != "sv" {
		t.Fatalf("meaningLang = %q, want the stored sv", l)
	}
}

func TestPlayableRefreshesWhenKatakanaOfferIsAccepted(t *testing.T) {
	gs := newTestState() // hiragana only
	withWords(gs, bw("アイ", 5, "eye"))
	secure(gs, "ア", "イ")
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable before katakana = %v", got)
	}
	gs.EnableKatakana()
	if got := playableKana(gs); !equalIDs(got, []string{"アイ"}) {
		t.Fatalf("playable after accepting the offer = %v, want [アイ]", got)
	}
}

func TestResetRebuildsPlayable(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	secure(gs, "あ", "い")
	if got := playableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("playable before Reset = %v", got)
	}
	gs.Reset() // no store: the stored stats reload as empty
	if got := playableKana(gs); len(got) != 0 {
		t.Fatalf("playable after Reset = %v, want none (cache not rebuilt?)", got)
	}
}

// wordTileAt is a word tile at (10, y) for a reading in gs.words.
func wordTileAt(gs *GameState, kana string, y float32) *KanaTile {
	e, ok := gs.words.Lookup(kana)
	if !ok {
		panic("no word " + kana)
	}
	tile := newWordTile(e, gs.canvasW, fontChoice{})
	tile.fallSeconds = 20
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestSpawnCandidatesIgnoreWordTiles(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("い", 5, "stomach"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "い", 0), tileAt("あ", "a", 0)}
	if got := gs.spawnCandidates([]string{"あ", "い"}); !equalIDs(got, []string{"い"}) {
		t.Fatalf("candidates = %v, want [い]: a falling word い must not block the kana い", got)
	}
}

func TestDropInactiveTilesRemovesUnplayableWords(t *testing.T) {
	gs := newTestState()
	gs.activeScripts[kanacore.ScriptKatakana] = true
	withWords(gs, bw("アイ", 5, "eye"), bw("あい", 5, "love"))
	secure(gs, "ア", "イ", "あ", "い")
	gs.tiles = []*KanaTile{wordTileAt(gs, "アイ", 0), wordTileAt(gs, "あい", 0), tileAt("あ", "a", 0)}
	gs.applySettings([]string{"vowels", "kata:vowels"}, []kanacore.Script{kanacore.ScriptHiragana}, false, 0, LevelNormal, pointsAuto)
	if len(gs.tiles) != 2 || gs.tiles[0].word == nil || gs.tiles[0].word.Kana != "あい" || gs.tiles[1].word != nil {
		t.Fatalf("expected あい (word) and あ (kana) to remain, got %d tiles", len(gs.tiles))
	}
}

func TestWordHitScoresPerUnit(t *testing.T) {
	cases := []struct {
		level        string
		points, want int
	}{
		{"normal", pointsAuto, 30},
		{"pro", pointsAuto, 90},
		{"normal", 2, 60},
	}
	for _, c := range cases {
		gs := newTestState()
		gs.level = LevelByName(c.level)
		gs.pointsSetting = c.points
		withWords(gs, bw("あおい", 5, "blue"))
		gs.tiles = []*KanaTile{wordTileAt(gs, "あおい", 0)}
		gs.checkAnswer("aoi")
		if len(gs.tiles) != 0 || gs.score != c.want {
			t.Errorf("%s/%d: tiles %d, score %d, want %d", c.level, c.points, len(gs.tiles), gs.score, c.want)
		}
	}
}

func TestWordHitRecordsWordProgressOnly(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("しお", 5, "salt"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "しお", 0)}
	gs.checkAnswer("sio") // any valid reading
	if len(gs.tiles) != 0 {
		t.Fatal("しお not answered by 'sio'")
	}
	if gs.wordSessionStats["しお"].CorrectCount != 1 || gs.answerStreak != 1 {
		t.Fatalf("word stats %+v, streak %d", gs.wordSessionStats["しお"], gs.answerStreak)
	}
	if _, ok := gs.wordMemory["しお"]; !ok {
		t.Fatal("a word hit must create word memory")
	}
	if len(gs.sessionStats) != 0 || len(gs.memory) != 0 {
		t.Fatal("a word hit must not change any kana's stats")
	}
}

func TestLowerTileWinsAcrossKinds(t *testing.T) {
	for _, wordLower := range []bool{true, false} {
		gs := newTestState()
		withWords(gs, bw("え", 5, "picture"))
		wy, ky := float32(100), float32(300)
		if wordLower {
			wy, ky = 300, 100
		}
		word := wordTileAt(gs, "え", wy)
		kana := tileAt("え", "e", ky)
		gs.tiles = []*KanaTile{word, kana}
		gs.checkAnswer("e")
		if len(gs.tiles) != 1 {
			t.Fatalf("wordLower=%v: %d tiles left", wordLower, len(gs.tiles))
		}
		if wordLower && (gs.tiles[0] != kana || gs.wordSessionStats["え"].CorrectCount != 1 || len(gs.sessionStats) != 0) {
			t.Fatal("the lower word tile should have been answered")
		}
		if !wordLower && (gs.tiles[0] != word || gs.sessionStats["え"].CorrectCount != 1 || len(gs.wordSessionStats) != 0) {
			t.Fatal("the lower kana tile should have been answered")
		}
	}
}

func TestWordMiss(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	gs.wordMemory["あい"] = memory.State{Stability: 1, LastSeen: time.Now()}
	gs.tempoStep = 3
	gs.canvasH = 600
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 599)}
	gs.tick()
	if len(gs.tiles) != 0 || gs.missed != 1 {
		t.Fatalf("tiles %d, missed %d", len(gs.tiles), gs.missed)
	}
	if gs.wordSessionStats["あい"].MissCount != 1 {
		t.Fatalf("word stats %+v", gs.wordSessionStats["あい"])
	}
	if s := gs.wordMemory["あい"].Stability; s < 0.29 || s > 0.31 {
		t.Fatalf("word memory after miss = %v, want 0.3", s)
	}
	if gs.tempoStep != 1 {
		t.Fatalf("tempoStep = %d, want 1", gs.tempoStep)
	}
	if !equalIDs(gs.missedWords, []string{"あい"}) || len(gs.missedKanas) != 0 || len(gs.sessionStats) != 0 {
		t.Fatalf("missedWords %v, missedKanas %v", gs.missedWords, gs.missedKanas)
	}
}

func TestWordMissesCountTowardsLimit(t *testing.T) {
	gs := newTestState()
	gs.level = LevelByName("pro")
	gs.missed = 4
	gs.canvasH = 600
	withWords(gs, bw("あい", 5, "love"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 599)}
	gs.tick()
	if !gs.over || gs.overReason != "misses" {
		t.Fatalf("over=%v reason=%q", gs.over, gs.overReason)
	}
}

func TestSnapshotCarriesMissedWordsAndResetClears(t *testing.T) {
	gs := newTestState()
	gs.missedWords = []string{"あい"}
	if snap := gs.snapshot(); !equalIDs(snap.MissedWords, []string{"あい"}) {
		t.Fatalf("snapshot MissedWords = %v", snap.MissedWords)
	}
	gs.Reset()
	if len(gs.missedWords) != 0 {
		t.Fatal("Reset must clear missed words")
	}
}

// vowelWords returns n distinct words written with vowels only (at most 150).
func vowelWords(n int) []vocab.Word {
	vowels := []string{"あ", "い", "う", "え", "お"}
	var out []vocab.Word
	for _, a := range vowels {
		for _, b := range vowels {
			for _, c := range append([]string{""}, vowels...) {
				if len(out) == n {
					return out
				}
				out = append(out, bw(a+b+c, 5, "word"))
			}
		}
	}
	return out
}

func countWordSpawns(gs *GameState, spawns int) int {
	words := 0
	for i := 0; i < spawns; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if gs.tiles[0].word != nil {
			words++
		}
	}
	return words
}

func TestWordShareOfSpawns(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.rnd = rand.New(rand.NewSource(7)).Float64
	withWords(gs, vowelWords(60)...)
	if n := countWordSpawns(gs, 200); n != 0 {
		t.Fatalf("%d words spawned while no vowel is secure", n)
	}
	secure(gs, "あ", "い", "う", "え", "お")
	if n := countWordSpawns(gs, 1000); n < 70 || n > 130 {
		t.Fatalf("%d/1000 word spawns with 60 playable, want about 100 (the 10 %% cap)", n)
	}
}

func TestFewPlayableWordsSpawnRarely(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.rnd = rand.New(rand.NewSource(3)).Float64
	withWords(gs, vowelWords(5)...)
	secure(gs, "あ", "い", "う", "え", "お")
	if n := countWordSpawns(gs, 1000); n < 25 || n > 80 {
		t.Fatalf("%d/1000 word spawns with 5 playable, want about 50", n)
	}
}

func TestWordSpawnFallTimeAndPlacement(t *testing.T) {
	gs := newTestState() // Normal: 16–24 s
	selectRows(gs, "vowels")
	gs.rnd = func() float64 { return 0 } // always a word, always the first new candidate
	gs.canvasW = 150
	withWords(gs, bw("あおい", 5, "blue"))
	secure(gs, "あ", "お", "い")
	for i := 0; i < 30; i++ {
		gs.tiles = nil
		gs.spawnKana()
		tile := gs.tiles[0]
		if tile.word == nil || tile.word.Kana != "あおい" {
			t.Fatalf("spawned %+v, want the word あおい", tile.kana)
		}
		if tile.fallSeconds < 16*1.4-0.01 || tile.fallSeconds > 24*1.4+0.01 {
			t.Fatalf("fallSeconds = %v, want [22.4, 33.6]", tile.fallSeconds)
		}
		if tile.pos.X < 0 || tile.pos.X+tile.Width() > gs.canvasW+0.5 {
			t.Fatalf("tile at x=%.1f width %.1f leaves the %v px field", tile.pos.X, tile.Width(), gs.canvasW)
		}
	}
}

func TestWordSpawnAvoidsFallingWords(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.rnd = func() float64 { return 0 }
	withWords(gs, bw("あい", 5, "love"), bw("あお", 5, "blue"))
	secure(gs, "あ", "い", "お")
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 0)}
	gs.spawnKana()
	if got := gs.tiles[1].word; got == nil || got.Kana != "あお" {
		t.Fatal("expected あお while あい is falling")
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 0), wordTileAt(gs, "あお", 0)}
	gs.spawnKana()
	if got := gs.tiles[2].word; got == nil || got.Kana != "あい" {
		t.Fatal("with every word falling, the exclusion must be dropped")
	}
}

func TestWordHitShowsFeedbackAtTile(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	gs.levelOfferDeclined = true // N5 is all introduced; the offer would pause the game
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 200), tileAt("か", "ka", 0)}
	gs.checkAnswer("ka")
	if len(gs.feedback) != 0 {
		t.Fatal("a kana hit shows no feedback")
	}
	gs.checkAnswer("ai")
	if len(gs.feedback) != 1 {
		t.Fatalf("feedback = %d labels, want 1", len(gs.feedback))
	}
	f := gs.feedback[0]
	if f.kind != feedbackHit || f.text.Text != "あい — love" || f.ticksLeft != feedbackTicks {
		t.Fatalf("label = %v %q %d", f.kind, f.text.Text, f.ticksLeft)
	}
	if p := f.text.Position(); p.X != 10 || p.Y != 200 {
		t.Fatalf("label at %v, want the tile's last position (10, 200)", p)
	}
}

func TestWordMissShowsFeedbackAtBottom(t *testing.T) {
	gs := newTestState()
	gs.canvasH = 600
	withWords(gs, bw("あい", 5, "love"))
	gs.tiles = []*KanaTile{wordTileAt(gs, "あい", 599)}
	gs.tick()
	if len(gs.feedback) != 1 {
		t.Fatalf("feedback = %d labels, want 1", len(gs.feedback))
	}
	f := gs.feedback[0]
	if f.kind != feedbackMiss || f.text.Text != "あい · ai — love" || f.text.Color != cellMissedColor {
		t.Fatalf("label = %v %q %v", f.kind, f.text.Text, f.text.Color)
	}
	if p := f.text.Position(); p.Y < 600-60 || p.Y+f.text.Size().Height > 600 {
		t.Fatalf("miss label at y=%v, want at the bottom of the 600 px field", p.Y)
	}
}

func TestLongFeedbackFitsTheField(t *testing.T) {
	gs := newTestState()
	gs.canvasW = 400
	long := "あいうえお · aiueo — " + strings.Repeat("a very long meaning; ", 5)
	gs.addFeedback(feedbackMiss, long, fyne.NewPos(300, 10))
	f := gs.feedback[0]
	if f.text.TextSize < feedbackMinTextSize {
		t.Fatalf("text size %v below the %v pt minimum", f.text.TextSize, feedbackMinTextSize)
	}
	if w := f.text.MinSize().Width; w > 400 {
		t.Fatalf("label is %v px wide, want at most the 400 px field", w)
	}
	if !strings.HasSuffix(f.text.Text, "…") || !strings.HasPrefix(f.text.Text, "あいうえお · aiueo") {
		t.Fatalf("text = %q, want the start kept and an ellipsis", f.text.Text)
	}
	if p := f.text.Position(); p.X < 0 || p.X+f.text.Size().Width > 400 {
		t.Fatalf("label at x=%v (width %v), want inside the field", p.X, f.text.Size().Width)
	}

	gs.addFeedback(feedbackHit, "あい — love", fyne.NewPos(10, 10))
	if f := gs.feedback[1]; f.text.TextSize != feedbackTextSize || f.text.Text != "あい — love" {
		t.Fatalf("short label = %q at %v pt, want unchanged at %v pt", f.text.Text, f.text.TextSize, feedbackTextSize)
	}
}

func TestFeedbackExpiresAfter20Ticks(t *testing.T) {
	gs := newTestState()
	gs.addFeedback(feedbackHit, "あい — love", fyne.NewPos(10, 10))
	for i := 0; i < feedbackTicks-1; i++ {
		gs.tick()
	}
	if len(gs.feedback) != 1 {
		t.Fatal("label gone before 20 ticks")
	}
	gs.tick()
	if len(gs.feedback) != 0 {
		t.Fatal("label still shown after 20 ticks")
	}
}

func TestFeedbackFreezesAndHidesWhilePaused(t *testing.T) {
	gs := newTestState()
	gs.addFeedback(feedbackHit, "あい — love", fyne.NewPos(10, 10))
	gs.buildSnapshot()
	if n := len(tileObjects(gs)); n != 1 {
		t.Fatalf("snapshot objects = %d, want the label", n)
	}
	gs.TogglePause()
	if n := len(tileObjects(gs)); n != 0 {
		t.Fatalf("label visible while paused: %d objects", n)
	}
	for i := 0; i < 30; i++ {
		gs.tick()
	}
	if len(gs.feedback) != 1 || gs.feedback[0].ticksLeft != feedbackTicks {
		t.Fatal("labels must not count down while paused")
	}
	gs.TogglePause()
	if n := len(tileObjects(gs)); n != 1 {
		t.Fatalf("label hidden after resuming: %d objects", n)
	}
}

func TestResetClearsFeedback(t *testing.T) {
	gs := newTestState()
	gs.addFeedback(feedbackMiss, "あい · ai — love", fyne.NewPos(10, 10))
	gs.Reset()
	if len(gs.feedback) != 0 || len(tileObjects(gs)) != 0 {
		t.Fatal("Reset must clear feedback labels")
	}
}

func TestSnapshotWordsStatus(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("あお", 5, "blue"), bw("いえ", 5, "house"))
	secure(gs, "あ", "い", "お")
	gs.wordOverallStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 2}
	gs.wordSessionStats["あい"] = store.WordStats{Kana: "あい", CorrectCount: 1}
	snap := gs.snapshot()
	if snap.Words != (WordsStatus{Introduced: 3, Secure: 1, Source: "JLPT N5", Level: 5, Next: wordsFinished}) {
		t.Fatalf("Words = %+v, want 3 introduced, 1 secure, JLPT N5 finished", snap.Words)
	}
	if snap.WordCorrect != 1 {
		t.Fatalf("WordCorrect = %d, want the session's 1", snap.WordCorrect)
	}
}

func TestSnapshotShowsBundledError(t *testing.T) {
	test.NewApp()
	orig := bundledWords
	bundledWords = func() ([]vocab.Word, error) { return nil, errors.New("broken data") }
	t.Cleanup(func() { bundledWords = orig })
	gs := NewGameState(nil)
	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	if got := formatWordsLine(snap.Words); got != "Words unavailable: broken data" {
		t.Fatalf("line = %q", got)
	}
}

func TestFitTextKeepsTheLongestFittingPrefix(t *testing.T) {
	test.NewApp()
	full := strings.Repeat("abcdefghij ", 20)
	txt := canvas.NewText(full, kanaTextColor)
	txt.TextSize = 12
	fitText(txt, 150, 12)
	if w := txt.MinSize().Width; w > 150 {
		t.Fatalf("text is %v px wide, want at most 150", w)
	}
	n := len([]rune(txt.Text))
	longer := canvas.NewText(truncateRunes(full, n+1), kanaTextColor)
	longer.TextSize = 12
	if longer.MinSize().Width <= 150 {
		t.Fatalf("cut to %d runes (%q), but %d runes still fit", n, txt.Text, n+1)
	}
}

func TestApplySettingsKeepsPlayableCacheUnlessScriptsChange(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	withWords(gs, bw("あい", 5, "love"))
	secure(gs, "あ", "い")
	cached := gs.playableWords()
	if len(cached) != 1 {
		t.Fatalf("playable = %v, want [あい]", cached)
	}
	hira := []kanacore.Script{kanacore.ScriptHiragana}

	gs.applySettings([]string{"vowels"}, hira, false, 50, LevelNormal, pointsAuto)
	if !gs.playableOK || &gs.playable[0] != &cached[0] {
		t.Fatal("changing only the score limit must keep the playable cache")
	}

	gs.applySettings([]string{"vowels", "kata:vowels"}, []kanacore.Script{kanacore.ScriptHiragana, kanacore.ScriptKatakana}, false, 50, LevelNormal, pointsAuto)
	if gs.playableOK {
		t.Fatal("changing the active scripts must clear the playable cache")
	}
}

func TestSpawnFallsBackToWordWithoutKana(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"))
	secure(gs, "あ", "い")
	gs.playableWords() // cached while hiragana is active
	gs.activeScripts = map[kanacore.Script]bool{}
	if chars := gs.availableCharacters(); len(chars) != 0 {
		t.Fatalf("availableCharacters = %v, want none", chars)
	}
	gs.rnd = func() float64 { return 0.99 } // a kana spawn by wordShare
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].word == nil || gs.tiles[0].word.Kana != "あい" {
		t.Fatalf("tiles = %d, want one word tile あい when no kana is available", len(gs.tiles))
	}
}

// isolateWordsDir points the data directory at a fresh temp dir and returns
// its (empty, not yet created) words folder.
func isolateWordsDir(t *testing.T) string {
	t.Helper()
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	return filepath.Join(data, "kana", "words")
}
