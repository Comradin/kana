package main

import (
	"fmt"

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
	ActiveScripts map[kanacore.Script]bool
}

// StatsPanel shows hiragana progress, active rows, and missed characters.
type StatsPanel struct {
	widget.BaseWidget

	charLabels map[string]*widget.Label
	rowLabels  map[string]*widget.Label
	missLabels map[string]*widget.Label
	missEmpty  *widget.Label
	rowBox     *fyne.Container
	missBox    *fyne.Container
	container  *container.Scroll

	// rowCells maps row ID to all 6 labels in that row (row-label + 5 char cells).
	// Used to show/hide entire rows together.
	rowCells map[string][6]*widget.Label

	// basicGrid is the always-visible 6-column grid for the basic kana group.
	basicGrid *fyne.Container

	// groupSections holds the label+grid section for each non-basic group,
	// shown or hidden together based on whether any row in the group is selected.
	groupSections map[kanacore.Group]*fyne.Container
}

// vowelColIndex returns the column index (0–4) for a kana based on its romaji vowel ending.
// Returns -1 if the mapping is unknown.
func vowelColIndex(romaji string) int {
	if len(romaji) == 0 {
		return -1
	}
	switch romaji[len(romaji)-1] {
	case 'a':
		return 0
	case 'i':
		return 1
	case 'u':
		return 2
	case 'e':
		return 3
	case 'o':
		return 4
	}
	// "n" (ん) maps to column 0
	if romaji == "n" {
		return 0
	}
	return -1
}

// rowShortLabel returns the short consonant label shown at the left of each row.
func rowShortLabel(rowID string) string {
	switch rowID {
	case "vowels":
		return "–"
	case "n-only":
		return "n"
	case "sy":
		return "sh"
	default:
		return rowID
	}
}

func groupLabel(g kanacore.Group) string {
	for _, info := range kanacore.Groups() {
		if info.Group == g {
			return info.Label
		}
	}
	return string(g)
}

func newStatsPanel() *StatsPanel {
	p := &StatsPanel{
		charLabels:    make(map[string]*widget.Label),
		rowLabels:     make(map[string]*widget.Label),
		missLabels:    make(map[string]*widget.Label),
		rowCells:      make(map[string][6]*widget.Label),
		groupSections: make(map[kanacore.Group]*fyne.Container),
	}

	// Build one 6-column progress grid per group, so a group's label never
	// widens another group's columns (Fyne sizes every grid cell to the
	// widest cell in that same grid).
	sections := container.NewVBox()

	for _, info := range kanacore.Groups() {
		g := info.Group
		gridItems := make([]fyne.CanvasObject, 0)

		if g == kanacore.GroupBasic {
			// Header row: blank + vowel headers.
			headerCells := []*widget.Label{
				widget.NewLabel(""),
				widget.NewLabel("a"),
				widget.NewLabel("i"),
				widget.NewLabel("u"),
				widget.NewLabel("e"),
				widget.NewLabel("o"),
			}
			for _, lbl := range headerCells {
				gridItems = append(gridItems, lbl)
			}
		}

		for _, row := range kanacore.RowsInGroup(kanacore.ScriptHiragana, g) {
			// Create the row-label cell.
			rowLbl := widget.NewLabel(rowShortLabel(row.ID))

			// Create 5 placeholder cells (one per vowel column), initially "".
			// cells[0..4] correspond to columns a/i/u/e/o; only cells that
			// get a real kana below are replaced with a label starting "-".
			cells := [5]*widget.Label{}
			for i := range cells {
				cells[i] = widget.NewLabel("")
			}

			// Place each character into the correct column slot.
			for _, e := range row.Entries {
				char := e.Char
				col := vowelColIndex(e.Romaji)
				if col < 0 || col > 4 {
					continue
				}
				lbl := widget.NewLabel("-")
				p.charLabels[char] = lbl
				cells[col] = lbl
			}

			// Store all 6 cells for this row so Update can show/hide them.
			var row6 [6]*widget.Label
			row6[0] = rowLbl
			for i, c := range cells {
				row6[i+1] = c
			}
			p.rowCells[row.ID] = row6

			// Add to grid.
			gridItems = append(gridItems, rowLbl)
			for _, c := range cells {
				gridItems = append(gridItems, c)
			}

			// Initially hide all row cells; Update() will show active ones.
			for _, lbl := range row6 {
				lbl.Hide()
			}
		}

		grid := container.NewGridWithColumns(6, gridItems...)

		if g == kanacore.GroupBasic {
			p.basicGrid = grid
			sections.Add(grid)
			continue
		}

		label := widget.NewLabelWithStyle(groupLabel(g), fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
		section := container.NewVBox(label, grid)
		section.Hide()
		p.groupSections[g] = section
		sections.Add(section)
	}

	// Pre-create row labels (one per known row), hidden by default.
	p.rowBox = container.NewVBox()
	for _, row := range kanacore.AllKanaRows {
		lbl := widget.NewLabel("")
		lbl.Hide()
		p.rowLabels[row.ID] = lbl
		p.rowBox.Add(lbl)
	}

	// Pre-create missed-kana labels (one per character), hidden by default.
	p.missBox = container.NewVBox()
	for _, row := range kanacore.AllKanaRows {
		for _, char := range row.Characters() {
			lbl := widget.NewLabel("")
			lbl.Hide()
			p.missLabels[char] = lbl
			p.missBox.Add(lbl)
		}
	}
	p.missEmpty = widget.NewLabel("None yet!")
	p.missBox.Add(p.missEmpty)

	p.container = container.NewVScroll(container.NewVBox(
		widget.NewLabel("PROGRESS"),
		sections,
		widget.NewSeparator(),
		widget.NewLabel("ACTIVE ROWS"),
		p.rowBox,
		widget.NewSeparator(),
		widget.NewLabel("MISSED"),
		p.missBox,
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

	for _, row := range kanacore.AllKanaRows {
		row6, ok := p.rowCells[row.ID]
		if !ok {
			continue
		}
		if snap.SelectedRows[row.ID] {
			for _, lbl := range row6 {
				lbl.Show()
			}
		} else {
			for _, lbl := range row6 {
				lbl.Hide()
			}
		}
	}

	for _, row := range kanacore.AllKanaRows {
		lbl, ok := p.rowLabels[row.ID]
		if !ok {
			continue
		}
		if snap.SelectedRows[row.ID] {
			lbl.SetText("• " + row.Label)
			lbl.Show()
		} else {
			lbl.SetText("")
			lbl.Hide()
		}
	}

	for group, section := range p.groupSections {
		visible := false
		for _, row := range kanacore.RowsInGroup(kanacore.ScriptHiragana, group) {
			if snap.SelectedRows[row.ID] {
				visible = true
				break
			}
		}
		if visible {
			section.Show()
		} else {
			section.Hide()
		}
	}

	seen := make(map[string]bool)
	for _, k := range snap.MissedKanas {
		if seen[k.Char] {
			continue
		}
		seen[k.Char] = true
		if lbl, ok := p.missLabels[k.Char]; ok {
			lbl.SetText(k.Char + " (" + k.Romaji + ")")
			lbl.Show()
		}
	}
	for char, lbl := range p.missLabels {
		if !seen[char] {
			lbl.Hide()
		}
	}
	if len(seen) == 0 {
		p.missEmpty.Show()
	} else {
		p.missEmpty.Hide()
	}

	p.container.Refresh()
}
