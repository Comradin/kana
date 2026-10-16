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

// versionRow is the Sources tab's "Version v0.2.0 · Releases" line.
func versionRow() fyne.CanvasObject {
	u, _ := url.Parse(releasesURL)
	return container.NewHBox(
		widget.NewLabel("Version "+versionLabel(fyne.CurrentApp().Metadata())+" ·"),
		widget.NewHyperlink("Releases", u),
	)
}

// sourceLink is a link on the Sources tab.
type sourceLink struct{ label, url string }

// dataSource is one source the app uses, with its licence and links.
type dataSource struct {
	name   string
	detail string
	links  []sourceLink
}

// dataSources are credited on the Sources tab.
var dataSources = []dataSource{
	{"JMdict", "© Electronic Dictionary Research and Development Group (EDRDG), CC BY-SA 4.0. Word meanings.", []sourceLink{
		{"Licence", "https://www.edrdg.org/edrdg/licence.html"},
		{"JMdict", "https://www.edrdg.org/jmdict/j_jmdict.html"},
	}},
	{"JLPT levels", "Jonathan Waller, CC BY. Unofficial: there are no official JLPT word lists since 2010.", []sourceLink{
		{"tanos.co.uk", "http://www.tanos.co.uk/jlpt/"},
	}},
	{"JMdict mapping", "stephenmk/yomitan-jlpt-vocab, CC BY-SA 4.0. Matches the JLPT lists to JMdict.", []sourceLink{
		{"GitHub", "https://github.com/stephenmk/yomitan-jlpt-vocab"},
	}},
}

// newSourcesTab shows the version and credits every data source.
func newSourcesTab() fyne.CanvasObject {
	box := container.NewVBox(versionRow(), widget.NewSeparator())
	for _, s := range dataSources {
		detail := widget.NewLabel(s.detail)
		detail.Wrapping = fyne.TextWrapWord
		links := container.NewHBox()
		for _, l := range s.links {
			u, _ := url.Parse(l.url)
			links.Add(widget.NewHyperlink(l.label, u))
		}
		box.Add(widget.NewLabelWithStyle(s.name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		box.Add(detail)
		box.Add(links)
	}
	box.Add(widget.NewLabel("The word data is licensed under CC BY-SA 4.0; the app's code under MIT."))
	scroll := container.NewVScroll(box)
	scroll.SetMinSize(fyne.NewSize(460, 300))
	return scroll
}
