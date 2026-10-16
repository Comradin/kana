package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/vocab"
)

// thai is a character neither bundled font has (checked against the pinned
// files), so a text with it falls back to the theme font.
const thai = "ก"

func TestFontResourcesLoad(t *testing.T) {
	names := map[string]bool{}
	for _, res := range []fyne.Resource{notoRegular, notoBold, mplusRegular, mplusBold} {
		if len(res.Content()) < 1_000_000 {
			t.Fatalf("%s: %d bytes, want the full font file", res.Name(), len(res.Content()))
		}
		if fontFaces()[res] == nil {
			t.Fatalf("%s does not parse", res.Name())
		}
		if !covers(res, "あいうえおアイウエオきゃキャっッーヴ…") {
			t.Fatalf("%s lacks kana", res.Name())
		}
		names[res.Name()] = true
	}
	if len(names) != 4 {
		t.Fatalf("resource names are not distinct: %v", names)
	}
}

func TestFontChoiceCoverage(t *testing.T) {
	noto, mplus, system := fontByID("noto"), fontByID("mplus"), fontByID("system")
	if noto.source("あ", false) != notoRegular || noto.source("あ", true) != notoBold {
		t.Fatal("Noto Sans JP: regular and bold files by style")
	}
	if mplus.source("ア — blue", false) != mplusRegular || mplus.source("ア", true) != mplusBold {
		t.Fatal("M PLUS Rounded 1c: regular and bold files by style")
	}
	for _, c := range []fontChoice{noto, mplus} {
		if src := c.source("あ"+thai, false); src != nil {
			t.Fatalf("%s: a text with a missing rune must use the theme font, got %v", c.id, src.Name())
		}
	}
	if system.source("あ", false) != nil || system.source("あ", true) != nil {
		t.Fatal("System uses the theme font")
	}
	if fontByID("bogus").id != "noto" || fontByID("").id != "noto" {
		t.Fatal("an unknown setting gives Noto Sans JP")
	}
}

func TestTilesUseTheKanaFontAndAreSizedWithIt(t *testing.T) {
	test.NewApp()
	noto := fontByID("noto")
	k := newKanaTile(kanacore.Kana{Char: "き", Romaji: "ki"}, noto)
	if k.text.FontSource != notoRegular || k.text.Size() != k.text.MinSize() {
		t.Fatalf("kana tile: source %v, size %v, min size with the font %v", k.text.FontSource, k.text.Size(), k.text.MinSize())
	}
	w := newWordTile(vocab.Entry{Kana: "コンピューター"}, 150, noto)
	if w.text.FontSource != notoRegular {
		t.Fatal("word tile not in the kana font")
	}
	if got, want := w.Width(), w.text.MinSize().Width+2*wordTilePad; got != want || w.Width() > 150 {
		t.Fatalf("word tile width %v, want %v (text measured in its font) and at most 150", got, want)
	}
	if s := newKanaTile(kanacore.Kana{Char: "き"}, fontByID("system")); s.text.FontSource != nil {
		t.Fatal("System: no font source")
	}
}

func TestGameUsesTheChosenFont(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.SetKanaFont("mplus")
	gs.spawnKana()
	if len(gs.tiles) != 1 || gs.tiles[0].text.FontSource != mplusRegular {
		t.Fatal("a new tile must use the chosen font")
	}
	gs.SetKanaFont("noto")
	if gs.tiles[0].text.FontSource != mplusRegular {
		t.Fatal("a falling tile keeps its font")
	}
	gs.addFeedback(feedbackHit, "あい — love", fyne.NewPos(10, 10))
	if gs.feedback[0].text.FontSource != notoBold {
		t.Fatalf("feedback label: %v, want the bold kana font", gs.feedback[0].text.FontSource)
	}
	gs.addFeedback(feedbackHit, "あい — "+thai, fyne.NewPos(10, 10))
	if gs.feedback[1].text.FontSource != nil {
		t.Fatal("a label with a missing rune keeps the theme font")
	}
}

func TestFeedbackFontCheckedOnTheFinalText(t *testing.T) {
	gs := newTestState()
	gs.canvasW = 300
	gs.kanaFont = fontByID("noto")
	gs.addFeedback(feedbackMiss, "あいうえお · aiueo — "+strings.Repeat("a long meaning; ", 6)+thai, fyne.NewPos(0, 0))
	f := gs.feedback[0].text
	if !strings.HasSuffix(f.Text, "…") || strings.Contains(f.Text, thai) {
		t.Fatalf("precondition: the label is cut before the Thai rune: %q", f.Text)
	}
	if f.FontSource != notoBold {
		t.Fatal("the cut text has no missing rune, so it uses the kana font")
	}
	if w := f.MinSize().Width; w > 300 {
		t.Fatalf("label %v px wide, more than the field", w)
	}
}

func TestIntroAndPackageUseTheKanaFont(t *testing.T) {
	test.NewApp()
	mplus := fontByID("mplus")
	card := newIntroCard(kanacore.Entry{Char: "か", Romaji: "ka"}, mplus)
	found := false
	for _, o := range test.LaidOutObjects(card) {
		if txt, ok := o.(*canvas.Text); ok && txt.Text == "か" {
			found = txt.FontSource == mplusRegular
		}
	}
	if !found {
		t.Fatal("the intro card's kana must use the kana font")
	}
	v := newPackageView([]vocab.Entry{{Kana: "かさ", Meanings: vocab.Meanings{}}}, "en", mplus, func() (vocab.Entry, bool) { return vocab.Entry{}, false })
	if v.rows[0].kana.FontSource != mplusRegular {
		t.Fatal("the package intro's kana must use the kana font")
	}
}

func TestStatsPanelKanaFont(t *testing.T) {
	test.NewApp()
	p := newStatsPanel()
	p.SetKanaFont(fontByID("noto"))
	if p.cells["あ"].text.FontSource != notoRegular || p.cells["キャ"].text.FontSource != notoRegular {
		t.Fatal("grid cells must use the kana font")
	}
	p.SetKanaFont(fontByID("system"))
	if p.cells["あ"].text.FontSource != nil {
		t.Fatal("System: no font source in the grid")
	}
}

func TestKanaFontSetting(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if id := NewGameState(st).KanaFont().id; id != "noto" {
		t.Fatalf("default font = %q, want noto", id)
	}
	gs := NewGameState(st)
	f := newSettingsForm(gs.settingsState())
	if f.kanaFont.Selected != "Noto Sans JP" || len(f.kanaFont.Options) != 3 {
		t.Fatalf("font options %v, selected %q", f.kanaFont.Options, f.kanaFont.Selected)
	}
	f.kanaFont.SetSelected("M PLUS Rounded 1c")
	f.apply(gs)
	if gs.KanaFont().id != "mplus" {
		t.Fatalf("font after Save = %q", gs.KanaFont().id)
	}
	if id, _ := st.KanaFont(); id != "mplus" {
		t.Fatalf("stored font = %q", id)
	}
	if id := NewGameState(st).KanaFont().id; id != "mplus" {
		t.Fatalf("reloaded font = %q", id)
	}
}

func TestGameTabPreviewsEachFont(t *testing.T) {
	test.NewApp()
	f := newSettingsForm(newTestState().settingsState())
	f.tabs.Select(f.gameTab)
	sources := map[fyne.Resource]bool{}
	previews := 0
	for _, o := range test.LaidOutObjects(f.gameTab.Content) {
		if txt, ok := o.(*canvas.Text); ok && txt.Text == fontPreview {
			previews++
			sources[txt.FontSource] = true
		}
	}
	if previews != 3 || !sources[notoRegular] || !sources[mplusRegular] || !sources[nil] {
		t.Fatalf("%d previews with fonts %v; want one per choice", previews, sources)
	}
}
