# PROJECT KNOWLEDGE BASE

**Generated:** 2026-02-17 14:10:00 +08:00
**Commit:** 6bdf6be
**Branch:** master

## OVERVIEW
`bamboo-base-go-template` is a backend scaffold built on `bamboo-base-go`.
It wires startup nodes (DB, Redis, seed), Gin routes, and layered business modules (`handler -> logic -> repository`).

## STRUCTURE
```text
./
├── main.go                     # Entry; delegates lifecycle to xMain.Runner
├── api/                        # Request/response DTOs
├── internal/
│   ├── app/route/              # Router + middleware binding
│   ├── app/startup/            # Infra init and startup node registration
│   ├── handler/                # HTTP handlers (thin controller layer)
│   ├── logic/                  # Business orchestration
│   ├── repository/             # DB/Redis access
│   ├── entity/                 # GORM entities
│   └── constant/               # Shared business constants
└── Makefile                    # Dev/test/fmt commands
```

## WHERE TO LOOK
| Task | Location | Notes |
|---|---|---|
| Add API endpoint | `internal/app/route/`, `internal/handler/` | Register route first, then handler |
| Add business logic | `internal/logic/` | Keep handler thin |
| Add persistence logic | `internal/repository/` | Return `*xError.Error` from repo methods |
| Add/modify entity | `internal/entity/`, `internal/app/startup/startup_database.go` | Update `migrateTables` with dependency order |
| Add startup capability | `internal/app/startup/startup.go` + `startup_*.go` | Register as `xRegNode.RegNodeList` |
| Seed default data | `internal/app/startup/prepare/` | Must be idempotent |
| Tune config/env | `.env.example`, `internal/app/startup/*.go` | Always provide defaults in `xEnv.GetEnv*` |

## CODE MAP
LSP workspace view is unavailable in this repo; map built from code scan.

| Symbol | Type | Location | Role |
|---|---|---|---|
| `main` | func | `main.go` | Register startup, run app |
| `Init` | func | `internal/app/startup/startup.go` | Startup node list factory |
| `NewRoute` | func | `internal/app/route/route.go` | Global middleware + route group |
| `NewHandler[T]` | generic func | `internal/handler/handler.go` | Handler construction pattern |
| `HealthLogic.Ping` | method | `internal/logic/health.go` | Service health orchestration |
| `HealthRepo.DatabaseReady` | method | `internal/repository/health.go` | DB readiness check |

## CONVENTIONS
- Import aliases: bamboo-base-go packages use `x*` aliases (`xLog`, `xEnv`, `xError`, `xResult`, `xReg`, ...).
- Layering is strict: route -> handler -> logic -> repository; skip-layer calls are not used.
- Context DI pattern: startup registers infra in context; logic retrieves with `xCtxUtil.MustGetDB/MustGetRDB`.
- Response pattern: handlers return success via `xResult.SuccessHasData`; errors are passed through `ctx.Error`.
- Error type pattern: repo/logic use `*xError.Error` for business/infrastructure failures.
- Env key families: `XLF_*`, `APP_*`, `DATABASE_*`, `NOSQL_*`, `SNOWFLAKE_*`.

## ANTI-PATTERNS (THIS PROJECT)
- Do not call repository directly from route or bypass logic layer.
- Do not use `os.Getenv` directly; use `xEnv.GetEnv*` with defaults.
- Do not write raw Gin JSON responses in handlers when `xResult` helpers are available.
- Do not create DB/Redis clients inside logic/repository constructors; get injected dependencies from startup/context.
- Do not add new entities without appending them to `migrateTables`.

## UNIQUE STYLES
- Log naming follows module tags (`NamedMAIN`, `NamedINIT`, `NamedCONT`, `NamedLOGC`, `NamedREPO`).
- Startup seed phase is explicit (`xCtx.Exec` node) and separated into `prepare/`.
- Entity IDs follow snowflake gene strategy; entity-level gene binding is expected.

## COMMANDS
```bash
cp .env.example .env
go mod tidy
make run
make test
make fmt
curl http://localhost:8080/api/v1/health/ping
```

## NOTES
- No CI workflow exists yet (`.github/workflows` absent).
- Test command exists but test files are currently not scaffolded.
- Keep AGENTS hierarchy concise; domain details belong to `internal/AGENTS.md` and deeper scoped files.
