package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"kana/memory"
	"kana/store"
)

func TestRecordWordCorrectTouchesOnlyWordProgress(t *testing.T) {
	gs := newTestState()
	gs.recordWordCorrect("あい")
	if got := gs.wordSessionStats["あい"]; got.CorrectCount != 1 || got.Kana != "あい" || !gs.wordSessionDirty {
		t.Fatalf("word session = %+v, dirty %v", got, gs.wordSessionDirty)
	}
	if s, ok := gs.wordMemory["あい"]; !ok || s.Stability != memory.MinStability || !gs.wordMemoryDirty["あい"] {
		t.Fatalf("word memory = %+v, %v", s, ok)
	}
	if len(gs.sessionStats) != 0 || len(gs.memory) != 0 || gs.sessionDirty {
		t.Fatal("a word hit must not touch kana stats or kana memory")
	}
	if gs.answerStreak != 1 {
		t.Fatalf("answerStreak = %d, want 1", gs.answerStreak)
	}
}

func TestRecordWordMiss(t *testing.T) {
	gs := newTestState()
	gs.tempoStep, gs.answerStreak = 3, 4
	gs.recordWordMiss("あい")
	if got := gs.wordSessionStats["あい"]; got.MissCount != 1 || !gs.wordSessionDirty {
		t.Fatalf("word session = %+v", got)
	}
	if _, ok := gs.wordMemory["あい"]; ok {
		t.Fatal("a miss on a new word must not create memory")
	}
	if gs.tempoStep != 1 || gs.answerStreak != 0 {
		t.Fatalf("tempoStep=%d streak=%d, want 1/0", gs.tempoStep, gs.answerStreak)
	}
	gs.wordMemory["あお"] = memory.State{Stability: 1, LastSeen: time.Now()}
	gs.recordWordMiss("あお")
	if s := gs.wordMemory["あお"].Stability; s < 0.29 || s > 0.31 || !gs.wordMemoryDirty["あお"] {
		t.Fatalf("miss on known word → %v, want 0.3 and dirty", s)
	}
}

func TestMergePersistsWordProgress(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)

	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.recordWordCorrect("あい")
	gs.recordWordMiss("あい")
	gs.mergeSessionStats() // only words changed: the kana merge returns early
	sessionLeft, dirty, memDirty := len(gs.wordSessionStats), gs.wordSessionDirty, len(gs.wordMemoryDirty)
	overall := gs.wordOverallStats["あい"]
	gs.mu.Unlock()

	if sessionLeft != 0 || dirty || memDirty != 0 {
		t.Fatalf("after merge: session %d, dirty %v, memory dirty %d", sessionLeft, dirty, memDirty)
	}
	if overall.CorrectCount != 2 || overall.MissCount != 1 {
		t.Fatalf("overall = %+v, want 2/1", overall)
	}
	if words, _ := st.WordStatistics(); words["あい"].CorrectCount != 2 || words["あい"].MissCount != 1 {
		t.Fatalf("persisted = %+v", words["あい"])
	}
	if mem, _ := st.MemoryStates("word:"); len(mem) != 1 {
		t.Fatalf("word memory = %v, want あい only", mem)
	}
	if kana, _ := st.MemoryStates("kana:"); len(kana) != 0 {
		t.Fatalf("word memory leaked into kana memory: %v", kana)
	}

	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if words, _ := st.WordStatistics(); words["あい"].CorrectCount != 3 {
		t.Fatalf("after second merge = %+v, want 3 (double-count?)", words["あい"])
	}

	reloaded := NewGameState(st)
	if reloaded.wordOverallStats["あい"].CorrectCount != 3 {
		t.Fatalf("reloaded overall = %+v", reloaded.wordOverallStats["あい"])
	}
	if _, ok := reloaded.wordMemory["あい"]; !ok {
		t.Fatal("word memory must load on start")
	}
}

// TestMergesAreIndependent breaks one stats table through a second
// connection: the other kind must still be merged.
func TestMergesAreIndependent(t *testing.T) {
	for _, broken := range []string{"kana_stats", "word_stats"} {
		t.Run(broken, func(t *testing.T) {
			test.NewApp()
			path := filepath.Join(t.TempDir(), "test.db")
			st, err := store.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = st.Close() })
			_ = st.SaveSelectedRows([]string{"vowels"})
			gs := NewGameState(st)

			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec("DROP TABLE " + broken); err != nil {
				t.Fatal(err)
			}

			gs.mu.Lock()
			gs.recordCorrect("か")
			gs.recordWordCorrect("あい")
			gs.mergeSessionStats()
			kanaDirty, wordDirty := gs.sessionDirty, gs.wordSessionDirty
			gs.mu.Unlock()

			if broken == "kana_stats" {
				if !kanaDirty || wordDirty {
					t.Fatalf("kanaDirty=%v wordDirty=%v, want true/false", kanaDirty, wordDirty)
				}
				if words, err := st.WordStatistics(); err != nil || words["あい"].CorrectCount != 1 {
					t.Fatalf("words = %v, %v", words, err)
				}
			} else {
				if kanaDirty || !wordDirty {
					t.Fatalf("kanaDirty=%v wordDirty=%v, want false/true", kanaDirty, wordDirty)
				}
				if kana, err := st.KanaStatistics(); err != nil || kana["か"].CorrectCount != 1 {
					t.Fatalf("kana = %v, %v", kana, err)
				}
			}
		})
	}
}

func TestResetReloadsWordStatsAndKeepsMemory(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	_ = st.SaveWordStats("あい", 2, 0)
	gs := NewGameState(st)
	if gs.wordOverallStats["あい"].CorrectCount != 2 {
		t.Fatalf("loaded = %+v", gs.wordOverallStats["あい"])
	}
	gs.mu.Lock()
	gs.recordWordCorrect("あい")
	gs.mu.Unlock()
	gs.Reset()
	if len(gs.wordSessionStats) != 0 || gs.wordSessionDirty {
		t.Fatal("Reset must clear the word session")
	}
	if gs.wordOverallStats["あい"].CorrectCount != 3 {
		t.Fatalf("after Reset overall = %+v, want 3", gs.wordOverallStats["あい"])
	}
	if _, ok := gs.wordMemory["あい"]; !ok {
		t.Fatal("Reset must keep word memory")
	}
}

func TestFailedWordMemoryLoadSkipsSave(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	gs := NewGameState(st)
	gs.mu.Lock()
	gs.wordMemoryLoadFailed = true
	gs.recordWordCorrect("あい")
	gs.mergeSessionStats()
	gs.mu.Unlock()
	if got, _ := st.MemoryStates("word:"); len(got) != 0 {
		t.Fatalf("failed load must not write word memory, got %v", got)
	}
}

func TestStoredWordLevelIsLoaded(t *testing.T) {
	test.NewApp()
	st := openTestStore(t)
	_ = st.SaveSelectedRows([]string{"vowels"})
	if l := NewGameState(st).wordLevel; l != 5 {
		t.Fatalf("default wordLevel = %d, want 5", l)
	}
	_ = st.SaveWordLevel(3)
	if l := NewGameState(st).wordLevel; l != 3 {
		t.Fatalf("wordLevel = %d, want 3", l)
	}
}
