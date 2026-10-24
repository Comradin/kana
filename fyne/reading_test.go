package main

import (
	"math/rand"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/store"
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

func TestWordSettingsRefreshTheReadablePool(t *testing.T) {
	gs := newTestState()
	withWords(gs, bw("あい", 5, "love"), bw("いえ", 4, "house"))
	learnKana(gs, "あ", "い", "う", "え", "お")
	if got := readableKana(gs); !equalIDs(got, []string{"あい"}) {
		t.Fatalf("N5: %v", got)
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 4, packageSize: gs.packageSize})
	if got := readableKana(gs); !equalIDs(got, []string{"あい", "いえ"}) {
		t.Fatalf("after choosing N4: %v", got)
	}

	words := []vocab.Word{lw("minna", "あい", "1"), lw("minna", "あお", "2")}
	withList(gs, "minna", words...)
	if got := readableKana(gs); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("all lessons: %v", got)
	}
	gs.applyWordSettings(wordSettings{source: listSourcePrefix + "minna", level: 4, packageSize: gs.packageSize, lessons: []string{"2"}})
	if got := readableKana(gs); !equalIDs(got, []string{"あお"}) {
		t.Fatalf("after choosing lesson 2: %v", got)
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
	gs.readingPractice = false
	for i := 0; i < 60*ticksPerSecond; i++ { // nor does play without reading practice
		gs.tick()
	}
	gs.readingPractice = true
	gs.tiles = []*KanaTile{practiceTileAt(gs, "かさ", 100)}
	gs.checkAnswer("kasa")
	gs.tiles = []*KanaTile{practiceTileAt(gs, "さか", 100)}
	gs.checkAnswer("saka")
	if got := formatWordsLine(gs.wordsStatus()); got != "Reading practice: 2 words to read · 4 kana/min" {
		t.Fatalf("4 kana in one minute of reading practice: %q", got)
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

func TestReadingHitCreditsOnlyLearnedKanaOfAnotherVariant(t *testing.T) {
	asu := vocab.Word{Kana: "あす", Alts: []string{"あした"}, Source: "course", Meanings: vocab.Meanings{"en": {"tomorrow"}}} // あした // あす
	gs := newTestState()
	gs.readingPractice = true
	withList(gs, "course", asu)
	learnKana(gs, "あ", "す", "し") // た is not learned
	gs.tiles = []*KanaTile{practiceTileAt(gs, "あす", 100)}
	gs.checkAnswer("ashita")
	if len(gs.tiles) != 0 {
		t.Fatal("the other variant answers the tile")
	}
	count := func(c string) int { return gs.sessionStats[c].CorrectCount }
	if count("あ") != 1 || count("し") != 1 || count("す") != 0 {
		t.Fatalf("あ %d, し %d, す %d: the learned kana of the typed form count", count("あ"), count("し"), count("す"))
	}
	if _, ok := gs.sessionStats["た"]; ok || gs.kanaProgress["た"].Level != 0 || gs.currentStreak["た"] != 0 {
		t.Fatalf("た is not learned and must not be credited: stats %+v, progress %+v", gs.sessionStats["た"], gs.kanaProgress["た"])
	}
	if gs.readKana != 2 || gs.answerStreak != 1 {
		t.Fatalf("read kana %d (want 2), answer streak %d", gs.readKana, gs.answerStreak)
	}
}

func TestReadingPracticeRefreshesTheStatusLineEverySecond(t *testing.T) {
	gs := newTestState() // calls test.NewApp(), so fyne.Do runs the hook
	calls := 0
	gs.onStatsChanged = func() { calls++ }
	ticks := func(n int) {
		for i := 0; i < n; i++ {
			gs.tick()
		}
	}
	gs.readingPractice = true
	ticks(3 * ticksPerSecond)
	if calls != 0 {
		t.Fatalf("%d refreshes before a word is read: kana/min shows – and does not change", calls)
	}
	gs.readKana = 2
	ticks(3 * ticksPerSecond)
	if calls != 3 {
		t.Fatalf("%d refreshes in 3 s of reading practice, want 3", calls)
	}
	gs.paused = true
	ticks(3 * ticksPerSecond)
	gs.paused = false
	gs.readingPractice = false
	ticks(3 * ticksPerSecond)
	if calls != 3 {
		t.Fatalf("%d refreshes, want none while paused or without reading practice", calls)
	}
}
