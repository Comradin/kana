package main

import (
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/store"
)

// TestMergeSessionStatsRoundTrip verifies that mergeSessionStats correctly
// writes (baseline + session) to the store exactly once, deletes session
// entries after saving, and does not double-count on subsequent merges.
func TestMergeSessionStatsRoundTrip(t *testing.T) {
	test.NewApp()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	gs := NewGameState(st)

	gs.mu.Lock()
	gs.recordCorrect("か")
	gs.recordCorrect("か")
	gs.recordCorrect("か")
	gs.recordMiss("あ")

	if got := gs.sessionStats["か"].CorrectCount; got != 3 {
		gs.mu.Unlock()
		t.Fatalf("sessionStats[か].CorrectCount = %d, want 3", got)
	}
	if got := gs.sessionStats["あ"].MissCount; got != 1 {
		gs.mu.Unlock()
		t.Fatalf("sessionStats[あ].MissCount = %d, want 1", got)
	}

	gs.mergeSessionStats()

	// session entries should be deleted after merge
	if _, ok := gs.sessionStats["か"]; ok {
		gs.mu.Unlock()
		t.Errorf("expected sessionStats[か] removed after merge")
	}
	if _, ok := gs.sessionStats["あ"]; ok {
		gs.mu.Unlock()
		t.Errorf("expected sessionStats[あ] removed after merge")
	}

	if got := gs.overallStats["か"].CorrectCount; got != 3 {
		gs.mu.Unlock()
		t.Errorf("overallStats[か].CorrectCount = %d, want 3", got)
	}
	if got := gs.overallStats["あ"].MissCount; got != 1 {
		gs.mu.Unlock()
		t.Errorf("overallStats[あ].MissCount = %d, want 1", got)
	}
	gs.mu.Unlock()

	// Verify persistence.
	persisted, err := st.KanaStatistics()
	if err != nil {
		t.Fatalf("KanaStatistics: %v", err)
	}
	if got := persisted["か"].CorrectCount; got != 3 {
		t.Errorf("persisted か.CorrectCount = %d, want 3", got)
	}
	if got := persisted["あ"].MissCount; got != 1 {
		t.Errorf("persisted あ.MissCount = %d, want 1", got)
	}

	// Record another correct and merge again; verify no double-count.
	gs.mu.Lock()
	gs.recordCorrect("か")
	gs.mergeSessionStats()
	gs.mu.Unlock()

	persisted2, err := st.KanaStatistics()
	if err != nil {
		t.Fatalf("KanaStatistics: %v", err)
	}
	if got := persisted2["か"].CorrectCount; got != 4 {
		t.Errorf("after second merge: persisted か.CorrectCount = %d, want 4 (double-count regression?)", got)
	}
}

// TestGameStateLifecycle verifies Start/Reset/Stop can be called in sequence
// without panicking and without leaking channel state.
func TestGameStateLifecycle(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	canvas := newGameCanvas(gs)

	gs.Start(canvas)
	time.Sleep(50 * time.Millisecond)

	// Reset closes stopCh and eventCh; should not panic.
	gs.Reset()

	// Restart with the new stopCh/eventCh.
	gs.Start(canvas)
	time.Sleep(50 * time.Millisecond)

	gs.Stop()
}

// TestResetClosesEventCh verifies that Reset closes the event channel so any
// watcher goroutine ranging over it exits cleanly.
func TestResetClosesEventCh(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	canvas := newGameCanvas(gs)
	gs.Start(canvas)

	// Capture the current eventCh before Reset swaps it.
	gs.mu.Lock()
	ch := gs.eventCh
	gs.mu.Unlock()

	done := make(chan struct{})
	go func() {
		for range ch {
			// drain
		}
		close(done)
	}()

	gs.Reset()

	select {
	case <-done:
		// good
	case <-time.After(time.Second):
		t.Fatal("watcher did not exit after Reset closed eventCh")
	}

	gs.Stop()
}

// TestCheckAutoProgressionUnlocksNextStep verifies that when all selected rows
// are mastered the whole next progression step is unlocked.
func TestCheckAutoProgressionUnlocksNextStep(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	gs.selectedRows = map[string]bool{"vowels": true}
	masterRows(gs, "vowels")

	unlocked := gs.checkAutoProgression()
	if !equalIDs(unlocked, []string{"k", "s"}) {
		t.Fatalf("unlocked %v, want [k s]", unlocked)
	}
	if !gs.selectedRows["k"] || !gs.selectedRows["s"] {
		t.Error("expected k and s selected after unlock")
	}
}

// TestCheckAutoProgressionNoUnlockWhenNotAllMastered verifies that auto
// progression does not fire when mastery is below the 80% threshold.
func TestCheckAutoProgressionNoUnlockWhenNotAllMastered(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true
	gs.selectedRows = map[string]bool{"vowels": true}

	for _, char := range []string{"あ", "い", "う"} {
		gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
	}

	unlocked := gs.checkAutoProgression()
	if len(unlocked) != 0 {
		t.Errorf("expected no unlock at 60%%, got %d rows: %v", len(unlocked), unlocked)
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestFreshStartBeginsLearningPath(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)

	gs := NewGameState(st)

	if len(gs.selectedRows) != 1 || !gs.selectedRows["vowels"] {
		t.Fatalf("selected rows = %v, want only vowels", gs.selectedRows)
	}
	if !gs.autoProgress || !gs.paused || !equalIDs(gs.pendingIntro, []string{"vowels"}) {
		t.Fatalf("auto=%v paused=%v pending=%v", gs.autoProgress, gs.paused, gs.pendingIntro)
	}
	rows, _ := st.SelectedRows()
	auto, _ := st.AutoProgress()
	if !equalIDs(rows, []string{"vowels"}) || !auto {
		t.Fatalf("persisted rows=%v auto=%v", rows, auto)
	}
}

func TestFreshStartOverridesStoredAutoProgressOff(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveAutoProgress(false)

	gs := NewGameState(st)
	if !gs.autoProgress {
		t.Fatal("fresh start should turn auto-progression on")
	}
}

func TestFreshStartCatchesUpMasteredSteps(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	for _, id := range []string{"vowels", "k", "s", "t", "n"} {
		row, _ := kanacore.RowByID(id)
		for _, char := range row.Characters() {
			_ = st.SaveKanaStats(char, 3, 0, 3)
		}
	}

	gs := NewGameState(st)

	for _, id := range []string{"vowels", "k", "s", "t", "n"} {
		if !gs.selectedRows[id] {
			t.Errorf("expected %s selected by catch-up", id)
		}
	}
	if gs.selectedRows["h"] {
		t.Error("catch-up must stop at the first unmastered step")
	}
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("returning user should get no intro (paused=%v pending=%v)", gs.paused, gs.pendingIntro)
	}
}

func TestReturningUserWithUnmasteredVowelsGetsNoIntro(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveKanaStats("あ", 1, 0, 1)

	gs := NewGameState(st)
	if len(gs.selectedRows) != 1 || !gs.selectedRows["vowels"] {
		t.Fatalf("selected rows = %v, want only vowels", gs.selectedRows)
	}
	if gs.paused || gs.pendingIntro != nil {
		t.Fatal("user with stats should start without intro")
	}
}

func TestSavedSelectionIsKept(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels", "k"})
	_ = st.SaveAutoProgress(false)

	gs := NewGameState(st)
	if len(gs.selectedRows) != 2 || !gs.selectedRows["k"] || gs.autoProgress || gs.paused {
		t.Fatalf("rows=%v auto=%v paused=%v", gs.selectedRows, gs.autoProgress, gs.paused)
	}
}

func TestSetSelectedRowsPersistsInRowOrder(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)

	gs := NewGameState(st)
	gs.SetSelectedRows([]string{"ky", "vowels", "g"})

	rows, err := st.SelectedRows()
	if err != nil {
		t.Fatalf("SelectedRows: %v", err)
	}
	if !equalIDs(rows, []string{"vowels", "g", "ky"}) {
		t.Fatalf("persisted rows = %v, want [vowels g ky]", rows)
	}
}

func TestStoreErrorDoesNotTriggerFreshStart(t *testing.T) {
	test.NewApp()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_ = st.Close() // every query now fails

	gs := NewGameState(st)
	if len(gs.selectedRows) != len(kanacore.DefaultRowIDs()) || gs.paused {
		t.Fatalf("rows=%v paused=%v, want basic defaults without pause", gs.selectedRows, gs.paused)
	}
}
