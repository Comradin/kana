# Kana

A typing game for learning Japanese Hiragana and Katakana characters.

## Overview

Kana is an interactive typing game that helps you learn Japanese Hiragana and Katakana through practice. Characters fall from the top of the screen, and you type their romaji (romanized) equivalents to score points. Track your progress, focus on specific character rows, and improve your kana recognition skills.

## Apps

Kana is a Fyne v2 desktop app (`kana-desktop`).

## Features

### Core Gameplay
- **104 Hiragana and 104 Katakana (basic, Dakuon, Handakuon, Yōon)**: Practice the full practical hiragana and katakana sets, from あ/ア (a) to the combined Yōon with (han)dakuten
- **Falling Character Mechanic**: Characters spawn at the top and fall for a fixed time (16–24 s, randomized per tile), independent of window height
- **Romaji Input**: Type the romanized equivalent and press Enter to score
- **Learning path**: start with あいうえお, unlock two rows at a time; each automatically unlocked row is introduced before it is asked, and the game pauses while new rows are introduced
- **Two scripts**: hiragana and katakana progress independently; switch scripts in the settings. After the basic hiragana you are offered katakana once.
- **Score Limit Mode**: Set a target score or 0 for endless practice
- **Miss Limit**: Game ends after 10 missed characters

### Progress Tracking
- **Persistent Statistics**: Progress is saved to a local SQLite database (`kana.db`)
- **Per-Character Stats**: Correct answers, misses, and current streak per hiragana and katakana character
- **Session vs Overall Stats**: The stats panel shows a session line (correct, missed, accuracy), a learning-path line per active script and a kana grid of each character's learning state; overall stats are persisted across sessions and drive progression

### Customization
- **Row Selection**: Choose which hiragana and katakana rows to practice, grouped by Basic, Dakuon, Handakuon, Yōon and Yōon + Dakuten
- **Auto-Progression**: per script, unlocks the next learning-path step (one or two rows) once every active row of that script is mastered: at least 80% of its kana (rounded down, at least one) answered correctly three times
- **Configurable Score Limit**: Set a target or play endlessly

### Desktop App (Fyne)
- Warm paper tile aesthetic — stamp-style kana tiles on a parchment background
- Split layout: game canvas on the left, stats panel on the right; window starts at 1000 × 760
- Stats panel: a session line, one learning-path line per active script, and a side-by-side hiragana/katakana kana grid (new/learning/mastered, with a red border on a character missed this session)
- In-game settings via gear icon (no pre-game setup form)
- Game-over dialog with missed character review and Play Again

## Installation

### Prerequisites

- Go 1.25.1 or later
- Fyne requires CGO and system graphics libraries:
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

go build -o kana-desktop ./fyne/
```

### Run directly

```bash
go run ./fyne/
```

## How to Play

1. Launch the app (`./kana-desktop`)
2. Jump straight into the game; open the gear icon to adjust rows, auto-progression, and score limit
3. As characters fall, type their romaji equivalent and press Enter
4. Each correct answer scores 10 points
5. The game ends when you reach your score target, miss 10 characters, or quit

### Controls

- **Type + Enter**: Submit answer
- **Gear icon**: Open settings during a game
- **Game-over dialog**: Play Again or Quit

### Scoring
- **+10 points** per correct answer
- Session ends on: target score reached, 10 misses, or manual quit

## Character Set

### Basic (46)

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

### Dakuon (20) and Handakuon (5)

| Row ID | Label | a | i | u | e | o |
|---|---|---|---|---|---|---|
| `g` | G-row (が) | が ga | ぎ gi | ぐ gu | げ ge | ご go |
| `z` | Z-row (ざ) | ざ za | じ ji (zi) | ず zu | ぜ ze | ぞ zo |
| `d` | D-row (だ) | だ da | ぢ ji (di) | づ zu (du) | で de | ど do |
| `b` | B-row (ば) | ば ba | び bi | ぶ bu | べ be | ぼ bo |
| `p` | P-row (ぱ) | ぱ pa | ぴ pi | ぷ pu | ぺ pe | ぽ po |

### Yōon (21)

| Row ID | Label | ya | yu | yo |
|---|---|---|---|---|
| `ky` | KY (きゃ) | きゃ kya | きゅ kyu | きょ kyo |
| `sy` | SH (しゃ) | しゃ sha (sya) | しゅ shu (syu) | しょ sho (syo) |
| `ch` | CH (ちゃ) | ちゃ cha (tya, cya) | ちゅ chu (tyu, cyu) | ちょ cho (tyo, cyo) |
| `ny` | NY (にゃ) | にゃ nya | にゅ nyu | にょ nyo |
| `hy` | HY (ひゃ) | ひゃ hya | ひゅ hyu | ひょ hyo |
| `my` | MY (みゃ) | みゃ mya | みゅ myu | みょ myo |
| `ry` | RY (りゃ) | りゃ rya | りゅ ryu | りょ ryo |

### Yōon with (han)dakuten (12)

| Row ID | Label | ya | yu | yo |
|---|---|---|---|---|
| `gy` | GY (ぎゃ) | ぎゃ gya | ぎゅ gyu | ぎょ gyo |
| `j` | J (じゃ) | じゃ ja (zya, jya) | じゅ ju (zyu, jyu) | じょ jo (zyo, jyo) |
| `by` | BY (びゃ) | びゃ bya | びゅ byu | びょ byo |
| `py` | PY (ぴゃ) | ぴゃ pya | ぴゅ pyu | ぴょ pyo |

Hepburn romaji is canonical and is what the UI shows. Common alternatives are also
accepted: si, ti, tu, hu, zi, di, du, sya, cya, jya, nn (and the others shown in
parentheses above). じ/ぢ (ji) and ず/づ (zu) share an answer; typing di or du
matches the ぢ/づ tile unambiguously.

Katakana mirror the hiragana tables with the same romaji and alternatives, e.g.
カ = ka, シャ = sha.

## Architecture

### Character Data (`kanacore/`)

Character data and row definitions live in `kanacore/`:

- `kana.go`: `Kana` struct, `CharacterSet`, `NewCharacterSet`, `Matches` (canonical romaji plus accepted alternatives)
- `kana_rows.go`: groups, rows with entries and alternative romaji, `ProgressionStepsFor`; scripts (`Script`, `Scripts`), katakana rows generated from the hiragana rows by a Unicode shift, `kata:`-prefixed row IDs

### Desktop App (`fyne/`)

Built on [Fyne v2](https://fyne.io) with goroutine-based game loop:

- `main.go`: Entry point, opens store, runs window
- `app.go`: Window construction, event watcher goroutine
- `game.go`: `GameState` — tick/spawn goroutines, answer checking, stats, auto-progression
- `canvas.go`: `GameCanvas` widget, atomic snapshot renderer (avoids mutex/render-thread deadlock)
- `tile.go`: `KanaTile` — shadow + face + text canvas objects
- `intro.go`: Pending dialogs (katakana offer and row intros) through one entry point, `showPendingDialogs`
- `stats.go`: `StatsPanel` widget — a session line, one learning-path line per active script, and a side-by-side hiragana/katakana kana grid of each character's learning state
- `progress.go`: computes each script's learning-path status (`PathStatus`) and total correct counts per character
- `statswidgets.go`: the fixed-column grid layout and the kana cell widget used by the stats panel
- `input.go`: `InputBar` — score, miss count, text entry
- `settings.go`: In-game settings dialog with scripts and row selection (one tab per script, per-group "all" toggle)
- `theme.go`: `KanaTheme` — warm paper colour palette

### Persistence (`store/`)

- `store/store.go`: SQLite persistence, including active scripts and the katakana offer flag

### Game Timing
- Tick loop: 100ms
- Spawn interval: 4 seconds
- Fall time: each tile takes 16–24 s (randomized) to cross the canvas, independent of window height

## Data Persistence

The app stores `kana.db` (SQLite) in the user's data directory: `$XDG_DATA_HOME/kana/kana.db`,
or `~/.local/share/kana/kana.db` if `XDG_DATA_HOME` is unset, on every platform (Linux, macOS
and Windows alike). Set `XDG_DATA_HOME` to override the location.

It holds:

- Selected hiragana and katakana rows
- Active scripts and whether katakana has been offered
- Auto-progression setting
- Score limit preference
- Per-character statistics (correct count, miss count, current streak)

The database is created automatically on first run. If a `kana.db` from an older version is
found in the working directory, it is moved to the new location once on startup.

## Dependencies

- [Fyne v2](https://fyne.io) — Desktop GUI framework
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite driver

## Development

```bash
# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Build the desktop app
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
