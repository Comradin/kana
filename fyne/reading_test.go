package main

import (
	"math/rand"
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
