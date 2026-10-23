# Meaning mode

Date: 2026-10-23. Status: design approved in conversation.

## Goal

Since v0.4.1 a word tile is answered by its reading or by its meaning, and
both raise the same learning level. A learner cannot tell how well they know
what their words mean. The **meaning mode** asks a word tile for its meaning
only, and every word keeps a **reading level** and a **meaning level** of
its own.

The reverse direction (the tile shows the meaning, the learner types the
reading) is a different exercise and has its own backlog issue.

Terms (see `GLOSSARY.md`): **meaning mode**, **learning level**, **reading
level**, **meaning level**, **learned**, **slipped**, **weak item**,
**waiting word**, **words-only mode**.

## Decisions

| Question | Decision |
| --- | --- |
| Direction | Kana → meaning: the tile shows the word as today (its shown form), the learner types a meaning. |
| Choosing the mode | A saved setting in the settings Words tab, "Words ask for: Reading / Meaning", default Reading. It applies to word tiles only; kana tiles still ask for the romaji. It is independent of "Tiles: Kana and words / Words only". A change applies to tiles spawned afterwards; falling tiles keep their question. |
| Two levels | Every word has a reading level and a meaning level, both 0–5, with the same rules (hit +1, mistake −2, slipped until learned again), each with its own slipped flag, spaced-repetition memory and hit/miss counts. |
| Which answer raises which level | Reading mode: the reading raises the reading level; a meaning is still accepted and raises the meaning level, not the reading level. Meaning mode: only a meaning counts and raises the meaning level. |
| Reading typed in meaning mode | When the input matches a falling word tile's reading but none of its meanings, the game shows the hint "type the meaning" at the tile; it is no mistake and the tile keeps falling. Meanings are checked first: an input that matches both a meaning and a reading is a meaning hit. |
| Mistakes | A tile that falls through lowers the level of the chosen mode only: the meaning level in meaning mode, the reading level in reading mode. |
| Eligible words | All introduced words, bundled JLPT words included. Meaning matching is as today: any meaning in the meaning language or in English, ignoring brackets, articles and "to". A word with no meaning in either language does not fall in meaning mode. |
| Waiting words | Unchanged: a word falls only when all its kana are learned, in both modes. |
| Selection | In meaning mode, words are picked by their meaning level and meaning memory, also for "Focus on weak items". |
| Slipped frame | A word tile's red frame follows the level of the chosen mode. |
| Tile marker | In meaning mode, word tiles carry a small marker so a one-kana word (き, え) is not mistaken for a kana tile. Its look is settled in the plan; it must read in both themes and not hide the kana. |
| Feedback | Hit, miss and the game-over list show "kana · romaji — meaning" as today, in both modes. |
| Status line | Shows the counts of the chosen mode: "Words: N introduced · N learned · …" in reading mode, "Meanings: N introduced · N learned · …" in meaning mode. Showing both levels per word belongs to the level-dots backlog issue. |
| Statistics | Per-word hit/miss counts are kept apart for reading and meaning; wherever they are shown, the chosen mode's counts are shown. |
| Word packages | Shared: a word is introduced once and then belongs to both modes; the package intro is unchanged. Whether the newest package is done (80 % learned) is judged by the chosen mode's levels. |
| Speed | No special fall speed in meaning mode; the difficulty setting covers it. |
| Existing progress | Today's word levels, slipped flags, memory and hit/miss counts become the reading values. All meaning levels start at 0. |

## Behaviour details

- Progress and memory of a word's meaning are stored next to the reading's,
  keyed by the word's reading as today (e.g. `meaning:<reading>` beside
  `word:<reading>`); the existing keys keep meaning the reading, so no
  migration of existing rows is needed.
- Kana tiles, the kana learning path, katakana offer and auto-progression are
  unchanged by the mode.
- Words-only mode plus meaning mode is a pure meaning drill.
- In reading mode, "answered by meaning" (the hit feedback that also names the
  meaning) keeps its look; only which level it raises changes.
- The hint "type the meaning" uses the existing feedback mechanism, with its
  own feedback kind so it does not look like a hit or a miss.

## Units (indicative)

- `store`: meaning progress, memory and hit/miss counts beside the reading's;
  the "Words ask for" setting.
- `fyne`: the setting in the Words tab; answer matching per mode; level,
  memory and stats updates per mode; selection, weak items, slipped frame,
  package rhythm and status line by the chosen mode; tile marker; hint
  feedback.
- README (meaning mode) and GLOSSARY (done).

## Testing

- `store`: meaning progress and stats saved and loaded apart from reading;
  the setting round-trips with Reading as default.
- `fyne`: in reading mode a meaning hit raises the meaning level and leaves
  the reading level; in meaning mode a meaning hit raises the meaning level,
  a reading gives the hint and changes nothing, an input matching both is a
  meaning hit, a miss lowers only the meaning level; words without meanings
  do not fall in meaning mode; selection and "Focus on weak items" use the
  meaning level; the status line says "Meanings"; the newest package is done
  by meaning levels; a change of the setting leaves falling tiles alone;
  existing progress loads as reading progress with meaning levels at 0.
