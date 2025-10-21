package main

// Kana represents a falling character in the game
type Kana struct {
	Char   string
	Romaji string
	X      int
	Y      float64
	Speed  float64
}

// CharacterSet represents a collection of kana characters with their romaji
type CharacterSet struct {
	Name string
	Data map[string]string
}

// Hiragana returns the basic hiragana character set
func Hiragana() CharacterSet {
	return CharacterSet{
		Name: "Hiragana",
		Data: map[string]string{
			"あ": "a", "い": "i", "う": "u", "え": "e", "お": "o",
			"か": "ka", "き": "ki", "く": "ku", "け": "ke", "こ": "ko",
			"さ": "sa", "し": "shi", "す": "su", "せ": "se", "そ": "so",
			"た": "ta", "ち": "chi", "つ": "tsu", "て": "te", "と": "to",
			"な": "na", "に": "ni", "ぬ": "nu", "ね": "ne", "の": "no",
			"は": "ha", "ひ": "hi", "ふ": "fu", "へ": "he", "ほ": "ho",
			"ま": "ma", "み": "mi", "む": "mu", "め": "me", "も": "mo",
			"や": "ya", "ゆ": "yu", "よ": "yo",
			"ら": "ra", "り": "ri", "る": "ru", "れ": "re", "ろ": "ro",
			"わ": "wa", "を": "wo", "ん": "n",
		},
	}
}

// GetCharacters returns a slice of all characters in the set
func (cs CharacterSet) GetCharacters() []string {
	chars := make([]string, 0, len(cs.Data))
	for char := range cs.Data {
		chars = append(chars, char)
	}
	return chars
}

// GetRomaji returns the romaji for a given character
func (cs CharacterSet) GetRomaji(char string) (string, bool) {
	romaji, exists := cs.Data[char]
	return romaji, exists
}
