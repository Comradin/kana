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
	w.Resize(fyne.NewSize(1000, 760))

	gs := NewGameState(st)

	statsPanel := newStatsPanel()
	gameCanvas := newGameCanvas(gs)

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
	// Keyboard focus starts in the romaji entry, so the learner can type
	// right away without reaching for the mouse.
	w.Canvas().Focus(inputBar.entry)

	// Esc pauses when nothing has focus; the focused entry handles it itself.
	// Behind a dialog, Esc is left alone so the dialog keeps the keyboard.
	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		if key.Name == fyne.KeyEscape && w.Canvas().Overlays().Top() == nil {
			inputBar.TogglePause()
		}
	})

	// Initial stats render
	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	statsPanel.Update(snap)

	// Start game loop
	gs.Start(gameCanvas)

	// Watch for game events. Capture the current channel under the lock so
	// this watcher is tied to this game's channel, not whatever gs.eventCh
	// points to later (Reset() swaps it in for Play Again).
	gs.mu.Lock()
	ch := gs.eventCh
	gs.mu.Unlock()
	go watchEvents(ch, gs, statsPanel, gameCanvas, inputBar, w)

	// buildWindow owns the app's started hook; nothing else may call
	// SetOnStarted without clobbering this callback.
	// First launch: introduce the first row once the window is up.
	a.Lifecycle().SetOnStarted(func() {
		// Raise and activate the window: launched without an app bundle
		// (e.g. `go run` on macOS) it would otherwise leave the keyboard
		// with the terminal.
		w.RequestFocus()
		showPendingDialogs(gs, statsPanel, inputBar, w)
	})

	w.SetOnClosed(func() {
		gs.Stop() // closes stopCh; safe if already stopped
		gs.mu.Lock()
		gs.mergeSessionStats()
		gs.mu.Unlock()
	})

	return w
}

// watchEvents drains a single game's event channel. ch is captured by the
// caller under gs.mu at goroutine-start time; it identifies which game this
// watcher belongs to. Reset() swaps gs.eventCh for a new channel on Play
// Again, so any event arriving after that swap (drained from the old,
// closed channel) is stale and must be ignored rather than acted on for the
// new game.
func watchEvents(ch chan gameEvent, gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, w fyne.Window) {
	for event := range ch {
		gs.mu.Lock()
		stale := ch != gs.eventCh
		over := gs.over
		gs.mu.Unlock()
		if stale {
			continue
		}

		switch event.kind {
		case rowsUnlockedEvent, katakanaOfferEvent:
			if over {
				// Game over wins; an intro stays pending for Play Again.
				// A pending offer cannot occur at game over: Reset drops it
				// (see GameState.Reset), so dropping the event here is safe
				// for both kinds.
				continue
			}
			fyne.Do(func() { showPendingDialogs(gs, statsPanel, inputBar, w) })
		case gameOverEvent:
			gs.mu.Lock()
			snap := gs.snapshot()
			reason := gs.overReason
			gs.mu.Unlock()
			fyne.Do(func() {
				showGameOverDialog(gs, snap, reason, statsPanel, gameCanvas, inputBar, w)
			})
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
		gs.Reset()           // closes old stopCh and eventCh; old watcher drains and exits on stale events
		gs.Start(gameCanvas) // launches new tick/spawn goroutines
		gs.mu.Lock()
		ch := gs.eventCh
		gs.mu.Unlock()
		go watchEvents(ch, gs, statsPanel, gameCanvas, inputBar, w) // new watcher tied to the new eventCh

		gs.mu.Lock()
		snap := gs.snapshot()
		gs.mu.Unlock()
		statsPanel.Update(snap)
		inputBar.Update(snap)
		inputBar.SetPaused(false) // Reset cleared the user pause
		gameCanvas.Refresh()
		showPendingDialogs(gs, statsPanel, inputBar, w)
	}, w)
}
