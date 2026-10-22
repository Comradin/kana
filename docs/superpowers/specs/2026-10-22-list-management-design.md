# Word list management

Date: 2026-10-22. Status: design approved in conversation.

## Goal

Since v0.4.0 a learner can import a word list, but can only add or replace
lists. Deleting, renaming or understanding a broken list means working in a
hidden folder by hand. Stage 4b, part 2: manage word lists from the game.

Terms (see `GLOSSARY.md`): **word list**, **source file**, **import**,
**skipped row**, **delete a word list**, **rename a word list**, **lesson
filter**.

## Decisions

| Question | Decision |
| --- | --- |
| Where | A button "Manage word lists…" in the Words tab opens its own dialog. The existing "Import word list…" button stays. |
| Dialog rows | One row per private list, ordered by name like the source select: name, number of words and lessons, "N rows skipped" with a "Details" button when the list has skipped rows, and the buttons "Rename" and "Delete". JLPT is not listed. Without lists: "No word lists yet". |
| Dialog footer | "Import…" (the same flow as "Import word list…") and "Open folder". |
| Skipped rows | "Details" shows the list's messages (at most 10, as loaded) and one sentence on how to fix the file. The status panel's words line adds " · N rows skipped" when the chosen source has skipped rows. The source select shows no mark. The UI never says "problem". |
| Reload | Automatic each time the settings dialog opens; no button. Files copied or edited by hand come in then. |
| Open folder | Opens the word list folder in the system file manager (`OpenURL` with a `file://` URL). If that fails, a dialog shows the path to copy. The line "Word lists are kept in …" stays. |
| Delete | After a confirmation ("Delete minna? Its file is removed; your progress on its words is kept."), the list's file is removed for good (no trash: the learner's source file stays where it was imported from). Progress (keyed by reading) stays. The list's stored lesson filter is removed. If it was the chosen source, JLPT becomes the source. Introduced words of the list fall afterwards only if JLPT or another list has the same reading. Resetting word progress is a separate issue. |
| Rename | The list's name is its file name without the extension; renaming renames the file and keeps the extension. Valid: not empty after trimming, no `/` or `\`, not `jmdict` (any case), not the name of another list (ignoring case). A change of case only is allowed (done via a temporary name). The lesson filter, the chosen source and the newest package follow the new name. |
| Duplicate names | Files sharing a name are loaded as "minna (csv)", "minna (json)"; each list has exactly one file. Delete and rename act on that one file; renaming "minna (json)" to "minna2" makes `minna2.json`. |
| When they take effect | Delete and rename act at once in the management dialog, not on the settings' Save (Cancel cannot bring a file back). Afterwards the settings dialog refreshes its source select and lessons, as after an import. |
| Drag & drop | A file dropped anywhere on the game window opens the same import preview as "Import word list…"; the game pauses as for the settings. Several files: one preview after another. A folder or unsuitable file gets the import's error message. The preview opens on top of any open dialog (intro, package, game over, settings); afterwards that dialog is there again, and an open settings dialog shows the new list. |

## Behaviour details

- A list remembers the path of its file (`vocab.List` gains it), so delete
  and rename know which file to touch.
- Delete and rename go through `vocab` (file operations, validation) and
  `GameState` (settings that follow the list, reload, playable cache); the
  store gains a way to move or drop a list's lesson filter.
- After delete or rename the lists are reloaded, as after an import.
- Rename into a name that only differs in case from the list's own name is
  not a collision.
- Errors from the file system (permissions; for rename also a file that is
  gone) are shown in a dialog; the lists are reloaded either way. Deleting
  a file that is already gone counts as deleted.
- Reload on opening the settings happens under the game lock before the
  dialog reads its state; it invalidates the playable words.

## Units (indicative)

- `vocab`: `List.Path`; `DeleteList(path)`, `RenameList(dir, l, newName)`
  with validation errors as `ImportError`-like reasons.
- `store`: move a lesson filter to a new list name, delete it.
- `fyne`: `GameState.DeleteList`, `RenameList`, `ReloadLists` (reload,
  source/filter/newest package follow), skipped rows in `WordsStatus`; the
  management dialog; "Open folder"; drop handler on the main window that
  feeds the import preview; settings open triggers a reload.
- README and GLOSSARY.

## Testing

- `vocab`: delete removes the file; rename keeps the extension, rejects
  empty, separators, jmdict, another list's name (any case), allows a case
  change; duplicate-name lists rename their own file.
- `store`: lesson filter moved and deleted.
- `fyne`: delete keeps progress, drops the filter, falls back to JLPT when
  it was the source; rename carries filter, source and newest package;
  opening settings reloads a hand-copied list; the words line shows skipped
  rows of the chosen source only; the management dialog lists rows, shows
  "No word lists yet", and "Details" shows the messages; a dropped file
  reaches the import preview, several files in turn, a folder gives the
  error.
