# Kana

A typing game for learning Japanese Hiragana and Katakana characters.

## Overview

Kana is an interactive typing game that helps you learn Japanese Hiragana and Katakana through practice. Characters fall from the top of the screen, and you type their romaji (romanized) equivalents to score points. Track your progress, focus on specific character rows, and improve your kana recognition skills.

## Apps

Kana is a Fyne v2 desktop app (`kana-desktop`).

## Features

### Core Gameplay
- **104 Hiragana and 127 Katakana (basic, Dakuon, Handakuon, Yōon, plus 23 extended katakana)**: Practice the full practical hiragana and katakana sets, from あ/ア (a) to the combined Yōon with (han)dakuten, and the extended katakana used in loanwords (ティ, ファ, ヴ, …)
- **Falling Character Mechanic**: Characters spawn at the top and fall for a time that depends
  on the chosen difficulty level, a per-character factor and the session tempo (see
  Difficulty below), independent of window height
- **Romaji Input**: Type the romanized equivalent and press Enter to score
- **Words**: Japanese words fall among the kana tiles (green border), introduced in small
  packages like kana rows. As soon as a word of your source can be read (every kana in it is
  secure, answered correctly three times, in an active script), a package of 3, 5 or 10 words
  is shown first: each word with its romaji and meaning, and an "Already known" box that
  counts the word as learned and brings in another one. Only introduced words fall. The next
  package comes when 80 % of the newest one is secure. New words come from one of your own
  lists (by lesson, then file order; see [Private word lists](#private-word-lists)) or from the
  bundled JLPT words (N5 first, the most common words first); once a JLPT level is fully
  introduced, the game offers the next one. Type the whole reading (`sensei`, `kitte`,
  `koohii`, `ko-hi-`; spaces are ignored, so `shitsurei desu ga` works) or its meaning in the
  meaning language or in English (`Regenschirm` or `umbrella` for かさ; case, text in
  parentheses, a leading `to` or article and `ae`/`oe`/`ue`/`ss` for umlauts do not matter);
  a hit shows the meaning (and the romaji when answered by meaning), a missed word shows its
  romaji and meaning. One spawn in a hundred is a word per
  playable word, at most 10 %; longer words fall more slowly (up to twice the time).
- **Words only**: Settings → Words → Tiles "Words only" lets only words fall: every word you
  were introduced to (of the chosen source and lessons, in an active script) falls, whether or
  not you have learned its kana, and new word packages come the same way. The kana learning
  path waits meanwhile (no new rows, no katakana offer) and goes on when you switch back. If no
  word can fall (no word in the chosen lessons, list missing), kana fall and the status panel
  says why.
- **Meaning mode**: Settings → Words → "Words ask for: Meaning" makes word tiles ask for the
  word's meaning instead of its reading. They carry a small "?" in the corner, so a one-kana
  word such as き is not taken for a kana tile; kana tiles still ask for the romaji. Type a
  meaning in the meaning language or in English, as for any word; typing the reading instead
  shows "type the meaning" at the tile and costs nothing. Every word has a reading level and a
  meaning level of its own: in meaning mode only meanings count and a missed tile lowers its
  meaning level; in reading mode the reading raises the reading level, and a meaning is still
  accepted and raises the meaning level. In meaning mode the choice of tiles, "Focus on weak
  items", the status line ("Meanings: 12 introduced · 4 learned · …") and when the next word
  package comes go by the meaning levels, and words without a meaning do not fall. Together
  with "Words only" it is a pure meaning drill. A change applies to new tiles; tiles already
  falling keep their question. Progress from before counts as reading progress; all meaning
  levels start at 0, and "Already known" in a package intro counts for the reading only.
- **Learning path**: start with あいうえお, unlock two rows at a time; each automatically unlocked row is introduced before it is asked, and the game pauses while new rows are introduced
- **Two scripts**: hiragana and katakana progress independently; switch scripts in the settings. After the basic hiragana you are offered katakana once.
- **Score Limit Mode**: Set a target score or 0 for endless practice
- **Miss Limit**: Game ends when the chosen level's miss limit is reached (5–10, see Difficulty below)

### Difficulty

- **Levels**: choose Relaxed, Normal, Fast or Pro in the settings. A level sets the fall-time
  range, how often a new tile spawns, and the miss limit. Unset or unknown stored levels fall
  back to Normal, which matches the original game exactly.

  | Level | Fall time | New tile every | Misses allowed | Auto points factor |
  |---|---|---|---|---|
  | Relaxed | 18–26 s | 4.5 s | 10 | ×1 |
  | Normal | 16–24 s | 4.0 s | 10 | ×1 |
  | Fast | 10–15 s | 2.5 s | 7 | ×2 |
  | Pro | 6–9 s | 1.6 s | 5 | ×3 |

- **Per-character factor**: a newly spawned tile's fall time is also scaled by how well you
  know that character — new characters get more time (×1.2), learning characters are
  unchanged (×1.0), and mastered characters fall faster (×0.7).
- **Session tempo**: five correct answers in a row speed the game up one tempo step (fall
  time and spawn interval both shrink); a miss resets the streak and costs two tempo steps,
  down to a floor at step 0. The input bar shows the resulting speed-up, e.g. `Tempo ×1.17`,
  capped at `×1.67`.
- **Points per hit**: the settings' "Points per hit" field sets how many points a correct
  answer is worth — Auto (by level, ×1/×2/×3 for Relaxed–Normal/Fast/Pro), or a fixed ×1, ×2,
  ×3 or ×5. A correct answer scores `10 × factor`.

### Spaced repetition

- **Forgetting-curve model**: each character has a memory state (stability, last seen) that
  estimates the chance you'd still recall it right now. A correct answer grows that stability;
  a miss shrinks it and treats the character as half forgotten, so it comes back soon — a
  just-missed kana is one of the most likely tiles to spawn next.
- **Weighted spawning**: new tiles are chosen by that estimate. Likely-forgotten characters
  spawn often, characters you know well spawn rarely but never stop appearing entirely (their
  weight never drops below a tenth of a just-missed character's), and brand-new characters make
  up at most 40% of spawns. A character already falling on screen is not spawned again while
  it's still there, unless it is the only choice.
- **Focus on weak items** (Settings → Game): half the tiles are drawn from what you are
  learning (levels 1–4) or slipped, slipped ones twice as often; the other half and all new
  items come as usual. It does not change how often a word falls.

### Progress Tracking
- **Persistent Statistics**: Progress is saved to a local SQLite database (`kana.db`)
- **Per-Character Stats**: Correct answers, misses, and current streak per hiragana and katakana character
- **Session vs Overall Stats**: The stats panel shows a session line (correct, missed, accuracy), a learning-path line per active script and a kana grid of each character's learning state; overall stats are persisted across sessions and drive progression

### Customization
- **Row Selection**: Choose which hiragana and katakana rows to practice, grouped by Basic, Dakuon, Handakuon, Yōon, Yōon + Dakuten and (katakana only) Extended katakana
- **Auto-Progression**: per script, unlocks the next learning-path step (one or two rows) once every active row of that script is mastered: at least 80% of its kana (rounded down, at least one) answered correctly three times
- **Configurable Score Limit**: Set a target or play endlessly
- **Word packages**: where new words come from (a private list or JLPT), the JLPT level, the
  lessons of a list (checkboxes with "All lessons" and "No lesson", stored per list: only these
  lessons' words fall and new packages come from the earliest chosen lesson), the package size
  (3, 5 or 10), what word tiles ask for ("Words ask for": Reading or Meaning) and the meaning
  language ("Meanings in": System, or any language of the word data; English answers are
  always accepted)
- **Kana font**: Noto Sans JP (default), M PLUS Rounded 1c or the system font, for tiles,
  intros, the kana grid and the word labels; both fonts ship with the app

### Desktop App (Fyne)
- Warm paper tile aesthetic — stamp-style kana tiles on a parchment background
- Split layout: game canvas on the left, stats panel on the right; window starts at 1000 × 760
- Stats panel: a session line, one learning-path line per active script, a side-by-side hiragana/katakana kana grid (new/learning/mastered, with a red border on a character missed this session), a words line (introduced and secure words and their source, or what to do for new words)
- In-game settings via gear icon (no pre-game setup form), in the tabs Kana, Words, Game and
  Sources (version and the credits for the word data and fonts)
- Game-over dialog with missed character and word review and Play Again

## Download

Ready-made builds are on the [latest release](https://github.com/Comradin/kana/releases/latest).
Pick the file for your system:

| System | File |
|---|---|
| macOS (Apple Silicon) | `kana-vX.Y.Z-macos-arm64.dmg` |
| Windows (x64) | `kana-vX.Y.Z-windows-x64.zip` |
| Linux (x64, glibc 2.35+, e.g. Ubuntu 22.04 or newer) | `kana-vX.Y.Z-linux-x64.tar.xz` |

The builds are not signed, so your system warns about them on first start:

- **macOS:** open the `.dmg` and drag Kana to Applications. On first start macOS blocks it;
  open System Settings → Privacy & Security and click "Open Anyway" (on older macOS:
  right-click the app → Open). Alternatively run
  `xattr -dr com.apple.quarantine /Applications/Kana.app`.
- **Windows:** unzip and start `Kana.exe`. At the SmartScreen warning click "More info" →
  "Run anyway".
- **Linux:** unpack and install for your user:

  ```bash
  tar xf kana-vX.Y.Z-linux-x64.tar.xz
  cd kana && make user-install
  ```

  Or run `kana/usr/local/bin/kana` directly without installing.

Each file has a `.sha256` checksum next to it. To verify a download, put both in the same
folder and run `sha256sum -c <file>.sha256` (macOS: `shasum -a 256 -c <file>.sha256`).

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
2. Jump straight into the game; open the gear icon to adjust rows, auto-progression, where new words come from, difficulty level, points per hit, score limit and the kana font
3. As characters fall, type their romaji equivalent and press Enter
4. Each correct kana scores 10 points (a word 10 per kana unit), scaled by the points setting (Auto by level, or a fixed ×1/×2/×3/×5)
5. The game ends when you reach your score target, miss as many tiles as the level allows (5–10), or quit

### Controls

- **Type + Enter**: Submit answer
- **Gear icon**: Open settings during a game, including difficulty level and points per hit
- **Game-over dialog**: Play Again or Quit

### Scoring
- **+10 points** per correct kana and per kana unit of a correct word, times the points factor (Auto uses the level's factor: ×1 Relaxed/Normal, ×2 Fast, ×3 Pro)
- Session ends on: target score reached, the level's miss limit reached, or manual quit

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
selects the ぢ/づ tile rather than じ/ず. With extended katakana active, di, du, ti,
tu and wo are shared with ディ, ドゥ, ティ, トゥ and ウォ, and the lowest matching tile
wins; thi, dhi, twu, dwu and who always target the extended tile.

Katakana mirror the hiragana tables with the same romaji and alternatives, e.g.
カ = ka, シャ = sha.

### Extended katakana (23, katakana only)

| Row ID | Label | Characters |
|---|---|---|
| `kata:ext-t` | T/D (ティ) | ティ ti (thi), ディ di (dhi), トゥ tu (twu), ドゥ du (dwu) |
| `kata:ext-f` | F (ファ) | ファ fa, フィ fi, フェ fe, フォ fo |
| `kata:ext-w` | W (ウィ) | ウィ wi (whi), ウェ we (whe), ウォ wo (who) |
| `kata:ext-sh` | SH/J/CH (シェ) | シェ she (sye), ジェ je (zye, jye), チェ che (tye, cye) |
| `kata:ext-v` | V (ヴ) | ヴァ va, ヴィ vi, ヴ vu, ヴェ ve, ヴォ vo |
| `kata:ext-ts` | TS (ツァ) | ツァ tsa, ツィ tsi, ツェ tse, ツォ tso |

They are the last three steps of the katakana learning path, after Yōon + Dakuten:
ティ + ファ, ウィ + シェ, ヴ + ツァ. In the stats panel they form a katakana-only block in
entry order.

ti, tu, di, du and wo also answer チ, ツ, ヂ, ヅ and ヲ; when both are falling, the lowest
matching tile is taken, so use thi, twu, dhi, dwu or who to aim at the extended tile.

## Architecture

### Character Data (`kanacore/`)

Character data and row definitions live in `kanacore/`:

- `kana.go`: `Kana` struct, `CharacterSet`, `NewCharacterSet`, `Matches` (canonical romaji plus accepted alternatives)
- `kana_rows.go`: groups (`Groups`, `GroupsFor`), rows with entries and alternative romaji, `ProgressionStepsFor`; scripts (`Script`, `Scripts`), katakana rows generated from the hiragana rows by a Unicode shift plus the katakana-only extended rows, `kata:`-prefixed row IDs
- `words.go`: splits and matches kana words for the vocabulary trainer — `KanaUnits` (split a word into kana units), `ReadingMatches` (accept every allowed romaji spelling, including っ, ー and ん variants; whitespace is ignored) and `Romaji` (the canonical spelling)

### Word Lists (`vocab/`)

Word data for the vocabulary trainer, independent of the game and the store:

- `vocab.go`: `Word`, `Meanings` (per language, with fallback for display), `List`
- `csvlist.go`, `jsonlist.go`, `dir.go`: private word lists (CSV/TSV/JSON, with header aliases) and `LoadDir`
- `notation.go`: the notation of a kana field (placeholders, optional parts, punctuation,
  variants): reading, shown form and other accepted readings, or why the row is skipped
- `import.go`: `Inspect` and `Import`, checking and storing a word list for the settings' import;
  `InspectAll` keeps every skipped row's message for the check command (for a file without
  words in the `ImportError`'s `List`)
- `manage.go`: `DeleteList`, `CheckListName` and `RenameList` for the word list dialog
- `meaning.go`: `MeaningMatches`, answering a word by its meaning
- `index.go`: `NewIndex` (one entry per reading across all sources, with each list entry's first
  row, its shown form and its other accepted readings; `Entry.ReadingMatches`) and `Playable` (words made only of known kana, filtered by JLPT level, list and lesson)
- `package.go`: `Source` and `NextPackage` (the next words to introduce from a list or JLPT, in
  learning order)
- `bundled.go`: `Bundled`, the embedded JLPT word list (`data/words.json.gz`, with a frequency
  rank per word)

### Word Data Generator (`cmd/vocabgen/`)

A maintainer tool that builds `vocab/data/words.json.gz` from JMdict and the JLPT lists
(see [Word data](#word-data)).

### List Checker (`cmd/listcheck/`)

A command that checks word list files as the import does, without importing them, and prints
every skipped row (see [Private word lists](#private-word-lists)).

### Spaced Repetition (`memory/`)

A pure package modelling the forgetting curve, independent of the game and the store:

- `memory.go`: `State` (stability, last seen), `Retrievability` (estimated recall chance),
  `Correct` and `Miss` (state updates — a miss leaves `R = 0.5`), `Weight` (pick strength, with
  a floor), `Seed` (starting state from a past correct count) and `Pick` (two-stage weighted
  choice: at most `NewCap` share new, otherwise weighted by `Weight`)

### Desktop App (`fyne/`)

Built on [Fyne v2](https://fyne.io) with goroutine-based game loop:

- `main.go`: Entry point, opens store, runs window
- `app.go`: Window construction, event watcher goroutine
- `game.go`: `GameState` — tick/spawn goroutines, answer checking, stats, auto-progression, spaced-repetition memory updates and weighted spawning
- `difficulty.go`: the level table (`Level`, `LevelInfo`, `Levels`, `LevelByName`), the points
  setting (`pointsChoices`, `pointsLabel`, `validPoints`), and the pure helpers for the
  per-character fall factor, the session tempo factor, the input bar's status line and the
  game-over reason text
- `canvas.go`: `GameCanvas` widget, atomic snapshot renderer (avoids mutex/render-thread deadlock)
- `tile.go`: `KanaTile` — shadow + face + text canvas objects, for a kana or a word; a meaning tile adds its "?" marker
- `wordkind.go`: reading and meaning as two kinds of word progress (`wordKind`, `track`), which answer counts for which, and the words a meaning tile may ask about
- `words.go`: word helpers — meaning language, missed-word and feedback texts, the words line
- `wordgame.go`: word progress (stats, memory, merge), the playable-word rule and cache, word spawning
- `feedback.go`: the labels shown after a word hit or miss, and the "type the meaning" hint
- `focus.go`: the focus pick on weak items
- `packages.go`: word packages — when the next one is due, finishing it ("already known" words
  count as secure), replacement words, the JLPT level offer
- `packageintro.go`: the word package intro and the level offer dialogs
- `fonts.go`: the bundled kana fonts (`fonts/`), the font choice and the glyph coverage check
- `intro.go`: Pending dialogs (row intros, katakana offer, word packages, level offer) through one entry point, `showPendingDialogs`
- `stats.go`: `StatsPanel` widget — a session line, the words line (introduced and secure words, or a hint), one learning-path line per active script and a side-by-side hiragana/katakana kana grid of each character's learning state
- `progress.go`: computes each script's learning-path status (`PathStatus`) and total correct counts per character
- `statswidgets.go`: the fixed-column grid layout and the kana cell widget used by the stats panel
- `input.go`: `InputBar` — score, level, tempo, miss count, text entry
- `settings.go`: In-game settings dialog with the tabs Kana (scripts and row selection, one tab per script, per-group "all" toggle, auto-progression), Words (tiles, words ask for, source, lessons, JLPT level, package size, meaning language), Game (difficulty level, points per hit, focus on weak items, score limit, kana font) and Sources
- `lists.go`, `listsdialog.go`: the word lists in the game (reload, delete, rename) and the Manage word lists dialog
- `drop.go`: importing files dropped on the window
- `version.go`: the version label and the Sources tab
- `theme.go`: `KanaTheme` — warm paper colour palette

### Persistence (`store/`)

- `store/store.go`: SQLite persistence, including active scripts, the katakana offer flag, the chosen difficulty level, the points setting and per-character spaced-repetition memory states (seeded once from past correct counts), per-word statistics (reading and meaning apart), the introduced words and the word settings (`word_level`, `word_source`, `package_size`, `newest_package`, `meaning_lang`, `words_only`, `focus_weak`, `words_ask`, `lesson_filter:<list>`) and the kana font (`kana_font`)

### Game Timing
- Tick loop: 100ms
- Spawn interval: depends on the chosen level (4.5 s Relaxed, 4.0 s Normal, 2.5 s Fast, 1.6 s
  Pro), scaled by the session tempo
- Fall time: each tile's base range also depends on the level (e.g. 16–24 s Normal, 6–9 s
  Pro), scaled by the per-character factor and the session tempo; independent of window height

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
- Per-character spaced-repetition memory state (`memory` table), used to choose which kana
  spawn next
- Per-word statistics (`word_stats`: correct and miss count per reading) and per-word
  spaced-repetition memory (`word:` items in the `memory` table), both for typing the reading;
  the same for typing the meaning in `meaning_stats` and as `meaning:` items in the `memory`
  table. Learning levels and slipped flags are `kana:`, `word:` and `meaning:` items in the
  `progress` table
- The words shown in a package intro (`introduced_words`; when the table is created, every
  word with stats or memory counts as introduced) and the newest package (`newest_package`)
- Where new words come from (`word_source`: `jlpt` or `list:<name>`; chosen once on the first
  start: your first list, else JLPT), the JLPT level packages come down to (`word_level`,
  default N5), the package size (`package_size`, default 5) and the meaning language
  (`meaning_lang`; unset means the system language if the word list has it, else English)
- What word tiles ask for (`words_ask`: `reading`, the default, or `meaning`)
- The kana font (`kana_font`: `noto`, `mplus` or `system`)

The database is created automatically on first run. If a `kana.db` from an older version is
found in the working directory, it is moved to the new location once on startup.

## Word data

The vocabulary trainer ships with a list of about 7,000 JLPT words
(`vocab/data/words.json.gz`). It is built from:

- **JMdict** by the [Electronic Dictionary Research and Development Group](https://www.edrdg.org/)
  (meanings in nine languages), CC BY-SA 4.0
- **JLPT levels** from [Jonathan Waller's lists](http://www.tanos.co.uk/jlpt/), CC BY
  (unofficial; there are no official JLPT vocabulary lists since 2010)
- the **JMdict sequence mapping** of
  [stephenmk/yomitan-jlpt-vocab](https://github.com/stephenmk/yomitan-jlpt-vocab), CC BY-SA 4.0

The data file is licensed under CC BY-SA 4.0; the code stays MIT. Details, versions and how to
regenerate it: [`vocab/data/ATTRIBUTION.md`](vocab/data/ATTRIBUTION.md).

The game credits these sources on the settings' Sources tab.

### Private word lists

Add your own lists (for example from a textbook) with **Settings → Words → Import word
list…**. The game checks the file, shows a preview (words, lessons, meaning languages, skipped
rows, the first words) and copies it into the `words` folder next to `kana.db`:
`$XDG_DATA_HOME/kana/words/`, or `~/.local/share/kana/words/` (created at the first start; the
Words tab shows where). You can also drop files anywhere on the game window: each gets the same
preview, one after another, and the game pauses meanwhile. Importing a file with the name of an
existing list replaces that list; your progress is kept. The file must be UTF-8 ("CSV UTF-8" in
spreadsheet programs) or UTF-16 with a byte order mark (Excel's "Unicode Text", a `.txt` file,
stored as `.tsv`); `.xlsx` and `.ods` sheets must be saved as CSV first; at most 5 MB. Files can
also be copied into the folder or edited there by hand; they are read each time the settings
open. Lists are read from there only and never belong in the repository.

**Settings → Words → Manage word lists…** lists every word list with its number of words and
lessons. A list with rows the game could not read says "N rows skipped"; **Details** shows the
lines and why (at most 100), so you can correct your source file and import it again.
**Rename** renames the list's file (the extension stays); its lesson choice and its place as the
source of new words go with it. The new name must not contain `/ \ < > : " | ? *`, start with
".", be `jmdict` or a name Windows keeps for devices (CON, PRN, AUX, NUL, COM1–9, LPT1–9), on
every system, so the folder can move between computers. **Delete** removes the file for good after asking; your progress
on its words is kept, and if it was the source of new words, JLPT takes its place. Both act at
once, not on Save. **Open folder** opens the words folder in your file manager. The status
panel's words line adds "· N rows skipped" when the chosen list has skipped rows.

**Checking a file without importing it:** in a clone of this repository,
`go run ./cmd/listcheck FILE...` reads each file as the import does and changes nothing. For each
file it prints the word list's name, its number of words and skipped rows, then the message of
every skipped row (all of them, where the game keeps the first 100), or why the file cannot be
imported and what to do about it, followed by every skipped row also then (a file whose rows
are all skipped). With `-words` it also lists every word as shown, with the
romaji of its reading, its other accepted readings (an optional part left out, another variant)
and its lesson. The output can be passed on as it is, for example to whoever keeps the list; the exit
status is 1 when a file cannot be imported.

Choose a list as the source of new words in the settings (Words tab; an imported list is
preselected); its words are introduced lesson by lesson (numeric lessons in number order,
others in the order they first appear), then in file order, with words without a lesson last.
If the chosen list is missing, the status panel says where to add it.

Choose lessons below the source to practise only them (all lessons by default); renaming a list
with Manage word lists keeps its lesson choice; renaming its file by hand forgets it.

- **CSV/TSV** (`*.csv`, `*.tsv`): a header row; the separator (tab, semicolon or comma) is
  taken from it. Columns: `kana` (required), `meaning` (any language), `meaning_de`,
  `meaning_en` … per language, `lesson`, `jlpt` (`5` or `N5`). Several meanings are
  separated by `;`. A field containing the file's separator is quoted, as spreadsheet
  programs do:

  ```csv
  kana;meaning;lesson
  せんせい;"Lehrer; Lehrerin";1
  ```

  Column names are case-insensitive, and some other names are understood: `japanisch`,
  `japanische schrift` or `reading` for `kana`; `deutsch` or `german` for `meaning_de`;
  `englisch` or `english` for `meaning_en`; `lektion` for `lesson`. A column under its own
  name wins over one of these, and any other column (such as a romaji column) is ignored,
  so an export with the header `Japanische Schrift,Japanisch (Romaji),Deutsch,Lektion`
  loads as is. Of two columns with the same name the first is used. Files whose name
  starts with `.` are ignored.
- **JSON** (`*.json`): an array of `{"kana": "せんせい", "meaning": "teacher", "lesson": "1"}`;
  `meaning` may also be a list or an object per language (`{"de": "Lehrer", "en": "teacher"}`).

**Notation in the kana column** (CSV and JSON alike), for course lists that write more than
plain kana:

- `～` (also `〜` or `~`) stands for something you fill in, such as a name: `～さん`,
  `こちらは～さんです`. The tile shows it; you type only the kana: `san`.
- Text in brackets, `（…）` or `(…)`, is optional: `（どうぞ）よろしく（おねがいします）` is
  answered with the full form `douzoyoroshikuonegaishimasu` or the short form `yoroshiku`,
  nothing in between. Brackets must pair and must not nest.
- Punctuation (`？ ！ 。 、 ・ ? ! . ,`) and spaces are shown, not typed: `おげんきですか？` is
  `ogenkidesuka`.
- `//` separates two writings of one word: `あめりか // アメリカ`. The tile shows the last one,
  which also decides the script (`アメリカ` needs katakana switched on); every writing's
  reading counts.
- In an entry that uses any of the above except `//` alone (a phrase), は may also be typed
  `wa` and へ `e`, as particles are spoken: `こちらは～さんです` takes `kochirawasandesu`.
  Everywhere, を takes `o` as well as `wo`.

The tile, the feedback, the word package intro, the import preview and the game-over list show
a word as the list writes it, with the romaji of its full form. Progress belongs to the
reading, the kana without the notation: `～さん` in two lists, or `～さん` in a list and `さん` in
the JLPT list, is one word, shown as the first list writes it.

Rows are skipped and reported when their kana field is empty after the notation, contains
anything other than kana (kanji, Latin letters; a variant that is not kana skips the whole
row), has unpaired or nested brackets, or has an empty variant (`アメリカ //`). A short form
that is no word on its own (`いっ（て）`: っ, ー or a small kana at its edge) is not skipped;
only its full form counts.

## Fonts

The kana font can be Noto Sans JP (default) or M PLUS Rounded 1c; both ship with the app in
`fyne/fonts/` (regular and bold, about 16 MB) under the SIL Open Font License 1.1
(`fyne/fonts/OFL-*.txt`, also included in the release packages). Noto Sans JP is
© 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'. `scripts/fetch-fonts.sh` downloads the pinned files (Noto CJK release
`Sans2.004`, google/fonts at a fixed commit) and checks their SHA-256 sums. A text is shown in
the chosen font only if the font has every character of it; otherwise it uses the system font,
so one text never mixes two fonts.

## Dependencies

- [Fyne v2](https://fyne.io) — Desktop GUI framework
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite driver
- [go-text/typesetting](https://github.com/go-text/typesetting) — font parsing for the kana
  font's coverage check (also used by Fyne)

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

## Releasing (maintainers)

Releases are cut by [release-please](https://github.com/googleapis/release-please) from the
conventional commit messages on `main`:

1. Commits land on `main` through PRs (rebase merge, so each commit's type counts). `feat:`
   commits bump the minor version, `fix:` commits the patch version; before 1.0 a breaking
   change also bumps only the minor version. `docs:`, `chore:` and other types alone do not
   start a release.
2. On every push to `main` the `release` workflow opens or updates the release PR
   "chore(main): release 0.x.y" with the `CHANGELOG.md` entry and the new version.
3. Just before merging, rewrite the release notes for players: a few short paragraphs on
   what is new and better, in their words, above a `### Commits` heading that keeps the
   generated list. Put the same text into the PR description (between the `---` lines; the
   GitHub Release takes its notes from there) and into the version's `CHANGELOG.md` entry on
   the release branch. Do it last: any new commit on `main` regenerates the PR. Without a
   rewrite the notes use the section names from `release-please-config.json` ("New",
   "Improvements and fixes").
4. Merging that PR creates the tag `v0.x.y` and the GitHub Release. In the same run the
   Windows and Linux packages and their checksums are built and attached, and the release
   notes get a line saying the macOS build follows.
5. Build and upload the macOS package from an Apple Silicon Mac:

   ```bash
   git fetch --tags && git checkout v0.x.y
   scripts/release-macos.sh v0.x.y
   ```

   Without a tag the script uses the newest release. `scripts/release-macos.sh --no-upload`
   makes a local trial build in `dist/` without touching the release. Prerequisites: Go in the
   version from `go.mod`, the Xcode command-line tools (`xcode-select --install`), and the
   GitHub CLI `gh`, logged in (`gh auth login`).

Repository settings this relies on (set once by hand):

- Settings → Actions → General → "Allow GitHub Actions to create and approve pull requests"
  must be enabled, so release-please can open its PR.
- `main` must have no required status checks: release PRs opened with the default token do
  not trigger CI, so a required check would block them forever.

### Word data updates

The EDRDG licence for JMdict asks for a procedure that regularly updates the data from the
most recent version. `.github/workflows/update-words.yml` runs on the 3rd of every month (and
by hand via "Run workflow"): it downloads the latest `JMdict.gz` and JLPT mapping, regenerates
`vocab/data/words.json.gz` and `vocab/data/SOURCES`, runs the `vocab` tests and, if the words
changed, opens or updates a `fix(vocab): update word data from JMdict` pull request on the
`update-words` branch. Merging it makes release-please cut a patch release, so the new data
reaches users.

Two repository settings matter:

- GitHub disables scheduled workflows in public repositories after 60 days without
  repository activity. If that happens, re-enable `update-words` under Actions or run it by
  hand.
- The workflow opens the pull request with the default token, which needs
  Settings → Actions → General → "Allow GitHub Actions to create and approve pull requests".

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
