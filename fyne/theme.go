package main

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type themeColorKey int

const (
	colorBackground themeColorKey = iota
	colorTileFace
	colorTileShadow
	colorKanaText
	colorStatsBg
	colorInputBg
	colorAccent
	colorMiss
)

type KanaTheme struct {
	name   string
	colors map[themeColorKey]color.Color
}

func WarmPaperTheme() *KanaTheme {
	return &KanaTheme{
		name: "Warm Paper",
		colors: map[themeColorKey]color.Color{
			colorBackground: hexColor("#f0e6d3"),
			colorTileFace:   hexColor("#eedfc0"),
			colorTileShadow: hexColor("#b8956a"),
			colorKanaText:   hexColor("#2c1a0e"),
			colorStatsBg:    hexColor("#e8dbc8"),
			colorInputBg:    hexColor("#e0d4bc"),
			colorAccent:     hexColor("#8b5e3c"),
			colorMiss:       hexColor("#cc4444"),
		},
	}
}

func (t *KanaTheme) kanaColor(key themeColorKey) color.Color {
	if c, ok := t.colors[key]; ok {
		return c
	}
	return color.Black
}

func (t *KanaTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return t.kanaColor(colorBackground)
	case theme.ColorNameButton:
		return t.kanaColor(colorTileFace)
	case theme.ColorNamePrimary:
		return t.kanaColor(colorAccent)
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *KanaTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *KanaTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *KanaTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func hexColor(hex string) color.RGBA {
	var r, g, b uint8
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
