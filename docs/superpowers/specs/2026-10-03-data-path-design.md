# Design: Data Path

**Date:** 2026-10-03
**Status:** Approved
**Convention:** `claude-setup/rules/comradin.md` → "Where apps keep user data"

## Overview

`kana-desktop` opens `kana.db` in the current working directory (`fyne/main.go`). That fails for
a macOS `.app` launched from the Finder (cwd `/`) and scatters databases across directories.
The database moves to a per-user data directory, the same on every OS.

## Decisions

| Topic | Decision |
|---|---|
| Location | `$XDG_DATA_HOME/kana/kana.db`; if `XDG_DATA_HOME` is unset or not absolute, `<home>/.local/share/kana/kana.db`. Same rule on Linux, macOS and Windows. `os.UserConfigDir()` / `~/Library/Application Support` are not used. |
| Config files | None today; settings live in the database. `~/.config/kana/` is not created. |
| Legacy database | If the new file does not exist and `./kana.db` exists in the working directory, move it (plus `kana.db-wal` and `kana.db-shm` if present) to the new location on start, and print one line to stderr. If the new file exists, never touch the legacy one. |
| Move failure | Try `os.Rename`; if that fails (e.g. across filesystems), copy the files and leave the originals in place. If copying fails too, start with the new location anyway and print the error; never delete user data. |

## Code

- `store.DefaultPath() (string, error)`: resolves the path above. Uses `os.Getenv("XDG_DATA_HOME")` and `os.UserHomeDir()`; returns an error only if no home directory can be found.
- `store.MigrateLegacy(legacy, target string) (moved bool, err error)`: does the move or copy described above for the main file and its `-wal`/`-shm` siblings. It creates the target directory as needed.
- `fyne/main.go`: `path, err := store.DefaultPath()`; on error, print it and exit. Then `MigrateLegacy("kana.db", path)`, print a line if moved or on error, then `store.Open(path)` (which already creates the directory).

## Testing (`store/path_test.go`)

- `DefaultPath` honours an absolute `XDG_DATA_HOME` and ignores a relative one; falls back to `$HOME/.local/share/kana/kana.db` (set `HOME` and, for Windows, `USERPROFILE` via `t.Setenv`).
- `MigrateLegacy`:
  - legacy present, target missing → target has the contents, legacy and its `-wal`/`-shm` are gone, `moved == true`;
  - target present → nothing changes, `moved == false`;
  - legacy missing → no-op, `moved == false`, no error;
  - target directory is created when missing.

## Docs

README "Data Persistence": state the new location, the `XDG_DATA_HOME` override and the one-time move from the working directory.
