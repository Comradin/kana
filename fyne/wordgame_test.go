package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// withWords replaces the word index with the given bundled words.
func withWords(gs *GameState, words ...vocab.Word) {
	gs.words = vocab.NewIndex(words, nil)
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
