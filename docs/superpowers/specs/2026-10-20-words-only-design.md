# Words-only mode, lesson filter, meaning language, focus on weak items

Date: 2026-10-20. Status: design approved in conversation.

## Goal

Learners from a Japanese course want to practise the vocabulary of the
current lesson, in their language, and to drill what they keep getting
wrong. Stage 4b, part 1.

Terms (see `GLOSSARY.md`): **words-only mode**, **lesson filter**, **weak
item**, **learning level**, **learned**, **slipped**, **waiting word**.

## Decisions

| Question | Decision |
| --- | --- |
| Words-only mode | A stored setting in the Words tab: "Tiles: Kana and words / Words only". Default: kana and words (today's game). |
| What falls in words-only mode | Only word tiles. The rule that every kana of a word must be learned does **not** apply, neither for falling nor for new packages; it stays for the mixed mode. |
| Kana learning path in words-only mode | Rests: no row intros, no automatic unlocking, no katakana offer, no kana tiles. Switching back to mixed continues where it stood. Word hits do not change kana levels (as today). |
| No word can fall in words-only mode | When the lesson filter or source leaves no introduced word and no package can come (filter empty, list missing, everything filtered out), kana fall as in the mixed mode and the status line says why, e.g. "Words only: no words in the chosen lessons". When words of the filter are merely not introduced yet, the next package comes. |
| Lesson filter | For a private list: the lessons of the chosen list as checkboxes plus "All lessons", in the Words tab below the source select, in a scroll area. Stored per list. JLPT has no lesson filter (its level select stays). |
| What the filter decides | Which introduced words fall, and where new packages come from (the earliest chosen lesson with words not introduced yet). In both modes. "All lessons" (the default) is today's behaviour. |
| Words without a lesson | Shown as one more checkbox, "No lesson", when the list has such words. |
| Meaning language | A select "Meanings in" in the Words tab: "System" (today's rule: the system language if the words have it, else English) and every language found in the word data, by its own name (Deutsch, English, Español, Français, Magyar, Nederlands, Русский, Slovenščina, …). It sets the meaning shown and the meaning accepted as an answer; English is always accepted too. Stored in the existing `meaning_lang` setting. |
| Focus on weak items | A checkbox "Focus on weak items" in the Game tab, stored. A **weak item** is a kana or word that is learning (level 1–4) or slipped. With focus on, every spawn first draws: with probability ½ it picks among the weak candidates (slipped ones weigh twice as much as learning ones); otherwise, or when there is no weak candidate, the usual spaced-repetition pick. It applies to the kana pick and the word pick alike; it does not change how often a word rather than a kana falls. |

## Behaviour details

### Words-only mode

- Every spawn is a word: the word share is 1, not `wordShare`.
- Playable words in words-only mode: introduced words that pass the source
  and lesson filter and whose script(s) are active — active scripts still
  matter (a katakana word needs katakana switched on), learned kana do not.
- Packages in words-only mode ignore whether kana are known
  (`NextPackage`'s `known` accepts every unit).
- The kana learning path rests: `checkAutoProgression`, row intros and the
  katakana offer do nothing while words-only mode is on. Pending kana intros
  stay pending until the mode is switched back.
- The status panel's kana grid stays as it is; the words line in
  words-only mode starts with "Words only:" when it reports a problem.
- "Waiting for kana" (introduced words that cannot fall) is counted with
  the rule of the current mode, so it is 0 in words-only mode.

### Lesson filter

- Stored per list, e.g. setting `lesson_filter:<list name>` = JSON array of
  lesson names; absent or empty = all lessons. Renaming a list loses its
  filter (acceptable; list management comes later).
- `vocab.Source` gains the chosen lessons; `Has` accepts an entry of the
  list in at least one chosen lesson (or, for "No lesson", without one).
- A lesson that no longer exists in the list is ignored. A filter that
  matches no word is "no words in the chosen lessons".
- Changing the filter in the settings invalidates the playable words and,
  like a source change, starts the package rhythm afresh (the newest
  package no longer holds the next one back).

### Meaning language

- The list of languages is the union of the meaning languages of the
  bundled data and the loaded lists, sorted by display name, with "System"
  first. The language-less `meaning` column counts as no language.
- Changing it takes effect at once for feedback and answers.

### Focus on weak items

- Kana: the weak candidates are the available characters (as today, minus
  those already falling) with level 1–4 or slipped.
- Words: the weak candidates are the playable words with level 1–4 or
  slipped.
- Weight: slipped 2, learning 1; pick proportionally.
- New items (level 0, never seen) are not weak; they keep coming through
  the usual pick.

## Units (indicative)

- `vocab`: `Source.Lessons` (or a lesson set) and `Has`; package order
  within the chosen lessons; lessons of a list (`Index.Lessons(list)`), and
  whether it has words without a lesson; languages present
  (`Index.Languages()`).
- `store`: words-only, focus and per-list lesson filter settings (plus
  tests); `meaning_lang` exists.
- `fyne`: mode and focus state on `GameState`; `playableWords` and
  `pickWord` per mode; spawn per mode; kana path resting; a weak-item pick
  around `memory.Pick`; status line texts; settings widgets (mode radio,
  lesson checkboxes with "All lessons" / "No lesson", meaning language
  select, focus checkbox) and their apply.
- README: the new settings.

## Testing

- `vocab`: `Has` with lessons and "No lesson"; packages limited to the
  chosen lessons; `Lessons`, `Languages`.
- `store`: the new settings round-trip and default.
- `fyne`:
  - words-only: only word tiles spawn; words with unlearned kana fall and
    come in packages; a katakana word needs katakana active; no kana rows
    are unlocked and no katakana offer comes; kana fall and the status says
    why when no word can fall; switching back restores the mixed rules.
  - lesson filter: only words of the chosen lessons fall; packages come
    from the earliest chosen lesson; stored per list; "No lesson".
  - meaning language: shown meaning and accepted answers follow the
    setting; English is still accepted; "System" keeps today's rule.
  - focus: with a seeded `rnd`, about half the picks are weak items and
    slipped ones twice as often as learning ones; no weak item, usual pick;
    off, today's behaviour.
  - settings: the widgets carry and apply the values; the lesson list
    follows the chosen source.
