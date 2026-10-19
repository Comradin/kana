package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"kana/kanacore"
	"kana/vocab"
)

// Word package intro layout: the kana column, the row height and the
// scroll area, sized so ten words fit the 1000 × 760 window.
const (
	packageKanaSize    float32 = 32
	packageKanaMinSize float32 = 16
	packageKanaW       float32 = 240
	packageRowH        float32 = 56
	packageViewW       float32 = 680
	packageViewMaxH    float32 = 480
)

// packageIntro is what the word package dialog shows.
type packageIntro struct {
	Title   string
	Entries []vocab.Entry
	Lang    string // meaning language
}

// PackageIntro returns the pending package's entries, in order, with the
// dialog title and the meaning language.
func (gs *GameState) PackageIntro() packageIntro {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	in := packageIntro{Title: "New words", Lang: gs.meaningLang}
	if gs.words == nil {
		return in
	}
	for _, kana := range gs.pendingPackage {
		if e, ok := gs.words.Lookup(kana); ok {
			in.Entries = append(in.Entries, e)
		}
	}
	src, _ := gs.source()
	in.Title = packageTitle(gs.words, src, in.Entries)
	return in
}

// packageTitle names a package: "New words · JLPT N5" (or "N5–N4" when it
// spans levels), "New words · minna no nihongo · lesson 2" (or "lessons
// 2–3"); a list without lessons gives just its name.
func packageTitle(ix *vocab.Index, src vocab.Source, entries []vocab.Entry) string {
	var parts []string
	if src.List == "" {
		for _, e := range entries {
			if l := fmt.Sprintf("N%d", e.JLPT); !slices.Contains(parts, l) {
				parts = append(parts, l)
			}
		}
		switch len(parts) {
		case 0:
			return "New words"
		case 1:
			return "New words · JLPT " + parts[0]
		}
		return "New words · JLPT " + parts[0] + "–" + parts[len(parts)-1]
	}
	title := "New words · " + prettyListName(src.List)
	for _, e := range entries {
		if l := ix.Lesson(e, src.List); l != "" && !slices.Contains(parts, l) {
			parts = append(parts, l)
		}
	}
	switch len(parts) {
	case 0:
		return title
	case 1:
		return title + " · lesson " + parts[0]
	}
	return title + " · lessons " + parts[0] + "–" + parts[len(parts)-1]
}

// packageView is the word package intro's content: one row per word with
// its kana, romaji, meaning and an "Already known" check. Ticking a word
// strikes it through and asks once for a replacement, which gets a row of
// its own; unticking keeps the replacement.
type packageView struct {
	list    *fyne.Container
	rows    []*packageRow
	lang    string
	font    fontChoice
	replace func() (vocab.Entry, bool)
}

type packageRow struct {
	entry    vocab.Entry
	kana     *canvas.Text
	strike   *canvas.Rectangle
	known    *widget.Check
	replaced bool // a replacement was already asked for
}

func newPackageView(entries []vocab.Entry, lang string, kf fontChoice, replace func() (vocab.Entry, bool)) *packageView {
	v := &packageView{list: container.NewVBox(), lang: lang, font: kf, replace: replace}
	for _, e := range entries {
		v.addRow(e)
	}
	return v
}

func (v *packageView) addRow(e vocab.Entry) {
	row := &packageRow{entry: e}
	row.kana = canvas.NewText(e.Kana, kanaTextColor)
	row.kana.TextSize = packageKanaSize
	fitKanaText(row.kana, v.font, packageKanaW, packageKanaMinSize)
	row.strike = canvas.NewRectangle(kanaTextColor)
	row.strike.Hide()
	word := container.NewGridWrap(fyne.NewSize(packageKanaW, packageRowH), container.New(strikeLayout{}, row.kana, row.strike))
	romaji := widget.NewLabelWithStyle(kanacore.Romaji(e.Kana), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	meaning := widget.NewLabel(meaningSnippet(e.Meanings, v.lang))
	meaning.Truncation = fyne.TextTruncateEllipsis
	row.known = widget.NewCheck("Already known", func(on bool) { v.toggle(row, on) })
	v.rows = append(v.rows, row)
	v.list.Add(container.NewBorder(nil, nil, word, row.known, container.NewVBox(romaji, meaning)))
}

func (v *packageView) toggle(row *packageRow, on bool) {
	if on {
		row.strike.Show()
	} else {
		row.strike.Hide()
	}
	if !on || row.replaced {
		return
	}
	row.replaced = true
	if e, ok := v.replace(); ok {
		v.addRow(e)
		v.list.Refresh()
	}
}

// known returns the readings ticked as already known.
func (v *packageView) known() []string {
	var out []string
	for _, r := range v.rows {
		if r.known.Checked {
			out = append(out, r.entry.Kana)
		}
	}
	return out
}

// strikeLayout shows a text at the left, centred vertically, with a thin
// bar (the second object) across its middle.
type strikeLayout struct{}

func (strikeLayout) MinSize(objs []fyne.CanvasObject) fyne.Size { return objs[0].MinSize() }

func (strikeLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	text := objs[0].MinSize()
	y := (size.Height - text.Height) / 2
	objs[0].Resize(text)
	objs[0].Move(fyne.NewPos(0, y))
	objs[1].Resize(fyne.NewSize(text.Width, 2))
	objs[1].Move(fyne.NewPos(0, y+text.Height/2-1))
}

// showWordPackageDialog introduces the pending word package. "Let's go"
// finishes it. Call it through showPendingDialogs. Must run on the Fyne
// thread.
func showWordPackageDialog(gs *GameState, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	intro := gs.PackageIntro()
	view := newPackageView(intro.Entries, intro.Lang, gs.KanaFont(), gs.ReplacementWord)
	scroll := container.NewVScroll(view.list)
	scroll.SetMinSize(fyne.NewSize(packageViewW, min(float32(len(intro.Entries))*packageRowH, packageViewMaxH)))

	inputBar.SetEnabled(false, w)
	d := dialog.NewCustom(intro.Title, "Let's go", scroll, w)
	d.SetOnClosed(func() {
		dialogShowing = false
		enterConfirms = nil
		gs.FinishPackage(view.known())
		continueAfterDialog(gs, statsPanel, inputBar, w)
	})
	dialogShowing = true
	d.Show()
	confirmOnEnter(d)
}

// levelOfferText asks whether to continue with the next JLPT level.
func levelOfferText(level int) string {
	return fmt.Sprintf("N%d finished — continue with N%d?", level, level-1)
}

// showLevelOffer offers the next JLPT level once the current one is fully
// introduced. Call it through showPendingDialogs. Must run on the Fyne
// thread.
func showLevelOffer(gs *GameState, level int, statsPanel *StatsPanel, inputBar *InputBar, w fyne.Window) {
	body := widget.NewLabel(levelOfferText(level))
	inputBar.SetEnabled(false, w)
	d := dialog.NewCustomConfirm("More words", fmt.Sprintf("Continue with N%d", level-1), "Later", body, func(accept bool) {
		// The response callback runs before SetOnClosed; clear the guard
		// first so the chained package dialog is not swallowed.
		dialogShowing = false
		if accept {
			gs.AcceptLevelOffer()
		} else {
			gs.DeclineLevelOffer()
		}
		continueAfterDialog(gs, statsPanel, inputBar, w)
	}, w)
	dialogShowing = true
	d.Show()
}
