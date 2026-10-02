package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

func TestGridColumnsAre288Wide(t *testing.T) {
	if w := gridColumns().width(); w != 288 {
		t.Fatalf("grid width = %v, want 288", w)
	}
}

func TestFixedColumnsSkipsHiddenRows(t *testing.T) {
	test.NewApp()
	l := &fixedColumns{widths: []float32{10, 10}, rowHeight: 18, gap: 2}
	objs := []fyne.CanvasObject{
		canvas.NewRectangle(nil), canvas.NewRectangle(nil),
		canvas.NewRectangle(nil), canvas.NewRectangle(nil),
	}
	c := container.New(l, objs...)
	if h := c.MinSize().Height; h != 38 {
		t.Fatalf("two rows height = %v, want 38", h)
	}
	objs[2].Hide()
	objs[3].Hide()
	if h := c.MinSize().Height; h != 18 {
		t.Fatalf("one visible row height = %v, want 18", h)
	}
}

func TestKanaCellStates(t *testing.T) {
	test.NewApp()
	c := newKanaCell("か")
	c.set(cellLearning, false)
	if c.bg.FillColor != cellLearningColor {
		t.Fatalf("learning fill = %v", c.bg.FillColor)
	}
	c.set(cellNew, true)
	if c.bg.StrokeColor != cellMissedColor || c.bg.StrokeWidth != 2 {
		t.Fatalf("missed stroke = %v / %v", c.bg.StrokeColor, c.bg.StrokeWidth)
	}
	if newKanaCell("きゃ").text.TextSize != 10 {
		t.Fatal("two-glyph cells use 10 px text")
	}
}
