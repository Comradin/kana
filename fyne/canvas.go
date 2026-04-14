package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

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
	return &gameCanvasRenderer{canvas: gc, bg: bg}
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
}

func (r *gameCanvasRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
}

func (r *gameCanvasRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 300)
}

func (r *gameCanvasRenderer) Refresh() {
	r.bg.FillColor = theme.Color(theme.ColorNameBackground)
	canvas.Refresh(r.bg)
}

func (r *gameCanvasRenderer) Destroy() {}

func (r *gameCanvasRenderer) Objects() []fyne.CanvasObject {
	snap, _ := r.canvas.state.objectSnapshot.Load().([]fyne.CanvasObject)
	if snap == nil {
		return []fyne.CanvasObject{r.bg}
	}
	// Prepend background so tiles render on top.
	all := make([]fyne.CanvasObject, 0, len(snap)+1)
	all = append(all, r.bg)
	all = append(all, snap...)
	return all
}
