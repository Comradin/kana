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
