# GrimAC 源码分析报告

> **分析版本:** Commit `ec08cbee1d8037988153781fb42a7432c3da188f`
> **分析日期:** 2026-05-27
> **源码仓库:** https://github.com/GrimAnticheat/Grim
> **许可证:** GPL-3.0

**合规声明:** 本报告基于对 GrimAC 源码的阅读和分析撰写。所有算法描述均使用伪代码或自然语言表述，不包含源码原文复制。报告中出现的代码片段均为独立撰写的伪代码，非 GPL-3.0 许可源码的摘录。本报告的目的在于理解反作弊检测原理，为 FrontLeaves 项目提供技术参考，不涉及 GrimAC 源码的分发。

---

## 目录

1. [概述](#1-概述)
2. [项目架构地图](#2-项目架构地图)
3. [预测引擎深度分析](#3-预测引擎深度分析)
4. [移动检测实现分析](#4-移动检测实现分析)
5. [战斗检测实现分析](#5-战斗检测实现分析)
6. [与 FrontLeaves 的对比与启示](#6-与-frontleaves-的对比与启示)
7. [GPL-3.0 合规声明](#7-gpl-30-合规声明)

---

## 1. 概述

### 1.1 分析范围

GrimAC 是一个基于预测引擎的 Minecraft 反作弊插件，运行在服务端 JVM 内部。本次分析聚焦以下三个核心领域：

- **预测引擎**：暴力枚举玩家输入向量，模拟 MC 客户端移动物理，比对预测与实际位置
- **移动检测**：基于预测偏差（offset）的 Speed/Fly/NoFall/Phase 等检测体系
- **战斗检测**：Raycast 射线-AABB 碰撞的 Reach 检测、Timer 事务时钟法、KillAura 多目标分析

### 1.2 源码规模

| 统计项 | 数值 |
|--------|------|
| Java 文件数 | 495 |
| 代码总行数 | ~50,064 |
| 核心模块数 | 7（predictionengine, checks, utils, manager, events, command, player） |
| 检测子目录 | 18（aim, badpackets, breaking, combat, elytra 等） |

### 1.3 分析方法

所有分析基于源码阅读，产出物为算法原理描述（伪代码）和架构分析。不包含 GrimAC 源码的复制粘贴。

---

## 2. 项目架构地图

### 2.1 核心模块统计

| 模块 | 路径 | Java 文件数 | 代码行数 | 说明 |
|------|------|------------|---------|------|
| predictionengine | `ac/grim/grimac/predictionengine` | 39 | 6,236 | 预测引擎核心：物理模拟、移动预测、方块碰撞 |
| checks | `ac/grim/grimac/checks` | 148 | 8,962 | 各类反作弊检测模块 |
| utils | `ac/grim/grimac/utils` | 193 | 20,247 | 工具类集合：碰撞、数学、NMS 反射等 |
| manager | `ac/grim/grimac/manager` | 62 | 6,488 | 检测管理器、配置、处罚系统 |
| events | `ac/grim/grimac/events` | 26 | 4,184 | 事件处理：包事件、玩家事件 |
| command | `ac/grim/grimac/command` | 26 | 2,863 | 命令系统 |
| player | `ac/grim/grimac/player` | 1 | 1,084 | 玩家数据容器（GrimPlayer 类） |

### 2.2 核心类职责

| 类名 | 行数 | 职责 |
|------|------|------|
| PredictionEngine | 942 | 预测引擎主类，输入向量转换、边界处理 |
| MovementCheckRunner | 683 | 移动检测运行器，预测执行入口 |
| PlayerBaseTick | 575 | 玩家基础 Tick 逻辑（位置更新、属性应用） |
| MovementTicker | 517 | 移动 Tick 核心（重力、速度、碰撞） |
| PointThreeEstimator | 486 | 0.03 偏移估算器（浮点精度问题） |
| UncertaintyHandler | 380 | 不确定性处理器（网络延迟补偿） |
| Reach | 371 | 攻击距离检测（Raycast 射线-AABB 碰撞） |
| CheckManager | 502 | 检测管理器（注册所有 Check，分发事件） |
| SetbackTeleportUtil | 464 | Setback 传送工具（回滚玩家位置） |
| GrimPlayer | 1,084 | 玩家数据容器（存储所有预测状态、检测状态） |

### 2.3 目录结构

```
common/src/main/java/ac/grim/grimac/
├── checks/                    # 检测模块（148 文件）
│   ├── impl/
│   │   ├── aim/               # 瞄准检测
│   │   ├── badpackets/        # 恶意包检测（18+）
│   │   ├── breaking/          # 方块破坏检测
│   │   ├── combat/            # 战斗检测（Reach, MultiInteract）
│   │   ├── elytra/            # 鞘翅检测（6 检测）
│   │   ├── flight/            # 飞行检测
│   │   ├── groundspoof/       # 地面欺骗（NoFall）
│   │   ├── movement/          # 移动检测（NoSlow, PredictionRunner）
│   │   ├── prediction/        # 预测相关（OffsetHandler, Phase）
│   │   ├── scaffolding/       # 脚手架检测
│   │   ├── sprint/            # 疾跑检测
│   │   ├── timer/             # 计时器检测
│   │   ├── velocity/          # 击退检测
│   │   └── ...                # 其他检测子目录
│   └── type/                  # 检测接口定义
├── predictionengine/          # 预测引擎（39 文件）
│   ├── predictions/           # 预测实现（Normal, Water, Elytra）
│   ├── movementtick/          # 移动 Tick
│   └── blockeffects/          # 方块效果
├── utils/                     # 工具类（193 文件）
│   ├── collisions/            # 碰撞检测
│   ├── latency/               # 延迟处理
│   ├── math/                  # 数学工具
│   └── nmsutil/               # NMS 反射工具
├── manager/                   # 管理器（62 文件）
├── events/                    # 事件处理
├── command/                   # 命令系统
└── player/                    # 玩家数据（GrimPlayer）
```

### 2.4 检测接入模型

GrimAC 定义了四种检测接入时机：

| 接口 | 触发时机 | 典型用途 |
|------|----------|----------|
| PacketCheck | 每个客户端包 | 包格式/状态验证 |
| PositionCheck | 位置更新 | 触发预测引擎 |
| PostPredictionCheck | 预测引擎完成后 | 偏差分析、违规判定 |
| VehicleCheck | 载具位置更新 | 载具移动检测 |

### 2.5 架构核心原则

1. **预测驱动**：通过 PredictionEngine 模拟 MC 原版移动逻辑，生成理论位置，与客户端报告位置比对
2. **不确定性处理**：UncertaintyHandler 补偿网络延迟导致的预测误差
3. **Setback 机制**：检测到作弊时强制回滚玩家到合法位置
4. **Check 注册系统**：CheckManager 统一管理所有检测模块
5. **版本兼容**：通过 nmsutil 包实现 NMS 反射，兼容多个 MC 版本

---

## 3. 预测引擎深度分析

### 3.1 整体数据流

预测引擎是 GrimAC 的核心，它的工作流程可以概括为：**接收位置包 → 模拟所有可能的合法移动 → 找最接近实际的预测 → 计算偏差**。

```
客户端位置包
    │
    ▼
┌─── MovementCheckRunner ────────────────────────┐
│                                                  │
│  ① 计算 actualMovement = newPos - lastPos       │
│                                                  │
│  ② PlayerBaseTick.doBaseTick()                  │
│     更新流体状态、姿势、慢行状态                  │
│                                                  │
│  ③ 根据状态分发到移动模式:                        │
│     ├── 水中 → PredictionEngineWater             │
│     ├── 岩浆 → doLavaMove()                      │
│     ├── 鞘翅 → PredictionEngineElytra            │
│     └── 普通 → PredictionEngineNormal            │
│                                                  │
│  ④ PredictionEngine.guessBestMovement()          │
│     暴力枚举输入 × 碰撞检测 → 找最优预测          │
│                                                  │
│  ⑤ offset = predicted.distance(actual)           │
│     offset = uncertaintyHandler.reduceOffset()    │
│                                                  │
│  ⑥ 分发: CheckManager.onPredictionFinish()       │
│                                                  │
└──────────────────────────────────────────────────┘
```

### 3.2 核心算法: guessBestMovement

这是预测引擎的主入口，负责从所有可能的输入组合中找到最匹配的预测。

```
PROCEDURE guessBestMovement(speed, player):
    // 1. 收集起始速度向量（上一 tick 速度 + 各种推力叠加）
    init ← fetchPossibleStartTickVectors(player)
    //   来源: 上一 tick 速度 → 爆炸力 → 激流推力
    //         → 流体推力 → 攻击减速 → 阈值裁剪 → 跳跃

    // 2. 弹跳方块不确定性（史莱姆/蜂蜜块）
    IF 受弹跳方块影响:
        计算 Y 方向额外不确定性

    // 3. 0.03 判定
    player.couldSkipTick ← determineCanSkipTick(speed, init)

    // 4. 暴力枚举：对所有起始向量 × 所有输入组合
    possibleVelocities ← applyInputsToVelocityPossibilities(player, init, speed)

    // 5. 如果可能跳过 tick，添加 0.03 额外向量
    IF player.couldSkipTick:
        addZeroPointThreeToPossibilities(speed, player, possibleVelocities)

    // 6. 核心搜索
    doPredictions(player, possibleVelocities, speed)

    // 7. 碰撞处理 + tick 末尾
    MovementTickerPlayer(player).move(clientVelocity, predictedVelocity)
    endOfTick(player, gravity)
```

### 3.3 暴力枚举: loopVectors

这是预测引擎的计算密集核心。对每种可能的玩家状态组合生成候选速度向量。

```
PROCEDURE loopVectors(player, possibleVectors, speed, returnVectors):
    // 四层嵌套循环遍历所有不确定性维度
    FOR slowed ∈ {false, true}:           // 蹲下减速
        FOR usingItem ∈ {false, true}:     // 使用物品减速
            FOR each startingVelocity:      // 每个起始速度向量
                FOR strafe ∈ [-1, 0, 1]:
                    FOR forward ∈ [-1, 0, 1]:
                        FOR stuckSpeed ∈ {true, false}:  // 卡住速度

                            // 转换输入到世界坐标
                            input ← transformInputsToVector(strafe, forward)
                            result ← startingVelocity + inputToWorld(input, speed, yaw)

                            IF stuckSpeed: result *= stuckSpeedMultiplier
                            result ← handleOnClimbable(result)

                            returnVectors.add(result)
```

**复杂度:** 旧版约 `2 × 2 × N × 3 × 3 × 2 = ~72N` 个候选向量。1.21.2+ 已知精确输入后大幅减少。

### 3.4 最优搜索: doPredictions

从所有候选向量中选择与实际移动最接近的预测。

```
PROCEDURE doPredictions(player, possibleVelocities, speed):
    // 按优先级排序: 击退/爆炸 > 0.03 > 普通移动 > FirstBread > 翻转物品
    SORT possibleVelocities BY priority

    bestInput ← MAX_DOUBLE
    bestCollisionVel ← null

    FOR each candidate IN possibleVelocities:
        // 扩展为不确定性盒（处理网络延迟）
        expanded ← handleStartingVelocityUncertainty(player, candidate, actualMovement)

        // 早期裁剪优化
        IF 理论最优距离 > bestInput AND 非击退非爆炸: SKIP

        // 碰撞检测
        outputVel ← Collisions.collide(player, expanded)

        // 计算偏差
        accuracy ← outputVel.distanceSquared(player.actualMovement)

        // 更新最优解
        IF accuracy < bestInput:
            bestCollisionVel ← outputVel
            bestInput ← accuracy

            // 精度足够 → 提前退出
            IF bestInput < 1e-5² AND 无待处理击退/爆炸: BREAK

    player.predictedVelocity ← bestCollisionVel
```

### 3.5 不确定性盒: handleStartingVelocityUncertainty

将单一预测向量扩展为 3D 搜索空间，由多种不确定性来源动态计算。

```
PROCEDURE handleStartingVelocityUncertainty(player, vector, target):
    additionHorizontal ← 0, additionVertical ← 0

    // 各种状态导致的不确定性增量
    IF 飞行状态切换 (4 tick 内):  additionHorizontal += 0.3, bonusY += 0.3
    IF 水下飞行 (9 tick 内):      bonusY += 0.2
    IF 硬碰撞实体 (2 tick 内):    additionHorizontal += 0.1, bonusY += 0.1
    IF 活塞推动:                  additionHorizontal += 0.1, bonusY += 0.1
    additionHorizontal += 流体推力不确定性

    // 构建搜索空间盒 [minVector, maxVector]
    minVector ← vector - (horizontal + negUnc, bonusY + yNegUnc, horizontal + negUnc)
    maxVector ← vector + (horizontal + posUnc, bonusY + yPosUnc, horizontal + posUnc)

    // 地面不确定性: 着地瞬间 Y 速度可能归零
    IF onGroundUncertain AND vector.y < 0: maxVector.y ← 0

    // 重力/弹跳/烟花等额外来源扩展
    box ← combine(minVector, maxVector)
    box 扩展: 烟花盒、钓鱼竿拉力盒

    RETURN cutBoxToVector(target, box)
```

### 3.6 0.03 偏移估算器: PointThreeEstimator

**问题背景:** MC 1.9 移除了空闲位置包。客户端不移动时不发送位置更新，服务端可能丢失整个 tick 的移动信息。

**核心判定逻辑:**

```
PROCEDURE determineCanSkipTick(speed, init):
    // 如果上一包包含位置且不在史莱姆上 → 不是 0.03
    IF didLastMovementIncludePosition AND NOT onSlime: RETURN false

    minimum ← MAX_DOUBLE
    FOR each data IN init:
        // 将向量压向零（模拟最小移动）
        toZeroVec ← handleStartingVelocityUncertainty(player, data, ZERO)
        collisionResult ← Collisions.collide(player, toZeroVec)
        // 计算最小可能移动距离
        minHorizLength ← max(0, hypot(result.x, result.z) - speed)
        minimum ← min(minimum, hypot(verticalComponent, minHorizLength))

    RETURN minimum < threshold  // 最小可能移动 < 0.03 → 可以跳过
```

**跟踪的 0.03 相关状态:**

| 状态 | 触发条件 | 影响 |
|------|----------|------|
| isNearFluid | 0.03 范围内有水/岩浆 | 可能隐藏游泳移动 |
| headHitter | 头顶 0.03 内有方块 | Y 速度可能被截断 |
| isNearClimbable | 0.03 范围内有梯子 | 可能隐藏攀爬 |
| isGliding | 0.03 期间切换鞘翅 | 垂直控制不可预测 |
| isNearBubbleColumn | 0.03 范围内有气泡柱 | ±0.35 垂直不确定性 |

### 3.7 移动状态处理

预测引擎根据玩家所处环境分发到不同的物理模拟：

| 状态 | 引擎 | 关键参数 |
|------|------|----------|
| 陆地 | PredictionEngineNormal | 摩擦 = blockFriction × 0.91, 重力 = -0.08, 跳跃 ≈ 0.42 |
| 水中 | PredictionEngineWater | 游泳摩擦 0.8~0.96, 游泳速度 0.02, 跳跃力 0.04 |
| 岩浆 | doLavaMove | 摩擦 (0.5, 0.8, 0.5), 重力 ÷ 4 |
| 鞘翅 | PredictionEngineElytra | 摩擦 0.99/0.98, 俯冲加速, 无输入影响 |
| 飞行 | 直接信任 | predictedVelocity = actualMovement, gravity = 0 |

### 3.8 延迟补偿机制

GrimAC 使用**事务系统（Transaction）**测量往返延迟，并通过 Compensated 系列类补偿：

| 类 | 用途 |
|-----|------|
| CompensatedWorld | 回溯方块状态到玩家视角的 tick |
| CompensatedEntities | 回溯实体位置和属性 |
| CompensatedInventory | 回溯物品栏状态 |

**FirstBread 机制**处理击退/爆炸延迟：

```
击退包到达 → 标记 firstBreadKB（面包屑，可能已应用）
事务确认   → 升级为 likelyKB（确认击退）
预测运行   → handlePredictionAnalysis: 取最小偏差
只有面包屑 → 仍考虑但不完全信任
```

### 3.9 关键常量

| 常量 | 值 | 含义 |
|------|-----|------|
| 移动阈值 (1.9+) | 0.003 | 低于此值的速度分量归零 |
| 默认重力 | 0.08 | 每 tick 向下加速度 |
| 跳跃初速度 | ~0.42 | 地面跳跃 Y 初速度 |
| 空气摩擦 | 0.98 | Y 轴 tick 末尾摩擦 |
| 使用物品减速 | 0.2 | 使用物品时移动降至 20% |
| 预测提前退出 | 1e-5² | 偏差低于此值停止搜索 |
| 0.03 判定阈值 | 0.001² | 标记为 0.03 的距离阈值 |

---

## 4. 移动检测实现分析

### 4.1 检测体系架构

GrimAC 的移动检测**没有独立的 Speed 检测**。预测引擎本身就是 Speed 检测：如果实际移动超出所有预测向量的范围，offset 超阈值即判定作弊。这是一个根本性的架构决策。

**四层防护模型:**

```
Layer 1: SetbackBlocker (包级拦截)
    → 阻止明显的非法包（死亡后移动、非载具发载具包等）

Layer 2: PacketCheck (包验证)
    → FlightA, SprintA, NoFall — 状态一致性验证

Layer 3: PredictionEngine + PostPredictionCheck
    → Simulation, Phase, GroundSpoof — 高精度移动验证

Layer 4: 特化检测
    → Knockback, Explosion, NoSlow, Sprint, Elytra — 专项场景验证
```

### 4.2 Simulation 检测（OffsetHandler）

这是 GrimAC **最核心的移动检测**，直接消费预测引擎输出的偏差值。

**算法流程:**

```
PROCEDURE onPredictionComplete(offset):
    IF offset >= threshold (0.001):
        advantageGained += offset         // 累积偏差优势
        giveOffsetLenienceNextTick(offset) // 下一 tick 容错
        flag()

        // Setback 条件
        IF (advantageGained >= 1.0 OR offset >= 0.1)
           AND violations >= 1.0:
            executeSetback()

        advantageGained = min(advantageGained, 4.0)  // 封顶
    ELSE:
        advantageGained *= 0.999  // 每正确 tick 衰减 0.1%
```

**关键参数:**

| 参数 | 默认值 | 含义 |
|------|--------|------|
| threshold | 0.001 | 触发 flag 的偏差下限（约 1mm） |
| immediateSetbackThreshold | 0.1 | 立即 setback 的偏差（约 10cm） |
| maxAdvantage | 1.0 | 累积优势上限，超过则 setback |
| setbackDecayMultiplier | 0.999 | 优势衰减速率 |

**设计分析:** GrimAC 不设固定速度阈值，而是为每个玩家的每个 tick 动态计算合法速度范围。这意味着疾跑、药水、方块摩擦等状态都被自动纳入考虑，精度约 1mm。

### 4.3 Phase 检测 — 穿墙检测

Phase 检测使用**碰撞箱重叠检测**，而非预测偏差。

```
STATE: oldBB = player.boundingBox  // 上一 tick 碰撞箱

PROCEDURE onPredictionComplete():
    newBB = player.boundingBox
    boxes = Collisions.getCollisionBoxes(player, newBB)

    FOR each box IN boxes:
        IF newBB.isIntersected(box)         // 当前与方块相交
           AND NOT oldBB.isIntersected(box): // 上一 tick 不相交
            flagAndSetback()  // 一 tick 内"穿过"了方块
            RETURN

    oldBB = newBB
    reward()
```

**与预测引擎的关系:** Phase 不消费 offset，直接检查碰撞箱。两者互补，预测引擎可能因不确定性容错而错过微小穿墙。

### 4.4 GroundSpoof + NoFall — 地面欺骗与坠落伤害

**GroundSpoof（PostPredictionCheck）:**

```
IF player.clientClaimsLastOnGround != player.onGround:
    flag()
    // 通知 NoFall 翻转包中的地面状态
    noFall.flipPlayerGroundStatus = true
```

**NoFall（PacketCheck）双路并行:**

路线 A — 纯包检测（无位置变化时）：
```
IF 包类型 == LOOK/ROTATION AND 客户端声称在地面:
    feetBB = 极薄碰撞箱 (0.6 × 0.001)
    feetBB.expand(movementThreshold)
    IF NOT checkForBoxes(feetBB):  // 脚下无碰撞方块
        flag()
        wrapper.setOnGround(false)  // 修正为不在地面
```

路线 B — GroundSpoof 联动：
```
IF flipPlayerGroundStatus:
    wrapper.setOnGround(!isOnGround)  // 翻转地面状态
```

### 4.5 FlightA 检测

```
IF isFlyingPacket(event) AND NOT player.isFlying:
    flag()  // 客户端声称飞行但服务端不认可
```

设计哲学：飞行模式的复杂检测由预测引擎承担。如果玩家没有飞行权限但悬空，预测引擎的 offset 会很大，Simulation 检测会触发。

### 4.6 Knockback/Explosion 检测

使用 FirstBread 机制追踪击退/爆炸是否被正确接受。

**KnockbackHandler 算法:**

```
// 预测运行时（handlePredictionAnalysis）
firstBreadKB.offset = min(firstBreadKB.offset, offset)
likelyKB.offset = min(likelyKB.offset, offset)

// 预测完成后（onPredictionComplete）
IF likelyKB.offset > threshold (0.001):
    flag()  // 玩家未正确接受击退
    threshold = min(threshold + offset, ceiling)
    IF threshold >= maxAdvantage OR offset >= immediate:
        setback()
ELSE:
    threshold *= 0.999  // 衰减
```

### 4.7 NoSlow 检测

验证使用物品时的减速是否生效。

```
STATE: bestOffset = 1, flaggedLastTick = false

handlePredictionAnalysis(offset):
    bestOffset = min(bestOffset, offset)

onPredictionComplete():
    IF isSlowedByUsingItem():
        IF bestOffset > threshold (0.001):
            IF flaggedLastTick:  // 需要连续两 tick 违规
                flagAndSetback()
            flaggedLastTick = true
        ELSE:
            reward()
            flaggedLastTick = false
    bestOffset = 1
```

**关键洞察:** 预测引擎在暴力枚举时包含"使用物品减速"分支（speed × 0.2）。如果实际移动比所有减速预测都快，bestOffset 就会很大。

### 4.8 VL 系统

```
flag()  → violations++（每次触发 +1）
reward() → violations -= decay（每次正确行为减少）

触发 Setback: violations > setbackVL 且无 nosetback 权限
```

| 检测 | decay | setback VL |
|------|-------|-----------|
| Simulation | 0.02 | 可配置 |
| Phase | 0.005 | 1 |
| GroundSpoof | 0.01 | 10 |
| NoFall | - | 10 |
| AntiKB | 0.025 | 10 |
| NoSlow | - | 5 |

### 4.9 各检测与预测引擎的消费关系

```
预测引擎输出 (offset):
    │
    ├── OffsetHandler (Simulation)
    │   直接消费 offset → VL 累积 → setback
    │
    ├── KnockbackHandler (AntiKB)
    │   handlePredictionAnalysis(offset) → 更新击退偏差
    │
    ├── ExplosionHandler (AntiExplosion)
    │   同 KnockbackHandler 模式
    │
    ├── NoSlow
    │   handlePredictionAnalysis(offset) → bestOffset
    │
    ├── Phase
    │   不消费 offset → 碰撞箱重叠检测
    │
    ├── GroundSpoof
    │   不消费 offset → 地面状态比对
    │
    └── SprintB~G / ElytraA~I
        不消费 offset → 状态一致性/前置条件验证
```

---

## 5. 战斗检测实现分析

### 5.1 检测模块清单

| 检测名 | 行数 | 类型 | 说明 |
|--------|------|------|------|
| Reach | 371 | PacketCheck | 攻击距离 + 未命中碰撞箱检测 |
| Hitboxes | 12 | Check | 纯标记类（由 Reach 内部触发） |
| MultiInteractA | 82 | PostPredictionCheck | 同 tick 交互多个不同实体 |
| MultiInteractB | 68 | PostPredictionCheck | INTERACT_AT 位置变更 |
| SelfInteract | 45 | PacketCheck | 攻击自己 |
| AimDuplicateLook | 33 | RotationCheck | 朝向完全重复 |
| AimModulo360 | 39 | RotationCheck | 模 360° 旋转跳变 |
| AimProcessor | 75 | RotationCheck | 灵敏度估算（非检测） |
| Timer | 112 | PacketCheck | 事务时钟法包频率检测 |
| NegativeTimer | 42 | PostPredictionCheck | 负 Timer（慢包/暂停） |
| TickTimer | 44 | PacketCheck | 1.21.2+ tick 包频率 |

### 5.2 Reach 检测 — 双阶段架构

这是 GrimAC 战斗检测最核心的部分，采用两阶段设计：

**阶段 1: 实时快速检查（isKnownInvalid）**
- 在 ATTACK 包到达时立即执行
- 使用当前 tick 朝向 + 简化计算
- 用途：拦截明显不可能的攻击
- 仅在 cancelBuffer 非零时执行完整 raytrace

**阶段 2: 延迟精确检查（tickBetterReachCheckWithAngle）**
- 在下一个位置包到达时执行（拥有最新朝向）
- 考虑多种可能的朝向组合
- 产出最终判定：REACH / HITBOX / NONE

### 5.3 Reach 核心: Raycast 方法

#### 射线-AABB 碰撞检测

```
PROCEDURE checkReach(entity, x, y, z):
    targetBox ← entity.getPossibleCollisionBoxes()
    maxReach ← applyReachModifiers(targetBox, ...)

    // 收集所有可能的视线方向
    possibleLookDirs ← [(yaw, pitch)]              // 当前朝向
    IF client >= 1.8: add (lastYaw, pitch)         // 上一个 yaw
    IF client >= 1.9: add (lastYaw, lastPitch)     // 上一个朝向

    FOR lookDir IN possibleLookDirs:
        lookDir.multiply(maxReach + 3)  // 射线长度

        FOR eyeHeight IN possibleEyeHeights:
            eyePos ← (x, y + eyeHeight, z)
            endPos ← eyePos + lookDir

            // 射线-AABB 碰撞
            intercept ← calculateIntercept(targetBox, eyePos, endPos)

            IF eyePos 在 targetBox 内部:
                minDistance ← 0
            ELIF intercept 不为空:
                minDistance ← min(distance(eyePos, intercept), minDistance)
```

#### 碰撞箱扩展（而非增加距离）

```
PROCEDURE applyReachModifiers(targetBox, ...):
    maxReach ← ENTITY_INTERACTION_RANGE 属性值

    IF client < 1.9: hitboxMargin += 0.1  // 1.8 原版行为
    IF 0.03 场景: hitboxMargin += 0.03     // 移动不确定性
    targetBox.expand(hitboxMargin)         // 扩展碰撞箱

    RETURN maxReach
```

**关键设计:** 0.03 不确定性通过扩展碰撞箱而非增加距离来补偿。原因是玩家可能因网络延迟"错过"目标，扩展碰撞箱比增加距离更合理。

#### 判定与惩罚

```
REACH  → minDistance > maxReach（超出攻击距离）
HITBOX → minDistance == MAX（射线完全未命中碰撞箱）
NONE   → 合法攻击

cancelBuffer 机制:
    IF REACH/HITBOX: cancelBuffer = 1（激活激进检查）
    ELSE: cancelBuffer = max(0, cancelBuffer - 0.25)（缓慢衰减）
```

### 5.4 实体插值双边界追踪（ReachInterpolationData）

这是 GrimAC Reach 检测最精密的部分。

**问题:** MC 客户端的实体移动通过插值平滑显示。攻击检测必须考虑玩家攻击时实体处于插值的哪个阶段。

**插值步数:**

| 实体类型 | 插值步数 |
|----------|----------|
| 船 | 10 |
| 矿车 | 5 |
| 生物 | 3 |
| 其他 | 1 |

**双边界追踪:**

```
interpolationStepsLowBound  — 插值最少进行到的步数
interpolationStepsHighBound — 插值最多可能进行到的步数

// 1.9+ 可能跳过 tick 时:
highBound 从 0 开始，每 tick +1（不确定）
// 可靠 tick 时:
lowBound = min(lowBound + 1, totalSteps)
highBound = min(highBound + 1, totalSteps)
```

**碰撞箱合并:** 利用线性函数极值性质，只需计算 low 和 high 两步的并集。

```
loBox ← startPos + lowBound × stepDelta
hiBox ← startPos + highBound × stepDelta
RETURN combine(loBox, hiBox)  // 外包盒
```

### 5.5 攻击排队机制

```
// 攻击时保存上下文（不立即判定）
playerAttackQueue.put(entityId, InteractionData(
    player.x, player.y, player.z,  // 攻击者位置快照
    hasAttackRange, maxReach, hitboxMargin
))

// 下一个位置包到达时，使用保存的位置 + 最新朝向判定
tickBetterReachCheckWithAngle():
    FOR each queued attack:
        result ← checkReach(entity, savedX, savedY, savedZ, ...)
```

### 5.6 七层延迟补偿体系

| 层级 | 机制 | 补偿内容 |
|------|------|----------|
| L1 | 事务 RTT | 所有检测的时间基准 |
| L2 | 攻击排队 | 位置-朝向时序偏差 |
| L3 | 实体插值双边界 | 目标实体位置不确定性 |
| L4 | 朝向多假设 | 1~3 种视线方向 |
| L5 | 眼高多假设 | 不同姿态的眼高 |
| L6 | 碰撞箱扩展 | 测量误差容忍 |
| L7 | 0.03 延迟判定 | PostPredictionCheck 延迟 flag |

### 5.7 Timer 检测 — 事务时钟法

**核心创新:** 不使用真实时间，以事务往返时间为基准测量包频率。

```
// 初始化
timerBalanceRealTime ← 0
knownPlayerClockTime ← 当前时间 - 60秒

// 每个移动包到达:
timerBalanceRealTime += 50ms  // 累加一个 tick

// 每个事务确认到达:
knownPlayerClockTime ← lastMovementPlayerClock
lastMovementPlayerClock ← player.getPlayerClockAtLeast()

// 检测判定:
IF timerBalanceRealTime > System.nanoTime():
    flag()  // 包累计时间超过当前真实时间 → 发包过快
    timerBalanceRealTime -= 50ms

// 防止累积延迟:
timerBalanceRealTime ← max(timerBalanceRealTime,
                           lastMovementPlayerClock - clockDrift)
```

**clockDrift 默认 120ms**，允许时钟漂移防止网络波动误判。TimerLimit 针对高 ping（>1000ms）玩家限制 balance 滥用。

### 5.8 瞄准检测

**AimProcessor — 灵敏度估算（GCD 众数法）:**

```
PROCEDURE process(rotationUpdate):
    deltaXRot ← |currentYaw - lastYaw|
    divisorX ← gcd(deltaXRot, lastXRot)

    IF valid(divisorX):
        xRotMode.add(divisorX)  // 滑动窗口（80 样本）

    IF xRotMode 样本 > 15 AND 众数计数 > 15:
        sensitivityX ← convertToSensitivity(众数)

// 灵敏度转换:
sensitivity = (cbrt(divisor / 0.15 / 8.0) - 0.2) / 0.6
```

从旋转变化中反推鼠标灵敏度，80 样本窗口 + 15 次确认阈值。这对 Aimbot 分析非常有价值。

**AimDuplicateLook:** 连续两个包朝向完全相同 → Aimbot 特征。

**AimModulo360:** 检测作弊客户端 % 360 操作导致的旋转跳变（变化从 <30° 突然跳到 >320°）。

### 5.9 多目标/KillAura 检测

**MultiInteractA:** 同一 tick 内交互了不同实体 → KillAura 特征。0.03 场景下延迟到 PredictionComplete 判定。

**Hitboxes:** Raycast 完全未命中碰撞箱时触发。含义是攻击方向上根本不可能命中目标，这是 KillAura/ForceField 的强烈特征。

### 5.10 战斗检测关键常量

| 常量 | 值 | 来源 |
|------|-----|------|
| Reach threshold | 0.0005 | 默认碰撞箱扩展量 |
| 1.8 碰撞箱余量 | 0.1 | 原版行为 |
| Timer clockDrift | 120ms | Timer 默认值 |
| 攻击队列上限 | 10 条 | playerAttackQueue |
| cancelBuffer 衰减 | 0.25/次 | 每次合法攻击 |
| Reach setback | 10 次 VL | @CheckData |
| AimProcessor 窗口 | 80 样本 | TOTAL_SAMPLES_THRESHOLD |
| 灵敏度确认阈值 | 15 样本 | SIGNIFICANT_SAMPLES_THRESHOLD |
| 1.8→1.9 精度补偿 | 0.03125 | 版本间精度损失 |
| Timer tick 累加 | 50ms | 每次移动包 |

---

## 6. 与 FrontLeaves 的对比与启示

### 6.1 架构差异

| 维度 | GrimAC | FrontLeaves (AntiCheatSub) |
|------|--------|---------------------------|
| 运行位置 | MC 服务端 JVM 内部 | 外部 Go 进程 |
| 数据来源 | 直接拦截客户端包 | gRPC Client Stream 转发 |
| 检测精度 | per-tick 预测引擎，~1mm | 500ms 批量处理 |
| 延迟补偿 | 事务包 + CompensatedWorld | 无 |
| 惩罚机制 | Setback + VL 衰减 + 包修改 | 仅记录警告 + 风险分 |

### 6.2 检测能力差距

| 检测类型 | GrimAC | FrontLeaves | 差距 |
|----------|--------|-------------|------|
| Speed | 预测引擎隐含，~1mm 精度 | speed > 12.0 格/秒固定阈值 | 巨大 |
| Fly | FlightA + Simulation | 无 | 巨大 |
| NoFall | GroundSpoof + NoFall 双层 | 无 | 巨大 |
| Phase | 碰撞箱重叠检测 | 无 | 巨大 |
| Reach | Raycast 射线-AABB | 欧氏距离 > 3.5 | 较大 |
| Velocity | FirstBread 追踪 + 偏差 | 无 | 巨大 |
| Timer | 事务时钟法 | 无 | 巨大 |

### 6.3 可借鉴的技术（按优先级）

**P0 — Timer 检测（事务时钟法）**

可行性：高。利用 gRPC 心跳 RTT 作为事务基准，纯逻辑实现，不依赖世界数据。

**P1 — Raycast Reach 检测**

可行性：高。Go 端可实现精确的射线-AABB 碰撞检测，替代当前的欧氏距离判定。

**P2 — 灵敏度估算（GCD 众数法）**

可行性：高。纯数学算法，从朝向包变化中提取鼠标灵敏度，无需世界数据。

**P3 — 动态速度阈值**

可行性：高。基于玩家状态（飞行/疾跑/蹲下/药水）动态调整速度阈值，替代固定 12.0。

**P4 — 攻击排队机制**

可行性：中。需要 MC 插件层保存攻击上下文，通过 gRPC 转发后延迟判定。

**不可移植的技术:**

| 技术 | 原因 |
|------|------|
| 预测引擎本体 | 外部进程无法获取精确碰撞箱 |
| 实体插值追踪 | 需要实时追踪所有实体移动包 |
| CompensatedEntities | 需要 NMS 级别实体属性访问 |
| 包级拦截/修改 | 需要在 Java 插件层实现 |

### 6.4 核心结论

FrontLeaves 最适合借鉴 GrimAC 的**检测分层思想**和 **VL 衰减 + Setback 系统**，在 Java 插件层增强数据采集，Go 端做统计型/趋势型检测。完全复制预测引擎不可行（外部架构限制），但可以逐步提升检测精度。

---

## 7. GPL-3.0 合规声明

### 7.1 分析依据

本报告基于对 GrimAC（https://github.com/GrimAnticheat/Grim）源码的阅读和分析撰写。GrimAC 采用 GPL-3.0 许可证发布。

### 7.2 合规措施

1. **所有算法描述使用伪代码**：报告中出现的代码片段均为独立撰写的伪代码（PROCEDURE/IF/FOR 等关键字），非 GPL-3.0 许可源码的原文摘录
2. **不包含源码复制**：报告中不包含 GrimAC 源码的连续 5 行以上复制
3. **分析目的**：本报告旨在理解反作弊检测原理，为 FrontLeaves 项目提供技术参考
4. **非衍生作品**：本报告是对 GrimAC 源码的描述性分析文档，不构成 GPL-3.0 定义的衍生作品

### 7.3 使用范围

本报告仅供 FrontLeaves 项目团队内部参考。报告中描述的算法原理可供实现参考，但 FrontLeaves 的实现应为独立编写，不复制 GrimAC 的源码。

---

*分析完成时间: 2026-05-27*
*分析方法: 源码阅读 + 伪代码描述*
*源码版本: GrimAC commit ec08cbee*
