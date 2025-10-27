package main

// KanaRow groups related kana characters by their consonant row.
type KanaRow struct {
	ID         string
	Label      string
	Characters []string
}

// AllKanaRows lists the basic hiragana rows used for practice.
var AllKanaRows = []KanaRow{
	{ID: "vowels", Label: "Vowels (あ)", Characters: []string{"あ", "い", "う", "え", "お"}},
	{ID: "k", Label: "K-row (か)", Characters: []string{"か", "き", "く", "け", "こ"}},
	{ID: "s", Label: "S-row (さ)", Characters: []string{"さ", "し", "す", "せ", "そ"}},
	{ID: "t", Label: "T-row (た)", Characters: []string{"た", "ち", "つ", "て", "と"}},
	{ID: "n", Label: "N-row (な)", Characters: []string{"な", "に", "ぬ", "ね", "の"}},
	{ID: "h", Label: "H-row (は)", Characters: []string{"は", "ひ", "ふ", "へ", "ほ"}},
	{ID: "m", Label: "M-row (ま)", Characters: []string{"ま", "み", "む", "め", "も"}},
	{ID: "y", Label: "Y-row (や)", Characters: []string{"や", "ゆ", "よ"}},
	{ID: "r", Label: "R-row (ら)", Characters: []string{"ら", "り", "る", "れ", "ろ"}},
	{ID: "w", Label: "W-row (わ)", Characters: []string{"わ", "を"}},
	{ID: "n-only", Label: "N (ん)", Characters: []string{"ん"}},
}

var charToRow map[string]string

func init() {
	charToRow = make(map[string]string)
	for _, row := range AllKanaRows {
		for _, char := range row.Characters {
			charToRow[char] = row.ID
		}
	}
}

func defaultRowIDs() []string {
	ids := make([]string, 0, len(AllKanaRows))
	for _, row := range AllKanaRows {
		ids = append(ids, row.ID)
	}
	return ids
}
