package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

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
	gs.raiseWord(kana)

	gs.raiseTempo()
}

// applyWordMistake records a mistake on a word: session miss, word memory
// miss (if known), level down and flag. Its kana are not touched. Must be
// called under lock.
func (gs *GameState) applyWordMistake(kana string) {
	stat := gs.wordSessionStats[kana]
	stat.Kana = kana
	stat.MissCount++
	gs.wordSessionStats[kana] = stat
	gs.wordSessionDirty = true

	if prev, known := gs.wordMemory[kana]; known {
		gs.wordMemory[kana] = memory.Miss(prev, gs.now())
		gs.wordMemoryDirty[kana] = true
	}
	gs.lowerWord(kana)
}

// recordWordMiss records a word that fell through: a mistake plus the
// tempo. Must be called under lock.
func (gs *GameState) recordWordMiss(kana string) {
	gs.applyWordMistake(kana)
	gs.lowerTempo()
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

// wordsDirPerm is the permission of the word list folder.
const wordsDirPerm = 0o755

// loadWords builds the word index: the private lists merged into the
// bundled index, which is built once per process. A bundled load error is
// kept for the status panel; kana play is unaffected. The word list folder
// is created if it is missing. Called from NewGameState before the game
// starts, so no lock is needed.
func (gs *GameState) loadWords() {
	if _, err := bundledWords(); err != nil {
		gs.wordsErr = err.Error()
	}
	if dir, err := wordsDir(); err == nil {
		_ = os.MkdirAll(dir, wordsDirPerm) // an error shows as listsLoadFailed
	}
	gs.loadLists()
}

// loadLists reads the private lists and rebuilds the word index on the
// bundled one (an empty index if that failed to load). An unreadable folder
// gives no private lists and sets listsLoadFailed. Must be called under lock
// (or before the game starts).
func (gs *GameState) loadLists() {
	base := vocab.NewIndex(nil, nil)
	if gs.wordsErr == "" {
		base = bundledIndex()
	}
	var lists []vocab.List
	gs.listsLoadFailed = true
	if dir, err := wordsDir(); err == nil {
		gs.wordsDir = dir
		var loadErr error
		lists, loadErr = vocab.LoadDir(dir)
		gs.listsLoadFailed = loadErr != nil
	}
	gs.listNames = nil
	for _, l := range lists {
		gs.listNames = append(gs.listNames, l.Name)
	}
	gs.words = base.WithLists(lists)
	gs.invalidatePlayable()
}

// ListNames returns the private word lists found, sorted by name.
func (gs *GameState) ListNames() []string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return append([]string(nil), gs.listNames...)
}

// MeaningLang returns the language meanings are shown in.
func (gs *GameState) MeaningLang() string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.meaningLang
}

// ListFiles returns the names (with extension) of the word list files named
// like name, ignoring case and extension.
func (gs *GameState) ListFiles(name string) []string {
	gs.mu.Lock()
	dir := gs.wordsDir
	gs.mu.Unlock()
	if dir == "" {
		return nil
	}
	return vocab.Existing(dir, name)
}

// ImportList stores c in the word list folder, reloads the lists and
// returns the word list's name (an existing list keeps its name).
func (gs *GameState) ImportList(c vocab.Candidate) (string, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.wordsDir == "" {
		return "", errors.New("no folder for word lists")
	}
	name, err := vocab.Import(gs.wordsDir, c)
	// Also after an error: a partial failure may have changed the folder.
	gs.loadLists()
	return name, err
}

// unitActive reports whether a word unit's script is active: a kana's own
// script; っ and ッ need their script, ー either script. Must be called
// under lock.
func (gs *GameState) unitActive(unit string) bool {
	switch unit {
	case "っ":
		return gs.activeScripts[kanacore.ScriptHiragana]
	case "ッ":
		return gs.activeScripts[kanacore.ScriptKatakana]
	case "ー":
		return gs.activeScripts[kanacore.ScriptHiragana] || gs.activeScripts[kanacore.ScriptKatakana]
	}
	rowID, ok := kanacore.CharToRow[unit]
	return ok && gs.activeScripts[kanacore.ScriptOf(rowID)]
}

// unitKnown reports whether a word unit is known in mixed mode: its script
// is active and, for a kana, the kana is learned. Must be called under lock.
func (gs *GameState) unitKnown(unit string) bool {
	switch unit {
	case "っ", "ッ", "ー":
		return gs.unitActive(unit)
	}
	return gs.unitActive(unit) && learned(gs.kanaLevel(unit))
}

// unitsKnown reports whether every unit of a word is known. Must be called
// under lock.
func (gs *GameState) unitsKnown(units []string) bool {
	for _, u := range units {
		if !gs.unitKnown(u) {
			return false
		}
	}
	return true
}

// unitFits reports whether a word unit lets its word fall and be
// introduced in the current mode: known in mixed mode, of an active script
// in words-only mode. Must be called under lock.
func (gs *GameState) unitFits(unit string) bool {
	if gs.wordsOnly {
		return gs.unitActive(unit)
	}
	return gs.unitKnown(unit)
}

// unitsFit reports whether every unit of a word fits (see unitFits). Must
// be called under lock.
func (gs *GameState) unitsFit(units []string) bool {
	for _, u := range units {
		if !gs.unitFits(u) {
			return false
		}
	}
	return true
}

// practised reports whether an introduced word belongs to what is
// practised: in words-only mode the source with its lesson filter; in mixed
// mode every introduced word, unless a lesson filter is set, which limits
// it to the source as well. found is whether the source exists. Must be
// called under lock.
func (gs *GameState) practised(e vocab.Entry, src vocab.Source, found bool) bool {
	if !gs.wordsOnly && len(src.Lessons) == 0 {
		return true
	}
	return found && src.Has(e)
}

// playableWords returns the entries that may fall: introduced words that
// are practised (see practised) and whose units fit the mode (see
// unitFits), sorted by reading. None while the introduced words could not
// be loaded. The result is cached until invalidatePlayable; do not modify
// it. Must be called under lock.
func (gs *GameState) playableWords() []vocab.Entry {
	if gs.words == nil || gs.introducedLoadFailed {
		return nil
	}
	if !gs.playableOK {
		src, found := gs.source()
		var out []vocab.Entry
		for kana := range gs.introduced {
			if e, ok := gs.words.Lookup(kana); ok && gs.practised(e, src, found) && gs.unitsFit(e.Units) {
				out = append(out, e)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Kana < out[j].Kana })
		gs.playable = out
		gs.playableOK = true
	}
	return gs.playable
}

// invalidatePlayable marks the playable cache stale. Call it whenever a kana
// may have become learned or stopped being learned, or the active scripts,
// the introduced words, the index, the mode, the source or its lesson
// filter changed. Must be called under lock.
func (gs *GameState) invalidatePlayable() {
	gs.playableOK = false
}

// wordCandidates returns the playable readings without those already
// falling as word tiles, unless that would leave none. Must be called under
// lock.
func (gs *GameState) wordCandidates(playable []vocab.Entry) []string {
	falling := make(map[string]bool)
	for _, t := range gs.tiles {
		if t.word != nil {
			falling[t.word.Kana] = true
		}
	}
	all := make([]string, 0, len(playable))
	free := make([]string, 0, len(playable))
	for _, e := range playable {
		all = append(all, e.Kana)
		if !falling[e.Kana] {
			free = append(free, e.Kana)
		}
	}
	if len(free) == 0 {
		return all
	}
	return free
}

// pickWord decides with gs.rnd whether this spawn is a word (wordShare) and,
// if so, picks one with memory.Pick over the word memory. With always set
// it skips the wordShare draw and picks a word whenever one is playable.
// With no playable word it returns false without drawing from gs.rnd. Must
// be called under lock.
func (gs *GameState) pickWord(always bool) (vocab.Entry, bool) {
	playable := gs.playableWords()
	if len(playable) == 0 || (!always && gs.rnd() >= wordShare(len(playable))) {
		return vocab.Entry{}, false
	}
	kana := memory.Pick(gs.wordCandidates(playable), gs.wordMemory, gs.now(), gs.rnd)
	return gs.words.Lookup(kana)
}

// source returns where new words come from and whether that source exists:
// a chosen list that was not found does not. Must be called under lock.
func (gs *GameState) source() (vocab.Source, bool) {
	if name, ok := strings.CutPrefix(gs.wordSource, listSourcePrefix); ok {
		for _, l := range gs.listNames {
			if l == name {
				return vocab.Source{List: name}, true
			}
		}
		return vocab.Source{List: name}, false
	}
	return vocab.Source{Level: gs.wordLevel}, true
}

// sourceLeft counts the source's entries that are not introduced yet (for
// JLPT: bundled entries with JLPT >= Level). Must be called under lock.
func (gs *GameState) sourceLeft(src vocab.Source) int {
	if gs.words == nil {
		return 0
	}
	n := 0
	for _, e := range gs.words.Entries() {
		if src.Has(e) && !gs.introduced[e.Kana] {
			n++
		}
	}
	return n
}

// introducedCounts counts the introduced readings in the word index, those
// of them that are learned and, in mixed mode, the practised ones that wait
// for a kana to be learned before they can fall; words-only mode does not
// wait for kana. Must be called under lock.
func (gs *GameState) introducedCounts() (introduced, learnedN, waiting int) {
	src, found := gs.source()
	for kana := range gs.introduced {
		e, ok := gs.words.Lookup(kana)
		if !ok {
			continue
		}
		introduced++
		if gs.wordLearned(kana) {
			learnedN++
		}
		if !gs.wordsOnly && gs.practised(e, src, found) && !gs.unitsKnown(e.Units) {
			waiting++
		}
	}
	return introduced, learnedN, waiting
}

// wordsSummary is the settings' Words tab line: "Introduced: 25 · learned:
// 18 · waiting for kana: 4 · left in source: 101" (the waiting part only when
// some wait), or that the chosen list is missing. Must be called under lock.
func (gs *GameState) wordsSummary() string {
	src, found := gs.source()
	if !found {
		return fmt.Sprintf("List %s not found", prettyListName(src.List))
	}
	if gs.words == nil {
		return "No words available"
	}
	introduced, learnedN, waiting := gs.introducedCounts()
	s := fmt.Sprintf("Introduced: %d · learned: %d", introduced, learnedN)
	if waiting > 0 {
		s += fmt.Sprintf(" · waiting for kana: %d", waiting)
	}
	return s + fmt.Sprintf(" · left in source: %d", gs.sourceLeft(src))
}

// wordsStatus summarises the words for the status panel. Must be called
// under lock.
func (gs *GameState) wordsStatus() WordsStatus {
	src, found := gs.source()
	ws := WordsStatus{LoadFailed: gs.introducedLoadFailed, Source: sourceLabel(src), Missing: !found}
	if src.List == "" {
		ws.Err, ws.Level = gs.wordsErr, src.Level
	}
	if gs.wordsDir != "" {
		home, _ := os.UserHomeDir()
		ws.Dir = displayDir(gs.wordsDir, home)
	}
	if gs.words == nil {
		return ws
	}
	ws.Introduced, ws.Learned, ws.Waiting = gs.introducedCounts()
	if found && !ws.LoadFailed && len(vocab.NextPackage(gs.words, src, gs.introduced, 1, gs.unitFits)) == 0 {
		ws.Next = wordsFinished
		if gs.sourceLeft(src) > 0 {
			ws.Next = wordsNeedKana
		}
	}
	return ws
}

// sessionWordCorrect is the number of word hits this session. Must be
// called under lock.
func (gs *GameState) sessionWordCorrect() int {
	n := 0
	for _, st := range gs.wordSessionStats {
		n += st.CorrectCount
	}
	return n
}
