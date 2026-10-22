# Word list notation

Date: 2026-10-22. Status: design approved in conversation.

## Goal

Course word lists write more than plain kana in their kana column: a
placeholder for a name (`～さん`, `こちらは～さんです`), optional parts in
brackets (`（どうぞ）よろしく（おねがいします）`), punctuation (`おなまえは？`)
and two writings of one word (`あめりか // アメリカ`). Today every such row
is a skipped row. The game should read these rows, show them as written and
ask only for the kana.

A small command lists every skipped row of a file, so a learner can check a
list outside the game and pass the result on.

Terms (see `GLOSSARY.md`): **word list**, **source file**, **skipped row**,
**placeholder**, **optional part**, **variant**.

## Decisions

| Question | Decision |
| --- | --- |
| Placeholder | `～` (U+FF5E), `〜` (U+301C) and `~` stand for something the learner fills in, anywhere in the field. The tile shows them; they are not typed: `～さん` is answered `san`, `こちらは～さんです` `kochirawasandesu`. |
| Optional parts | Text in brackets, `（…）` or `(…)`, is optional. The tile shows the field with brackets. Accepted readings: the full form (brackets removed, contents kept) and the short form (bracketed parts removed). Nothing in between: `（どうぞ）よろしく（おねがいします）` takes `douzoyoroshikuonegaishimasu` or `yoroshiku`, not `douzoyoroshiku`. Brackets must pair and must not nest; otherwise the row is skipped. |
| Punctuation and spaces | `？ ！ 。 、 ・ ? ! . ,` and spaces (also the full-width space) are shown, not typed. |
| Variants | `//` separates writings of one word; the row is one word. The tile shows the last variant (in course lists the usual writing, e.g. `アメリカ`). Every variant's reading is accepted. The word's kana and its script are those of the last variant, so `アメリカ` needs katakana switched on. |
| Reading | A word's **reading** is the shown form without placeholders, punctuation, spaces and bracket characters (for optional parts: the full form). Progress, statistics, merging across lists and the bundled list, and introductions are keyed by it, as today. `～さん` in two lists, or `さん` in JLPT and `～さん` in a list, is one word. |
| Shown form | Wherever the game shows a word to the learner: tile, hit and miss feedback, package intro, import preview, game-over list. When an entry has several shown forms (the same reading written differently in two lists, or a plain bundled word and a list's `～さん`), the first one a private list gives wins, in list order; a bundled word alone is shown as its reading. |
| Particles | In a word whose field uses a placeholder, an optional part, punctuation or a space (a **phrase**), は may also be typed `wa` and へ `e`, as particles are spoken: `こちらは～さんです` takes `kochirawasandesu` and `kochirahasandesu`. Plain words and variants alone are not phrases, so `はな` stays `hana`. を takes `o` as well as `wo` everywhere, also as a kana tile. |
| Romaji shown | The romaji of the full reading (as in the package intro and miss feedback today). |
| Still skipped | A field that is empty after removing the notation ("empty kana"), one with other non-kana characters ("… is not kana", quoting the field as written), unpaired or nested brackets ("… has unpaired brackets"), an empty variant ("… has an empty variant"). A variant that is not kana skips the whole row. A short form that is no word on its own (っ, ー or a small kana at its edge: `いっ（て）`, `（コ）ーヒー`) is dropped as a reading and only the full form counts; the row is kept. |
| Skipped rows shown | Up to 100 messages per list (today 10). The "Details" dialog and the import preview show them in a scroll area; the count of skipped rows stays exact. |
| Check command | `go run ./cmd/listcheck [-words] FILE...` reads each file as the import does (encoding, columns, notation), touches nothing, and prints per file the name, words and skipped rows, then every skipped row's message (all of them, not 100), or the import's error with its hint. With `-words` it also prints every word: shown form, reading in romaji, other accepted readings, lesson. Exit status 1 when a file cannot be imported. |

## Behaviour details

- The notation is read in `vocab`, for CSV and JSON alike, before the kana
  check. `vocab.Word` and `vocab.Entry` gain the shown form (empty = the
  reading) and the other accepted readings.
- The kana check (`kanacore.KanaUnits`) applies to every accepted reading.
- An entry's units (which kana a word needs, which script) are those of its
  reading; the short form's kana are a subset of them.
- Answer matching accepts the reading or any other accepted reading, then
  the meaning, as today. "Answered by meaning" means none of the readings
  matched.
- Accepted readings merge across sources like meanings: unique, in
  first-seen order.
- Meanings, lessons and JLPT columns are unchanged.
- The bundled list does not use the notation; `cmd/vocabgen` is unchanged.
- Words already played under a list keep their progress: a row that was
  skipped before was never played.

## Units (indicative)

- `vocab`: a notation parser (field → reading, shown form, other readings,
  or the reason for skipping); `Word.Display`, `Word.Alts`;
  `Entry.Display`, `Entry.Alts`; a reading match over all accepted
  readings; `maxProblems` 100; a problem limit that the check command can
  lift.
- `cmd/listcheck`: the check command.
- `fyne`: shown form on tiles, feedback, package intro, import preview and
  game over; answers through the entry's readings; the skipped rows in a
  scroll area.
- README (notation for list authors, check command) and GLOSSARY.

## Testing

- `vocab`: each notation alone and combined (placeholder in brackets,
  variants with placeholders); both readings of optional parts and not the
  mixed ones; unpaired and nested brackets; empty after notation; a non-kana
  variant; CSV and JSON; merging of shown form and readings across lists
  and with a bundled word; up to 100 messages.
- `cmd/listcheck`: output for a good file, a file with skipped rows beyond
  100, a file that cannot be imported (exit 1), `-words`.
- `fyne`: a placeholder word's tile shows `～さん` and takes `san`; an
  optional-part word takes both readings and is "by meaning" for neither; a
  variant word needs katakana; feedback and package intro show the shown
  form; a word missed is listed in its shown form.
