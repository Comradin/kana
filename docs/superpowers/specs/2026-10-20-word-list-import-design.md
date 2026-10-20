# Word list import

Date: 2026-10-20. Status: design approved in conversation.

## Goal

A learner without file-system experience can add their own word list from
inside the game. Today they have to copy the file into
`~/.local/share/kana/words/` by hand, a folder they only learn about from the
README, and restart the app.

Terms (see `GLOSSARY.md`): the learner's **source file** becomes a **word
list** by **import**.

## Decisions

| Question | Decision |
| --- | --- |
| Name | "Import", not "upload". Button: "Import word list…". |
| Copy or reference | The game copies the source file into the words folder. Later edits to the source file are not picked up; importing again replaces the list. |
| Where | A button in the settings' Words tab, below the source select. No drag and drop for now. |
| File dialog | Fyne's open dialog, starting in the home directory, showing `.csv`, `.tsv`, `.json` and the spreadsheet formats `.xlsx`, `.xls`, `.ods` (only so their hint can be shown). |
| Check first | The file is read with the list parser and a preview is shown; the learner confirms with Import or cancels. |
| Rejected | Not UTF-8 (after the UTF-16 conversion below), spreadsheet formats, no `kana` column, no words, a kana column of only `?`, larger than 5 MB, a name starting with `.`, the reserved name `jmdict`. Each with an English hint. |
| Encoding | UTF-8 with or without BOM as is; UTF-16 with BOM (Excel "Unicode Text") is converted to UTF-8. Shift-JIS and ANSI are rejected with the hint to save as "CSV UTF-8". |
| Name of the list | The file name without extension, as `LoadDir` names lists. Not editable (renaming belongs to list management, stage 4b). |
| Name exists | If a word list with the same name exists (any of the three extensions, ignoring case), the preview says so and its button reads Replace. Replace removes the old file(s) of that name and writes the new one, so the list keeps its name and the learner's selection. No automatic " 2" suffix. |
| Progress | Replacing never deletes progress: levels, stats and memory are keyed by reading. Words missing from the new version keep their state and come back with it if they return. |
| Afterwards | The lists are reloaded at once; the new list appears in the source select and is preselected. The source change takes effect with the settings' OK like any other setting. |
| Cancel | The import is final when Import/Replace is clicked; cancelling the settings afterwards keeps the word list and drops only the preselection. |
| Folder | The Words tab shows where word lists are kept, e.g. "Word lists are kept in ~/.local/share/kana/words". The folder is created at startup if it is missing. |
| Language | All text in English, like the rest of the UI. |
| Not included | Drag and drop, deleting and renaming lists, opening the folder, Shift-JIS. |

## Preview

A dialog titled "Import word list" with:

- the list's name and, if it exists, "Replaces the word list of the same name; your progress is kept.";
- counts: words, lessons (if any), meaning languages found (e.g. "de, en");
- skipped rows with up to ten problems, as the parser reports them;
- the first three words: kana, meaning (in the meaning language), lesson.

Buttons: Import (or Replace) and Cancel. A rejected file gets an error dialog
with the reason and the hint instead of a preview.

Hints:

- not UTF-8 or only `?` in the kana column: "Save the file as \"CSV UTF-8\" in your spreadsheet program and import it again."
- spreadsheet format: "Save the sheet as \"CSV UTF-8\" in your spreadsheet program and import the .csv file."
- no `kana` column: the parser's problem, plus an example header `kana;meaning;lesson`.

## Units

### `vocab`: checking and writing

- `Inspect(fileName string, data []byte) (Candidate, error)`: decodes (BOM,
  UTF-16), checks the rules above, parses with `readCSV`/`readJSON` and returns
  `Candidate{Name, Ext string; List List; Data []byte}` where `Data` is the
  UTF-8 bytes to write. Errors are a type with the reason and the hint, so the
  UI can show both. Pure: no file system access, easy to test.
- `Exists(dir, name string) bool`: a list file of that name is in dir.
- `Import(dir string, c Candidate) error`: creates dir if needed, writes
  `<name><ext>` through a temporary file and a rename, then removes other files
  of the same name (other extension or case). A failure leaves the old list in
  place.

### `fyne`: reloading and the dialog

- `wordsDir` is created in `loadWords` (startup).
- `GameState.ReloadLists()`: under the lock, reads the folder again, rebuilds
  `listNames` and the word index (`bundledIndex().WithLists`) and invalidates
  the playable cache. The stored word source is not changed here.
- `GameState.ImportList(c vocab.Candidate) error`: `vocab.Import` into
  `wordsDir`, then `ReloadLists`.
- Words tab: the "Import word list…" button and the folder line. After an
  import, the form rebuilds the source select's options from the new list
  names and selects the imported list.
- Reading the chosen file goes through Fyne's `URIReadCloser`; the size limit
  is checked while reading.

## Testing

- `vocab`: `Inspect` for UTF-8 with and without BOM, UTF-16 LE/BE with BOM,
  invalid UTF-8, `?` kana, missing kana column, empty file, spreadsheet
  extension, size limit, dot name, `jmdict`, JSON; `Import` for a new list, a
  replacement with another extension and case, and a failed write keeping the
  old file.
- `fyne`: `ReloadLists` adds the new list and keeps progress of a replaced
  list; the startup creates the words folder; the Words tab preselects the
  imported list and keeps it after Cancel of the settings (the word list stays,
  the source does not change); the preview shows Replace for an existing name.
- README: the private word lists section describes the import first and the
  folder second.
