package main

import (
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// releasesURL is where packaged builds are published.
const releasesURL = "https://github.com/Comradin/kana/releases"

// versionLabel returns the version to show: "v0.2.0" for a release
// package, "dev" otherwise (go run, go build, or a package without a
// real version).
func versionLabel(meta fyne.AppMetadata) string {
	v := strings.TrimPrefix(meta.Version, "v")
	if !meta.Release || v == "" || v == "0.0.0" {
		return "dev"
	}
	return "v" + v
}

// windowTitle is the main window's title, e.g. "Kana v0.2.0".
func windowTitle(meta fyne.AppMetadata) string {
	return "Kana " + versionLabel(meta)
}

// versionRow is the settings dialog's "Version v0.2.0 · Releases" line.
func versionRow() fyne.CanvasObject {
	u, _ := url.Parse(releasesURL)
	return container.NewHBox(
		widget.NewLabel("Version "+versionLabel(fyne.CurrentApp().Metadata())+" ·"),
		widget.NewHyperlink("Releases", u),
	)
}
