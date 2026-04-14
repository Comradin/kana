package main

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/store"
)

func buildWindow(a fyne.App, st *store.Store) fyne.Window {
	a.Settings().SetTheme(WarmPaperTheme())

	w := a.NewWindow("Kana")
	w.Resize(fyne.NewSize(900, 620))

	gs := NewGameState(st)

	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)
	gs.statsPanel = statsPanel
	gs.canvas = gameCanvas

	inputBar := newInputBar(gs, statsPanel, gameCanvas, w)

	statsContainer := container.NewPadded(statsPanel)

	layout := container.NewBorder(
		nil,                // top
		inputBar.Container, // bottom
		nil,                // left
		statsContainer,     // right
		gameCanvas,         // centre
	)

	w.SetContent(layout)

	// Initial stats render
	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	statsPanel.Update(snap)

	// Start game loop
	gs.Start(gameCanvas)

	// Watch for game events
	go watchEvents(gs, statsPanel, gameCanvas, inputBar, w)

	w.SetOnClosed(func() {
		gs.Stop() // closes stopCh; safe if already stopped
		gs.mu.Lock()
		gs.mergeSessionStats()
		gs.mu.Unlock()
	})

	return w
}

func watchEvents(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, w fyne.Window) {
	for event := range gs.eventCh {
		switch event.kind {
		case gameOverEvent:
			gs.mu.Lock()
			snap := gs.snapshot()
			reason := gs.overReason
			gs.mu.Unlock()

			// Run on a new goroutine so this watcher loop isn't blocked by the dialog.
			// Fyne dialog calls schedule themselves on the main thread internally.
			go showGameOverDialog(gs, snap, reason, statsPanel, gameCanvas, inputBar, w)
		}
	}
}

func showGameOverDialog(gs *GameState, snap StatsSnapshot, reason string, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, w fyne.Window) {
	title := "GAME OVER"
	if reason == "score" {
		title = "SESSION COMPLETE"
	}

	scoreText := fmt.Sprintf("Score: %d", snap.Score)
	if snap.ScoreLimit > 0 {
		scoreText = fmt.Sprintf("Score: %d/%d", snap.Score, snap.ScoreLimit)
	}

	var reasonText string
	switch reason {
	case "score":
		reasonText = "You reached your target score!"
	case "misses":
		reasonText = "10 kana slipped through."
	default:
		reasonText = "Session ended."
	}

	// Unique missed characters
	unique := make(map[string]kanacore.Kana)
	for _, k := range snap.MissedKanas {
		if _, exists := unique[k.Char]; !exists {
			unique[k.Char] = k
		}
	}
	keys := make([]string, 0, len(unique))
	for k := range unique {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	missedParts := make([]string, 0, len(keys))
	for _, char := range keys {
		k := unique[char]
		missedParts = append(missedParts, fmt.Sprintf("%s (%s)", k.Char, k.Romaji))
	}

	missedText := "None!"
	if len(missedParts) > 0 {
		missedText = strings.Join(missedParts, ", ")
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel(scoreText),
		widget.NewLabel(fmt.Sprintf("Missed: %d/10", snap.Missed)),
		widget.NewLabel(reasonText),
		widget.NewSeparator(),
		widget.NewLabel("Characters missed:"),
		widget.NewLabel(missedText),
	)

	dialog.ShowCustomConfirm("", "Play Again", "Quit", content, func(playAgain bool) {
		if !playAgain {
			gs.mu.Lock()
			gs.mergeSessionStats()
			gs.mu.Unlock()
			w.Close()
			return
		}
		gs.Reset()                                              // closes old stopCh and eventCh; old watcher exits
		gs.Start(gameCanvas)                                    // launches new tick/spawn goroutines
		go watchEvents(gs, statsPanel, gameCanvas, inputBar, w) // new watcher on new eventCh

		gs.mu.Lock()
		snap := gs.snapshot()
		gs.mu.Unlock()
		statsPanel.Update(snap)
		inputBar.Update(snap)
		gameCanvas.Refresh()
	}, w)
}
