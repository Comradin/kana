package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/vocab"
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
