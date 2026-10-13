package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type feedbackKind int

const (
	feedbackHit feedbackKind = iota
	feedbackMiss
)

const (
	feedbackTicks               = 20 // 2 s at 100 ms ticks
	feedbackTextSize    float32 = 18
	feedbackMinTextSize float32 = 10 // long labels shrink down to this size
	feedbackBottomGap   float32 = 8
)

var feedbackHitColor = color.RGBA{R: 0x3d, G: 0x6b, B: 0x35, A: 0xff}

// feedbackLabel is a short text shown after a word hit or miss. The text
// object carries the message and its position; it is created once and
// published in the tile snapshot.
type feedbackLabel struct {
	kind      feedbackKind
	text      *canvas.Text
	ticksLeft int // tick() counts down; the label goes at 0
}

// addFeedback shows msg for feedbackTicks ticks. A hit label sits at pos
// (the tile's last position), a miss label at the bottom of the field in the
// error colour; both are kept inside the field. A label wider than the field
// shrinks down to feedbackMinTextSize and is then cut with "…". Must be
// called under lock.
func (gs *GameState) addFeedback(kind feedbackKind, msg string, pos fyne.Position) {
	var c color.Color = feedbackHitColor
	if kind == feedbackMiss {
		c = cellMissedColor
	}
	txt := canvas.NewText(msg, c)
	txt.TextSize = feedbackTextSize
	txt.TextStyle = fyne.TextStyle{Bold: true}
	fitText(txt, gs.canvasW)
	size := txt.MinSize()
	txt.Resize(size)
	y := pos.Y
	if kind == feedbackMiss {
		y = gs.canvasH - size.Height - feedbackBottomGap
	}
	txt.Move(fyne.NewPos(clampTo(pos.X, gs.canvasW-size.Width), clampTo(y, gs.canvasH-size.Height)))
	gs.feedback = append(gs.feedback, &feedbackLabel{kind: kind, text: txt, ticksLeft: feedbackTicks})
}

// fitText makes txt at most maxW wide: it lowers the text size down to
// feedbackMinTextSize, then drops characters from the end, ending in "…".
// A maxW of 0 or less (the canvas size is not known yet) leaves txt as is.
func fitText(txt *canvas.Text, maxW float32) {
	if maxW <= 0 {
		return
	}
	for txt.MinSize().Width > maxW && txt.TextSize > feedbackMinTextSize {
		txt.TextSize--
	}
	full := txt.Text
	for n := len([]rune(full)) - 1; n > 1 && txt.MinSize().Width > maxW; n-- {
		txt.Text = truncateRunes(full, n)
	}
}

// clampTo keeps v within [0, hi]; a negative hi gives 0.
func clampTo(v, hi float32) float32 {
	if v > hi {
		v = hi
	}
	if v < 0 {
		v = 0
	}
	return v
}

// ageFeedback counts every label down by one tick and drops expired ones.
// Only tick() calls it, so labels freeze while the game is paused. Must be
// called under lock.
func (gs *GameState) ageFeedback() {
	kept := gs.feedback[:0]
	for _, f := range gs.feedback {
		f.ticksLeft--
		if f.ticksLeft > 0 {
			kept = append(kept, f)
		}
	}
	gs.feedback = kept
}
