package kanacore

import "strings"

// Kana represents a falling character in the game.
type Kana struct {
	Char   string
	Romaji string
	Speed  float32
}

// CharacterSet represents a collection of kana characters with their romaji.
type CharacterSet struct {
	Name string
	Data map[string]string // char → canonical romaji
	alts map[string][]string
}

// NewCharacterSet builds a character set from the entries of the given rows.
func NewCharacterSet(name string, rows []KanaRow) CharacterSet {
	cs := CharacterSet{
		Name: name,
		Data: make(map[string]string),
		alts: make(map[string][]string),
	}
	for _, row := range rows {
		for _, e := range row.Entries {
			cs.Data[e.Char] = e.Romaji
			if len(e.Alt) > 0 {
				cs.alts[e.Char] = e.Alt
			}
		}
	}
	return cs
}

// Hiragana returns the character set of all hiragana rows.
func Hiragana() CharacterSet {
	return NewCharacterSet("Hiragana", HiraganaRows)
}

// Katakana returns the character set of all katakana rows.
func Katakana() CharacterSet {
	return NewCharacterSet("Katakana", KatakanaRows)
}

// AllKana returns the combined hiragana and katakana character set.
func AllKana() CharacterSet {
	return NewCharacterSet("Kana", AllKanaRows)
}

// GetCharacters returns a slice of all characters in the set.
func (cs CharacterSet) GetCharacters() []string {
	chars := make([]string, 0, len(cs.Data))
	for char := range cs.Data {
		chars = append(chars, char)
	}
	return chars
}

// GetRomaji returns the canonical romaji for a given character.
func (cs CharacterSet) GetRomaji(char string) (string, bool) {
	romaji, exists := cs.Data[char]
	return romaji, exists
}

// Matches reports whether input is the canonical romaji or an accepted
// alternative for char. Input is trimmed and lower-cased first.
func (cs CharacterSet) Matches(char, input string) bool {
	in := strings.ToLower(strings.TrimSpace(input))
	if in == "" {
		return false
	}
	romaji, ok := cs.Data[char]
	if !ok {
		return false
	}
	if in == romaji {
		return true
	}
	for _, alt := range cs.alts[char] {
		if in == alt {
			return true
		}
	}
	return false
}
