package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"kana/vocab"
)

func TestMeaningLangSettingDecidesShownAndAcceptedMeaning(t *testing.T) {
	orig := systemLanguage
	systemLanguage = func() string { return "de-DE" }
	t.Cleanup(func() { systemLanguage = orig })
	kasa := vocab.Word{Kana: "かさ", JLPT: 5, Meanings: vocab.Meanings{"de": {"Regenschirm"}, "en": {"umbrella"}, "fr": {"parapluie"}}}
	answer := func(gs *GameState, input string) bool {
		gs.tiles = []*KanaTile{wordTileAt(gs, "かさ", 0)}
		gs.checkAnswer(input)
		return len(gs.tiles) == 0
	}

	gs := newTestState()
	withWords(gs, kasa)
	gs.levelOfferDeclined = true // かさ is the whole source: no level offer may pause the game
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, meaningLang: "fr"})
	if gs.meaningLang != "fr" || gs.meaningSetting != "fr" {
		t.Fatalf("meaningLang %q, setting %q", gs.meaningLang, gs.meaningSetting)
	}
	if !answer(gs, "parapluie") || gs.feedback[len(gs.feedback)-1].text.Text != "かさ · kasa — parapluie" {
		t.Fatal("the chosen language is accepted and shown")
	}
	if !answer(gs, "umbrella") {
		t.Fatal("English is always accepted")
	}
	if answer(gs, "Regenschirm") {
		t.Fatal("German is not the meaning language now")
	}
	gs.applyWordSettings(wordSettings{source: jlptSource, level: 5, packageSize: 5, meaningLang: ""})
	if gs.meaningLang != "de" || gs.meaningSetting != "" {
		t.Fatalf("System: meaningLang %q, setting %q; want today's rule (the system's de)", gs.meaningLang, gs.meaningSetting)
	}
}

func TestMeaningLangSettingIsStored(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.applyWordSettings(wordSettings{source: gs.wordSource, level: 5, packageSize: 5, meaningLang: "nl"})
	if l, _ := st.MeaningLang(); l != "nl" {
		t.Fatalf("stored %q", l)
	}
	if gs := NewGameState(st); gs.meaningSetting != "nl" || gs.meaningLang != "nl" {
		t.Fatalf("loaded: setting %q, lang %q", gs.meaningSetting, gs.meaningLang)
	}
}

func TestMeaningLangOptions(t *testing.T) {
	labels, values := meaningLangOptions([]string{"de", "en", "fr", "ru", "xx"}, "")
	if !equalIDs(labels, []string{"System", "Deutsch", "English", "Français", "Русский", "xx"}) ||
		!equalIDs(values, []string{"", "de", "en", "fr", "ru", "xx"}) {
		t.Fatalf("labels %v, values %v", labels, values)
	}
	labels, values = meaningLangOptions([]string{"de", "en"}, "it")
	if !equalIDs(labels, []string{"System", "Deutsch", "English", "Italiano"}) || values[3] != "it" {
		t.Fatalf("a stored language the words lack stays: %v %v", labels, values)
	}
}
