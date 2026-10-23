package main

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/memory"
	"kana/store"
)

func TestWordKindPrefixAndSetting(t *testing.T) {
	if readingKind.prefix() != "word:" || meaningKind.prefix() != "meaning:" {
		t.Fatalf("prefixes %q, %q", readingKind.prefix(), meaningKind.prefix())
	}
	if readingKind.setting() != store.WordsAskReading || meaningKind.setting() != store.WordsAskMeaning {
		t.Fatalf("settings %q, %q", readingKind.setting(), meaningKind.setting())
	}
	if kindFromSetting(store.WordsAskMeaning) != meaningKind || kindFromSetting(store.WordsAskReading) != readingKind || kindFromSetting("x") != readingKind {
		t.Fatal("kindFromSetting: meaning is meaningKind, anything else readingKind")
	}
}

func TestMeaningTrackIsKeptApart(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("かさ", meaningKind)
	gs.recordWordCorrect("かさ", meaningKind)
	gs.recordWordMiss("かさ", meaningKind)
	if p := gs.meaningProgress["かさ"]; p != (store.Progress{Level: 0, Flagged: true}) || !gs.meaningProgressDirty["かさ"] {
		t.Fatalf("meaning progress %+v, dirty %v", p, gs.meaningProgressDirty["かさ"])
	}
	if s := gs.meaningSessionStats["かさ"]; s.CorrectCount != 2 || s.MissCount != 1 || s.Kana != "かさ" || !gs.meaningSessionDirty {
		t.Fatalf("meaning session %+v, dirty %v", s, gs.meaningSessionDirty)
	}
	if _, ok := gs.meaningMemory["かさ"]; !ok || !gs.meaningMemoryDirty["かさ"] {
		t.Fatal("a meaning hit must create meaning memory")
	}
	if len(gs.wordProgress) != 0 || len(gs.wordSessionStats) != 0 || len(gs.wordMemory) != 0 || gs.wordSessionDirty {
		t.Fatal("the reading track must stay untouched")
	}
	if gs.sessionWordCorrect() != 2 {
		t.Fatalf("session word hits %d, want 2 (both kinds count)", gs.sessionWordCorrect())
	}
	gs.meaningProgress["かさ"] = store.Progress{Level: learnedLevel}
	if !gs.wordLearned("かさ", meaningKind) || gs.wordLearned("かさ", readingKind) {
		t.Fatal("wordLearned goes by the kind's level")
	}
}

func TestMeaningTrackSavedAndReloaded(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.recordWordCorrect("かさ", meaningKind)
	gs.recordWordCorrect("いぬ", readingKind)
	gs.mergeSessionStats()
	gs.mu.Unlock()

	meanings, _ := st.Progress("meaning:")
	readings, _ := st.Progress("word:")
	if meanings["かさ"].Level != 1 || readings["かさ"].Level != 0 || readings["いぬ"].Level != 1 || meanings["いぬ"].Level != 0 {
		t.Fatalf("stored progress: meaning %v, reading %v", meanings, readings)
	}
	if ms, _ := st.MemoryStates("meaning:"); len(ms) != 1 {
		t.Fatalf("stored meaning memory %v, want かさ only", ms)
	}
	if s, _ := st.MeaningStatistics(); s["かさ"].CorrectCount != 1 || s["いぬ"].CorrectCount != 0 {
		t.Fatalf("stored meaning stats %v", s)
	}
	if s, _ := st.WordStatistics(); s["いぬ"].CorrectCount != 1 || s["かさ"].CorrectCount != 0 {
		t.Fatalf("stored word stats %v", s)
	}

	gs.mu.Lock()
	gs.recordWordMiss("かさ", meaningKind)
	gs.mu.Unlock()
	gs.Reset() // merges, then reloads the overall stats
	if len(gs.meaningSessionStats) != 0 || gs.meaningSessionDirty || gs.meaningOverallStats["かさ"] != (store.WordStats{Kana: "かさ", CorrectCount: 1, MissCount: 1}) {
		t.Fatalf("after Reset: session %v, overall %v", gs.meaningSessionStats, gs.meaningOverallStats)
	}

	again := NewGameState(st)
	if again.meaningProgress["かさ"] != (store.Progress{Flagged: true}) || again.wordProgress["かさ"].Level != 0 || again.meaningOverallStats["かさ"].MissCount != 1 {
		t.Fatalf("reloaded: meaning %+v, reading %+v, stats %+v", again.meaningProgress["かさ"], again.wordProgress["かさ"], again.meaningOverallStats["かさ"])
	}
	if _, ok := again.meaningMemory["かさ"]; !ok {
		t.Fatal("meaning memory not reloaded")
	}
}

func TestExistingWordProgressLoadsAsReading(t *testing.T) {
	test.NewApp()
	isolateWordsDir(t)
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveProgress("word:かさ", store.Progress{Level: 3, Flagged: true})
	_ = st.SaveMemoryState("word:かさ", memory.State{Stability: 2, LastSeen: time.Now()})
	_ = st.SaveWordStats("かさ", 7, 2)
	gs := NewGameState(st)
	if gs.wordProgress["かさ"] != (store.Progress{Level: 3, Flagged: true}) || gs.wordOverallStats["かさ"].CorrectCount != 7 {
		t.Fatalf("reading: %+v, stats %+v", gs.wordProgress["かさ"], gs.wordOverallStats["かさ"])
	}
	if _, ok := gs.wordMemory["かさ"]; !ok {
		t.Fatal("reading memory not loaded")
	}
	if len(gs.meaningProgress) != 0 || len(gs.meaningMemory) != 0 || len(gs.meaningOverallStats) != 0 {
		t.Fatalf("meaning must start empty: %v %v %v", gs.meaningProgress, gs.meaningMemory, gs.meaningOverallStats)
	}
}
