# 用户体系与 GameProfile 重命名设计

## 背景

`frontleaves-plugin` 当前缺少 User 表，且 Player 实体命名与上游 Yggleaf 认证中心的 GameProfile 不一致。需要：
1. 新增 User 本地表
2. Player 重命名为 GameProfile，统一 Yggleaf 生态命名
3. Yggleaf 新增 GetUserInfo gRPC，Plugin 通过 gRPC 同步用户数据

## 架构关系

```
Yggleaf (认证中心)                    frontleaves-plugin (插件中枢)
┌─────────────┐                     ┌──────────────┐
│ User        │◄──── 1:N ────┐      │ User         │
│ GameProfile │◄──── 所属 ───┘      │ GameProfile  │
│ AuthService │                     │ Title        │
│ └GetUserInfo│──gRPC 同步──►       │ Achievement  │
└─────────────┘                     └──────────────┘
```

Plugin 的 User + GameProfile 为 Yggleaf 的**本地子集副本**，通过 gRPC 同步保持数据一致。

## 一、数据模型

### 1.1 User 实体（新增）

```go
// entity/user.go
type User struct {
    xModels.BaseEntity
    Username string `gorm:"not null;type:varchar(64);comment:用户名" json:"username"`
    GameProfiles []GameProfile `gorm:"foreignKey:UserID" json:"-"`
}

func (_ *User) GetGene() xSnowflake.Gene {
    return xSnowflake.GeneUser // = 31
}
```

### 1.2 GameProfile 实体（重命名自 Player）

```go
// entity/game_profile.go（原 player.go）
type GameProfile struct {
    UserID     SnowflakeID `gorm:"not null;comment:所属用户ID" json:"user_id"`
    UUID       uuid.UUID   `gorm:"type:uuid;primaryKey;comment:玩家UUID" json:"uuid"`
    Username   string      `gorm:"not null;type:varchar(64);comment:MC用户名" json:"username"`
    GroupName  string      `gorm:"not null;type:varchar(64);comment:当前权限组" json:"group_name"`
    ReportedAt time.Time   `gorm:"not null;type:timestamptz;comment:最后上报时间" json:"reported_at"`
    CreatedAt  time.Time   `gorm:"not null;type:timestamptz;autoCreateTime:milli" json:"-"`
    UpdatedAt  time.Time   `gorm:"not null;type:timestamptz;autoUpdateTime:milli" json:"-"`
}
```

### 1.3 关联表重命名

| 原名 | 新名 | 字段变更 |
|------|------|----------|
| `PlayerTitle` | `GameProfileTitle` | `PlayerUUID` → `GameProfileUUID` |
| `PlayerAchievement` | `GameProfileAchievement` | `PlayerUUID` → `GameProfileUUID` |
| `PlayerAchievementClaim` | `GameProfileAchievementClaim` | `PlayerUUID` → `GameProfileUUID` |

### 1.4 Gene 编号

```go
GeneUser = 31  // 新增，占用 32 之前的空位
// GeneTitle = 33
// GenePlayerTitle → GeneGameProfileTitle = 34
// GeneAchievement = 35
// GenePlayerAchievement → GeneGameProfileAchievement = 36
// GeneAchievementClaim → GeneGameProfileAchievementClaim = 38
```

### 1.5 迁移策略

- `startup_database.go` 中 `migrateTables` 顺序：User → GameProfile → Title → Achievement → 关联表
- GORM AutoMigrate 自动处理列变更（新增 UserID、字段重命名）
- 表名 `fp_player` → `fp_game_profile` 需幂等迁移脚本：先创建新表、复制数据、删除旧表
- 迁移脚本放在 `app/startup/prepare/` 下，必须是幂等的

## 二、gRPC 接口设计

### 2.1 Yggleaf 侧 - 新增 GetUserInfo RPC

在 `proto/auth/auth.proto` 的 `AuthService` 中新增：

```protobuf
rpc GetUserInfo(GetUserInfoRequest) returns (GetUserInfoResponse);

message GetUserInfoRequest {
    string user_id = 1;
}

message GetUserInfoResponse {
    string user_id   = 1;
    string username  = 2;
    string role_name = 3;
    bool   has_ban   = 4;
    repeated GameProfileInfo profiles = 5;
}

message GameProfileInfo {
    string uuid       = 1;
    string name       = 2;
    string skin_url   = 3;
    string cape_url   = 4;
    string skin_model = 5;
}
```

### 2.2 Plugin 侧 - 扩展 gRPC Client

`internal/app/grpc/client.go` 新增方法：

```go
func (c *AuthClient) GetUserInfo(ctx context.Context, userID string) (*authpb.GetUserInfoResponse, error)
```

携带 metadata：`app-access-id`、`app-secret-key`（与现有 ValidateToken 一致）。

### 2.3 认证鉴权

复用 Yggleaf 的 `UnaryAppVerify` 拦截器，调用方必须是已注册内部服务。

## 三、代码层变更

### 3.1 文件清单

**新增（4 个）：**
- `entity/user.go`
- `repository/user_repo.go`
- `repository/cache/user_cache.go`（可选，先搭框架）
- `logic/user_logic.go`

**重命名（10 个）：**
- `entity/player.go` → `entity/game_profile.go`
- `entity/player_title.go` → `entity/game_profile_title.go`
- `entity/player_achievement.go` → `entity/game_profile_achievement.go`
- `entity/player_achievement_claim.go` → `entity/game_profile_achievement_claim.go`
- `repository/player_repo.go` → `repository/game_profile_repo.go`
- `repository/player_title_repo.go` → `repository/game_profile_title_repo.go`
- `repository/player_achievement_repo.go` → `repository/game_profile_achievement_repo.go`
- `repository/player_achievement_claim_repo.go` → `repository/game_profile_achievement_claim_repo.go`
- `logic/player_logic.go` → `logic/game_profile_logic.go`
- `handler/player_handler.go` → `handler/game_profile_handler.go`

**修改（8+ 个）：**
- `constant/gene_number.go` — 新增 GeneUser = 31
- `route/route_player.go` → `route/route_game_profile.go` — 路径 `/players/` → `/game-profiles/`
- `route/route_title.go` — handler 引用更新
- `route/route_achievement.go` — handler 引用更新
- `route/route.go` — 子路由注册函数名更新
- `app/startup/startup_database.go` — migrateTables 更新
- `app/startup/prepare/` — 新增幂等迁移脚本
- `app/grpc/client.go` — 新增 GetUserInfo 方法

### 3.2 API 路径变更

| 原路径 | 新路径 |
|--------|--------|
| `/api/v1/players/:uuid` | `/api/v1/game-profiles/:uuid` |
| `/api/v1/players` | `/api/v1/game-profiles` |
| `/api/v1/internal/players/:uuid/group` | `/api/v1/internal/game-profiles/:uuid/group` |
| `/api/v1/players/:uuid/titles` | `/api/v1/game-profiles/:uuid/titles` |
| `/api/v1/players/:uuid/titles/equip` | `/api/v1/game-profiles/:uuid/titles/equip` |
| `/api/v1/players/:uuid/titles/equipped` | `/api/v1/game-profiles/:uuid/titles/equipped` |
| `/api/v1/players/:uuid/achievements` | `/api/v1/game-profiles/:uuid/achievements` |
| `/api/v1/players/:uuid/achievements/:ach_id/claim` | `/api/v1/game-profiles/:uuid/achievements/:ach_id/claim` |

### 3.3 数据同步流程

```
请求 → LoginAuth 中间件
  ├── ValidateToken (gRPC, 现有) → 写入 AuthUserInfo 缓存
  └── GetUserInfo (gRPC, 新增)
        ├── UserRepo.Upsert(user)          // 同步 User
        └── GameProfileRepo.UpsertEach()   // 同步 GameProfiles
```

GetUserInfo 失败时**不阻断请求**，降级使用现有缓存数据。

## 四、Verification

1. `make dev` 编译通过，Swagger 生成无报错
2. 数据库自动迁移无报错，`fp_user` 和 `fp_game_profile` 表正确创建
3. 现有 API 路径返回 404（原 `/players/` 路径移除），新路径 `/game-profiles/` 正常响应
4. gRPC GetUserInfo 调用返回正确数据（需 Yggleaf 侧同步实现）
5. 中间件 LoginAuth 中 GetUserInfo 同步不阻断正常请求流程
