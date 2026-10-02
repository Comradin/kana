# Design: Katakana

**Date:** 2026-10-01
**Status:** Approved (design discussion and spec review)
**Builds on:** `2026-09-30-complete-hiragana-design.md` (PR #8, branch `feat/complete-hiragana`)

## Overview

Add katakana as a second script next to hiragana. The katakana set mirrors the 104 hiragana exactly: basic, Dakuon, Handakuon, Yōon and Yōon with (han)dakuten. Each script is its own **track**, with its own selection, learning path and progression. The learner switches scripts on and off in the settings. After the hiragana basic rows are mastered, the app offers katakana once.

Work happens on `feat/katakana`, stacked on `feat/complete-hiragana`. It is rebased onto `main` once PR #8 is merged.

## Decisions

| Topic | Decision |
|---|---|
| Learning path | Two tracks. Katakana starts at ア イ ウ エ オ, independent of hiragana progress, with the same steps and groups. |
| Scope | The 104 mirrors of the hiragana. Extended katakana (ティ, ファ, ヴ …), small ッ and the long mark ー are deferred to the vocabulary-trainer milestone. |
| Activation | A "Scripts" setting (☑ Hiragana ☐ Katakana). New learners start with hiragana only. When all 11 hiragana basic rows are mastered, the app asks once: "Add Katakana?" |
| Mixed play | With both scripts on, tiles of both fall together. か and カ both answer "ka"; the lowest tile wins, as today. |
| Romaji | Identical to the hiragana: canonical Hepburn plus the same alternatives. |

## Character Data (`kanacore`)

### Script and generated rows

```go
type Script string

const (
    ScriptHiragana Script = "hiragana"
    ScriptKatakana Script = "katakana"
)
```

- `KanaRow` gains a `Script` field.
- The existing hiragana rows become `HiraganaRows` and are unchanged apart from `Script: ScriptHiragana`.
- `KatakanaRows` is **generated** from `HiraganaRows` by `toKatakana`:
  - Every rune in U+3041–U+3096 is shifted by +0x60 to the katakana block (か U+304B → カ U+30AB, ゃ → ャ). Other runes stay as they are.
  - The row ID gets the prefix `kata:` (`kata:vowels`, `kata:k`, `kata:ky` …), so stored hiragana IDs keep their meaning.
  - The label is converted the same way: "K-row (か)" becomes "K-row (カ)".
  - Romaji, alternatives and group are copied.
- `AllKanaRows` is `HiraganaRows` followed by `KatakanaRows`: 54 rows, 208 characters. Existing iterations over `AllKanaRows` keep working and now cover both scripts in a stable order.
- `RowsFor(s Script)` returns the rows of one script. `RowsInGroup(s Script, g Group)` replaces `RowsInGroup(g)`.
- `ProgressionStepsFor(s Script) [][]string` replaces the `ProgressionSteps` variable. Katakana steps are the hiragana steps with `kata:` prefixed.
- `Hiragana()` and `Katakana()` return per-script character sets. `AllKana()` returns the combined set used by the game. `CharToRow` covers all 208 characters.
- `ScriptOf(rowID string) Script` derives the script from a row ID (`kata:` prefix → katakana).
- `DefaultRowIDs()` still returns the 11 hiragana basic rows.
- `Groups()` is unchanged: the groups are the same for both scripts.
- `Scripts() []ScriptInfo` returns both scripts with their labels ("Hiragana", "Katakana") in display order.
- API changes to carry through:
  - `BasicRows()` stays hiragana-only.
  - `Hiragana()` is built from `HiraganaRows`, not from `AllKanaRows`.
  - Existing tests that assert 27 rows or use the `ProgressionSteps` variable are updated.

A test checks the generated table explicitly against hand-written expectations: ア=a, シ=shi (si), チ=chi, ツ=tsu, フ=fu, ヲ=wo, ン=n (nn), ガ=ga, ヂ=ji (di), ヅ=zu (du), パ=pa, キャ=kya, シャ=sha, ジャ=ja, ピョ=pyo. It also checks that every katakana character lies in U+30A1–U+30F6 and that no hiragana rune remains in a katakana row.

## Persistence (`store`)

Two new settings keys, no schema change:

| Key | Methods | Default when unset |
|---|---|---|
| `active_scripts` | `ActiveScripts() ([]string, error)`, `SaveActiveScripts([]string) error` (JSON array, like `selected_rows`) | `ActiveScripts()` returns `nil`; `GameState` treats that as hiragana-only |
| `katakana_offered` | `KatakanaOffered() (bool, error)`, `SaveKatakanaOffered(bool) error` | `false` |

Stats are keyed by character, so カ and か are counted separately without any change.

## Game State (`fyne/game.go`)

### State and the pause invariant

- `GameState` gains `activeScripts map[kanacore.Script]bool`, `katakanaOffered bool` and `pendingOffer bool`. `charSet` becomes `kanacore.AllKana()`.
- **Invariant:** `paused` is true only while something is pending, i.e. `pendingOffer` is set or `pendingIntro` is non-empty.
  - A helper `settle()` (under lock) enforces it: if `paused` is true but nothing is pending, it sets `paused = false`.
  - Every operation that can clear pending state enforces the invariant at the end: `EnableKatakana`, `DeclineKatakana`, `FinishIntro` and `applySettings` call `settle()`; `Reset` applies the same invariant directly, since `settle()` can only unpause and `Reset` must also be able to pause (see below).

### Loading

- `NewGameState` reads `ActiveScripts()` and `KatakanaOffered()`.
  - Unknown script names are ignored.
  - An empty or unreadable value means hiragana only.
- Fresh start and catch-up stay hiragana-only and use `ProgressionStepsFor(ScriptHiragana)`.

### Spawning

- `availableCharacters` only uses characters whose row is selected **and** whose script is active.
- If nothing qualifies, it falls back to the basic rows of the first active script, in `Scripts()` order. The old fallback to all characters is removed.

### Progression per script

- `checkAutoProgression` runs once for each active script, in `Scripts()` order. For script *s*, it unlocks the next incomplete step of `ProgressionStepsFor(s)` once every selected row **of script s** is mastered.
- The rows unlocked across all scripts in one answer are combined into one `rowsUnlockedEvent`, so a single intro dialog shows them, hiragana rows first.
- Rows of an inactive script never unlock.

### Offer

- The check runs in `checkAnswer` after the score-limit check. It fires only when all of these hold:
  - the game is not over;
  - this answer did not announce rows;
  - hiragana is active and katakana is not;
  - `katakanaOffered` is false;
  - all 11 hiragana basic rows are selected and mastered.
- It sends a new `katakanaOfferEvent` without blocking, like the other events. Only if the send succeeds does it set `paused` and `pendingOffer` and persist `katakanaOffered = true`.
- It skips answers that end the game or announce rows, so it never competes with the game-over dialog or an intro.
- In practice, the answer that masters the last basic row also unlocks `g`+`z`. The offer therefore comes on the *next* correct answer, early in the dakuon rows. This is intended.
- Closing the window while the offer dialog is open loses the offer for good, because `katakanaOffered` is already persisted. This is accepted.

### Operations (all under the lock, all ending with `settle()`)

**`EnableKatakana()`**
- Clears `pendingOffer`.
- Activates katakana and persists the active scripts.
- Sets `katakanaOffered = true` and persists it.
- If no `kata:` row is selected, it selects `kata:vowels`, persists the selection and sets `pendingIntro = [kata:vowels]` (paused).
- If `kata:` rows are already selected, there is no intro and `settle()` resumes the game.

**`DeclineKatakana()`**
- Clears `pendingOffer`.

**`applySettings(rows []string, scripts []kanacore.Script, auto bool, limit int)`**
- Replaces the inline block in the settings save callback with one locked operation, in this order:
  1. Apply and persist the row selection (existing `setSelectedRowsLocked`, which also prunes `pendingIntro`).
  2. Apply and persist the scripts. An empty set falls back to hiragana.
  3. If katakana went from off to on and the **new** selection contains no `kata:` row:
     - select `kata:vowels` and persist;
     - set `pendingIntro` to include `kata:vowels`;
     - pause.
     Additionally, for every other active script with no selected row, select that script's first step (`vowels` / `kata:vowels`) without an intro. This replaces the old “empty selection → `DefaultRowIDs()`” fallback.
  4. Whenever katakana is switched on here, set `katakanaOffered = true` and persist it, so a later switch-off cannot bring the offer back.
  5. Apply auto-progression and the score limit.
  6. Remove tiles whose row is not selected or whose script is inactive. This is a shared helper, `dropInactiveTiles()`.
  7. Rebuild the snapshot and call `settle()`.
- Turning a script off keeps its rows selected and stored, so they come back when it is switched on again.

**`Pause()`** is removed, because it would break the invariant. Tests set `paused` directly.

**`Reset`**
- Clears `pendingOffer`. The offer is already marked as made, and it cannot be pending at game over anyway, since a paused game cannot end.
- Then sets `paused` to whether an intro is still pending (equivalent to `settle()`'s invariant, but expressed directly: `settle()` itself only ever unpauses, so it cannot express the case where Reset must pause because the dropped game-over intro is still pending).

## UI

### Pending dialogs: one entry point (`fyne/intro.go`)

`showPendingDialogs(gs, statsPanel, inputBar, w)` must run on the Fyne thread. It replaces the scattered `if rows := gs.PendingIntro() …` calls and is used by `SetOnStarted`, Play Again, `watchEvents` (for both `rowsUnlockedEvent` and `katakanaOfferEvent`), the offer dialog and the settings save:
- If a dialog is already showing (`introShowing`, renamed `dialogShowing`), it does nothing.
- Otherwise it refreshes the stats panel from a fresh snapshot, so a change that only becomes visible once a dialog is resolved (e.g. a newly active script's section) shows up immediately rather than waiting for the next Enter.
- If `pendingOffer` is set, it shows the offer dialog.
- Otherwise, if `pendingIntro` is non-empty, it shows the intro dialog.
- Otherwise it calls `gs.Resume()`. This is defensive and keeps the invariant even if pending state was cleared elsewhere.

Each dialog clears `dialogShowing` in the first callback that runs when it closes, **before** calling into `GameState`. For the offer, a confirm dialog, that is its response callback, which Fyne runs before `SetOnClosed`. For the intro, a custom dialog with one dismiss button, it is `SetOnClosed`. The dialog then schedules `fyne.Do(func() { showPendingDialogs(...) })`, which chains the next pending dialog, for example offer → katakana intro, or resumes the game. Input is disabled while any of these dialogs is open. The resume branch of `showPendingDialogs` re-enables it, next to `Resume()`.

`watchEvents` treats both event kinds the same way:
- it drops stale events and events that arrive after game over;
- otherwise it calls `showPendingDialogs` via `fyne.Do`.

### Offer dialog (`fyne/intro.go`)

`showKatakanaOffer`:
- Title "Add Katakana?".
- Text "You know the basic hiragana. Katakana use the same sounds; start with ア イ ウ エ オ?"
- A preview line of the five vowel cards, in the intro's card style.
- Buttons "Add Katakana", which calls `EnableKatakana()`, and "Not now", which calls `DeclineKatakana()`. Both are followed by `showPendingDialogs` as described above.

### Settings dialog (`fyne/settings.go`)

- A "Scripts" row at the top, with one check per script.
- Below it, `container.NewAppTabs` with one tab per script. Each tab holds that script's groups, built by the existing `newGroupChecks`, in its own scroll container. The dialog height stays as it is today.
- The tab of an inactive script stays visible and editable, so rows can be preselected.
- On save:
  - Collect the checked rows of both tabs as they are, with no fallback in the UI.
  - Call `gs.applySettings(...)`.
  - Then call `showPendingDialogs`, which shows the katakana intro if one was created.

### Stats panel (`fyne/stats.go`)

- One section per script, with a heading ("HIRAGANA", "KATAKANA") above the existing structure: a basic grid with the header row, then one grid per further group. Group sections are keyed by script and group.
- `StatsSnapshot` gains `ActiveScripts map[kanacore.Script]bool`. A script's section is shown while the script is active. Inside it, rows and groups are shown or hidden as today.
- `rowShortLabel` strips the `kata:` prefix before mapping. Katakana rows therefore show `–`, `k`, `sh`, … like the hiragana rows.
- The "ACTIVE ROWS" list only shows selected rows of active scripts.
- The missed list covers all 208 characters.

### Tiles and intro dialog

No change. Katakana glyphs render in the same tiles and cards, and the intro dialog works for any row ID.

## Testing

- `kanacore`:
  - the generated table matches the explicit expectations above;
  - `AllKanaRows` has 54 rows and 208 unique characters;
  - every katakana row has the `kata:` prefix and mirrors its hiragana row in group, romaji and alternatives;
  - `ProgressionStepsFor(ScriptKatakana)` covers `RowsFor(ScriptKatakana)` in order without crossing groups;
  - `ScriptOf` works for both kinds of ID;
  - `Matches` accepts the katakana alternatives (シ/si, ン/nn, ヂ/di).
- `store`: `ActiveScripts` and `KatakanaOffered` round-trip; `ActiveScripts` returns `nil` when unset and `KatakanaOffered` returns `false` when unset.
- Game:
  - With only hiragana active, no katakana tile spawns, even if `kata:` rows are selected.
  - With both scripts active, both spawn.
  - Progression is per script: mastering the hiragana rows does not unlock katakana steps, and vice versa.
  - Both scripts unlocking in the same answer give one event with the hiragana rows first.
  - The offer fires once all hiragana basic rows are mastered. It does not fire if `katakanaOffered` is set, if katakana is already active, on the answer that ends the game, or on an answer that announced rows. It pauses and persists `katakanaOffered`.
  - `EnableKatakana` selects `kata:vowels`, sets the intro and persists the scripts. `DeclineKatakana` resumes.
  - Reset clears a pending offer and stays paused only while an intro is pending.
  - `EnableKatakana` with `kata:` rows already selected creates no intro and resumes.
  - `applySettings`:
    - switching katakana on with only `kata:k` preselected creates no intro and is not paused;
    - switching it on with no `kata:` row selects `kata:vowels`, creates the intro and pauses;
    - it sets `katakanaOffered`;
    - unchecking all hiragana rows while switching katakana on selects `vowels` again, as well as `kata:vowels`.
  - `settle()` unpauses when nothing is pending.
  - `applySettings` with an empty script set keeps hiragana. Switching katakana off removes katakana tiles on screen but keeps the `kata:` rows selected.
  - With only katakana active and nothing selected, the spawn fallback uses `kata:` basic rows.
  - The fresh start is unchanged: hiragana vowels only, katakana inactive.
- Stats panel:
  - the katakana section is hidden while katakana is inactive and shown once it is active;
  - `rowShortLabel("kata:sy") == "sh"`;
  - the ACTIVE ROWS list hides rows of inactive scripts.

## Out of Scope

- **Extended katakana** for loanwords (ティ ディ ファ フィ フェ フォ ウィ ウェ ウォ シェ チェ ジェ ヴ トゥ ドゥ ツァ …, following the 1991 外来語の表記 tables), **small ッ/っ** and the **long mark ー**. They are deferred to the vocabulary-trainer milestone, where they make sense inside words. Note for then: ティ = "ti" collides with the existing ち alternative "ti". This is also recorded in `PLAN.md`.
- The vocabulary trainer itself (JMdict, private word lists).
- A katakana-only fresh start.
