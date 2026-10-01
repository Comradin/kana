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

// introShowing guards against two intro dialogs stacking (e.g. a stale
// rowsUnlockedEvent racing Play Again's direct showIntroDialog call).
// Fyne-thread-only: read and written exclusively from code that runs via
// fyne.Do or a Fyne lifecycle callback, so it needs no lock.
var introShowing bool

// showIntroDialog presents newly unlocked rows as large tiles with their
// romaji. The game stays paused and input disabled until the dialog closes.
// Must run on the Fyne thread.
func showIntroDialog(gs *GameState, rowIDs []string, inputBar *InputBar, w fyne.Window) {
	if introShowing {
		return
	}

	labels := make([]string, 0, len(rowIDs))
	body := container.NewVBox()
	for _, id := range rowIDs {
		row, ok := kanacore.RowByID(id)
		if !ok {
			continue
		}
		labels = append(labels, row.Label)
		cards := container.NewHBox()
		for _, e := range row.Entries {
			cards.Add(newIntroCard(e))
		}
		body.Add(widget.NewLabelWithStyle(row.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		body.Add(cards)
	}

	inputBar.SetEnabled(false, w)
	d := dialog.NewCustom("New: "+strings.Join(labels, ", "), "Let's go", body, w)
	d.SetOnClosed(func() {
		introShowing = false
		gs.FinishIntro()
		inputBar.SetEnabled(true, w)
	})
	introShowing = true
	d.Show()
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
