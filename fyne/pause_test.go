package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// tileObjects returns the published tile snapshot.
func tileObjects(gs *GameState) []fyne.CanvasObject {
	objs, _ := gs.objectSnapshot.Load().([]fyne.CanvasObject)
	return objs
}

func TestTogglePausePausesHidesTilesAndResumes(t *testing.T) {
	gs := newTestState()
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.buildSnapshot()

	if !gs.TogglePause() {
		t.Fatal("first TogglePause should report paused")
	}
	if !gs.paused || !gs.IsUserPaused() {
		t.Fatalf("paused=%v userPaused=%v", gs.paused, gs.IsUserPaused())
	}
	if n := len(tileObjects(gs)); n != 0 {
		t.Fatalf("tiles visible while user-paused: %d objects", n)
	}
	if !gs.showPauseHint.Load() {
		t.Fatal("expected the pause hint while user-paused")
	}

	if gs.TogglePause() {
		t.Fatal("second TogglePause should report not paused")
	}
	if gs.paused || gs.IsUserPaused() {
		t.Fatalf("paused=%v userPaused=%v after second toggle", gs.paused, gs.IsUserPaused())
	}
	if n := len(tileObjects(gs)); n != 3 {
		t.Fatalf("expected the tile's 3 objects after resuming, got %d", n)
	}
	if gs.showPauseHint.Load() {
		t.Fatal("pause hint still set after resuming")
	}
}

func TestUserPauseBlocksTickSpawnAndAnswers(t *testing.T) {
	gs := newTestState()
	selectRows(gs, "vowels")
	gs.tiles = []*KanaTile{tileAt("あ", "a", 0)}
	gs.TogglePause()

	gs.tick()
	if y := gs.tiles[0].pos.Y; y != 0 {
		t.Fatalf("tile moved while user-paused: y=%.1f", y)
	}
	gs.spawnKana()
	if len(gs.tiles) != 1 {
		t.Fatalf("spawned while user-paused: %d tiles", len(gs.tiles))
	}
	gs.checkAnswer("a")
	if len(gs.tiles) != 1 || gs.score != 0 {
		t.Fatal("answer accepted while user-paused")
	}
}

func TestUserPauseOutlivesDialogs(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a") // queues an intro and pauses
	gs.TogglePause()
	gs.FinishIntro()
	if !gs.paused {
		t.Fatal("FinishIntro must not resume a user-paused game")
	}

	gs = newOfferReadyState()
	gs.checkAnswer("a") // raises the offer and pauses
	gs.TogglePause()
	gs.DeclineKatakana()
	if !gs.paused {
		t.Fatal("DeclineKatakana must not resume a user-paused game")
	}

	gs.Resume()
	if !gs.paused {
		t.Fatal("Resume must not lift a user pause")
	}
	gs.TogglePause()
	if gs.paused {
		t.Fatal("toggling off should resume once nothing else holds the pause")
	}
}

func TestTogglePauseOffKeepsPendingDialogPause(t *testing.T) {
	gs := newUnlockReadyState()
	gs.checkAnswer("a") // intro pending
	gs.TogglePause()
	gs.TogglePause()
	if !gs.paused {
		t.Fatal("ending the user pause must not resume while an intro is pending")
	}
}

func TestTogglePauseDoesNothingAfterGameOver(t *testing.T) {
	gs := newTestState()
	gs.over = true
	if gs.TogglePause() {
		t.Fatal("TogglePause should report false after game over")
	}
	if gs.paused || gs.IsUserPaused() {
		t.Fatalf("paused=%v userPaused=%v after game over", gs.paused, gs.IsUserPaused())
	}
}

func TestResetClearsUserPause(t *testing.T) {
	gs := newTestState()
	gs.TogglePause()
	gs.Reset()
	if gs.paused || gs.IsUserPaused() || gs.showPauseHint.Load() {
		t.Fatalf("after Reset paused=%v userPaused=%v hint=%v", gs.paused, gs.IsUserPaused(), gs.showPauseHint.Load())
	}
}

func TestRomajiEntryEscapeCallsToggle(t *testing.T) {
	test.NewApp()
	calls := 0
	e := newRomajiEntry(func() { calls++ })
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if calls != 1 {
		t.Fatalf("Escape called the toggle %d times, want 1", calls)
	}
}

func TestRomajiEntryOtherKeysReachEntry(t *testing.T) {
	test.NewApp()
	calls := 0
	e := newRomajiEntry(func() { calls++ })
	e.TypedRune('k')
	e.TypedRune('a')
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if e.Text != "k" {
		t.Fatalf("entry text = %q, want %q", e.Text, "k")
	}
	if calls != 0 {
		t.Fatalf("non-Escape keys called the toggle %d times", calls)
	}
}

func TestPauseButtonTogglesGameAndIcon(t *testing.T) {
	test.NewApp()
	defer test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()

	gs := newTestState()
	gameCanvas := newGameCanvas(gs)
	ib := newInputBar(gs, newStatsPanel(), gameCanvas, w)

	if ib.pauseBtn.Icon != theme.MediaPauseIcon() {
		t.Fatal("pause button should start with the pause icon")
	}
	test.Tap(ib.pauseBtn)
	if !gs.IsUserPaused() || ib.pauseBtn.Icon != theme.MediaPlayIcon() {
		t.Fatalf("after tap userPaused=%v, icon is play=%v", gs.IsUserPaused(), ib.pauseBtn.Icon == theme.MediaPlayIcon())
	}
	ib.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if gs.IsUserPaused() || ib.pauseBtn.Icon != theme.MediaPauseIcon() {
		t.Fatalf("after Esc userPaused=%v, icon is pause=%v", gs.IsUserPaused(), ib.pauseBtn.Icon == theme.MediaPauseIcon())
	}
}

func TestCanvasShowsPauseHintOnlyWhileUserPaused(t *testing.T) {
	test.NewApp()
	gs := newTestState()
	gc := newGameCanvas(gs)
	r := test.WidgetRenderer(gc)

	hasHint := func() bool {
		for _, o := range r.Objects() {
			if txt, ok := o.(*canvas.Text); ok && txt.Text == pauseHintText {
				return true
			}
		}
		return false
	}

	if hasHint() {
		t.Fatal("hint shown before pausing")
	}
	gs.TogglePause()
	if !hasHint() {
		t.Fatal("hint missing while user-paused")
	}
	gs.TogglePause()
	if hasHint() {
		t.Fatal("hint still shown after resuming")
	}
}
