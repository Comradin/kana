package main

import (
	"kana/memory"
	"kana/store"
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
