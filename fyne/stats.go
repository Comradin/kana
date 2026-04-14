package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/store"
)

// StatsSnapshot is a lock-free copy of the game state fields needed by the panel.
type StatsSnapshot struct {
	SessionStats  map[string]store.KanaStats
	SelectedRows  map[string]bool
	MissedKanas   []kanacore.Kana
	Score         int
	ScoreLimit    int
	Missed        int
	UnlockMessage string
	UnlockAt      time.Time
}

// StatsPanel shows hiragana progress, active rows, and missed characters.
type StatsPanel struct {
	widget.BaseWidget

	charLabels  map[string]*widget.Label
	rowBox      *fyne.Container
	missBox     *fyne.Container
	unlockLabel *widget.Label
	container   *container.Scroll
}

func newStatsPanel() *StatsPanel {
	p := &StatsPanel{
		charLabels:  make(map[string]*widget.Label),
		unlockLabel: widget.NewLabel(""),
	}

	// Build progress grid (5 columns: a, i, u, e, o)
	gridItems := make([]fyne.CanvasObject, 0)
	for _, row := range kanacore.AllKanaRows {
		for _, char := range row.Characters {
			lbl := widget.NewLabel("-")
			p.charLabels[char] = lbl
			gridItems = append(gridItems, lbl)
		}
	}
	grid := container.NewGridWithColumns(5, gridItems...)

	p.rowBox = container.NewVBox()
	p.missBox = container.NewVBox()

	p.container = container.NewVScroll(container.NewVBox(
		widget.NewLabel("PROGRESS"),
		grid,
		widget.NewSeparator(),
		widget.NewLabel("ACTIVE ROWS"),
		p.rowBox,
		widget.NewSeparator(),
		widget.NewLabel("MISSED"),
		p.missBox,
		p.unlockLabel,
	))

	p.ExtendBaseWidget(p)
	return p
}

func (p *StatsPanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

// Update refreshes all labels from the snapshot.
func (p *StatsPanel) Update(snap StatsSnapshot) {
	for char, lbl := range p.charLabels {
		count := snap.SessionStats[char].CorrectCount
		if count > 0 {
			lbl.SetText(fmt.Sprintf("%d", count))
		} else {
			lbl.SetText("-")
		}
	}

	p.rowBox.RemoveAll()
	for _, row := range kanacore.AllKanaRows {
		if snap.SelectedRows[row.ID] {
			p.rowBox.Add(widget.NewLabel("• " + row.Label))
		}
	}

	p.missBox.RemoveAll()
	seen := make(map[string]bool)
	for _, k := range snap.MissedKanas {
		if !seen[k.Char] {
			p.missBox.Add(widget.NewLabel(k.Char + " (" + k.Romaji + ")"))
			seen[k.Char] = true
		}
	}
	if len(snap.MissedKanas) == 0 {
		p.missBox.Add(widget.NewLabel("None yet!"))
	}

	if snap.UnlockMessage != "" && time.Since(snap.UnlockAt) < 5*time.Second {
		p.unlockLabel.SetText(snap.UnlockMessage)
	} else {
		p.unlockLabel.SetText("")
	}

	p.container.Refresh()
}
