package main

import (
	"time"

	"kana/memory"
	"kana/vocab"
)

// packageSizes are the package sizes the settings offer.
var packageSizes = []int{3, 5, 10}

func validPackageSize(n int) bool {
	for _, s := range packageSizes {
		if s == n {
			return true
		}
	}
	return false
}

// wordSettings are the settings dialog's Words tab values.
type wordSettings struct {
	source      string // jlptSource or listSourcePrefix + a list name
	level       int    // JLPT level packages come down to, 1–5
	packageSize int    // 3, 5 or 10
}

// readingsOf returns the readings of entries, in order.
func readingsOf(entries []vocab.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Kana
	}
	return out
}

// newestPackageDone reports whether the newest package is learned well
// enough for the next one: at least ceil(0.8 × p) of its p readings that
// are playable now are secure. With none playable (a script switched off,
// a list gone) it counts as done. Must be called under lock.
func (gs *GameState) newestPackageDone() bool {
	playable, secured := 0, 0
	for _, kana := range gs.newestPackage {
		e, ok := gs.words.Lookup(kana)
		if !ok || !gs.unitsKnown(e.Units) {
			continue
		}
		playable++
		if gs.wordCorrect(kana) >= secureCorrect {
			secured++
		}
	}
	return playable == 0 || secured*5 >= playable*4
}

// packageDue returns the next word package if one is due, or reports that
// the level offer is due instead: when the JLPT source has no word left at
// the current level and an easier level remains. Nothing is due when over,
// while a package or the level offer is pending, without words or without
// the introduced words, while the newest package is not learned yet, or
// when the chosen list is missing. Must be called under lock.
func (gs *GameState) packageDue() (pkg []vocab.Entry, levelOffer bool) {
	if gs.over || gs.words == nil || gs.introducedLoadFailed || len(gs.pendingPackage) > 0 || gs.pendingLevelOffer {
		return nil, false
	}
	if !gs.newestPackageDone() {
		return nil, false
	}
	src, found := gs.source()
	if !found {
		return nil, false
	}
	if pkg := vocab.NextPackage(gs.words, src, gs.introduced, gs.packageSize, gs.unitKnown); len(pkg) > 0 {
		return pkg, false
	}
	offer := src.List == "" && gs.wordsErr == "" && gs.wordLevel > 1 && !gs.levelOfferDeclined && gs.sourceLeft(src) == 0
	return nil, offer
}

// maybeOfferPackage queues the next word package or the level offer if one
// is due, and pauses the game until its dialog is done. Pending is set only
// when the event is delivered; otherwise the next re-check point tries
// again. Must be called under lock.
func (gs *GameState) maybeOfferPackage() {
	pkg, level := gs.packageDue()
	switch {
	case len(pkg) > 0:
		select {
		case gs.eventCh <- gameEvent{kind: wordPackageEvent}:
			gs.pendingPackage = readingsOf(pkg)
			gs.paused = true
		default:
		}
	case level:
		select {
		case gs.eventCh <- gameEvent{kind: levelOfferEvent}:
			gs.pendingLevelOffer = true
			gs.paused = true
		default:
		}
	}
}

// offerPackageAtStart sets a due package or level offer pending without an
// event, like startLearningPath sets the first intro: the first
// showPendingDialogs shows it. Called from NewGameState before the game
// starts, so no lock is needed.
func (gs *GameState) offerPackageAtStart() {
	pkg, level := gs.packageDue()
	switch {
	case len(pkg) > 0:
		gs.pendingPackage = readingsOf(pkg)
		gs.paused = true
	case level:
		gs.pendingLevelOffer = true
		gs.paused = true
	}
}

// introduce marks a reading as introduced and saves it at once; a failed
// save is kept in introducedDirty, which mergeSessionStats retries. Must be
// called under lock.
func (gs *GameState) introduce(kana string, now time.Time) {
	gs.introduced[kana] = true
	if gs.store == nil || gs.introducedLoadFailed {
		return
	}
	if err := gs.store.SaveIntroduced(kana, now); err != nil {
		gs.introducedDirty[kana] = true
	}
}

// saveIntroducedDirty retries introduced readings whose save failed. Must
// be called under lock.
func (gs *GameState) saveIntroducedDirty() {
	if gs.store == nil || gs.introducedLoadFailed {
		return
	}
	for kana := range gs.introducedDirty {
		if err := gs.store.SaveIntroduced(kana, gs.now()); err == nil {
			delete(gs.introducedDirty, kana)
		}
	}
}

// markKnown records a word the learner already knows as secure: its stored
// correct count becomes at least secureCorrect, written at once and kept
// out of the session stats so the session tiles stay honest, and it gets
// the memory state of three correct answers. No points, no tempo change.
// Must be called under lock.
func (gs *GameState) markKnown(kana string, now time.Time) {
	ws := gs.wordOverallStats[kana]
	ws.Kana = kana
	ws.CorrectCount = max(ws.CorrectCount, secureCorrect)
	if gs.store != nil {
		_ = gs.store.SaveWordStats(kana, ws.CorrectCount, ws.MissCount)
	}
	gs.wordOverallStats[kana] = ws
	gs.wordMemory[kana] = memory.Seed(secureCorrect, now)
	gs.wordMemoryDirty[kana] = true
}

// FinishPackage closes the word package intro: every pending reading
// becomes introduced, the ones in known count as secure, and the others
// form the newest package for the 80 % rule. Both are saved at once. Then
// the game resumes unless another dialog is due.
func (gs *GameState) FinishPackage(known []string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	now := gs.now()
	isKnown := make(map[string]bool, len(known))
	for _, kana := range known {
		isKnown[kana] = true
	}
	var newest []string
	for _, kana := range gs.pendingPackage {
		gs.introduce(kana, now)
		if isKnown[kana] {
			gs.markKnown(kana, now)
		} else {
			newest = append(newest, kana)
		}
	}
	gs.newestPackage = newest
	if gs.store != nil {
		_ = gs.store.SaveNewestPackage(newest)
	}
	gs.pendingPackage = nil
	gs.invalidatePlayable()
	gs.settle()
	gs.maybeOfferPackage()
}

// ReplacementWord adds the next word of the source to the pending package,
// for a word the learner ticked as already known, and returns it; false
// when the source has none left.
func (gs *GameState) ReplacementWord() (vocab.Entry, bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.words == nil || len(gs.pendingPackage) == 0 {
		return vocab.Entry{}, false
	}
	src, found := gs.source()
	if !found {
		return vocab.Entry{}, false
	}
	skip := make(map[string]bool, len(gs.introduced)+len(gs.pendingPackage))
	for kana := range gs.introduced {
		skip[kana] = true
	}
	for _, kana := range gs.pendingPackage {
		skip[kana] = true
	}
	next := vocab.NextPackage(gs.words, src, skip, 1, gs.unitKnown)
	if len(next) == 0 {
		return vocab.Entry{}, false
	}
	gs.pendingPackage = append(gs.pendingPackage, next[0].Kana)
	return next[0], true
}

// AcceptLevelOffer lowers the JLPT level by one (N5 → N4) and offers the
// first package of the new level.
func (gs *GameState) AcceptLevelOffer() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.pendingLevelOffer = false
	if gs.wordLevel > 1 {
		gs.wordLevel--
		if gs.store != nil {
			_ = gs.store.SaveWordLevel(gs.wordLevel)
		}
	}
	gs.invalidatePlayable()
	gs.settle()
	gs.maybeOfferPackage()
}

// DeclineLevelOffer dismisses the level offer for the rest of the app run.
func (gs *GameState) DeclineLevelOffer() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.pendingLevelOffer = false
	gs.levelOfferDeclined = true
	gs.settle()
}

// applyWordSettings applies the settings dialog's Words tab. A new source
// clears the newest package, so its first package can come at once; words
// already introduced keep falling. The caller re-checks packages
// (applySettings does).
func (gs *GameState) applyWordSettings(ws wordSettings) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	changed := false
	if ws.source != "" && ws.source != gs.wordSource {
		gs.wordSource = ws.source
		gs.newestPackage = nil
		changed = true
		if gs.store != nil {
			_ = gs.store.SaveWordSource(ws.source)
			_ = gs.store.SaveNewestPackage(nil)
		}
	}
	if ws.level >= 1 && ws.level <= 5 && ws.level != gs.wordLevel {
		gs.wordLevel = ws.level
		changed = true
		if gs.store != nil {
			_ = gs.store.SaveWordLevel(ws.level)
		}
	}
	if validPackageSize(ws.packageSize) && ws.packageSize != gs.packageSize {
		gs.packageSize = ws.packageSize
		changed = true
		if gs.store != nil {
			_ = gs.store.SavePackageSize(ws.packageSize)
		}
	}
	if changed {
		gs.invalidatePlayable()
		gs.dropInactiveTiles()
	}
}
