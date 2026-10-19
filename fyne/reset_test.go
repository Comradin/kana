package main

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/memory"
	"kana/store"
)

// resetGame returns a game on a store with both scripts active, a few rows
// of each selected and every kana of them learned, flagged on か and カ.
func resetGame(t *testing.T) (*GameState, *store.Store) {
	t.Helper()
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveActiveScripts([]string{"hiragana", "katakana"})
	_ = st.SaveSelectedRows([]string{"vowels", "k", "kata:vowels", "kata:k"})
	_ = st.SaveKatakanaOffered(true)
	for _, id := range []string{"vowels", "k", "kata:vowels", "kata:k"} {
		row, _ := kanacore.RowByID(id)
		for _, c := range row.Characters() {
			_ = st.SaveKanaStats(c, 6, 1, 0)
			_ = st.SaveMemoryState("kana:"+c, memory.State{Stability: 2})
			_ = st.SaveProgress("kana:"+c, store.Progress{Level: 5})
		}
	}
	_ = st.SaveWordStats("あか", 2, 0)
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.kanaProgress["か"] = store.Progress{Level: 3, Flagged: true}
	gs.recordCorrect("き") // unsaved session data
	// The bundled JLPT list makes a word package due at start; clear it so
	// the reset is not refused as busy.
	gs.pendingPackage = nil
	gs.pendingLevelOffer = false
	gs.paused = false
	gs.mu.Unlock()
	if p := gs.PendingDialogs(); len(p.Intro) > 0 || p.KatakanaOffer || len(p.Package) > 0 || p.LevelOffer != 0 {
		t.Fatalf("fixture must start without a pending dialog: %+v", p)
	}
	return gs, st
}

// selectedIDs returns the selected row IDs in kanacore.AllKanaRows order.
// Must be called under lock.
func selectedIDs(gs *GameState) []string {
	var ids []string
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

func TestResetScriptActive(t *testing.T) {
	gs, st := resetGame(t)
	if err := gs.ResetScript(kanacore.ScriptHiragana); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	for _, c := range []string{"あ", "か", "き"} {
		if _, ok := gs.overallStats[c]; ok {
			t.Errorf("%s still has stats", c)
		}
		if _, ok := gs.sessionStats[c]; ok {
			t.Errorf("%s still has session stats", c)
		}
		if _, ok := gs.memory[c]; ok || gs.kanaProgress[c] != (store.Progress{}) || gs.kanaProgressDirty[c] || gs.memoryDirty[c] {
			t.Errorf("%s keeps memory or progress", c)
		}
	}
	if gs.kanaProgress["カ"].Level != 5 || gs.overallStats["カ"].CorrectCount != 6 {
		t.Fatal("katakana must be untouched")
	}
	stored, _ := st.KanaStatistics()
	if _, ok := stored["き"]; ok || stored["カ"].CorrectCount != 6 {
		t.Fatalf("store after reset: %v", stored)
	}
	if !equalIDs(selectedIDs(gs), []string{"vowels", "kata:vowels", "kata:k"}) || !gs.autoProgress {
		t.Fatalf("rows = %v, auto %v; want hiragana back at its first step", selectedIDs(gs), gs.autoProgress)
	}
	if !equalIDs(gs.pendingIntro, []string{"vowels"}) || !gs.activeScripts[kanacore.ScriptHiragana] {
		t.Fatalf("pending intro %v; want the first step's intro, hiragana still active", gs.pendingIntro)
	}
	if words, _ := st.WordStatistics(); words["あか"].CorrectCount != 2 || !gs.katakanaOffered {
		t.Fatal("word data and the katakana offer stay")
	}
}

func TestResetScriptPausesItsWords(t *testing.T) {
	gs, _ := resetGame(t)
	gs.mu.Lock()
	withWords(gs, bw("あい", 5, "love"))
	before := len(gs.playableWords())
	gs.mu.Unlock()
	if err := gs.ResetScript(kanacore.ScriptHiragana); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	after := len(gs.playableWords())
	gs.mu.Unlock()
	if before != 1 || after != 0 {
		t.Fatalf("playable before %d, after %d; want あい paused by the reset", before, after)
	}
}

func TestResetScriptInactive(t *testing.T) {
	gs, _ := resetGame(t)
	gs.mu.Lock()
	gs.activeScripts = map[kanacore.Script]bool{kanacore.ScriptHiragana: true}
	gs.mu.Unlock()
	if err := gs.ResetScript(kanacore.ScriptKatakana); err != nil {
		t.Fatal(err)
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.hasSelectedRowOf(kanacore.ScriptKatakana) || len(gs.pendingIntro) != 0 || gs.kanaProgress["カ"].Level != 0 {
		t.Fatalf("inactive katakana: rows %v, intro %v", selectedIDs(gs), gs.pendingIntro)
	}
	gs.enableKatakanaLocked()
	if !equalIDs(gs.pendingIntro, []string{"kata:vowels"}) {
		t.Fatalf("turned on later: intro %v, want its first step", gs.pendingIntro)
	}
}

func TestResetScriptRefusedWhileBusy(t *testing.T) {
	for name, busy := range map[string]func(gs *GameState){
		"intro":    func(gs *GameState) { gs.pendingIntro = []string{"s"} },
		"offer":    func(gs *GameState) { gs.pendingOffer = true },
		"package":  func(gs *GameState) { gs.pendingPackage = []string{"あい"} },
		"level":    func(gs *GameState) { gs.pendingLevelOffer = true },
		"gameover": func(gs *GameState) { gs.over = true },
	} {
		gs, st := resetGame(t)
		gs.mu.Lock()
		busy(gs)
		gs.mu.Unlock()
		if err := gs.ResetScript(kanacore.ScriptHiragana); !errors.Is(err, errResetBusy) {
			t.Errorf("%s: err = %v, want errResetBusy", name, err)
		}
		if stored, _ := st.KanaStatistics(); stored["あ"].CorrectCount != 6 {
			t.Errorf("%s: store changed", name)
		}
	}
}

func TestResetScriptStoreFailureChangesNothing(t *testing.T) {
	gs, st := resetGame(t)
	_ = st.Close() // every store call fails now
	if err := gs.ResetScript(kanacore.ScriptHiragana); err == nil || errors.Is(err, errResetBusy) {
		t.Fatalf("err = %v, want the store error", err)
	}
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.kanaProgress["あ"].Level != 5 || !gs.selectedRows["k"] || len(gs.pendingIntro) != 0 {
		t.Fatal("a failed reset must leave the game as it was")
	}
}
