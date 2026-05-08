#!/usr/bin/env bash
# Usage: ./testdata/repo-setup.sh <target-dir>
# Creates <target-dir>/repo (svnadmin repo) and <target-dir>/wc (working copy)
# with files in modified / added / deleted / untracked states.
set -euo pipefail

TARGET="${1:?target dir required}"
mkdir -p "$TARGET"
REPO="$TARGET/repo"
WC="$TARGET/wc"

rm -rf "$REPO" "$WC"

svnadmin create "$REPO"
svn checkout "file://$REPO" "$WC" >/dev/null

cd "$WC"
mkdir -p src
echo "original modified" > src/modified.sv
echo "to stay"            > src/unchanged.sv
echo "to delete"          > src/deleted.sv
svn add src >/dev/null
svn commit -m "initial" >/dev/null

# Produce a second revision so Log() has 2 entries.
echo "second rev" > src/unchanged.sv
svn commit -m "second revision" >/dev/null

# Now create the status states the tests expect.
echo "modified body" > src/modified.sv
echo "brand new"     > src/added.sv
svn add src/added.sv >/dev/null
svn rm src/deleted.sv >/dev/null
echo "untracked body" > src/untracked.sv

echo "$WC"
