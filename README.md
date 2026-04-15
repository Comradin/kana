# Kana

A typing game for learning Japanese Hiragana characters.

## Overview

Kana is an interactive typing game that helps you learn Japanese Hiragana through practice. Characters fall from the top of the screen, and you type their romaji (romanized) equivalents to score points. Track your progress, focus on specific character rows, and improve your hiragana recognition skills.

## Apps

Kana currently ships as two separate binaries:

| Binary | Framework | Status |
|--------|-----------|--------|
| `kana` | Bubble Tea (terminal) | Maintained, likely to be dropped once the desktop app matures |
| `kana-desktop` | Fyne v2 (desktop GUI) | Active development, intended long-term home |

The terminal app will remain available while the desktop app is still finding its feet, but expect it to be retired once feature parity is solid.

## Features

### Core Gameplay
- **46 Basic Hiragana Characters**: Practice all fundamental hiragana from あ (a) to ん (n)
- **Falling Character Mechanic**: Characters spawn at the top and fall at variable speeds
- **Romaji Input**: Type the romanized equivalent and press Enter to score
- **Score Limit Mode**: Set a target score or 0 for endless practice
- **Miss Limit**: Game ends after 10 missed characters

### Progress Tracking
- **Persistent Statistics**: Progress is saved to a local SQLite database (`kana.db`)
- **Per-Character Stats**: Correct answers, misses, and current streak per hiragana
- **Session vs Overall Stats**: See how this session compares to your cumulative history

### Customization
- **Row Selection**: Choose which hiragana rows to practice (vowels, k-row, s-row, etc.)
- **Auto-Progression**: Automatically unlock new rows as you master previous ones (80% threshold)
- **Configurable Score Limit**: Set a target or play endlessly

### Desktop App (Fyne)
- Warm paper tile aesthetic — stamp-style kana tiles on a parchment background
- Split layout: game canvas on the left, stats panel on the right
- In-game settings via gear icon (no pre-game setup form)
- Game-over dialog with missed character review and Play Again

### Terminal App (Bubble Tea)
- Runs in any terminal emulator
- Split-screen: game field left, progress table right
- Interactive setup form before each session
- Accessible mode via `KANAGAME_ACCESSIBLE_UI` environment variable

## Installation

### Prerequisites

**Both apps:**
- Go 1.25.1 or later

**Desktop app only (Fyne requires CGO and system graphics libraries):**
- **Linux**: `libGL`, `libX11`, `libXrandr`, `libXi`, `libXcursor` dev headers
  - Arch/CachyOS: `sudo pacman -S mesa libx11 libxrandr libxi libxcursor`
  - Debian/Ubuntu: `sudo apt install libgl1-mesa-dev libx11-dev libxrandr-dev libxi-dev libxcursor-dev`
- **macOS**: Xcode Command Line Tools (`xcode-select --install`)
- Must be compiled natively per target OS (CGO does not cross-compile easily)

### Build from Source

```bash
git clone <repository-url>
cd kana
go mod tidy

# Terminal app
go build -o kana .

# Desktop app
go build -o kana-desktop ./fyne/
```

### Run directly

```bash
# Terminal app
go run main.go

# Desktop app
go run ./fyne/
```

## How to Play

1. Launch the app (`./kana` or `./kana-desktop`)
2. **Desktop**: jump straight into the game; open the gear icon to adjust rows, auto-progression, and score limit
   **Terminal**: configure your session in the setup form before play begins
3. As characters fall, type their romaji equivalent and press Enter
4. Each correct answer scores 10 points
5. The game ends when you reach your score target, miss 10 characters, or quit

### Controls

**Desktop app:**
- **Type + Enter**: Submit answer
- **Gear icon**: Open settings during a game
- **Game-over dialog**: Play Again or Quit

**Terminal app:**
- **Type + Enter**: Submit answer
- **Backspace**: Delete last character
- **ESC**: Quit current game (first press) or exit (on game over screen)
- **Ctrl+C**: Exit immediately

### Scoring
- **+10 points** per correct answer
- Session ends on: target score reached, 10 misses, or manual quit

## Character Set

All 46 basic hiragana characters, organized by row:

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

### Shared Core (`kanacore/`)

Character data and row definitions live in `kanacore/`, shared by both apps:

- `kana.go`: `Kana` struct, `CharacterSet` with all 46 hiragana
- `kana_rows.go`: `KanaRow` definitions, `AllKanaRows`, `CharToRow` lookup

### Desktop App (`fyne/`)

Built on [Fyne v2](https://fyne.io) with goroutine-based game loop:

- `main.go`: Entry point, opens store, runs window
- `app.go`: Window construction, event watcher goroutine
- `game.go`: `GameState` — tick/spawn goroutines, answer checking, stats, auto-progression
- `canvas.go`: `GameCanvas` widget, atomic snapshot renderer (avoids mutex/render-thread deadlock)
- `tile.go`: `KanaTile` — shadow + face + text canvas objects
- `stats.go`: `StatsPanel` widget with persistent label pool
- `input.go`: `InputBar` — score, miss count, text entry
- `settings.go`: In-game settings dialog
- `theme.go`: `KanaTheme` — warm paper colour palette

### Terminal App (``)

Built on [Bubble Tea](https://github.com/charmbracelet/bubbletea) using the Elm Architecture:

- `main.go`: Entry point
- `game.go`: Model, Update, game logic
- `ui.go`: View rendering with Lipgloss
- `kana.go`: Character definitions (legacy; kanacore is the canonical source)
- `settings_form.go`: Pre-game setup form using Huh
- `store/store.go`: SQLite persistence (shared with desktop app)

### Game Timing
- Tick loop: 100ms
- Spawn interval: 4 seconds
- Desktop tile speed: 3.75–6.25 px/tick
- Terminal fall speed: 0.15–0.25 units/tick

## Data Persistence

Both apps share `kana.db` (SQLite) in the current working directory:

- Selected hiragana rows
- Auto-progression setting
- Score limit preference
- Per-character statistics (correct count, miss count, current streak)

The database is created automatically on first run.

## Dependencies

- [Fyne v2](https://fyne.io) — Desktop GUI framework (desktop app)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Terminal UI framework (terminal app)
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Terminal styling (terminal app)
- [Huh](https://github.com/charmbracelet/huh) — Interactive forms (terminal app)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite driver

## Development

```bash
# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Build both apps
go build -o kana .
go build -o kana-desktop ./fyne/

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
