# Local setup on Ubuntu/Linux (backend + web admin)

Baseline: upstream `ngnclht1102/bibo-emplooyee-tracking` @ `f41cdc4f80c13c7c0a3c33775d08d7b2375c6902` (cloned 2026-10-19).

The desktop agent has no Linux backend (`src-tauri/src/platform/` is macOS + Windows only);
build and test it on Windows.

## Toolchain (user-local, no sudo)

- Go 1.26.x in `~/.local/go` (matches `apps/backend/go.mod`)
- pnpm 10.30.1 via `corepack enable --install-directory ~/.local/bin pnpm`
- Node >= 20, Docker
- PATH: `export PATH="$HOME/.local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH"`

## Ports

| Service            | Port |
|--------------------|------|
| Go backend         | 8090 (matches scripts + Vite proxy; `.env.example` says 8080) |
| web-admin (Vite)   | 5174, `/admin/` |
| Docker Postgres 16 | 5433 when a system Postgres already owns 5432 (`PG_PORT=5433`) |

## Run

```sh
pnpm install
PG_PORT=5433 scripts/dev-db.sh      # once per boot
# apps/backend/.env (gitignored): PORT=8090, DATABASE_URL=...localhost:5433/ctracking..., random JWT_SECRET
scripts/dev-webadmin.sh             # backend + web admin
curl localhost:8090/healthz         # {"status":"ok"}
```

Open http://localhost:5174/admin/signup to create a disposable test owner. Use fake data only.
