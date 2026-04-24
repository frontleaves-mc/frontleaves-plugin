# 称号系统 + 成就系统 设计文档

> 日期：2026-04-25
> 项目：frontleaves-plugin
> 状态：待审阅

## Context

frontleaves-plugin 是 FrontLeaves MC 服务器生态的插件中枢后端。当前项目仅有健康检查示例代码，需要从零设计并实现"称号系统"和"成就系统"两个核心业务模块。

**问题**：MC 服务器缺乏统一的称号和成就管理能力，无法灵活地为玩家分配称号、追踪成就进度。

**目标**：设计一套完整的称号 + 成就系统，通过 RESTful API 提供 Web 管理端接口，通过 gRPC（后续）向 MC 插件传递数据。

---

## 1. 数据库设计

### 1.1 fp_player（玩家主表）— UUID 主键

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `uuid` | uuid | PK | Minecraft UUID |
| `username` | varchar(64) | NOT NULL | MC 用户名 |
| `group_name` | varchar(64) | NOT NULL | 当前权限组 |
| `reported_at` | timestamptz | NOT NULL | 最后上报时间 |
| `created_at` | timestamptz | autoCreateTime | 创建时间 |
| `updated_at` | timestamptz | autoUpdateTime | 更新时间 |

> 不继承 BaseEntity，UUID 作为主键，不需要 `GetGene()` 方法。权限组信息由 MC 插件通过 gRPC 上报更新。

### 1.2 fp_title（称号定义表）— Gene=33

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | bigint | PK, Snowflake | 主键 |
| `name` | varchar(64) | NOT NULL, UNIQUE | 称号名称 |
| `description` | varchar(255) | NOT NULL | 称号描述 |
| `type` | smallint | NOT NULL | 1=通用, 2=权限组, 3=玩家专属 |
| `permission_group` | varchar(64) | NULL | 关联权限组（type=2 时） |
| `is_active` | boolean | NOT NULL, DEFAULT true | 是否启用 |
| `created_at` | timestamptz | BaseEntity | 创建时间 |
| `updated_at` | timestamptz | BaseEntity | 更新时间 |

**枚举定义**：
- `TitleTypeGeneral TitleType = 1` — 通用称号，所有玩家可自动获得
- `TitleTypeGroup TitleType = 2` — 权限组称号，与 MC 权限组关联
- `TitleTypeExclusive TitleType = 3` — 玩家专属称号，管理员手动分配

### 1.3 fp_player_title（玩家称号关联表）— Gene=34

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | bigint | PK, Snowflake | 主键 |
| `player_uuid` | uuid | NOT NULL, FK→fp_player.uuid | 玩家 UUID |
| `title_id` | bigint | NOT NULL, FK→fp_title.id | 称号 ID |
| `source` | smallint | NOT NULL | 获得来源 |
| `is_equipped` | boolean | NOT NULL, DEFAULT false | 是否当前装备 |
| `granted_at` | timestamptz | NOT NULL | 授予时间 |
| `created_at` | timestamptz | BaseEntity | 创建时间 |
| `updated_at` | timestamptz | BaseEntity | 更新时间 |
| **UNIQUE(player_uuid, title_id)** | | | 防重复 |

**枚举定义**：
- `TitleSourceAuto TitleSource = 1` — 自动获得（通用称号）
- `TitleSourceGroup TitleSource = 2` — 权限组匹配
- `TitleSourceAdmin TitleSource = 3` — 管理员分配
- `TitleSourceAchievement TitleSource = 4` — 成就奖励

### 1.4 fp_achievement（成就定义表）— Gene=35

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | bigint | PK, Snowflake | 主键 |
| `name` | varchar(64) | NOT NULL | 成就名称 |
| `description` | varchar(255) | NOT NULL | 成就描述 |
| `type` | smallint | NOT NULL | 成就类型 |
| `condition_key` | varchar(64) | NOT NULL, UNIQUE | 条件标识 |
| `condition_params` | jsonb | NULL | 条件参数 |
| `reward_config` | jsonb | NULL | 奖励配置 |
| `is_active` | boolean | NOT NULL, DEFAULT true | 是否启用 |
| `sort_order` | int | NOT NULL, DEFAULT 0 | 展示排序 |
| `created_at` | timestamptz | BaseEntity | 创建时间 |
| `updated_at` | timestamptz | BaseEntity | 更新时间 |

**枚举定义**：
- `AchievementTypeStat AchievementType = 1` — 统计类（累计在线时长等）
- `AchievementTypeEvent AchievementType = 2` — 事件类（击杀末影龙等）
- `AchievementTypeSpecial AchievementType = 3` — 特殊条件
- `AchievementTypeManual AchievementType = 4` — 管理员手动

**condition_params 示例**：`{"threshold": 360000, "unit": "seconds"}`
**reward_config 示例**：`{"title_id": 1234567890}` 或 `{}`（无奖励，title_id 为 Snowflake ID 数字类型）

### 1.5 fp_player_achievement（玩家成就表）— Gene=36

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | bigint | PK, Snowflake | 主键 |
| `player_uuid` | uuid | NOT NULL, FK→fp_player.uuid | 玩家 UUID |
| `achievement_id` | bigint | NOT NULL, FK→fp_achievement.id | 成就 ID |
| `status` | smallint | NOT NULL, DEFAULT 0 | 状态 |
| `progress` | bigint | NOT NULL, DEFAULT 0 | 当前进度 |
| `completed_at` | timestamptz | NULL | 完成时间 |
| `created_at` | timestamptz | BaseEntity | 创建时间 |
| `updated_at` | timestamptz | BaseEntity | 更新时间 |
| **UNIQUE(player_uuid, achievement_id)** | | | 防重复 |

**枚举定义**：
- `AchievementStatusInProgress AchievementStatus = 0` — 进行中
- `AchievementStatusCompleted AchievementStatus = 1` — 已完成
- `AchievementStatusClaimed AchievementStatus = 2` — 已领奖

> `progress`：统计类存累计数值，事件类存 0/1（未完成/已完成）。

### 1.6 fp_player_achievement_claim（成就申领记录表）— Gene=38

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | bigint | PK, Snowflake | 主键 |
| `player_uuid` | uuid | NOT NULL, FK→fp_player.uuid | 玩家 UUID |
| `achievement_id` | bigint | NOT NULL, FK→fp_achievement.id | 成就 ID |
| `title_claimed` | boolean | NOT NULL, DEFAULT false | 称号奖励是否已发放 |
| `created_at` | timestamptz | BaseEntity | 创建时间 |
| `updated_at` | timestamptz | BaseEntity | 更新时间 |
| **UNIQUE(player_uuid, achievement_id)** | | | 防重复 |

> 扩展机制：后续新增奖励类型时，只需加 `xxx_claimed bool DEFAULT false` 字段。管理员可手动触发同步，将符合条件的记录改为 true 并执行对应逻辑。

### 1.7 Gene 编号分配

```
GeneTitle              Gene = 33  // 称号定义
GenePlayerTitle        Gene = 34  // 玩家称号关联
GeneAchievement        Gene = 35  // 成就定义
GenePlayerAchievement  Gene = 36  // 玩家成就
// 37 空缺
GeneAchievementClaim   Gene = 38  // 成就申领记录
```

### 1.8 ER 关系

```
fp_player (uuid PK)
  ├── 1:N ── fp_player_title (player_uuid FK)
  ├── 1:N ── fp_player_achievement (player_uuid FK)
  └── 1:N ── fp_player_achievement_claim (player_uuid FK)

fp_title (id PK)
  └── 1:N ── fp_player_title (title_id FK)

fp_achievement (id PK)
  ├── 1:N ── fp_player_achievement (achievement_id FK)
  └── 1:N ── fp_player_achievement_claim (achievement_id FK)
```

---

## 2. RESTful API 设计

### 2.1 称号管理 API（管理端）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/admin/titles` | 创建称号 |
| `PUT` | `/api/v1/admin/titles/:id` | 更新称号 |
| `DELETE` | `/api/v1/admin/titles/:id` | 删除称号 |
| `GET` | `/api/v1/admin/titles` | 称号列表（分页） |
| `GET` | `/api/v1/admin/titles/:id` | 称号详情 |
| `POST` | `/api/v1/admin/titles/:id/assign` | 分配称号给玩家 |
| `DELETE` | `/api/v1/admin/titles/:id/assign` | 撤销玩家称号 |

Swagger Tag：`管理员-称号接口`

### 2.2 玩家称号 API（玩家端）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/players/:uuid/titles` | 获取玩家拥有的称号列表 |
| `PUT` | `/api/v1/players/:uuid/titles/equip` | 装备称号 |
| `DELETE` | `/api/v1/players/:uuid/titles/equip` | 卸下当前称号 |
| `GET` | `/api/v1/players/:uuid/titles/equipped` | 获取当前装备的称号 |

Swagger Tag：`玩家-称号接口`

### 2.3 成就管理 API（管理端）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/admin/achievements` | 创建成就 |
| `PUT` | `/api/v1/admin/achievements/:id` | 更新成就 |
| `DELETE` | `/api/v1/admin/achievements/:id` | 删除成就 |
| `GET` | `/api/v1/admin/achievements` | 成就列表（分页） |
| `GET` | `/api/v1/admin/achievements/:id` | 成就详情 |
| `POST` | `/api/v1/admin/achievements/:id/grant` | 手动授予玩家成就 |

Swagger Tag：`管理员-成就接口`

### 2.4 玩家成就 API（玩家端）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/players/:uuid/achievements` | 玩家成就列表（含进度） |
| `POST` | `/api/v1/players/:uuid/achievements/:achId/claim` | 领取成就奖励 |
| `GET` | `/api/v1/achievements` | 所有成就定义（公开） |

Swagger Tag：`玩家-成就接口` / `成就接口`

### 2.5 玩家信息 API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/players/:uuid` | 获取玩家信息 |
| `GET` | `/api/v1/players` | 玩家列表（管理端，分页） |
| `PUT` | `/api/v1/internal/players/:uuid/group` | 更新玩家权限组（内部接口） |

Swagger Tag：`玩家接口` / `管理员-玩家接口` / `内部接口`

---

## 3. DTO 定义

### 3.1 api/title/

**请求**：
- `CreateTitleRequest`: Name, Description, Type, PermissionGroup
- `UpdateTitleRequest`: Name, Description, Type, PermissionGroup, IsActive
- `AssignTitleRequest`: PlayerUUID
- `EquipTitleRequest`: TitleID

**响应**：
- `TitleResponse`: ID, Name, Description, Type, PermissionGroup, IsActive, CreatedAt
- `PlayerTitleResponse`: TitleResponse 内嵌 + Source, IsEquipped, GrantedAt
- `EquippedTitleResponse`: TitleID, Name, Description, Type

### 3.2 api/achievement/

**请求**：
- `CreateAchievementRequest`: Name, Description, Type, ConditionKey, ConditionParams(json), RewardConfig(json), SortOrder
- `UpdateAchievementRequest`: 同上 + IsActive
- `GrantAchievementRequest`: PlayerUUID

**响应**：
- `AchievementResponse`: ID, Name, Description, Type, ConditionKey, ConditionParams, RewardConfig, SortOrder
- `PlayerAchievementResponse`: AchievementResponse 内嵌 + Status, Progress, CompletedAt
- `AchievementClaimResponse`: AchievementID, TitleClaimed

### 3.3 api/player/

**请求**：
- `UpdatePlayerGroupRequest`: GroupName

**响应**：
- `PlayerResponse`: UUID, Username, GroupName, ReportedAt
- `PlayerListResponse`: 分页列表

---

## 4. 缓存策略

### 4.1 缓存键（前缀 `tpl:`）

| 缓存键 | 类型 | TTL | 说明 |
|--------|------|-----|------|
| `title:all` | Hash | 30min | 所有启用称号 |
| `title:type:{type}` | Set | 30min | 按类型分组的称号 ID |
| `player_title:equipped:{uuid}` | String | 15min | 当前装备称号 ID |
| `player_title:list:{uuid}` | List | 10min | 玩家拥有的称号 ID |
| `player_achievement:progress:{uuid}:{ach_id}` | Hash | 5min | 单个成就进度 |
| `achievement:all` | Hash | 30min | 所有启用成就 |
| `achievement:condition:{key}` | String | 30min | condition_key 索引 |
| `player:{uuid}` | String | 10min | 玩家基本信息 |

### 4.2 更新策略

- **定义数据**（称号/成就）：管理端 CRUD 时主动删除缓存，读取时重建
- **玩家装备状态**：装备/卸载时 Write-through 更新
- **成就进度**：gRPC 上报时累加更新
- **不缓存**：管理端分页列表查询

---

## 5. 核心业务流程

### 5.1 玩家装备称号

```
PUT /api/v1/players/:uuid/titles/equip
→ Handler: 绑定 EquipTitleRequest，校验 UUID
→ Logic:
  1. 校验玩家是否拥有该称号 (fp_player_title)
  2. 校验称号存在且启用 (缓存优先)
  3. 事务: 全部 is_equipped→false, 目标→true
  4. 更新 Redis: player_title:equipped:{uuid}
→ 返回成功
```

### 5.2 权限组称号自动匹配

```
PUT /api/v1/internal/players/:uuid/group
→ Logic:
  1. 更新/创建 fp_player 记录
  2. 查询 fp_title (type=2 AND permission_group 匹配)
  3. 匹配成功且玩家未拥有 → 创建 fp_player_title (source=2)
  4. 旧权限组称号不自动移除（保留历史获得）
  5. 更新缓存
```

### 5.3 成就达成判定（gRPC 预留入口）

```
MC 插件上报事件 {condition_key, player_uuid, value}
→ AchievementLogic.EvaluateEvent:
  1. 查询 condition_key 匹配且 is_active 的成就
  2. 对每个匹配成就:
     a. 获取/创建 fp_player_achievement（status=0）
     b. 根据 type 更新 progress:
        - 统计类: progress += value，判断 >= threshold
        - 事件类: progress = 1，判断首次
        - 特殊条件: 自定义判定
        - 管理员手动: 跳过自动判定，仅通过管理员 API 触发
     c. 若达成且 status=0:
        → 更新 status=1, completed_at=now
        → 若 reward_config 非空: 创建 fp_player_achievement_claim（各字段默认 false）
        → 若 reward_config 为空: 直接更新 status=2（无奖励，自动完成）
```

### 5.4 成就奖励领取

```
POST /api/v1/players/:uuid/achievements/:achId/claim
→ Logic:
  1. 校验 fp_player_achievement.status >= 1
  2. 查询 fp_player_achievement_claim
  3. 检查 reward_config:
     - 有 title_id 且 title_claimed=false:
       a. 创建 fp_player_title (source=4)
       b. title_claimed=true
  4. 幂等处理：已领取不重复发放
  5. status=2（已领奖）
```

---

## 6. 文件组织

### 新增文件（约 28 个）

```
internal/constant/gene_number.go              # 修改: Gene 33-38
internal/entity/player.go                     # Player (UUID 主键)
internal/entity/title.go                      # Title
internal/entity/player_title.go               # PlayerTitle
internal/entity/achievement.go                # Achievement
internal/entity/player_achievement.go         # PlayerAchievement
internal/entity/player_achievement_claim.go   # PlayerAchievementClaim

api/title/request.go                          # 称号请求 DTO
api/title/response.go                         # 称号响应 DTO
api/achievement/request.go                    # 成就请求 DTO
api/achievement/response.go                   # 成就响应 DTO
api/player/request.go                         # 玩家请求 DTO
api/player/response.go                        # 玩家响应 DTO

internal/repository/player_repo.go            # 玩家数据访问
internal/repository/title_repo.go             # 称号数据访问
internal/repository/player_title_repo.go      # 玩家称号数据访问
internal/repository/achievement_repo.go       # 成就数据访问
internal/repository/player_achievement_repo.go # 玩家成就数据访问
internal/repository/player_achievement_claim_repo.go # 申领记录数据访问

internal/logic/player_logic.go                # 玩家业务逻辑
internal/logic/title_logic.go                 # 称号业务逻辑
internal/logic/achievement_logic.go           # 成就业务逻辑

internal/handler/title_admin.go               # 管理端称号 Handler
internal/handler/title_player.go              # 玩家端称号 Handler
internal/handler/achievement_admin.go         # 管理端成就 Handler
internal/handler/achievement_player.go        # 玩家端成就 Handler
internal/handler/player.go                    # 玩家 Handler

internal/app/route/route_title.go             # 称号路由
internal/app/route/route_achievement.go       # 成就路由
internal/app/route/route_player.go            # 玩家路由
```

### 修改文件（2 个）

```
internal/app/startup/startup_database.go      # migrateTables 追加新实体
internal/handler/handler.go                   # 扩展 service 结构体
```

---

## 7. 实现顺序

**阶段 1: 基础设施**
1. Gene 编号定义
2. 6 个 Entity 文件
3. startup_database.go 迁移注册
4. 所有 DTO 文件

**阶段 2: 玩家模块**
1. player_repo → player_logic → player handler → route

**阶段 3: 称号模块**
1. title_repo + player_title_repo
2. title_logic
3. title_admin + title_player handler
4. route_title
5. 种子数据

**阶段 4: 成就模块**
1. achievement_repo + player_achievement_repo + player_achievement_claim_repo
2. achievement_logic
3. achievement_admin + achievement_player handler
4. route_achievement
5. 种子数据

---

## 8. 验证方式

1. `make dev` 启动服务，确认数据库自动迁移成功（6 张新表）
2. 访问 Swagger UI，验证 21 个 API 端点文档生成
3. 通过 Swagger 测试称号 CRUD + 分配 + 装备完整流程
4. 通过 Swagger 测试成就 CRUD + 手动授予 + 领取奖励流程
5. 测试内部接口：更新玩家权限组并验证自动称号匹配
6. 验证缓存策略：Redis 中确认缓存键正确生成和更新
