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
	currentRowIDs := make([]string, 0, len(gs.selectedRows))
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] {
			currentRowIDs = append(currentRowIDs, row.ID)
		}
	}
	currentAuto := gs.autoProgress
	currentLimit := gs.scoreLimit
	gs.mu.Unlock()

	// Build options list (labels)
	options := make([]string, len(kanacore.AllKanaRows))
	for i, row := range kanacore.AllKanaRows {
		options[i] = row.Label
	}

	// Currently selected as labels
	selectedLabels := make([]string, 0, len(currentRowIDs))
	for _, id := range currentRowIDs {
		for _, row := range kanacore.AllKanaRows {
			if row.ID == id {
				selectedLabels = append(selectedLabels, row.Label)
			}
		}
	}

	rowCheck := widget.NewCheckGroup(options, nil)
	rowCheck.SetSelected(selectedLabels)

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
		rowCheck,
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

		// Map selected labels back to IDs
		labelToID := make(map[string]string)
		for _, row := range kanacore.AllKanaRows {
			labelToID[row.Label] = row.ID
		}
		newRows := make([]string, 0)
		for _, lbl := range rowCheck.Selected {
			if id, ok := labelToID[lbl]; ok {
				newRows = append(newRows, id)
			}
		}
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
		gs.applySelectedRows(newRows)
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
			_ = gs.store.SaveSelectedRows(newRows)
			_ = gs.store.SaveAutoProgress(newAuto)
			_ = gs.store.SaveScoreLimit(newLimit)
		}

		statsPanel.Update(snap)
		gameCanvas.Refresh()
	}, win)
}
