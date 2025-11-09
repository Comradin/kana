package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const kanaCellWidth = 4

var (
	kanaStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FFFF")).
			Width(kanaCellWidth).
			Align(lipgloss.Center)

	statusStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00"))

	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFF00"))

	tableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	gameOverStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FFFF00")).
			Padding(1, 3).
			Align(lipgloss.Left).
			Background(lipgloss.Color("#1C1C1C")).
			Foreground(lipgloss.Color("#FFFFFF"))
)

// Hiragana structure for table display
// Organized by consonant rows and vowel columns
var kanaTable = []struct {
	consonant string
	vowels    []string // a, i, u, e, o
}{
	{"", []string{"あ", "い", "う", "え", "お"}},  // vowels
	{"k", []string{"か", "き", "く", "け", "こ"}}, // k-row
	{"s", []string{"さ", "し", "す", "せ", "そ"}}, // s-row
	{"t", []string{"た", "ち", "つ", "て", "と"}}, // t-row
	{"n", []string{"な", "に", "ぬ", "ね", "の"}}, // n-row
	{"h", []string{"は", "ひ", "ふ", "へ", "ほ"}}, // h-row
	{"m", []string{"ま", "み", "む", "め", "も"}}, // m-row
	{"y", []string{"や", "", "ゆ", "", "よ"}},   // y-row
	{"r", []string{"ら", "り", "る", "れ", "ろ"}}, // r-row
	{"w", []string{"わ", "", "", "", "を"}},    // w-row
	{"", []string{"ん", "", "", "", ""}},      // n
}

// View renders the game state to a string
func (m Model) View() string {
	if m.GameOver {
		return renderGameOverScreen(m)
	}
	return renderGameScreen(m)
}

func renderGameScreen(m Model) string {
	gameArea := renderGameArea(m)
	border := renderVerticalBorder(m.Height)
	infoArea := renderInfoArea(m)

	top := lipgloss.JoinHorizontal(lipgloss.Top, gameArea, border, infoArea)
	status := renderStatus(m)

	return lipgloss.JoinVertical(lipgloss.Left, top, status)
}

func renderGameOverScreen(m Model) string {
	title := "GAME OVER!"
	if m.GameOverReason == "score" {
		title = "SESSION COMPLETE!"
	}

	scoreLine := fmt.Sprintf("Final Score: %d", m.Score)
	if m.ScoreLimit > 0 {
		scoreLine = fmt.Sprintf("Final Score: %d/%d", m.Score, m.ScoreLimit)
	}

	lines := []string{
		title,
		scoreLine,
		fmt.Sprintf("Missed: %d/10", m.Missed),
	}

	switch m.GameOverReason {
	case "score":
		lines = append(lines, "", "You reached your target score. Nice work!")
	case "misses":
		lines = append(lines, "", "10 kana slipped through. Review them and try again.")
	case "quit":
		lines = append(lines, "", "You ended the session early. Review your progress below.")
	default:
		lines = append(lines, "", "Session ended.")
	}

	unique := make(map[string]Kana)
	for _, k := range m.MissedKanas {
		if _, exists := unique[k.Char]; !exists {
			unique[k.Char] = k
		}
	}

	if len(unique) > 0 {
		lines = append(lines, "", "Characters you missed:")
		keys := make([]string, 0, len(unique))
		for char := range unique {
			keys = append(keys, char)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, char := range keys {
			k := unique[char]
			lines = append(lines, fmt.Sprintf("  %s -> %s", k.Char, k.Romaji))
		}
	} else {
		lines = append(lines, "", "No missed characters this round!")
	}

	lines = append(lines, "", "Press ESC to exit")

	box := gameOverStyle.Render(strings.Join(lines, "\n"))

	height := m.Height + 2
	width := m.Width
	if height <= 0 {
		height = lipgloss.Height(box)
	}
	if width <= 0 {
		width = lipgloss.Width(box)
	}

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1C1C1C")))
}

func renderGameArea(m Model) string {
	if m.Height <= 0 || m.GameWidth <= 0 {
		return ""
	}

	rows := make([]string, m.Height)
	rowKanas := make(map[int][]*Kana)
	for _, k := range m.Kanas {
		y := int(k.Y)
		if y < 0 || y >= m.Height {
			continue
		}
		if k.X < 0 || k.X >= m.GameWidth {
			continue
		}
		rowKanas[y] = append(rowKanas[y], k)
	}

	for row := 0; row < m.Height; row++ {
		kanas := rowKanas[row]
		sort.Slice(kanas, func(i, j int) bool { return kanas[i].X < kanas[j].X })

		var builder strings.Builder
		current := 0
		for _, k := range kanas {
			x := k.X
			if x > m.GameWidth-kanaCellWidth {
				x = m.GameWidth - kanaCellWidth
			}
			if x > current {
				builder.WriteString(strings.Repeat(" ", x-current))
			}
			builder.WriteString(kanaStyle.Render(k.Char))
			current = x + kanaCellWidth
		}
		if current < m.GameWidth {
			builder.WriteString(strings.Repeat(" ", m.GameWidth-current))
		}

		line := builder.String()
		lineWidth := lipgloss.Width(line)
		if lineWidth < m.GameWidth {
			line += strings.Repeat(" ", m.GameWidth-lineWidth)
		}
		rows[row] = line
	}

	return strings.Join(rows, "\n")
}

func renderVerticalBorder(height int) string {
	if height <= 0 {
		return ""
	}
	borderChar := borderStyle.Render("│")
	lines := make([]string, height)
	for i := range lines {
		lines[i] = borderChar
	}
	return strings.Join(lines, "\n")
}

func renderStatus(m Model) string {
	scoreDisplay := fmt.Sprintf("%d", m.Score)
	if m.ScoreLimit > 0 {
		scoreDisplay = fmt.Sprintf("%d/%d", m.Score, m.ScoreLimit)
	}
	statusLine := statusStyle.Render(fmt.Sprintf("Score: %s | Missed: %d/10 | Type: %s",
		scoreDisplay, m.Missed, inputStyle.Render(m.Input)))
	instructions := "Type the romaji and press ENTER | ESC to quit"
	if m.ScoreLimit > 0 {
		instructions = fmt.Sprintf("Goal: %d points | %s", m.ScoreLimit, instructions)
	}
	return lipgloss.JoinVertical(lipgloss.Left, statusLine, instructions)
}

func renderInfoArea(m Model) string {
	if m.Height <= 0 {
		return ""
	}

	lines := []string{
		tableHeaderStyle.Render("HIRAGANA PROGRESS"),
		"",
		tableHeaderStyle.Render("   | a | i | u | e | o |"),
		tableHeaderStyle.Render("---+---+---+---+---+---|"),
	}

	for _, row := range kanaTable {
		label := row.consonant
		if label == "" {
			label = " "
		}

		var rowBuilder strings.Builder
		rowBuilder.WriteString(tableHeaderStyle.Render(fmt.Sprintf(" %s ", label)))
		rowBuilder.WriteString(tableCellStyle.Render("|"))

		for _, char := range row.vowels {
			switch {
			case char == "":
				rowBuilder.WriteString(tableCellStyle.Render("   |"))
			default:
				count := m.sessionCorrectCount(char)
				if count > 0 {
					rowBuilder.WriteString(tableCellStyle.Render(fmt.Sprintf(" %2d|", count)))
				} else {
					rowBuilder.WriteString(tableCellStyle.Render("  -|"))
				}
			}
		}
		lines = append(lines, rowBuilder.String())
	}

	lines = append(lines, "", tableHeaderStyle.Render("MISSED CHARACTERS"), "")

	if len(m.MissedKanas) > 0 {
		seen := make(map[string]bool)
		for _, k := range m.MissedKanas {
			if seen[k.Char] {
				continue
			}
			lines = append(lines, tableCellStyle.Render(k.Char))
			seen[k.Char] = true
		}
	} else {
		lines = append(lines, tableCellStyle.Render("None yet!"))
	}

	for len(lines) < m.Height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
