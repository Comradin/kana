package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

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
