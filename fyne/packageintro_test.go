package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"kana/vocab"
)

func TestPendingDialogsReportsEveryKind(t *testing.T) {
	gs := newTestState()
	if p := gs.PendingDialogs(); len(p.Intro) != 0 || p.KatakanaOffer || len(p.Package) != 0 || p.LevelOffer != 0 {
		t.Fatalf("nothing pending: %+v", p)
	}
	gs.pendingIntro = []string{"k"}
	gs.pendingOffer = true
	gs.pendingPackage = []string{"あい"}
	gs.pendingLevelOffer = true
	gs.wordLevel = 4
	p := gs.PendingDialogs()
	if !equalIDs(p.Intro, []string{"k"}) || !p.KatakanaOffer || !equalIDs(p.Package, []string{"あい"}) || p.LevelOffer != 4 {
		t.Fatalf("pending = %+v", p)
	}
	p.Package[0] = "x"
	if gs.pendingPackage[0] != "あい" {
		t.Fatal("PendingDialogs must return copies")
	}
}

func TestPackageTitle(t *testing.T) {
	list := vocab.List{Name: "minna-no-nihongo", Words: []vocab.Word{
		{Kana: "あい", Lesson: "2", Meanings: vocab.Meanings{}},
		{Kana: "あお", Lesson: "3", Meanings: vocab.Meanings{}},
		{Kana: "いえ", Meanings: vocab.Meanings{}},
	}}
	ix := vocab.NewIndex([]vocab.Word{bw("あい", 5, "love"), bw("あお", 4, "blue")}, []vocab.List{list})
	e := func(kana string) vocab.Entry { x, _ := ix.Lookup(kana); return x }
	minna := vocab.Source{List: "minna-no-nihongo"}
	cases := []struct {
		name    string
		src     vocab.Source
		entries []vocab.Entry
		want    string
	}{
		{"JLPT", vocab.Source{Level: 5}, []vocab.Entry{e("あい")}, "New words · JLPT N5"},
		{"JLPT across levels", vocab.Source{Level: 4}, []vocab.Entry{e("あい"), e("あお")}, "New words · JLPT N5–N4"},
		{"one lesson", minna, []vocab.Entry{e("あい")}, "New words · minna no nihongo · lesson 2"},
		{"two lessons", minna, []vocab.Entry{e("あい"), e("あお"), e("いえ")}, "New words · minna no nihongo · lessons 2–3"},
		{"no lessons", minna, []vocab.Entry{e("いえ")}, "New words · minna no nihongo"},
		{"empty", vocab.Source{Level: 5}, nil, "New words"},
	}
	for _, c := range cases {
		if got := packageTitle(ix, c.src, c.entries); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPackageViewAlreadyKnown(t *testing.T) {
	test.NewApp()
	entries := []vocab.Entry{
		{Kana: "あい", Meanings: vocab.Meanings{"en": {"love"}}},
		{Kana: "あお", Meanings: vocab.Meanings{"en": {"blue"}}},
	}
	extra := []vocab.Entry{{Kana: "いえ", Meanings: vocab.Meanings{"en": {"house"}}}}
	calls := 0
	v := newPackageView(entries, "en", fontChoice{}, func() (vocab.Entry, bool) {
		calls++
		if len(extra) == 0 {
			return vocab.Entry{}, false
		}
		e := extra[0]
		extra = extra[1:]
		return e, true
	})
	if len(v.rows) != 2 || len(v.list.Objects) != 2 {
		t.Fatalf("rows = %d", len(v.rows))
	}
	v.rows[0].known.SetChecked(true)
	if calls != 1 || len(v.rows) != 3 || v.rows[2].entry.Kana != "いえ" || len(v.list.Objects) != 3 {
		t.Fatalf("ticking must add the replacement いえ: calls %d, rows %d", calls, len(v.rows))
	}
	if !v.rows[0].strike.Visible() {
		t.Fatal("a ticked word is struck through")
	}
	v.rows[0].known.SetChecked(false)
	if len(v.rows) != 3 || v.rows[0].strike.Visible() {
		t.Fatal("unticking keeps the replacement and removes the strike")
	}
	v.rows[0].known.SetChecked(true)
	if calls != 1 {
		t.Fatal("ticking the same word again must not ask for another replacement")
	}
	v.rows[1].known.SetChecked(true)
	if calls != 2 || len(v.rows) != 3 || !v.rows[1].known.Checked {
		t.Fatalf("no replacement left: the box stays ticked and nothing is added (calls %d, rows %d)", calls, len(v.rows))
	}
	if got := v.known(); !equalIDs(got, []string{"あい", "あお"}) {
		t.Fatalf("known = %v", got)
	}
}

// buttonIn finds the button with the given text in a dialog's objects.
func buttonIn(o fyne.CanvasObject, text string) *widget.Button {
	for _, obj := range test.LaidOutObjects(o) {
		if b, ok := obj.(*widget.Button); ok && b.Text == text {
			return b
		}
	}
	return nil
}

func TestPackageDialogFinishesThePackage(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	gs := packageReady(20)
	statsPanel := newStatsPanel()
	ib := newInputBar(gs, statsPanel, newGameCanvas(gs), w)
	gs.maybeOfferPackage()
	drainEvents(gs)
	pkg := append([]string(nil), gs.pendingPackage...)

	showPendingDialogs(gs, statsPanel, ib, w)
	top := w.Canvas().Overlays().Top()
	if top == nil || !dialogShowing {
		t.Fatal("the word package dialog is not showing")
	}
	if !ib.entry.Disabled() {
		t.Fatal("input must be disabled while the dialog shows")
	}
	if top.Size().Height > 760 {
		t.Fatalf("dialog is %v px high, more than the window", top.Size().Height)
	}
	b := buttonIn(top, "Let's go")
	if b == nil {
		t.Fatal("no Let's go button")
	}
	test.Tap(b)
	if dialogShowing || len(gs.pendingPackage) != 0 || gs.paused {
		t.Fatalf("after Let's go: showing %v, pending %v, paused %v", dialogShowing, gs.pendingPackage, gs.paused)
	}
	for _, kana := range pkg {
		if !gs.introduced[kana] {
			t.Fatalf("%s not introduced", kana)
		}
	}
	if ib.entry.Disabled() {
		t.Fatal("input must be enabled again")
	}
}

func TestLevelOfferDialog(t *testing.T) {
	for _, accept := range []bool{true, false} {
		test.NewApp()
		w := test.NewWindow(nil)
		gs := levelDone()
		statsPanel := newStatsPanel()
		ib := newInputBar(gs, statsPanel, newGameCanvas(gs), w)
		gs.maybeOfferPackage()
		drainEvents(gs)
		showPendingDialogs(gs, statsPanel, ib, w)
		top := w.Canvas().Overlays().Top()
		label := "Later"
		if accept {
			label = "Continue with N4"
		}
		b := buttonIn(top, label)
		if b == nil {
			t.Fatalf("no %q button", label)
		}
		test.Tap(b)
		if accept && (gs.wordLevel != 4 || len(gs.pendingPackage) == 0 || !dialogShowing) {
			t.Fatalf("accept: level %d, pending %v; want N4 and its package dialog", gs.wordLevel, gs.pendingPackage)
		}
		if !accept && (gs.wordLevel != 5 || !gs.levelOfferDeclined || gs.paused || dialogShowing) {
			t.Fatalf("later: level %d, declined %v, paused %v", gs.wordLevel, gs.levelOfferDeclined, gs.paused)
		}
		dialogShowing = false
		w.Close()
	}
	test.NewApp()
	if got := levelOfferText(5); got != "N5 finished — continue with N4?" {
		t.Fatalf("text = %q", got)
	}
}

func TestEnterClosesTheIntroDialogs(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 760))
	gs := packageReady(20)
	gs.pendingIntro = []string{"k"}
	gs.paused = true
	statsPanel := newStatsPanel()
	ib := newInputBar(gs, statsPanel, newGameCanvas(gs), w)
	enter := &fyne.KeyEvent{Name: fyne.KeyReturn}

	showPendingDialogs(gs, statsPanel, ib, w)
	if len(gs.pendingIntro) == 0 || enterConfirms == nil {
		t.Fatal("the kana intro is not showing")
	}
	windowKey(enter, w, ib) // right after it opened: an answer, not a confirmation
	if len(gs.pendingIntro) == 0 {
		t.Fatal("an Enter right after the dialog opened must not close it")
	}
	time.Sleep(enterGrace)
	windowKey(enter, w, ib)
	if len(gs.pendingIntro) != 0 {
		t.Fatal("Enter must close the kana intro")
	}

	gs.maybeOfferPackage()
	drainEvents(gs)
	showPendingDialogs(gs, statsPanel, ib, w)
	if len(gs.pendingPackage) == 0 || enterConfirms == nil {
		t.Fatal("the word package dialog is not showing")
	}
	time.Sleep(enterGrace)
	windowKey(enter, w, ib)
	if len(gs.pendingPackage) != 0 || dialogShowing || enterConfirms != nil {
		t.Fatalf("Enter must finish the package: pending %v, showing %v", gs.pendingPackage, dialogShowing)
	}
}
