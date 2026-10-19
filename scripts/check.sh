#!/usr/bin/env bash
# One command to verify the whole repo before pushing (also what CI should run):
#   backend (Go) · shared admin UI · web admin · desktop UI + Rust agent.
# Optional: CHECK_DB_URL=postgres://… also runs the backend integration tests.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
step() { printf '\n\033[1m▶ %s\033[0m\n' "$*"; }

step "backend: gofmt · vet · test"
cd "$ROOT/apps/backend"
unformatted=$(gofmt -l .)
[[ -z "$unformatted" ]] || { echo "gofmt needed:"; echo "$unformatted"; exit 1; }
go vet ./...
go test ./...
if [[ -n "${CHECK_DB_URL:-}" ]]; then
  DATABASE_URL="$CHECK_DB_URL" go test -count=1 -tags integration ./internal/store/
fi

step "packages/admin-ui: typecheck"
cd "$ROOT/packages/admin-ui" && npx tsc --noEmit

step "web-admin: typecheck · build"
cd "$ROOT/apps/web-admin" && npx tsc --noEmit && npx vite build --logLevel error

step "desktop UI: typecheck · build"
cd "$ROOT/apps/desktop" && npx tsc --noEmit && npx vite build --logLevel error

step "desktop agent (Rust): check · test"
cd "$ROOT/apps/desktop/src-tauri" && cargo check --quiet && cargo test --quiet

printf '\n\033[32m✔ all checks passed\033[0m\n'
