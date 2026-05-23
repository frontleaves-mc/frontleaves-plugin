# FrontLeaves 成就系统机制文档

> **项目**: `frontleaves-plugin` | **版本基准**: `master` 分支 | **生成时间**: 2026-05-11

---

## 1. 系统概览

成就系统为 Minecraft 服务器玩家提供**目标追踪**与**奖励发放**能力。核心设计遵循**定义 → 进度 → 领奖**三阶段生命周期。

```
┌─────────────────────────────────────────────────────────────────┐
│                       成就系统架构                               │
│                                                                 │
│  Minecraft 插件 (gRPC)          网页端 (REST API)               │
│       │                              │                          │
│       │ EvaluateEvent()              │ CRUD / Claim             │
│       ▼                              ▼                          │
│  ┌──────────────────────────────────────────┐                   │
│  │           AchievementLogic               │                   │
│  │  (业务编排：评估/授予/领奖)              │                   │
│  └──┬──────────┬──────────┬────────────┬────┘                  │
│     │          │          │            │                        │
│  AchievementRepo  GameProfileAchRepo  ClaimRepo  TitleRepo     │
│     │          │          │            │                        │
│  ┌──▼──────────▼──────────▼────────────▼───────────────────┐   │
│  │              PostgreSQL (GORM)                           │   │
│  │  fp_achievements | fp_game_profile_achievements          │   │
│  │  fp_game_profile_achievement_claims | fp_titles          │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.1 设计原则

| 原则 | 说明 |
|------|------|
| **定义与实例分离** | `Achievement` 表存储模板定义，`GameProfileAchievement` 存储玩家实例 |
| **条件驱动** | 通过 `ConditionKey` + `ConditionParams(JSONB)` 实现灵活的条件匹配 |
| **奖励可配置** | `RewardConfig(JSONB)` 支持多种奖励类型，当前已实现「称号」奖励 |
| **幂等安全** | 同一玩家同一成就只能有一条进度记录（复合唯一索引保证） |
| **分层严格** | Handler → Logic → Repository，不跨层调用 |

---

## 2. 数据模型

### 2.1 ER 关系图

```
GameProfile (1) ──< (N) GameProfileAchievement >── (1) Achievement
GameProfile (1) ──< (N) GameProfileAchievementClaim >── (1) Achievement
GameProfile (1) ──< (N) GameProfileTitle >── (1) Title
                                              │
                          RewardConfig.title_id (JSONB 引用)
```

### 2.2 表结构详细设计

#### 表 `fp_achievements` — 成就定义（核心模板）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | `bigint` (Snowflake) | PK, Gene=35 | 雪花 ID |
| `name` | `varchar(64)` | NOT NULL | 成就名称 |
| `description` | `varchar(255)` | NOT NULL | 成就描述 |
| `type` | `smallint` | NOT NULL, INDEX `idx_ach_type` | 成就类型 (1~4) |
| `condition_key` | `varchar(64)` | NOT NULL, UNIQUE `uk_ach_condition_key` | 条件标识符（唯一） |
| `condition_params` | `jsonb` | — | 条件参数，如 `{"threshold": 1000}` |
| `reward_config` | `jsonb` | — | 奖励配置，如 `{"title_id": "123456789"}` |
| `is_active` | `boolean` | NOT NULL, DEFAULT `true` | 是否启用 |
| `sort_order` | `int` | NOT NULL, DEFAULT `0` | 展示排序（ASC） |
| `created_at` | `timestamptz` | — | 创建时间 |
| `updated_at` | `timestamptz` | — | 更新时间 |
| `deleted_at` | `timestamptz` | INDEX | 软删除时间 |

**`type` 枚举 (`AchievementType`):**

| 值 | 常量 | 含义 | 触发方式 |
|----|------|------|----------|
| `1` | `AchievementTypeStat` | **统计类** | 累加进度，到达阈值完成 |
| `2` | `AchievementTypeEvent` | **事件类** | 触发即完成 |
| `3` | `AchievementTypeSpecial` | **特殊条件** | 传入值 ≥ 阈值时完成 |
| `4` | `AchievementTypeManual` | **管理员手动** | 仅通过管理端 API 授予 |

---

#### 表 `fp_game_profile_achievements` — 玩家成就进度

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | `bigint` (Snowflake) | PK, Gene=36 | 雪花 ID |
| `game_profile_uuid` | `uuid` | NOT NULL, UNIQUE(`uk_gpa_gameprofile_ach`) | 玩家 UUID |
| `achievement_id` | `bigint` | NOT NULL, UNIQUE(`uk_gpa_gameprofile_ach`), INDEX `idx_gpa_ach` | 成就 ID |
| `status` | `smallint` | NOT NULL, DEFAULT `0` | 状态 (0/1/2) |
| `progress` | `bigint` | NOT NULL, DEFAULT `0` | 当前进度值 |
| `completed_at` | `timestamptz` | — | 完成时间（可为 NULL） |
| `created_at` | `timestamptz` | — | 创建时间 |
| `updated_at` | `timestamptz` | — | 更新时间 |
| `deleted_at` | `timestamptz` | INDEX | 软删除时间 |

**复合唯一索引 `uk_gpa_gameprofile_ach(game_profile_uuid, achievement_id)`**：保证同一玩家对同一成就只有一条记录。

**FK**: `achievement_id → fp_achievements.id` (CASCADE), `game_profile_uuid → game_profiles.uuid` (CASCADE)

**`status` 枚举 (`AchievementStatus`):**

| 值 | 常量 | 含义 |
|----|------|------|
| `0` | `AchievementStatusInProgress` | **进行中** — 正在追踪进度 |
| `1` | `AchievementStatusCompleted` | **已完成** — 条件达成，等待领奖 |
| `2` | `AchievementStatusClaimed` | **已领奖** — 奖励已领取（终态） |

---

#### 表 `fp_game_profile_achievement_claims` — 领奖记录

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | `bigint` (Snowflake) | PK, Gene=38 | 雪花 ID |
| `game_profile_uuid` | `uuid` | NOT NULL, UNIQUE(`uk_gpac_gameprofile_ach`) | 玩家 UUID |
| `achievement_id` | `bigint` | NOT NULL, UNIQUE(`uk_gpac_gameprofile_ach`) | 成就 ID |
| `title_claimed` | `boolean` | NOT NULL, DEFAULT `false` | 称号奖励是否已发放 |
| `created_at` | `timestamptz` | — | 创建时间 |
| `updated_at` | `timestamptz` | — | 更新时间 |
| `deleted_at` | `timestamptz` | INDEX | 软删除时间 |

> 此表**仅在有奖励配置时**创建记录。无奖励的成就完成时直接跳到 `Claimed` 状态，不创建 Claim 记录。

---

#### 关联表 `fp_game_profile_titles` — 玩家称号

成就奖励的称号通过此表关联，`source = 4` (TitleSourceAchievement) 标识来源为成就系统。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | `bigint` (Snowflake) | PK, Gene=34 | 雪花 ID |
| `game_profile_uuid` | `uuid` | NOT NULL, UNIQUE(`uk_gpt_gameprofile_title`) | 玩家 UUID |
| `title_id` | `bigint` | NOT NULL, UNIQUE(`uk_gpt_gameprofile_title`) | 称号 ID |
| `source` | `smallint` | NOT NULL | 获得来源 (1~4) |
| `granted_at` | `timestamptz` | NOT NULL | 授予时间 |
| `is_equipped` | `boolean` | NOT NULL, DEFAULT `false` | 是否装备 |

**`source` 枚举 (`TitleSource`):**

| 值 | 常量 | 含义 |
|----|------|------|
| `1` | `TitleSourceAuto` | 自动获得 |
| `2` | `TitleSourceGroup` | 权限组匹配 |
| `3` | `TitleSourceAdmin` | 管理员分配 |
| `4` | `TitleSourceAchievement` | **成就奖励** |

---

### 2.3 Snowflake Gene 编号分配

| Gene | 常量 | 实体 |
|------|------|------|
| `33` | `GeneTitle` | Title |
| `34` | `GenePlayerTitle` | GameProfileTitle |
| `35` | `GeneAchievement` | Achievement |
| `36` | `GenePlayerAchievement` | GameProfileAchievement |
| `38` | `GeneAchievementClaim` | GameProfileAchievementClaim |

> Gene 37 保留未使用，可供成就子系统未来扩展。

---

## 3. 生命周期与状态流转

```
                    玩家触发条件              条件满足
  [不存在] ──────────► [进行中(0)] ──────────► [已完成(1)] ──────────► [已领奖(2)]
                          │                       │
                          │  (管理员手动授予)       │  (无奖励配置时)
                          │  直接跳到 已完成        │  直接跳到 已领奖
                          ▼                       ▼
                     [已完成(1)]              [已领奖(2)]
```

### 3.1 三阶段详解

**阶段一：进度追踪** (`InProgress`)

- Minecraft 插件通过 gRPC 调用 `EvaluateEvent(conditionKey, playerUUID, value)`
- 系统查找所有 `condition_key` 匹配且 `is_active = true` 的成就
- 按类型处理：
  - **Stat(统计类)**: `progress += value`，与 `threshold` 比较
  - **Event(事件类)**: 直接触发完成
  - **Special(特殊条件)**: `value >= threshold` 时完成
  - **Manual(手动类)**: 在 `EvaluateEvent` 中被跳过

**阶段二：条件达成** (`Completed`)

- 进度达到阈值或事件触发后，状态变为 `Completed(1)`
- 设置 `completed_at` 时间戳
- 如果 `reward_config` 非空 → 创建 `GameProfileAchievementClaim` 记录等待领奖
- 如果 `reward_config` 为空 → 直接跳到 `Claimed(2)`（无奖励，无需领取）

**阶段三：领奖** (`Claimed`)

- 玩家通过 API `POST /game-profiles/:uuid/achievements/:achId/claim` 领取
- 系统解析 `reward_config`，当前支持 `{"title_id": "xxx"}` 格式
- 如果包含 `title_id`：
  1. 检查玩家是否已有该称号（防重复）
  2. 创建 `GameProfileTitle` 记录，`source = 4` (TitleSourceAchievement)
  3. 标记 `title_claimed = true`
- 状态变更为 `Claimed(2)`（终态）

---

## 4. 条件匹配机制

### 4.1 ConditionKey 设计

`ConditionKey` 是成就系统最核心的设计概念。它是一个**全局唯一**的字符串标识符，用于将「游戏事件」映射到「成就定义」。

```
┌────────────────────┐        ConditionKey         ┌──────────────────────┐
│  Minecraft 插件     │  ──── "mine_stone" ──────►  │  Achievement         │
│  玩家挖了一块石头    │                              │  name: "采石大师"     │
│                    │                              │  threshold: 1000     │
└────────────────────┘                              └──────────────────────┘
```

**规则**：
- 一个 `ConditionKey` 可以对应**多个成就**（不同阈值的多级成就）
- 一个成就只能有**一个** `ConditionKey`（唯一索引保证）

### 4.2 ConditionParams 结构

```json
{
  "threshold": 1000
}
```

当前仅使用 `threshold` 字段，用于 Stat 和 Special 类型的阈值判断。Logic 层通过 `getThreshold()` 辅助方法解析。

### 4.3 各类型匹配逻辑

| 类型 | 匹配公式 | 说明 |
|------|----------|------|
| **Stat(1)** | `newProgress = oldProgress + value`; 完成: `newProgress >= threshold` | 累加式，适合「挖 N 个方块」「击杀 N 个怪物」 |
| **Event(2)** | 触发即完成 | 一次性，适合「首次登录」「首次进入末地」 |
| **Special(3)** | 完成: `value >= threshold` | 值比较式，适合「达到某等级」「血量低于 X」 |
| **Manual(4)** | 不参与自动匹配 | 仅管理端 `GrantAchievement` 手动授予 |

### 4.4 匹配流程伪代码

```go
// EvaluateEvent(conditionKey, playerUUID, value)
achievements ← 查找所有 condition_key == conditionKey AND is_active == true

FOR EACH achievement IN achievements:
    IF achievement.Type == Manual: CONTINUE  // 手动类跳过

    pa ← 查找玩家该成就进度
    IF pa 已完成/已领奖: CONTINUE  // 已完成的跳过
    IF pa 不存在: 创建新进度记录(status=InProgress)

    completed ← false
    SWITCH achievement.Type:
        CASE Stat:
            pa.Progress += value
            IF pa.Progress >= threshold: completed = true
        CASE Event:
            completed = true  // 直接触发
        CASE Special:
            IF value >= threshold: completed = true

    IF completed:
        更新状态为 Completed
        IF 有奖励配置:
            创建 Claim 记录  // 等待玩家领取
        ELSE:
            直接标记为 Claimed  // 无奖励直接终态
```

---

## 5. 奖励机制

### 5.1 RewardConfig 结构

当前已实现的奖励类型只有**称号**：

```json
{
  "title_id": "1234567890123456789"
}
```

### 5.2 奖励发放流程

```
玩家点击「领取」
      │
      ▼
检查成就状态 ≥ Completed？
      │ 是
      ▼
存在 Claim 记录？
      │
      ├─ 否 → 直接标记为 Claimed（无奖励配置的情况）
      │
      └─ 是 → 解析 RewardConfig
              │
              ├─ 有 title_id → 检查玩家是否已有
              │     │
              │     ├─ 没有 → 创建 GameProfileTitle(source=Achievement)
              │     └─ 已有 → 跳过（幂等）
              │     │
              │     └─ 标记 title_claimed = true
              │
              └─ 标记状态为 Claimed
```

### 5.3 奖励类型扩展性

`RewardConfig` 使用 JSONB 存储，天然支持扩展。未来可添加的奖励类型示例：

```json
// 多奖励组合
{
  "title_id": "123456789",
  "coins": 500,
  "items": [{"id": "diamond", "count": 1}]
}
```

---

## 6. API 接口总览

### 6.1 管理端接口 (需 LoginAuth + Admin 中间件)

| 方法 | 路径 | 说明 | 请求体 |
|------|------|------|--------|
| `POST` | `/api/v1/admin/achievements` | 创建成就 | `CreateAchievementRequest` |
| `PUT` | `/api/v1/admin/achievements/:id` | 更新成就 | `UpdateAchievementRequest` |
| `DELETE` | `/api/v1/admin/achievements/:id` | 删除成就 | — |
| `GET` | `/api/v1/admin/achievements` | 分页列表 | Query: `page`, `page_size`, `type` |
| `GET` | `/api/v1/admin/achievements/:id` | 成就详情 | — |
| `POST` | `/api/v1/admin/achievements/:id/grant` | 手动授予 | `GrantAchievementRequest` |

### 6.2 公开接口

| 方法 | 路径 | 说明 | 中间件 |
|------|------|------|--------|
| `GET` | `/api/v1/achievements` | 公开成就列表（所有启用的） | 无 |

### 6.3 玩家接口 (需 LoginAuth + Player 中间件)

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/game-profiles/:uuid/achievements` | 玩家成就列表 |
| `POST` | `/api/v1/game-profiles/:uuid/achievements/:achId/claim` | 领取奖励 |

### 6.4 gRPC 内部接口（非 HTTP）

| 方法 | 说明 |
|------|------|
| `EvaluateEvent(conditionKey, playerUUID, value)` | MC 插件上报事件，触发成就评估 |

---

## 7. DTO 结构定义

### 7.1 请求 DTO

**CreateAchievementRequest:**

```json
{
  "name": "采石大师",
  "description": "累计挖掘 1000 个石头",
  "type": 1,
  "condition_key": "mine_stone",
  "condition_params": {"threshold": 1000},
  "reward_config": {"title_id": "123456789"},
  "sort_order": 10
}
```

**UpdateAchievementRequest:**

```json
{
  "name": "采石大师",
  "description": "累计挖掘 1000 个石头",
  "type": 1,
  "condition_key": "mine_stone",
  "condition_params": {"threshold": 1000},
  "reward_config": {"title_id": "123456789"},
  "sort_order": 10,
  "is_active": true
}
```

**GrantAchievementRequest:**

```json
{
  "player_uuid": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 7.2 响应 DTO

**AchievementResponse:**

```json
{
  "id": "1234567890123456789",
  "name": "采石大师",
  "description": "累计挖掘 1000 个石头",
  "type": 1,
  "condition_key": "mine_stone",
  "condition_params": {"threshold": 1000},
  "reward_config": {"title_id": "123456789"},
  "sort_order": 10,
  "is_active": true
}
```

**PlayerAchievementResponse** (嵌套 AchievementResponse):

```json
{
  "id": "1234567890123456789",
  "name": "采石大师",
  "description": "累计挖掘 1000 个石头",
  "type": 1,
  "condition_key": "mine_stone",
  "condition_params": {"threshold": 1000},
  "reward_config": {"title_id": "123456789"},
  "sort_order": 10,
  "is_active": true,
  "status": 1,
  "progress": 1200,
  "completed_at": "2026-05-11T12:00:00Z"
}
```

**AchievementClaimResponse:**

```json
{
  "achievement_id": "1234567890123456789",
  "title_claimed": true
}
```

**AchievementListResponse:**

```json
{
  "list": ["...AchievementResponse 数组"],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

---

## 8. Repository 方法清单

### AchievementRepo

| 方法 | 签名 | 说明 |
|------|------|------|
| `Create` | `(ctx, *Achievement) → *xError.Error` | 创建成就 |
| `Update` | `(ctx, *Achievement) → *xError.Error` | 更新成就 (Save) |
| `Delete` | `(ctx, SnowflakeID) → *xError.Error` | 软删除 |
| `GetByID` | `(ctx, SnowflakeID) → (*Achievement, *xError.Error)` | 按 ID 查询 |
| `List` | `(ctx, page, pageSize, *int16) → ([]Achievement, int64, *xError.Error)` | 分页列表，可按 type 过滤，排序 `sort_order ASC, created_at DESC` |
| `GetActiveByConditionKey` | `(ctx, string) → ([]Achievement, *xError.Error)` | **核心查询** — 按条件标识查活跃成就 (EvaluateEvent 使用) |
| `ListActive` | `(ctx) → ([]Achievement, *xError.Error)` | 所有活跃成就列表 |

### GameProfileAchievementRepo

| 方法 | 签名 | 说明 |
|------|------|------|
| `Create` | `(ctx, *GPA) → *xError.Error` | 创建玩家进度 |
| `GetByGameProfileAndAchievement` | `(ctx, UUID, SnowflakeID) → (*GPA, *xError.Error)` | 查询玩家某成就进度 (Preload Achievement) |
| `ListByGameProfile` | `(ctx, UUID) → ([]GPA, *xError.Error)` | 玩家全部成就 (Preload Achievement) |
| `UpdateProgress` | `(ctx, SnowflakeID, int64) → *xError.Error` | 更新进度值 |
| `UpdateStatus` | `(ctx, SnowflakeID, Status) → *xError.Error` | 更新状态，Completed 时自动设 `completed_at = NOW()` |

### GameProfileAchievementClaimRepo

| 方法 | 签名 | 说明 |
|------|------|------|
| `Create` | `(ctx, *Claim) → *xError.Error` | 创建领奖记录 |
| `GetByGameProfileAndAchievement` | `(ctx, UUID, SnowflakeID) → (*Claim, *xError.Error)` | 查询领奖记录 |
| `UpdateTitleClaimed` | `(ctx, SnowflakeID, bool) → *xError.Error` | 标记称号已发放 |

---

## 9. 代码组织映射

```
internal/
├── entity/
│   ├── achievement.go                          # Achievement 实体 + AchievementType 枚举
│   ├── game_profile_achievement.go             # 玩家成就进度实体 + AchievementStatus 枚举
│   └── game_profile_achievement_claim.go       # 领奖记录实体
├── constant/
│   └── gene_number.go                          # Gene 35/36/38 编号
├── repository/
│   ├── achievement_repo.go                     # 7 方法: Create/Update/Delete/GetByID/List/GetActiveByConditionKey/ListActive
│   ├── game_profile_achievement_repo.go        # 5 方法: Create/GetByGPA/ListByGP/UpdateProgress/UpdateStatus
│   └── game_profile_achievement_claim_repo.go  # 3 方法: Create/GetByGPA/UpdateTitleClaimed
├── logic/
│   └── achievement_logic.go                    # 9 公开 + 2 私有方法（核心业务编排）
├── handler/
│   ├── achievement_admin.go                    # 管理端 6 个端点
│   └── achievement_player.go                   # 玩家端 3 个端点
└── app/route/
    └── route_achievement.go                    # 路由注册 + 中间件绑定

api/achievement/
├── request.go                                  # DTO: Create/Update/Grant 请求
└── response.go                                 # DTO: Achievement/Player/Claim/List 响应
```

---

## 10. 创建成就的完整示例

### 示例 1：统计类成就 — 「矿洞探险家」

```bash
POST /api/v1/admin/achievements
{
  "name": "矿洞探险家",
  "description": "累计挖掘 500 个石头",
  "type": 1,
  "condition_key": "mine_stone",
  "condition_params": {"threshold": 500},
  "reward_config": {"title_id": "1234567890"},
  "sort_order": 1
}
```

**触发方式**: MC 插件每次玩家挖石头时调用 `EvaluateEvent("mine_stone", playerUUID, 1)`

### 示例 2：事件类成就 — 「初入末地」

```bash
POST /api/v1/admin/achievements
{
  "name": "初入末地",
  "description": "首次进入末地维度",
  "type": 2,
  "condition_key": "enter_the_end",
  "condition_params": {},
  "reward_config": {},
  "sort_order": 2
}
```

**触发方式**: MC 插件在玩家进入末地时调用 `EvaluateEvent("enter_the_end", playerUUID, 1)`

### 示例 3：特殊条件成就 — 「战斗力突破」

```bash
POST /api/v1/admin/achievements
{
  "name": "战斗力突破",
  "description": "战斗力达到 10000",
  "type": 3,
  "condition_key": "combat_power",
  "condition_params": {"threshold": 10000},
  "reward_config": {"title_id": "9876543210"},
  "sort_order": 3
}
```

**触发方式**: MC 插件在战斗力变化时调用 `EvaluateEvent("combat_power", playerUUID, currentValue)`

### 示例 4：手动授予成就 — 「杰出贡献者」

```bash
POST /api/v1/admin/achievements
{
  "name": "杰出贡献者",
  "description": "对服务器做出杰出贡献的玩家",
  "type": 4,
  "condition_key": "manual_outstanding_contributor",
  "condition_params": {},
  "reward_config": {"title_id": "1111111111"},
  "sort_order": 99
}
```

**授予方式**: 管理员通过 `POST /api/v1/admin/achievements/:id/grant` 手动授予

```bash
POST /api/v1/admin/achievements/:id/grant
{
  "player_uuid": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

## 11. 当前状态与未来扩展

### 11.1 已实现

- [x] 成就定义 CRUD（管理端 API）
- [x] 成就进度追踪（EvaluateEvent 事件评估）
- [x] 手动授予（管理员 API）
- [x] 奖励领取（玩家 API）
- [x] 称号奖励发放
- [x] 公开成就列表
- [x] 玩家成就列表
- [x] 分页查询 + 类型筛选
- [x] 幂等防重复

### 11.2 待实现 / 可扩展

| 方向 | 说明 |
|------|------|
| **gRPC 接口暴露** | `EvaluateEvent` 方法已写好，需 proto 定义 + gRPC server 注册 |
| **奖励类型扩展** | `RewardConfig` 已用 JSONB，可扩展 `coins`、`items` 等奖励类型 |
| **通知机制** | 成就完成时通知玩家（WebSocket / 游戏内消息） |
| **成就分类** | 增加 `category` 字段，支持成就树/成就分组展示 |
| **隐藏成就** | 增加 `is_hidden` 字段，满足条件前不公开显示 |
| **多级成就** | 同一 `ConditionKey` 不同阈值（铜/银/金），当前架构已支持 |
| **批量事件评估** | 当前单条处理，可增加批量接口提高吞吐 |
| **缓存优化** | Redis 缓存活跃成就列表、玩家进度等热数据 |
| **统计面板** | 成就达成率、热门成就等运营数据 |
| **前置依赖** | 完成 A 成就才能触发 B 成就的依赖链 |
