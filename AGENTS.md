# PROJECT KNOWLEDGE BASE

**Generated:** 2026-05-24 00:00:00 +08:00
**Commit:** 9be247d
**Branch:** master

## OVERVIEW
`frontleaves-plugin` 是 FrontLeaves MC 的**外部独立中枢服务**——运行于 Minecraft 服务端进程之外，作为 MC 服务端、玩家与网页端三者之间的统一媒介。

核心定位与解决痛点：
- **桥接隔离**：MC 服务端（Java/Bukkit）不直接操作数据库，也不承载网页端 API，所有数据交互均通过本服务中转。
- **计算卸载**：部分原本由 MC 服务端实时处理的高开销计算（调度、统计、数据聚合等）在本服务完成后下发结果，MC 端仅做应用，规避 Java 单线程主线程的性能瓶颈。
- **网页赋能**：为网页端用户提供 RESTful API，使玩家能在浏览器中完成原本需要进入 MC 客户端才能进行的操作。

技术层面：Go 后端，基于 `bamboo-base-go` 脚手架，严格分层架构（`handler → logic → repository`），双协议入口（Gin HTTP REST + gRPC），通过 gRPC stream 与 MC 插件层保持实时通信。

## STRUCTURE
```text
./
├── main.go                     # Entry; runs xMain.Runner with Gin + gRPC tasks
├── api/                        # Request/response DTOs (per domain subdirectory)
├── proto/                      # Buf protobuf definitions + buf.gen.yaml
├── internal/
│   ├── app/
│   │   ├── grpc/               # Auth gRPC client (yggleaf inter-service)
│   │   ├── middleware/          # Gin middleware (LoginAuth, RequireRole)
│   │   ├── route/              # Route groups + middleware binding
│   │   └── startup/            # Infra init + startup node registration
│   ├── handler/                # HTTP handlers (thin controller layer)
│   ├── logic/                  # Business orchestration + scheduler engine
│   ├── repository/             # DB/Redis/Cache access
│   │   └── cache/              # Redis hash-based cache managers
│   ├── entity/                 # GORM entities with snowflake gene binding
│   ├── constant/               # Shared constants (cache keys, env keys, gene numbers)
│   ├── grpc/
│   │   ├── handler/            # gRPC service handlers (stream + unary)
│   │   ├── middleware/          # gRPC interceptor (PluginVerify)
│   │   ├── register/           # Service registration entry
│   │   └── gen/                # Generated protobuf Go code (buf output)
│   ├── sse/                    # Server-Sent Events (chat stream)
│   └── util/                   # Shared utilities (markdown)
├── docs/                       # Swagger generated docs
├── script/                     # Docker build + server deploy scripts
└── Makefile                    # Dev/test/proto/docker commands
```

## WHERE TO LOOK
| Task | Location | Notes |
|---|---|---|
| Add REST endpoint | `internal/app/route/route_*.go` + `internal/handler/` | Register route first, then handler |
| Add gRPC service | `proto/*.proto` → `internal/grpc/handler/` + `register/register.go` | Run `make proto` after proto change |
| Add business logic | `internal/logic/` | Keep handler thin |
| Add persistence | `internal/repository/` | Return `*xError.Error` from repo methods |
| Add cache layer | `internal/repository/cache/` | Use `xCache.Cache` pattern with Redis hash |
| Add/modify entity | `internal/entity/` + `startup_database.go` | Update `migrateTables` with FK dependency order |
| Add startup node | `internal/app/startup/startup.go` | Append to `xRegNode.RegNodeList` in `Init()` |
| Add middleware (HTTP) | `internal/app/middleware/` | Use `xLog.NamedMIDE` |
| Add middleware (gRPC) | `internal/grpc/middleware/` | Unary + Stream interceptor pair |
| Add env config | `internal/constant/environment.go` | Always provide defaults via `xEnv.GetEnv*` |
| Add Snowflake gene | `internal/constant/gene_number.go` | One gene per entity type |
| Add Redis cache key | `internal/constant/cache.go` | Use `RedisKey.Get()` for prefix + formatting |

## CODE MAP

| Symbol | Type | Location | Role |
|---|---|---|---|
| `main` | func | `main.go` | Register startup + gRPC task, run Runner |
| `Init` | func | `internal/app/startup/startup.go` | Startup node list factory (DB → Redis → AuthClient → Seed) |
| `NewRoute` | func | `internal/app/route/route.go` | Gin global middleware + route groups |
| `NewHandler[T]` | generic func | `internal/handler/handler.go` | HTTP handler construction with auto-wired service deps |
| `RegisterGRPCServices` | func | `internal/grpc/register/register.go` | gRPC service registration + scheduler engine init |
| `NewAuthClient` | func | `internal/app/grpc/client.go` | yggleaf inter-service gRPC client |
| `LoginAuth` | middleware | `internal/app/middleware/login_auth.go` | Bearer token → cache → gRPC validate → async profile sync |
| `UnaryPluginVerify` | interceptor | `internal/grpc/middleware/plugin_verify.go` | gRPC plugin secret-key auth |
| `SchedulerEngine` | struct | `internal/logic/announcement_scheduler_engine.go` | Cron-like announcement push engine |

## CONVENTIONS
- Import aliases: bamboo-base-go packages use `x*` prefixes (`xLog`, `xEnv`, `xError`, `xResult`, `xReg`, `xMain`, `xCtxUtil`, ...).
- Internal constants use `bConst` alias for `internal/constant` package.
- Layering is strict: route → handler → logic → repository; skip-layer calls forbidden.
- Context DI: startup registers infra in context; logic/repo retrieve via `xCtxUtil.MustGetDB/MustGetRDB`.
- Handler pattern: `NewHandler[T]` auto-wires all `*Logic` deps; each handler file defines its own type embedding `handler`.
- gRPC handler pattern: similar to HTTP but with `grpcHandler` base struct + service-specific sub-structs.
- Response: HTTP uses `xResult.SuccessHasData` / `xResult.AbortError`; gRPC uses `xError.NewError` / standard status.
- Error type: repo/logic return `*xError.Error` everywhere, never raw `error`.
- Cache pattern: `repository/cache/` uses Redis hash with `xCache.Cache` type alias + `RedisKey.Get()` prefix.
- Env key families: `XLF_*`, `APP_*`, `DATABASE_*`, `NOSQL_*`, `SNOWFLAKE_*`, `YGGLEAF_*`.
- Log naming: `NamedMAIN`, `NamedINIT`, `NamedCONT`, `NamedLOGC`, `NamedREPO`, `NamedMIDE`, `NamedGRPC`.

## ANTI-PATTERNS (THIS PROJECT)
- Do not call repository directly from route or bypass logic layer.
- Do not use `os.Getenv` directly; use `xEnv.GetEnv*` with defaults.
- Do not write raw Gin JSON responses in handlers when `xResult` helpers are available.
- Do not create DB/Redis clients inside logic/repository constructors; get injected deps from startup/context.
- Do not add new entities without appending them to `migrateTables` in dependency order.
- Do not edit files in `internal/grpc/gen/` — they are auto-generated by `buf generate`.
- Do not place business constants inside handler/logic files; keep in `constant/`.
- Do not construct ad-hoc handlers bypassing `NewHandler[T]` or gRPC handler patterns.

## UNIQUE STYLES
- Dual protocol: Gin (HTTP REST) + gRPC server run concurrently via `xMain.Runner`.
- Auth flow: Bearer token → Redis cache check → gRPC call to yggleaf → async profile sync via `xAsync.Async`.
- Plugin auth: gRPC metadata `plugin-secret-key` verified via `PluginCredentialLogic`.
- Announcement scheduler: `SchedulerEngine` with DB recovery on startup, pushes via gRPC stream.
- SSE: `internal/sse/chat_stream.go` manages in-process client registry for real-time chat.
- Entity IDs: Snowflake gene strategy with per-entity gene numbers defined in `constant/gene_number.go`.

## COMMANDS
```bash
cp .env.example .env
go mod tidy
make dev          # swag init + run (recommended)
make run          # run only
make swag         # regenerate Swagger docs
make proto-init   # symlink base.proto from bamboo-base-go
make proto        # buf generate protobuf Go code
make tidy         # go mod tidy
make docker USER=x PASS=x [VERSION=x]  # build + push Docker image
make upload DEPLOY_SERVER=x             # deploy to server
```

## NOTES
- No CI workflow (`.github/workflows` absent); deployment via `make docker` + `make upload`.
- Tests exist only for gRPC handlers (`internal/grpc/handler/*_test.go`).
- Proto generation requires `buf` CLI + `make proto-init` symlink step.
- `docs/` is Swagger auto-generated; do not edit manually.
- Auth depends on `frontleaves-yggleaf` gRPC service running (inter-service call).
- Keep AGENTS hierarchy concise; domain details in `internal/AGENTS.md` and `internal/grpc/AGENTS.md`.
