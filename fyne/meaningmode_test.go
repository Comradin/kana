package main

import (
	"math/rand"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"kana/memory"
	"kana/store"
	"kana/vocab"
)

// kasa is a synthetic word with a German and an English meaning.
var kasa = vocab.Word{Kana: "かさ", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"de": {"Regenschirm"}, "en": {"umbrella"}}}

// meaningTileAt is a word tile asking for the meaning at (10, y) for a
// reading in gs.words.
func meaningTileAt(gs *GameState, kana string, y float32) *KanaTile {
	e, ok := gs.words.Lookup(kana)
	if !ok {
		panic("no word " + kana)
	}
	tile := newMeaningTile(e, gs.canvasW, fontChoice{})
	tile.fallSeconds = 20
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestReadingTileCountsTheAnswerItGot(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	gs.levelOfferDeclined = true // かさ is the whole source: no level offer may pause the game
	gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
	gs.checkAnswer("umbrella")
	if len(gs.tiles) != 0 || gs.meaningProgress["かさ"].Level != 1 || gs.meaningSessionStats["かさ"].CorrectCount != 1 {
		t.Fatalf("a meaning on a reading tile is a meaning hit: tiles %d, meaning %+v", len(gs.tiles), gs.meaningProgress["かさ"])
	}
	if _, ok := gs.wordProgress["かさ"]; ok || len(gs.wordSessionStats) != 0 {
		t.Fatal("a meaning hit must leave the reading level alone")
	}
	if got := gs.feedback[0].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("feedback %q: answered by meaning keeps its look", got)
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
	gs.checkAnswer("kasa")
	if gs.wordProgress["かさ"].Level != 1 || gs.meaningProgress["かさ"].Level != 1 {
		t.Fatalf("the reading raises the reading level only: reading %+v, meaning %+v", gs.wordProgress["かさ"], gs.meaningProgress["かさ"])
	}
}

func TestMeaningTileTakesOnlyTheMeaning(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	tile := meaningTileAt(gs, "かさ", 100)
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("kasa")
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 1 || gs.tiles[0] != tile {
		t.Fatal("the reading must not answer a meaning tile")
	}
	if len(gs.feedback) != 1 || gs.feedback[0].kind != feedbackHint || gs.feedback[0].text.Text != meaningHintText || gs.feedback[0].text.Color != feedbackHintColor {
		t.Fatalf("want one hint label, got %d labels", len(gs.feedback))
	}
	if len(gs.wordProgress) != 0 || len(gs.meaningProgress) != 0 || len(gs.wordSessionStats) != 0 || len(gs.meaningSessionStats) != 0 || gs.score != 0 || gs.answerStreak != 0 {
		t.Fatal("the hint is no hit and no mistake")
	}
	gs.checkAnswer("Umbrella")
	if len(gs.tiles) != 0 || gs.meaningProgress["かさ"].Level != 1 || len(gs.wordProgress) != 0 {
		t.Fatalf("a meaning answers it and raises the meaning level: tiles %d, meaning %+v", len(gs.tiles), gs.meaningProgress["かさ"])
	}
	if got := gs.feedback[len(gs.feedback)-1].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("hit feedback %q", got)
	}
}

func TestMeaningFirstWhenTheInputIsBoth(t *testing.T) {
	sushi := vocab.Word{Kana: "すし", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {"sushi"}}}
	gs := newTestState()
	withWords(gs, sushi)
	gs.levelOfferDeclined = true // すし is the whole source: no level offer may pause the game
	gs.tiles = []*KanaTile{meaningTileAt(gs, "すし", 0)}
	gs.checkAnswer("sushi")
	if len(gs.tiles) != 0 || gs.meaningProgress["すし"].Level != 1 || len(gs.wordProgress) != 0 {
		t.Fatal("meaning mode: an input that is reading and meaning is a meaning hit")
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "すし", 0)}
	gs.checkAnswer("sushi")
	if gs.wordProgress["すし"].Level != 1 || gs.meaningProgress["すし"].Level != 1 {
		t.Fatal("reading mode: an input that is both stays a reading hit")
	}
}

func TestMeaningTileMissLowersOnlyTheMeaningLevel(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	setWordLevel(gs, "かさ", 3)
	gs.meaningProgress["かさ"] = store.Progress{Level: 4}
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 599)}
	gs.tick()
	if len(gs.tiles) != 0 || gs.missed != 1 {
		t.Fatalf("tiles %d, missed %d", len(gs.tiles), gs.missed)
	}
	if gs.meaningProgress["かさ"] != (store.Progress{Level: 2, Flagged: true}) || gs.meaningSessionStats["かさ"].MissCount != 1 {
		t.Fatalf("meaning %+v, stats %+v", gs.meaningProgress["かさ"], gs.meaningSessionStats["かさ"])
	}
	if gs.wordProgress["かさ"] != (store.Progress{Level: 3}) || len(gs.wordSessionStats) != 0 {
		t.Fatalf("reading must stay: %+v", gs.wordProgress["かさ"])
	}
	if got := gs.feedback[0].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("miss feedback %q", got)
	}
}

func TestWrongInputOnAMeaningTileLowersTheMeaningLevel(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 100)}
	gs.checkAnswer("parasol")
	if len(gs.tiles) != 1 || gs.meaningProgress["かさ"] != (store.Progress{Flagged: true}) || len(gs.wordProgress) != 0 {
		t.Fatalf("meaning %+v, reading %v", gs.meaningProgress["かさ"], gs.wordProgress)
	}
}

func TestLateReadingForAFallenMeaningTileIsNoMistake(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	kana := tileAt("あ", "a", 100)
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 599), kana}
	gs.tick() // かさ falls through
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 1 || gs.kanaProgress["あ"].Flagged {
		t.Fatal("the reading of a meaning tile that just fell is a late Enter, not a mistake on あ")
	}
	if gs.feedback[len(gs.feedback)-1].kind == feedbackHint {
		t.Fatal("no hint for a tile that is gone")
	}
}

func TestWordsAskIsStored(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	if gs.wordsAsk != readingKind {
		t.Fatalf("default wordsAsk %v, want reading", gs.wordsAsk)
	}
	gs.applyWordSettings(wordSettings{source: gs.wordSource, level: gs.wordLevel, packageSize: gs.packageSize, ask: meaningKind})
	if a, _ := st.WordsAsk(); a != store.WordsAskMeaning || gs.wordsAsk != meaningKind {
		t.Fatalf("stored %q, state %v", a, gs.wordsAsk)
	}
	if again := NewGameState(st); again.wordsAsk != meaningKind {
		t.Fatalf("loaded %v, want meaning", again.wordsAsk)
	}
}

func TestMeaningModeSpawnsMeaningTiles(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	withWords(gs, bw("あい", 5, "love"))
	gs.wordsAsk = meaningKind
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].word == nil || gs.tiles[0].ask != meaningKind || gs.tiles[0].marker == nil {
		t.Fatal("in meaning mode a word spawns as a meaning tile")
	}

	gs = newTestState() // no words: kana fall
	selectRows(gs, "vowels")
	gs.wordsAsk = meaningKind
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].word != nil || gs.tiles[0].ask != readingKind || gs.tiles[0].marker != nil {
		t.Fatal("kana tiles still ask for the romaji")
	}
}

func TestChangingWordsAskLeavesFallingTiles(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う", "え", "お")
	withWords(gs, bw("あい", 5, "love"), vocab.Word{Kana: "あお", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{}})
	reading := wordTileAt(gs, "あお", 100) // no meaning at all
	meaning := meaningTileAt(gs, "あい", 50)
	gs.tiles = []*KanaTile{reading, meaning}
	// A package size change as well, so dropInactiveTiles runs.
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 3, ask: meaningKind})
	if len(gs.tiles) != 2 || gs.tiles[0] != reading || gs.tiles[1] != meaning || reading.ask != readingKind {
		t.Fatal("switching to meaning mode keeps falling tiles and their questions")
	}
	gs.checkAnswer("ao")
	if len(gs.tiles) != 1 || gs.wordProgress["あお"].Level != 1 {
		t.Fatal("the falling reading tile still takes its reading")
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, ask: readingKind})
	if len(gs.tiles) != 1 || gs.tiles[0] != meaning || meaning.ask != meaningKind {
		t.Fatal("switching back keeps the meaning tile")
	}
}

func TestMeaningModeAsksOnlyWordsWithAMeaning(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.rnd = rand.New(rand.NewSource(11)).Float64
	withWords(gs,
		bw("あい", 5, "love"),
		vocab.Word{Kana: "あお", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{}},
		vocab.Word{Kana: "あう", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {"(to meet)"}}},
		vocab.Word{Kana: "いえ", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"de": {"Haus"}}},
	)
	if got := readingsOf(gs.askableWords()); !equalIDs(got, []string{"あい", "あう", "あお", "いえ"}) {
		t.Fatalf("reading mode asks every playable word, got %v", got)
	}
	gs.wordsAsk = meaningKind
	if got := readingsOf(gs.askableWords()); !equalIDs(got, []string{"あい", "いえ"}) {
		t.Fatalf("meaning mode: %v, want あい and いえ (a meaning only in brackets or none does not count)", got)
	}
	for i := 0; i < 300; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if k := gs.tiles[0].word.Kana; k != "あい" && k != "いえ" {
			t.Fatalf("spawned %s in meaning mode", k)
		}
	}
	if got := playableKana(gs); len(got) != 4 {
		t.Fatalf("playable %v: words without a meaning stay playable", got)
	}
}

func TestMeaningModeWithoutMeaningsFallsBackToKana(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	withWords(gs, vocab.Word{Kana: "あお", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{}})
	if got := formatWordsLine(gs.wordsStatus()); got != "Words: JLPT N5 finished" {
		t.Fatalf("reading mode line %q", got)
	}
	gs.wordsAsk = meaningKind
	gs.spawnKana()
	if gs.tiles[0].word != nil {
		t.Fatal("no word with a meaning: a kana falls")
	}
	if got := formatWordsLine(gs.wordsStatus()); got != "Words only: no word has a meaning to ask for" {
		t.Fatalf("meaning mode line %q", got)
	}
}

func TestMeaningModePicksByMeaningLevel(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.focusWeak = true
	gs.wordsAsk = meaningKind
	gs.rnd = rand.New(rand.NewSource(5)).Float64
	words := vowelWords(10)
	withWords(gs, words...)
	meaningWeak, readingWeak := words[3].Kana, words[6].Kana
	gs.meaningProgress[meaningWeak] = store.Progress{Level: 2, Flagged: true}
	gs.wordProgress[readingWeak] = store.Progress{Level: 2, Flagged: true}
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		gs.tiles = nil
		gs.spawnKana()
		counts[gs.tiles[0].word.Kana]++
	}
	if n := counts[meaningWeak]; n < 480 || n > 640 {
		t.Fatalf("slipped meaning in %d/1000 spawns, want about 550", n)
	}
	if n := counts[readingWeak]; n > 150 {
		t.Fatalf("a slipped reading is no weak item in meaning mode, got %d/1000", n)
	}
}

func TestMeaningModePicksByMeaningMemory(t *testing.T) {
	gs := newTestState()
	now := time.Unix(1_700_000_000, 0)
	gs.now = func() time.Time { return now }
	gs.rnd = rand.New(rand.NewSource(9)).Float64
	fresh := memory.State{Stability: 100, LastSeen: now}
	missed := memory.Miss(fresh, now) // half forgotten
	gs.wordMemory["あい"], gs.wordMemory["あお"] = missed, fresh
	gs.meaningMemory["あい"], gs.meaningMemory["あお"] = fresh, missed
	count := func() int {
		n := 0
		for i := 0; i < 1000; i++ {
			if gs.pickReading([]string{"あい", "あお"}) == "あお" {
				n++
			}
		}
		return n
	}
	if n := count(); n > 200 {
		t.Fatalf("reading mode picked あお %d/1000; its reading is fresh", n)
	}
	gs.wordsAsk = meaningKind
	if n := count(); n < 800 {
		t.Fatalf("meaning mode picked あお %d/1000; its meaning was just missed", n)
	}
}
