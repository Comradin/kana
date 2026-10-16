package main

import (
	"image/color"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"kana/kanacore"
	"kana/vocab"
)

const (
	tileW     float32 = 52 // single glyph
	tileWideW float32 = 84 // two glyphs (yōon)
	tileH     float32 = 60

	wordTextSize    float32 = 32
	minWordTextSize float32 = 10
	wordTilePad     float32 = 12 // space left and right of a word
)

var (
	tileFaceColor   = color.RGBA{R: 0xee, G: 0xdf, B: 0xc0, A: 0xff}
	tileShadowColor = color.RGBA{R: 0xb8, G: 0x95, B: 0x6a, A: 0xff}
	kanaTextColor   = color.RGBA{R: 0x2c, G: 0x1a, B: 0x0e, A: 0xff}

	wordTileFaceColor   = color.RGBA{R: 0xdc, G: 0xe6, B: 0xd2, A: 0xff}
	wordTileBorderColor = color.RGBA{R: 0x5e, G: 0x7f, B: 0x52, A: 0xff}
)

// tileWidthFor returns the tile width for a kana, wider for two-glyph kana.
func tileWidthFor(char string) float32 {
	if utf8.RuneCountInString(char) > 1 {
		return tileWideW
	}
	return tileW
}

// KanaTile is a falling card rendered as three canvas objects. It carries
// either a kana or, when word is set, a word; a word tile's kana is unset.
type KanaTile struct {
	kana        kanacore.Kana
	word        *vocab.Entry // nil for a kana tile
	pos         fyne.Position
	width       float32
	shadow      *canvas.Rectangle
	face        *canvas.Rectangle
	text        *canvas.Text
	fallSeconds float32 // time to cross the canvas; see fallStep
}

// newKanaTile builds a kana tile in the kana font kf.
func newKanaTile(k kanacore.Kana, kf fontChoice) *KanaTile {
	w := tileWidthFor(k.Char)

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(tileFaceColor)
	face.Resize(fyne.NewSize(w, tileH))

	text := canvas.NewText(k.Char, kanaTextColor)
	text.TextSize = 32
	setKanaFont(text, kf)
	text.Resize(text.MinSize())

	t := &KanaTile{kana: k, width: w, shadow: shadow, face: face, text: text}
	t.Move(fyne.NewPos(0, 0))
	return t
}

// newWordTile builds a word tile in the kana font kf: a sage face with a
// border, as wide as the word plus padding, so even a one-unit word (か)
// does not look like a kana tile. A word wider than maxW (the field width)
// gets a smaller font, down to minWordTextSize, so the tile fits.
func newWordTile(e vocab.Entry, maxW float32, kf fontChoice) *KanaTile {
	text := canvas.NewText(e.Kana, kanaTextColor)
	text.TextSize = wordTextSize
	setKanaFont(text, kf)
	size := text.MinSize()
	avail := maxW - 2*wordTilePad
	for size.Width > avail && text.TextSize > minWordTextSize {
		text.TextSize--
		size = text.MinSize()
	}
	text.Resize(size)
	w := size.Width + 2*wordTilePad

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(wordTileFaceColor)
	face.StrokeColor = wordTileBorderColor
	face.StrokeWidth = 2
	face.Resize(fyne.NewSize(w, tileH))

	word := e
	t := &KanaTile{word: &word, width: w, shadow: shadow, face: face, text: text}
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
