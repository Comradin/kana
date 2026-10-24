package main

import (
	"slices"
	"strings"
	"time"

	"kana/memory"
	"kana/store"
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
	source          string   // jlptSource or listSourcePrefix + a list name
	level           int      // JLPT level packages come down to, 1–5
	packageSize     int      // 3, 5 or 10
	wordsOnly       bool     // only words fall
	lessons         []string // the lesson filter of a list source; nil = all lessons
	meaningLang     string   // "" = System
	ask             wordKind // what word tiles ask for
	readingPractice bool     // reading practice is on
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
// can fall now are learned in what tiles ask for (gs.wordsAsk). In meaning
// mode a word without a meaning to ask for (see askable) counts by its
// reading level: it never falls there, so skipping it would let packages
// come back to back. With none of them (a script switched off, a list
// gone) it counts as done. Must be called under lock.
func (gs *GameState) newestPackageDone() bool {
	playable, learnedN := 0, 0
	for _, kana := range gs.newestPackage {
		e, ok := gs.words.Lookup(kana)
		if !ok || !gs.unitsFit(e.Units) {
			continue
		}
		playable++
		kind := gs.wordsAsk
		if !gs.askable(e) {
			kind = readingKind
		}
		if gs.wordLearned(kana, kind) {
			learnedN++
		}
	}
	return playable == 0 || learnedN*5 >= playable*4
}

// packageDue returns the next word package if one is due, or reports that
// the level offer is due instead: when the JLPT source has no word left at
// the current level and an easier level remains. Nothing is due when over,
// in reading practice, while a package or the level offer is pending,
// without words or without the introduced words, while the newest package
// is not learned yet, or when the chosen list is missing. Must be called
// under lock.
func (gs *GameState) packageDue() (pkg []vocab.Entry, levelOffer bool) {
	if gs.over || gs.readingPractice || gs.words == nil || gs.introducedLoadFailed || len(gs.pendingPackage) > 0 || gs.pendingLevelOffer {
		return nil, false
	}
	if !gs.newestPackageDone() {
		return nil, false
	}
	src, found := gs.source()
	if !found {
		return nil, false
	}
	if pkg := vocab.NextPackage(gs.words, src, gs.introduced, gs.packageSize, gs.unitFits); len(pkg) > 0 {
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

// knownSeed is the memory a word ticked as "already known" starts with: the
// state of three correct answers.
const knownSeed = 3

// markKnown records a word the learner already knows: it becomes learned
// (level 5, not flagged), saved at once, and gets the memory state of
// knownSeed correct answers. Its stats keep their real counts. No points,
// no tempo change. A failed save stays dirty and mergeSessionStats retries
// it. Must be called under lock.
func (gs *GameState) markKnown(kana string, now time.Time) {
	gs.wordProgress[kana] = store.Progress{Level: learnedLevel}
	gs.wordProgressDirty[kana] = true
	gs.saveProgress()
	gs.wordMemory[kana] = memory.Seed(knownSeed, now)
	gs.wordMemoryDirty[kana] = true
}

// saveNewestPackage saves the newest package; a failed save sets
// newestPackageDirty, which mergeSessionStats retries. Must be called under
// lock.
func (gs *GameState) saveNewestPackage() {
	if gs.store == nil {
		return
	}
	gs.newestPackageDirty = gs.store.SaveNewestPackage(gs.newestPackage) != nil
}

// FinishPackage closes the word package intro: every pending reading
// becomes introduced, the ones in known count as learned, and the others
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
	gs.saveNewestPackage()
	gs.pendingPackage = nil
	gs.invalidatePlayable()
	gs.settle()
	gs.maybeOfferPackage()
}

// ReplacementWord adds the next word of the source to the pending package,
// for a word the learner ticked as already known, and returns it; false
// when the source has none left. The replacement keeps to what the dialog
// title names: a list package's lesson, a JLPT package's levels.
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
	var pending []vocab.Entry
	skip := make(map[string]bool, len(gs.introduced)+len(gs.pendingPackage))
	for kana := range gs.introduced {
		skip[kana] = true
	}
	for _, kana := range gs.pendingPackage {
		skip[kana] = true
		if e, ok := gs.words.Lookup(kana); ok {
			pending = append(pending, e)
		}
	}
	if len(pending) == 0 {
		return vocab.Entry{}, false
	}
	next := vocab.NextPackageWhere(gs.words, src, skip, 1, gs.unitFits, samePackage(gs.words, src, pending))
	if len(next) == 0 {
		return vocab.Entry{}, false
	}
	gs.pendingPackage = append(gs.pendingPackage, next[0].Kana)
	return next[0], true
}

// samePackage accepts the entries that fit a package of pending: for a list
// the lesson of its words (a list package holds one lesson), for JLPT the
// levels its words span.
func samePackage(ix *vocab.Index, src vocab.Source, pending []vocab.Entry) func(vocab.Entry) bool {
	if src.List != "" {
		lesson := ix.LessonIn(pending[0], src)
		return func(e vocab.Entry) bool { return ix.LessonIn(e, src) == lesson }
	}
	lo, hi := pending[0].JLPT, pending[0].JLPT
	for _, e := range pending[1:] {
		lo, hi = min(lo, e.JLPT), max(hi, e.JLPT)
	}
	return func(e vocab.Entry) bool { return e.JLPT >= lo && e.JLPT <= hi }
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

// applyWordSettings applies the settings dialog's Words tab. A new source or
// lesson filter clears the newest package, so its first package can come at
// once; words already introduced keep falling; switching words-only mode
// keeps it. A new "Words ask for" applies to tiles spawned afterwards. So does reading practice. The
// caller re-checks packages (applySettings does).
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
		}
		gs.saveNewestPackage()
	}
	if name, ok := strings.CutPrefix(ws.source, listSourcePrefix); ok && !slices.Equal(ws.lessons, gs.lessonFilters[name]) {
		// Only a change in the lessons that count restarts the rhythm; a
		// stale lesson or another order does not.
		if !sameLessons(gs.effectiveLessons(name, ws.lessons), gs.effectiveLessons(name, gs.lessonFilters[name])) {
			gs.newestPackage = nil
			gs.saveNewestPackage()
			changed = true
		}
		gs.setLessonFilter(name, ws.lessons)
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
	if ws.wordsOnly != gs.wordsOnly {
		gs.wordsOnly = ws.wordsOnly
		if gs.introPending() {
			gs.paused = true // the kana intros that waited are shown now
		}
		changed = true
		if gs.store != nil {
			_ = gs.store.SaveWordsOnly(ws.wordsOnly)
		}
	}
	if ws.meaningLang != gs.meaningSetting {
		gs.meaningSetting = ws.meaningLang
		gs.meaningLang = resolveMeaningLang(ws.meaningLang, systemLanguage())
		if gs.store != nil {
			_ = gs.store.SaveMeaningLang(ws.meaningLang)
		}
	}
	if ws.ask != gs.wordsAsk {
		// Only new tiles follow; falling tiles keep their question, and
		// the playable words do not depend on it, so nothing is dropped.
		gs.wordsAsk = ws.ask
		if gs.store != nil {
			_ = gs.store.SaveWordsAsk(ws.ask.setting())
		}
	}
	if ws.readingPractice != gs.readingPractice {
		// Only new tiles follow; falling tiles keep their rules (see
		// KanaTile.practice), and the readable words do not depend on
		// it, so nothing is dropped.
		gs.readingPractice = ws.readingPractice
		if gs.store != nil {
			_ = gs.store.SaveReadingPractice(ws.readingPractice)
		}
	}
	if changed {
		gs.invalidatePlayable()
		gs.dropInactiveTiles()
	}
}

// sameLessons reports whether two lesson filters hold the same lessons, in
// any order.
func sameLessons(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// setLessonFilter sets and saves a list's lesson filter; nil or empty means
// all lessons. Must be called under lock.
func (gs *GameState) setLessonFilter(list string, lessons []string) {
	if gs.lessonFilters == nil {
		gs.lessonFilters = map[string][]string{}
	}
	if len(lessons) == 0 {
		delete(gs.lessonFilters, list)
	} else {
		gs.lessonFilters[list] = slices.Clone(lessons)
	}
	if gs.store != nil {
		_ = gs.store.SaveLessonFilter(list, lessons)
	}
}
