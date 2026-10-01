package main

import (
	"image/color"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"kana/kanacore"
)

const (
	tileW     float32 = 52 // single glyph
	tileWideW float32 = 84 // two glyphs (yōon)
	tileH     float32 = 60
)

var (
	tileFaceColor   = color.RGBA{R: 0xee, G: 0xdf, B: 0xc0, A: 0xff}
	tileShadowColor = color.RGBA{R: 0xb8, G: 0x95, B: 0x6a, A: 0xff}
	kanaTextColor   = color.RGBA{R: 0x2c, G: 0x1a, B: 0x0e, A: 0xff}
)

// tileWidthFor returns the tile width for a kana, wider for two-glyph kana.
func tileWidthFor(char string) float32 {
	if utf8.RuneCountInString(char) > 1 {
		return tileWideW
	}
	return tileW
}

// KanaTile is a falling kana card rendered as three canvas objects.
type KanaTile struct {
	kana   kanacore.Kana
	pos    fyne.Position
	width  float32
	shadow *canvas.Rectangle
	face   *canvas.Rectangle
	text   *canvas.Text
}

func newKanaTile(k kanacore.Kana) *KanaTile {
	w := tileWidthFor(k.Char)

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(tileFaceColor)
	face.Resize(fyne.NewSize(w, tileH))

	text := canvas.NewText(k.Char, kanaTextColor)
	text.TextSize = 32
	text.Resize(fyne.MeasureText(text.Text, text.TextSize, text.TextStyle))

	t := &KanaTile{kana: k, width: w, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// Width returns the tile's width in device-independent pixels.
func (t *KanaTile) Width() float32 {
	return t.width
}

// Move updates the positions of all three canvas objects atomically.
func (t *KanaTile) Move(pos fyne.Position) {
	t.pos = pos
	t.shadow.Move(fyne.NewPos(pos.X+3, pos.Y+3))
	t.face.Move(pos)
	ts := t.text.Size()
	t.text.Move(fyne.NewPos(pos.X+(t.width-ts.Width)/2, pos.Y+(tileH-ts.Height)/2))
}

// Objects returns the canvas objects for the renderer, shadow first so face renders on top.
func (t *KanaTile) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{t.shadow, t.face, t.text}
}
