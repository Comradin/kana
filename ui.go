package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI
var (
	kanaStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FFFF")).
			Padding(0, 2)

	statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00"))
)

// View renders the game state to a string
func (m Model) View() string {
	if m.GameOver {
		return renderGameOver(m)
	}
	return renderGame(m)
}

// renderGameOver renders the game over screen
func renderGameOver(m Model) string {
	var gameOverMsg strings.Builder
	gameOverMsg.WriteString("\nGame Over!\n\n")
	gameOverMsg.WriteString(fmt.Sprintf("Final Score: %d\nMissed: %d\n\n", m.Score, m.Missed))

	if len(m.MissedKanas) > 0 {
		gameOverMsg.WriteString("Characters you missed:\n")
		gameOverMsg.WriteString("━━━━━━━━━━━━━━━━━━━━\n")

		// Group missed kanas to avoid duplicates
		seen := make(map[string]bool)
		for _, k := range m.MissedKanas {
			if !seen[k.Char] {
				seen[k.Char] = true
				gameOverMsg.WriteString(fmt.Sprintf("  %s  →  %s\n", k.Char, k.Romaji))
			}
		}
		gameOverMsg.WriteString("\n")
	}

	gameOverMsg.WriteString("Press ESC to exit\n")
	return gameOverMsg.String()
}

// renderGame renders the main game screen
func renderGame(m Model) string {
	var output strings.Builder

	// Clear and position at home
	output.WriteString("\033[2J\033[H")

	// Render each kana using absolute positioning
	for _, k := range m.Kanas {
		y := int(k.Y)
		if y >= 0 && y < m.Height && k.X >= 0 && k.X < m.Width {
			// Move cursor to position and render styled kana
			output.WriteString(fmt.Sprintf("\033[%d;%dH", y+1, k.X+1))
			output.WriteString(kanaStyle.Render(k.Char))
		}
	}

	// Move to bottom for status bar
	output.WriteString(fmt.Sprintf("\033[%d;1H", m.Height+1))
	statusLine := statusStyle.Render(fmt.Sprintf("Score: %d | Missed: %d/10 | Type: %s",
		m.Score, m.Missed, inputStyle.Render(m.Input)))
	output.WriteString(statusLine)

	// Instructions on next line
	output.WriteString(fmt.Sprintf("\033[%d;1H", m.Height+2))
	output.WriteString("Type the romaji and press ENTER | ESC to quit")

	return output.String()
}
