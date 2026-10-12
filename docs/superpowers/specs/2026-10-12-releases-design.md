# Design: Versioning and Release Builds

**Date:** 2026-10-12
**Status:** Approved (design discussion)
**Branch:** `feat/releases` (from `main` after PR #12)

## Overview

Kana gets versions, a changelog and downloadable builds on GitHub Releases. Versions are
cut by release-please from the conventional commit messages the repository already uses. A
GitHub Actions workflow builds Windows and Linux packages for each release; the macOS package
is built and uploaded by the maintainer from their Mac with a script. Builds are unsigned for
now. The app shows its version.

## Decisions

| Topic | Decision |
|---|---|
| Platforms | macOS arm64 (Apple Silicon), Windows x64, Linux x64. No Intel Mac, Linux ARM or mobile. |
| Versioning | SemVer via release-please, release type `go`, `bump-minor-pre-major`: before 1.0, `feat:` bumps minor, `fix:` bumps patch. First release **v0.1.0**. |
| Release trigger | Merging the release-please PR creates the tag and the GitHub Release. |
| Windows/Linux builds | GitHub Actions on `windows-latest` and `ubuntu-latest`, with `fyne package`. |
| macOS build | Local script `scripts/release-macos.sh` on the maintainer's Mac; no macOS runner. |
| Packages | macOS `.dmg` with `kana.app`; Windows `.zip` with `kana.exe`; Linux `.tar.xz` from `fyne package`. Each with a `.sha256` file. |
| Signing | None. Steps are separated so signing and notarisation can be added later with secrets. |
| Version in the app | Window title and settings dialog, with a link to the releases page; "dev" for local builds. No network access. |
| Icon | A new icon in the tile style (か on the tile colour with its shadow), SVG source plus 1024 px PNG. |

## Release flow

1. Commits land on `main` through PRs as today.
2. `release.yml` runs release-please on every push to `main`. It opens or updates the release
   PR "chore(main): release 0.x.y" with `CHANGELOG.md` and the manifest version.
3. Merging that PR makes release-please create tag `v0.x.y` and the GitHub Release with the
   changelog section as notes.
4. In the same workflow run, the build jobs (only when a release was created) build the
   Windows and Linux packages and upload them with `gh release upload`. A final step appends
   to the release notes: "The macOS build is uploaded separately and may follow shortly."
5. The maintainer runs `scripts/release-macos.sh v0.x.y` on their Mac, which builds and
   uploads the `.dmg` and its `.sha256`.

## Repository files

### release-please

- `release-please-config.json`:
  ```json
  {
    "packages": {
      ".": {
        "release-type": "go",
        "package-name": "kana",
        "bump-minor-pre-major": true,
        "include-component-in-tag": false
      }
    },
    "bootstrap-sha": "da0eaba1deae2d0966722e19985cac5818da7f76"
  }
  ```
  `bootstrap-sha` is the full SHA of `main` after PR #12 (`da0eaba`). release-please then
  ignores the older history, and the first changelog lists only the commits of this work and
  later ones, so v0.1.0 starts with "release builds".
- `.release-please-manifest.json`: `{ ".": "0.0.0" }`, so the first `feat:` gives v0.1.0.
- Repository setting (manual, by the maintainer): Settings → Actions → General →
  "Allow GitHub Actions to create and approve pull requests". The README's maintainer
  section says so.

### `.github/workflows/ci.yml`

- On pull requests and pushes to `main`.
- One job on `ubuntu-latest`: `actions/setup-go` with `go-version-file: go.mod`, the Fyne
  build dependencies (`sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev`), then
  `test -z "$(gofmt -l .)"`, `go vet ./...`, `go test -count=1 ./...`.
- Release PRs opened by release-please with the default token do not trigger this workflow
  (GitHub's rule). Acceptable: they change only `CHANGELOG.md` and the manifest.

### `.github/workflows/release.yml`

- Triggers: `push` to `main`, and `workflow_dispatch` (dry run).
- Permissions: `contents: write`, `pull-requests: write`.
- **Job `release-please`** (push only): `googleapis/release-please-action@v4`; outputs
  `release_created` and `tag_name`.
- **Job `build`** (needs `release-please`; runs when `release_created == 'true'` or on
  `workflow_dispatch`): matrix

  | `os` | target | output |
  |---|---|---|
  | `windows-latest` | `windows` | `kana-<tag>-windows-x64.zip` containing `kana.exe` |
  | `ubuntu-latest` | `linux` | `kana-<tag>-linux-x64.tar.xz` from `fyne package` |

  Steps: checkout (at the tag for a release), setup-go from `go.mod`, Linux build deps on
  Ubuntu, install the Fyne CLI at the pinned version, `fyne package --os <target> --release
  --app-version <version> --src fyne`, rename to the asset name, write `<asset>.sha256`
  (`sha256sum` on Linux, `Get-FileHash` or Git Bash `sha256sum` on Windows; the file holds
  `<hex>  <asset>`), then:
  - release: `gh release upload <tag> <asset> <asset>.sha256`;
  - dry run: `actions/upload-artifact` with both files; `<tag>` is `dev-<short sha>` and the
    version `0.0.0`.
  Windows uses the preinstalled MinGW gcc for CGO; the step sets `shell: bash` where
  convenient.
- **Job `notes`** (needs `build`, release only): appends the macOS note to the release body
  with `gh release edit <tag> --notes "<existing body>\n\n<note>"` (read via
  `gh release view --json body`).
- The Fyne CLI version is pinned in one place: an `env: FYNE_CLI_VERSION` at the top of the
  workflow; the script uses the same value, read from a small file `scripts/fyne-cli-version`
  (one line, e.g. `v1.7.0` — the plan picks the current release of `fyne.io/tools` and checks
  it packages Fyne v2.7). The workflow reads that file too, so there is exactly one source.

### `scripts/release-macos.sh`

Bash, `set -euo pipefail`. Usage: `scripts/release-macos.sh [--no-upload] [vX.Y.Z]`.

1. Requires macOS (`uname -s` = `Darwin`) and arm64; otherwise exits with a message.
2. Tag: the argument, or without one the newest release (`gh release view --json tagName`).
   With `--no-upload` and no tag: version `0.0.0`, name `dev`.
3. Checks (unless `--no-upload`): `gh auth status` succeeds; the tag exists; the working
   tree is clean; `HEAD` is exactly the tag (`git describe --exact-match --tags` equals it),
   otherwise it tells the user to `git checkout <tag>`.
4. Installs the Fyne CLI: `go install fyne.io/tools/cmd/fyne@$(cat scripts/fyne-cli-version)`.
5. Builds in a temporary directory: `fyne package --os darwin --release --app-version
   <version> --src fyne`, giving `Kana.app`.
6. Builds the `.dmg` with `hdiutil create -volname "Kana" -srcfolder <dir with Kana.app and
   an /Applications symlink> -ov -format UDZO kana-<tag>-macos-arm64.dmg`, then writes
   `kana-<tag>-macos-arm64.dmg.sha256` with `shasum -a 256`.
7. Uploads both with `gh release upload <tag> … --clobber` (unless `--no-upload`), and
   prints where the files are.

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
  `Version` here is only the default; packaging passes `--app-version`.
- `fyne/Icon.png` (1024 × 1024) and its source `docs/icon/icon.svg`: a rounded square in the
  tile face colour `#eedfc0` with the tile shadow `#b8956a` along the bottom edge, and か in
  the kana text colour `#2c1a0e`, centred. The PNG is rendered from the SVG once and
  committed; the build does not need an SVG tool. The maintainer sees a preview before it is
  committed.
- `main.go` switches from `app.New()` to `app.NewWithID("io.github.comradin.kana")`, matching
  the ID. This changes nothing for user data: the database path does not depend on the app ID.

### Version in the app

- `fyne/version.go`:
  ```go
  // versionLabel returns the version to show: "v0.2.0", or "dev" for an
  // unpackaged build (empty or 0.0.0 metadata version).
  func versionLabel(meta string) string
  ```
  Read from `fyne.CurrentApp().Metadata().Version`.
- Window title: `"Kana " + versionLabel(...)`, e.g. "Kana v0.1.0" or "Kana dev".
- Settings dialog: a line at the bottom "Version v0.1.0 · Releases" where "Releases" is a
  `widget.Hyperlink` to `https://github.com/Comradin/kana/releases`.

### README

- **Download:** links to the latest release, which file to pick per system, and how to start
  an unsigned build:
  - macOS: open the `.dmg`, drag Kana to Applications; on first start right-click → Open, or
    `xattr -dr com.apple.quarantine /Applications/Kana.app`.
  - Windows: unzip; at the SmartScreen warning "More info" → "Run anyway".
  - Linux: `tar xf kana-…-linux-x64.tar.xz` then `make user-install` (or run the binary from
    `usr/local/bin`).
  - Verify with the `.sha256` file.
- **Releasing (maintainers):** the release-please flow, the repository setting, and the
  macOS script with prerequisites (Go from `go.mod`, Xcode command-line tools, `gh` logged in).

## Testing

- `versionLabel`: `""` → "dev", `"0.0.0"` → "dev", `"0.2.0"` → "v0.2.0", `"v0.2.0"` → "v0.2.0".
- The settings dialog shows the version line (existing settings tests' style, if the dialog
  content is reachable; otherwise covered by the manual check).
- `ci.yml` proves itself on this branch's PR.
- `release.yml` is tested by a `workflow_dispatch` dry run on the branch before merging: both
  artifacts build, each `.sha256` verifies (`sha256sum -c`).
- `scripts/release-macos.sh --no-upload` is run by the maintainer on their Mac before the first
  release; the `.dmg` mounts and Kana starts.
- `shellcheck scripts/release-macos.sh` is clean.
- First real release: merging the first release PR produces v0.1.0 with the two CI assets;
  the maintainer then uploads the macOS build.

## Out of Scope

- Code signing and notarisation (macOS Developer ID, Windows signing).
- Intel Mac, Linux ARM, Android/iOS.
- Installers (MSI, Inno Setup), AppImage, Flatpak, Homebrew.
- Update checks or self-update.
- A macOS runner in CI (free for public repos, but not wanted for now; the script's steps
  could be moved into a job later).
