package main

import (
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"kana/store"
)

// setupSettingsForm displays a terminal form to collect user preferences.
func setupSettingsForm(st *store.Store) ([]string, bool, error) {
	selectedRows := defaultRowIDs()
	autoProgress := false

	if st != nil {
		if rows, err := st.SelectedRows(); err == nil && len(rows) > 0 {
			selectedRows = rows
		}
		if auto, err := st.AutoProgress(); err == nil {
			autoProgress = auto
		}
	}

	selection := append([]string(nil), selectedRows...)

	options := make([]huh.Option[string], 0, len(AllKanaRows))
	selectedSet := make(map[string]struct{}, len(selection))
	for _, id := range selection {
		selectedSet[id] = struct{}{}
	}
	for _, row := range AllKanaRows {
		option := huh.NewOption(row.Label, row.ID)
		if _, ok := selectedSet[row.ID]; ok {
			option = option.Selected(true)
		}
		options = append(options, option)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Kana Practice Setup").
				Description("Select the rows you want to study. You can change these later."),
			huh.NewMultiSelect[string]().
				Title("Kana Rows").
				Options(options...).
				Value(&selection).
				Limit(len(options)).
				Height(min(len(options), 8)),
			huh.NewConfirm().
				Title("Enable automatic progression?").
				Affirmative("Yes").
				Negative("No").
				Value(&autoProgress),
		),
	)

	if isAccessibleMode() {
		form.WithAccessible(true)
	}

	if err := form.Run(); err != nil {
		return nil, false, err
	}

	normalized := normalizeRowSelection(selection)
	return normalized, autoProgress, nil
}

func normalizeRowSelection(selection []string) []string {
	if len(selection) == 0 {
		return defaultRowIDs()
	}
	unique := make(map[string]struct{}, len(selection))
	for _, id := range selection {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		unique[id] = struct{}{}
	}

	normalized := make([]string, 0, len(unique))
	for _, row := range AllKanaRows {
		if _, ok := unique[row.ID]; ok {
			normalized = append(normalized, row.ID)
		}
	}

	if len(normalized) == 0 {
		return defaultRowIDs()
	}

	return normalized
}

func isAccessibleMode() bool {
	flag := os.Getenv("KANAGAME_ACCESSIBLE_UI")
	return flag != "" && !strings.EqualFold(flag, "false") && !strings.EqualFold(flag, "0")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
