package main

import (
	"math/rand"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/store"
)

func TestWeakWeight(t *testing.T) {
	cases := []struct {
		p    store.Progress
		want float64
	}{
		{store.Progress{}, 0},                    // new
		{store.Progress{Level: 1}, 1},            // learning
		{store.Progress{Level: 4}, 1},            // learning
		{store.Progress{Level: learnedLevel}, 0}, // learned
		{store.Progress{Level: 0, Flagged: true}, 2},
		{store.Progress{Level: 3, Flagged: true}, 2},
	}
	for _, c := range cases {
		if got := weakWeight(c.p); got != c.want {
			t.Errorf("weakWeight(%+v) = %v, want %v", c.p, got, c.want)
		}
	}
}

func TestFocusPicksWeakItems(t *testing.T) {
	gs := newTestState()
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(3)).Float64
	gs.kanaProgress["あ"] = store.Progress{Level: 1, Flagged: true}
	gs.kanaProgress["い"] = store.Progress{Level: 3}
	gs.kanaProgress["う"] = store.Progress{Level: learnedLevel}
	cands := []string{"あ", "い", "う", "え", "お"}
	counts := map[string]int{}
	weak := 0
	for i := 0; i < 3000; i++ {
		if c, ok := gs.pickFocus(cands, gs.kanaProgress); ok {
			weak++
			counts[c]++
		}
	}
	if weak < 1350 || weak > 1650 {
		t.Fatalf("%d/3000 focus picks, want about half", weak)
	}
	if r := float64(counts["あ"]) / float64(counts["い"]); r < 1.6 || r > 2.5 {
		t.Fatalf("slipped/learning = %d/%d, want about 2", counts["あ"], counts["い"])
	}
	if counts["う"]+counts["え"]+counts["お"] != 0 {
		t.Fatalf("learned and new items are not weak: %v", counts)
	}
}

func TestFocusDrawsNothingWithoutWeakItemsOrWhenOff(t *testing.T) {
	gs := newTestState()
	draws := 0
	gs.rnd = func() float64 { draws++; return 0 }
	gs.kanaProgress["あ"] = store.Progress{Level: 2, Flagged: true}
	if _, ok := gs.pickFocus([]string{"あ"}, gs.kanaProgress); ok || draws != 0 {
		t.Fatalf("focus off: ok %v, %d draws; want the usual pick untouched", ok, draws)
	}
	gs.focusWeak = true
	if _, ok := gs.pickFocus([]string{"い", "う"}, gs.kanaProgress); ok || draws != 0 {
		t.Fatalf("no weak candidate: ok %v, %d draws", ok, draws)
	}
}

func TestFocusAppliesToSpawnedKanaAndWords(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(5)).Float64
	gs.kanaProgress["え"] = store.Progress{Level: 1, Flagged: true}
	n := 0
	for i := 0; i < 1000; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if gs.tiles[0].kana.Char == "え" {
			n++
		}
	}
	if n < 520 || n > 680 {
		t.Fatalf("slipped え in %d/1000 spawns, want about 600 (half, plus its usual share)", n)
	}

	gs = newTestState()
	selectRows(gs, "vowels")
	gs.wordsOnly = true
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(5)).Float64
	words := vowelWords(10)
	withWords(gs, words...)
	weakWord := words[3].Kana
	gs.wordProgress[weakWord] = store.Progress{Level: 2, Flagged: true}
	n = 0
	for i := 0; i < 1000; i++ {
		gs.tiles = nil
		gs.spawnKana()
		if gs.tiles[0].word.Kana == weakWord {
			n++
		}
	}
	if n < 480 || n > 640 {
		t.Fatalf("slipped word in %d/1000 spawns, want about 550", n)
	}
}

func TestFocusKeepsTheWordShare(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.focusWeak = true
	gs.rnd = rand.New(rand.NewSource(7)).Float64
	withWords(gs, vowelWords(60)...)
	learnKana(gs, "あ", "い", "う", "え", "お")
	for kana := range gs.introduced {
		gs.wordProgress[kana] = store.Progress{Level: 1, Flagged: true}
	}
	if n := countWordSpawns(gs, 1000); n < 70 || n > 130 {
		t.Fatalf("%d/1000 word spawns with focus on, want about 100 as without", n)
	}
}

func TestFocusWeakIsStored(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.SetFocusWeak(true)
	if on, _ := st.FocusWeak(); !on || !NewGameState(st).focusWeak {
		t.Fatal("focus on weak items not saved or not loaded")
	}
}
