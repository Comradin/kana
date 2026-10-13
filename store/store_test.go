package store

import (
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"

	"kana/memory"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestActiveScriptsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.ActiveScripts()
	if err != nil || got != nil {
		t.Fatalf("unset ActiveScripts = %v, %v; want nil, nil", got, err)
	}
	if err := st.SaveActiveScripts([]string{"hiragana", "katakana"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err = st.ActiveScripts()
	if err != nil || len(got) != 2 || got[0] != "hiragana" || got[1] != "katakana" {
		t.Fatalf("ActiveScripts = %v, %v", got, err)
	}
}

func TestKatakanaOfferedRoundTrip(t *testing.T) {
	st := openTemp(t)
	if got, err := st.KatakanaOffered(); err != nil || got {
		t.Fatalf("unset KatakanaOffered = %v, %v; want false, nil", got, err)
	}
	if err := st.SaveKatakanaOffered(true); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got, err := st.KatakanaOffered(); err != nil || !got {
		t.Fatalf("KatakanaOffered = %v, %v; want true", got, err)
	}
}

func TestDifficultyAndPointsRoundTrip(t *testing.T) {
	st := openTemp(t)
	if d, err := st.Difficulty(); err != nil || d != "" {
		t.Fatalf("unset Difficulty = %q, %v", d, err)
	}
	if p, err := st.PointsFactor(); err != nil || p != 0 {
		t.Fatalf("unset PointsFactor = %d, %v", p, err)
	}
	if err := st.SaveDifficulty("pro"); err != nil {
		t.Fatal(err)
	}
	if err := st.SavePointsFactor(3); err != nil {
		t.Fatal(err)
	}
	if d, _ := st.Difficulty(); d != "pro" {
		t.Fatalf("Difficulty = %q", d)
	}
	if p, _ := st.PointsFactor(); p != 3 {
		t.Fatalf("PointsFactor = %d", p)
	}
}

func TestMemoryStatesRoundTrip(t *testing.T) {
	st := openTemp(t)
	seen := time.Date(2026, 10, 12, 8, 30, 0, 0, time.UTC)
	if err := st.SaveMemoryState("kana:か", memory.State{Stability: 2, LastSeen: seen}); err != nil {
		t.Fatal(err)
	}
	_ = st.SaveMemoryState("word:x", memory.State{Stability: 1, LastSeen: seen})
	got, err := st.MemoryStates("kana:")
	if err != nil || len(got) != 1 {
		t.Fatalf("MemoryStates = %v, %v", got, err)
	}
	if s := got["か"]; s.Stability != 2 || s.LastSeen.Unix() != seen.Unix() {
		t.Fatalf("か = %+v", s)
	}
}

func TestMemorySeededOnceFromKanaStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE kana_stats (char TEXT PRIMARY KEY, correct_count INTEGER NOT NULL DEFAULT 0, miss_count INTEGER NOT NULL DEFAULT 0, streak INTEGER NOT NULL DEFAULT 0)`,
		`INSERT INTO kana_stats VALUES ('あ', 0, 2, 0), ('か', 3, 0, 3)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()

	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := st.MemoryStates("kana:")
	if len(got) != 1 || math.Abs(got["か"].Stability-memory.MinStability*8) > 1e-9 {
		t.Fatalf("seeded = %+v", got)
	}
	_ = st.SaveMemoryState("kana:か", memory.State{Stability: 5, LastSeen: time.Now()})
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if got, _ := st.MemoryStates("kana:"); got["か"].Stability != 5 {
		t.Fatalf("re-open must not re-seed: %+v", got["か"])
	}
}

func TestWordStatsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.WordStatistics()
	if err != nil || len(got) != 0 {
		t.Fatalf("empty WordStatistics = %v, %v", got, err)
	}
	if err := st.SaveWordStats("せんせい", 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("せんせい", 4, 1); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("ねこ", -2, -1); err != nil {
		t.Fatal(err)
	}
	got, err = st.WordStatistics()
	if err != nil {
		t.Fatal(err)
	}
	if w := got["せんせい"]; w != (WordStats{Kana: "せんせい", CorrectCount: 4, MissCount: 1}) {
		t.Fatalf("せんせい = %+v, want absolute values 4/1", w)
	}
	if w := got["ねこ"]; w.CorrectCount != 0 || w.MissCount != 0 {
		t.Fatalf("negative counts must be stored as 0, got %+v", w)
	}
	if err := st.SaveWordStats("", 1, 0); err == nil {
		t.Fatal("an empty reading must be rejected")
	}
}

func TestWordStatsSeparateFromKanaStats(t *testing.T) {
	st := openTemp(t)
	if err := st.SaveKanaStats("か", 5, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("か", 1, 0); err != nil {
		t.Fatal(err)
	}
	kana, _ := st.KanaStatistics()
	words, _ := st.WordStatistics()
	if kana["か"].CorrectCount != 5 || words["か"].CorrectCount != 1 {
		t.Fatalf("kana か = %+v, word か = %+v; the mosquito must not share the character's stats", kana["か"], words["か"])
	}
}

func TestWordStatsSurviveReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordStats("ねこ", 2, 0); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	st, err = Open(path) // migrate runs again on an existing database
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if got, _ := st.WordStatistics(); got["ねこ"].CorrectCount != 2 {
		t.Fatalf("after reopen = %+v", got)
	}
}

func TestWordLevelAndMeaningLang(t *testing.T) {
	st := openTemp(t)
	if l, err := st.WordLevel(); err != nil || l != DefaultWordLevel || DefaultWordLevel != 5 {
		t.Fatalf("unset WordLevel = %d, %v; want 5", l, err)
	}
	if m, err := st.MeaningLang(); err != nil || m != "" {
		t.Fatalf("unset MeaningLang = %q, %v; want \"\" (follow the system)", m, err)
	}
	if err := st.SaveWordLevel(3); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveWordLevel(0); err == nil {
		t.Fatal("level 0 must be rejected")
	}
	if err := st.SaveWordLevel(6); err == nil {
		t.Fatal("level 6 must be rejected")
	}
	if l, _ := st.WordLevel(); l != 3 {
		t.Fatalf("WordLevel = %d, want 3", l)
	}
	if err := st.SaveMeaningLang("de"); err != nil {
		t.Fatal(err)
	}
	if m, _ := st.MeaningLang(); m != "de" {
		t.Fatalf("MeaningLang = %q, want de", m)
	}
	if err := st.setSetting(wordLevelKey, "N9"); err != nil {
		t.Fatal(err)
	}
	if l, err := st.WordLevel(); err != nil || l != DefaultWordLevel {
		t.Fatalf("invalid stored level = %d, %v; want the default", l, err)
	}
}
