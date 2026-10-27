#!/usr/bin/env bash
# Puts player notes into an open release-please PR: its body and the CHANGELOG on its branch.
#
# Usage: scripts/release-notes.sh <pr-number> <notes.md>
#
# The notes go right under the version heading; release-please's commit list moves under
# "### Commits", its sections become "#### New" and "#### Improvements and fixes".
set -euo pipefail

usage() {
	cat <<'USAGE'
Usage: scripts/release-notes.sh <pr-number> <notes.md>

Inserts the player notes from notes.md under the version heading of the open
release-please PR, in the PR body and in CHANGELOG.md on the PR's branch, and
moves the generated commit list under "### Commits". The CHANGELOG change is
committed as "docs: describe X.Y.Z for players" and pushed to the PR branch.

  -h, --help    show this help

Run it again after release-please rewrites the PR (a new commit on main).
USAGE
}

die() {
	echo "error: $*" >&2
	exit 1
}

case "${1:-}" in
-h | --help)
	usage
	exit 0
	;;
esac
[[ $# -eq 2 ]] || {
	usage >&2
	exit 2
}
pr="$1"
notes="$2"
[[ -s "$notes" ]] || die "no notes in $notes"
notes="$(cd "$(dirname "$notes")" && pwd)/$(basename "$notes")"

repo="$(git rev-parse --show-toplevel)"
branch="$(gh pr view "$pr" --json headRefName,state -q 'select(.state == "OPEN") | .headRefName')"
[[ -n "$branch" ]] || die "PR #$pr is not open"
[[ "$branch" == release-please--* ]] || die "PR #$pr is not a release-please PR ($branch)"

work="$(mktemp -d)"
cleanup() {
	git -C "$repo" worktree remove --force "$work/tree" 2>/dev/null || true
	rm -rf "$work"
}
trap cleanup EXIT

# insert TEXT_FILE: rewrites the first version section of TEXT_FILE in place and prints
# the version. Fails when the section already has "### Commits".
insert() {
	python3 -I - "$1" "$notes" <<'PY' || return 1
import re, sys
path, notes_path = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as fh:
    text = fh.read()
with open(notes_path, encoding="utf-8") as fh:
    notes = fh.read().strip()
head = re.search(r"^## \[(\d+\.\d+\.\d+)\].*$", text, re.M)
if not head:
    sys.exit("no version heading in " + path)
end = re.compile(r"^(## |---)", re.M).search(text, head.end())
stop = end.start() if end else len(text)
section = text[head.end():stop]
if re.search(r"^### Commits$", section, re.M):
    sys.exit(path + " already has player notes")
first = re.search(r"^### ", section, re.M)
if not first:
    sys.exit("no commit list under " + head.group(0))
commits = re.sub(r"^### ", "#### ", section[first.start():], flags=re.M).strip("\n")
section = "\n\n" + notes + "\n\n### Commits\n\n" + commits + "\n\n"
text = text[:head.end()] + section + text[stop:].lstrip("\n")
with open(path + ".new", "w", encoding="utf-8") as fh:
    fh.write(text)
print(head.group(1))
PY
	mv "$1.new" "$1"
}

git -C "$repo" fetch -q origin "$branch"
git -C "$repo" worktree add -q --detach "$work/tree" "origin/$branch"
version="$(insert "$work/tree/CHANGELOG.md")"

gh pr view "$pr" --json body -q .body >"$work/body.md"
body_version="$(insert "$work/body.md")"
[[ "$body_version" == "$version" ]] || die "PR body is for $body_version, CHANGELOG for $version"

git -C "$work/tree" commit -q -m "docs: describe $version for players" CHANGELOG.md
git -C "$work/tree" push -q origin "HEAD:$branch"
gh pr edit "$pr" --body-file "$work/body.md" >/dev/null
echo "PR #$pr: player notes for $version in the body and in CHANGELOG.md on $branch"
