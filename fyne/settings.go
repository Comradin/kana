package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

func showSettingsDialog(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, win fyne.Window) {
	gs.mu.Lock()
	selected := make(map[string]bool, len(gs.selectedRows))
	for id, ok := range gs.selectedRows {
		selected[id] = ok
	}
	currentAuto := gs.autoProgress
	currentLimit := gs.scoreLimit
	gs.mu.Unlock()

	rowChecks := make(map[string]*widget.Check)
	sections := container.NewVBox()
	for _, g := range kanacore.Groups() {
		groupRows := kanacore.RowsInGroup(g.Group)
		all, checks, grid := newGroupChecks(groupRows, selected)
		for id, c := range checks {
			rowChecks[id] = c
		}
		sections.Add(container.NewHBox(
			widget.NewLabelWithStyle(g.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			all,
		))
		sections.Add(grid)
	}
	rowScroll := container.NewVScroll(sections)
	rowScroll.SetMinSize(fyne.NewSize(460, 320))

	autoCheck := widget.NewCheck("Enable auto-progression", nil)
	autoCheck.SetChecked(currentAuto)

	limitEntry := widget.NewEntry()
	limitEntry.SetText(strconv.Itoa(currentLimit))
	limitEntry.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return fmt.Errorf("enter a whole number ≥ 0")
		}
		return nil
	}

	form := container.NewVBox(
		widget.NewLabel("Kana Rows"),
		rowScroll,
		widget.NewSeparator(),
		autoCheck,
		widget.NewSeparator(),
		widget.NewLabel("Score limit (0 = endless)"),
		limitEntry,
	)

	dialog.ShowCustomConfirm("Settings", "Save", "Cancel", form, func(save bool) {
		if !save {
			return
		}

		if err := limitEntry.Validate(); err != nil {
			dialog.ShowError(err, win)
			return
		}

		newRows := checkedRowIDs(rowChecks)
		if len(newRows) == 0 {
			newRows = kanacore.DefaultRowIDs()
		}

		newAuto := autoCheck.Checked
		newLimit := currentLimit
		if n, err := strconv.Atoi(strings.TrimSpace(limitEntry.Text)); err == nil && n >= 0 {
			newLimit = n
		}

		// Apply under lock
		gs.mu.Lock()
		gs.setSelectedRowsLocked(newRows)
		gs.autoProgress = newAuto
		gs.scoreLimit = newLimit

		// Remove in-flight tiles whose row is now deselected
		filtered := gs.tiles[:0]
		for _, t := range gs.tiles {
			if rowID, ok := kanacore.CharToRow[t.kana.Char]; ok && !gs.selectedRows[rowID] {
				continue
			}
			filtered = append(filtered, t)
		}
		gs.tiles = filtered

		// Rebuild the canvas-object snapshot so the renderer reflects removals.
		gs.buildSnapshot()
		snap := gs.snapshot()
		gs.mu.Unlock()

		// Persist to store
		if gs.store != nil {
			_ = gs.store.SaveAutoProgress(newAuto)
			_ = gs.store.SaveScoreLimit(newLimit)
		}

		statsPanel.Update(snap)
		gameCanvas.Refresh()
	}, win)
}

// newGroupChecks builds the per-row checks and the "all" check for one kana
// group, keeping them in sync: checking/unchecking "all" sets every row, and
// checking/unchecking a row updates "all" to match. A per-group syncing flag
// guards against the two handlers recursing into each other.
func newGroupChecks(rows []kanacore.KanaRow, selected map[string]bool) (all *widget.Check, checks map[string]*widget.Check, grid *fyne.Container) {
	checks = make(map[string]*widget.Check, len(rows))
	checkList := make([]*widget.Check, 0, len(rows))
	grid = container.NewGridWithColumns(2)

	for _, row := range rows {
		c := widget.NewCheck(row.Label, nil)
		c.SetChecked(selected[row.ID])
		checks[row.ID] = c
		checkList = append(checkList, c)
		grid.Add(c)
	}

	all = widget.NewCheck("all", nil)
	all.SetChecked(allChecked(checkList))

	syncing := false

	all.OnChanged = func(on bool) {
		if syncing {
			return
		}
		syncing = true
		for _, c := range checkList {
			c.SetChecked(on)
		}
		syncing = false
	}

	for _, c := range checkList {
		c.OnChanged = func(bool) {
			if syncing {
				return
			}
			syncing = true
			all.SetChecked(allChecked(checkList))
			syncing = false
		}
	}

	return all, checks, grid
}

// checkedRowIDs returns the IDs of checked rows in progression order.
func checkedRowIDs(checks map[string]*widget.Check) []string {
	ids := make([]string, 0, len(checks))
	for _, row := range kanacore.AllKanaRows {
		if c, ok := checks[row.ID]; ok && c.Checked {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

func allChecked(checks []*widget.Check) bool {
	for _, c := range checks {
		if !c.Checked {
			return false
		}
	}
	return len(checks) > 0
}
