package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"kana/kanacore"
)

const (
	tileW float32 = 52
	tileH float32 = 60
)

// KanaTile is a falling kana card rendered as three canvas objects.
type KanaTile struct {
	kana   kanacore.Kana
	pos    fyne.Position
	shadow *canvas.Rectangle
	face   *canvas.Rectangle
	text   *canvas.Text
}

func newKanaTile(k kanacore.Kana) *KanaTile {
	shadow := canvas.NewRectangle(color.RGBA{R: 0xb8, G: 0x95, B: 0x6a, A: 0xff})
	shadow.Resize(fyne.NewSize(tileW, tileH))

	face := canvas.NewRectangle(color.RGBA{R: 0xee, G: 0xdf, B: 0xc0, A: 0xff})
	face.Resize(fyne.NewSize(tileW, tileH))

	text := canvas.NewText(k.Char, color.RGBA{R: 0x2c, G: 0x1a, B: 0x0e, A: 0xff})
	text.TextSize = 32
	text.Alignment = fyne.TextAlignCenter

	t := &KanaTile{kana: k, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// Move updates the positions of all three canvas objects atomically.
func (t *KanaTile) Move(pos fyne.Position) {
	t.pos = pos
	t.shadow.Move(fyne.NewPos(pos.X+3, pos.Y+3))
	t.face.Move(pos)
	// Centre text within face (approximate — kana glyphs ~18px wide at TextSize 32)
	textX := pos.X + (tileW-18)/2
	t.text.Move(fyne.NewPos(textX, pos.Y+10))
}

// Objects returns the canvas objects for the renderer, shadow first so face renders on top.
func (t *KanaTile) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{t.shadow, t.face, t.text}
}
