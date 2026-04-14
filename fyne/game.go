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

// GameCanvas is a stub; replaced in Task 6.
type GameCanvas struct{}

// Refresh is a stub; replaced in Task 6.
func (*GameCanvas) Refresh() {}

// StatsPanel is a stub; replaced in Task 7.
type StatsPanel struct{}

type gameEventType int

const (
	gameOverEvent gameEventType = iota
)

type gameEvent struct {
	kind gameEventType
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
	newlyUnlocked []string
	unlockMessage string
	unlockAt      time.Time

	store *store.Store

	stopCh  chan struct{}
	eventCh chan gameEvent

	objectSnapshot atomic.Value // []fyne.CanvasObject

	canvasW float32
	canvasH float32

	charSet kanacore.CharacterSet

	// stub, replaced in Task 6
	canvas *GameCanvas
	// stub, replaced in Task 7
	statsPanel *StatsPanel
}

// NewGameState constructs a new GameState, loading persisted state if store is non-nil.
func NewGameState(st *store.Store) *GameState {
	gs := &GameState{
		sessionStats:  make(map[string]store.KanaStats),
		overallStats:  make(map[string]store.KanaStats),
		currentStreak: make(map[string]int),
		selectedRows:  make(map[string]bool),
		eventCh:       make(chan gameEvent, 4),
		stopCh:        make(chan struct{}),
		scoreLimit:    store.DefaultScoreLimit,
		charSet:       kanacore.Hiragana(),
		store:         st,
		canvasW:       400,
		canvasH:       600,
	}

	for _, id := range kanacore.DefaultRowIDs() {
		gs.selectedRows[id] = true
	}

	if st != nil {
		if rows, err := st.SelectedRows(); err == nil && len(rows) > 0 {
			gs.applySelectedRows(rows)
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
		if stats, err := st.KanaStatistics(); err == nil {
			for _, stat := range stats {
				copied := stat
				gs.overallStats[stat.Char] = copied
				gs.currentStreak[stat.Char] = stat.Streak
			}
		}
	}

	return gs
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
	func() {
		defer func() { _ = recover() }()
		close(gs.eventCh)
	}()
	gs.eventCh = make(chan gameEvent, 4)

	gs.tiles = nil
	gs.score = 0
	gs.missed = 0
	gs.over = false
	gs.overReason = ""
	gs.missedKanas = nil
	gs.sessionStats = make(map[string]store.KanaStats)
	gs.currentStreak = make(map[string]int)
	gs.sessionDirty = false
	gs.newlyUnlocked = nil
	gs.unlockMessage = ""
	gs.unlockAt = time.Time{}

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
	if gs.over {
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
		canvas.Refresh()
	}
}

func (gs *GameState) spawnKana() {
	gs.mu.Lock()
	if gs.over {
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
	maxX := gs.canvasW - tileW
	if maxX < 0 {
		maxX = 0
	}
	x := rand.Float32() * maxX

	kana := kanacore.Kana{
		Char:   char,
		Romaji: romaji,
		Speed:  speed,
	}
	tile := newKanaTile(kana)
	tile.Move(fyne.NewPos(x, 0))
	gs.tiles = append(gs.tiles, tile)

	gs.buildSnapshot()
	canvas := gs.canvas
	gs.mu.Unlock()

	if canvas != nil {
		canvas.Refresh()
	}
}

// checkAnswer processes an input string and removes a matching tile if found.
func (gs *GameState) checkAnswer(input string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	for i, tile := range gs.tiles {
		if tile.kana.Romaji == input {
			gs.tiles = append(gs.tiles[:i], gs.tiles[i+1:]...)
			gs.score += 10
			gs.recordCorrect(tile.kana.Char)
			if gs.scoreLimit > 0 && gs.score >= gs.scoreLimit {
				gs.endGame("score")
			}
			gs.buildSnapshot()
			return
		}
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
func (gs *GameState) recordCorrect(char string) {
	streak := gs.currentStreak[char] + 1
	gs.currentStreak[char] = streak

	stat := gs.sessionStats[char]
	stat.Char = char
	stat.CorrectCount++
	stat.Streak = streak
	gs.sessionStats[char] = stat
	gs.sessionDirty = true

	if unlocked := gs.checkAutoProgression(); len(unlocked) > 0 {
		gs.newlyUnlocked = append(gs.newlyUnlocked, unlocked...)
		gs.showUnlockMessage(unlocked)
	}
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

// availableCharacters returns characters filtered by selectedRows. Must be called under lock.
func (gs *GameState) availableCharacters() []string {
	chars := gs.charSet.GetCharacters()
	if len(chars) == 0 {
		return nil
	}
	if len(gs.selectedRows) == 0 {
		return chars
	}

	filtered := make([]string, 0, len(chars))
	for _, char := range chars {
		rowID, ok := kanacore.CharToRow[char]
		if !ok {
			filtered = append(filtered, char)
			continue
		}
		if gs.selectedRows[rowID] {
			filtered = append(filtered, char)
		}
	}
	if len(filtered) == 0 {
		return chars
	}
	return filtered
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

// SetSelectedRows updates selection and persists to store.
func (gs *GameState) SetSelectedRows(rows []string) {
	gs.mu.Lock()
	gs.applySelectedRows(rows)
	st := gs.store
	gs.mu.Unlock()
	if st != nil {
		_ = st.SaveSelectedRows(rows)
	}
}

// SelectedRowIDs returns the currently selected row IDs.
func (gs *GameState) SelectedRowIDs() []string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if len(gs.selectedRows) == 0 {
		return nil
	}
	rows := make([]string, 0, len(gs.selectedRows))
	for id, ok := range gs.selectedRows {
		if ok {
			rows = append(rows, id)
		}
	}
	return rows
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

// checkAutoProgression unlocks the next row when selected rows are all mastered.
// Must be called under lock. Returns the IDs unlocked.
func (gs *GameState) checkAutoProgression() []string {
	if !gs.autoProgress {
		return nil
	}

	var nextRow *kanacore.KanaRow
	for i := range kanacore.AllKanaRows {
		row := kanacore.AllKanaRows[i]
		if !gs.selectedRows[row.ID] {
			nextRow = &row
			break
		}
	}
	if nextRow == nil {
		return nil
	}

	allMastered := true
	for _, row := range kanacore.AllKanaRows {
		if !gs.selectedRows[row.ID] {
			continue
		}
		if !gs.isRowMastered(row) {
			allMastered = false
			break
		}
	}

	if allMastered {
		gs.selectedRows[nextRow.ID] = true
		if gs.store != nil {
			ids := make([]string, 0, len(gs.selectedRows))
			for id, ok := range gs.selectedRows {
				if ok {
					ids = append(ids, id)
				}
			}
			_ = gs.store.SaveSelectedRows(ids)
		}
		return []string{nextRow.ID}
	}

	return nil
}

// isRowMastered returns true when at least 80% of the row's characters have a
// combined (overall+session) correct count of 3 or more.
func (gs *GameState) isRowMastered(row kanacore.KanaRow) bool {
	if len(row.Characters) == 0 {
		return true
	}

	masteredCount := 0
	for _, char := range row.Characters {
		total := gs.overallStats[char].CorrectCount + gs.sessionStats[char].CorrectCount
		if total >= 3 {
			masteredCount++
		}
	}

	threshold := int(float64(len(row.Characters)) * 0.8)
	if threshold == 0 {
		threshold = 1
	}
	return masteredCount >= threshold
}

// showUnlockMessage composes an "unlocked" notification. Must be called under lock.
func (gs *GameState) showUnlockMessage(rowIDs []string) {
	if len(rowIDs) == 0 {
		return
	}

	labels := make([]string, 0, len(rowIDs))
	for _, id := range rowIDs {
		for _, row := range kanacore.AllKanaRows {
			if row.ID == id {
				labels = append(labels, row.Label)
				break
			}
		}
	}

	if len(labels) == 1 {
		gs.unlockMessage = "New row unlocked: " + labels[0]
	} else if len(labels) > 1 {
		gs.unlockMessage = "New rows unlocked: " + labels[0]
		for i := 1; i < len(labels); i++ {
			gs.unlockMessage += ", " + labels[i]
		}
	}
	gs.unlockAt = time.Now()
}

// buildSnapshot rebuilds the atomic snapshot of canvas objects.
// Must be called under lock.
func (gs *GameState) buildSnapshot() {
	objs := make([]fyne.CanvasObject, 0, len(gs.tiles)*3)
	for _, tile := range gs.tiles {
		objs = append(objs, tile.Objects()...)
	}
	gs.objectSnapshot.Store(objs)
}

// snapshot() and statsPanel integration added in Task 7.
