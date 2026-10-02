package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// pauseHintText is shown in the middle of the play field while the learner
// has paused.
const pauseHintText = "Paused — press Esc to continue"

// GameCanvas is the falling-tile play field.
type GameCanvas struct {
	widget.BaseWidget
	state *GameState
}

func newGameCanvas(state *GameState) *GameCanvas {
	gc := &GameCanvas{state: state}
	gc.ExtendBaseWidget(gc)
	return gc
}

func (gc *GameCanvas) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	hint := canvas.NewText(pauseHintText, kanaTextColor)
	hint.TextSize = 24
	hint.TextStyle = fyne.TextStyle{Bold: true}
	return &gameCanvasRenderer{canvas: gc, bg: bg, hint: hint}
}

// Resize stores dimensions so the game loop can use them for bounds checking.
func (gc *GameCanvas) Resize(size fyne.Size) {
	gc.BaseWidget.Resize(size)
	gc.state.mu.Lock()
	gc.state.canvasW = size.Width
	gc.state.canvasH = size.Height
	gc.state.mu.Unlock()
}

type gameCanvasRenderer struct {
	canvas *GameCanvas
	bg     *canvas.Rectangle
	hint   *canvas.Text
}

func (r *gameCanvasRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
	r.layoutHint(size)
}

// layoutHint centres the pause hint in the play field.
func (r *gameCanvasRenderer) layoutHint(size fyne.Size) {
	hs := r.hint.MinSize()
	r.hint.Resize(hs)
	r.hint.Move(fyne.NewPos((size.Width-hs.Width)/2, (size.Height-hs.Height)/2))
}

func (r *gameCanvasRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 300)
}

func (r *gameCanvasRenderer) Refresh() {
	r.bg.FillColor = theme.Color(theme.ColorNameBackground)
	canvas.Refresh(r.bg)
	r.layoutHint(r.canvas.Size())
	canvas.Refresh(r.hint)
}

func (r *gameCanvasRenderer) Destroy() {}

func (r *gameCanvasRenderer) Objects() []fyne.CanvasObject {
	snap, _ := r.canvas.state.objectSnapshot.Load().([]fyne.CanvasObject)
	// Prepend background so tiles render on top.
	all := make([]fyne.CanvasObject, 0, len(snap)+2)
	all = append(all, r.bg)
	all = append(all, snap...)
	if r.canvas.state.showPauseHint.Load() {
		all = append(all, r.hint)
	}
	return all
}
