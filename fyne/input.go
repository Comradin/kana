package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// romajiEntry is the answer entry. Esc toggles the pause instead of reaching
// the entry, so the learner can pause without leaving the keyboard.
type romajiEntry struct {
	widget.Entry
	onEscape func()
}

func newRomajiEntry(onEscape func()) *romajiEntry {
	e := &romajiEntry{onEscape: onEscape}
	e.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	e.ExtendBaseWidget(e)
	return e
}

// TypedKey handles Esc and passes every other key to the entry.
func (e *romajiEntry) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyEscape {
		if e.onEscape != nil {
			e.onEscape()
		}
		return
	}
	e.Entry.TypedKey(key)
}

// InputBar holds the score label, romaji entry, missed count, pause button
// and settings gear.
type InputBar struct {
	scoreLabel  *widget.Label
	missedLabel *widget.Label
	entry       *romajiEntry
	pauseBtn    *widget.Button
	togglePause func()
	Container   *fyne.Container
}

func newInputBar(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, win fyne.Window) *InputBar {
	ib := &InputBar{
		scoreLabel:  widget.NewLabel("Score: 0"),
		missedLabel: widget.NewLabel("Missed: 0/10"),
	}
	// The button, Esc on the entry and Esc on the window all use this toggle.
	ib.togglePause = func() {
		ib.SetPaused(gs.TogglePause())
		gameCanvas.Refresh()
		if !ib.entry.Disabled() {
			win.Canvas().Focus(ib.entry)
		}
	}
	ib.entry = newRomajiEntry(ib.togglePause)
	ib.entry.SetPlaceHolder("type romaji…")

	ib.entry.OnSubmitted = func(text string) {
		// checkAnswer acquires its own lock and calls canvas.Refresh internally
		gs.checkAnswer(text)

		// Build snapshot for stats/score updates
		gs.mu.Lock()
		snap := gs.snapshot()
		gs.mu.Unlock()

		ib.entry.SetText("")
		ib.scoreLabel.SetText(ib.formatScore(snap.Score, snap.ScoreLimit))
		ib.missedLabel.SetText(fmt.Sprintf("Missed: %d/10", snap.Missed))
		statsPanel.Update(snap)
	}

	gearBtn := widget.NewButton("⚙", func() {
		showSettingsDialog(gs, statsPanel, gameCanvas, ib, win)
	})

	ib.pauseBtn = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), ib.togglePause)

	rightCluster := container.NewHBox(ib.missedLabel, ib.pauseBtn, gearBtn)
	ib.Container = container.NewBorder(nil, nil, ib.scoreLabel, rightCluster, ib.entry)
	return ib
}

func (ib *InputBar) formatScore(score, limit int) string {
	if limit > 0 {
		return fmt.Sprintf("Score: %d/%d", score, limit)
	}
	return fmt.Sprintf("Score: %d", score)
}

// Update refreshes score and missed labels (used after Reset/PlayAgain).
func (ib *InputBar) Update(snap StatsSnapshot) {
	ib.scoreLabel.SetText(ib.formatScore(snap.Score, snap.ScoreLimit))
	ib.missedLabel.SetText(fmt.Sprintf("Missed: %d/10", snap.Missed))
}

// TogglePause pauses or resumes the game, as the pause button and Esc do.
func (ib *InputBar) TogglePause() {
	ib.togglePause()
}

// SetPaused shows the play icon while paused and the pause icon otherwise.
func (ib *InputBar) SetPaused(paused bool) {
	if paused {
		ib.pauseBtn.SetIcon(theme.MediaPlayIcon())
		return
	}
	ib.pauseBtn.SetIcon(theme.MediaPauseIcon())
}

// SetEnabled enables or disables romaji entry; focus returns to the entry when enabled.
func (ib *InputBar) SetEnabled(enabled bool, win fyne.Window) {
	if enabled {
		ib.entry.Enable()
		win.Canvas().Focus(ib.entry)
		return
	}
	ib.entry.Disable()
}
