package main

import "kana/vocab"

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
