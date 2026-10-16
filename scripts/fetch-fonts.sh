#!/bin/sh
# Downloads the kana fonts that ship with the app into fyne/fonts/ and checks
# each file against the SHA-256 sum recorded here. Run from the repository
# root. The files are committed; run this only to restore or verify them, or
# after changing a pinned version (then update the sums below).
#
#   Noto Sans JP:      notofonts/noto-cjk, tag Sans2.004, Sans/SubsetOTF/JP
#   M PLUS Rounded 1c: google/fonts, commit 84efd8ad78c3, ofl/mplusrounded1c
#                      (its OFL.txt lives in ofl/roundedmplus1c)
# Both are licensed under the SIL Open Font License 1.1.
set -eu

NOTO=https://raw.githubusercontent.com/notofonts/noto-cjk/Sans2.004
GFONTS=https://raw.githubusercontent.com/google/fonts/84efd8ad78c3710ad14bd909e3bc407151885628/ofl
DIR=fyne/fonts

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	else
		shasum -a 256 "$1" | cut -d ' ' -f 1
	fi
}

# fetch URL FILE SHA256 downloads URL to $DIR/FILE if the sum matches.
fetch() {
	tmp="$DIR/$2.tmp"
	curl -fsSL -o "$tmp" "$1"
	got=$(sha256 "$tmp")
	if [ "$got" != "$3" ]; then
		rm -f "$tmp"
		echo "$2: SHA-256 $got, expected $3" >&2
		exit 1
	fi
	mv "$tmp" "$DIR/$2"
	echo "$2 ok"
}

mkdir -p "$DIR"
fetch "$NOTO/Sans/SubsetOTF/JP/NotoSansJP-Regular.otf" NotoSansJP-Regular.otf \
	dff723ba59d57d136764a04b9b2d03205544f7cd785a711442d6d2d085ac5073
fetch "$NOTO/Sans/SubsetOTF/JP/NotoSansJP-Bold.otf" NotoSansJP-Bold.otf \
	1b0edfb500b73a4fa8a4fcaae1bbbd403994e08e73e3e0da37e70d3853f42c5f
fetch "$NOTO/LICENSE" OFL-NotoSansCJK.txt \
	6a73f9541c2de74158c0e7cf6b0a58ef774f5a780bf191f2d7ec9cc53efe2bf2
fetch "$GFONTS/mplusrounded1c/MPLUSRounded1c-Regular.ttf" MPLUSRounded1c-Regular.ttf \
	b75708b53e45b06d17d470aeeca5b766e3d1b3999f03f13ec4eb863ca846c14c
fetch "$GFONTS/mplusrounded1c/MPLUSRounded1c-Bold.ttf" MPLUSRounded1c-Bold.ttf \
	c358630584e8e2d8fbd6121d0f4693255ffef6d1e6d4f3441fd6e5a963a11f9e
fetch "$GFONTS/roundedmplus1c/OFL.txt" OFL-MPLUSRounded1c.txt \
	67f64c5509e5151796599e3ad47c3131cbe0c80c4f9430b90236a1249c2eacc9
