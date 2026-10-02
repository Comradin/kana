# Design: Pause

**Date:** 2026-10-02
**Status:** Approved (design discussion)
**Builds on:** `2026-10-01-katakana-design.md` (pause invariant, `settle()`, `showPendingDialogs`)

## Overview

The learner can pause the game with a button next to the gear or with **Esc**. While paused,
the falling tiles are hidden and a "Paused" hint is shown, so the pause cannot be used to study
the tiles. The settings dialog also pauses the game while it is open.

## Decisions

| Topic | Decision |
|---|---|
| Controls | A pause/play icon button left of the gear; Esc toggles too, also while the romaji entry has focus. |
| What pauses | Ticking, spawning and answers, as for dialogs. |
| Tiles while paused | Hidden only for a user pause and an open settings dialog; a centred "Paused — press Esc to continue" hint is shown for a user pause. Intro and offer dialogs keep the tiles visible behind the dialog, as today. |
| Settings | Opening the settings dialog pauses; closing it (Save or Cancel) releases that pause. |
| Invariant | The pause invariant gains two reasons. `paused` is true while an intro or offer is pending, the user paused, or the settings dialog is open. `settle()` unpauses only when none holds. Closing an intro while the user has paused does not resume the game. |

## Game state (`fyne/game.go`)

- New fields `userPaused bool` and `settingsOpen bool`.
- `settle()` unpauses only if nothing is pending **and** neither `userPaused` nor `settingsOpen`.
- `TogglePause() bool` (locked): flips `userPaused`. Turning it on sets `paused = true`; turning
  it off calls `settle()`. It rebuilds the tile snapshot and returns the new `userPaused`.
  While the game is over it does nothing and returns false.
- `SetSettingsOpen(open bool)` (locked): sets `settingsOpen`, pauses when opening, calls
  `settle()` when closing, rebuilds the snapshot.
- `IsUserPaused() bool` (locked read) for the UI.
- `buildSnapshot` publishes no tile objects while `userPaused || settingsOpen`, and records that
  state in an atomic flag the canvas renderer reads to draw the hint.
- `Reset` clears `userPaused` and `settingsOpen` before deciding `paused`.

## UI

- `InputBar` gets a pause button (`theme.MediaPauseIcon` / `theme.MediaPlayIcon`) left of the
  gear and a `SetPaused(bool)` method that swaps the icon.
- The romaji entry becomes a small `widget.Entry` wrapper whose `TypedKey` handles
  `fyne.KeyEscape` by calling the toggle and passes every other key to the embedded entry. The
  window canvas also handles Esc via `SetOnTypedKey` when nothing has focus.
- Toggling refreshes the canvas and the pause icon. The entry stays enabled during a user pause
  so Esc keeps working; answers are ignored while paused anyway.
- `GameCanvas` renders a centred "Paused — press Esc to continue" text while the user pause is
  active.
- `showSettingsDialog` calls `SetSettingsOpen(true)` before showing and, as the **first** thing in
  its response callback (Save and Cancel), `SetSettingsOpen(false)`; the existing
  `showPendingDialogs` call then resumes or shows a pending dialog. Cancel also calls
  `showPendingDialogs`.

## Testing

- `TogglePause` pauses, hides tiles (empty snapshot), and a second toggle resumes and shows them.
- While user-paused: `tick`, `spawnKana` and `checkAnswer` do nothing; `FinishIntro`,
  `DeclineKatakana` and `Resume` keep the game paused.
- `SetSettingsOpen(true)` pauses and hides tiles; `SetSettingsOpen(false)` resumes, unless the
  user pause or a pending dialog still holds.
- `TogglePause` does nothing after game over.
- `Reset` clears both new reasons.
- Esc on the entry wrapper calls the toggle; other keys still reach the entry.

## Out of Scope

- Pausing automatically when the window loses focus.
