# Design: Versioning and Release Builds

**Date:** 2026-10-12
**Status:** Approved (design discussion), revised after spec review
**Branch:** `feat/releases` (from `main` after PR #12)

## Overview

Kana gets versions, a changelog and downloadable builds on GitHub Releases. Versions are
cut by release-please from the conventional commit messages the repository already uses. A
GitHub Actions workflow builds Windows and Linux packages for each release; the macOS package
is built and uploaded by the maintainer from their Mac with a script. Builds are unsigned for
now (macOS gets an ad-hoc signature only). The app shows its version.

## Decisions

| Topic | Decision |
|---|---|
| Platforms | macOS arm64 (Apple Silicon), Windows x64, Linux x64. No Intel Mac, Linux ARM or mobile. |
| Versioning | SemVer via release-please, release type `go`. First release **v0.1.0** (`initial-version`). Before 1.0, `feat:` bumps minor, `fix:` bumps patch, and with `bump-minor-pre-major` a breaking change bumps minor instead of going to 1.0. |
| Release trigger | Merging the release-please PR creates the tag and the GitHub Release. |
| Windows/Linux builds | GitHub Actions on `windows-latest` and `ubuntu-22.04`, with `fyne package`. |
| macOS build | Local script `scripts/release-macos.sh` on the maintainer's Mac; no macOS runner. |
| Packages | macOS `.dmg` with `Kana.app`; Windows `.zip` with `Kana.exe`; Linux `.tar.xz` from `fyne package`. Each with a `.sha256` file. |
| Signing | None, except an ad-hoc macOS signature (`codesign -s -`) so the app is not reported as damaged. Steps are separated so real signing and notarisation can be added later with secrets. |
| Version in the app | Window title and settings dialog, with a link to the releases page; "dev" for builds not made by `fyne package --release`. No network access. |
| Icon | A new icon in the tile style (か on the tile colour with its shadow), SVG source plus 1024 px PNG. |

## Release flow

1. Commits land on `main` through PRs as today (rebase merge, so each commit's type counts).
   Only `feat:` and `fix:` commits (and breaking changes) open a release PR; `docs:`/`chore:`
   alone do not.
2. `release.yml` runs release-please on every push to `main`. It opens or updates the release
   PR "chore(main): release 0.x.y" with `CHANGELOG.md` and the manifest version.
3. Merging that PR makes release-please create tag `v0.x.y` and the GitHub Release with the
   changelog section as notes.
4. In the same workflow run, the build jobs (only when a release was created) build the
   Windows and Linux packages and upload them with `gh release upload`. A final job appends to
   the release notes: "The macOS build is uploaded separately and may follow shortly."
5. The maintainer runs `scripts/release-macos.sh v0.x.y` on their Mac, which builds and
   uploads the `.dmg` and its `.sha256`.

## Repository files

### release-please

- `release-please-config.json`:
  ```json
  {
    "bootstrap-sha": "da0eaba1deae2d0966722e19985cac5818da7f76",
    "packages": {
      ".": {
        "release-type": "go",
        "package-name": "kana",
        "initial-version": "0.1.0",
        "bump-minor-pre-major": true,
        "include-component-in-tag": false
      }
    }
  }
  ```
  `bootstrap-sha` (top level) is `main` after PR #12: release-please ignores the older
  history, so the first changelog lists only this work and later commits. `initial-version`
  is required: without it a manifest at 0.0.0 makes the first release 1.0.0
  (release-please issue #2087).
- `.release-please-manifest.json`: `{ ".": "0.0.0" }`.
- Repository settings (manual, by the maintainer), documented in the README's maintainer
  section:
  - Settings → Actions → General → "Allow GitHub Actions to create and approve pull requests".
  - No required status checks on `main`: release PRs opened with the default token do not
    trigger CI, so a required check would block them forever.

### `.github/workflows/ci.yml`

- On pull requests and pushes to `main`.
- One job on `ubuntu-22.04`: `actions/setup-go` with `go-version-file: go.mod`,
  `sudo apt-get update && sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev`, then
  `test -z "$(gofmt -l .)"`, `go vet ./...`, `go test -count=1 ./...`.

### Fyne CLI version

`scripts/fyne-cli-version` holds one line, the pinned `fyne.io/tools` version (`v1.7.3`, the
current release, which packages Fyne v2.7). The workflow and the script both read it, so there
is one source. Installed with `go install fyne.io/tools/cmd/fyne@<version>`.

### `fyne package` facts the build relies on

- Run with `--src <absolute path to fyne/>`; it reads `fyne/FyneApp.toml` there.
- `--executable kana`: without it the binary is named after the source folder (`fyne`). The
  path must not exist yet, otherwise packaging skips its own build and injects no metadata.
- `--app-version` takes the version **without** `v` (`0.1.0`); callers pass `${tag#v}`.
- Outputs: Linux `Kana.tar.xz` and macOS `Kana.app` in the current directory; Windows
  `Kana.exe` in the `--src` directory.
- It increments `Build` in `fyne/FyneApp.toml`. The script restores the file afterwards
  (`git checkout -- fyne/FyneApp.toml`, also in an `EXIT` trap); in CI the change is
  discarded with the runner.

### `.github/workflows/release.yml`

- Triggers: `push` to `main`; `workflow_dispatch` (dry run); and, **temporarily for testing
  on this branch**, `push` to `feat/releases` (a workflow file can only be dispatched once it
  is on the default branch). The branch trigger is removed in the last commit before merge.
- `concurrency: release` (no cancel), so two quick pushes do not race release-please.
- Permissions: `contents: write`, `pull-requests: write`.
- **Job `release-please`** (only on push to `main`): `googleapis/release-please-action@v4`;
  outputs `release_created` and `tag_name`.
- **Job `build`**, `needs: release-please`, with
  `if: ${{ !cancelled() && (needs.release-please.outputs.release_created == 'true' || github.ref != 'refs/heads/main') }}`
  (the second part covers the temporary branch trigger and a dispatch from another ref; a
  dispatch on `main` without a release builds nothing). Matrix:

  | `os` | target | asset |
  |---|---|---|
  | `windows-latest` | `windows` | `kana-<tag>-windows-x64.zip` containing `Kana.exe` |
  | `ubuntu-22.04` | `linux` | `kana-<tag>-linux-x64.tar.xz` (renamed `Kana.tar.xz`) |

  All steps use `shell: bash` (Git Bash on Windows). Steps:
  1. Checkout (at the tag for a release).
  2. setup-go from `go.mod`; on Ubuntu the apt dependencies as in CI; on Windows
     `gcc --version` so a runner image without MinGW fails clearly.
  3. Install the Fyne CLI from `scripts/fyne-cli-version`.
  4. Version and tag: release → `tag` from the output, `version=${tag#v}`; otherwise
     `tag=dev-<short sha>`, `version=0.0.0`.
  5. `fyne package --os <target> --release --app-version <version> --src "$PWD/fyne"
     --executable kana`.
  6. Rename/zip to the asset name (Windows: zip `fyne/Kana.exe`, with `7z a` if `zip` is
     unavailable), write `<asset>.sha256` with `sha256sum <asset> > <asset>.sha256`.
  7. Release: `gh release upload <tag> <asset> <asset>.sha256`. Otherwise:
     `actions/upload-artifact` with both files.
- **Job `notes`** (needs `build`, release only): `gh release view <tag> --json body -q .body`
  into a file, append a blank line and the macOS note with `printf`, then
  `gh release edit <tag> --notes-file <file>`.

### `scripts/release-macos.sh`

Bash, `set -euo pipefail`, `shellcheck`-clean. Usage:
`scripts/release-macos.sh [--no-upload] [vX.Y.Z]`.

1. Requires macOS (`uname -s` = `Darwin`) and arm64 (`uname -m` = `arm64`); otherwise exits
   with a message.
2. `repo` = the repository root (`git rev-parse --show-toplevel`).
3. Tag: the argument, or without one the newest release
   (`gh release view --json tagName -q .tagName`). With `--no-upload` and no tag:
   `tag=dev`, `version=0.0.0`. Otherwise `version=${tag#v}`.
4. Checks (unless `--no-upload`): `gh auth status`; `git fetch --tags`; the tag exists; the
   working tree is clean; `git describe --exact-match --tags` equals the tag, otherwise it
   tells the user to `git checkout <tag>`.
5. Installs the Fyne CLI from `scripts/fyne-cli-version`.
6. In a temporary directory `tmp` (removed by the `EXIT` trap, which also restores
   `fyne/FyneApp.toml`): `cd "$tmp" && fyne package --os darwin --release --app-version
   "$version" --src "$repo/fyne" --executable kana`, giving `$tmp/Kana.app`.
7. `codesign --force --deep -s - "$tmp/Kana.app"` (ad-hoc signature).
8. A staging folder with `Kana.app` and an `Applications` symlink to `/Applications`;
   `hdiutil create -volname Kana -srcfolder <staging> -ov -format UDZO
   "$repo/dist/kana-$tag-macos-arm64.dmg"`; then, inside `dist/` so the file names no path,
   `shasum -a 256 kana-$tag-macos-arm64.dmg > kana-$tag-macos-arm64.dmg.sha256`. `dist/` is
   git-ignored.
9. Unless `--no-upload`: `gh release upload "$tag" <dmg> <dmg>.sha256 --clobber`. Prints the
   file paths.

### App metadata and icon

- `fyne/FyneApp.toml`:
  ```toml
  [Details]
  Icon = "Icon.png"
  Name = "Kana"
  ID = "io.github.comradin.kana"
  Version = "0.0.0"
  Build = 1
  ```
  `Version` is only a default; packaging passes `--app-version`.
- `main.go` keeps `app.New()`. Packaged builds take the ID from the metadata anyway; for
  `go run`/`go build` an explicit `NewWithID` would only make Fyne create an unused
  preferences directory. The app uses neither Fyne Preferences nor Storage; `kana.db` stays
  where `store.DefaultPath` puts it.
- `fyne/Icon.png` (1024 × 1024) and its source `docs/icon/icon.svg`: a rounded square in the
  tile face colour `#eedfc0` with the tile shadow `#b8956a` along the bottom edge, and か in
  the kana text colour `#2c1a0e`, centred. The PNG is rendered from the SVG once and
  committed; the build needs no SVG tool. The maintainer sees a preview before it is
  committed.
- `.gitignore` gains `dist/`.

### Version in the app

- `fyne/version.go`:
  ```go
  // versionLabel returns the version to show: "v0.2.0" for a release
  // package, "dev" otherwise (go run, go build, or a package without a
  // real version).
  func versionLabel(meta fyne.AppMetadata) string
  ```
  "dev" when `!meta.Release` or the version is `""` or `"0.0.0"`; otherwise `"v" + version`
  (without doubling an existing `v`). Called with `fyne.CurrentApp().Metadata()`.
- Window title: `"Kana " + versionLabel(...)`, e.g. "Kana v0.1.0" or "Kana dev".
- Settings dialog: a line at the bottom "Version v0.1.0 · Releases", where "Releases" is a
  `widget.Hyperlink` to `https://github.com/Comradin/kana/releases`.

### README

- **Download:** link to the latest release, which file to pick per system, and how to start
  an unsigned build:
  - macOS: open the `.dmg`, drag Kana to Applications. On first start macOS blocks it; open
    System Settings → Privacy & Security and click "Open Anyway" (on older macOS:
    right-click → Open). Alternative: `xattr -dr com.apple.quarantine /Applications/Kana.app`.
  - Windows: unzip; at the SmartScreen warning "More info" → "Run anyway".
  - Linux (glibc 2.35+, e.g. Ubuntu 22.04 or newer): `tar xf kana-…-linux-x64.tar.xz`, then
    `make user-install` in the extracted folder (or run `usr/local/bin/kana` directly).
  - Verify a download: `sha256sum -c <file>.sha256` (macOS: `shasum -a 256 -c`).
- **Releasing (maintainers):** the release-please flow, which commit types trigger a release,
  the two repository settings, and the macOS script with prerequisites (Go from `go.mod`,
  Xcode command-line tools, `gh` logged in).

## Testing

- `versionLabel`: `{Release:false, Version:"0.0.1"}` → "dev"; `{Release:true, Version:""}` →
  "dev"; `{Release:true, Version:"0.0.0"}` → "dev"; `{Release:true, Version:"0.1.0"}` →
  "v0.1.0"; `{Release:true, Version:"v0.1.0"}` → "v0.1.0".
- The settings dialog's version line, if the dialog content is reachable in tests;
  otherwise the manual check.
- `ci.yml` proves itself on this branch's PR.
- `release.yml` dry run via the temporary branch trigger: both artifacts build, each
  `.sha256` verifies, the Linux archive contains `usr/local/bin/kana` and a `.desktop` file
  with `Exec=kana`, the Windows zip contains `Kana.exe`.
- `scripts/release-macos.sh --no-upload` on the maintainer's Mac before the first release:
  the `.dmg` mounts, Kana starts and shows "Kana dev" (no tag means version 0.0.0), and the
  working tree is clean afterwards.
- `shellcheck scripts/release-macos.sh` is clean.
- First real release: merging the first release PR produces v0.1.0 with the two CI assets
  and the macOS note; the maintainer then uploads the macOS build.

## Out of Scope

- Code signing with a Developer ID, notarisation, Windows signing.
- Intel Mac, Linux ARM, Android/iOS.
- Installers (MSI, Inno Setup), AppImage, Flatpak, Homebrew.
- Update checks or self-update.
- A macOS runner in CI (free for public repos, but not wanted for now; the script's steps
  could move into a job later).
