package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
)

func testGroupRows() []kanacore.KanaRow {
	return []kanacore.KanaRow{
		{ID: "r1", Label: "Row 1"},
		{ID: "r2", Label: "Row 2"},
		{ID: "r3", Label: "Row 3"},
	}
}

func TestAllChecked(t *testing.T) {
	tests := []struct {
		name    string
		checked []bool
		want    bool
	}{
		{"empty", nil, false},
		{"all true", []bool{true, true, true}, true},
		{"mixed", []bool{true, false, true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.NewApp()
			checks := make([]*widget.Check, len(tt.checked))
			for i, on := range tt.checked {
				checks[i] = widget.NewCheck("", nil)
				checks[i].SetChecked(on)
			}
			if got := allChecked(checks); got != tt.want {
				t.Errorf("allChecked(%v) = %v, want %v", tt.checked, got, tt.want)
			}
		})
	}
}

func TestNewGroupChecksInitialStateAllSelected(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": true, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)

	if !all.Checked {
		t.Fatalf("expected 'all' checked when every row is selected")
	}
	for _, row := range rows {
		if !checks[row.ID].Checked {
			t.Errorf("expected row %s to stay checked from initial selection", row.ID)
		}
	}
}

func TestNewGroupChecksInitialStateNotAllSelected(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": false, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)

	if all.Checked {
		t.Fatalf("expected 'all' unchecked when not every row is selected")
	}
	if !checks["r1"].Checked || checks["r2"].Checked || !checks["r3"].Checked {
		t.Fatalf("expected initial row state preserved exactly, got r1=%v r2=%v r3=%v",
			checks["r1"].Checked, checks["r2"].Checked, checks["r3"].Checked)
	}
}

func TestNewGroupChecksToggleAllChecksEveryRow(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": false, "r2": false, "r3": false}

	all, checks, _ := newGroupChecks(rows, selected)

	all.SetChecked(true)
	for _, row := range rows {
		if !checks[row.ID].Checked {
			t.Errorf("expected row %s checked after toggling 'all' on", row.ID)
		}
	}

	all.SetChecked(false)
	for _, row := range rows {
		if checks[row.ID].Checked {
			t.Errorf("expected row %s unchecked after toggling 'all' off", row.ID)
		}
	}
}

func TestNewGroupChecksUncheckingOneRowUnchecksAll(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": true, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)
	if !all.Checked {
		t.Fatalf("precondition failed: expected 'all' checked initially")
	}

	checks["r2"].SetChecked(false)

	if all.Checked {
		t.Errorf("expected 'all' unchecked after unchecking row r2")
	}
}

func TestNewGroupChecksCheckingLastRowChecksAll(t *testing.T) {
	test.NewApp()
	rows := testGroupRows()
	selected := map[string]bool{"r1": true, "r2": false, "r3": true}

	all, checks, _ := newGroupChecks(rows, selected)
	if all.Checked {
		t.Fatalf("precondition failed: expected 'all' unchecked initially")
	}

	checks["r2"].SetChecked(true)

	if !all.Checked {
		t.Errorf("expected 'all' checked after checking the last unchecked row")
	}
}

func TestCheckedRowIDsFollowsRowOrder(t *testing.T) {
	test.NewApp()
	checks := map[string]*widget.Check{
		"ky":     widget.NewCheck("", nil),
		"vowels": widget.NewCheck("", nil),
		"g":      widget.NewCheck("", nil),
		"k":      widget.NewCheck("", nil),
	}
	checks["ky"].SetChecked(true)
	checks["vowels"].SetChecked(true)
	checks["g"].SetChecked(true)

	got := checkedRowIDs(checks)
	if !equalIDs(got, []string{"vowels", "g", "ky"}) {
		t.Fatalf("checkedRowIDs = %v, want [vowels g ky]", got)
	}
}
