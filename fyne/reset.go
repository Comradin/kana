package main

import (
	"errors"

	"kana/kanacore"
)

// errResetBusy refuses a reset while a dialog is pending or the game is
// over: Reset drops pending offers, and the katakana offer, already saved
// as made, would be lost.
var errResetBusy = errors.New("Finish the open dialog first.") //nolint:staticcheck // shown to the learner as is

// scriptChars returns every character of the script's rows.
func scriptChars(s kanacore.Script) []string {
	var chars []string
	for _, row := range kanacore.RowsFor(s) {
		chars = append(chars, row.Characters()...)
	}
	return chars
}

// ResetScript deletes the stats, memory and learning progress of every
// character of the script and starts its learning path again: all its rows
// are deselected and, if the script is active, its first step is selected
// with its intro queued and auto-progression on. The other script, the
// katakana offer and all word data stay. Nothing changes on error. The
// caller restarts the round.
func (gs *GameState) ResetScript(s kanacore.Script) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.over || len(gs.pendingIntro) > 0 || gs.pendingOffer || len(gs.pendingPackage) > 0 || gs.pendingLevelOffer {
		return errResetBusy
	}
	// Merge first, so a later merge of older session data cannot bring
	// the deleted counts back.
	gs.mergeSessionStats()
	chars := scriptChars(s)
	if gs.store != nil {
		if err := gs.store.ResetKana(chars); err != nil {
			return err
		}
	}
	for _, c := range chars {
		delete(gs.overallStats, c)
		delete(gs.sessionStats, c)
		delete(gs.currentStreak, c)
		delete(gs.memory, c)
		delete(gs.memoryDirty, c)
		delete(gs.kanaProgress, c)
		delete(gs.kanaProgressDirty, c)
	}
	gs.invalidatePlayable()

	var rows []string
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] && row.Script != s {
			rows = append(rows, row.ID)
		}
	}
	first := kanacore.ProgressionStepsFor(s)[0]
	active := gs.activeScripts[s]
	if active {
		rows = append(rows, first...)
	}
	gs.setSelectedRowsLocked(rows)
	gs.autoProgress = true
	if gs.store != nil {
		_ = gs.store.SaveAutoProgress(true)
	}
	if active {
		gs.pendingIntro = append(gs.pendingIntro, first...)
	}
	return nil
}
