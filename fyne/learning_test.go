package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/store"
)

// learnKana puts each character at the learned level.
func learnKana(gs *GameState, chars ...string) {
	for _, c := range chars {
		gs.kanaProgress[c] = store.Progress{Level: learnedLevel}
	}
	gs.invalidatePlayable()
}

// setKanaLevel puts one character at level n, not flagged.
func setKanaLevel(gs *GameState, char string, n int) {
	gs.kanaProgress[char] = store.Progress{Level: n}
	gs.invalidatePlayable()
}

// setWordLevel puts one reading at level n, not flagged.
func setWordLevel(gs *GameState, kana string, n int) {
	gs.wordProgress[kana] = store.Progress{Level: n}
}

func TestHitAndMistake(t *testing.T) {
	p := store.Progress{}
	for want := 1; want <= 5; want++ {
		if p = hit(p); p.Level != want {
			t.Fatalf("hit: level %d, want %d", p.Level, want)
		}
	}
	if p = hit(p); p.Level != learnedLevel {
		t.Fatalf("level capped at 5, got %d", p.Level)
	}
	if p = mistake(p); p != (store.Progress{Level: 3, Flagged: true}) {
		t.Fatalf("mistake from 5 = %+v, want level 3 flagged", p)
	}
	if p = mistake(mistake(p)); p != (store.Progress{Level: 0, Flagged: true}) {
		t.Fatalf("level floored at 0, got %+v", p)
	}
	p = store.Progress{Level: 4, Flagged: true}
	if p = hit(p); p != (store.Progress{Level: 5}) {
		t.Fatalf("reaching 5 clears the flag, got %+v", p)
	}
	if !learned(5) || learned(4) {
		t.Fatal("learned means level 5")
	}
}

func TestKanaHitAndMissChangeTheLevel(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	setKanaLevel(gs, "か", 4)
	gs.recordCorrect("か")
	if p := gs.kanaProgress["か"]; p.Level != 5 || !gs.kanaProgressDirty["か"] {
		t.Fatalf("after a hit: %+v dirty %v", p, gs.kanaProgressDirty["か"])
	}
	gs.recordMiss("か")
	if p := gs.kanaProgress["か"]; p != (store.Progress{Level: 3, Flagged: true}) {
		t.Fatalf("after a miss: %+v", p)
	}
}

func TestWordHitAndMissChangeTheLevel(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("あい")
	gs.recordWordCorrect("あい")
	if p := gs.wordProgress["あい"]; p.Level != 2 || !gs.wordProgressDirty["あい"] {
		t.Fatalf("after two hits: %+v", p)
	}
	gs.recordWordMiss("あい")
	if p := gs.wordProgress["あい"]; p != (store.Progress{Level: 0, Flagged: true}) {
		t.Fatalf("after a miss: %+v", p)
	}
	if gs.kanaProgress["あ"].Level != 0 {
		t.Fatal("a word miss must not touch its kana")
	}
}

func TestLevelRisesBeforeAutoProgression(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels")
	learnKana(gs, "あ", "い", "う")
	setKanaLevel(gs, "え", 4)
	gs.recordCorrect("え") // the hit that reaches 5 completes 80 % of the row
	// After Task A3 isRowMastered reads levels; until then this test only
	// checks the level. Task A3 extends it to expect the unlock.
	if gs.kanaProgress["え"].Level != 5 {
		t.Fatal("level must rise in recordCorrect")
	}
}

func TestProgressSavedAndReloaded(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordCorrect("か")
	gs.recordMiss("き")
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	dirty := len(gs.kanaProgressDirty) + len(gs.wordProgressDirty)
	gs.mu.Unlock()
	if dirty != 0 {
		t.Fatalf("%d progress entries still dirty after the merge", dirty)
	}
	again := NewGameState(st)
	if again.kanaProgress["か"].Level != 1 || again.kanaProgress["き"] != (store.Progress{Flagged: true}) || again.wordProgress["あい"].Level != 1 {
		t.Fatalf("reloaded kana %v words %v", again.kanaProgress, again.wordProgress)
	}
}

func TestProgressNotWrittenWhenLoadFailed(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveProgress("kana:か", store.Progress{Level: 5})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.kanaProgressLoadFailed = true
	gs.kanaProgress = map[string]store.Progress{}
	gs.recordCorrect("か")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if p, _ := st.Progress("kana:"); p["か"].Level != 5 {
		t.Fatalf("stored progress overwritten: %+v", p["か"])
	}
}
