package store

import (
	"database/sql"
	"math"
	"path/filepath"
	"reflect"
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

func TestIntroducedWordsRoundTrip(t *testing.T) {
	st := openTemp(t)
	got, err := st.IntroducedWords()
	if err != nil || len(got) != 0 {
		t.Fatalf("fresh IntroducedWords = %v, %v", got, err)
	}
	at := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	if err := st.SaveIntroduced("ねこ", at); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveIntroduced("ねこ", at.Add(time.Hour)); err != nil {
		t.Fatalf("saving an introduced word again: %v", err)
	}
	if err := st.SaveIntroduced("", at); err == nil {
		t.Fatal("an empty reading must be rejected")
	}
	got, err = st.IntroducedWords()
	if err != nil || !reflect.DeepEqual(got, map[string]bool{"ねこ": true}) {
		t.Fatalf("IntroducedWords = %v, %v", got, err)
	}
	var stamp int64
	if err := st.db.QueryRow(`SELECT introduced_at FROM introduced_words WHERE kana = 'ねこ'`).Scan(&stamp); err != nil || stamp != at.Unix() {
		t.Fatalf("introduced_at = %d, %v; want the first time %d", stamp, err, at.Unix())
	}
}

func TestIntroducedSeededOnceFromWordProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// A stage 3 database: word progress, but no introduced_words table yet.
	_ = st.SaveWordStats("ねこ", 2, 1)
	_ = st.SaveWordStats("いぬ", 0, 1)
	_ = st.SaveMemoryState("word:あめ", memory.State{Stability: 1, LastSeen: time.Now()})
	_ = st.SaveMemoryState("word:ねこ", memory.State{Stability: 1, LastSeen: time.Now()})
	_ = st.SaveMemoryState("kana:か", memory.State{Stability: 1, LastSeen: time.Now()})
	if _, err := st.db.Exec(`DROP TABLE introduced_words`); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.IntroducedWords()
	if want := map[string]bool{"ねこ": true, "いぬ": true, "あめ": true}; err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("seeded = %v, %v; want %v", got, err, want)
	}
	_ = st.SaveWordStats("みず", 1, 0)
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if got, _ := st.IntroducedWords(); got["みず"] {
		t.Fatal("re-open must not seed again: word stats alone do not introduce a word any more")
	}
}

func TestWordPackageSettings(t *testing.T) {
	st := openTemp(t)
	if n, err := st.PackageSize(); err != nil || n != DefaultPackageSize || DefaultPackageSize != 5 {
		t.Fatalf("unset PackageSize = %d, %v; want 5", n, err)
	}
	if s, err := st.WordSource(); err != nil || s != "" {
		t.Fatalf("unset WordSource = %q, %v; want \"\" (auto)", s, err)
	}
	if f, err := st.KanaFont(); err != nil || f != "noto" {
		t.Fatalf("unset KanaFont = %q, %v; want noto", f, err)
	}
	if p, err := st.NewestPackage(); err != nil || p != nil {
		t.Fatalf("unset NewestPackage = %v, %v; want nil", p, err)
	}

	for _, n := range []int{3, 10} {
		if err := st.SavePackageSize(n); err != nil {
			t.Fatal(err)
		}
		if got, _ := st.PackageSize(); got != n {
			t.Fatalf("PackageSize = %d, want %d", got, n)
		}
	}
	if err := st.SavePackageSize(4); err == nil {
		t.Fatal("package size 4 must be rejected")
	}
	if err := st.setSetting(packageSizeKey, "7"); err != nil {
		t.Fatal(err)
	}
	if n, _ := st.PackageSize(); n != DefaultPackageSize {
		t.Fatalf("invalid stored size = %d, want the default", n)
	}

	if err := st.SaveWordSource("list:minna-no-nihongo"); err != nil {
		t.Fatal(err)
	}
	if s, _ := st.WordSource(); s != "list:minna-no-nihongo" {
		t.Fatalf("WordSource = %q", s)
	}
	if err := st.SaveKanaFont("mplus"); err != nil {
		t.Fatal(err)
	}
	if f, _ := st.KanaFont(); f != "mplus" {
		t.Fatalf("KanaFont = %q", f)
	}
	if err := st.SaveNewestPackage([]string{"ねこ", "いぬ"}); err != nil {
		t.Fatal(err)
	}
	if p, _ := st.NewestPackage(); !reflect.DeepEqual(p, []string{"ねこ", "いぬ"}) {
		t.Fatalf("NewestPackage = %v", p)
	}
	if err := st.SaveNewestPackage(nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := st.NewestPackage(); p != nil {
		t.Fatalf("NewestPackage after saving nil = %v, want nil", p)
	}
}

func TestProgressRoundTrip(t *testing.T) {
	st := openTemp(t)
	if got, err := st.Progress("kana:"); err != nil || len(got) != 0 {
		t.Fatalf("empty store: %v, %v", got, err)
	}
	_ = st.SaveProgress("kana:か", Progress{Level: 3, Flagged: true})
	_ = st.SaveProgress("kana:か", Progress{Level: 4, Flagged: true})
	_ = st.SaveProgress("word:ねこ", Progress{Level: 5})
	kana, err := st.Progress("kana:")
	if err != nil || len(kana) != 1 || kana["か"] != (Progress{Level: 4, Flagged: true}) {
		t.Fatalf("kana = %v, %v", kana, err)
	}
	if words, _ := st.Progress("word:"); words["ねこ"] != (Progress{Level: 5}) || len(words) != 1 {
		t.Fatalf("words = %v", words)
	}
}

func TestProgressSeededOnceFromStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// A database from before learning levels: stats, but no progress table.
	_ = st.SaveKanaStats("か", 3, 4, 0)
	_ = st.SaveKanaStats("き", 2, 1, 0)
	_ = st.SaveKanaStats("く", 0, 2, 0)
	_ = st.SaveKanaStats("け", 9, 0, 9)
	_ = st.SaveWordStats("ねこ", 4, 1)
	_ = st.SaveWordStats("いぬ", 1, 3)
	if _, err := st.db.Exec(`DROP TABLE progress`); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	kana, _ := st.Progress("kana:")
	want := map[string]Progress{"か": {Level: 5}, "き": {Level: 2}, "け": {Level: 5}}
	if !reflect.DeepEqual(kana, want) {
		t.Fatalf("kana = %v, want %v (no flags, nothing for 0 correct)", kana, want)
	}
	words, _ := st.Progress("word:")
	if want := map[string]Progress{"ねこ": {Level: 5}, "いぬ": {Level: 1}}; !reflect.DeepEqual(words, want) {
		t.Fatalf("words = %v, want %v", words, want)
	}
	_ = st.SaveKanaStats("こ", 5, 0, 5)
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if kana, _ := st.Progress("kana:"); len(kana) != 3 {
		t.Fatalf("re-open must not seed again: %v", kana)
	}
}

func TestResetKana(t *testing.T) {
	st := openTemp(t)
	now := time.Now()
	for _, c := range []string{"か", "カ"} {
		_ = st.SaveKanaStats(c, 3, 1, 2)
		_ = st.SaveMemoryState("kana:"+c, memory.State{Stability: 1, LastSeen: now})
		_ = st.SaveProgress("kana:"+c, Progress{Level: 5})
	}
	_ = st.SaveWordStats("か", 1, 0)
	_ = st.SaveMemoryState("word:か", memory.State{Stability: 1, LastSeen: now})
	_ = st.SaveProgress("word:か", Progress{Level: 1})

	if err := st.ResetKana([]string{"か", "き"}); err != nil {
		t.Fatal(err)
	}
	stats, _ := st.KanaStatistics()
	kmem, _ := st.MemoryStates("kana:")
	kprog, _ := st.Progress("kana:")
	if _, ok := stats["か"]; ok || len(stats) != 1 || len(kmem) != 1 || len(kprog) != 1 {
		t.Fatalf("か must be gone, カ kept: stats %v memory %v progress %v", stats, kmem, kprog)
	}
	words, _ := st.WordStatistics()
	wmem, _ := st.MemoryStates("word:")
	wprog, _ := st.Progress("word:")
	if len(words) != 1 || len(wmem) != 1 || len(wprog) != 1 {
		t.Fatal("the word か is not the character か: word data stays")
	}
}

func TestWordsOnlyFocusAndLessonFilter(t *testing.T) {
	st := openTemp(t)
	if on, err := st.WordsOnly(); err != nil || on {
		t.Fatalf("unset WordsOnly = %v, %v; want false", on, err)
	}
	if on, err := st.FocusWeak(); err != nil || on {
		t.Fatalf("unset FocusWeak = %v, %v; want false", on, err)
	}
	if l, err := st.LessonFilter("minna"); err != nil || l != nil {
		t.Fatalf("unset LessonFilter = %v, %v; want nil (all lessons)", l, err)
	}
	if err := st.SaveWordsOnly(true); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFocusWeak(true); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveLessonFilter("minna", []string{"2", ""}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveLessonFilter("other", []string{"1"}); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.WordsOnly(); !on {
		t.Fatal("WordsOnly not saved")
	}
	if on, _ := st.FocusWeak(); !on {
		t.Fatal("FocusWeak not saved")
	}
	if l, _ := st.LessonFilter("minna"); !reflect.DeepEqual(l, []string{"2", ""}) {
		t.Fatalf("minna = %v, want [2 \"\"]", l)
	}
	if err := st.SaveLessonFilter("minna", nil); err != nil {
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("minna"); l != nil {
		t.Fatalf("cleared filter = %v, want nil", l)
	}
	if l, _ := st.LessonFilter("other"); !reflect.DeepEqual(l, []string{"1"}) {
		t.Fatalf("other = %v: filters are per list", l)
	}
	if err := st.SaveWordsOnly(false); err != nil {
		t.Fatal(err)
	}
	if on, _ := st.WordsOnly(); on {
		t.Fatal("WordsOnly off not saved")
	}
}

func TestRenameLessonFilter(t *testing.T) {
	st := openTemp(t)
	_ = st.SaveLessonFilter("minna", []string{"2", ""})
	_ = st.SaveLessonFilter("ours", []string{"9"}) // stale, from an earlier list of that name
	if err := st.RenameLessonFilter("minna", "ours"); err != nil {
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("ours"); !reflect.DeepEqual(l, []string{"2", ""}) {
		t.Fatalf("ours = %v, want the moved filter", l)
	}
	if l, _ := st.LessonFilter("minna"); l != nil {
		t.Fatalf("minna = %v, want none left", l)
	}
	if err := st.RenameLessonFilter("ours", "ours"); err != nil {
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("ours"); !reflect.DeepEqual(l, []string{"2", ""}) {
		t.Fatalf("the same name must keep the filter: %v", l)
	}
	if err := st.RenameLessonFilter("Ours", "ours"); err != nil { // no filter under "Ours"
		t.Fatal(err)
	}
	if l, _ := st.LessonFilter("ours"); l != nil {
		t.Fatalf("a list without a filter must not inherit a stale one: %v", l)
	}
}
