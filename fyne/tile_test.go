package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/vocab"
)

func TestKanaTileMoveUpdatesAllObjects(t *testing.T) {
	test.NewApp()
	k := kanacore.Kana{Char: "か", Romaji: "ka"}
	tile := newKanaTile(k, fontChoice{})
	pos := fyne.NewPos(100, 50)
	tile.Move(pos)

	if tile.face.Position() != pos {
		t.Errorf("face position: got %v, want %v", tile.face.Position(), pos)
	}
	wantShadow := fyne.NewPos(103, 53)
	if tile.shadow.Position() != wantShadow {
		t.Errorf("shadow position: got %v, want %v", tile.shadow.Position(), wantShadow)
	}
}

func TestKanaTileObjectsReturnsThree(t *testing.T) {
	test.NewApp()
	k := kanacore.Kana{Char: "あ", Romaji: "a"}
	tile := newKanaTile(k, fontChoice{})
	if len(tile.Objects()) != 3 {
		t.Errorf("expected 3 canvas objects, got %d", len(tile.Objects()))
	}
}

func TestWideTileForYoon(t *testing.T) {
	test.NewApp()
	narrow := newKanaTile(kanacore.Kana{Char: "き", Romaji: "ki"}, fontChoice{})
	wide := newKanaTile(kanacore.Kana{Char: "きゃ", Romaji: "kya"}, fontChoice{})
	if wide.Width() <= narrow.Width() {
		t.Fatalf("wide tile %.0f should be wider than narrow tile %.0f", wide.Width(), narrow.Width())
	}
}

func TestTileTextCentredInsideFace(t *testing.T) {
	test.NewApp()
	for _, char := range []string{"か", "きゃ"} {
		tile := newKanaTile(kanacore.Kana{Char: char}, fontChoice{})
		tile.Move(fyne.NewPos(40, 20))
		textX := tile.text.Position().X
		textW := tile.text.Size().Width
		left := textX - 40
		right := 40 + tile.Width() - (textX + textW)
		if left < 0 || right < 0 {
			t.Fatalf("%s: text outside face (left %.1f, right %.1f)", char, left, right)
		}
		if diff := left - right; diff > 1 || diff < -1 {
			t.Fatalf("%s: text not centred (left %.1f, right %.1f)", char, left, right)
		}
	}
}

func TestWordTileLooksDifferent(t *testing.T) {
	test.NewApp()
	kana := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"}, fontChoice{})
	word := newWordTile(vocab.Entry{Kana: "か", Units: []string{"か"}}, 400, fontChoice{})
	if word.word == nil || word.word.Kana != "か" {
		t.Fatalf("word = %+v", word.word)
	}
	if kana.word != nil {
		t.Fatal("a kana tile carries no word")
	}
	if word.face.FillColor == kana.face.FillColor || word.face.StrokeWidth == 0 {
		t.Fatal("a one-unit word must not look like a kana tile")
	}
	if len(word.Objects()) != 3 {
		t.Fatalf("objects = %d, want 3", len(word.Objects()))
	}
}

func TestWordTileWidthFollowsText(t *testing.T) {
	test.NewApp()
	short := newWordTile(vocab.Entry{Kana: "ねこ"}, 600, fontChoice{})
	long := newWordTile(vocab.Entry{Kana: "かんきょう"}, 600, fontChoice{})
	if long.Width() <= short.Width() {
		t.Fatalf("long %.0f should be wider than short %.0f", long.Width(), short.Width())
	}
	if long.text.TextSize != wordTextSize {
		t.Fatalf("a word that fits keeps the full size, got %v", long.text.TextSize)
	}
	long.Move(fyne.NewPos(40, 20))
	left := long.text.Position().X - 40
	right := 40 + long.Width() - (long.text.Position().X + long.text.Size().Width)
	if diff := left - right; left < 0 || right < 0 || diff > 1 || diff < -1 {
		t.Fatalf("text not centred (left %.1f, right %.1f)", left, right)
	}
}

func TestWordTileShrinksToFitField(t *testing.T) {
	test.NewApp()
	tile := newWordTile(vocab.Entry{Kana: "コンピューター"}, 150, fontChoice{})
	if tile.Width() > 150 {
		t.Fatalf("width %.1f exceeds the 150 px field", tile.Width())
	}
	if tile.text.TextSize >= wordTextSize {
		t.Fatalf("font not shrunk: %v", tile.text.TextSize)
	}
}

func TestWordTileShowsTheShownForm(t *testing.T) {
	test.NewApp()
	tile := newWordTile(vocab.Entry{Kana: "さん", Display: "～さん"}, 400, fontChoice{})
	if tile.text.Text != "～さん" || tile.word.Kana != "さん" {
		t.Fatalf("text %q, word %q; want ～さん on the tile, さん as its reading", tile.text.Text, tile.word.Kana)
	}
}

func TestMeaningTileHasAMarkerBesideTheWord(t *testing.T) {
	test.NewApp()
	e := vocab.Entry{Kana: "き", Units: []string{"き"}}
	reading := newWordTile(e, 400, fontChoice{})
	tile := newMeaningTile(e, 400, fontChoice{})
	if reading.ask != readingKind || reading.marker != nil || len(reading.Objects()) != 3 {
		t.Fatalf("a reading tile asks for the reading and has no marker: ask %v, marker %v", reading.ask, reading.marker)
	}
	if tile.ask != meaningKind || tile.marker == nil || tile.word.Kana != "き" {
		t.Fatalf("meaning tile: ask %v, marker %v", tile.ask, tile.marker)
	}
	if objs := tile.Objects(); len(objs) != 4 || objs[3] != tile.marker {
		t.Fatalf("objects = %d, want shadow, face, text and the marker on top", len(objs))
	}
	if tile.marker.Text != meaningMarkerText || tile.marker.Color != wordTileBorderColor {
		t.Fatalf("marker %q in %v", tile.marker.Text, tile.marker.Color)
	}
	if tile.Width() <= reading.Width() {
		t.Fatalf("meaning tile %.1f should be wider than the reading tile %.1f", tile.Width(), reading.Width())
	}
	tile.Move(fyne.NewPos(40, 20))
	mp, ms := tile.marker.Position(), tile.marker.Size()
	textLeft := tile.text.Position().X
	textRight := textLeft + tile.text.Size().Width
	if mp.X < textRight {
		t.Fatalf("marker at %.1f covers the word ending at %.1f", mp.X, textRight)
	}
	if mp.X+ms.Width > 40+tile.Width() || mp.Y < 20 || mp.Y+ms.Height > 20+tileH {
		t.Fatalf("marker %v %v outside the face at (40, 20) %.1f×%.0f", mp, ms, tile.Width(), tileH)
	}
	if left, right := textLeft-40, 40+tile.Width()-textRight; left-right > 1 || right-left > 1 {
		t.Fatalf("word not centred (left %.1f, right %.1f)", left, right)
	}
}

func TestMeaningTileShrinksToFitField(t *testing.T) {
	test.NewApp()
	tile := newMeaningTile(vocab.Entry{Kana: "コンピューター"}, 150, fontChoice{})
	if tile.Width() > 150 {
		t.Fatalf("width %.1f exceeds the 150 px field", tile.Width())
	}
}
