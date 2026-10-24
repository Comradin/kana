package main

import (
	"math"

	"kana/kanacore"
	"kana/vocab"
)

// Reading practice: every word of the source whose kana are all learned
// may fall, introduced or not, and a word read counts for its kana, not
// for the word.

// readableWords returns the words that may fall in reading practice: every
// entry of the source (see source: JLPT up to the chosen level, or the
// chosen list with its lesson filter) whose units are all known (see
// unitsKnown), introduced or not, sorted by reading. None while the source
// is missing. The result is cached until invalidatePlayable; do not modify
// it. Must be called under lock.
func (gs *GameState) readableWords() []vocab.Entry {
	if gs.words == nil {
		return nil
	}
	if !gs.readableOK {
		var out []vocab.Entry
		if src, found := gs.source(); found {
			for _, e := range gs.words.Entries() {
				if src.Has(e) && gs.unitsKnown(e.Units) {
					out = append(out, e)
				}
			}
		}
		gs.readable = out
		gs.readableOK = true
	}
	return gs.readable
}

// pickReadable is pickWord in reading practice: whether this spawn is a
// word is drawn with gs.rnd as in pickWord, over the readable words; the
// word is drawn uniformly with gs.rnd among those not already falling (see
// wordCandidates). The focus on weak items applies to kana tiles only.
// With no readable word it returns false without drawing. Must be called
// under lock.
func (gs *GameState) pickReadable(always bool) (vocab.Entry, bool) {
	pool := gs.readableWords()
	if len(pool) == 0 || (!always && gs.rnd() >= wordShare(len(pool))) {
		return vocab.Entry{}, false
	}
	candidates := gs.wordCandidates(pool)
	i := min(int(gs.rnd()*float64(len(candidates))), len(candidates)-1)
	return gs.words.Lookup(candidates[i])
}

// recordReading counts a word read in reading practice: every known kana
// unit (see unitKnown) of the reading input is a romaji reading of (see
// vocab.Entry.MatchedReading) gets a hit as on a kana tile (see
// creditKana) — a short form only its own kana, a kana that occurs twice
// two hits, っ, ッ and ー none — then the tempo rises once and
// auto-progression runs. The shown reading's kana are all known (see
// readableWords), but another variant or merged reading may have kana that
// are not learned or not in an active script; those get nothing. The
// word's own progress is not touched. Reports whether rows were announced.
// Must be called under lock.
func (gs *GameState) recordReading(e vocab.Entry, input string) bool {
	reading, ok := e.MatchedReading(input)
	if !ok {
		return false
	}
	units, _ := kanacore.KanaUnits(reading)
	for _, u := range units {
		if _, isKana := kanacore.CharToRow[u]; isKana && gs.unitKnown(u) {
			gs.creditKana(u)
			gs.readKana++
		}
	}
	gs.raiseTempo()
	return gs.progressAfterHit()
}

// readingSet returns the readings of entries as a set.
func readingSet(entries []vocab.Entry) map[string]bool {
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		set[e.Kana] = true
	}
	return set
}

// kanaPerMinute is the kana units of words read per minute of play this
// round (see readKana and playTicks), rounded; 0 before any play. Must be
// called under lock.
func (gs *GameState) kanaPerMinute() int {
	if gs.playTicks == 0 {
		return 0
	}
	minutes := float64(gs.playTicks) / (ticksPerSecond * 60)
	return int(math.Round(float64(gs.readKana) / minutes))
}
