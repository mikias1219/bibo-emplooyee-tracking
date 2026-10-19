#!/usr/bin/env bash
# Run the desktop app (Tauri dev). Points it at the web-admin dev server (Vite
# proxies /v1 → backend :8090), so API calls and the "open web admin" link resolve
# in one knob. Override CTRACKING_BACKEND_URL to hit the backend (:8090) directly.
# Sign in with a Selamnew organization (Org&Emp) account — there is no signup.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/apps/desktop"

# Terminals inside the VS Code *snap* inject the snap's own GTK/GIO paths and
# gsettings schemas, which crash GTK apps started from them ("schema ... does not
# contain a key named 'antialiasing'"). Restore the host desktop environment.
if [[ "${SNAP_NAME:-}" == "code" ]]; then
  unset GSETTINGS_SCHEMA_DIR GTK_PATH GTK_EXE_PREFIX GTK_IM_MODULE_FILE GIO_MODULE_DIR \
        LOCPATH GDK_PIXBUF_MODULE_FILE GDK_PIXBUF_MODULEDIR
  [[ -n "${XDG_DATA_DIRS_VSCODE_SNAP_ORIG:-}" ]] && export XDG_DATA_DIRS="$XDG_DATA_DIRS_VSCODE_SNAP_ORIG"
  [[ -n "${XDG_CONFIG_DIRS_VSCODE_SNAP_ORIG:-}" ]] && export XDG_CONFIG_DIRS="$XDG_CONFIG_DIRS_VSCODE_SNAP_ORIG"
fi

export CTRACKING_BACKEND_URL="${CTRACKING_BACKEND_URL:-http://localhost:5174}"
echo "→ desktop (tauri dev) → backend at $CTRACKING_BACKEND_URL"
echo "   first build compiles Rust — give it a minute."
pnpm tauri dev
