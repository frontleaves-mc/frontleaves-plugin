# TPS 下降对外部反作弊检测精度影响分析

> **文档类型**: 理论验证分析  
> **日期**: 2026-05-27  
> **状态**: 已完成  
> **目标读者**: 未来维护者、决策者  
> **前置文档**: [anti-cheat-research.md](./anti-cheat-research.md)  
> **关联代码**: `internal/logic/matrix/sub_anti_cheat.go`, `internal/logic/matrix/player_session.go`

---

## 目录

1. [问题描述](#1-问题描述)
2. [内部检测的 TPS 困境](#2-内部检测的-tps-困境)
3. [外部检测的 TPS 优势分析](#3-外部检测的-tps-优势分析)
4. [量化分析](#4-量化分析)
5. [与 GrimAC 的对比](#5-与-grimac-的对比)
6. [结论](#6-结论)
7. [局限性声明](#7-局限性声明)

---

## 1. 问题描述

Minecraft 服务端以 TPS（Ticks Per Second）作为核心时序基准，标准值为 20 TPS，即每 Tick 固定 50ms。当服务器负载升高时，TPS 下降，实际 Tick 间隔拉长。

反作弊系统的检测精度与时序基准直接耦合。问题核心在于：

- **内部检测**（NoCheatPlus、Vulcan 等纯插件方案）运行在 JVM 内部，检测引擎与游戏逻辑共享 CPU，TPS 下降时双方互相拖累，检测精度随之劣化。
- **外部检测**（本项目 Matrix）运行在独立 Go 进程中，使用操作系统系统时钟而非游戏 Tick。TPS 下降时，检测进程的计算资源和时序基准不受影响。

本文从理论层面量化分析 TPS 下降对两类架构检测精度的影响差异，验证 Matrix 外部架构在 TPS 劣化场景下的天然优势。

---

## 2. 内部检测的 TPS 困境

### 2.1 CPU 竞争与精度下降

内部反作弊插件运行在 MC 服务端的 Bukkit 线程（或异步线程池）上，与游戏主循环共享 JVM 堆和 CPU 时间片。当 TPS 已经下降时，反作弊检测逻辑进一步加剧 CPU 竞争，形成恶性循环：

```
TPS 下降 → 游戏主线程变慢 → 反作弊检测排队等待 → 检测延迟增大 → 检测精度下降
                                                    ↓
                                               CPU 开销增加 → TPS 进一步下降
```

### 2.2 移动数据包间隔变长

在标准 TPS=20 下，客户端每 50ms 发送一次移动数据包。TPS 下降时，服务端处理能力跟不上，移动数据包的实际接收间隔变大：

| 场景 | 理论 Tick 间隔 | 实际数据包间隔（典型值） |
|---|---|---|
| TPS = 20 | 50ms | ~50ms |
| TPS = 10 | 100ms | ~100ms |
| TPS = 5 | 200ms | ~200ms+ |

### 2.3 典型表现

内部检测引擎基于 Tick 时序构建预测模型，假设每个 Tick 固定 50ms。当实际间隔偏离时：

1. **误报激增**：预测引擎基于 50ms 间隔计算预期位移，但玩家在拉长的间隔内正常移动了更远距离，被错误判定为加速。
2. **检测失效**：部分系统（如 NCP 的 Tasklag 补偿）在检测到 TPS 异常后自动放宽阈值，导致真正作弊者逃过检测。

这两个结果互相矛盾，却同时发生：正常玩家被误判，作弊玩家被放过。

---

## 3. 外部检测的 TPS 优势分析

### 3.1 独立计算环境

Matrix 的 Go 检测进程运行在完全独立的进程中，与 MC 服务端无共享资源：

- **CPU**：Go 进程独占 CPU 核心，不受 JVM GC 停顿和线程调度影响
- **内存**：独立堆空间，不受 Minecraft 内存压力影响
- **时钟**：使用 `time.Ticker` 系统时钟，manageInterval 精确为 500ms，与 TPS 无关

### 3.2 检测阈值不变

Matrix 的 Speed 检测核心逻辑（`sub_anti_cheat.go`）：

```go
dt := float64(ts - s.lastTimestamp) / 1000.0  // 使用数据包自带的时间戳
speed := distance / dt                         // speed = Δdistance / Δtime
if speed > speedThreshold {                    // speedThreshold = 12.0 blocks/s
    s.triggerWarning(...)
}
```

关键点：`ts` 是 MC 插件在采集时打入的毫秒时间戳，`dt` 是两个采集点之间的真实时间差。无论 MC 服务端 TPS 如何波动，Go 端的阈值 `12.0 blocks/s` 始终不变。

### 3.3 Δtime 增大时的数学分析

Speed 检测的核心公式：

```
speed = Δdistance / Δtime
```

其中：
- `Δdistance` = √(Δx² + Δy² + Δz²)，玩家在两次采样之间的位移
- `Δtime` = 两个 TelemetryTick 之间的时间差（毫秒级时间戳差值）

**当 TPS 下降时，Δtime 增大。** 对于正常玩家和作弊者，影响如下：

**正常玩家（步行速度 ~4.3 blocks/s）**：

即使在更长的时间窗口内，正常玩家的位移也遵循物理规律。Δtime 增大时，Δdistance 按比例增大，计算出的 speed 仍然在正常范围内。不会触发误判。

**作弊者（使用 Speed hack，实际速度 > 12 blocks/s）**：

作弊者以超常速度移动。即使 Δtime 增大，Δdistance 的增长速度仍远超正常范围。但需注意一个微妙的影响：当 Δtime 过大时，短时加速作弊可能被"稀释"到更长的时间窗口中，导致计算出的平均 speed 低于阈值。

这意味着外部检测在 TPS 下降时：

- **误判概率降低**：计算速度倾向于偏低，更不容易将正常行为判定为作弊
- **漏检概率可能增大**：短时瞬移作弊可能被时间窗口稀释

这是可接受的 tradeoff。宁可漏掉一些短时作弊，也不要大规模误判正常玩家。

### 3.4 数学推导

设作弊者在 t₀ 到 t₁ 之间以速度 v_cheat 移动。MC 服务端在 t₀ 和 t₂ 时刻各上报一个 TelemetryTick（t₂ > t₁，因为 TPS 下降导致第二个 Tick 延迟）。

在 TPS=20 正常情况下，t₂ - t₀ ≈ 50ms。在 TPS=5 极端情况下，t₂ - t₀ ≈ 200ms。

```
实际作弊距离: Δd_cheat = v_cheat × (t₁ - t₀)
正常移动距离: Δd_normal = v_normal × (t₂ - t₁)
总位移: Δdistance = Δd_cheat + Δd_normal

检测速度: speed_detected = Δdistance / (t₂ - t₀)
        = [v_cheat × (t₁ - t₀) + v_normal × (t₂ - t₁)] / (t₂ - t₀)
```

当 `t₂ - t₀` 因 TPS 下降而增大时（假设作弊窗口 `t₁ - t₀` 不变）：

```
speed_detected → v_normal  （当 t₂ - t₀ >> t₁ - t₀ 时）
```

即作弊信号被稀释为正常速度，漏检概率增大。但对于持续性作弊（v_cheat 恒定），作弊者在整个时间窗口内都保持高速：

```
speed_detected = v_cheat × (t₁ - t₀) / (t₂ - t₀) + v_normal × (t₂ - t₁) / (t₂ - t₀)
```

只要 `v_cheat > speedThreshold` 且作弊持续时间占比足够高，检测仍然有效。

---

## 4. 量化分析

### 4.1 基准参数

| 参数 | 值 | 来源 |
|---|---|---|
| speedThreshold | 12.0 blocks/s | `sub_anti_cheat.go` 第 20 行 |
| manageInterval | 500ms | `player_session.go` 第 20 行 |
| popBatchSize | 100 条/批 | `player_session.go` 第 21 行 |
| MC 标准步行速度 | ~4.3 blocks/s | Minecraft Wiki |
| MC 疾跑速度 | ~5.6 blocks/s | Minecraft Wiki |

### 4.2 场景分析：正常玩家

假设玩家以疾跑速度 5.6 blocks/s 移动：

| TPS | Tick 间隔 | 两次采样间隔（Δtime） | Δdistance（疾跑） | speed 检测值 | 是否触发阈值 |
|---|---|---|---|---|---|
| 20 | 50ms | 50ms | 0.28 blocks | 5.6 blocks/s | 否 |
| 10 | 100ms | 100ms | 0.56 blocks | 5.6 blocks/s | 否 |
| 5 | 200ms | 200ms | 1.12 blocks | 5.6 blocks/s | 否 |

**结论**：正常玩家在任何 TPS 下都不会触发 Speed 检测，误判概率为零。

### 4.3 场景分析：持续性 Speed 作弊

假设作弊者以恒定 15 blocks/s 移动（Speed hack）：

| TPS | Tick 间隔 | 两次采样间隔（Δtime） | Δdistance | speed 检测值 | 是否触发阈值 (>12.0) |
|---|---|---|---|---|---|
| 20 | 50ms | 50ms | 0.75 blocks | 15.0 blocks/s | **是** |
| 10 | 100ms | 100ms | 1.50 blocks | 15.0 blocks/s | **是** |
| 5 | 200ms | 200ms | 3.00 blocks | 15.0 blocks/s | **是** |

**结论**：持续性作弊在任何 TPS 下都能被准确检测，speed 检测值恒等于实际速度，不受 TPS 影响。

### 4.4 场景分析：短时瞬移作弊

假设作弊者在 50ms 内瞬移 1.0 blocks（相当于 20 blocks/s），但正常行走其余时间：

| TPS | 采样窗口（Δtime） | 作弊占比 | 检测 speed | 是否触发阈值 |
|---|---|---|---|---|
| 20 | 50ms | 100%（整个窗口都是作弊） | 20.0 blocks/s | **是** |
| 10 | 100ms | 50%（50ms 作弊 + 50ms 正常） | ~12.15 blocks/s | **是**（边界） |
| 5 | 200ms | 25%（50ms 作弊 + 150ms 正常） | ~7.425 blocks/s | **否** |

**结论**：短时瞬移作弊在 TPS 极低时可能漏检。这是 Δtime 增大导致作弊信号稀释的直接体现。但考虑到：

1. 实际作弊者很少只做一次短时瞬移，通常是持续性操作
2. TPS=5 已属于极端异常场景，服务器本身已经无法正常运行
3. 误判的代价远高于漏检的代价（误判驱逐正常玩家 vs 漏检让个别作弊者暂时逃过）

此 tradeoff 可接受。

### 4.5 manageInterval 的影响

Matrix 的 manageLoop 以 500ms 为周期从 Redis 批量消费数据（popBatchSize=100）。这个周期与 TPS 完全无关，由 `time.Ticker` 驱动。

- TPS=20 时：每个 manage 周期内约有 10 个 TelemetryTick 被处理
- TPS=10 时：每个 manage 周期内约有 5 个 TelemetryTick 被处理
- TPS=5 时：每个 manage 周期内约有 2-3 个 TelemetryTick 被处理

处理量减少不影响检测精度，因为 Speed 检测基于相邻两个 TelemetryTick 的时间戳差值，而非 manage 周期的固定间隔。

---

## 5. 与 GrimAC 的对比

### 5.1 GrimAC 的 Ping-Pong Sandwich 方案

GrimAC 采用预测引擎（Prediction Engine），核心机制：

1. 服务端模拟完整客户端移动模型，预测玩家下一 Tick 的位置
2. 将玩家上报位置与预测位置比对，偏差超阈值即判定作弊
3. 通过 Ping-Pong 机制校准网络延迟：服务端发送 Ping，客户端回 Pong，测量真实 RTT
4. 将 RTT 纳入位置补偿，减少因网络延迟导致的误报

### 5.2 对比分析

| 维度 | GrimAC（内部预测引擎） | Matrix（外部阈值检测） |
|---|---|---|
| **TPS 耦合度** | 中等，预测模型基于 Tick 时序 | 零，完全独立进程 |
| **TPS 下降时误报** | 可能增大（预测基准偏移） | 降低（speed 计算值偏低） |
| **TPS 下降时漏检** | 可能增大（补偿机制放宽） | 可能增大（短时作弊被稀释） |
| **计算开销** | 高（需模拟 MC 物理引擎） | 低（简单距离/时间比） |
| **延迟补偿** | Ping-Pong 主动测量 RTT | 依赖数据包时间戳 |
| **检测精度** | 正常 TPS 下极高 | 满足基础检测需求 |
| **部署复杂度** | 低（拖入 plugins） | 高（需额外部署 Go 服务） |

### 5.3 Matrix 的天然优势

GrimAC 需要复杂的 Ping-Pong Sandwich 机制来应对网络延迟和 TPS 波动，本质上是在内部架构下弥补先天不足。Matrix 作为外部进程，天然具备以下优势：

1. **无需 TPS 补偿**：系统时钟与 TPS 完全解耦，不需要任何适配逻辑
2. **无需延迟补偿**：Speed 检测基于数据对的时间戳差值，不依赖 Tick 级别的实时性
3. **零性能反噬**：检测计算不会影响 TPS，不存在恶性循环
4. **阈值稳定**：12.0 blocks/s 在任何 TPS 下含义一致

GrimAC 在正常 TPS 下的检测精度确实更高（预测引擎 vs 简单阈值），但这种精度优势在 TPS 下降时会被削弱甚至逆转。Matrix 的"简单但稳健"策略在恶劣环境下反而更可靠。

---

## 6. 结论

### 6.1 核心结论

外部检测架构（Matrix）在 TPS 下降时的表现优于内部检测架构，原因如下：

1. **误判概率更低**：speed = Δdistance / Δtime 公式中，Δtime 因 TPS 下降而增大，计算出的速度偏低，倾向于"放过"而非"冤枉"
2. **检测逻辑不变**：Go 进程的计算资源、时序基准、阈值配置均不受 MC TPS 影响
3. **无恶性循环**：检测不会加剧 TPS 下降，避免内部架构的根本矛盾

### 6.2 Tradeoff 确认

漏检概率在极端低 TPS 场景下可能增大（短时瞬移作弊被时间窗口稀释），但这是可接受的：

- 误判驱逐正常玩家是不可逆的伤害（玩家流失）
- 漏检可以通过多轮检测、历史累积风险分来弥补（`riskScorePerHit=20`, `maxRiskScore=100` 的累积机制）
- TPS 极端低的场景本身就不常见，且服务器管理员通常会优先处理性能问题

### 6.3 建议

1. 保持当前 speedThreshold=12.0 不变，该阈值在 TPS 波动下表现稳定
2. 未来可考虑引入 TPS 感知机制：当检测到 MC 端 TPS 持续低于 15 时，在警告描述中标注"低 TPS 环境"供人工复核
3. Reach 检测（攻击距离）不受 Δtime 影响，完全不受 TPS 波动干扰，可优先推进实现

---

## 7. 局限性声明

1. **理论性质**：本文所有量化分析基于数学推导和理想化假设，未使用真实 MC 服务端数据进行验证。实际场景中，网络延迟抖动、数据包丢失、MC 服务端内部调度策略等因素可能导致偏差。
2. **假设条件**：分析假设 MC 插件端在 TelemetryTick 中打入的时间戳是准确的，且数据包通过 gRPC Stream 可靠传输到 Go 端。如果时间戳本身因 TPS 下降而不准确，结论需要修正。
3. **单一检测维度**：本文仅分析 Speed 检测。Reach、X-Ray 等其他检测类型受 TPS 的影响机制不同，需单独分析。
4. **manageInterval 固定性**：当前 manageInterval 硬编码为 500ms。如果未来改为动态调整，需重新评估。
5. **数据样本缺失**：缺少在真实低 TPS 环境下的对比测试数据。建议在测试环境中模拟 TPS=10 和 TPS=5 场景，收集实际检测率和误报率。

---

## 附录: 参数速查

| 参数 | 值 | 位置 |
|---|---|---|
| speedThreshold | 12.0 blocks/s | `sub_anti_cheat.go` 常量 |
| reachThreshold | 3.5 blocks | `sub_anti_cheat.go` 常量 |
| riskScorePerHit | 20 | `sub_anti_cheat.go` 常量 |
| maxRiskScore | 100 | `sub_anti_cheat.go` 常量 |
| manageInterval | 500ms | `player_session.go` 常量 |
| popBatchSize | 100 条/批 | `player_session.go` 常量 |
| inputChSize | 5000 条 | `player_session.go` 常量 |
| drainTimeout | 5s | `player_session.go` 常量 |
