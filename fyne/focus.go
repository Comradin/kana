package main

import (
	"kana/memory"
	"kana/store"
)

// focusShare is the share of picks that, with the focus on weak items, go
// to the weak candidates.
const focusShare = 0.5

// weakWeight is how strongly the focus pick favours an item with progress
// p: 2 if it slipped, 1 if it is learning (level 1–4), 0 if it is not weak
// (new or learned).
func weakWeight(p store.Progress) float64 {
	switch {
	case p.Flagged:
		return 2
	case p.Level >= 1 && !learned(p.Level):
		return 1
	}
	return 0
}

// pickFocus is the focus pick. With the focus on weak items and a weak
// candidate, it draws with gs.rnd whether this pick goes to the weak
// candidates (focusShare) and, if so, picks one weighted by weakWeight. It
// returns false without drawing when the focus is off or no candidate is
// weak, and false after the draw when the usual pick applies. Must be
// called under lock.
func (gs *GameState) pickFocus(candidates []string, progress map[string]store.Progress) (string, bool) {
	if !gs.focusWeak {
		return "", false
	}
	var weak []string
	var weights []float64
	total := 0.0
	for _, c := range candidates {
		if w := weakWeight(progress[c]); w > 0 {
			weak = append(weak, c)
			weights = append(weights, w)
			total += w
		}
	}
	if total == 0 || gs.rnd() >= focusShare {
		return "", false
	}
	x := gs.rnd() * total
	for i, w := range weights {
		if x < w {
			return weak[i], true
		}
		x -= w
	}
	return weak[len(weak)-1], true
}

// pickKana picks the next kana from candidates: the focus pick, else the
// spaced-repetition pick. Must be called under lock.
func (gs *GameState) pickKana(candidates []string) string {
	if c, ok := gs.pickFocus(candidates, gs.kanaProgress); ok {
		return c
	}
	return memory.Pick(candidates, gs.memory, gs.now(), gs.rnd)
}

// pickReading picks the next word reading from candidates, like pickKana
// over the word progress and memory. Must be called under lock.
func (gs *GameState) pickReading(candidates []string) string {
	if c, ok := gs.pickFocus(candidates, gs.wordProgress); ok {
		return c
	}
	return memory.Pick(candidates, gs.wordMemory, gs.now(), gs.rnd)
}
