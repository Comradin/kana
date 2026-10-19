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

func TestSnapshotLevelsAndFlags(t *testing.T) {
	gs := newTestState()
	gs.kanaProgress["か"] = store.Progress{Level: 3, Flagged: true}
	snap := gs.snapshot()
	if snap.Levels["か"] != 3 || !snap.Flagged["か"] {
		t.Fatalf("Levels[か] = %d, Flagged[か] = %v, want 3 true", snap.Levels["か"], snap.Flagged["か"])
	}
	gs.kanaProgress["か"] = store.Progress{Level: 5}
	gs.kanaProgress["き"] = store.Progress{Level: 1, Flagged: true}
	if snap.Levels["か"] != 3 || !snap.Flagged["か"] || snap.Levels["き"] != 0 || snap.Flagged["き"] {
		t.Fatal("the snapshot maps must be copies")
	}
}

func TestProgressionAfterGeneratedKatakanaUnlocksExtended(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	gs.activeScripts = map[kanacore.Script]bool{kanacore.ScriptKatakana: true}
	var ids []string
	for _, step := range kanacore.ProgressionStepsFor(kanacore.ScriptKatakana)[:15] {
		ids = append(ids, step...)
	}
	selectRows(gs, ids...)
	masterRows(gs, ids...)
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"kata:ext-t", "kata:ext-f"}) {
		t.Fatalf("unlocked %v, want [kata:ext-t kata:ext-f]", got)
	}
}
