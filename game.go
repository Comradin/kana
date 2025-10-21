package main

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Model holds the game state
type Model struct {
	Kanas        []*Kana
	CharacterSet CharacterSet
	Width        int
	Height       int
	Score        int
	Missed       int
	Input        string
	GameOver     bool
	LastSpawn    time.Time
	LastUpdate   time.Time
	MissedKanas  []Kana // Track kanas that reached the bottom
}

// Message types for the Bubble Tea update loop
type tickMsg time.Time
type spawnMsg time.Time

// InitialModel creates a new game model with default values
func InitialModel() Model {
	return Model{
		Kanas:        make([]*Kana, 0),
		CharacterSet: Hiragana(),
		Width:        80,
		Height:       24,
		LastSpawn:    time.Now(),
		LastUpdate:   time.Now(),
		MissedKanas:  make([]Kana, 0),
	}
}

// Init initializes the game and returns the initial commands
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), spawnCmd())
}

// tickCmd returns a command that sends tick messages at regular intervals
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// spawnCmd returns a command that sends spawn messages at regular intervals
func spawnCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(t time.Time) tea.Msg {
		return spawnMsg(t)
	})
}

// Update handles messages and updates the game state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height - 3 // Reserve space for status bar and instructions

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.checkAnswer()
			m.Input = ""
		case "backspace":
			if len(m.Input) > 0 {
				m.Input = m.Input[:len(m.Input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.Input += msg.String()
			}
		}

	case tickMsg:
		if !m.GameOver {
			m.update()
			return m, tickCmd()
		}

	case spawnMsg:
		if !m.GameOver {
			m.spawnKana()
			return m, spawnCmd()
		}
	}

	return m, nil
}

// checkAnswer checks if the player's input matches any falling kana
func (m *Model) checkAnswer() {
	for i, k := range m.Kanas {
		if k.Romaji == m.Input {
			m.Kanas = append(m.Kanas[:i], m.Kanas[i+1:]...)
			m.Score += 10
			return
		}
	}
}

// spawnKana creates a new falling kana at a random position
func (m *Model) spawnKana() {
	chars := m.CharacterSet.GetCharacters()
	char := chars[rand.Intn(len(chars))]
	romaji, _ := m.CharacterSet.GetRomaji(char)

	kana := &Kana{
		Char:   char,
		Romaji: romaji,
		X:      rand.Intn(m.Width-10) + 5,
		Y:      0,
		Speed:  0.15 + rand.Float64()*0.1,
	}
	m.Kanas = append(m.Kanas, kana)
}

// update moves all falling kanas and checks for misses
func (m *Model) update() {
	for i := len(m.Kanas) - 1; i >= 0; i-- {
		m.Kanas[i].Y += m.Kanas[i].Speed

		if int(m.Kanas[i].Y) >= m.Height {
			// Store the missed kana before removing it
			m.MissedKanas = append(m.MissedKanas, *m.Kanas[i])
			m.Kanas = append(m.Kanas[:i], m.Kanas[i+1:]...)
			m.Missed++
			if m.Missed >= 10 {
				m.GameOver = true
			}
		}
	}
}
