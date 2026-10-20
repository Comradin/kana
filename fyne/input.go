package main

import (
	"fmt"
	"strconv"

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
	scoreBox    *fyne.Container // scoreLabel at a steady width
	missedBox   *fyne.Container // missedLabel at a steady width
	entry       *romajiEntry
	pauseBtn    *widget.Button
	togglePause func()
	Container   *fyne.Container
}

func newInputBar(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, win fyne.Window) *InputBar {
	ib := &InputBar{
		scoreLabel:  widget.NewLabel("Score: 0"),
		missedLabel: widget.NewLabelWithStyle("", fyne.TextAlignTrailing, fyne.TextStyle{}),
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
	ib.entry.SetPlaceHolder("type romaji or a word's meaning…")

	ib.entry.OnSubmitted = func(text string) {
		// checkAnswer acquires its own lock and calls canvas.Refresh internally
		gs.checkAnswer(text)

		// Build snapshot for stats/score updates
		gs.mu.Lock()
		snap := gs.snapshot()
		gs.mu.Unlock()

		ib.entry.SetText("")
		ib.Update(snap)
		statsPanel.Update(snap)
	}

	gearBtn := widget.NewButton("⚙", func() {
		showSettingsDialog(gs, statsPanel, gameCanvas, ib, win)
	})

	ib.pauseBtn = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), ib.togglePause)

	ib.scoreBox = container.New(&steadyWidth{}, ib.scoreLabel)
	ib.missedBox = container.New(&steadyWidth{}, ib.missedLabel)
	rightCluster := container.NewHBox(ib.missedBox, ib.pauseBtn, gearBtn)
	ib.Container = container.NewBorder(nil, nil, ib.scoreBox, rightCluster, ib.entry)
	return ib
}

// Update refreshes score and missed labels (used after Reset/PlayAgain).
// Each label keeps the width of its widest likely text, so the entry does
// not shift when the score gains a digit.
func (ib *InputBar) Update(snap StatsSnapshot) {
	ib.scoreLabel.SetText(formatStatus(snap.Score, snap.ScoreLimit, snap.LevelLabel, snap.Tempo))
	ib.missedLabel.SetText(fmt.Sprintf("Missed: %d/%d", snap.Missed, snap.MissLimit))
	setSteadyWidth(ib.scoreBox, formatStatus(wideNumber(snap.Score, 6), wideNumber(snap.ScoreLimit, 0), snap.LevelLabel, snap.Tempo))
	setSteadyWidth(ib.missedBox, fmt.Sprintf("Missed: %d/%d", wideNumber(snap.MissLimit, 0), wideNumber(snap.MissLimit, 0)))
}

// wideNumber returns a number of at least minDigits digits, and as many as
// n has, made of 8s, a wide digit, for measuring the widest text to come.
func wideNumber(n, minDigits int) int {
	if n == 0 && minDigits == 0 {
		return 0
	}
	w := 0
	for d := max(minDigits, len(strconv.Itoa(n))); d > 0; d-- {
		w = w*10 + 8
	}
	return w
}

// setSteadyWidth reserves the width a label with text would take.
func setSteadyWidth(box *fyne.Container, text string) {
	l := box.Layout.(*steadyWidth)
	w := widget.NewLabel(text).MinSize().Width
	if w != l.width {
		l.width = w
		box.Refresh()
	}
}

// steadyWidth lays out one object at least width wide, so a label does not
// change its size with every character it gains or loses.
type steadyWidth struct{ width float32 }

func (l *steadyWidth) MinSize(objects []fyne.CanvasObject) fyne.Size {
	s := objects[0].MinSize()
	s.Width = max(s.Width, l.width)
	return s
}

func (l *steadyWidth) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
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
