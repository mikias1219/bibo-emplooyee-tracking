#!/usr/bin/env bash
# Install the Focused App GNOME Shell extension for the current user (no sudo).
# On Wayland, GNOME only discovers new extensions at login: log out and back in
# once, then this script (re-run) or the Extensions app can enable it.
set -euo pipefail
UUID="focused-app@workforce-intelligence"
SRC="$(cd "$(dirname "$0")" && pwd)/gnome-extension/$UUID"
DEST="$HOME/.local/share/gnome-shell/extensions/$UUID"
mkdir -p "$DEST"
cp "$SRC/metadata.json" "$SRC/extension.js" "$DEST/"
echo "→ installed to $DEST"
if gnome-extensions enable "$UUID" 2>/dev/null; then
  echo "→ enabled"
else
  echo "→ GNOME hasn't loaded it yet: log out and back in, then run this script again."
fi
