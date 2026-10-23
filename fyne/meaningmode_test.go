package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"kana/store"
	"kana/vocab"
)

// kasa is a synthetic word with a German and an English meaning.
var kasa = vocab.Word{Kana: "かさ", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"de": {"Regenschirm"}, "en": {"umbrella"}}}

// meaningTileAt is a word tile asking for the meaning at (10, y) for a
// reading in gs.words.
func meaningTileAt(gs *GameState, kana string, y float32) *KanaTile {
	e, ok := gs.words.Lookup(kana)
	if !ok {
		panic("no word " + kana)
	}
	tile := newMeaningTile(e, gs.canvasW, fontChoice{})
	tile.fallSeconds = 20
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestReadingTileCountsTheAnswerItGot(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	gs.levelOfferDeclined = true // かさ is the whole source: no level offer may pause the game
	gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
	gs.checkAnswer("umbrella")
	if len(gs.tiles) != 0 || gs.meaningProgress["かさ"].Level != 1 || gs.meaningSessionStats["かさ"].CorrectCount != 1 {
		t.Fatalf("a meaning on a reading tile is a meaning hit: tiles %d, meaning %+v", len(gs.tiles), gs.meaningProgress["かさ"])
	}
	if _, ok := gs.wordProgress["かさ"]; ok || len(gs.wordSessionStats) != 0 {
		t.Fatal("a meaning hit must leave the reading level alone")
	}
	if got := gs.feedback[0].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("feedback %q: answered by meaning keeps its look", got)
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
	gs.checkAnswer("kasa")
	if gs.wordProgress["かさ"].Level != 1 || gs.meaningProgress["かさ"].Level != 1 {
		t.Fatalf("the reading raises the reading level only: reading %+v, meaning %+v", gs.wordProgress["かさ"], gs.meaningProgress["かさ"])
	}
}

func TestMeaningTileTakesOnlyTheMeaning(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	tile := meaningTileAt(gs, "かさ", 100)
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("kasa")
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 1 || gs.tiles[0] != tile {
		t.Fatal("the reading must not answer a meaning tile")
	}
	if len(gs.feedback) != 1 || gs.feedback[0].kind != feedbackHint || gs.feedback[0].text.Text != meaningHintText || gs.feedback[0].text.Color != feedbackHintColor {
		t.Fatalf("want one hint label, got %d labels", len(gs.feedback))
	}
	if len(gs.wordProgress) != 0 || len(gs.meaningProgress) != 0 || len(gs.wordSessionStats) != 0 || len(gs.meaningSessionStats) != 0 || gs.score != 0 || gs.answerStreak != 0 {
		t.Fatal("the hint is no hit and no mistake")
	}
	gs.checkAnswer("Umbrella")
	if len(gs.tiles) != 0 || gs.meaningProgress["かさ"].Level != 1 || len(gs.wordProgress) != 0 {
		t.Fatalf("a meaning answers it and raises the meaning level: tiles %d, meaning %+v", len(gs.tiles), gs.meaningProgress["かさ"])
	}
	if got := gs.feedback[len(gs.feedback)-1].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("hit feedback %q", got)
	}
}

func TestMeaningFirstWhenTheInputIsBoth(t *testing.T) {
	sushi := vocab.Word{Kana: "すし", JLPT: 5, Source: vocab.BundledSource, Meanings: vocab.Meanings{"en": {"sushi"}}}
	gs := newTestState()
	withWords(gs, sushi)
	gs.levelOfferDeclined = true // すし is the whole source: no level offer may pause the game
	gs.tiles = []*KanaTile{meaningTileAt(gs, "すし", 0)}
	gs.checkAnswer("sushi")
	if len(gs.tiles) != 0 || gs.meaningProgress["すし"].Level != 1 || len(gs.wordProgress) != 0 {
		t.Fatal("meaning mode: an input that is reading and meaning is a meaning hit")
	}
	gs.tiles = []*KanaTile{wordTileAt(gs, "すし", 0)}
	gs.checkAnswer("sushi")
	if gs.wordProgress["すし"].Level != 1 || gs.meaningProgress["すし"].Level != 1 {
		t.Fatal("reading mode: an input that is both stays a reading hit")
	}
}

func TestMeaningTileMissLowersOnlyTheMeaningLevel(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	setWordLevel(gs, "かさ", 3)
	gs.meaningProgress["かさ"] = store.Progress{Level: 4}
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 599)}
	gs.tick()
	if len(gs.tiles) != 0 || gs.missed != 1 {
		t.Fatalf("tiles %d, missed %d", len(gs.tiles), gs.missed)
	}
	if gs.meaningProgress["かさ"] != (store.Progress{Level: 2, Flagged: true}) || gs.meaningSessionStats["かさ"].MissCount != 1 {
		t.Fatalf("meaning %+v, stats %+v", gs.meaningProgress["かさ"], gs.meaningSessionStats["かさ"])
	}
	if gs.wordProgress["かさ"] != (store.Progress{Level: 3}) || len(gs.wordSessionStats) != 0 {
		t.Fatalf("reading must stay: %+v", gs.wordProgress["かさ"])
	}
	if got := gs.feedback[0].text.Text; got != "かさ · kasa — umbrella" {
		t.Fatalf("miss feedback %q", got)
	}
}

func TestWrongInputOnAMeaningTileLowersTheMeaningLevel(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 100)}
	gs.checkAnswer("parasol")
	if len(gs.tiles) != 1 || gs.meaningProgress["かさ"] != (store.Progress{Flagged: true}) || len(gs.wordProgress) != 0 {
		t.Fatalf("meaning %+v, reading %v", gs.meaningProgress["かさ"], gs.wordProgress)
	}
}

func TestLateReadingForAFallenMeaningTileIsNoMistake(t *testing.T) {
	gs := newTestState()
	withWords(gs, kasa)
	kana := tileAt("あ", "a", 100)
	gs.tiles = []*KanaTile{meaningTileAt(gs, "かさ", 599), kana}
	gs.tick() // かさ falls through
	gs.checkAnswer("kasa")
	if len(gs.tiles) != 1 || gs.kanaProgress["あ"].Flagged {
		t.Fatal("the reading of a meaning tile that just fell is a late Enter, not a mistake on あ")
	}
	if gs.feedback[len(gs.feedback)-1].kind == feedbackHint {
		t.Fatal("no hint for a tile that is gone")
	}
}
