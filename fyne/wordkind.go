package main

import (
	"maps"

	"kana/memory"
	"kana/store"
)

// wordKind is what a word tile asks for and what a word answer counts for:
// the word's reading or its meaning. Each kind has its own learning level,
// slipped flag, memory and hit/miss counts per reading.
type wordKind int

const (
	readingKind wordKind = iota // the zero value: tiles asked for the reading before meanings had levels
	meaningKind
)

// wordKinds lists every kind, reading first.
var wordKinds = []wordKind{readingKind, meaningKind}

// prefix is the kind's item prefix in the store's progress and memory
// tables: "word:" for the reading (the items as they were before meaning
// levels), "meaning:" for the meaning.
func (k wordKind) prefix() string {
	if k == meaningKind {
		return "meaning:"
	}
	return "word:"
}

// setting is the kind as stored in the words_ask setting.
func (k wordKind) setting() string {
	if k == meaningKind {
		return store.WordsAskMeaning
	}
	return store.WordsAskReading
}

// kindFromSetting reads the words_ask setting: store.WordsAskMeaning is the
// meaning, anything else the reading.
func kindFromSetting(s string) wordKind {
	if s == store.WordsAskMeaning {
		return meaningKind
	}
	return readingKind
}

// wordTrack is one kind's per-reading data in GameState. The maps are
// GameState's own, so writes through the view land there; the flags point
// at GameState's fields. Build it with track for each use; do not keep it.
type wordTrack struct {
	overallStats       map[string]store.WordStats
	sessionStats       map[string]store.WordStats
	sessionDirty       *bool
	memory             map[string]memory.State
	memoryDirty        map[string]bool
	memoryLoadFailed   *bool
	progress           map[string]store.Progress
	progressDirty      map[string]bool
	progressLoadFailed *bool
}

// track returns kind k's data. Must be called under lock.
func (gs *GameState) track(k wordKind) wordTrack {
	if k == meaningKind {
		return wordTrack{
			overallStats: gs.meaningOverallStats, sessionStats: gs.meaningSessionStats, sessionDirty: &gs.meaningSessionDirty,
			memory: gs.meaningMemory, memoryDirty: gs.meaningMemoryDirty, memoryLoadFailed: &gs.meaningMemoryLoadFailed,
			progress: gs.meaningProgress, progressDirty: gs.meaningProgressDirty, progressLoadFailed: &gs.meaningProgressLoadFailed,
		}
	}
	return wordTrack{
		overallStats: gs.wordOverallStats, sessionStats: gs.wordSessionStats, sessionDirty: &gs.wordSessionDirty,
		memory: gs.wordMemory, memoryDirty: gs.wordMemoryDirty, memoryLoadFailed: &gs.wordMemoryLoadFailed,
		progress: gs.wordProgress, progressDirty: gs.wordProgressDirty, progressLoadFailed: &gs.wordProgressLoadFailed,
	}
}

// loadWordStats reads kind k's stored hit and miss counts. gs.store must
// not be nil. Must be called under lock (or before the game starts).
func (gs *GameState) loadWordStats(k wordKind) (map[string]store.WordStats, error) {
	if k == meaningKind {
		return gs.store.MeaningStatistics()
	}
	return gs.store.WordStatistics()
}

// saveWordStats stores one reading's counts of kind k. gs.store must not be
// nil. Must be called under lock.
func (gs *GameState) saveWordStats(k wordKind, kana string, correct, miss int) error {
	if k == meaningKind {
		return gs.store.SaveMeaningStats(kana, correct, miss)
	}
	return gs.store.SaveWordStats(kana, correct, miss)
}

// loadTrack loads kind k's stats, memory and progress from gs.store into
// its (empty) maps. Unreadable memory or progress sets its load-failed
// flag, so the stored values are never overwritten. Called from
// NewGameState before the game starts, so no lock is needed.
func (gs *GameState) loadTrack(k wordKind) {
	t := gs.track(k)
	if ws, err := gs.loadWordStats(k); err == nil {
		maps.Copy(t.overallStats, ws)
	}
	if ms, err := gs.store.MemoryStates(k.prefix()); err == nil {
		maps.Copy(t.memory, ms)
	} else {
		*t.memoryLoadFailed = true
	}
	if p, err := gs.store.Progress(k.prefix()); err == nil {
		maps.Copy(t.progress, p)
	} else {
		*t.progressLoadFailed = true
	}
}

// answeredKind is what a hit on a word tile counts for: everything on a
// tile asking for the meaning; on a tile asking for the reading, the
// reading when input is one of its readings, else (answered by meaning) the
// meaning.
func answeredKind(tile *KanaTile, input string) wordKind {
	if tile.ask == meaningKind || !tile.word.ReadingMatches(input) {
		return meaningKind
	}
	return readingKind
}

// typedReading reports whether input is the reading of a word tile that
// asks for its meaning.
func typedReading(tile *KanaTile, input string) bool {
	return tile.word != nil && tile.ask == meaningKind && tile.word.ReadingMatches(input)
}

// readingTyped returns the lowest falling tile that asks for its meaning
// and whose reading input is, or nil. Must be called under lock.
func (gs *GameState) readingTyped(input string) *KanaTile {
	var best *KanaTile
	for _, t := range gs.tiles {
		if typedReading(t, input) && (best == nil || t.pos.Y > best.pos.Y) {
			best = t
		}
	}
	return best
}
