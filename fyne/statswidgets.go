package main

import (
	"image/color"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	cellW     float32 = 24
	cellH     float32 = 18
	rowLabelW float32 = 20
	halfGapW  float32 = 6
	gridGap   float32 = 2
)

var (
	cellOutlineColor  = color.RGBA{R: 0xcd, G: 0xb9, B: 0x9c, A: 0xff}
	cellNewTextColor  = color.RGBA{R: 0x9b, G: 0x8a, B: 0x76, A: 0xff}
	cellLearningColor = color.RGBA{R: 0xe9, G: 0xc4, B: 0x6a, A: 0xff}
	cellMasteredColor = color.RGBA{R: 0x7f, G: 0xa3, B: 0x6b, A: 0xff}
	cellMissedColor   = color.RGBA{R: 0xcc, G: 0x44, B: 0x44, A: 0xff}
)

// fixedColumns lays objects out row by row in columns of fixed widths, so a
// wide cell can never widen its neighbours (container.NewGridWithColumns
// sizes every column to the widest cell). Rows whose objects are all hidden
// take no space.
type fixedColumns struct {
	widths    []float32
	rowHeight float32
	gap       float32
}

func (l *fixedColumns) width() float32 {
	w := l.gap * float32(len(l.widths)-1)
	for _, cw := range l.widths {
		w += cw
	}
	return w
}

func (l *fixedColumns) visibleRows(objs []fyne.CanvasObject) [][]fyne.CanvasObject {
	n := len(l.widths)
	var rows [][]fyne.CanvasObject
	for i := 0; i < len(objs); i += n {
		end := i + n
		if end > len(objs) {
			end = len(objs)
		}
		row := objs[i:end]
		for _, o := range row {
			if o.Visible() {
				rows = append(rows, row)
				break
			}
		}
	}
	return rows
}

func (l *fixedColumns) MinSize(objs []fyne.CanvasObject) fyne.Size {
	rows := len(l.visibleRows(objs))
	if rows == 0 {
		return fyne.NewSize(l.width(), 0)
	}
	return fyne.NewSize(l.width(), float32(rows)*l.rowHeight+float32(rows-1)*l.gap)
}

func (l *fixedColumns) Layout(objs []fyne.CanvasObject, _ fyne.Size) {
	var y float32
	for _, row := range l.visibleRows(objs) {
		var x float32
		for j, o := range row {
			o.Resize(fyne.NewSize(l.widths[j], l.rowHeight))
			o.Move(fyne.NewPos(x, y))
			x += l.widths[j] + l.gap
		}
		y += l.rowHeight + l.gap
	}
}

// gridColumns is the kana grid: row label, five hiragana cells, a spacer and
// five katakana cells – 288 px wide.
func gridColumns() *fixedColumns {
	w := []float32{rowLabelW}
	for i := 0; i < 5; i++ {
		w = append(w, cellW)
	}
	w = append(w, halfGapW)
	for i := 0; i < 5; i++ {
		w = append(w, cellW)
	}
	return &fixedColumns{widths: w, rowHeight: cellH, gap: gridGap}
}

type cellState int

const (
	cellNew      cellState = iota // never answered correctly
	cellLearning                  // 1–2 correct
	cellMastered                  // 3 or more correct
)

// stateFor maps a total correct count to a cell state.
func stateFor(correct int) cellState {
	switch {
	case correct >= 3:
		return cellMastered
	case correct > 0:
		return cellLearning
	default:
		return cellNew
	}
}

// kanaCell is one character in the stats grid.
type kanaCell struct {
	widget.BaseWidget
	bg      *canvas.Rectangle
	text    *canvas.Text
	state   cellState
	missed  bool
	painted bool // false until the first set call, which must always paint
}

func newKanaCell(char string) *kanaCell {
	c := &kanaCell{bg: canvas.NewRectangle(color.Transparent), text: canvas.NewText(char, kanaTextColor)}
	c.bg.CornerRadius = 3
	c.text.TextSize = 12
	if utf8.RuneCountInString(char) > 1 {
		c.text.TextSize = 10
	}
	c.ExtendBaseWidget(c)
	c.set(cellNew, false)
	return c
}

func (c *kanaCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.bg, container.NewCenter(c.text)))
}

func (c *kanaCell) MinSize() fyne.Size {
	return fyne.NewSize(cellW, cellH)
}

// set applies a learning state and the missed-this-session border. It is a
// no-op when neither has changed since the last call, so Update doesn't
// repaint every cell on every tick; the first call always paints.
func (c *kanaCell) set(state cellState, missed bool) {
	if c.painted && state == c.state && missed == c.missed {
		return
	}
	c.state, c.missed, c.painted = state, missed, true
	switch state {
	case cellMastered:
		c.bg.FillColor, c.text.Color = cellMasteredColor, color.White
	case cellLearning:
		c.bg.FillColor, c.text.Color = cellLearningColor, kanaTextColor
	default:
		c.bg.FillColor, c.text.Color = color.Transparent, cellNewTextColor
	}
	c.bg.StrokeColor, c.bg.StrokeWidth = cellOutlineColor, 1
	if state != cellNew {
		c.bg.StrokeWidth = 0
	}
	if missed {
		c.bg.StrokeColor, c.bg.StrokeWidth = cellMissedColor, 2
	}
	c.bg.Refresh()
	c.text.Refresh()
}
