package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// TestThemeIgnoresDarkVariant guards against white text on the beige
// background when the OS runs in dark mode: every colour must come from the
// light palette, whatever variant the system asks for.
func TestThemeIgnoresDarkVariant(t *testing.T) {
	th := WarmPaperTheme()
	names := []fyne.ThemeColorName{
		theme.ColorNameForeground,
		theme.ColorNameInputBackground,
		theme.ColorNamePlaceHolder,
		theme.ColorNameSeparator,
		theme.ColorNameBackground,
	}
	for _, name := range names {
		dark := th.Color(name, theme.VariantDark)
		light := th.Color(name, theme.VariantLight)
		if dark != light {
			t.Errorf("%s: dark variant %v differs from light %v", name, dark, light)
		}
	}
	if fg := th.Color(theme.ColorNameForeground, theme.VariantDark); fg != th.kanaColor(colorKanaText) {
		t.Errorf("foreground = %v, want the kana text colour", fg)
	}
}
