package main

import "kana/store"

// A kana or word is learned at learnedLevel. A hit raises its level by one,
// a mistake lowers it by mistakeDrop and flags it until it is learned again.
const (
	learnedLevel = 5
	mistakeDrop  = 2
)

// learned reports whether level counts as learned.
func learned(level int) bool {
	return level >= learnedLevel
}

// hit returns p after a hit: one level up, at most learnedLevel. Reaching
// learnedLevel clears the flag.
func hit(p store.Progress) store.Progress {
	p.Level = min(p.Level+1, learnedLevel)
	if learned(p.Level) {
		p.Flagged = false
	}
	return p
}

// mistake returns p after a mistake: mistakeDrop levels down, at least 0,
// and flagged.
func mistake(p store.Progress) store.Progress {
	p.Level = max(p.Level-mistakeDrop, 0)
	p.Flagged = true
	return p
}

// kanaLevel is a character's learning level. Must be called under lock.
func (gs *GameState) kanaLevel(char string) int {
	return gs.kanaProgress[char].Level
}

// wordLearned reports whether a reading is learned. Must be called under
// lock.
func (gs *GameState) wordLearned(kana string) bool {
	return learned(gs.wordProgress[kana].Level)
}

// raiseKana records a hit on a character and reports whether it became
// learned with it. Must be called under lock.
func (gs *GameState) raiseKana(char string) bool {
	before := gs.kanaProgress[char]
	gs.kanaProgress[char] = hit(before)
	gs.kanaProgressDirty[char] = true
	return !learned(before.Level) && learned(gs.kanaProgress[char].Level)
}

// lowerKana records a mistake on a character and reports whether it was
// learned before. Must be called under lock.
func (gs *GameState) lowerKana(char string) bool {
	before := gs.kanaProgress[char]
	gs.kanaProgress[char] = mistake(before)
	gs.kanaProgressDirty[char] = true
	return learned(before.Level)
}

// raiseWord records a hit on a reading. Must be called under lock.
func (gs *GameState) raiseWord(kana string) {
	gs.wordProgress[kana] = hit(gs.wordProgress[kana])
	gs.wordProgressDirty[kana] = true
}

// lowerWord records a mistake on a reading. Must be called under lock.
func (gs *GameState) lowerWord(kana string) {
	gs.wordProgress[kana] = mistake(gs.wordProgress[kana])
	gs.wordProgressDirty[kana] = true
}

// saveProgress writes changed kana and word progress. A failed save stays
// dirty for the next merge. If loading a kind failed, nothing of that kind
// is written, so the stored levels are not replaced by fresh ones. Must be
// called under lock.
func (gs *GameState) saveProgress() {
	if gs.store == nil {
		return
	}
	if !gs.kanaProgressLoadFailed {
		for char := range gs.kanaProgressDirty {
			if err := gs.store.SaveProgress("kana:"+char, gs.kanaProgress[char]); err == nil {
				delete(gs.kanaProgressDirty, char)
			}
		}
	}
	if !gs.wordProgressLoadFailed {
		for kana := range gs.wordProgressDirty {
			if err := gs.store.SaveProgress("word:"+kana, gs.wordProgress[kana]); err == nil {
				delete(gs.wordProgressDirty, kana)
			}
		}
	}
}
