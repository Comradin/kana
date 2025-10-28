package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"kana/store"
)

// setupSettingsForm displays a terminal form to collect user preferences.
func setupSettingsForm(st *store.Store) ([]string, bool, int, error) {
	selectedRows := defaultRowIDs()
	autoProgress := false
	scoreLimit := store.DefaultScoreLimit

	if st != nil {
		if rows, err := st.SelectedRows(); err == nil && len(rows) > 0 {
			selectedRows = rows
		}
		if auto, err := st.AutoProgress(); err == nil {
			autoProgress = auto
		}
		if limit, err := st.ScoreLimit(); err == nil {
			scoreLimit = limit
		}
	}

	selection := append([]string(nil), selectedRows...)
	scoreLimitStr := strconv.Itoa(scoreLimit)

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
			huh.NewInput().
				Title("Score limit before the session ends").
				Description("Reach this score to finish. Use 0 for endless practice.").
				Value(&scoreLimitStr).
				Validate(func(v string) error {
					v = strings.TrimSpace(v)
					if v == "" {
						return errors.New("enter a number")
					}
					n, err := strconv.Atoi(v)
					if err != nil {
						return errors.New("enter a valid whole number")
					}
					if n < 0 {
						return errors.New("score limit must be zero or greater")
					}
					return nil
				}),
		),
	)

	if isAccessibleMode() {
		form.WithAccessible(true)
	}

	if err := form.Run(); err != nil {
		return nil, false, 0, err
	}

	normalized := normalizeRowSelection(selection)
	limit := store.DefaultScoreLimit
	if trimmed := strings.TrimSpace(scoreLimitStr); trimmed != "" {
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			limit = parsed
		}
	}

	return normalized, autoProgress, limit, nil
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
