package kanacore

import "strings"

// Group classifies rows by kind of kana; the order of Groups() is the
// progression order.
type Group string

const (
	GroupBasic      Group = "basic"
	GroupDakuon     Group = "dakuon"
	GroupHandakuon  Group = "handakuon"
	GroupYoon       Group = "yoon"
	GroupYoonDakuon Group = "yoon-dakuon"
)

// GroupInfo pairs a group with its display label.
type GroupInfo struct {
	Group Group
	Label string
}

// Groups returns all groups in display and progression order.
func Groups() []GroupInfo {
	return []GroupInfo{
		{GroupBasic, "Basic"},
		{GroupDakuon, "Dakuon"},
		{GroupHandakuon, "Handakuon"},
		{GroupYoon, "Yōon"},
		{GroupYoonDakuon, "Yōon + Dakuten"},
	}
}

// Script identifies a kana writing system.
type Script string

const (
	ScriptHiragana Script = "hiragana"
	ScriptKatakana Script = "katakana"
)

// ScriptInfo pairs a script with its display label.
type ScriptInfo struct {
	Script Script
	Label  string
}

// Scripts returns both scripts in display order.
func Scripts() []ScriptInfo {
	return []ScriptInfo{
		{ScriptHiragana, "Hiragana"},
		{ScriptKatakana, "Katakana"},
	}
}

// katakanaPrefix marks katakana row IDs so they never collide with the
// hiragana IDs already stored by existing users.
const katakanaPrefix = "kata:"

// ScriptOf returns the script a row ID belongs to.
func ScriptOf(rowID string) Script {
	if strings.HasPrefix(rowID, katakanaPrefix) {
		return ScriptKatakana
	}
	return ScriptHiragana
}

// BaseRowID strips the katakana prefix, giving the script-independent row ID.
func BaseRowID(rowID string) string {
	return strings.TrimPrefix(rowID, katakanaPrefix)
}

// Entry is one kana with its canonical Hepburn romaji and accepted alternatives.
type Entry struct {
	Char   string
	Romaji string
	Alt    []string
}

// KanaRow groups related kana characters by their consonant row.
type KanaRow struct {
	ID      string
	Label   string
	Group   Group
	Script  Script
	Entries []Entry
}

// Characters returns the row's kana in entry order.
func (r KanaRow) Characters() []string {
	chars := make([]string, len(r.Entries))
	for i, e := range r.Entries {
		chars[i] = e.Char
	}
	return chars
}

func entry(char, romaji string, alt ...string) Entry {
	return Entry{Char: char, Romaji: romaji, Alt: alt}
}

// HiraganaRows lists every hiragana row in progression order.
var HiraganaRows = withScript(ScriptHiragana, []KanaRow{
	{ID: "vowels", Label: "Vowels (あ)", Group: GroupBasic, Entries: []Entry{entry("あ", "a"), entry("い", "i"), entry("う", "u"), entry("え", "e"), entry("お", "o")}},
	{ID: "k", Label: "K-row (か)", Group: GroupBasic, Entries: []Entry{entry("か", "ka"), entry("き", "ki"), entry("く", "ku"), entry("け", "ke"), entry("こ", "ko")}},
	{ID: "s", Label: "S-row (さ)", Group: GroupBasic, Entries: []Entry{entry("さ", "sa"), entry("し", "shi", "si"), entry("す", "su"), entry("せ", "se"), entry("そ", "so")}},
	{ID: "t", Label: "T-row (た)", Group: GroupBasic, Entries: []Entry{entry("た", "ta"), entry("ち", "chi", "ti"), entry("つ", "tsu", "tu"), entry("て", "te"), entry("と", "to")}},
	{ID: "n", Label: "N-row (な)", Group: GroupBasic, Entries: []Entry{entry("な", "na"), entry("に", "ni"), entry("ぬ", "nu"), entry("ね", "ne"), entry("の", "no")}},
	{ID: "h", Label: "H-row (は)", Group: GroupBasic, Entries: []Entry{entry("は", "ha"), entry("ひ", "hi"), entry("ふ", "fu", "hu"), entry("へ", "he"), entry("ほ", "ho")}},
	{ID: "m", Label: "M-row (ま)", Group: GroupBasic, Entries: []Entry{entry("ま", "ma"), entry("み", "mi"), entry("む", "mu"), entry("め", "me"), entry("も", "mo")}},
	{ID: "y", Label: "Y-row (や)", Group: GroupBasic, Entries: []Entry{entry("や", "ya"), entry("ゆ", "yu"), entry("よ", "yo")}},
	{ID: "r", Label: "R-row (ら)", Group: GroupBasic, Entries: []Entry{entry("ら", "ra"), entry("り", "ri"), entry("る", "ru"), entry("れ", "re"), entry("ろ", "ro")}},
	{ID: "w", Label: "W-row (わ)", Group: GroupBasic, Entries: []Entry{entry("わ", "wa"), entry("を", "wo")}},
	{ID: "n-only", Label: "N (ん)", Group: GroupBasic, Entries: []Entry{entry("ん", "n", "nn")}},
	// Dakuon
	{ID: "g", Label: "G-row (が)", Group: GroupDakuon, Entries: []Entry{entry("が", "ga"), entry("ぎ", "gi"), entry("ぐ", "gu"), entry("げ", "ge"), entry("ご", "go")}},
	{ID: "z", Label: "Z-row (ざ)", Group: GroupDakuon, Entries: []Entry{entry("ざ", "za"), entry("じ", "ji", "zi"), entry("ず", "zu"), entry("ぜ", "ze"), entry("ぞ", "zo")}},
	{ID: "d", Label: "D-row (だ)", Group: GroupDakuon, Entries: []Entry{entry("だ", "da"), entry("ぢ", "ji", "di"), entry("づ", "zu", "du"), entry("で", "de"), entry("ど", "do")}},
	{ID: "b", Label: "B-row (ば)", Group: GroupDakuon, Entries: []Entry{entry("ば", "ba"), entry("び", "bi"), entry("ぶ", "bu"), entry("べ", "be"), entry("ぼ", "bo")}},
	// Handakuon
	{ID: "p", Label: "P-row (ぱ)", Group: GroupHandakuon, Entries: []Entry{entry("ぱ", "pa"), entry("ぴ", "pi"), entry("ぷ", "pu"), entry("ぺ", "pe"), entry("ぽ", "po")}},
	// Yōon
	{ID: "ky", Label: "KY (きゃ)", Group: GroupYoon, Entries: []Entry{entry("きゃ", "kya"), entry("きゅ", "kyu"), entry("きょ", "kyo")}},
	{ID: "sy", Label: "SH (しゃ)", Group: GroupYoon, Entries: []Entry{entry("しゃ", "sha", "sya"), entry("しゅ", "shu", "syu"), entry("しょ", "sho", "syo")}},
	{ID: "ch", Label: "CH (ちゃ)", Group: GroupYoon, Entries: []Entry{entry("ちゃ", "cha", "tya", "cya"), entry("ちゅ", "chu", "tyu", "cyu"), entry("ちょ", "cho", "tyo", "cyo")}},
	{ID: "ny", Label: "NY (にゃ)", Group: GroupYoon, Entries: []Entry{entry("にゃ", "nya"), entry("にゅ", "nyu"), entry("にょ", "nyo")}},
	{ID: "hy", Label: "HY (ひゃ)", Group: GroupYoon, Entries: []Entry{entry("ひゃ", "hya"), entry("ひゅ", "hyu"), entry("ひょ", "hyo")}},
	{ID: "my", Label: "MY (みゃ)", Group: GroupYoon, Entries: []Entry{entry("みゃ", "mya"), entry("みゅ", "myu"), entry("みょ", "myo")}},
	{ID: "ry", Label: "RY (りゃ)", Group: GroupYoon, Entries: []Entry{entry("りゃ", "rya"), entry("りゅ", "ryu"), entry("りょ", "ryo")}},
	// Yōon with dakuten / handakuten
	{ID: "gy", Label: "GY (ぎゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("ぎゃ", "gya"), entry("ぎゅ", "gyu"), entry("ぎょ", "gyo")}},
	{ID: "j", Label: "J (じゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("じゃ", "ja", "zya", "jya"), entry("じゅ", "ju", "zyu", "jyu"), entry("じょ", "jo", "zyo", "jyo")}},
	{ID: "by", Label: "BY (びゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("びゃ", "bya"), entry("びゅ", "byu"), entry("びょ", "byo")}},
	{ID: "py", Label: "PY (ぴゃ)", Group: GroupYoonDakuon, Entries: []Entry{entry("ぴゃ", "pya"), entry("ぴゅ", "pyu"), entry("ぴょ", "pyo")}},
})

// KatakanaRows mirrors HiraganaRows in katakana.
var KatakanaRows = toKatakanaRows(HiraganaRows)

// AllKanaRows lists the hiragana rows followed by the katakana rows.
var AllKanaRows = append(append([]KanaRow(nil), HiraganaRows...), KatakanaRows...)

func withScript(s Script, rows []KanaRow) []KanaRow {
	for i := range rows {
		rows[i].Script = s
	}
	return rows
}

// toKatakana shifts every hiragana rune (U+3041–U+3096) to its katakana
// counterpart 0x60 code points later; other runes are kept.
func toKatakana(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0x3041 && r <= 0x3096 {
			return r + 0x60
		}
		return r
	}, s)
}

func toKatakanaRows(rows []KanaRow) []KanaRow {
	out := make([]KanaRow, len(rows))
	for i, row := range rows {
		entries := make([]Entry, len(row.Entries))
		for j, e := range row.Entries {
			entries[j] = Entry{Char: toKatakana(e.Char), Romaji: e.Romaji, Alt: append([]string(nil), e.Alt...)}
		}
		out[i] = KanaRow{
			ID:      katakanaPrefix + row.ID,
			Label:   toKatakana(row.Label),
			Group:   row.Group,
			Script:  ScriptKatakana,
			Entries: entries,
		}
	}
	return out
}

// CharToRow maps each kana character to its row ID.
var CharToRow map[string]string

func init() {
	CharToRow = make(map[string]string)
	for _, row := range AllKanaRows {
		for _, char := range row.Characters() {
			CharToRow[char] = row.ID
		}
	}
}

// RowsFor returns the rows of one script in progression order.
func RowsFor(s Script) []KanaRow {
	rows := make([]KanaRow, 0)
	for _, row := range AllKanaRows {
		if row.Script == s {
			rows = append(rows, row)
		}
	}
	return rows
}

// RowsInGroup returns the rows of one script and group in progression order.
func RowsInGroup(s Script, g Group) []KanaRow {
	rows := make([]KanaRow, 0)
	for _, row := range AllKanaRows {
		if row.Script == s && row.Group == g {
			rows = append(rows, row)
		}
	}
	return rows
}

// BasicRows returns the 46-character hiragana basic rows.
func BasicRows() []KanaRow {
	return RowsInGroup(ScriptHiragana, GroupBasic)
}

// RowByID looks up a row by its ID.
func RowByID(id string) (KanaRow, bool) {
	for _, row := range AllKanaRows {
		if row.ID == id {
			return row, true
		}
	}
	return KanaRow{}, false
}

// DefaultRowIDs returns the IDs of the basic rows, used as the fallback selection.
func DefaultRowIDs() []string {
	rows := BasicRows()
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// hiraganaSteps lists the hiragana row IDs that auto-progression unlocks
// together, in order. A step never crosses a group boundary.
var hiraganaSteps = [][]string{
	{"vowels"}, {"k", "s"}, {"t", "n"}, {"h", "m"}, {"y", "r"}, {"w", "n-only"},
	{"g", "z"}, {"d", "b"},
	{"p"},
	{"ky", "sy"}, {"ch", "ny"}, {"hy", "my"}, {"ry"},
	{"gy", "j"}, {"by", "py"},
}

// ProgressionStepsFor returns a fresh copy of the learning-path steps for a
// script; katakana steps are the hiragana steps with the katakana prefix.
func ProgressionStepsFor(s Script) [][]string {
	steps := make([][]string, len(hiraganaSteps))
	for i, step := range hiraganaSteps {
		ids := make([]string, len(step))
		for j, id := range step {
			if s == ScriptKatakana {
				id = katakanaPrefix + id
			}
			ids[j] = id
		}
		steps[i] = ids
	}
	return steps
}
