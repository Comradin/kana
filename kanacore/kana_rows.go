package kanacore

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

// AllKanaRows lists every row in progression order.
var AllKanaRows = []KanaRow{
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

// RowsInGroup returns the rows of one group in progression order.
func RowsInGroup(g Group) []KanaRow {
	rows := make([]KanaRow, 0)
	for _, row := range AllKanaRows {
		if row.Group == g {
			rows = append(rows, row)
		}
	}
	return rows
}

// BasicRows returns the 46-character basic rows.
func BasicRows() []KanaRow {
	return RowsInGroup(GroupBasic)
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
