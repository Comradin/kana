package main

import (
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
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

// TestCheckAutoProgressionUnlocksNextRow verifies that when all selected rows
// are mastered the next locked row is unlocked.
func TestCheckAutoProgressionUnlocksNextRow(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gs.autoProgress = true

	// Replace default (all) selection with only the vowels row.
	gs.selectedRows = map[string]bool{"vowels": true}

	for _, char := range []string{"あ", "い", "う", "え", "お"} {
		gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
	}

	unlocked := gs.checkAutoProgression()
	if len(unlocked) != 1 {
		t.Fatalf("expected 1 row unlocked, got %d (%v)", len(unlocked), unlocked)
	}
	if unlocked[0] != "k" {
		t.Errorf("expected k-row unlocked, got %q", unlocked[0])
	}
	if !gs.selectedRows["k"] {
		t.Error("expected k-row selected after unlock")
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
