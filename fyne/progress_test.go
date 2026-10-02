package main

import (
	"testing"

	"kana/kanacore"
	"kana/store"
)

func TestPathStatusStatesAndNext(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels", "k", "s")
	masterRows(gs, "vowels")
	ps := gs.pathStatus(kanacore.ScriptHiragana)
	if ps.Steps[0] != StepMastered || ps.Steps[1] != StepLearning || ps.Steps[2] != StepOpen {
		t.Fatalf("steps = %v", ps.Steps[:3])
	}
	if ps.Unlocked != 2 || !equalIDs(ps.Next, []string{"t", "n"}) {
		t.Fatalf("unlocked=%d next=%v, want 2 [t n]", ps.Unlocked, ps.Next)
	}
}

func TestPathStatusCountsStepsSelectedOutOfOrder(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels", "k", "s", "g", "z")
	ps := gs.pathStatus(kanacore.ScriptHiragana)
	if ps.Unlocked != 3 || !equalIDs(ps.Next, []string{"t", "n"}) {
		t.Fatalf("unlocked=%d next=%v, want 3 [t n]", ps.Unlocked, ps.Next)
	}
}

func TestPathStatusAllUnlocked(t *testing.T) {
	gs := newTestState()
	var ids []string
	for _, row := range kanacore.RowsFor(kanacore.ScriptHiragana) {
		ids = append(ids, row.ID)
	}
	selectRows(gs, ids...)
	ps := gs.pathStatus(kanacore.ScriptHiragana)
	if ps.Unlocked != 15 || ps.Next != nil {
		t.Fatalf("unlocked=%d next=%v, want 15 nil", ps.Unlocked, ps.Next)
	}
}

func TestSnapshotPathsOnlyForActiveScripts(t *testing.T) {
	gs := newTestState()
	snap := gs.snapshot()
	if _, ok := snap.Paths[kanacore.ScriptHiragana]; !ok {
		t.Fatal("missing hiragana path")
	}
	if _, ok := snap.Paths[kanacore.ScriptKatakana]; ok {
		t.Fatal("inactive katakana must have no path")
	}
}

func TestSnapshotTotalCorrectAddsStoredAndSession(t *testing.T) {
	gs := newTestState()
	gs.overallStats["か"] = store.KanaStats{Char: "か", CorrectCount: 2}
	gs.sessionStats["か"] = store.KanaStats{Char: "か", CorrectCount: 1}
	if got := gs.snapshot().TotalCorrect["か"]; got != 3 {
		t.Fatalf("TotalCorrect[か] = %d, want 3", got)
	}
}
