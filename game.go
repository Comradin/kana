package main

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"kana/store"
)

// Model holds the game state
type Model struct {
	Kanas          []*Kana
	CharacterSet   CharacterSet
	Width          int
	Height         int
	GameWidth      int // Width of the playing field (1/3 of total)
	Score          int
	ScoreLimit     int
	Missed         int
	Input          string
	GameOver       bool
	GameOverReason string
	LastSpawn      time.Time
	LastUpdate     time.Time
	MissedKanas    []Kana
	OverallStats   map[string]store.KanaStats
	SessionStats   map[string]store.KanaStats
	CurrentStreak  map[string]int
	SessionDirty   bool
	Store          *store.Store
	SelectedRows   map[string]bool
	AutoProgress   bool
}

// Message types for the Bubble Tea update loop
type tickMsg time.Time
type spawnMsg time.Time

// InitialModel creates a new game model with default values
func InitialModel(st *store.Store) Model {
	model := Model{
		Kanas:         make([]*Kana, 0),
		CharacterSet:  Hiragana(),
		Width:         80,
		Height:        24,
		GameWidth:     26, // 1/3 of 80
		LastSpawn:     time.Now(),
		LastUpdate:    time.Now(),
		ScoreLimit:    store.DefaultScoreLimit,
		MissedKanas:   make([]Kana, 0),
		OverallStats:  make(map[string]store.KanaStats),
		SessionStats:  make(map[string]store.KanaStats),
		CurrentStreak: make(map[string]int),
		Store:         st,
		SelectedRows:  make(map[string]bool),
	}

	model.applySelectedRows(defaultRowIDs())

	if st != nil {
		if rows, err := st.SelectedRows(); err == nil && len(rows) > 0 {
			model.applySelectedRows(rows)
		}

		if auto, err := st.AutoProgress(); err == nil {
			model.AutoProgress = auto
		}

		if limit, err := st.ScoreLimit(); err == nil {
			if limit < 0 {
				limit = 0
			}
			model.ScoreLimit = limit
		}

		if stats, err := st.KanaStatistics(); err == nil {
			for _, stat := range stats {
				copied := stat
				model.OverallStats[stat.Char] = copied
				model.CurrentStreak[stat.Char] = stat.Streak
			}
		}
	}

	return model
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
		m.GameWidth = m.Width / 3 // 1/3 for game area

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.mergeSessionStats()
			return m, tea.Quit
		case "esc":
			if !m.GameOver {
				m.endGame("quit")
				return m, nil
			}
			m.mergeSessionStats()
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
			m.recordCorrect(k.Char)
			if m.ScoreLimit > 0 && m.Score >= m.ScoreLimit {
				m.endGame("score")
			}
			return
		}
	}
}

// spawnKana creates a new falling kana at a random position
func (m *Model) spawnKana() {
	chars := m.availableCharacters()
	if len(chars) == 0 {
		return
	}
	char := chars[rand.Intn(len(chars))]
	romaji, _ := m.CharacterSet.GetRomaji(char)

	kana := &Kana{
		Char:   char,
		Romaji: romaji,
		X:      rand.Intn(m.GameWidth-10) + 5, // Spawn only in game area
		Y:      0,
		Speed:  0.15 + rand.Float64()*0.1,
	}
	m.Kanas = append(m.Kanas, kana)
}

// update moves all falling kanas and checks for misses
func (m *Model) update() {
	for i := len(m.Kanas) - 1; i >= 0; i-- {
		k := m.Kanas[i]
		k.Y += k.Speed
		m.Kanas[i] = k

		if int(k.Y) >= m.Height {
			m.recordMiss(k.Char)
			m.MissedKanas = append(m.MissedKanas, *k)
			m.Kanas = append(m.Kanas[:i], m.Kanas[i+1:]...)
			m.Missed++
			if m.Missed >= 10 {
				m.endGame("misses")
			}
		}
	}
}

func (m *Model) applySelectedRows(rows []string) {
	if m.SelectedRows == nil {
		m.SelectedRows = make(map[string]bool)
	}
	for key := range m.SelectedRows {
		delete(m.SelectedRows, key)
	}
	for _, id := range rows {
		m.SelectedRows[id] = true
	}
}

func (m *Model) SetSelectedRows(rows []string) {
	m.applySelectedRows(rows)
	if m.Store != nil {
		_ = m.Store.SaveSelectedRows(rows)
	}
}

func (m *Model) SelectedRowIDs() []string {
	if len(m.SelectedRows) == 0 {
		return nil
	}
	rows := make([]string, 0, len(m.SelectedRows))
	for id, ok := range m.SelectedRows {
		if ok {
			rows = append(rows, id)
		}
	}
	return rows
}

func (m *Model) SetAutoProgress(enabled bool) {
	m.AutoProgress = enabled
	if m.Store != nil {
		_ = m.Store.SaveAutoProgress(enabled)
	}
}

func (m *Model) SetScoreLimit(limit int) {
	if limit < 0 {
		limit = 0
	}
	m.ScoreLimit = limit
	if m.Store != nil {
		_ = m.Store.SaveScoreLimit(limit)
	}
}

func (m *Model) recordCorrect(char string) {
	streak := m.CurrentStreak[char] + 1
	m.CurrentStreak[char] = streak

	stat := m.SessionStats[char]
	stat.Char = char
	stat.CorrectCount++
	stat.Streak = streak
	m.SessionStats[char] = stat
	m.SessionDirty = true
}

func (m *Model) recordMiss(char string) {
	m.CurrentStreak[char] = 0

	stat := m.SessionStats[char]
	stat.Char = char
	stat.MissCount++
	stat.Streak = 0
	m.SessionStats[char] = stat
	m.SessionDirty = true
}

func (m *Model) endGame(reason string) {
	if !m.GameOver {
		m.GameOver = true
		if m.GameOverReason == "" {
			m.GameOverReason = reason
		}
	}
	m.mergeSessionStats()
}

func (m *Model) mergeSessionStats() {
	if !m.SessionDirty {
		return
	}

	for char, session := range m.SessionStats {
		if session.CorrectCount == 0 && session.MissCount == 0 && session.Streak == 0 && m.CurrentStreak[char] == 0 {
			continue
		}
		base := m.OverallStats[char]
		base.Char = char
		base.CorrectCount += session.CorrectCount
		base.MissCount += session.MissCount
		base.Streak = m.CurrentStreak[char]
		if m.Store != nil {
			_ = m.Store.SaveKanaStats(char, base.CorrectCount, base.MissCount, base.Streak)
		}
		m.OverallStats[char] = base
	}

	m.SessionDirty = false
}

func (m *Model) sessionCorrectCount(char string) int {
	if stat, ok := m.SessionStats[char]; ok {
		return stat.CorrectCount
	}
	return 0
}

func (m *Model) availableCharacters() []string {
	chars := m.CharacterSet.GetCharacters()
	if len(chars) == 0 {
		return nil
	}
	if len(m.SelectedRows) == 0 {
		return chars
	}

	filtered := make([]string, 0, len(chars))
	for _, char := range chars {
		rowID, ok := charToRow[char]
		if !ok {
			filtered = append(filtered, char)
			continue
		}
		if m.SelectedRows[rowID] {
			filtered = append(filtered, char)
		}
	}

	if len(filtered) == 0 {
		return chars
	}
	return filtered
}
