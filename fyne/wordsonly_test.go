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
