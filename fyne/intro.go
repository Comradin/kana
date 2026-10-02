package main

import (
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

const (
	introCardW     float32 = 104
	introCardWideW float32 = 168
	introCardH     float32 = 120
)

// dialogShowing guards against stacking the offer and intro dialogs.
// Fyne-thread-only: read and written exclusively from code that runs via
// fyne.Do or a Fyne callback, so it needs no lock.
var dialogShowing bool

// showPendingDialogs shows whatever the game is waiting for: the katakana
// offer first, then an intro. With nothing pending it resumes the game and
// re-enables input, so "paused" never outlives its dialog. It also refreshes
// the stats panel, so a change that became visible only by accepting or
// declining a dialog (e.g. a newly active script's section) shows up without
// waiting for the next Enter. Must run on the Fyne thread.
func showPendingDialogs(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	if dialogShowing {
		return
	}
	gs.mu.Lock()
	snap := gs.snapshot()
	gs.mu.Unlock()
	statsPanel.Update(snap)

	offer, intro := gs.PendingDialogs()
	switch {
	case offer:
		showKatakanaOffer(gs, statsPanel, inputBar, w)
	case len(intro) > 0:
		showIntroDialog(gs, intro, statsPanel, inputBar, w)
	default:
		gs.Resume()
		if !gs.IsOver() {
			inputBar.SetEnabled(true, w)
		}
	}
}

// continueAfterDialog chains to the next pending dialog, if any.
func continueAfterDialog(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	fyne.Do(func() { showPendingDialogs(gs, statsPanel, inputBar, w) })
}

// showIntroDialog presents newly unlocked rows as large tiles with their
// romaji. Call it through showPendingDialogs. Must run on the Fyne thread.
func showIntroDialog(gs *GameState, rowIDs []string, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	labels := make([]string, 0, len(rowIDs))
	body := container.NewVBox()
	for _, id := range rowIDs {
		row, ok := kanacore.RowByID(id)
		if !ok {
			continue
		}
		labels = append(labels, row.Label)
		body.Add(widget.NewLabelWithStyle(row.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		body.Add(introCards(row))
	}

	inputBar.SetEnabled(false, w)
	d := dialog.NewCustom("New: "+strings.Join(labels, ", "), "Let's go", body, w)
	d.SetOnClosed(func() {
		// A custom dialog has no response callback; SetOnClosed is the first
		// callback that runs on close.
		dialogShowing = false
		gs.FinishIntro()
		continueAfterDialog(gs, statsPanel, inputBar, w)
	})
	dialogShowing = true
	d.Show()
}

// showKatakanaOffer asks once whether to add katakana. Call it through
// showPendingDialogs. Must run on the Fyne thread.
func showKatakanaOffer(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	body := container.NewVBox(
		widget.NewLabel("You know the basic hiragana. Katakana use the same sounds;\nstart with ア イ ウ エ オ?"),
	)
	if row, ok := kanacore.RowByID("kata:vowels"); ok {
		body.Add(introCards(row))
	}

	inputBar.SetEnabled(false, w)
	d := dialog.NewCustomConfirm("Add Katakana?", "Add Katakana", "Not now", body, func(add bool) {
		// The response callback runs before SetOnClosed; clear the guard
		// first so the chained intro is not swallowed.
		dialogShowing = false
		if add {
			gs.EnableKatakana()
		} else {
			gs.DeclineKatakana()
		}
		continueAfterDialog(gs, statsPanel, inputBar, w)
	}, w)
	dialogShowing = true
	d.Show()
}

// introCards lays out a row's kana as large cards.
func introCards(row kanacore.KanaRow) fyne.CanvasObject {
	cards := container.NewHBox()
	for _, e := range row.Entries {
		cards.Add(newIntroCard(e))
	}
	return cards
}

// newIntroCard renders one kana as a large paper tile with its romaji below.
func newIntroCard(e kanacore.Entry) fyne.CanvasObject {
	width := introCardW
	if utf8.RuneCountInString(e.Char) > 1 {
		width = introCardWideW
	}
	face := canvas.NewRectangle(tileFaceColor)
	face.StrokeColor = tileShadowColor
	face.StrokeWidth = 3
	face.SetMinSize(fyne.NewSize(width, introCardH))

	glyph := canvas.NewText(e.Char, kanaTextColor)
	glyph.TextSize = 64

	romaji := widget.NewLabelWithStyle(e.Romaji, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	return container.NewVBox(container.NewStack(face, container.NewCenter(glyph)), romaji)
}
