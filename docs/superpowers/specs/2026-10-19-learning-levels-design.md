# Learning levels and progress reset

Date: 2026-10-19. Status: design approved in conversation, spec for review.

## Goal

Make "learned" mean more than three hits ever, let mistakes cost progress,
and let the learner start a script over.

- Every kana and every word has a **level 0–5**. A hit raises it by one, a
  mistake lowers it by two. **Learned** means level 5; it replaces the rule
  "3 or more correct answers" everywhere.
- A wrong answer counts as a mistake when it is unambiguous: exactly one tile
  is on the field.
- A kana that slipped keeps its **red frame** in the status panel across
  sessions until it is learned again.
- The Kana tab of the settings gets one **reset button per script**.

## Decisions

| Question | Decision |
| --- | --- |
| When is a kana or word learned? | At level 5. A hit is +1 (at most 5), a mistake −2 (at least 0). |
| Which wrong inputs count? | Only with exactly one tile on the field: that tile gets the mistake. With several tiles a wrong input has no effect, as today. |
| Mistakes on words | Count for the word only, not for its kana (which kana was misread is unknown). |
| What drops when a learned item falls below 5? | Unlocked rows and the katakana offer stay. A word with a kana below 5 pauses (does not fall) until the kana is learned again. |
| Migration | Items with ≥ 3 correct start at level 5, others at their correct count (0–2). Nothing the learner sees changes on update. |
| Status panel | Three colours stay: level 0 new, 1–4 learning (yellow), 5 learned (green). No level dots for now (the panel is full). The red frame shows a persistent flag: set on every mistake, cleared on reaching level 5. |
| Fall speed | By the same three bands: level 0 slow (×1.2), 1–4 normal, 5 fast (×0.7). |
| "Already known" (word packages) | Sets the word to level 5 and clears its flag. |
| Reset scope | Per script, kana only: stats, level, flag and memory state of every character of the script (extended katakana included). |
| Reset rows | The script restarts its learning path: first progression step only, auto-progression on, the intro of that step is shown again. The script stays active. |
| Reset timing | At once after a confirmation inside the app; independent of Save/Cancel. The settings dialog closes and the round restarts without the game-over dialog. |
| Reset leaves | The other script, the katakana offer (it does not come again) and all word data (introduced words, word stats, levels, packages). |

## Level model

### Storage

Levels are absolute state, not counters, so they cannot be merged the way
session stats are (stored baseline + session delta). They live in a new table
next to `memory`, keyed the same way:

```sql
CREATE TABLE progress (
    item    TEXT PRIMARY KEY,  -- "kana:<char>" or "word:<reading>"
    level   INTEGER NOT NULL,  -- 0..5
    flagged INTEGER NOT NULL   -- 1 after a mistake, until level 5
)
```

`kana_stats` and `word_stats` stay as they are (correct, miss and streak
counts as statistics; the status panel's session numbers keep using them).

Store API, mirroring the memory functions:

- `Progress(prefix string) (map[string]Progress, error)`, with
  `type Progress struct { Level int; Flagged bool }`.
- `SaveProgress(item string, p Progress) error` (upsert).

### Migration

`ensureProgress`, run from `migrate` after `ensureMemory` and
`ensureIntroduced`, follows their pattern: one transaction, skipped when the
table exists, nothing committed on error (the next open retries). It creates
the table and seeds it:

- from `kana_stats` as `kana:<char>`, from `word_stats` as `word:<kana>`;
- `level = 5` if `correct_count >= 3`, else `correct_count`;
- `flagged = miss_count > 0 AND level < 5`.

Items without a stats row have no progress row; a missing row means level 0,
not flagged.

### In the game

`GameState` keeps `levels map[string]store.Progress` for kana and
`wordLevels map[string]store.Progress` for words, loaded at start like the
memory states, with dirty sets saved in `mergeSessionStats` next to
`saveMemory` (a failed save stays dirty and is retried; if loading failed,
nothing of that kind is written, as for memory).

- **Hit** (`recordCorrect`, `recordWordCorrect`): level +1, at most 5. On
  reaching 5 the flag is cleared and, for a kana, the playable-word cache is
  invalidated (replacing the current `== secureCorrect` check).
- **Mistake** (`recordMiss`, `recordWordMiss`, and the new wrong-input case):
  level −2, at least 0; flag set. For a kana whose level drops from 5, the
  playable-word cache is invalidated.
- A helper `learned(level int) bool` (`level >= learnedLevel`, with
  `const learnedLevel = 5`) replaces `secureCorrect` and every `>= 3` check:
  - `isRowMastered`: 80 % of the row's characters learned;
  - `unitsKnown` / playable words: every unit learned;
  - `newestPackageDone` and the words status line: words learned;
  - `stateFor` and `charFallFactor` take the level: 0 new, 1–4 learning,
    5 learned.
- **"Already known"** sets the word's level to 5 and clears its flag, marked
  dirty and saved at once. This replaces the current boost of the correct
  count (`knownDirty`, `boostKnownDirty`, the `max(…, secureCorrect)` in
  `mergeWordStats`): the word stats keep their real counts, and a failed save
  is retried through the progress dirty set.

### Wrong input

In `checkAnswer`, when no tile matches, the input is not blank, and exactly
one tile (kana or word) is on the field, that tile's item gets a mistake:
session miss count, memory `Miss`, level −2, flag. The tile stays on the
field, so the learner can try again. A wrong input does **not** count towards
the miss limit, does not lower the tempo and shows no feedback text; those
stay tied to tiles falling through. If the tile later falls through, that is
a second mistake.

### Status panel

The kana grid cell gets its colour from the level and its red frame from the
flag (today it comes from this session's missed tiles). The game-over dialog
keeps listing the tiles missed this round.

## Reset per script

### UI

The Kana tab gets, below the auto-progression check, one button per script:
"Reset hiragana progress…" and "Reset katakana progress…". A click opens a
confirmation dialog: "Delete the statistics and learning state of all
hiragana? This cannot be undone." with "Reset" and "Cancel". On "Reset" the
reset runs at once; on success the settings dialog closes (unsaved changes in
the other tabs are discarded, as with Cancel) and the round restarts. On
failure an error dialog shows the store error and nothing changes.

### Store

`ResetKana(chars []string) error` deletes, in one transaction, the
`kana_stats` rows, the `memory` rows `kana:<char>` and the `progress` rows
`kana:<char>` of the given characters.

### Game

`GameState.ResetScript(s kanacore.Script) error`, under the lock:

1. Merge pending session data (`mergeSessionStats`), so the reset cannot be
   undone by a later merge of older session data.
2. `store.ResetKana` for every character of the script
   (`kanacore` characters of all rows of the script). On error return it and
   change nothing.
3. Remove those characters from `overallStats`, `sessionStats`,
   `currentStreak`, `memory`, `levels` and their dirty sets; invalidate the
   playable-word cache.
4. Deselect every row of the script, select its first progression step,
   keep the script active, turn auto-progression on, save rows and setting.
5. Queue the intro of the first step (`pendingIntro`).

The caller then restarts the round through the existing `Reset` path, which
shows the pending intro. Session score and misses start at zero; there is no
game-over dialog.

## Error handling

- Migration: one transaction; on error the app starts without the table being
  created and the next start retries (as for memory and introduced words).
- Progress load failure: levels of that kind are treated as 0 for the session
  and not written back, so stored levels are never overwritten by fresh ones.
- Progress save failure: stays dirty, retried on the next merge.
- Reset failure: nothing in memory changes; the error is shown.

## Testing

- **store:** migration seeding (≥ 3 correct → 5, else the count; flag only
  with misses below 5; words too), migration skipped when the table exists,
  `Progress`/`SaveProgress` round trip, `ResetKana` removes exactly the given
  characters from all three tables and leaves the rest.
- **levels:** hit +1 capped at 5, mistake −2 floored at 0, flag set on a
  mistake and cleared at 5; dropping from 5 makes a word unplayable and
  reaching 5 again makes it playable; unlocked rows stay after a drop.
- **wrong input:** one kana tile → mistake, tile stays, miss limit and tempo
  unchanged; one word tile → word mistake only; several tiles → no effect;
  blank input → no effect.
- **"Already known":** level 5, flag cleared, saved; retry after a failed
  save.
- **status panel:** colour by level band, red frame from the flag across a
  restart.
- **reset:** both scripts, the other one untouched; rows back to the first
  step with its intro; words with the script's kana pause; confirmation
  dialog: Cancel changes nothing, Reset closes the settings and restarts the
  round; a failing store leaves everything as it was.

## Later

- Recognising systematic confusions (ぎ shown, "ki" typed; シ/ツ) to attribute
  wrong inputs with several tiles on the field.
- Level dots in the status panel.
- A reset for words.
