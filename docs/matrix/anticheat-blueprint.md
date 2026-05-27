# FrontLeaves 反作弊方案说明书

> **版本**: 1.0.0
> **日期**: 2026-05-27
> **作者**: 筱锋
> **输入文档**: T10 Matrix 反编译报告 / T11 GrimAC 源码分析 / T12 对比分析 / Proto 契约 / 架构文档
> **定位**: 基于 T12 对比分析结论的可执行方案说明书，不重复 `anti-cheat-research.md` 的实现优先级建议

---

## 目录

1. [概述](#1-概述)
2. [检测模块规划](#2-检测模块规划)
3. [算法详解](#3-算法详解)
4. [外部进程架构适配](#4-外部进程架构适配)
5. [实现路线图](#5-实现路线图)
6. [风险与局限](#6-风险与局限)
7. [附录](#7-附录)

---

## 1. 概述

### 1.1 研究背景

FrontLeaves 的 Matrix 子系统采用**外部进程检测架构**（模式 C）。MC 服务端（Java/Paper）仅负责数据采集，通过 gRPC Client Stream 上报 18 种事件类型；Go 后端独立完成统计聚合和反作弊检测。

当前检测能力有限：Speed 使用固定阈值 12.0 blocks/s，Reach 使用欧氏距离 3.5 blocks。两者均为最基础的检测手段，与 Matrix（闭源）和 GrimAC（开源）等专业反作弊方案差距显著。

> **配置校准说明**: 以下 Matrix 参数均来自 Matrix 7.19.4 实际运行配置文件（`config.yml` + `checks.yml`），非反编译推断值。部分检测在实际部署中被禁用（Delay、Block、Interact、Phase: `enable: false`），多个子检测通过 `disabled_components` 禁用（包括 bp.pe、bp.freecam、bp.badpacket.a/b、ka.consume、hb.mis、move.sprint、move.gf、ely.*、vehicle.speed）。Matrix 还内置 TPS 保护（min_tps: 17.0, lag_threshold: 1000ms），在服务器卡顿时自动放宽检测。

### 1.2 研究方法

本次研究通过对 Matrix 7.19.4（CFR 反编译）和 GrimAC（GPL-3.0 源码阅读）进行 5 维度源码级对比分析，提取可直接在外部进程中实现的检测技术，形成本方案。

核心方法论：

1. **可移植性筛选** — 从 Matrix/GrimAC 的检测技术中，筛选不依赖 NMS/精确碰撞箱/包拦截的算法
2. **Proto 契约验证** — 每个检测模块必须对应 Proto 契约中已有的 18 种事件类型之一，或明确标注需要增强
3. **延迟兼容性评估** — 外部进程存在 50-200ms 延迟，评估每个算法在此延迟下的有效性

### 1.3 结论摘要

**核心发现**: 外部进程无法复制 GrimAC 的预测引擎（需要精确碰撞箱和 NMS 访问），也不能完全移植 Matrix 的碰撞检测（需要 PacketEvents 包拦截）。但存在大量**纯逻辑/纯数学**的检测技术可以移植。

**可直接实现的 6 大检测技术**:

| 技术 | 来源 | 依赖 | 适用性 |
|------|------|------|--------|
| 事务时钟法 Timer | GrimAC | gRPC 心跳 RTT | 外部可实现 |
| GCD 众数灵敏度估算 | GrimAC | 朝向数据 | 外部可实现 |
| Raycast Reach 检测 | Matrix/GrimAC | 攻击者/目标位置 | 外部可实现 |
| 动态速度阈值 | Matrix | 玩家状态 + 位置序列 | 外部可实现 |
| VL 衰减 + 分层判定 | GrimAC | 无外部依赖 | 外部可实现 |
| 滑动窗口频率分析 | Matrix | 事件时间戳 | 外部可实现 |

**需要 MC 端增强的 3 项**:

| 技术 | 缺失数据 | 增强方式 |
|------|---------|---------|
| 攻击排队机制 | 攻击时位置快照 + 目标 ID | MC 端保存上下文并扩展 EntityDamageEvent |
| 多目标 KillAura | 周围实体列表 | MC 端在攻击事件中上报附近实体 |
| X-Ray 矿石统计 | 方块破坏序列已有 | 无需增强，需新增分析逻辑 |

---

## 2. 检测模块规划

按实现优先级排列。每个模块标注适用性类别：

- **外部可实现**: Go 端可独立完成，数据来源已有 Proto 契约支撑
- **混合可实现**: Go 端 + MC 插件端协作，MC 端需小幅增强
- **内部可实现**: 需在 MC 插件端 Java 层实现（如需 NMS/包拦截）

### 模块 1: Speed — 动态摩擦力阈值检测

**优先级**: P0（已有基础，升级改造）

| 维度 | 内容 |
|------|------|
| **检测算法** | Matrix 摩擦力模型 + 加速度追踪。位移加速度 = Δcurrent − Δprevious × friction，结合玩家状态动态调整合法速度范围 |
| **数据来源** | TelemetryTick（pos_x/y/z, velocity_x/y/z, is_sneaking, is_sprinting, is_flying, active_effects） |
| **关键参数** | 基础速度 0.135 b/tick，摩擦系数 0.91，容差 1.4x，Speed 药水 +0.0265/level/tick，疾跑 ×1.3 |
| **实现难度** | 低 |
| **预期效果** | 检测持续 Speed 作弊，精度约 0.1 blocks。受采样频率限制（1-2s/次），短时加速可能漏检 |
| **适用性** | **外部可实现** |
| **Proto 支撑** | `TelemetryTick` 已包含 pos/velocity/sprinting/sneaking/flying/active_effects 全部字段 |

**升级路径**: 当前固定 12.0 → 动态阈值（根据 is_sprinting/is_sneaking/active_effects 调整基准速度）

### 模块 2: Reach — Raycast 射线-AABB 碰撞检测

**优先级**: P0（已有基础，核心升级）

| 维度 | 内容 |
|------|------|
| **检测算法** | Matrix/GrimAC 的射线-AABB 碰撞检测。从攻击者眼高发射射线到目标方向，检测射线与目标 AABB 的交点距离。替代当前欧氏距离 |
| **数据来源** | EntityDamageEvent（player_x/y/z, player_yaw/pitch, entity_x/y/z, entity_type） |
| **关键参数** | 基础攻击距离 3.15 blocks（Matrix checks.yml 配置，MC 标准 3.0），碰撞箱扩展 0.1（容错），眼高 1.62，AABB 宽 0.6 × 高 1.8（玩家），cancel_way: none（不取消攻击，仅通知），dynamic_vl: disable（未启用动态 VL），max_burst_vl: 12（5秒内 VL 最大增幅），trace_back_length: 8（回溯 tick 长度） |
| **实现难度** | 中 |
| **预期效果** | 精度从"±0.5 格误差"提升到"±0.1 格误差"。可检测 3.0-3.5 格之间的可疑攻击，减少 3.0 格内的误报 |
| **适用性** | **外部可实现** |
| **Proto 支撑** | `EntityDamageEvent` 已包含 player_x/y/z, player_yaw/pitch, entity_x/y/z, entity_type |

**关键改进**: 当前欧氏距离未考虑朝向（玩家面朝反方向也能"打到"身后的目标），Raycast 自然解决了这个问题。

### 模块 3: Timer — 事务时钟法

**优先级**: P1（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | GrimAC 事务时钟法。以 gRPC 心跳 RTT 为事务基准，每次收到 TelemetryTick 累加 50ms，balance 超过当前真实时间即判定发包过快 |
| **数据来源** | TelemetryTick（timestamp）+ gRPC 连接级 RTT 测量 |
| **关键参数** | tick 累加 50ms，clockDrift 容忍 120ms，TimerLimit 限制高 ping 玩家 |
| **实现难度** | 低 |
| **预期效果** | 检测 Timer 类作弊（加速游戏时钟），完全不受 ping 影响。精度 ±120ms |
| **适用性** | **外部可实现** |
| **Proto 支撑** | `TelemetryTick.timestamp` 提供时间序列，gRPC 连接级 RTT 可从 HTTP/2 PING 帧获取 |

**移植说明**: GrimAC 用事务包（Transaction Packet）测量 RTT，本项目用 gRPC HTTP/2 PING 帧替代，功能等价。

### 模块 4: Aimbot — GCD 众数灵敏度估算

**优先级**: P2（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | GrimAC AimProcessor 的 GCD 众数法。从连续朝向变化中计算 GCD，用众数统计（80 样本窗口）反推鼠标灵敏度。灵敏度值不一致或异常即 Aimbot 特征 |
| **数据来源** | TelemetryTick（yaw, pitch）+ EntityDamageEvent（player_yaw, player_pitch） |
| **关键参数** | 窗口 80 样本，确认阈值 15 次，灵敏度范围 [0, 1] |
| **实现难度** | 中 |
| **预期效果** | 检测 Aimbot 的鼠标模拟痕迹。正常玩家灵敏度稳定，Aimbot 灵敏度值跳变或不在合理范围 |
| **适用性** | **外部可实现** |
| **Proto 支撑** | `TelemetryTick.yaw/pitch` 和 `EntityDamageEvent.player_yaw/pitch` 提供朝向序列 |

### 模块 5: X-Ray — 矿石分布统计

**优先级**: P2（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | 统计玩家挖掘的矿石比例。正常玩家的钻石/绿宝石/远古残骸等稀有矿物占比符合正态分布，X-Ray 玩家显著偏高 |
| **数据来源** | BlockBreakEvent（material, x/y/z, timestamp） |
| **关键参数** | 滑动窗口 100 次挖掘，稀有矿物占比阈值（钻石 >8% 可疑），深度分层统计 |
| **实现难度** | 低 |
| **预期效果** | 检测 X-Ray/矿透作弊。对"偶尔看一次"的轻度作弊不敏感，对持续使用 X-Ray 的玩家有效 |
| **适用性** | **外部可实现** |
| **Proto 支撑** | `BlockBreakEvent` 已包含 material + x/y/z 坐标，足够做矿石比例和深度分布统计 |

### 模块 6: AutoClicker — 点击间隔统计分析

**优先级**: P2（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | Matrix CPS 时序分析 + 统计分布检验。统计攻击事件的 CPS（Clicks Per Second），分析间隔分布的标准差和自相关系数。AutoClicker 的间隔分布过于均匀或过于高频 |
| **数据来源** | EntityDamageEvent（timestamp, player_uuid） |
| **关键参数** | CPS 上限 18（Matrix checks.yml: max_cps: 18），滑动窗口 30 秒，标准差下限 0.02（过于均匀即可疑） |
| **实现难度** | 低 |
| **预期效果** | 检测高频 AutoClicker（>18 CPS）和低频均匀分布的 AutoClicker（"jitter" 类） |
| **适用性** | **外部可实现** |
| **Proto 支撑** | `EntityDamageEvent.timestamp` 提供攻击时间序列 |

### 模块 7: Fly — 多维度状态机

**优先级**: P2（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | Matrix 5 维状态机简化版。追踪 ΔY 模式（上升/下降是否符合重力）、水平位移上限、地面状态一致性、连续性追踪 |
| **数据来源** | TelemetryTick（pos_y, velocity_y, is_on_ground, is_flying, is_in_water） |
| **关键参数** | 跳跃初速 ~0.42 b/tick，重力 -0.08 b/tick²，空中摩擦 0.98，VL 上限 10.0 |
| **实现难度** | 高 |
| **预期效果** | 检测 Fly/HighJump/悬浮。精度受限（1-2s 采样），对短时跳飞可能漏检，对持续飞行有效 |
| **适用性** | **外部可实现**（简化版）/ 混合可实现（完整版需 MC 端增加碰撞检测） |
| **Proto 支撑** | `TelemetryTick` 包含 pos_y, velocity_y, is_on_ground, is_flying, is_in_water |

### 模块 8: KillAura — 多目标切换频率分析

**优先级**: P3（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | Matrix 多目标分析 + GrimAC MultiInteract 概念。统计短时间窗口内攻击不同目标的切换频率。KillAura 的切换频率显著高于正常战斗 |
| **数据来源** | EntityDamageEvent（entity_type, entity_x/y/z, player_uuid, timestamp） |
| **关键参数** | 窗口 5 秒，切换次数上限 8，距离关联 ≤ 5.0 blocks |
| **实现难度** | 中 |
| **预期效果** | 检测多目标自动切换行为。对"先打 A 再打 B"的正常战斗可能误报，需要足够样本量 |
| **适用性** | **混合可实现** |
| **Proto 支撑** | `EntityDamageEvent` 已有 entity_type 和 entity_x/y/z，但缺少 entity_id（无法区分同名实体） |

**需要增强**: EntityDamageEvent 增加 `entity_id` 字段，用于精确区分不同实体。

### 模块 9: FastBreak — 滑动窗口计时

**优先级**: P3（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | Matrix 滑动窗口法。追踪 BlockBreakEvent 的时间间隔，与预期挖掘时间（基于材质和工具）对比 |
| **数据来源** | BlockBreakEvent（material, tool_used, timestamp） |
| **关键参数** | 窗口大小 4，比率阈值 ≤ 0.82（参考 Matrix FastConsume） |
| **实现难度** | 低 |
| **预期效果** | 检测加速挖掘，对 Nuker（范围破坏）有一定检测能力 |
| **适用性** | **外部可实现** |
| **Proto 支撑** | `BlockBreakEvent` 已包含 material, tool_used, timestamp |

### 模块 10: NoFall — 坠落伤害规避

**优先级**: P3（全新模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | Matrix ground_spoof 标记简化版。追踪玩家声称 onGround 的状态与 Y 坐标变化的一致性。持续下降但声称在地面即可疑 |
| **数据来源** | TelemetryTick（pos_y, is_on_ground, velocity_y） |
| **关键参数** | 下降速度 > 0.5 blocks/s 且 is_on_ground == true，连续 ≥ 3 个 Tick |
| **实现难度** | 中 |
| **预期效果** | 检测 NoFall 作弊（客户端声称在地面避免坠落伤害）。精度受采样频率限制 |
| **适用性** | **混合可实现** |
| **Proto 支撑** | `TelemetryTick` 包含 pos_y, is_on_ground, velocity_y，但无法验证实际脚下碰撞 |

### 模块 11: Phase — 穿墙检测

**优先级**: P4（远期模块）

| 维度 | 内容 |
|------|------|
| **检测算法** | 速度突变检测。当水平位移突增（上一帧 <0.1，下一帧 >0.5）且排除传送/Velocity 后标记 |
| **数据来源** | TelemetryTick（pos_x/y/z） |
| **关键参数** | 水平突变阈值 >0.5 blocks，排除传送/重生事件 |
| **实现难度** | 中 |
| **预期效果** | 检测明显的穿墙行为（Phase/NoClip）。对渐进式穿墙无效 |
| **适用性** | **内部可实现**（精确检测需碰撞箱查询，外部只能做粗粒度突变检测） |

---

## 3. 算法详解

### 3.1 移动检测算法

#### 3.1.1 Speed — 动态摩擦力阈值

参考 Matrix `9q_0.java` 的摩擦力模型，适配外部进程的采样频率。

```
PROCEDURE detectSpeed(tick):
    // 1. 免检条件过滤
    IF tick.is_flying OR tick.is_in_water:
        SKIP  // 飞行/游泳状态不检测
    IF not hasPreviousTick:
        SAVE tick, RETURN

    // 2. 计算位移和时间间隔
    Δx = tick.pos_x - prev.pos_x
    Δy = tick.pos_y - prev.pos_y
    Δz = tick.pos_z - prev.pos_z
    Δt = (tick.timestamp - prev.timestamp) / 1000.0  // 秒
    IF Δt < 0.1: SKIP  // 间隔太短，跳过

    horizontalSpeed = sqrt(Δx² + Δz²) / Δt
    verticalSpeed = |Δy| / Δt

    // 3. 动态阈值计算
    baseSpeed = 4.317  // blocks/s，MC 步行速度（0.135 b/tick × 20 ticks/s ≈ 2.7，MC Wiki 步行实测约 4.317 blocks/s）
    IF tick.is_sprinting:
        baseSpeed *= 1.3  // 疾跑 ×1.3
    IF tick.is_sneaking:
        baseSpeed *= 0.3  // 蹲下 ×0.3

    // 4. 药水效果补偿
    FOR effect IN tick.active_effects:
        IF effect.startsWith("SPEED"):
            level = parseAmplifier(effect)
            baseSpeed += 0.85 * (level + 1)  // 0.0265 b/tick → blocks/s 转换

    // 5. 容差
    threshold = baseSpeed * 1.4  // Matrix 的 1.4x 容差

    // 6. 判定
    IF horizontalSpeed > threshold:
        VL += speedExcess * multiplier
        IF VL > VL_THRESHOLD:
            FLAG("SPEED", horizontalSpeed, threshold)
    ELSE:
        VL = max(0, VL - 0.5)  // 衰减
```

**与当前实现的对比**:

| 维度 | 当前 | 升级后 |
|------|------|--------|
| 阈值 | 固定 12.0 | 动态 4.317 × 状态系数 × 1.4 |
| 状态感知 | 仅排除飞行 | 疾跑/蹲下/药水/游泳全考虑 |
| 累积判定 | 单次超限即 flag | VL 累积 + 衰减，降低误报 |
| 方向检测 | 无 | 可扩展（参考 6D 方向性检测） |

#### 3.1.2 Fly — 多维度状态机

参考 Matrix `bg.java` 的 5 维状态机，简化为外部进程可实现的版本。

```
STATE:
    lastGroundY: float        // 最后确认的地面 Y 坐标
    consecutiveAirTicks: int  // 连续空中 Tick 数
    lastDeltaY: float         // 上一帧 ΔY
    flyVL: float              // Fly 违规等级

PROCEDURE detectFly(tick):
    IF tick.is_flying OR tick.is_in_water:
        RESET_STATE, RETURN

    Δy = tick.pos_y - prev.pos_y

    // 1. 地面状态一致性
    IF tick.is_on_ground:
        lastGroundY = tick.pos_y
        consecutiveAirTicks = 0
        flyVL = max(0, flyVL - 0.5)
    ELSE:
        consecutiveAirTicks++

    // 2. 空中上升检测（违反重力）
    IF NOT tick.is_on_ground AND Δy > 0:
        // 跳跃后 ΔY 应递减（空气摩擦 0.98）
        // 正常: ΔY(n) ≈ (ΔY(n-1) - 0.08) × 0.98
        expectedΔy = (lastDeltaY - 0.08) * 0.98
        IF consecutiveAirTicks > 2 AND Δy > expectedΔy + 0.05:
            flyVL += 1.0

    // 3. 悬浮检测
    IF consecutiveAirTicks > 20:  // 连续 20 个采样（约 20-40 秒）在空中
        IF |Δy| < 0.01:  // Y 坐标几乎不变
            flyVL += 0.5

    // 4. 判定
    IF flyVL > 5.0:
        FLAG("FLY", Δy, consecutiveAirTicks)

    lastDeltaY = Δy
```

**局限性**: 外部进程采样频率 1-2s/次，MC 内部 50ms/tick。这意味着连续 20 个采样对应 20-40 秒，而内部插件只需 1 秒就能检测到 Fly。此检测主要针对**持续性飞行**。

#### 3.1.3 NoFall — 坠落伤害规避

参考 Matrix/NoFall 嵌入 Fly 的 ground_spoof 标记。

```
PROCEDURE detectNoFall(tick):
    IF tick.is_on_ground AND prev.valid:
        // 玩家声称在地面，检查是否真的在下降
        Δy = tick.pos_y - prev.pos_y
        IF Δy < -0.5:  // 下降超过 0.5 格但声称在地面
            noFallCounter++
            IF noFallCounter >= 3:
                FLAG("NOFALL", Δy, tick.is_on_ground)
    ELSE:
        noFallCounter = max(0, noFallCounter - 1)
```

### 3.2 战斗检测算法

#### 3.2.1 Reach — Raycast 射线-AABB 碰撞检测

**注意**: Matrix 的 HitBox 检测当前配置为 `cancel_way: none`（不取消攻击，仅记录 VL），`cancel_vl: -1`（永不自动取消）。这意味着检测主要依赖 VL 累积触发命令（通知/踢出），而非实时阻止攻击。

这是本次升级的**核心算法**，参考 Matrix `6e_0.java` 和 GrimAC `Reach.java`。

```
STRUCT AABB:
    minX, minY, minZ: float
    maxX, maxY, maxZ: float

STRUCT Vector3:
    x, y, z: float

// 构建实体碰撞箱
FUNCTION buildEntityAABB(entityType, ex, ey, ez):
    IF entityType == "PLAYER":
        halfW = 0.3   // 宽 0.6 / 2
        height = 1.8
    ELSE IF entityType == "ZOMBIE" OR entityType == "SKELETON":
        halfW = 0.3
        height = 1.95
    ELSE:
        halfW = 0.3
        height = 1.8

    // 碰撞箱扩展（容错）
    expand = 0.1  // 1.8 客户端余量 + 测量误差
    RETURN AABB(
        ex - halfW - expand, ey,                ez - halfW - expand,
        ex + halfW + expand, ey + height + expand, ez + halfW + expand
    )

// Ray-AABB 碰撞检测
FUNCTION rayAABBIntersect(origin, direction, box):
    tMin = -INFINITY
    tMax = +INFINITY

    FOR axis IN {x, y, z}:
        IF direction[axis] ≈ 0:
            IF origin[axis] < box.min[axis] OR origin[axis] > box.max[axis]:
                RETURN null  // 射线平行于轴且不在范围内
        ELSE:
            t1 = (box.min[axis] - origin[axis]) / direction[axis]
            t2 = (box.max[axis] - origin[axis]) / direction[axis]
            IF t1 > t2: SWAP(t1, t2)
            tMin = max(tMin, t1)
            tMax = min(tMax, t2)
            IF tMin > tMax: RETURN null

    IF tMin < 0: tMin = tMax
    IF tMin < 0: RETURN null
    RETURN tMin  // 碰撞距离

// 主检测函数
PROCEDURE detectReach(event):
    // 攻击者位置和朝向
    eyeHeight = 1.62
    origin = Vector3(event.player_x, event.player_y + eyeHeight, event.player_z)

    // 朝向单位向量
    radYaw = radians(event.player_yaw)
    radPitch = radians(event.player_pitch)
    dirX = -sin(radYaw) * cos(radPitch)
    dirY = -sin(radPitch)
    dirZ = cos(radYaw) * cos(radPitch)
    direction = normalize(Vector3(dirX, dirY, dirZ))

    // 目标碰撞箱
    targetBox = buildEntityAABB(event.entity_type,
                                 event.entity_x, event.entity_y, event.entity_z)

    // 射线碰撞检测
    distance = rayAABBIntersect(origin, direction, targetBox)

    // 判定
    maxReach = 3.15  // Matrix checks.yml: max_reach（MC 标准 3.0，Matrix 配置 3.15 作为容差基准）
    IF distance != null AND distance > maxReach:
        reachVL += (distance - maxReach) * 2
        IF reachVL > 3.0:
            FLAG("REACH", distance, maxReach)
    ELSE IF distance == null:
        // 射线完全未命中碰撞箱 → Hitbox 检测（KillAura 特征）
        hitboxVL += 1.0
    ELSE:
        reachVL = max(0, reachVL - 0.25)  // cancelBuffer 衰减
```

**与当前欧氏距离的对比**:

```
欧氏距离: d = sqrt((px-ex)² + (py-ey)² + (pz-ez)²)
         问题: 不考虑朝向，面朝反方向也能"打到"

Raycast:  从眼睛发射射线，检测与目标碰撞箱的交点
         优势: 自然排除"打身后"，精度更高
```

#### 3.2.2 KillAura — 多目标切换频率分析

参考 Matrix `ci.java` 的多目标分析和 GrimAC `MultiInteractA` 的同 tick 多目标检测。

```
STATE:
    attackHistory: []AttackRecord  // 滑动窗口
    switchCounter: int

STRUCT AttackRecord:
    timestamp: int64
    targetEntity: string  // entity_type + entity_x/y/z 的哈希
    targetPos: Vector3

PROCEDURE detectKillAura(event):
    target = hash(event.entity_type, event.entity_x, event.entity_y, event.entity_z)
    record = AttackRecord(event.timestamp, target, Vector3(ex, ey, ez))
    attackHistory.append(record)

    // 清理超过 5 秒的记录
    cutoff = event.timestamp - 5000
    attackHistory = filter(r -> r.timestamp > cutoff)

    // 1. 切换频率统计
    uniqueTargets = countUnique(attackHistory.map(r -> r.targetEntity))
    IF uniqueTargets > 3 AND len(attackHistory) > 5:
        // 5 秒内攻击 3+ 不同目标，且攻击次数 >5
        switchScore = uniqueTargets / len(attackHistory)
        IF switchScore > 0.6:
            killAuraVL += switchScore * 2

    // 2. 方向一致性检测（攻击方向与目标方向的角度偏差）
    attackDir = normalize(targetPos - playerPos)
    lookDir = normalize(yawPitchToDir(event.player_yaw, event.player_pitch))
    angle = acos(dot(attackDir, lookDir))
    IF angle > 1.57:  // > 90° → 攻击方向与视线方向相差过大
        killAuraVL += 0.5

    // 3. 判定
    IF killAuraVL > 5.0:
        FLAG("KILLAURA", uniqueTargets, switchScore)
    ELSE:
        killAuraVL = max(0, killAuraVL - 0.1)
```

#### 3.2.3 AutoClicker — 点击间隔统计分析

参考 Matrix `9s_0.java` 的 CPS 时序分析。

```
STATE:
    clickTimestamps: []int64  // 滑动窗口
    lastCPSStdDev: float

PROCEDURE detectAutoClicker(event):
    clickTimestamps.append(event.timestamp)

    // 清理超过 30 秒的记录
    cutoff = event.timestamp - 30000
    clickTimestamps = filter(t -> t > cutoff)

    IF len(clickTimestamps) < 10: RETURN  // 样本不足

    // 1. CPS 计算
    timeSpan = (clickTimestamps[-1] - clickTimestamps[0]) / 1000.0
    IF timeSpan < 1.0: RETURN
    cps = len(clickTimestamps) / timeSpan

    // 2. 间隔分布分析
    intervals = []
    FOR i = 1 TO len(clickTimestamps) - 1:
        intervals.append(clickTimestamps[i] - clickTimestamps[i-1])

    meanInterval = average(intervals)
    stdDevInterval = standardDeviation(intervals)

    // 3. 判定
    IF cps > 18:
        // 高频 AutoClicker
        autoClickerVL += (cps - 18) * 0.5
        FLAG_IF_VL("AUTOCLICKER_HIGH_CPS", cps)

    IF stdDevInterval < 2.0 AND len(intervals) > 20:
        // 间隔过于均匀（Jitter 类 AutoClicker）
        autoClickerVL += 1.0
        FLAG_IF_VL("AUTOCLICKER_UNIFORM", stdDevInterval)

    // 衰减
    autoClickerVL = max(0, autoClickerVL - 0.1)
```

### 3.3 辅助检测算法

#### 3.3.1 Timer — 事务时钟法

参考 GrimAC `Timer.java`，用 gRPC 心跳 RTT 替代事务包。

```
STATE:
    timerBalance: float = 0       // 累计时间余额（ms）
    lastKnownClock: int64 = 0     // 上次确认的客户端时钟
    lastRTT: int64 = 0            // 最近一次 RTT（ms）

PROCEDURE detectTimer(tick):
    // 1. 累加期望时间
    timerBalance += 50  // 每个 TelemetryTick 对应约 50ms 的游戏时间

    // 2. 获取真实时间基准
    // gRPC HTTP/2 PING 帧可测量连接级 RTT
    // 或者使用 Go time.Now() 的系统时钟
    nowMs = currentTimeMillis()

    // 3. 判定
    IF timerBalance > nowMs + clockDrift:
        // 累计时间超过真实时间 + 容忍值 → 包过快
        timerVL += 1.0
        timerBalance -= 50  // 回退一个 tick，防止累积

        IF timerVL > 3.0:
            FLAG("TIMER", timerBalance - nowMs)
    ELSE:
        timerVL = max(0, timerVL - 0.05)

    // 4. 防止 balance 累积过大（ping 高时）
    maxBalance = lastKnownClock + lastRTT + clockDrift
    timerBalance = min(timerBalance, maxBalance)
```

**关键参数**: clockDrift 默认 120ms（与 GrimAC 一致），容忍网络波动。

#### 3.3.2 X-Ray — 矿石分布统计

```
STATE:
    breakHistory: map[material]int  // 滑动窗口内的方块统计
    totalBreaks: int
    sessionOreRatios: []float       // 分段矿石比例

PROCEDURE detectXRay(event):
    material = event.material
    breakHistory[material]++
    totalBreaks++

    // 每 100 次挖掘计算一次矿石比例
    IF totalBreaks % 100 == 0:
        diamondCount = breakHistory.get("DIAMOND_ORE", 0)
                    + breakHistory.get("DEEPSLATE_DIAMOND_ORE", 0)
        emeraldCount = breakHistory.get("EMERALD_ORE", 0)
                    + breakHistory.get("DEEPSLATE_EMERALD_ORE", 0)
        ancientCount = breakHistory.get("ANCIENT_DEBRIS", 0)

        diamondRatio = diamondCount / totalBreaks
        emeraldRatio = emeraldCount / totalBreaks
        ancientRatio = ancientCount / totalBreaks

        // 阈值参考：正常玩家钻石比例约 1-3%，X-Ray 可达 10%+
        suspiciousScore = 0
        IF diamondRatio > 0.08: suspiciousScore += 3
        IF emeraldRatio > 0.06: suspiciousScore += 2
        IF ancientRatio > 0.03: suspiciousScore += 4

        // 深度分布检查
        // 正常玩家在 Y=-59 附近的钻石比例应与整体一致
        // X-Ray 玩家集中在特定深度

        IF suspiciousScore >= 3:
            xrayVL += suspiciousScore
            IF xrayVL > 5.0:
                FLAG("XRAY", diamondRatio, emeraldRatio, ancientRatio)

        sessionOreRatios.append(diamondRatio)
        breakHistory.clear()
        totalBreaks = 0
```

#### 3.3.3 灵敏度估算 — GCD 众数法

参考 GrimAC `AimProcessor.java`，纯数学实现。

```
STATE:
    yawDeltas: []float    // 滑动窗口（最多 80）
    pitchDeltas: []float
    sensitivityEstimate: float = -1  // -1 表示未确定

PROCEDURE estimateSensitivity(tick):
    IF NOT hasPreviousTick: RETURN

    Δyaw = |tick.yaw - prev.yaw|
    Δpitch = |tick.pitch - prev.pitch|

    // 跳过微小变化（静止状态）
    IF Δyaw < 0.01 AND Δpitch < 0.01: RETURN

    IF Δyaw > 0.01:
        yawDeltas.append(Δyaw)
    IF Δpitch > 0.01:
        pitchDeltas.append(Δpitch)

    // 维护窗口大小
    IF len(yawDeltas) > 80: yawDeltas.pop(0)
    IF len(pitchDeltas) > 80: pitchDeltas.pop(0)

    // 需要至少 15 个样本
    IF len(pitchDeltas) < 15: RETURN

    // GCD 计算
    divisor = pitchDeltas[0]
    FOR i = 1 TO len(pitchDeltas) - 1:
        divisor = gcd(divisor, pitchDeltas[i])
        IF divisor < 0.001: RETURN  // 精度不足

    // 统计众数（对 divisor 量化后的分布）
    // 灵敏度转换公式（MC yaw/pitch → 灵敏度值）
    // sensitivity = (cbrt(divisor / 0.15 / 8.0) - 0.2) / 0.6
    IF divisor > 0.001:
        sensitivity = (cbrt(divisor / 0.15 / 8.0) - 0.2) / 0.6
        sensitivity = clamp(sensitivity, 0, 1)

        // 灵敏度一致性检查
        IF sensitivityEstimate > 0:
            IF |sensitivity - sensitivityEstimate| > 0.15:
                // 灵敏度突变 → Aimbot 或鼠标宏特征
                aimbotVL += 1.0
        sensitivityEstimate = sensitivity
```

### 3.4 通用基础设施

#### 3.4.1 VL 衰减系统

参考 GrimAC 的 VL + decay + Setback 框架，构建通用违规累积系统。

```
STRUCT VLTracker:
    violations: float = 0
    decayRate: float          // 每次正确行为衰减量
    setbackThreshold: float   // 触发惩罚的阈值
    maxViolations: float = 100

    PROCEDURE flag(amount):
        violations = min(violations + amount, maxViolations)
        IF violations > setbackThreshold:
            TRIGGER_PENALTY()

    PROCEDURE reward():
        violations = max(0, violations - decayRate)

    PROCEDURE shouldFlag():
        RETURN violations > setbackThreshold

// 各检测的 decay 配置
speedVL = VLTracker(decay=0.5, setback=5.0)
reachVL = VLTracker(decay=0.25, setback=3.0)
flyVL = VLTracker(decay=0.3, setback=8.0)
killAuraVL = VLTracker(decay=0.1, setback=5.0)
timerVL = VLTracker(decay=0.05, setback=3.0)
xrayVL = VLTracker(decay=0.2, setback=5.0)
```

#### 3.4.2 滑动窗口 + 环形缓冲区

参考 Matrix `2d_0` + `1g_0` 的通用频率分析基础设施。

```
STRUCT RingBuffer[T]:
    data: []T
    capacity: int
    head: int = 0
    size: int = 0

    PROCEDURE push(item):
        data[head] = item
        head = (head + 1) % capacity
        size = min(size + 1, capacity)

    PROCEDURE items():
        // 返回按时间顺序的元素
        RETURN data 的有序视图

STRUCT SlidingWindow:
    buffer: RingBuffer[TimestampedEvent]
    windowMs: int

    PROCEDURE add(event):
        buffer.push(event)
        // 清理过期事件
        cutoff = now() - windowMs
        WHILE buffer.size > 0 AND buffer.oldest().timestamp < cutoff:
            buffer.popOldest()
```

---

## 4. 外部进程架构适配

### 4.1 数据获取能力矩阵

基于 Proto 契约 `matrix_telemetry.proto` 的 18 种事件类型，分析每个检测所需数据的可获取性。

| 检测模块 | 所需数据 | Proto 事件 | 字段完整度 | 状态 |
|----------|---------|-----------|-----------|------|
| Speed | 位置 + 速度 + 状态 | TelemetryTick | 完整 | 可实现 |
| Reach | 攻击者/目标位置 + 朝向 | EntityDamageEvent | 完整 | 可实现 |
| Timer | 时间序列 + RTT | TelemetryTick + 连接级 | 完整 | 可实现 |
| Aimbot | 朝向序列 | TelemetryTick + EntityDamageEvent | 完整 | 可实现 |
| X-Ray | 方块类型 + 坐标 | BlockBreakEvent | 完整 | 可实现 |
| AutoClicker | 攻击时间序列 | EntityDamageEvent | 完整 | 可实现 |
| Fly | Y 坐标 + 地面状态 | TelemetryTick | 完整 | 可实现 |
| KillAura | 攻击目标序列 | EntityDamageEvent | **缺少 entity_id** | 需增强 |
| FastBreak | 方块材质 + 工具 | BlockBreakEvent | 完整 | 可实现 |
| NoFall | Y 坐标 + 地面状态 | TelemetryTick | **无法验证碰撞** | 混合 |
| Phase | 位置突变 + 碰撞 | TelemetryTick | **无碰撞数据** | 粗粒度 |

### 4.2 延迟补偿策略

外部进程的延迟链路：

```
MC 事件发生 → Bukkit Event → gRPC onNext → 网络传输 → Go Recv
    → session.Send() → Redis RPUSH → 500ms ticker → LPOP
    → 反序列化 → Sub.Process()

总延迟: 50-200ms（网络） + 0-500ms（Redis 缓冲）= 50-700ms
```

**应对策略**:

1. **持续性作弊容忍延迟**: Speed/Fly/X-Ray 等持续性作弊不会在 700ms 内停止，延迟不影响检测效果

2. **瞬时作弊依赖 VL 累积**: Reach/KillAura/AutoClicker 的单次事件可能因延迟导致时间戳不精确，但通过 VL 累积 + 多次确认可以补偿

3. **Timer 检测天然抗延迟**: 事务时钟法以 RTT 为基准，网络延迟自动被 clockDrift 容忍

4. **位置快照机制**: 对于 Reach 检测，建议 MC 端在攻击事件中记录 `tickTime`（已有字段），Go 端据此关联最近的 TelemetryTick

### 4.3 与 MC 插件端的协作分工

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: 实时拦截层（MC 插件端，建议未来实现）                     │
│                                                                   │
│   → 包格式验证（NaN/Infinity/极端值）                             │
│   → 明显违规直接拒绝（位置跳变 > 10 格）                          │
│   → Setback 执行（回弹到合法位置）                                │
│   → 数据增强（攻击上下文保存、entity_id 上报）                    │
│                                                                   │
│   说明: 这些操作需要 Bukkit API 和包级访问，只能在 Java 层完成    │
├─────────────────────────────────────────────────────────────────┤
│ Layer 2: 延迟判定层（Go 端，50-700ms 延迟）                      │
│                                                                   │
│   → Raycast Reach（攻击事件到达后判定）                           │
│   → Timer 事务时钟法                                              │
│   → 灵敏度估算                                                   │
│   → 动态 Speed 阈值                                              │
│   → Fly 状态机                                                   │
│                                                                   │
│   说明: 这些检测容忍延迟，精度由 VL 累积保证                      │
├─────────────────────────────────────────────────────────────────┤
│ Layer 3: 统计分析层（Go 端，累积分析）                            │
│                                                                   │
│   → VL 衰减系统                                                  │
│   → 多目标切换频率统计                                            │
│   → 点击间隔分布分析                                              │
│   → X-Ray 矿石比例统计                                           │
│   → FastBreak 时间窗口统计                                       │
│                                                                   │
│   说明: 需要大量样本才能判定，天然适合延迟架构                    │
├─────────────────────────────────────────────────────────────────┤
│ Layer 4: 离线分析层（Go 端，事后审计）                            │
│                                                                   │
│   → 跨会话行为追踪                                               │
│   → 历史数据比对                                                 │
│   → 矿石发现率的长期趋势分析                                     │
│   → 风险评分的跨会话累积                                         │
│                                                                   │
│   说明: 基于数据库中的 warning/statistic 数据做离线分析           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. 实现路线图

### Phase 1: 基础升级（2-3 周）

**目标**: 升级已有 Speed 和 Reach 检测，建立基础设施。

| 任务 | 工作量 | 说明 |
|------|--------|------|
| VL 衰减系统 | 2d | 实现 `VLTracker` 通用组件，替换当前固定 riskScore |
| Speed 动态阈值 | 1d | 改造 `AntiCheatSub`，根据 is_sprinting/is_sneaking/active_effects 调整阈值 |
| Raycast Reach | 3d | 实现 Ray-AABB 碰撞检测，替换欧氏距离 |
| 滑动窗口组件 | 1d | 实现 `RingBuffer` + `SlidingWindow` 通用组件 |
| Proto 增强 | 0.5d | EntityDamageEvent 增加 `entity_id` 字段 |

**交付物**:
- 通用 VL 衰减系统
- Speed 动态阈值（不再固定 12.0）
- Reach Raycast 检测（不再欧氏距离）
- 滑动窗口基础设施

### Phase 2: 增强检测（3-4 周）

**目标**: 新增 Fly、KillAura、X-Ray 检测。

| 任务 | 工作量 | 说明 |
|------|--------|------|
| Timer 检测 | 2d | 事务时钟法 + gRPC RTT 测量 |
| Fly 状态机 | 3d | 多维度状态机（ΔY/地面状态/连续性） |
| X-Ray 矿石统计 | 2d | 矿石比例 + 深度分布 + 滑动窗口 |
| KillAura 多目标分析 | 3d | 切换频率 + 方向一致性 |
| 新增 AntiCheat Sub 拆分 | 2d | 将 AntiCheatSub 拆分为独立 Sub（SpeedSub, ReachSub, TimerSub 等） |

**交付物**:
- Timer 检测（全新）
- Fly 检测（全新）
- X-Ray 检测（全新）
- KillAura 检测（全新）

### Phase 3: 高级检测（3-4 周）

**目标**: 新增 Aimbot、AutoClicker、FastBreak 检测。

| 任务 | 工作量 | 说明 |
|------|--------|------|
| 灵敏度估算 | 2d | GCD 众数法 + 灵敏度一致性检查 |
| AutoClicker 检测 | 2d | CPS 分析 + 间隔分布统计 |
| FastBreak 检测 | 1d | 滑动窗口 + 材质预期时间表 |
| NoFall 检测 | 2d | ground_spoof 简化版（粗粒度） |
| 管理后台 | 3d | 警告列表增强、检测参数配置 API |

**交付物**:
- Aimbot 检测（灵敏度估算）
- AutoClicker 检测
- FastBreak 检测
- NoFall 检测（简化版）

### Phase 4: 远期规划（持续迭代）

**目标**: 机器学习、预测引擎概念移植、跨会话分析。

| 任务 | 工作量 | 说明 |
|------|--------|------|
| 行为基线建模 | 4w | 为每个玩家建立行为基线，偏离基线即异常 |
| LSTM Aimbot 检测 | 4w | 基于朝向时序数据训练 LSTM 模型 |
| 攻击排队机制 | 2w | MC 端保存攻击上下文，Go 端延迟判定 |
| 跨会话追踪 | 2w | 基于 UUID 的跨会话风险评分累积 |
| 实时告警 | 1w | 高风险玩家通过 WebSocket/SSE 实时通知管理员 |

---

## 6. 风险与局限

### 6.1 外部架构的检测盲区

| 盲区 | 原因 | 影响 | 缓解措施 |
|------|------|------|---------|
| **预测引擎不可移植** | 需要精确碰撞箱和 NMS 访问 | Speed/Fly 检测精度无法达到 GrimAC 的 ~1mm 级别 | 动态阈值 + VL 累积补偿，精度约 0.1 blocks |
| **采样频率受限** | TelemetryTick 间隔 1-2 秒 | 短时作弊（<1 秒的 Speed/Fly）可能漏检 | Timer 检测可覆盖部分场景 |
| **无包级访问** | 外部进程无法拦截/修改客户端包 | 无法做 Setback（回弹玩家位置）| 仅记录警告，由管理员或 MC 插件端执行惩罚 |
| **碰撞箱不可查** | 需 NMS 级别世界数据访问 | Phase/GroundSpoof 等需要碰撞数据的检测不可实现 | 做粗粒度突变检测，精确检测留给 MC 插件端 |
| **实体插值不可追踪** | 需实时追踪所有实体移动包 | Reach 检测无法考虑目标实体的插值位置 | 使用碰撞箱扩展（+0.1）作为补偿 |
| **实时响应受限** | 50-700ms 延迟 | 对 Scaffold 等需要即时响应的作弊无法实时拦截 | 定位为"事后检测 + 证据收集"，不追求实时拦截 |

### 6.2 误报风险

| 检测模块 | 误报场景 | 缓解措施 |
|----------|---------|---------|
| Speed | 疾跑 + 跳跃 + 药水叠加 | 动态阈值覆盖所有状态组合 |
| Reach | 高 ping 玩家（位置延迟） | 碰撞箱扩展 + VL 累积（不因单次超限 flag） |
| Fly | Velocity（被击飞）/ 活塞推动 / 气泡柱 | 排除 Velocity 事件后的窗口期 |
| KillAura | 多人战斗（PvP 混战） | 要求足够样本量（5 秒窗口内 >5 次攻击） |
| X-Ray | 运气好的玩家 | 滑动窗口 100 次挖掘，需要持续偏高才 flag |
| Timer | gRPC 重连期间 | 重连后重置 timerBalance |
| NoFall | 梯子/水/蜘蛛网等特殊场景 | 结合 is_climbing/is_in_water 字段排除 |

### 6.3 性能预估

以 100 个在线玩家为基准：

| 检测模块 | 每次处理耗时 | 频率 | 100 人总开销 |
|----------|-------------|------|-------------|
| Speed | ~0.01ms | 1-2 次/秒/人 | ~2ms/秒 |
| Reach | ~0.05ms（Raycast） | 按攻击频率 | ~5ms/秒 |
| Timer | ~0.01ms | 1-2 次/秒/人 | ~2ms/秒 |
| Fly | ~0.02ms | 1-2 次/秒/人 | ~4ms/秒 |
| X-Ray | ~0.01ms | 按挖掘频率 | ~1ms/秒 |
| AutoClicker | ~0.05ms（统计） | 按攻击频率 | ~5ms/秒 |
| KillAura | ~0.03ms | 按攻击频率 | ~3ms/秒 |
| **总计** | | | **~22ms/秒** |

对比：GrimAC 预测引擎单 tick 约 0.3ms/人，20 tick/秒 = 6ms/秒/人，100 人 = 600ms/秒。本方案的总开销约 GrimAC 的 3.7%，且完全在独立进程中运行。

---

## 7. 附录

### 7.1 检测模块适用性汇总

| 模块 | 适用性 | Proto 契约支撑 | 实现阶段 |
|------|--------|:-------------:|:--------:|
| Speed（动态阈值） | 外部可实现 | TelemetryTick | Phase 1 |
| Reach（Raycast） | 外部可实现 | EntityDamageEvent | Phase 1 |
| Timer（事务时钟法） | 外部可实现 | TelemetryTick + gRPC RTT | Phase 2 |
| Fly（状态机） | 外部可实现 | TelemetryTick | Phase 2 |
| X-Ray（矿石统计） | 外部可实现 | BlockBreakEvent | Phase 2 |
| Aimbot（灵敏度估算） | 外部可实现 | TelemetryTick + EntityDamageEvent | Phase 3 |
| AutoClicker（CPS 分析） | 外部可实现 | EntityDamageEvent | Phase 3 |
| FastBreak（滑动窗口） | 外部可实现 | BlockBreakEvent | Phase 3 |
| KillAura（多目标分析） | 混合可实现 | EntityDamageEvent（需增强 entity_id） | Phase 2 |
| NoFall（地面欺骗） | 混合可实现 | TelemetryTick（无法验证碰撞） | Phase 3 |
| Phase（穿墙检测） | 内部可实现 | 需 NMS 碰撞箱 | 远期 |

**"外部可实现"标签数量: 8 个**（Speed / Reach / Timer / Fly / X-Ray / Aimbot / AutoClicker / FastBreak）

### 7.2 Proto 契约与检测模块映射

```
Proto 事件类型                 支撑的检测模块
─────────────────────────────────────────────────
TelemetryTick (13)           → Speed, Fly, Timer, Aimbot, NoFall
EntityDamageEvent (17)       → Reach, KillAura, AutoClicker, Aimbot
BlockBreakEvent (14)         → X-Ray, FastBreak
PlayerTeleportEvent (26)     → 所有检测（位置重置）
PlayerRespawnEvent (27)      → 所有检测（位置重置）
GameModeChangeEvent (28)     → 所有检测（模式重置）
gRPC 连接级 RTT              → Timer
```

### 7.3 关键参考资料

| 文档 | 路径 | 内容 |
|------|------|------|
| Matrix 反编译报告 | `docs/matrix/matrix-decompile-analysis.md` | Matrix 7.19.4 全部检测模块分析 |
| GrimAC 源码分析 | `docs/matrix/grimac-source-analysis.md` | 预测引擎 + 战斗检测算法分析 |
| T12 对比分析 | `.omo/evidence/task-12-comparison.md` | 5 维度源码级对比 |
| Matrix 架构 | `docs/matrix/architecture.md` | Go 后端架构、数据流、Sub 接口 |
| Java 对接教程 | `docs/matrix/java-integration.md` | 18 种事件类型、字段定义 |
| 反作弊生态调研 | `docs/matrix/anti-cheat-research.md` | 生态全景、架构模式对比 |
| Proto 契约 | `proto/matrix/v1/matrix_telemetry.proto` | 18 种事件的消息定义 |

### 7.4 术语表

| 术语 | 含义 |
|------|------|
| VL | Violation Level，违规等级。累积到阈值触发惩罚 |
| Setback | 回弹玩家到合法位置的惩罚机制 |
| Raycast | 射线投射，从视点发射射线检测碰撞 |
| AABB | Axis-Aligned Bounding Box，轴对齐包围盒 |
| GCD | Greatest Common Divisor，最大公约数 |
| CPS | Clicks Per Second，每秒点击次数 |
| RTT | Round-Trip Time，网络往返延迟 |
| NMS | net.minecraft.server，MC 服务端内部代码 |
| TelemetryTick | 每 1-2 秒采集的玩家完整状态快照 |
| Proto 契约 | Protobuf 定义的事件消息格式 |
| Sub | MatrixSub 接口，插件式检测模块注册 |
| clockDrift | 时钟漂移容忍值，Timer 检测的容错参数 |
| cancelBuffer | GrimAC 的激进检测缓冲区，触发后逐步衰减 |
| FirstBread | "面包屑"标记，用于追踪击退/爆炸是否被确认 |
| 预测引擎 | GrimAC 的核心，暴力枚举输入模拟客户端物理 |

---

*方案说明书生成时间: 2026-05-27*
*基于: T10 Matrix 反编译报告 + T11 GrimAC 源码分析 + T12 对比分析*
*分析师: 筱锋*
