#!/usr/bin/env bash
# Builds the macOS (Apple Silicon) package of Kana and uploads it to a GitHub release.
#
# Usage: scripts/release-macos.sh [--no-upload] [vX.Y.Z]
#
# Without a tag the newest release is used. --no-upload makes a local trial build
# in dist/ and skips all release checks; without a tag it builds version 0.0.0 ("dev").
set -euo pipefail

usage() {
	cat <<'USAGE'
Usage: scripts/release-macos.sh [--no-upload] [vX.Y.Z]

Builds Kana.app with fyne package, signs it ad hoc, wraps it in a dmg with a
SHA-256 checksum in dist/ and uploads both to the GitHub release for the tag.

  --no-upload   build locally only; skip the release checks and the upload
  -h, --help    show this help

Without a tag the newest GitHub release is used (with --no-upload: version 0.0.0).
USAGE
}

die() {
	echo "error: $*" >&2
	exit 1
}

upload=1
tag=""
while (($# > 0)); do
	case "$1" in
	--no-upload) upload=0 ;;
	-h | --help)
		usage
		exit 0
		;;
	-*)
		usage >&2
		exit 2
		;;
	*)
		[[ -z "$tag" ]] || {
			usage >&2
			exit 2
		}
		tag="$1"
		;;
	esac
	shift
done

# 1. Platform.
[[ "$(uname -s)" == "Darwin" ]] || die "this script requires macOS"
[[ "$(uname -m)" == "arm64" ]] || die "this script requires an Apple Silicon (arm64) Mac"

# 2. Repository root.
repo="$(git rev-parse --show-toplevel)" || die "not inside the kana git repository"

# The EXIT trap only undoes what has actually been done.
tmp=""
restore_toml=0
cleanup() {
	if [[ -n "$tmp" ]]; then
		rm -rf "$tmp"
	fi
	if ((restore_toml)); then
		git -C "$repo" checkout -- fyne/FyneApp.toml
	fi
}
trap cleanup EXIT

# 3. Tag and version.
if [[ -z "$tag" ]]; then
	if ((upload)); then
		command -v gh >/dev/null || die "the GitHub CLI (gh) is not installed"
		tag="$(gh release view --json tagName -q .tagName)" ||
			die "could not find the newest release; give the tag as an argument, e.g. v0.1.0"
	else
		tag="dev"
	fi
fi
if [[ "$tag" == "dev" ]]; then
	version="0.0.0"
else
	version="${tag#v}"
fi

# 4. Release checks.
if ((upload)); then
	command -v gh >/dev/null || die "the GitHub CLI (gh) is not installed"
	gh auth status >/dev/null 2>&1 || die "gh is not logged in; run: gh auth login"
	git -C "$repo" fetch --tags || die "git fetch --tags failed"
	git -C "$repo" rev-parse -q --verify "refs/tags/$tag" >/dev/null ||
		die "tag $tag does not exist"
	[[ -z "$(git -C "$repo" status --porcelain --untracked-files=no)" ]] ||
		die "the working tree has uncommitted changes; commit or stash them first"
	current="$(git -C "$repo" describe --exact-match --tags 2>/dev/null || true)"
	[[ "$current" == "$tag" ]] ||
		die "HEAD is not at $tag; run: git checkout $tag"
fi

# 5. Fyne CLI.
command -v go >/dev/null || die "Go is not installed"
fyne_version="$(<"$repo/scripts/fyne-cli-version")"
go install "fyne.io/tools/cmd/fyne@$fyne_version" || die "installing the Fyne CLI failed"
fyne="$(go env GOPATH)/bin/fyne"

# 6. Package Kana.app (fyne package bumps Build in FyneApp.toml; the trap restores it).
tmp="$(mktemp -d)"
restore_toml=1
(cd "$tmp" && "$fyne" package --os darwin --release --app-version "$version" \
	--src "$repo/fyne" --executable kana) || die "fyne package failed"
app="$tmp/Kana.app"
[[ -d "$app" ]] || die "fyne package did not produce Kana.app"

# 7. Ad-hoc signature.
codesign --force --deep -s - "$app" || die "codesign failed"

# 8. Disk image and checksum.
staging="$tmp/staging"
mkdir -p "$staging" "$repo/dist"
mv "$app" "$staging/"
ln -s /Applications "$staging/Applications"
dmg_name="kana-$tag-macos-arm64.dmg"
dmg="$repo/dist/$dmg_name"
hdiutil create -volname Kana -srcfolder "$staging" -ov -format UDZO "$dmg" ||
	die "hdiutil create failed"
(cd "$repo/dist" && shasum -a 256 "$dmg_name" >"$dmg_name.sha256")

# 9. Upload.
if ((upload)); then
	gh release upload "$tag" "$dmg" "$dmg.sha256" --clobber || die "upload to release $tag failed"
fi

echo "$dmg"
echo "$dmg.sha256"
