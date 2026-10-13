package main

import (
	"kana/kanacore"
	"kana/memory"
	"kana/store"
	"kana/vocab"
)

// recordWordCorrect records a hit on a word: word session stats, word memory
// and the answer streak. Kana stats are not touched. Must be called under
// lock.
func (gs *GameState) recordWordCorrect(kana string) {
	stat := gs.wordSessionStats[kana]
	stat.Kana = kana
	stat.CorrectCount++
	gs.wordSessionStats[kana] = stat
	gs.wordSessionDirty = true

	prev, known := gs.wordMemory[kana]
	gs.wordMemory[kana] = memory.Correct(prev, known, gs.now())
	gs.wordMemoryDirty[kana] = true

	gs.raiseTempo()
}

// recordWordMiss records a word that fell through: word session stats, word
// memory (only if the word has a state) and the tempo. Must be called under
// lock.
func (gs *GameState) recordWordMiss(kana string) {
	stat := gs.wordSessionStats[kana]
	stat.Kana = kana
	stat.MissCount++
	gs.wordSessionStats[kana] = stat
	gs.wordSessionDirty = true

	if prev, known := gs.wordMemory[kana]; known {
		gs.wordMemory[kana] = memory.Miss(prev, gs.now())
		gs.wordMemoryDirty[kana] = true
	}

	gs.lowerTempo()
}

// wordCorrect is a reading's stored plus session correct count. Must be
// called under lock.
func (gs *GameState) wordCorrect(kana string) int {
	return gs.wordOverallStats[kana].CorrectCount + gs.wordSessionStats[kana].CorrectCount
}

// mergeWordStats writes (baseline + session) word stats, independently of
// the kana stats. A failed baseline read keeps wordSessionDirty so the next
// merge retries. Must be called under lock.
func (gs *GameState) mergeWordStats() {
	if !gs.wordSessionDirty {
		return
	}

	var baseline map[string]store.WordStats
	if gs.store != nil {
		stats, err := gs.store.WordStatistics()
		if err != nil {
			return
		}
		baseline = stats
	}
	if baseline == nil {
		baseline = make(map[string]store.WordStats)
	}

	for kana, session := range gs.wordSessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 {
			continue
		}
		base := baseline[kana]
		base.Kana = kana
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		if gs.store != nil {
			if err := gs.store.SaveWordStats(kana, base.CorrectCount, base.MissCount); err != nil {
				continue
			}
		}
		gs.wordOverallStats[kana] = base
		delete(gs.wordSessionStats, kana)
	}

	gs.wordSessionDirty = false
}

// secureCorrect is the correct count at which a kana or word counts as
// secure (the panel's "mastered").
const secureCorrect = 3

// loadWords builds the word index: the private lists merged into the
// bundled index, which is built once per process. A bundled load error is
// kept for the status panel; kana play is unaffected. An unreadable list
// folder gives no private lists (stage 4 shows list problems). Called from
// NewGameState before the game starts, so no lock is needed.
func (gs *GameState) loadWords() {
	base := vocab.NewIndex(nil, nil)
	if _, err := bundledWords(); err != nil {
		gs.wordsErr = err.Error()
	} else {
		base = bundledIndex()
	}
	var lists []vocab.List
	if dir, err := wordsDir(); err == nil {
		lists, _ = vocab.LoadDir(dir)
	}
	gs.words = base.WithLists(lists)
	gs.invalidatePlayable()
}

// unitKnown reports whether a word unit is known in mixed mode: the kana is
// secure and its script is active. っ and ッ need their script, ー either
// script. Must be called under lock.
func (gs *GameState) unitKnown(unit string) bool {
	switch unit {
	case "っ":
		return gs.activeScripts[kanacore.ScriptHiragana]
	case "ッ":
		return gs.activeScripts[kanacore.ScriptKatakana]
	case "ー":
		return gs.activeScripts[kanacore.ScriptHiragana] || gs.activeScripts[kanacore.ScriptKatakana]
	}
	rowID, ok := kanacore.CharToRow[unit]
	if !ok || !gs.activeScripts[kanacore.ScriptOf(rowID)] {
		return false
	}
	return gs.overallStats[unit].CorrectCount+gs.sessionStats[unit].CorrectCount >= secureCorrect
}

// playableWords returns the entries playable in mixed mode: bundled words up
// to wordLevel plus every private word, all of whose units are known. The
// result is cached until invalidatePlayable; do not modify it. Must be
// called under lock.
func (gs *GameState) playableWords() []vocab.Entry {
	if gs.words == nil {
		return nil
	}
	if !gs.playableOK {
		gs.playable = vocab.Playable(gs.words.Entries(), gs.unitKnown, vocab.Filter{MaxLevel: gs.wordLevel})
		gs.playableOK = true
	}
	return gs.playable
}

// invalidatePlayable marks the playable cache stale. Call it whenever a kana
// may have become secure, the active scripts, the level or the index
// changed. Must be called under lock.
func (gs *GameState) invalidatePlayable() {
	gs.playableOK = false
}
