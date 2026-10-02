package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/kanacore"
)

// TestShowPendingDialogsRefreshesStatsPanel guards against the stats panel
// staying stale after a dialog is handled. Accepting the katakana offer (or
// any other change resolved through showPendingDialogs) must make the new
// script's section visible immediately, not just on the next Enter.
func TestShowPendingDialogsRefreshesStatsPanel(t *testing.T) {
	app := test.NewApp()
	defer test.NewApp() // reset the singleton app for later tests
	w := test.NewWindow(nil)
	defer w.Close()
	_ = app

	gs := newTestState()
	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, statsPanel, gameCanvas, w)

	// Simulate EnableKatakana's effect (activate the script and select its
	// first row) without going through the dialog, and with nothing pending
	// so showPendingDialogs takes the "resume" branch.
	gs.activeScripts[kanacore.ScriptKatakana] = true
	gs.selectedRows["kata:vowels"] = true

	showPendingDialogs(gs, statsPanel, ib, w)

	if !statsPanel.pathLines[kanacore.ScriptKatakana].Visible() {
		t.Error("expected katakana section visible after showPendingDialogs refreshed the stats panel")
	}
}
