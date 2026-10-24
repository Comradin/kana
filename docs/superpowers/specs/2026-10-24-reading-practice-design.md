# Reading practice

Date: 2026-10-24. Status: design approved in conversation.

## Goal

Words fall only after a package intro, so a learner reads few words and
always the same ones. **Reading practice** trains reading speed instead of
vocabulary: any word whose kana are all learned may fall, introduced or not,
and a word read counts for its kana, not for the word.

This replaces the "Reading practice" sketch in the vocab trainer design
(2026-10-12, stage 4): no word share slider, and only the kana of the form
typed count.

Terms (see `GLOSSARY.md`): **reading practice**, **reading**, **learned**,
**learning level**, **words-only mode**, **meaning mode**, **lesson
filter**, **word list**.

## Decisions

| Question | Decision |
| --- | --- |
| Form | A switch in the falling-tile game, not a screen of its own. Controls, points, miss limit and pause stay as they are; only which words fall and what a word hit counts change. |
| Setting | A saved checkbox "Reading practice" in the settings Words tab, a row of its own in the form, default off. While it is on, "Words ask for" is greyed out; its saved value stays and applies again when reading practice is off. |
| Which words | Every word of the chosen source whose kana are all learned (and whose script is on), introduced or not: the bundled JLPT list up to the chosen level, or the chosen word list with its lesson filter. This holds in mixed and in words-only mode. |
| Selection | Uniformly random among those words, avoiding words already falling unless that leaves none. "Focus on weak items" applies to kana tiles only. |
| Answer | Only a reading counts (the full form or any other accepted reading); a meaning is no hit, whatever "Words ask for" says. After a hit the feedback shows "kana · romaji — meaning" as today. |
| Word progress | None. A hit or a miss changes no word level, slipped flag, memory or hit/miss count, also for introduced words. |
| Kana progress | A word read counts as a hit for every learned kana unit of the reading typed (learned and in an active script): the short form of an optional part counts only its own kana; a kana that occurs twice (ここ) counts twice; another variant or merged reading may hold kana that are not learned or whose script is off (あした // あす, typed "ashita"), and those count for nothing. Each counts like a hit on a kana tile: hit count, streak, memory, learning level (already 5), and towards auto-progression and the katakana offer. The statistics show them like other kana hits. |
| Tempo | A word read raises the game tempo once, as a word hit does today, not once per kana. |
| Miss | A word that falls through counts towards the miss limit and lowers the tempo as today; it changes no kana and no word. |
| Packages and level offer | Not offered while reading practice is on; both resume when it is off. |
| Word share | As today: 5–10 % words in mixed mode; words-only mode is the reading drill. No slider. |
| Kana tiles | Unchanged: levels, auto-progression, katakana offer, focus on weak items. |
| Points and game over | Word hits score as today; the game-over list shows missed words as today. |
| Status line | The words line becomes "Reading practice: N words to read · N kana/min": N words is how many words may fall now; kana/min counts the kana units of words read (as for kana progress) per minute of play in the running round, pauses and dialogs excluded. Nothing of it is saved. |
| No word to read | Kana fall instead, as in words-only mode without playable words, and the status line says "No words to read yet — learn more kana or pick another list". |
| Changing the setting | Applies to tiles spawned afterwards; a falling tile keeps the rules it was spawned under. |

## Behaviour details

- A tile spawned in reading practice needs no marker: it asks for the
  reading like any word tile in reading mode.
- Words of a word list whose kana are learned may fall although the list's
  package rhythm has not reached them; they stay unintroduced.
- Word tiles in reading practice use the word fall factor as today.
- The kana/min figure starts at 0 with each round and shows "–" until the
  first word is read.

## Units (indicative)

- `store`: the `reading_practice` setting.
- `fyne`: the checkbox in the Words tab and greying "Words ask for"; the
  readable-word pool (source, lesson filter, learned kana, scripts);
  spawning and uniform pick; answer matching by reading only; kana credit
  per typed unit; tempo, miss and package/level-offer gating; the status
  line with words to read and kana/min.
- README (reading practice) and GLOSSARY (done).

## Testing

- `store`: the setting round-trips, default off.
- `fyne`: with reading practice on, an unintroduced word whose kana are
  learned falls and one with an unlearned kana does not; the lesson filter
  and the JLPT level limit the pool; a reading hit raises no word level and
  records no word stats or memory, and credits each typed kana unit (short
  form only its kana, a doubled kana twice); a meaning typed is no hit; a
  fall-through counts towards the miss limit and changes no kana or word;
  no package intro or level offer comes; the status line shows words to
  read and kana/min, and the reason when no word is readable; a falling
  tile keeps its rules when the setting changes; "Words ask for" is
  disabled while reading practice is on.
