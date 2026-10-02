package kanacore

import "testing"

func TestHiraganaGetRomaji(t *testing.T) {
	romaji, ok := Hiragana().GetRomaji("か")
	if !ok {
		t.Fatal("expected か to exist in Hiragana set")
	}
	if romaji != "ka" {
		t.Fatalf("expected romaji 'ka', got %q", romaji)
	}
}

func TestHiraganaGetCharacters(t *testing.T) {
	chars := Hiragana().GetCharacters()
	if len(chars) != 104 {
		t.Fatalf("expected 104 characters, got %d", len(chars))
	}
}

func TestAllKanaRowsCount(t *testing.T) {
	if len(AllKanaRows) != 54 {
		t.Fatalf("expected 54 rows, got %d", len(AllKanaRows))
	}
}

func TestCharToRow(t *testing.T) {
	rowID, ok := CharToRow["か"]
	if !ok {
		t.Fatal("expected か to be in CharToRow")
	}
	if rowID != "k" {
		t.Fatalf("expected row ID 'k', got %q", rowID)
	}
}

func TestDefaultRowIDs(t *testing.T) {
	ids := DefaultRowIDs()
	if len(ids) != 11 {
		t.Fatalf("expected 11 row IDs, got %d", len(ids))
	}
}

func TestMatchesBasic(t *testing.T) {
	cs := Hiragana()
	cases := []struct {
		char, input string
		want        bool
	}{
		{"か", "ka", true},
		{"か", "  KA ", true},
		{"し", "shi", true},
		{"し", "si", true},
		{"ち", "ti", true},
		{"つ", "tu", true},
		{"ふ", "hu", true},
		{"ん", "n", true},
		{"ん", "nn", true},
		{"を", "wo", true},
		{"を", "o", false},
		{"か", "ki", false},
		{"か", "", false},
		{"x", "ka", false},
	}
	for _, c := range cases {
		if got := cs.Matches(c.char, c.input); got != c.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", c.char, c.input, got, c.want)
		}
	}
}

func TestRowByID(t *testing.T) {
	row, ok := RowByID("k")
	if !ok || row.Label != "K-row (か)" {
		t.Fatalf("RowByID(k) = %+v, %v", row, ok)
	}
	if _, ok := RowByID("nope"); ok {
		t.Fatal("RowByID(nope) should not exist")
	}
}

func TestRowCharactersFollowEntries(t *testing.T) {
	row, _ := RowByID("vowels")
	got := row.Characters()
	want := []string{"あ", "い", "う", "え", "お"}
	if len(got) != len(want) {
		t.Fatalf("Characters() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Characters() = %v, want %v", got, want)
		}
	}
}

func TestAllKanaRowsHaveNoDuplicateCharacters(t *testing.T) {
	seen := make(map[string]string)
	total := 0
	for _, row := range AllKanaRows {
		for _, char := range row.Characters() {
			if other, dup := seen[char]; dup {
				t.Fatalf("%s appears in rows %s and %s", char, other, row.ID)
			}
			seen[char] = row.ID
			total++
		}
	}
	if total != 208 {
		t.Fatalf("expected 208 characters, got %d", total)
	}
}

func TestRowsAreOrderedByGroup(t *testing.T) {
	order := make(map[Group]int)
	for i, g := range Groups() {
		order[g.Group] = i
	}
	for _, info := range Scripts() {
		last := 0
		for _, row := range RowsFor(info.Script) {
			idx, ok := order[row.Group]
			if !ok {
				t.Fatalf("row %s has unknown group %q", row.ID, row.Group)
			}
			if idx < last {
				t.Fatalf("%s row %s (group %s) appears after a later group", info.Script, row.ID, row.Group)
			}
			last = idx
		}
	}
}

func TestBasicRowsUnchanged(t *testing.T) {
	rows := BasicRows()
	wantIDs := []string{"vowels", "k", "s", "t", "n", "h", "m", "y", "r", "w", "n-only"}
	if len(rows) != len(wantIDs) {
		t.Fatalf("BasicRows() has %d rows, want %d", len(rows), len(wantIDs))
	}
	chars := 0
	for i, row := range rows {
		if row.ID != wantIDs[i] {
			t.Errorf("BasicRows()[%d] = %s, want %s", i, row.ID, wantIDs[i])
		}
		chars += len(row.Entries)
	}
	if chars != 46 {
		t.Errorf("BasicRows() has %d characters, want 46", chars)
	}
}

func TestMatchesExtended(t *testing.T) {
	cs := Hiragana()
	cases := []struct {
		char, input string
		want        bool
	}{
		{"が", "ga", true},
		{"じ", "ji", true},
		{"じ", "zi", true},
		{"じ", "di", false},
		{"ぢ", "ji", true},
		{"ぢ", "di", true},
		{"ず", "zu", true},
		{"ず", "du", false},
		{"づ", "zu", true},
		{"づ", "du", true},
		{"ぱ", "pa", true},
		{"きゃ", "kya", true},
		{"しゃ", "sha", true},
		{"しゃ", "sya", true},
		{"ちゃ", "cha", true},
		{"ちゃ", "tya", true},
		{"ちゃ", "cya", true},
		{"じゃ", "ja", true},
		{"じゃ", "jya", true},
		{"じゃ", "zya", true},
		{"ぴょ", "pyo", true},
		{"きゃ", "ka", false},
	}
	for _, c := range cases {
		if got := cs.Matches(c.char, c.input); got != c.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", c.char, c.input, got, c.want)
		}
	}
}

func TestProgressionStepsCoverRowsInOrder(t *testing.T) {
	for _, info := range Scripts() {
		steps := ProgressionStepsFor(info.Script)
		rows := RowsFor(info.Script)
		var flat []string
		for i, step := range steps {
			if len(step) == 0 {
				t.Fatalf("%s step %d is empty", info.Script, i)
			}
			first, _ := RowByID(step[0])
			for _, id := range step {
				row, ok := RowByID(id)
				if !ok {
					t.Fatalf("%s step %d has unknown row %q", info.Script, i, id)
				}
				if row.Group != first.Group || row.Script != info.Script {
					t.Fatalf("%s step %d crosses groups or scripts: %v", info.Script, i, step)
				}
			}
			flat = append(flat, step...)
		}
		if len(flat) != len(rows) {
			t.Fatalf("%s steps cover %d rows, want %d", info.Script, len(flat), len(rows))
		}
		for i, row := range rows {
			if flat[i] != row.ID {
				t.Fatalf("%s step order position %d = %s, want %s", info.Script, i, flat[i], row.ID)
			}
		}
	}
	if got := ProgressionStepsFor(ScriptHiragana)[0]; len(got) != 1 || got[0] != "vowels" {
		t.Fatalf("first hiragana step = %v, want [vowels]", got)
	}
	if got := ProgressionStepsFor(ScriptKatakana)[0]; len(got) != 1 || got[0] != "kata:vowels" {
		t.Fatalf("first katakana step = %v, want [kata:vowels]", got)
	}
}

func TestKatakanaTable(t *testing.T) {
	cs := Katakana()
	cases := []struct {
		char, romaji string
		alts         []string
	}{
		{"ア", "a", nil},
		{"シ", "shi", []string{"si"}},
		{"チ", "chi", []string{"ti"}},
		{"ツ", "tsu", []string{"tu"}},
		{"フ", "fu", []string{"hu"}},
		{"ヲ", "wo", nil},
		{"ン", "n", []string{"nn"}},
		{"ガ", "ga", nil},
		{"ヂ", "ji", []string{"di"}},
		{"ヅ", "zu", []string{"du"}},
		{"パ", "pa", nil},
		{"キャ", "kya", nil},
		{"シャ", "sha", []string{"sya"}},
		{"ジャ", "ja", []string{"zya", "jya"}},
		{"ピョ", "pyo", nil},
	}
	for _, c := range cases {
		got, ok := cs.GetRomaji(c.char)
		if !ok || got != c.romaji {
			t.Errorf("Katakana %s = %q (%v), want %q", c.char, got, ok, c.romaji)
		}
		for _, alt := range c.alts {
			if !cs.Matches(c.char, alt) {
				t.Errorf("Katakana %s should accept %q", c.char, alt)
			}
		}
	}
	if len(cs.GetCharacters()) != 104 {
		t.Fatalf("Katakana() has %d characters, want 104", len(cs.GetCharacters()))
	}
}

func TestKatakanaRowsMirrorHiragana(t *testing.T) {
	if len(KatakanaRows) != len(HiraganaRows) {
		t.Fatalf("%d katakana rows, want %d", len(KatakanaRows), len(HiraganaRows))
	}
	for i, k := range KatakanaRows {
		h := HiraganaRows[i]
		if k.ID != "kata:"+h.ID || k.Group != h.Group || k.Script != ScriptKatakana || h.Script != ScriptHiragana {
			t.Fatalf("row %d: katakana %+v does not mirror hiragana %+v", i, k, h)
		}
		for j, e := range k.Entries {
			he := h.Entries[j]
			if e.Romaji != he.Romaji || len(e.Alt) != len(he.Alt) {
				t.Fatalf("%s: romaji/alternatives differ from %s", e.Char, he.Char)
			}
			for _, r := range e.Char {
				if r < 0x30A1 || r > 0x30F6 {
					t.Fatalf("%s contains non-katakana rune %U", e.Char, r)
				}
			}
		}
		for _, r := range k.Label {
			if r >= 0x3041 && r <= 0x3096 {
				t.Fatalf("label %q still contains hiragana", k.Label)
			}
		}
	}
	if row, _ := RowByID("kata:k"); row.Label != "K-row (カ)" {
		t.Fatalf("kata:k label = %q", row.Label)
	}
}

func TestScriptOfAndBaseRowID(t *testing.T) {
	if ScriptOf("k") != ScriptHiragana || ScriptOf("kata:k") != ScriptKatakana {
		t.Fatal("ScriptOf misclassifies row IDs")
	}
	if BaseRowID("kata:sy") != "sy" || BaseRowID("sy") != "sy" {
		t.Fatal("BaseRowID does not strip the katakana prefix")
	}
}

func TestAllKanaHasBothScripts(t *testing.T) {
	cs := AllKana()
	if len(cs.GetCharacters()) != 208 {
		t.Fatalf("AllKana() has %d characters, want 208", len(cs.GetCharacters()))
	}
	if !cs.Matches("か", "ka") || !cs.Matches("カ", "ka") {
		t.Fatal("AllKana should match both か and カ")
	}
	if len(BasicRows()) != 11 || BasicRows()[0].Script != ScriptHiragana {
		t.Fatal("BasicRows must stay the 11 hiragana basic rows")
	}
	if got := len(RowsInGroup(ScriptKatakana, GroupBasic)); got != 11 {
		t.Fatalf("katakana basic rows = %d, want 11", got)
	}
}
