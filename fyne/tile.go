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

	meaningMarkerText          = "?" // the sign on a word tile that asks for the meaning
	meaningMarkerSize  float32 = 13
	meaningMarkerInset float32 = 3 // from the face's top and right edges
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

// KanaTile is a falling card rendered as three canvas objects (four for a
// meaning tile). It carries either a kana or, when word is set, a word; a
// word tile's kana is unset.
type KanaTile struct {
	kana        kanacore.Kana
	word        *vocab.Entry // nil for a kana tile
	pos         fyne.Position
	width       float32
	shadow      *canvas.Rectangle
	face        *canvas.Rectangle
	text        *canvas.Text
	marker      *canvas.Text // a meaning tile's marker; nil otherwise
	ask         wordKind     // what a word tile asks for; readingKind for kana tiles
	fallSeconds float32      // time to cross the canvas; see fallStep
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

// newWordTile builds a word tile that asks for the word's reading, in the
// kana font kf: a sage face with a border, as wide as the word's shown form
// plus padding, so even a one-unit word (か) does not look like a kana tile.
// A word wider than maxW (the field width) gets a smaller font, down to
// minWordTextSize, so the tile fits.
func newWordTile(e vocab.Entry, maxW float32, kf fontChoice) *KanaTile {
	return buildWordTile(e, maxW, kf, nil)
}

// newMeaningTile builds a word tile that asks for the word's meaning: a word
// tile with a small "?" in its top-right corner, so a one-kana word (き) is
// not taken for a kana tile. The tile is wider by the marker, so the marker
// sits beside the centred word and never covers it; it is drawn in the
// border colour on the tile's own face, so it reads whatever the window
// background.
func newMeaningTile(e vocab.Entry, maxW float32, kf fontChoice) *KanaTile {
	m := canvas.NewText(meaningMarkerText, wordTileBorderColor)
	m.TextSize = meaningMarkerSize
	m.TextStyle = fyne.TextStyle{Bold: true}
	m.Resize(m.MinSize())
	t := buildWordTile(e, maxW, kf, m)
	t.ask = meaningKind
	return t
}

// buildWordTile builds a word tile (see newWordTile) with an optional
// marker, for which the tile is wider by the marker's width.
func buildWordTile(e vocab.Entry, maxW float32, kf fontChoice, marker *canvas.Text) *KanaTile {
	var extra float32
	if marker != nil {
		extra = marker.Size().Width
	}
	text := canvas.NewText(e.Shown(), kanaTextColor)
	text.TextSize = wordTextSize
	setKanaFont(text, kf)
	size := text.MinSize()
	avail := maxW - 2*wordTilePad - extra
	for size.Width > avail && text.TextSize > minWordTextSize {
		text.TextSize--
		size = text.MinSize()
	}
	text.Resize(size)
	w := size.Width + 2*wordTilePad + extra

	shadow := canvas.NewRectangle(tileShadowColor)
	shadow.Resize(fyne.NewSize(w, tileH))

	face := canvas.NewRectangle(wordTileFaceColor)
	face.StrokeColor = wordTileBorderColor
	face.StrokeWidth = 2
	face.Resize(fyne.NewSize(w, tileH))

	word := e
	t := &KanaTile{word: &word, width: w, shadow: shadow, face: face, text: text, marker: marker}
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
	if t.marker != nil {
		ms := t.marker.Size()
		t.marker.Move(fyne.NewPos(pos.X+t.width-ms.Width-meaningMarkerInset, pos.Y+meaningMarkerInset))
	}
}

// Objects returns the canvas objects for the renderer, shadow first so the
// face renders on top, and a meaning tile's marker last.
func (t *KanaTile) Objects() []fyne.CanvasObject {
	if t.marker != nil {
		return []fyne.CanvasObject{t.shadow, t.face, t.text, t.marker}
	}
	return []fyne.CanvasObject{t.shadow, t.face, t.text}
}
