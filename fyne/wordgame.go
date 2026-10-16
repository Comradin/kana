package main

import (
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
		if gs.knownDirty[kana] {
			base.CorrectCount = max(base.CorrectCount, secureCorrect)
		}
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		if gs.store != nil {
			if err := gs.store.SaveWordStats(kana, base.CorrectCount, base.MissCount); err != nil {
				continue
			}
		}
		gs.wordOverallStats[kana] = base
		delete(gs.wordSessionStats, kana)
		delete(gs.knownDirty, kana)
	}

	gs.wordSessionDirty = false
}

// secureCorrect is the correct count at which a kana or word counts as
// secure (the panel's "mastered").
const secureCorrect = 3

// loadWords builds the word index: the private lists merged into the
// bundled index, which is built once per process. A bundled load error is
// kept for the status panel; kana play is unaffected. An unreadable list
// folder gives no private lists (stage 4 shows list problems) and sets
// listsLoadFailed. Called from NewGameState before the game starts, so no
// lock is needed.
func (gs *GameState) loadWords() {
	base := vocab.NewIndex(nil, nil)
	if _, err := bundledWords(); err != nil {
		gs.wordsErr = err.Error()
	} else {
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

// playableWords returns the entries that may fall: introduced words all of
// whose units are known, from every source and level, sorted by reading.
// None while the introduced words could not be loaded. The result is
// cached until invalidatePlayable; do not modify it. Must be called under
// lock.
func (gs *GameState) playableWords() []vocab.Entry {
	if gs.words == nil || gs.introducedLoadFailed {
		return nil
	}
	if !gs.playableOK {
		var out []vocab.Entry
		for kana := range gs.introduced {
			if e, ok := gs.words.Lookup(kana); ok && gs.unitsKnown(e.Units) {
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
// may have become secure, or the active scripts, the introduced words or
// the index changed. Must be called under lock.
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

// introducedCounts counts the introduced readings in the word index and
// those of them that are secure. Must be called under lock.
func (gs *GameState) introducedCounts() (introduced, secure int) {
	for kana := range gs.introduced {
		if _, ok := gs.words.Lookup(kana); !ok {
			continue
		}
		introduced++
		if gs.wordCorrect(kana) >= secureCorrect {
			secure++
		}
	}
	return introduced, secure
}

// wordsSummary is the settings' Words tab line: "Introduced: 25 · secure:
// 18 · left in source: 101", or that the chosen list is missing. Must be
// called under lock.
func (gs *GameState) wordsSummary() string {
	src, found := gs.source()
	if !found {
		return fmt.Sprintf("List %s not found", prettyListName(src.List))
	}
	if gs.words == nil {
		return "No words available"
	}
	introduced, secure := gs.introducedCounts()
	return fmt.Sprintf("Introduced: %d · secure: %d · left in source: %d", introduced, secure, gs.sourceLeft(src))
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
	ws.Introduced, ws.Secure = gs.introducedCounts()
	if found && !ws.LoadFailed && len(vocab.NextPackage(gs.words, src, gs.introduced, 1, gs.unitKnown)) == 0 {
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
