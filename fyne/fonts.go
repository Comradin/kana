package main

import (
	"bytes"
	_ "embed"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/go-text/typesetting/font"
)

// The kana fonts ship with the app (SIL OFL 1.1, see fonts/OFL-*.txt);
// scripts/fetch-fonts.sh downloads and checks them.

//go:embed fonts/NotoSansJP-Regular.otf
var notoRegularData []byte

//go:embed fonts/NotoSansJP-Bold.otf
var notoBoldData []byte

//go:embed fonts/MPLUSRounded1c-Regular.ttf
var mplusRegularData []byte

//go:embed fonts/MPLUSRounded1c-Bold.ttf
var mplusBoldData []byte

// The font resources are created once, with distinct names: Fyne caches a
// custom font by its resource and never evicts it.
var (
	notoRegular  fyne.Resource = fyne.NewStaticResource("NotoSansJP-Regular.otf", notoRegularData)
	notoBold     fyne.Resource = fyne.NewStaticResource("NotoSansJP-Bold.otf", notoBoldData)
	mplusRegular fyne.Resource = fyne.NewStaticResource("MPLUSRounded1c-Regular.ttf", mplusRegularData)
	mplusBold    fyne.Resource = fyne.NewStaticResource("MPLUSRounded1c-Bold.ttf", mplusBoldData)
)

// fontChoice is a kana font the learner can pick. Without resources it is
// the theme font with the system fallback.
type fontChoice struct {
	id      string // the kana_font setting
	label   string
	regular fyne.Resource
	bold    fyne.Resource
}

// fontChoices are the kana fonts in the order the settings show them.
var fontChoices = []fontChoice{
	{id: "noto", label: "Noto Sans JP", regular: notoRegular, bold: notoBold},
	{id: "mplus", label: "M PLUS Rounded 1c", regular: mplusRegular, bold: mplusBold},
	{id: "system", label: "System"},
}

// fontPreview is the text each choice is previewed with: kana whose stroke
// ends and curves show the difference between the fonts (き is open in
// some fonts, as in handwriting, and closed in others).
const fontPreview = "きふそ ツシー"

// fontByID returns the choice for a kana_font setting; an unknown value
// gives Noto Sans JP, the default.
func fontByID(id string) fontChoice {
	for _, c := range fontChoices {
		if c.id == id {
			return c
		}
	}
	return fontChoices[0]
}

// fontFaces parses each bundled font once, for the coverage check.
var fontFaces = sync.OnceValue(func() map[fyne.Resource]*font.Font {
	out := map[fyne.Resource]*font.Font{}
	for _, res := range []fyne.Resource{notoRegular, notoBold, mplusRegular, mplusBold} {
		if face, err := font.ParseTTF(bytes.NewReader(res.Content())); err == nil {
			out[res] = face.Font
		}
	}
	return out
})

// covers reports whether res has a glyph for every rune of text.
func covers(res fyne.Resource, text string) bool {
	f := fontFaces()[res]
	if f == nil {
		return false
	}
	for _, r := range text {
		if _, ok := f.NominalGlyph(r); !ok {
			return false
		}
	}
	return true
}

// source returns the font file for text: the bold or regular one if it has
// every rune of text, else nil (the theme font, which falls back to system
// fonts rune by rune). So a text never mixes fonts.
func (c fontChoice) source(text string, bold bool) fyne.Resource {
	res := c.regular
	if bold {
		res = c.bold
	}
	if res == nil || !covers(res, text) {
		return nil
	}
	return res
}

// setKanaFont sets txt's font source for its text and style. Call it
// before measuring txt: MinSize uses the font source, fyne.MeasureText
// does not.
func setKanaFont(txt *canvas.Text, c fontChoice) {
	txt.FontSource = c.source(txt.Text, txt.TextStyle.Bold)
}

// fitKanaText sets txt's kana font and fits it into maxW (see fitText). The
// font is checked again on the final text, which may end in "…".
func fitKanaText(txt *canvas.Text, c fontChoice, maxW, minSize float32) {
	setKanaFont(txt, c)
	fitText(txt, maxW, minSize)
	if src := c.source(txt.Text, txt.TextStyle.Bold); src != txt.FontSource {
		txt.FontSource = src
		fitText(txt, maxW, minSize)
	}
}
