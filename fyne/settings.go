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

func showSettingsDialog(gs *GameState, statsPanel *StatsPanel, gameCanvas *GameCanvas, inputBar *InputBar, win fyne.Window) {
	gs.mu.Lock()
	selected := make(map[string]bool, len(gs.selectedRows))
	for id, ok := range gs.selectedRows {
		selected[id] = ok
	}
	active := make(map[kanacore.Script]bool, len(gs.activeScripts))
	for s, on := range gs.activeScripts {
		active[s] = on
	}
	currentAuto := gs.autoProgress
	currentLimit := gs.scoreLimit
	gs.mu.Unlock()

	scriptChecks := make(map[kanacore.Script]*widget.Check)
	scriptRow := container.NewHBox(widget.NewLabelWithStyle("Scripts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, info := range kanacore.Scripts() {
		c := widget.NewCheck(info.Label, nil)
		c.SetChecked(active[info.Script])
		scriptChecks[info.Script] = c
		scriptRow.Add(c)
	}

	rowChecks := make(map[string]*widget.Check)
	tabs := container.NewAppTabs()
	for _, info := range kanacore.Scripts() {
		sections := container.NewVBox()
		for _, g := range kanacore.Groups() {
			all, checks, grid := newGroupChecks(kanacore.RowsInGroup(info.Script, g.Group), selected)
			for id, c := range checks {
				rowChecks[id] = c
			}
			sections.Add(container.NewHBox(
				widget.NewLabelWithStyle(g.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				all,
			))
			sections.Add(grid)
		}
		scroll := container.NewVScroll(sections)
		scroll.SetMinSize(fyne.NewSize(460, 280))
		tabs.Append(container.NewTabItem(info.Label, scroll))
	}

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
		scriptRow,
		tabs,
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
		newAuto := autoCheck.Checked
		newLimit := currentLimit
		if n, err := strconv.Atoi(strings.TrimSpace(limitEntry.Text)); err == nil && n >= 0 {
			newLimit = n
		}

		gs.applySettings(newRows, checkedScripts(scriptChecks), newAuto, newLimit)

		gs.mu.Lock()
		snap := gs.snapshot()
		gs.mu.Unlock()
		statsPanel.Update(snap)
		gameCanvas.Refresh()
		showPendingDialogs(gs, statsPanel, inputBar, win)
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

// checkedScripts returns the checked scripts in display order.
func checkedScripts(checks map[kanacore.Script]*widget.Check) []kanacore.Script {
	var scripts []kanacore.Script
	for _, info := range kanacore.Scripts() {
		if c, ok := checks[info.Script]; ok && c.Checked {
			scripts = append(scripts, info.Script)
		}
	}
	return scripts
}

func allChecked(checks []*widget.Check) bool {
	for _, c := range checks {
		if !c.Checked {
			return false
		}
	}
	return len(checks) > 0
}
