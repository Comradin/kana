package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// InputBar holds the score label, romaji entry, missed count, and settings gear.
type InputBar struct {
	scoreLabel  *widget.Label
	missedLabel *widget.Label
	entry       *widget.Entry
	Container   *fyne.Container
}

func newInputBar(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, win fyne.Window) *InputBar {
	ib := &InputBar{
		scoreLabel:  widget.NewLabel("Score: 0"),
		missedLabel: widget.NewLabel("Missed: 0/10"),
		entry:       widget.NewEntry(),
	}
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
		showSettingsDialog(gs, statsPanel, gameCanvas, win)
	})

	rightCluster := container.NewHBox(ib.missedLabel, gearBtn)
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
