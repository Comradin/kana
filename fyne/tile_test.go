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
