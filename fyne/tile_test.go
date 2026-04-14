package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
)

func TestKanaTileMoveUpdatesAllObjects(t *testing.T) {
	test.NewApp()
	k := kanacore.Kana{Char: "か", Romaji: "ka"}
	tile := newKanaTile(k)
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
	tile := newKanaTile(k)
	if len(tile.Objects()) != 3 {
		t.Errorf("expected 3 canvas objects, got %d", len(tile.Objects()))
	}
}
