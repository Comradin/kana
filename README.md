# Kana

A terminal-based typing game for learning Japanese Hiragana characters.

## Overview

Kana is an interactive typing game that helps you learn Japanese Hiragana through practice. Characters fall from the top of your terminal screen, and you type their romaji (romanized) equivalents to score points. Track your progress, focus on specific character rows, and improve your hiragana recognition skills.

## Features

### Core Gameplay
- **46 Basic Hiragana Characters**: Practice all fundamental hiragana from あ (a) to ん (n)
- **Falling Character Mechanic**: Characters spawn at the top and fall at variable speeds
- **Romaji Input**: Type the romanized equivalent and press Enter to score
- **Visual Feedback**: Characters are highlighted in cyan boxes with real-time positioning

### Game Modes
- **Score Limit Mode**: Set a target score (default: 1000 points) to complete a session
- **Endless Practice**: Set score limit to 0 for unlimited practice
- **Multiple Ending Conditions**:
  - Reach your target score
  - Miss 10 characters
  - Quit anytime with ESC

### Progress Tracking
- **Persistent Statistics**: All your progress is saved to a local SQLite database (`kana.db`)
- **Per-Character Stats**: Track correct answers, misses, and current streak for each hiragana
- **Session Statistics**: View your performance during the current game session
- **Overall Statistics**: See your cumulative progress across all sessions
- **Live Progress Table**: Real-time display of successful matches per character

### Customization
- **Row Selection**: Choose which hiragana rows to practice (vowels, k-row, s-row, etc.)
- **Auto-Progression**: Automatically unlock new rows as you master previous ones
- **Configurable Difficulty**: Adjustable score limits and character sets
- **Setup Form**: Interactive configuration form before each session

### User Interface
- **Split-Screen Layout**:
  - Left (1/3): Game playing field with falling characters
  - Right (2/3): Hiragana progress table and statistics
- **Game Over Screen**: Review missed characters and session performance
- **Status Bar**: Live score, miss count, and input display
- **Accessible Mode**: Environment variable support for accessible UI (`KANAGAME_ACCESSIBLE_UI`)

## Installation

### Prerequisites
- Go 1.25.1 or later

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd kana

# Download dependencies
go mod tidy

# Build the binary
go build -o kana

# Run the game
./kana
```

Alternatively, run directly without building:

```bash
go run main.go
```

## How to Play

1. **Launch the game**: Run `./kana` or `go run main.go`
2. **Configure your session**:
   - Select which hiragana rows to practice
   - Enable/disable auto-progression
   - Set your target score (or 0 for endless mode)
3. **Type romaji equivalents**: As characters fall, type their romaji and press Enter
4. **Score points**: Each correct answer awards 10 points
5. **Avoid misses**: The game ends if 10 characters reach the bottom
6. **Review your progress**: Check the real-time stats table on the right side

### Controls
- **Type + Enter**: Submit your answer
- **Backspace**: Delete last character of input
- **ESC**: Quit current game (first press) or exit entirely (on game over screen)
- **Ctrl+C**: Exit immediately

### Scoring
- **+10 points** per correct answer
- Session ends when:
  - You reach your target score (configurable)
  - You miss 10 characters
  - You press ESC to quit

## Character Set

The game includes all 46 basic hiragana characters organized by rows:

| Row | Characters |
|-----|------------|
| Vowels | あ (a), い (i), う (u), え (e), お (o) |
| K-row | か (ka), き (ki), く (ku), け (ke), こ (ko) |
| S-row | さ (sa), し (shi), す (su), せ (se), そ (so) |
| T-row | た (ta), ち (chi), つ (tsu), て (te), と (to) |
| N-row | な (na), に (ni), ぬ (nu), ね (ne), の (no) |
| H-row | は (ha), ひ (hi), ふ (fu), へ (he), ほ (ho) |
| M-row | ま (ma), み (mi), む (mu), め (me), も (mo) |
| Y-row | や (ya), ゆ (yu), よ (yo) |
| R-row | ら (ra), り (ri), る (ru), れ (re), ろ (ro) |
| W-row | わ (wa), を (wo) |
| N | ん (n) |

## Architecture

Kana is built using the Elm Architecture pattern via the Bubble Tea framework:

- **Model**: Tracks game state including falling kanas, score, statistics, and configuration
- **Update**: Handles window sizing, keyboard input, character movement, and spawning
- **View**: Renders the split-screen interface using Lipgloss styling

### Key Components

- `game.go`: Core game logic, state management, and statistics tracking
- `ui.go`: Rendering logic for game area, info panel, and game over screen
- `kana.go`: Character set definitions and data structures
- `store/store.go`: SQLite-based persistence for settings and statistics
- `settings_form.go`: Interactive setup form using Huh
- `main.go`: Application entry point and initialization

### Game Timing
- Characters fall at 0.15-0.25 units per 100ms tick
- New characters spawn every 4 seconds
- Update loop runs every 100ms

## Data Persistence

All settings and statistics are stored in `kana.db` (SQLite database) in the current directory:

- Selected hiragana rows
- Auto-progression setting
- Score limit preference
- Per-character statistics (correct count, miss count, current streak)

The database is created automatically on first run.

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework using Elm Architecture
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions and terminal rendering
- [Huh](https://github.com/charmbracelet/huh) - Interactive forms and prompts
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) - Pure Go SQLite implementation

## Development

```bash
# Run the game in development
go run main.go

# Build for production
go build -o kana

# Run tests (if available)
go test ./...

# Update dependencies
go mod tidy
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Feel free to:

- Report bugs or request features by opening an issue
- Submit pull requests with improvements
- Share feedback on the learning experience
- Suggest additional character sets or game modes

When contributing code, please:
- Follow existing code style and patterns
- Test your changes thoroughly
- Provide clear commit messages
- Keep pull requests focused on a single improvement
