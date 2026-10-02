package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"kana/kanacore"
	"kana/store"
)

func newTestState() *GameState {
	test.NewApp()
	gs := &GameState{
		sessionStats:  make(map[string]store.KanaStats),
		overallStats:  make(map[string]store.KanaStats),
		currentStreak: make(map[string]int),
		selectedRows:  make(map[string]bool),
		activeScripts: map[kanacore.Script]bool{kanacore.ScriptHiragana: true},
		eventCh:       make(chan gameEvent, 4),
		stopCh:        make(chan struct{}),
		canvasW:       400,
		canvasH:       600,
	}
	gs.charSet = kanacore.AllKana()
	for _, id := range kanacore.DefaultRowIDs() {
		gs.selectedRows[id] = true
	}
	return gs
}

func TestCheckAnswerRemovesTile(t *testing.T) {
	gs := newTestState()
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("ka")
	if len(gs.tiles) != 0 {
		t.Errorf("expected tile removed, got %d tiles", len(gs.tiles))
	}
	if gs.score != 10 {
		t.Errorf("expected score 10, got %d", gs.score)
	}
}

func TestCheckAnswerIgnoredAfterGameOver(t *testing.T) {
	gs := newTestState()
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.over = true
	gs.checkAnswer("ka")
	if len(gs.tiles) != 1 {
		t.Fatalf("expected tile to remain after game over, got %d tiles", len(gs.tiles))
	}
	if gs.score != 0 {
		t.Fatalf("expected score unchanged after game over, got %d", gs.score)
	}
}

func TestIsOver(t *testing.T) {
	gs := newTestState()
	if gs.IsOver() {
		t.Fatalf("expected new test state to not be over")
	}
	gs.over = true
	if !gs.IsOver() {
		t.Fatalf("expected IsOver to report true after gs.over = true")
	}
}

func TestCheckAnswerNoMatchLeavestTile(t *testing.T) {
	gs := newTestState()
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("ki")
	if len(gs.tiles) != 1 {
		t.Errorf("expected tile to remain, got %d tiles", len(gs.tiles))
	}
}

func TestRecordCorrectUpdatesSessionOnly(t *testing.T) {
	gs := newTestState()
	gs.recordCorrect("か")
	if gs.sessionStats["か"].CorrectCount != 1 {
		t.Errorf("session correct count: got %d, want 1", gs.sessionStats["か"].CorrectCount)
	}
	// overallStats must NOT be touched by recordCorrect (see spec: stats double-counting fix)
	if gs.overallStats["か"].CorrectCount != 0 {
		t.Errorf("overallStats should not be updated by recordCorrect, got %d",
			gs.overallStats["か"].CorrectCount)
	}
}

func TestRecordMissResetsStreak(t *testing.T) {
	gs := newTestState()
	gs.currentStreak["か"] = 5
	gs.recordMiss("か")
	if gs.currentStreak["か"] != 0 {
		t.Errorf("expected streak 0, got %d", gs.currentStreak["か"])
	}
	if gs.sessionStats["か"].MissCount != 1 {
		t.Errorf("expected miss count 1, got %d", gs.sessionStats["か"].MissCount)
	}
}

func TestScoreLimitEndsGame(t *testing.T) {
	gs := newTestState()
	gs.scoreLimit = 10
	tile := newKanaTile(kanacore.Kana{Char: "か", Romaji: "ka"})
	gs.tiles = []*KanaTile{tile}
	gs.checkAnswer("ka")
	if !gs.over {
		t.Error("expected game over when score limit reached")
	}
	if gs.overReason != "score" {
		t.Errorf("expected reason 'score', got %q", gs.overReason)
	}
}

func TestMissLimitEndsGame(t *testing.T) {
	gs := newTestState()
	for i := 0; i < 9; i++ {
		gs.recordMiss("か")
		gs.missed++
	}
	gs.recordMiss("あ")
	gs.missed++
	gs.checkMissedLimit()
	if !gs.over {
		t.Error("expected game over at 10 misses")
	}
}

func tileAt(char, romaji string, y float32) *KanaTile {
	tile := newKanaTile(kanacore.Kana{Char: char, Romaji: romaji})
	tile.fallSeconds = 20
	tile.Move(fyne.NewPos(10, y))
	return tile
}

func TestCheckAnswerAcceptsAlternative(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("し", "shi", 0)}
	gs.checkAnswer("si")
	if len(gs.tiles) != 0 {
		t.Fatalf("expected し removed by 'si', %d tiles left", len(gs.tiles))
	}
}

func TestCheckAnswerNormalisesInput(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("か", "ka", 0)}
	gs.checkAnswer("  KA ")
	if len(gs.tiles) != 0 {
		t.Fatalf("expected か removed by '  KA ', %d tiles left", len(gs.tiles))
	}
}

func TestCheckAnswerRemovesLowestMatchingTile(t *testing.T) {
	gs := newTestState()
	high := tileAt("じ", "ji", 100)
	low := tileAt("ぢ", "ji", 300)
	gs.tiles = []*KanaTile{high, low}
	gs.checkAnswer("ji")
	if len(gs.tiles) != 1 || gs.tiles[0] != high {
		t.Fatalf("expected only the upper じ tile to remain")
	}
}

func TestSpawnKeepsWideTilesInsideCanvas(t *testing.T) {
	gs := newTestState()
	gs.selectedRows = map[string]bool{"ky": true}
	gs.canvasW = 100
	for i := 0; i < 50; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if tile.pos.X < 0 || tile.pos.X+tile.Width() > gs.canvasW {
			t.Fatalf("tile %s at x=%.1f width %.1f exceeds canvas width %.1f",
				tile.kana.Char, tile.pos.X, tile.Width(), gs.canvasW)
		}
	}
}

func TestIsRowMastered(t *testing.T) {
	gs := newTestState()
	row := kanacore.AllKanaRows[0] // vowels
	// Give 3 correct answers to 4 out of 5 characters (80%)
	for _, char := range row.Characters()[:4] {
		gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
	}
	if !gs.isRowMastered(row) {
		t.Error("expected row to be mastered at 80% threshold")
	}
}

func masterRows(gs *GameState, ids ...string) {
	for _, id := range ids {
		row, _ := kanacore.RowByID(id)
		for _, char := range row.Characters() {
			gs.overallStats[char] = store.KanaStats{Char: char, CorrectCount: 3}
		}
	}
}

func selectRows(gs *GameState, ids ...string) {
	gs.selectedRows = make(map[string]bool)
	for _, id := range ids {
		gs.selectedRows[id] = true
	}
}

func equalIDs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestProgressionCrossesIntoDakuon(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	basic := kanacore.DefaultRowIDs()
	selectRows(gs, basic...)
	masterRows(gs, basic...)
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"g", "z"}) {
		t.Fatalf("unlocked %v, want [g z]", got)
	}
}

func TestProgressionAfterHandakuonUnlocksYoon(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	ids := append(kanacore.DefaultRowIDs(), "g", "z", "d", "b", "p")
	selectRows(gs, ids...)
	masterRows(gs, ids...)
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"ky", "sy"}) {
		t.Fatalf("unlocked %v, want [ky sy]", got)
	}
}

func TestProgressionCompletesPartialStep(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels", "k")
	masterRows(gs, "vowels", "k")
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"s"}) {
		t.Fatalf("unlocked %v, want [s]", got)
	}
}

// newUnlockReadyState returns a state where one correct あ unlocks [k s].
func newUnlockReadyState() *GameState {
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels")
	masterRows(gs, "vowels")
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	return gs
}

func TestUnlockPausesAndQueuesEvent(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a")
	if !gs.paused {
		t.Fatal("expected game paused after unlock")
	}
	if !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("pendingIntro = %v, want [k s]", gs.pendingIntro)
	}
	select {
	case ev := <-gs.eventCh:
		if ev.kind != rowsUnlockedEvent || !equalIDs(ev.rows, []string{"k", "s"}) {
			t.Fatalf("unexpected event %+v", ev)
		}
	default:
		t.Fatal("expected rowsUnlockedEvent on eventCh")
	}
}

func TestUnlockWithFullEventChannelDoesNotPause(t *testing.T) {
	gs := newUnlockReadyState()
	for i := 0; i < cap(gs.eventCh); i++ {
		gs.eventCh <- gameEvent{kind: gameOverEvent}
	}
	gs.checkAnswer("a")
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("expected no pause without a delivered event (paused=%v, pending=%v)", gs.paused, gs.pendingIntro)
	}
	if !gs.selectedRows["k"] || !gs.selectedRows["s"] {
		t.Fatal("rows should still be unlocked")
	}
}

func TestPausedGameDoesNotMoveSpawnOrScore(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.paused = true

	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 0 {
		t.Fatalf("tile moved while paused: y=%.1f", y)
	}
	gs.spawnKana()
	if len(gs.tiles) != 1 {
		t.Fatalf("spawned while paused: %d tiles", len(gs.tiles))
	}
	gs.checkAnswer("a")
	if len(gs.tiles) != 1 || gs.score != 0 {
		t.Fatal("answer accepted while paused")
	}

	gs.Resume()
	gs.tick()
	if y := gs.tiles[0].pos.Y; y == 0 {
		t.Fatal("tile did not move after Resume")
	}
}

func TestResumeRespectsPauseInvariant(t *testing.T) {
	gs := newTestState()
	gs.pendingOffer = true
	gs.paused = true
	gs.Resume()
	if !gs.paused {
		t.Fatal("Resume must leave the game paused while the offer is pending")
	}

	gs.pendingOffer = false
	gs.Resume()
	if gs.paused {
		t.Fatal("Resume should unpause once nothing is pending")
	}
}

func TestFinishIntroClearsPendingAndResumes(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a")
	gs.FinishIntro()
	if gs.paused || gs.pendingIntro != nil {
		t.Fatalf("after FinishIntro paused=%v pending=%v", gs.paused, gs.pendingIntro)
	}
}

func TestUnlockAndScoreLimitKeepIntroForPlayAgain(t *testing.T) {
	gs := newUnlockReadyState()
	gs.scoreLimit = 10
	gs.checkAnswer("a")
	if !gs.over {
		t.Fatal("expected game over at score limit")
	}
	if !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("pendingIntro = %v, want [k s]", gs.pendingIntro)
	}
	first := <-gs.eventCh
	second := <-gs.eventCh
	if first.kind != rowsUnlockedEvent || second.kind != gameOverEvent {
		t.Fatalf("event order = %v, %v", first.kind, second.kind)
	}

	gs.Reset()
	if !gs.paused || !equalIDs(gs.pendingIntro, []string{"k", "s"}) {
		t.Fatalf("after Reset paused=%v pending=%v, want paused with [k s]", gs.paused, gs.pendingIntro)
	}
}

func TestResetWithoutPendingIntroUnpauses(t *testing.T) {
	gs := newTestState()
	gs.paused = true
	gs.Reset()
	if gs.paused {
		t.Fatal("expected Reset to clear pause without a pending intro")
	}
}

func TestPrunePendingIntroDropsDeselectedRows(t *testing.T) {
	gs := newTestState()
	gs.pendingIntro = []string{"k", "s"}
	selectRows(gs, "vowels", "k")
	gs.prunePendingIntro()
	if !equalIDs(gs.pendingIntro, []string{"k"}) {
		t.Fatalf("pendingIntro = %v, want [k]", gs.pendingIntro)
	}
}

func TestPrunePendingIntroClearsWhenAllDeselected(t *testing.T) {
	gs := newTestState()
	gs.pendingIntro = []string{"k", "s"}
	selectRows(gs, "vowels")
	gs.prunePendingIntro()
	if gs.pendingIntro != nil {
		t.Fatalf("pendingIntro = %v, want nil", gs.pendingIntro)
	}
}

func scriptOfChar(char string) kanacore.Script {
	return kanacore.ScriptOf(kanacore.CharToRow[char])
}

func TestInactiveScriptDoesNotSpawn(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels", "kata:vowels")
	for i := 0; i < 40; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if scriptOfChar(tile.kana.Char) != kanacore.ScriptHiragana {
			t.Fatalf("katakana %s spawned while katakana is inactive", tile.kana.Char)
		}
	}
}

func TestActiveKatakanaSpawns(t *testing.T) {
	gs := newTestState()
	gs.activeScripts[kanacore.ScriptKatakana] = true
	selectRows(gs, "kata:vowels")
	for i := 0; i < 10; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if scriptOfChar(tile.kana.Char) != kanacore.ScriptKatakana {
			t.Fatalf("%s spawned, want only katakana", tile.kana.Char)
		}
	}
}

func TestSpawnFallbackUsesActiveScriptBasics(t *testing.T) {
	gs := newTestState()
	gs.activeScripts = map[kanacore.Script]bool{kanacore.ScriptKatakana: true}
	selectRows(gs)
	gs.spawnKana()
	if len(gs.tiles) != 1 {
		t.Fatalf("expected a fallback tile, got %d", len(gs.tiles))
	}
	row, _ := kanacore.RowByID(kanacore.CharToRow[gs.tiles[0].kana.Char])
	if row.Script != kanacore.ScriptKatakana || row.Group != kanacore.GroupBasic {
		t.Fatalf("fallback spawned %s from %s", gs.tiles[0].kana.Char, row.ID)
	}
}

func TestProgressionIsPerScript(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	gs.activeScripts[kanacore.ScriptKatakana] = true
	selectRows(gs, "vowels", "kata:vowels")
	masterRows(gs, "vowels")
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"k", "s"}) {
		t.Fatalf("unlocked %v, want [k s]", got)
	}
	masterRows(gs, "kata:vowels")
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"kata:k", "kata:s"}) {
		t.Fatalf("unlocked %v, want [kata:k kata:s]", got)
	}
}

func TestBothScriptsUnlockTogether(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	gs.activeScripts[kanacore.ScriptKatakana] = true
	selectRows(gs, "vowels", "kata:vowels")
	masterRows(gs, "vowels", "kata:vowels")
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"k", "s", "kata:k", "kata:s"}) {
		t.Fatalf("unlocked %v, want hiragana first then katakana", got)
	}
}

func TestInactiveScriptNeverUnlocks(t *testing.T) {
	gs := newTestState()
	gs.autoProgress = true
	selectRows(gs, "vowels", "kata:vowels")
	masterRows(gs, "vowels", "kata:vowels")
	if got := gs.checkAutoProgression(); !equalIDs(got, []string{"k", "s"}) {
		t.Fatalf("unlocked %v, want only [k s]", got)
	}
}

func TestSettleResumesWhenNothingPending(t *testing.T) {
	gs := newTestState()
	gs.paused = true
	gs.settle()
	if gs.paused {
		t.Fatal("settle should resume when nothing is pending")
	}
	gs.paused = true
	gs.pendingIntro = []string{"k"}
	gs.settle()
	if !gs.paused {
		t.Fatal("settle must keep the pause while an intro is pending")
	}

	gs.paused = true
	gs.pendingIntro = nil
	gs.pendingOffer = true
	gs.settle()
	if !gs.paused {
		t.Fatal("settle must keep the pause while the offer is pending")
	}
}

// newOfferReadyState returns a hiragana-only state where one correct あ
// fulfils the offer condition without unlocking anything.
func newOfferReadyState() *GameState {
	gs := newTestState()
	basic := kanacore.DefaultRowIDs()
	selectRows(gs, basic...)
	masterRows(gs, basic...)
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	return gs
}

func TestOfferFiresOnceBasicsAreMastered(t *testing.T) {
	gs := newOfferReadyState()
	gs.checkAnswer("a")
	if !gs.pendingOffer || !gs.paused || !gs.katakanaOffered {
		t.Fatalf("pendingOffer=%v paused=%v offered=%v", gs.pendingOffer, gs.paused, gs.katakanaOffered)
	}
	select {
	case ev := <-gs.eventCh:
		if ev.kind != katakanaOfferEvent {
			t.Fatalf("event kind = %v", ev.kind)
		}
	default:
		t.Fatal("expected katakanaOfferEvent")
	}
}

func TestOfferSkippedWhenAlreadyOffered(t *testing.T) {
	gs := newOfferReadyState()
	gs.katakanaOffered = true
	gs.checkAnswer("a")
	if gs.pendingOffer || gs.paused {
		t.Fatal("offer must not repeat")
	}
	if len(gs.eventCh) != 0 {
		t.Fatalf("eventCh len = %d, want 0", len(gs.eventCh))
	}
}

func TestOfferSkippedWhenKatakanaActive(t *testing.T) {
	gs := newOfferReadyState()
	gs.activeScripts[kanacore.ScriptKatakana] = true
	gs.checkAnswer("a")
	if gs.pendingOffer {
		t.Fatal("no offer while katakana is active")
	}
	if gs.paused {
		t.Fatal("game must not be paused when no offer fires")
	}
	if len(gs.eventCh) != 0 {
		t.Fatalf("eventCh len = %d, want 0", len(gs.eventCh))
	}
}

func TestOfferSkippedOnGameEndingAnswer(t *testing.T) {
	gs := newOfferReadyState()
	gs.scoreLimit = 10
	gs.checkAnswer("a")
	if !gs.over || gs.pendingOffer || gs.katakanaOffered {
		t.Fatalf("over=%v pendingOffer=%v offered=%v", gs.over, gs.pendingOffer, gs.katakanaOffered)
	}
}

func TestOfferWaitsForAnswerWithoutUnlock(t *testing.T) {
	gs := newOfferReadyState()
	gs.autoProgress = true
	gs.checkAnswer("a") // unlocks g+z and announces them
	if gs.pendingOffer {
		t.Fatal("offer must not compete with an intro")
	}
	<-gs.eventCh
	gs.FinishIntro()
	// Auto-progression stays on; g and z are selected but not mastered, so
	// the next answer unlocks nothing and the offer can fire.
	gs.tiles = []*KanaTile{tileAt("い", "i", 0)}
	gs.checkAnswer("i")
	if !gs.pendingOffer {
		t.Fatal("offer should come with the next answer that announces nothing")
	}
}

func TestEnableKatakanaStartsKatakanaWithIntro(t *testing.T) {
	gs := newOfferReadyState()
	gs.checkAnswer("a")
	gs.EnableKatakana()
	if gs.pendingOffer || !gs.activeScripts[kanacore.ScriptKatakana] || !gs.selectedRows["kata:vowels"] {
		t.Fatalf("pendingOffer=%v scripts=%v", gs.pendingOffer, gs.activeScripts)
	}
	if !gs.paused || !equalIDs(gs.pendingIntro, []string{"kata:vowels"}) {
		t.Fatalf("paused=%v pendingIntro=%v, want paused with [kata:vowels]", gs.paused, gs.pendingIntro)
	}
}

func TestEnableKatakanaWithPreselectedRowsResumes(t *testing.T) {
	gs := newOfferReadyState()
	gs.selectedRows["kata:k"] = true
	gs.checkAnswer("a")
	gs.EnableKatakana()
	if gs.paused || gs.pendingIntro != nil || gs.selectedRows["kata:vowels"] {
		t.Fatalf("paused=%v pendingIntro=%v", gs.paused, gs.pendingIntro)
	}
}

func TestDeclineKatakanaResumes(t *testing.T) {
	gs := newOfferReadyState()
	gs.checkAnswer("a")
	gs.DeclineKatakana()
	if gs.paused || gs.pendingOffer || gs.activeScripts[kanacore.ScriptKatakana] {
		t.Fatalf("paused=%v pendingOffer=%v", gs.paused, gs.pendingOffer)
	}
}

func TestResetClearsPendingOffer(t *testing.T) {
	gs := newOfferReadyState()
	gs.checkAnswer("a")
	gs.Reset()
	if gs.pendingOffer || gs.paused {
		t.Fatalf("after Reset pendingOffer=%v paused=%v", gs.pendingOffer, gs.paused)
	}
}

var bothScripts = []kanacore.Script{kanacore.ScriptHiragana, kanacore.ScriptKatakana}

func TestApplySettingsTurningKatakanaOnIntroducesVowels(t *testing.T) {
	gs := newTestState()
	gs.applySettings([]string{"vowels", "k"}, bothScripts, true, 0)
	if !gs.selectedRows["kata:vowels"] || !equalIDs(gs.pendingIntro, []string{"kata:vowels"}) || !gs.paused {
		t.Fatalf("selected=%v pendingIntro=%v paused=%v", gs.selectedRows, gs.pendingIntro, gs.paused)
	}
	if !gs.katakanaOffered {
		t.Fatal("switching katakana on must mark the offer as made")
	}
}

func TestApplySettingsWithPreselectedKatakanaHasNoIntro(t *testing.T) {
	gs := newTestState()
	gs.applySettings([]string{"vowels", "kata:k"}, bothScripts, true, 0)
	if gs.paused || gs.pendingIntro != nil || gs.selectedRows["kata:vowels"] {
		t.Fatalf("paused=%v pendingIntro=%v", gs.paused, gs.pendingIntro)
	}
}

func TestApplySettingsEmptyScriptsKeepsHiragana(t *testing.T) {
	gs := newTestState()
	gs.applySettings([]string{"vowels"}, nil, false, 0)
	if len(gs.activeScripts) != 1 || !gs.activeScripts[kanacore.ScriptHiragana] {
		t.Fatalf("activeScripts = %v", gs.activeScripts)
	}
}

func TestApplySettingsKatakanaOffDropsTilesKeepsRows(t *testing.T) {
	gs := newTestState()
	gs.activeScripts[kanacore.ScriptKatakana] = true
	gs.tiles = []*KanaTile{tileAt("カ", "ka", 0), tileAt("か", "ka", 0)}
	gs.applySettings([]string{"k", "kata:k"}, []kanacore.Script{kanacore.ScriptHiragana}, false, 0)
	if len(gs.tiles) != 1 || gs.tiles[0].kana.Char != "か" {
		t.Fatalf("expected only か to remain, got %d tiles", len(gs.tiles))
	}
	if !gs.selectedRows["kata:k"] {
		t.Fatal("katakana rows must stay selected while katakana is off")
	}
}

func TestApplySettingsRefillsEmptyActiveScript(t *testing.T) {
	gs := newTestState()
	gs.applySettings(nil, bothScripts, false, 0)
	if !gs.selectedRows["vowels"] || !gs.selectedRows["kata:vowels"] {
		t.Fatalf("selected = %v, want vowels and kata:vowels", gs.selectedRows)
	}
}

func TestFallStep(t *testing.T) {
	cases := []struct{ h, secs, want float32 }{
		{600, 20, 3},
		{300, 20, 1.5},
		{600, 0, 600 / (minFallSeconds * ticksPerSecond)},
	}
	for _, c := range cases {
		if got := fallStep(c.h, c.secs); got != c.want {
			t.Errorf("fallStep(%v, %v) = %v, want %v", c.h, c.secs, got, c.want)
		}
	}
}

func TestTickMovesByFallTime(t *testing.T) {
	gs := newTestState()
	gs.canvasH = 600
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 3 {
		t.Fatalf("y after one tick = %v, want 3", y)
	}
	gs.canvasH = 300
	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 4.5 {
		t.Fatalf("y after a tick at half height = %v, want 4.5", y)
	}
}

func TestSpawnSetsFallTimeInRange(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	for i := 0; i < 50; i++ {
		gs.spawnKana()
	}
	for _, tile := range gs.tiles {
		if tile.fallSeconds < minFallSeconds || tile.fallSeconds > maxFallSeconds {
			t.Fatalf("fallSeconds = %v, want [%v, %v]", tile.fallSeconds, minFallSeconds, maxFallSeconds)
		}
	}
}

func TestMissCallsStatsHookOnce(t *testing.T) {
	gs := newTestState() // calls test.NewApp(), so fyne.Do runs the hook
	calls := 0
	gs.onStatsChanged = func() { calls++ }
	gs.canvasH = 600
	gs.tiles = []*KanaTile{tileAt("あ", "a", 599), tileAt("い", "i", 0)}

	gs.tick() // あ falls out, い moves
	if calls != 1 {
		t.Fatalf("hook calls after a miss = %d, want 1", calls)
	}
	gs.tick() // nothing falls out
	if calls != 1 {
		t.Fatalf("hook calls after a tick without a miss = %d, want 1", calls)
	}
}
