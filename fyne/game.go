package main

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"kana/kanacore"
	"kana/store"
)

type gameEventType int

const (
	gameOverEvent gameEventType = iota
	rowsUnlockedEvent
	katakanaOfferEvent
)

type gameEvent struct {
	kind gameEventType
	rows []string // rowsUnlockedEvent: IDs to introduce
}

// GameState holds all game state for the Fyne desktop game.
type GameState struct {
	mu sync.Mutex

	tiles       []*KanaTile
	score       int
	scoreLimit  int
	missed      int
	over        bool
	overReason  string
	missedKanas []kanacore.Kana

	sessionStats  map[string]store.KanaStats
	overallStats  map[string]store.KanaStats
	currentStreak map[string]int
	sessionDirty  bool

	selectedRows  map[string]bool
	autoProgress  bool
	activeScripts map[kanacore.Script]bool

	paused       bool     // tick, spawn and answers are suspended
	pendingIntro []string // row IDs waiting to be introduced

	pendingOffer    bool // the katakana offer dialog is waiting
	katakanaOffered bool // the offer was already made once

	userPaused bool // the learner paused with the button or Esc

	store *store.Store

	stopCh        chan struct{}
	eventCh       chan gameEvent
	eventChClosed bool

	objectSnapshot atomic.Value // []fyne.CanvasObject
	showPauseHint  atomic.Bool  // the canvas shows the "Paused" hint

	canvasW float32
	canvasH float32

	charSet kanacore.CharacterSet

	canvas *GameCanvas
}

// NewGameState constructs a new GameState, loading persisted state if store is non-nil.
func NewGameState(st *store.Store) *GameState {
	gs := &GameState{
		sessionStats:  make(map[string]store.KanaStats),
		overallStats:  make(map[string]store.KanaStats),
		currentStreak: make(map[string]int),
		selectedRows:  make(map[string]bool),
		activeScripts: map[kanacore.Script]bool{kanacore.ScriptHiragana: true},
		eventCh:       make(chan gameEvent, 4),
		stopCh:        make(chan struct{}),
		scoreLimit:    store.DefaultScoreLimit,
		charSet:       kanacore.AllKana(),
		store:         st,
		canvasW:       400,
		canvasH:       600,
	}

	for _, id := range kanacore.DefaultRowIDs() {
		gs.selectedRows[id] = true
	}

	if st != nil {
		if names, err := st.ActiveScripts(); err == nil && len(names) > 0 {
			gs.activeScripts = scriptSet(parseScripts(names))
		}
		if offered, err := st.KatakanaOffered(); err == nil {
			gs.katakanaOffered = offered
		}
		if stats, err := st.KanaStatistics(); err == nil {
			for _, stat := range stats {
				copied := stat
				gs.overallStats[stat.Char] = copied
				gs.currentStreak[stat.Char] = stat.Streak
			}
		}
		if auto, err := st.AutoProgress(); err == nil {
			gs.autoProgress = auto
		}
		if limit, err := st.ScoreLimit(); err == nil {
			if limit < 0 {
				limit = 0
			}
			gs.scoreLimit = limit
		}
		rows, err := st.SelectedRows()
		switch {
		case err != nil:
			// Keep the basic defaults; never overwrite what we could not read.
		case len(rows) > 0:
			gs.applySelectedRows(rows)
		default:
			gs.startLearningPath()
		}
	}

	return gs
}

// startLearningPath sets up a first launch: the first progression step plus
// every following step the learner has already mastered, with
// auto-progression on. Only a learner without any stats gets the intro for
// the first step. Must be called before the game starts (no lock needed).
func (gs *GameState) startLearningPath() {
	steps := kanacore.ProgressionStepsFor(kanacore.ScriptHiragana)
	gs.applySelectedRows(steps[0])
	for _, step := range steps[1:] {
		if !gs.rowsMastered(step) {
			break
		}
		for _, id := range step {
			gs.selectedRows[id] = true
		}
	}
	gs.autoProgress = true
	if gs.store != nil {
		_ = gs.store.SaveAutoProgress(true)
	}
	gs.saveSelectedRows()

	if !gs.hasStats() {
		gs.pendingIntro = append([]string(nil), steps[0]...)
		gs.paused = true
	}
}

// rowsMastered reports whether every given row is mastered. Must be called
// under lock, or before Start.
func (gs *GameState) rowsMastered(ids []string) bool {
	for _, id := range ids {
		row, ok := kanacore.RowByID(id)
		if !ok || !gs.isRowMastered(row) {
			return false
		}
	}
	return true
}

// hasStats reports whether the learner has answered or missed anything before.
func (gs *GameState) hasStats() bool {
	for _, stat := range gs.overallStats {
		if stat.CorrectCount > 0 || stat.MissCount > 0 {
			return true
		}
	}
	return false
}

// parseScripts converts stored names to scripts, dropping unknown names.
func parseScripts(names []string) []kanacore.Script {
	var scripts []kanacore.Script
	for _, name := range names {
		for _, info := range kanacore.Scripts() {
			if string(info.Script) == name {
				scripts = append(scripts, info.Script)
			}
		}
	}
	return scripts
}

// scriptSet builds the active-script set; an empty list means hiragana only.
func scriptSet(scripts []kanacore.Script) map[kanacore.Script]bool {
	set := make(map[kanacore.Script]bool)
	for _, s := range scripts {
		set[s] = true
	}
	if len(set) == 0 {
		set[kanacore.ScriptHiragana] = true
	}
	return set
}

// saveActiveScripts persists the active scripts in display order. Must be
// called under lock.
func (gs *GameState) saveActiveScripts() {
	if gs.store == nil {
		return
	}
	names := make([]string, 0, len(gs.activeScripts))
	for _, info := range kanacore.Scripts() {
		if gs.activeScripts[info.Script] {
			names = append(names, string(info.Script))
		}
	}
	_ = gs.store.SaveActiveScripts(names)
}

// hasSelectedRowOf reports whether any row of the script is selected. Must be
// called under lock.
func (gs *GameState) hasSelectedRowOf(s kanacore.Script) bool {
	for _, row := range kanacore.RowsFor(s) {
		if gs.selectedRows[row.ID] {
			return true
		}
	}
	return false
}

// Start launches the tick and spawn goroutines.
func (gs *GameState) Start(canvas *GameCanvas) {
	gs.mu.Lock()
	gs.canvas = canvas
	gs.mu.Unlock()
	go gs.tickLoop()
	go gs.spawnLoop()
}

// Reset clears state and prepares for a new session. Caller must call Start().
func (gs *GameState) Reset() {
	gs.mu.Lock()
	// Flush any pending session data to the store before wiping state.
	// mergeSessionStats requires the lock to be held.
	gs.mergeSessionStats()
	// close old channels
	select {
	case <-gs.stopCh:
		// already closed
	default:
		close(gs.stopCh)
	}
	gs.stopCh = make(chan struct{})
	// Close old event channel so any watchEvents goroutine exits cleanly.
	// Guard against double-close if Reset() is called repeatedly.
	if !gs.eventChClosed {
		close(gs.eventCh)
		gs.eventChClosed = true
	}
	gs.eventCh = make(chan gameEvent, 4)
	gs.eventChClosed = false

	gs.tiles = nil
	gs.score = 0
	gs.missed = 0
	gs.over = false
	gs.overReason = ""
	gs.missedKanas = nil
	gs.sessionStats = make(map[string]store.KanaStats)
	gs.currentStreak = make(map[string]int)
	gs.sessionDirty = false
	// An intro dropped by game over is shown before the new game starts; the
	// offer cannot be pending at game over and is already marked as made.
	// The new game stays paused only while that intro is still pending,
	// matching settle()'s invariant (settle itself only ever unpauses, so it
	// cannot express this one case where Reset must pause).
	gs.pendingOffer = false
	gs.userPaused = false
	gs.paused = len(gs.pendingIntro) > 0

	// reload overall stats
	gs.overallStats = make(map[string]store.KanaStats)
	if gs.store != nil {
		if stats, err := gs.store.KanaStatistics(); err == nil {
			for _, stat := range stats {
				copied := stat
				gs.overallStats[stat.Char] = copied
				gs.currentStreak[stat.Char] = stat.Streak
			}
		}
	}

	gs.buildSnapshot()
	gs.mu.Unlock()
}

// Stop halts the background goroutines.
func (gs *GameState) Stop() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	select {
	case <-gs.stopCh:
	default:
		close(gs.stopCh)
	}
}

func (gs *GameState) tickLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	gs.mu.Lock()
	stop := gs.stopCh
	gs.mu.Unlock()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			gs.tick()
		}
	}
}

func (gs *GameState) spawnLoop() {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	gs.mu.Lock()
	stop := gs.stopCh
	gs.mu.Unlock()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			gs.spawnKana()
		}
	}
}

func (gs *GameState) tick() {
	gs.mu.Lock()
	if gs.over || gs.paused {
		gs.mu.Unlock()
		return
	}

	for i := len(gs.tiles) - 1; i >= 0; i-- {
		tile := gs.tiles[i]
		tile.Move(fyne.NewPos(tile.pos.X, tile.pos.Y+tile.kana.Speed))

		if tile.pos.Y > gs.canvasH {
			gs.recordMiss(tile.kana.Char)
			gs.missedKanas = append(gs.missedKanas, tile.kana)
			gs.tiles = append(gs.tiles[:i], gs.tiles[i+1:]...)
			gs.missed++
			gs.checkMissedLimit()
		}
	}

	gs.buildSnapshot()
	canvas := gs.canvas
	gs.mu.Unlock()

	if canvas != nil {
		fyne.Do(func() { canvas.Refresh() })
	}
}

func (gs *GameState) spawnKana() {
	gs.mu.Lock()
	if gs.over || gs.paused {
		gs.mu.Unlock()
		return
	}

	chars := gs.availableCharacters()
	if len(chars) == 0 {
		gs.mu.Unlock()
		return
	}
	char := chars[rand.Intn(len(chars))]
	romaji, _ := gs.charSet.GetRomaji(char)

	speed := 3.75 + rand.Float32()*2.5
	kana := kanacore.Kana{
		Char:   char,
		Romaji: romaji,
		Speed:  speed,
	}
	tile := newKanaTile(kana)

	maxX := gs.canvasW - tile.Width()
	if maxX < 0 {
		maxX = 0
	}
	x := rand.Float32() * maxX
	tile.Move(fyne.NewPos(x, 0))
	gs.tiles = append(gs.tiles, tile)

	gs.buildSnapshot()
	canvas := gs.canvas
	gs.mu.Unlock()

	if canvas != nil {
		fyne.Do(func() { canvas.Refresh() })
	}
}

// checkAnswer removes the lowest tile whose kana matches input (canonical or
// alternative romaji). Acquires the lock itself; releases before triggering
// canvas.Refresh() so the UI updates immediately on correct answers.
func (gs *GameState) checkAnswer(input string) {
	gs.mu.Lock()
	if gs.paused || gs.over {
		gs.mu.Unlock()
		return
	}
	best := -1
	for i, tile := range gs.tiles {
		if !gs.charSet.Matches(tile.kana.Char, input) {
			continue
		}
		if best < 0 || tile.pos.Y > gs.tiles[best].pos.Y {
			best = i
		}
	}
	matched := best >= 0
	if matched {
		tile := gs.tiles[best]
		gs.tiles = append(gs.tiles[:best], gs.tiles[best+1:]...)
		gs.score += 10
		announced := gs.recordCorrect(tile.kana.Char)
		if gs.scoreLimit > 0 && gs.score >= gs.scoreLimit {
			gs.endGame("score")
		}
		if !announced {
			gs.maybeOfferKatakana()
		}
		gs.buildSnapshot()
	}
	canvas := gs.canvas
	gs.mu.Unlock()

	if matched && canvas != nil {
		canvas.Refresh()
	}
}

// checkMissedLimit ends the game if misses have reached the threshold.
// Must be called with lock held.
func (gs *GameState) checkMissedLimit() {
	if gs.missed >= 10 && !gs.over {
		gs.endGame("misses")
	}
}

// endGame sets game over flags and attempts to merge session stats. Must be called under lock.
func (gs *GameState) endGame(reason string) {
	if gs.over {
		return
	}
	gs.over = true
	if gs.overReason == "" {
		gs.overReason = reason
	}
	gs.mergeSessionStats()

	select {
	case gs.eventCh <- gameEvent{kind: gameOverEvent}:
	default:
	}
}

// recordCorrect updates session stats only. Must be called under lock.
// Reports whether it announced newly unlocked rows.
func (gs *GameState) recordCorrect(char string) bool {
	streak := gs.currentStreak[char] + 1
	gs.currentStreak[char] = streak

	stat := gs.sessionStats[char]
	stat.Char = char
	stat.CorrectCount++
	stat.Streak = streak
	gs.sessionStats[char] = stat
	gs.sessionDirty = true

	if unlocked := gs.checkAutoProgression(); len(unlocked) > 0 {
		gs.announceRows(unlocked)
		return true
	}
	return false
}

// recordMiss updates session stats + resets streak. Must be called under lock.
func (gs *GameState) recordMiss(char string) {
	gs.currentStreak[char] = 0

	stat := gs.sessionStats[char]
	stat.Char = char
	stat.MissCount++
	stat.Streak = 0
	gs.sessionStats[char] = stat
	gs.sessionDirty = true
}

// mergeSessionStats writes (baseline + session) to store. Must be called under lock.
func (gs *GameState) mergeSessionStats() {
	if !gs.sessionDirty {
		return
	}

	var baseline map[string]store.KanaStats
	if gs.store != nil {
		stats, err := gs.store.KanaStatistics()
		if err != nil {
			// Leave sessionDirty=true so next merge retries.
			return
		}
		baseline = stats
	}
	if baseline == nil {
		baseline = make(map[string]store.KanaStats)
	}

	for char, session := range gs.sessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 && session.Streak == 0 && gs.currentStreak[char] == 0 {
			continue
		}
		base := baseline[char]
		base.Char = char
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		base.Streak = gs.currentStreak[char]
		if gs.store != nil {
			if err := gs.store.SaveKanaStats(char, base.CorrectCount, base.MissCount, base.Streak); err != nil {
				continue
			}
		}
		gs.overallStats[char] = base
		delete(gs.sessionStats, char)
	}

	gs.sessionDirty = false
}

// availableCharacters returns the characters of selected rows whose script is
// active, falling back to the basic rows of the first active script. Must be
// called under lock.
func (gs *GameState) availableCharacters() []string {
	var chars []string
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] && gs.activeScripts[row.Script] {
			chars = append(chars, row.Characters()...)
		}
	}
	if len(chars) > 0 {
		return chars
	}
	for _, info := range kanacore.Scripts() {
		if !gs.activeScripts[info.Script] {
			continue
		}
		for _, row := range kanacore.RowsInGroup(info.Script, kanacore.GroupBasic) {
			chars = append(chars, row.Characters()...)
		}
		return chars
	}
	return nil
}

// applySelectedRows replaces the selection map.
func (gs *GameState) applySelectedRows(rows []string) {
	if gs.selectedRows == nil {
		gs.selectedRows = make(map[string]bool)
	}
	for k := range gs.selectedRows {
		delete(gs.selectedRows, k)
	}
	for _, id := range rows {
		gs.selectedRows[id] = true
	}
}

// setSelectedRowsLocked applies a new row selection, persists it and prunes
// any pending intro of rows no longer selected. Must be called under lock.
func (gs *GameState) setSelectedRowsLocked(rows []string) {
	gs.applySelectedRows(rows)
	gs.saveSelectedRows()
	gs.prunePendingIntro()
}

// SetSelectedRows updates selection and persists to store.
func (gs *GameState) SetSelectedRows(rows []string) {
	gs.mu.Lock()
	gs.setSelectedRowsLocked(rows)
	gs.mu.Unlock()
}

// applySettings applies a saved settings dialog as one locked operation:
// rows first, then scripts, so the katakana intro is decided against the new
// selection. Every active script ends up with at least its first step.
func (gs *GameState) applySettings(rows []string, scripts []kanacore.Script, auto bool, limit int) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	katakanaWasOn := gs.activeScripts[kanacore.ScriptKatakana]
	gs.setSelectedRowsLocked(rows)
	gs.activeScripts = scriptSet(scripts)
	gs.saveActiveScripts()
	if !katakanaWasOn && gs.activeScripts[kanacore.ScriptKatakana] {
		gs.enableKatakanaLocked()
	}
	for _, info := range kanacore.Scripts() {
		if gs.activeScripts[info.Script] && !gs.hasSelectedRowOf(info.Script) {
			for _, id := range kanacore.ProgressionStepsFor(info.Script)[0] {
				gs.selectedRows[id] = true
			}
			gs.saveSelectedRows()
		}
	}

	if limit < 0 {
		limit = 0
	}
	gs.autoProgress = auto
	gs.scoreLimit = limit
	if gs.store != nil {
		_ = gs.store.SaveAutoProgress(auto)
		_ = gs.store.SaveScoreLimit(limit)
	}

	gs.dropInactiveTiles()
	gs.buildSnapshot()
	gs.settle()
}

// dropInactiveTiles removes tiles whose row is deselected or whose script is
// inactive. Must be called under lock.
func (gs *GameState) dropInactiveTiles() {
	kept := gs.tiles[:0]
	for _, t := range gs.tiles {
		rowID, ok := kanacore.CharToRow[t.kana.Char]
		if ok && (!gs.selectedRows[rowID] || !gs.activeScripts[kanacore.ScriptOf(rowID)]) {
			continue
		}
		kept = append(kept, t)
	}
	gs.tiles = kept
}

// SetAutoProgress toggles auto-progression and persists.
func (gs *GameState) SetAutoProgress(enabled bool) {
	gs.mu.Lock()
	gs.autoProgress = enabled
	st := gs.store
	gs.mu.Unlock()
	if st != nil {
		_ = st.SaveAutoProgress(enabled)
	}
}

// SetScoreLimit sets and persists the score limit.
func (gs *GameState) SetScoreLimit(limit int) {
	if limit < 0 {
		limit = 0
	}
	gs.mu.Lock()
	gs.scoreLimit = limit
	st := gs.store
	gs.mu.Unlock()
	if st != nil {
		_ = st.SaveScoreLimit(limit)
	}
}

// checkAutoProgression unlocks, for each active script, the rest of its next
// progression step once all selected rows of that script are mastered. Must be
// called under lock. Returns the IDs unlocked, hiragana first.
func (gs *GameState) checkAutoProgression() []string {
	if !gs.autoProgress {
		return nil
	}
	var unlocked []string
	for _, info := range kanacore.Scripts() {
		s := info.Script
		if !gs.activeScripts[s] {
			continue
		}
		next := gs.nextStepRows(s)
		if len(next) == 0 || !gs.selectedRowsMastered(s) {
			continue
		}
		for _, id := range next {
			gs.selectedRows[id] = true
		}
		unlocked = append(unlocked, next...)
	}
	if len(unlocked) > 0 {
		gs.saveSelectedRows()
	}
	return unlocked
}

// nextStepRows returns the unselected rows of the script's first incomplete
// progression step. Must be called under lock.
func (gs *GameState) nextStepRows(s kanacore.Script) []string {
	for _, step := range kanacore.ProgressionStepsFor(s) {
		var missing []string
		for _, id := range step {
			if !gs.selectedRows[id] {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			return missing
		}
	}
	return nil
}

// selectedRowsMastered reports whether every selected row of the script is
// mastered. Must be called under lock.
func (gs *GameState) selectedRowsMastered(s kanacore.Script) bool {
	for _, row := range kanacore.RowsFor(s) {
		if gs.selectedRows[row.ID] && !gs.isRowMastered(row) {
			return false
		}
	}
	return true
}

// saveSelectedRows persists the selection in row order. Must be called under lock.
func (gs *GameState) saveSelectedRows() {
	if gs.store == nil {
		return
	}
	ids := make([]string, 0, len(gs.selectedRows))
	for _, row := range kanacore.AllKanaRows {
		if gs.selectedRows[row.ID] {
			ids = append(ids, row.ID)
		}
	}
	_ = gs.store.SaveSelectedRows(ids)
}

// prunePendingIntro removes any pending-intro row ID no longer selected,
// clearing pendingIntro to nil if nothing remains. Must be called under lock.
func (gs *GameState) prunePendingIntro() {
	if len(gs.pendingIntro) == 0 {
		return
	}
	pruned := make([]string, 0, len(gs.pendingIntro))
	for _, id := range gs.pendingIntro {
		if gs.selectedRows[id] {
			pruned = append(pruned, id)
		}
	}
	if len(pruned) == 0 {
		gs.pendingIntro = nil
		return
	}
	gs.pendingIntro = pruned
}

// isRowMastered returns true when at least 80% of the row's characters have a
// combined (overall+session) correct count of 3 or more.
func (gs *GameState) isRowMastered(row kanacore.KanaRow) bool {
	chars := row.Characters()
	if len(chars) == 0 {
		return true
	}

	masteredCount := 0
	for _, char := range chars {
		total := gs.overallStats[char].CorrectCount + gs.sessionStats[char].CorrectCount
		if total >= 3 {
			masteredCount++
		}
	}

	threshold := int(float64(len(chars)) * 0.8)
	if threshold == 0 {
		threshold = 1
	}
	return masteredCount >= threshold
}

// announceRows queues an intro for newly unlocked rows and pauses the game
// until it is dismissed. If the event cannot be delivered the game keeps
// running. Must be called under lock.
func (gs *GameState) announceRows(ids []string) {
	rows := append([]string(nil), ids...)
	select {
	case gs.eventCh <- gameEvent{kind: rowsUnlockedEvent, rows: rows}:
		gs.paused = true
		gs.pendingIntro = rows
	default:
	}
}

// settle enforces the pause invariant: the game is paused only while a dialog
// is pending or the learner has paused. It is the only place that unpauses.
// Must be called under lock.
func (gs *GameState) settle() {
	if gs.paused && len(gs.pendingIntro) == 0 && !gs.pendingOffer && !gs.userPaused {
		gs.paused = false
	}
}

// TogglePause flips the learner's pause and returns whether it is now on.
// Turning it off resumes only if no other reason holds the pause. It does
// nothing once the game is over.
func (gs *GameState) TogglePause() bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.over {
		return false
	}
	gs.userPaused = !gs.userPaused
	if gs.userPaused {
		gs.paused = true
	} else {
		gs.settle()
	}
	gs.buildSnapshot()
	return gs.userPaused
}

// IsUserPaused reports whether the learner has paused the game.
func (gs *GameState) IsUserPaused() bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.userPaused
}

// maybeOfferKatakana raises the one-time katakana offer once all hiragana
// basic rows are mastered. It never fires after game over, while katakana is
// active, or once already offered. Must be called under lock.
func (gs *GameState) maybeOfferKatakana() {
	if gs.over || gs.katakanaOffered || !gs.activeScripts[kanacore.ScriptHiragana] || gs.activeScripts[kanacore.ScriptKatakana] {
		return
	}
	for _, row := range kanacore.BasicRows() {
		if !gs.selectedRows[row.ID] || !gs.isRowMastered(row) {
			return
		}
	}
	select {
	case gs.eventCh <- gameEvent{kind: katakanaOfferEvent}:
		gs.paused = true
		gs.pendingOffer = true
		gs.katakanaOffered = true
		if gs.store != nil {
			_ = gs.store.SaveKatakanaOffered(true)
		}
	default:
	}
}

// enableKatakanaLocked activates katakana and marks the offer as made. If no
// katakana row is selected yet, it selects the first katakana step and queues
// its intro. Must be called under lock.
func (gs *GameState) enableKatakanaLocked() {
	gs.activeScripts[kanacore.ScriptKatakana] = true
	gs.saveActiveScripts()
	gs.katakanaOffered = true
	if gs.store != nil {
		_ = gs.store.SaveKatakanaOffered(true)
	}
	if gs.hasSelectedRowOf(kanacore.ScriptKatakana) {
		return
	}
	first := kanacore.ProgressionStepsFor(kanacore.ScriptKatakana)[0]
	for _, id := range first {
		gs.selectedRows[id] = true
	}
	gs.saveSelectedRows()
	gs.pendingIntro = append(gs.pendingIntro, first...)
	gs.paused = true
}

// EnableKatakana accepts the offer.
func (gs *GameState) EnableKatakana() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.pendingOffer = false
	gs.enableKatakanaLocked()
	gs.settle()
}

// DeclineKatakana dismisses the offer.
func (gs *GameState) DeclineKatakana() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.pendingOffer = false
	gs.settle()
}

// PendingDialogs returns whether the offer is pending and a copy of the rows
// waiting to be introduced.
func (gs *GameState) PendingDialogs() (offer bool, intro []string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.pendingOffer, append([]string(nil), gs.pendingIntro...)
}

// IsOver reports whether the current game has ended.
func (gs *GameState) IsOver() bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.over
}

// Resume continues a paused game unless a dialog is still pending.
func (gs *GameState) Resume() {
	gs.mu.Lock()
	gs.settle()
	gs.mu.Unlock()
}

// FinishIntro clears the pending intro and resumes the game.
func (gs *GameState) FinishIntro() {
	gs.mu.Lock()
	gs.pendingIntro = nil
	gs.settle()
	gs.mu.Unlock()
}

// buildSnapshot rebuilds the atomic snapshot of canvas objects. While the
// learner has paused, no tiles are published, so the pause cannot be used to
// study them, and the canvas is told to show the hint.
// Must be called under lock.
func (gs *GameState) buildSnapshot() {
	hide := gs.userPaused
	gs.showPauseHint.Store(gs.userPaused)
	if hide {
		gs.objectSnapshot.Store([]fyne.CanvasObject{})
		return
	}
	objs := make([]fyne.CanvasObject, 0, len(gs.tiles)*3)
	for _, tile := range gs.tiles {
		objs = append(objs, tile.Objects()...)
	}
	gs.objectSnapshot.Store(objs)
}

// snapshot builds a StatsSnapshot. Caller MUST hold gs.mu.
func (gs *GameState) snapshot() StatsSnapshot {
	sessionCopy := make(map[string]store.KanaStats, len(gs.sessionStats))
	for k, v := range gs.sessionStats {
		sessionCopy[k] = v
	}
	rowsCopy := make(map[string]bool, len(gs.selectedRows))
	for k, v := range gs.selectedRows {
		rowsCopy[k] = v
	}
	scriptsCopy := make(map[kanacore.Script]bool, len(gs.activeScripts))
	for k, v := range gs.activeScripts {
		scriptsCopy[k] = v
	}
	return StatsSnapshot{
		SessionStats:  sessionCopy,
		SelectedRows:  rowsCopy,
		MissedKanas:   append([]kanacore.Kana{}, gs.missedKanas...),
		Score:         gs.score,
		ScoreLimit:    gs.scoreLimit,
		Missed:        gs.missed,
		ActiveScripts: scriptsCopy,
	}
}
