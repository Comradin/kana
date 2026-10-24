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
