package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestVersionLabel(t *testing.T) {
	cases := []struct {
		meta fyne.AppMetadata
		want string
	}{
		{fyne.AppMetadata{Release: false, Version: "0.0.1"}, "dev"},
		{fyne.AppMetadata{Release: false, Version: "0.3.0"}, "dev"},
		{fyne.AppMetadata{Release: true, Version: ""}, "dev"},
		{fyne.AppMetadata{Release: true, Version: "0.0.0"}, "dev"},
		{fyne.AppMetadata{Release: true, Version: "0.1.0"}, "v0.1.0"},
		{fyne.AppMetadata{Release: true, Version: "v0.1.0"}, "v0.1.0"},
	}
	for _, c := range cases {
		if got := versionLabel(c.meta); got != c.want {
			t.Errorf("versionLabel(%+v) = %q, want %q", c.meta, got, c.want)
		}
	}
}

func TestWindowTitle(t *testing.T) {
	if got := windowTitle(fyne.AppMetadata{Release: true, Version: "0.2.0"}); got != "Kana v0.2.0" {
		t.Fatalf("windowTitle = %q", got)
	}
	if got := windowTitle(fyne.AppMetadata{}); got != "Kana dev" {
		t.Fatalf("windowTitle = %q", got)
	}
}

func TestVersionRow(t *testing.T) {
	test.NewApp()
	row := versionRow().(*fyne.Container)
	if len(row.Objects) != 2 {
		t.Fatalf("version row has %d objects", len(row.Objects))
	}
	if lbl := row.Objects[0].(*widget.Label); lbl.Text != "Version dev ·" {
		t.Fatalf("label = %q", lbl.Text)
	}
	if link := row.Objects[1].(*widget.Hyperlink); link.URL.String() != releasesURL {
		t.Fatalf("link = %v", link.URL)
	}
}
