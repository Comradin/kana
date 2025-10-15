package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Kana struct {
	char   string
	romaji string
	x      int
	y      float64
	speed  float64
}

var kanaMap = map[string]string{
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
}

type model struct {
	kanas       []*Kana
	width       int
	height      int
	score       int
	missed      int
	input       string
	gameOver    bool
	lastSpawn   time.Time
	lastUpdate  time.Time
	missedKanas []Kana // Track kanas that reached the bottom
}

type tickMsg time.Time
type spawnMsg time.Time

func initialModel() model {
	return model{
		kanas:       make([]*Kana, 0),
		width:       80,
		height:      24,
		lastSpawn:   time.Now(),
		lastUpdate:  time.Now(),
		missedKanas: make([]Kana, 0),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), spawnCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func spawnCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(t time.Time) tea.Msg {
		return spawnMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height - 3 // Reserve space for status bar and instructions

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.checkAnswer()
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}

	case tickMsg:
		if !m.gameOver {
			m.update()
			return m, tickCmd()
		}

	case spawnMsg:
		if !m.gameOver {
			m.spawnKana()
			return m, spawnCmd()
		}
	}

	return m, nil
}

func (m *model) checkAnswer() {
	for i, k := range m.kanas {
		if k.romaji == m.input {
			m.kanas = append(m.kanas[:i], m.kanas[i+1:]...)
			m.score += 10
			return
		}
	}
}

func (m *model) spawnKana() {
	kanaChars := make([]string, 0, len(kanaMap))
	for k := range kanaMap {
		kanaChars = append(kanaChars, k)
	}

	char := kanaChars[rand.Intn(len(kanaChars))]
	kana := &Kana{
		char:   char,
		romaji: kanaMap[char],
		x:      rand.Intn(m.width - 10) + 5,
		y:      0,
		speed:  0.15 + rand.Float64()*0.1,
	}
	m.kanas = append(m.kanas, kana)
}

func (m *model) update() {
	for i := len(m.kanas) - 1; i >= 0; i-- {
		m.kanas[i].y += m.kanas[i].speed

		if int(m.kanas[i].y) >= m.height {
			// Store the missed kana before removing it
			m.missedKanas = append(m.missedKanas, *m.kanas[i])
			m.kanas = append(m.kanas[:i], m.kanas[i+1:]...)
			m.missed++
			if m.missed >= 10 {
				m.gameOver = true
			}
		}
	}
}

func (m model) View() string {
	if m.gameOver {
		var gameOverMsg strings.Builder
		gameOverMsg.WriteString("\nGame Over!\n\n")
		gameOverMsg.WriteString(fmt.Sprintf("Final Score: %d\nMissed: %d\n\n", m.score, m.missed))

		if len(m.missedKanas) > 0 {
			gameOverMsg.WriteString("Characters you missed:\n")
			gameOverMsg.WriteString("━━━━━━━━━━━━━━━━━━━━\n")

			// Group missed kanas to avoid duplicates
			seen := make(map[string]bool)
			for _, k := range m.missedKanas {
				if !seen[k.char] {
					seen[k.char] = true
					gameOverMsg.WriteString(fmt.Sprintf("  %s  →  %s\n", k.char, k.romaji))
				}
			}
			gameOverMsg.WriteString("\n")
		}

		gameOverMsg.WriteString("Press ESC to exit\n")
		return gameOverMsg.String()
	}

	// Create styles
	kanaStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00FFFF")).
		Padding(0, 2)

	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#444444")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	inputStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00"))

	// Build output with absolute positioning
	var output strings.Builder

	// Clear and position at home
	output.WriteString("\033[2J\033[H")

	// Render each kana using absolute positioning
	for _, k := range m.kanas {
		y := int(k.y)
		if y >= 0 && y < m.height && k.x >= 0 && k.x < m.width {
			// Move cursor to position and render styled kana
			output.WriteString(fmt.Sprintf("\033[%d;%dH", y+1, k.x+1))
			output.WriteString(kanaStyle.Render(k.char))
		}
	}

	// Move to bottom for status bar
	output.WriteString(fmt.Sprintf("\033[%d;1H", m.height+1))
	statusLine := statusStyle.Render(fmt.Sprintf("Score: %d | Missed: %d/10 | Type: %s",
		m.score, m.missed, inputStyle.Render(m.input)))
	output.WriteString(statusLine)

	// Instructions on next line
	output.WriteString(fmt.Sprintf("\033[%d;1H", m.height+2))
	output.WriteString("Type the romaji and press ENTER | ESC to quit")

	return output.String()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
