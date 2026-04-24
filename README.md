# FrontLeaves Plugin

FrontLeaves MC 服务器的**插件中枢**——集中处理数据库操作与业务逻辑的后端服务。

插件（Java/Bukkit）不进行数据库处理与详细计算，通过 gRPC 与本服务对接；网页端用户通过 RESTful API 访问。

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                    网页端用户                              │
│            (RESTful API / Swagger)                       │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP (Gin)
┌──────────────────────▼──────────────────────────────────┐
│              frontleaves-plugin                          │
│         ┌──────────────────────────┐                     │
│         │  Route → Handler → Logic │                     │
│         │       → Repository      │                     │
│         └──────────┬───────────────┘                     │
│                    │                                     │
│         ┌──────────┼──────────┐                          │
│         │          │          │                          │
│    PostgreSQL   Redis     gRPC Client                    │
│    (GORM)       (go-redis)                               │
└──────────────────┼──────────────────────────────────────┘
                   │ gRPC
┌──────────────────▼──────────────────────────────────────┐
│              Minecraft 插件层                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │server-status│  │  plugin-b   │  │  plugin-c   │ ...  │
│  └─────────────┘  └─────────────┘  └─────────────┘      │
└─────────────────────────────────────────────────────────┘
```

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.25 |
| Web 框架 | Gin v1.12 |
| ORM | GORM v1.31 |
| 数据库 | PostgreSQL |
| 缓存 | Redis (go-redis/v9) |
| API 文档 | Swagger (swaggo/swag) |
| 基础框架 | bamboo-base-go |
| ID 生成 | Snowflake |

## 快速开始

### 环境要求

- Go 1.25+
- PostgreSQL 14+
- Redis 7+

### 配置

```bash
cp .env.example .env
# 编辑 .env 填入实际的数据库和 Redis 连接信息
```

### 运行

```bash
# 安装依赖
go mod tidy

# 推荐：生成 Swagger 文档 + 运行
make dev

# 或单独运行
make run
```

### 验证

```bash
curl http://localhost:8080/api/v1/health/ping
```

Swagger UI（需开启 `XLF_DEBUG=true`）：`http://localhost:8080/swagger/index.html`

## 目录结构

```
.
├── main.go                          # 入口：注册启动节点，委托 xMain.Runner
├── Makefile                         # 开发命令
├── .env.example                     # 环境变量模板
│
├── api/                             # 请求/响应 DTO
│   └── health/
│       └── health.go                # 健康检查响应结构
│
├── docs/                            # Swagger 生成产物（自动生成）
│
└── internal/                        # 私有业务代码
    ├── app/
    │   ├── route/                   # 路由注册 + 中间件
    │   │   ├── route.go             # 主路由入口
    │   │   ├── route_health.go      # 健康检查路由
    │   │   └── route_swagger.go     # Swagger UI 路由
    │   └── startup/                 # 基础设施初始化
    │       ├── startup.go           # 启动节点列表
    │       ├── startup_database.go  # PostgreSQL 连接 + 自动迁移
    │       ├── startup_redis.go     # Redis 连接
    │       ├── startup_prepare.go   # 数据准备编排
    │       └── prepare/             # 种子数据（幂等）
    │           ├── prepare.go
    │           └── prepare_role.go  # 默认角色种子
    │
    ├── handler/                     # HTTP 处理层（薄控制器）
    │   ├── handler.go               # NewHandler[T] 泛型构造
    │   └── health.go                # 健康检查 handler
    │
    ├── logic/                       # 业务编排层
    │   ├── logic.go
    │   └── health.go
    │
    ├── repository/                  # 数据访问层（DB + Cache）
    │   └── health.go
    │
    ├── entity/                      # GORM 实体模型
    │   ├── user.go                  # 用户实体
    │   └── role.go                  # 角色实体
    │
    └── constant/                    # 共享常量
        └── gene_number.go           # Snowflake Gene 编号
```

## 环境变量

### 应用配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `XLF_DEBUG` | `true` | 调试模式（开启 Swagger UI） |
| `XLF_HOST` | `0.0.0.0` | 监听地址 |
| `XLF_PORT` | `8080` | 监听端口 |
| `APP_NAME` | — | 应用名称 |
| `APP_VERSION` | — | 应用版本 |

### PostgreSQL

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DATABASE_HOST` | `localhost` | 数据库主机 |
| `DATABASE_PORT` | `5432` | 数据库端口 |
| `DATABASE_USER` | — | 数据库用户 |
| `DATABASE_PASS` | — | 数据库密码 |
| `DATABASE_NAME` | — | 数据库名称 |
| `DATABASE_PREFIX` | `fp_` | 表前缀 |
| `DATABASE_TIMEZONE` | `Asia/Shanghai` | 时区 |

### Redis

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `NOSQL_HOST` | `localhost` | Redis 主机 |
| `NOSQL_PORT` | `6379` | Redis 端口 |
| `NOSQL_PASS` | — | Redis 密码 |
| `NOSQL_DATABASE` | `1` | 数据库编号 |
| `NOSQL_PREFIX` | `tpl:` | 键前缀 |
| `NOSQL_POOL_SIZE` | `100` | 连接池大小 |

### Snowflake

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SNOWFLAKE_DATACENTER_ID` | `1` | 数据中心 ID |
| `SNOWFLAKE_NODE_ID` | `1` | 节点 ID |

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/health/ping` | 健康检查（含数据库和 Redis 状态） |

## 开发命令

```bash
make help    # 查看所有可用命令
make swag    # 生成 Swagger 文档
make run     # 运行程序
make dev     # 生成文档并运行（推荐）
make tidy    # 清理依赖
```

## 相关项目

| 项目 | 说明 |
|------|------|
| [frontleaves-yggleaf](../frontleaves-yggleaf) | YggLeaf 用户中心（Go/Gin，OAuth2 SSO + Yggdrasil 协议） |
| [yggleaf-frontend](../yggleaf-frontend) | 用户面板前端（React 19 + TanStack Start） |
| [frontleaves-sync](../frontleaves-sync) | 同步工具（Go + Bubble Tea TUI） |
| [plugins/server-status](../plugins/server-status) | 服务器状态监控插件（Java/gRPC） |

## 许可证

私有项目，未公开授权。
