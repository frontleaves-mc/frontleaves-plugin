# Matrix 7.19.4 反编译检测分析报告

> **本文补充 `docs/matrix/` 下已有文档未覆盖的反编译细节。**
> 生态对比、架构设计、Java 对接、TPS 影响等内容请分别参阅 `anti-cheat-research.md`、`architecture.md`、`java-integration.md`、`tps-impact-analysis.md`。

---

## 1 概述

| 项目 | 值 |
|------|-----|
| 目标版本 | Matrix 7.19.4 |
| JAR 大小 | 13 MB |
| 反编译工具 | CFR 0.152 |
| Java 版本 | 21.0.7 (Zulu) |
| 分析日期 | 2026-05-27 |
| 分析师 | 筱锋 |

### 1.1 分析范围

本次分析覆盖 Matrix 7.19.4 的全部检测模块，按三大领域组织：

1. **移动检测** — Speed / Fly / NoFall / Phase / NoClip / Velocity
2. **战斗检测** — Reach / KillAura / AutoClicker / Aimbot / 交叉验证
3. **辅助检测** — World / Inventory / Timer / Packet 四大类

### 1.2 混淆严重程度

Matrix 采用**工业级混淆**，评级为"严重"：

- **类名混淆**: 单字母 `a.java`、数字 `0.java`、混合 `1W.java`、`6j_0.java`
- **方法名混淆**: `g()`, `a()`, `2()`, `9()`
- **成员变量混淆**: `2_`, `21`, `2K`, `2F`, `2L`
- **字符串常量加密**: DES/CBC/PKCS5Padding，密钥基于类名动态计算
- **整数常量加密**: DES/CBC/NoPadding，ThreadLocal 密钥池 + MethodHandle 注入
- **控制流混淆**: 嵌套 try-catch、NoSuchElementException 虚假分支、LambdaMetafactory 动态调用

### 1.3 分析方法论

由于混淆严重，直接阅读反编译代码的可行性极低。本次分析采用以下策略：

1. **字符串常量驱动** — 定位 Bukkit API 调用（如 `getWalkSpeed()`, `isFlying()`）和 PacketEvents 类型名
2. **控制流模式识别** — 通过数学运算模式（距离公式、摩擦力模型）推断检测逻辑
3. **HackType 枚举映射** — `api/HackType` 未混淆，15 种检测类型为分析提供骨架
4. **数值常量提取** — 硬编码的 double 常量（0.135, 0.91, 3.1 等）未被加密
5. **交叉验证** — 多个检测器之间的共享状态验证分析推断

---

## 2 包结构与模块地图

### 2.1 包结构概览

```
decompile/matrix-src/
├── me/rerere/matrix/
│   ├── Matrix.java              # 插件主类（部分混淆）
│   ├── api/                     # API 接口层（可读）
│   │   └── HackType.java        # 15 种检测类型枚举（未混淆）
│   ├── commands/                # 命令层（可读）
│   ├── misc/                    # 工具类（可读）
│   ├── thirdparty/              # 第三方库嵌入（约 3500 类）
│   │   └── PacketEvents/        # 网络包处理框架
│   └── _/                       # 混淆包（462 类）
│       ├── 9q_0.java            # Speed/Knockback 检测器
│       ├── bg.java              # 主 Move 引擎（Fly/Speed/NoFall/Phase）
│       ├── 6D.java              # 方向性 Speed / 摩擦力验证
│       ├── 9T.java              # Velocity 纵向验证
│       ├── 6B.java              # Velocity 队列验证
│       ├── 90.java              # Phase 穿墙检测
│       ├── 6e_0.java            # Reach 射线检测
│       ├── ci.java              # Hitbox/多目标 KillAura
│       ├── 2m_0.java            # KillAura 开关映射表
│       ├── ym.java              # 主 Bukkit 事件监听器
│       ├── vd.java              # 玩家生命周期管理器
│       ├── 8v_0.java            # PacketEvents 监听注册
│       ├── hg.java              # 检测模块抽象基类
│       ├── lo.java              # per-player 数据对象
│       ├── ez.java              # 检测类别枚举
│       ├── bo.java              # Scaffold 射线追踪
│       ├── zk.java              # Scaffold 角度检测
│       ├── gk.java              # Scaffold 多模式综合
│       ├── dm.java              # Tower 搭高跳
│       ├── 9i_0.java            # FastBreak / Nuker
│       ├── 9y_0.java            # FastPlace
│       ├── 9E.java              # FastConsume
│       ├── 64.java              # FastBow
│       ├── 9L.java              # FastHeal
│       ├── 6z_0.java            # AutoInventory / FastSlot
│       ├── 9o_0.java            # 包频率 / Timer
│       ├── 9H.java              # 无效位置数据
│       ├── or.java              # 位置包有序性
│       └── ...                  # 其余约 430 个混淆类
├── summary.txt                  # CFR 汇总信息
└── aeba3600-.../                # 元数据
```

### 2.2 规模统计

| 指标 | 数量 |
|------|------|
| 总类数 | 4105 |
| 可读类数 | 3643 |
| 混淆核心类 | 462 |
| 第三方嵌入类 | ~3500 |

### 2.3 核心架构类映射

| 混淆名 | 实际职责 |
|--------|---------|
| `ym` | 主 Bukkit 事件监听器，捕获 31 种事件，分发给各检测模块 |
| `vd` | 玩家生命周期管理器，`Map<UUID, lo>` 管理 per-player 数据 |
| `8v_0` | PacketEvents 监听注册，路由网络包到检测模块 |
| `hg` | 检测模块抽象基类，实现违规标记引擎、限速日志 |
| `lo` | per-player 数据对象，持有检测实例列表和计时状态 |
| `ez` | 检测类别枚举，15 个值映射到 HackType |
| `2d_0` | 滑动窗口计时器，`1g_0<6o_0>` 环形缓冲存储包+时间戳对 |
| `1g_0` | 有界环形缓冲区，ArrayDeque 实现，固定大小 |
| `qb` | 通用限速器，基于 `System.currentTimeMillis()` 差值 |
| `rb` | AABB 碰撞箱，用于移动和战斗检测 |
| `2w_0` | 射线（Ray），用于碰撞检测 |
| `ga` | 碰撞箱射线检测，沿移动方向检测方块碰撞 |

---

## 3 检测模块清单

> 按检测类型分组。分析可信度分为高（常量/API 直接可见）、中（模式推断但阈值加密）、低（碎片化推断）三级。

| 模块名 | 检测类型 | 检测原理概述 | 关键参数/阈值 | 分析可信度 |
|--------|---------|-------------|-------------|-----------|
| **Speed** (`9q_0`) | MOVE | 摩擦力模型 + 加速度追踪 + 药水补偿，累积超速量判定 | 基础 0.135 b/tick，容差 1.4x，药水 +0.0265/level | 高 |
| **Speed/方向** (`6D`) | DELAY | 速度向量方向分析，检测加速度向量与朝向的偏角 | 加速度 ≥0.1 b/tick²，方向角 5°~40° | 高 |
| **Fly** (`bg`) | MOVE | 多维度状态机（ΔY/水平速度/地面状态），15+ 豁免场景 | VL 上限 10.0，惩罚伤害 5.0 HP | 高 |
| **NoFall** (`bg` + `ra`) | MOVE | ground_spoof 标记 + setFallDistance，嵌入 Fly 检测 | 坠落距离 = 伤害值，MC 标准 3 格起伤 | 中 |
| **Phase** (`90`) | PHASE | AABB 射线碰撞检测，沿移动路径检测不可穿透方块 | 距离上限 16.0 blocks，严重距离 0.3/0.4 | 高 |
| **NoClip** (`bg`) | PHASE | 连续 tick 水平速度恒定 >0.11 检测，排除地面/梯子/液体 | 速度阈值 0.11，连续 ≥2 tick | 中 |
| **Velocity** (`9T` + `6B`) | VELOCITY | 速度包队列追踪 + tick 窗口验证消耗 | Y 速度 0.1~0.75，验证窗口可配置 | 高 |
| **Reach** (`6e_0`) | HITBOX | 射线-包围盒碰撞检测（非欧氏距离），多眼高偏移 | 基础 3.1 blocks，创造 +3.0，快速拒绝 5.0 | 高 |
| **KillAura** (`ci`) | KILLAURA | PacketEvents 包监听 + 多目标射线分析 | 目标范围 ≤3.0 blocks，射线范围 0~10.0 | 高 |
| **AutoClicker** (`9s_0`/`6V`) | CLICK | ANIMATION 包时序分析，统计 CPS | 阈值加密，推断 >16-20 CPS | 中 |
| **Aimbot** (`ug`/`el`) | AUTOBOT | 朝向变化模式分析 + 角度偏差检测 | 角度偏差阈值推断 >3.0° | 中 |
| **FastBreak** (`9i_0`) | BLOCK | 滑动窗口跟踪 DIGGING 包时间间隔 | 窗口大小 4，时间阈值加密 | 高 |
| **FastPlace** (`9y_0` + `62`) | BLOCK | 时间窗口内放置计数 + 重置周期 | 阈值推断 >5 次/窗口 | 高 |
| **Scaffold-射线** (`bo`) | SCAFFOLD | 多角度多眼高射线追踪命中目标方块 | 眼高 {1.62, 1.54, 1.27} | 高 |
| **Scaffold-角度** (`zk`) | SCAFFOLD | 放置包 cursor 位置 + 射线交点距离检测 | 阈值 0.084 + (潜行 +0.15) | 高 |
| **Scaffold-模式** (`gk`) | SCAFFOLD | 10+ 子检测（Pitch/边缘/方向/180°旋转/高速链等） | Pitch 区间 0.77~0.8 | 中 |
| **Tower** (`dm`) | SCAFFOLD | 下方放置 + 跳跃高度精确检测 | ΔY == 0.42 ±0.001 | 高 |
| **FastConsume** (`9E`) | DELAY | 食物消耗计时比率，实际 vs 预期 | 比率 ≤0.82 | 高 |
| **FastBow** (`64`) | DELAY | 弓箭蓄力时间 vs 发射力度匹配 | 蓄力 ≤ force × 1075ms，力度 ≥0.274 | 高 |
| **FastHeal** (`9L`) | DELAY | 自然饱和回血间隔检测 | 默认 495ms（困难 3800ms） | 高 |
| **AutoInv/FastSlot** (`6z_0`) | DELAY | Shift-Click/ClickWindow/HeldItemChange 间隔 | 三合一检测，阈值加密 | 高 |
| **包频率/Timer** (`9o_0`) | DELAY | 通用包间隔计数 + 滑动窗口 | 连续 ≥3 次低于阈值 | 高 |
| **无效位置** (`9H`) | BADPACKETS | NaN/Infinity/极端值验证，重复则 kick | 直接协议验证 | 高 |
| **位置有序性** (`or`) | BADPACKETS | Δ位置分析 + Entity Action 排序验证 | Δ > 0.03 触发计数 | 高 |
| **旋转分析** (`6p_0`) | BADPACKETS | yaw/pitch GCD 一致性 + 灵敏度检查 | 数学复杂，混淆严重 | 低 |

---

## 4 移动检测深度分析

### 4.1 Speed 检测

Matrix 的 Speed 检测**远比简单阈值复杂**。它使用完整的 MC 物理模型进行判断。

#### 4.1.1 核心算法：位移增量追踪

入口文件 `9q_0.java`（811 行），监听 `PlayerMoveEvent` 和 `ENTITY_VELOCITY` 包。

**流程**:

```
PlayerMoveEvent
  ├─ 免检条件过滤（12 项排除条件）
  ├─ 摩擦力模型计算
  │   └─ friction = block_slipperiness × 0.91（地面）或 0.91（空中）
  ├─ 位移加速度 = Δcurrent − Δprevious × friction
  ├─ 速度比率判定
  │   └─ 基础速度 × 1.4 = 容差上限
  ├─ 药水效果补偿
  │   └─ Speed 药水: +0.0265 blocks/tick/level
  ├─ 累积超速量判定
  │   └─ 超速累积 > 0.5 或 (偏移比 < 1.53 且有累积)
  └─ VL 累积/衰减（0.5/tick 衰减，>2.0 触发 flag）
```

#### 4.1.2 药水效果补偿

```
基础水平速度: 0.135 blocks/tick
Speed I:      0.135 + 0.0265 × 2 = 0.188 blocks/tick
Speed II:     0.135 + 0.0265 × 3 = 0.2145 blocks/tick
```

每级 Speed 药水增加 0.0265 blocks/tick，加成基于 `amplifier + 1`。

#### 4.1.3 免检条件（12 项）

Speed 检测在以下条件之一成立时跳过：

死亡 / Slow 药水 / 坠落 > 3 格 / 飞行 / 载具中 / 睡觉 / 滑翔 / Levitation / 游泳 / 激流 / 步行速度偏差 > 0.02 / 无敌 tick 异常

#### 4.1.4 方向性 Speed 检测（`6D.java`）

独立于 `9q_0`，检测加速度向量的方向异常：

- 构建玩家碰撞箱（宽 0.8，高 1.81），检查周围无碰撞箱遮挡
- 计算加速度向量 = 当前移动 − 上帧移动 × 摩擦力
- 加速度长度 ≥ 0.1 且方向角在 5°~40° 时标记违规
- 需连续累积（每合法 tick 衰减 2），通常由受击触发检测窗口

### 4.2 Fly 检测

#### 4.2.1 核心算法：多维度状态机

入口文件 `bg.java`（1568 行），是 Matrix 最复杂的检测类。

**四个分析维度**:

1. **纵向位移** — ΔY vs 预期（重力、跳跃）
2. **水平位移** — 实际速度 vs 属性上限
3. **地面状态** — 客户端声称 onGround vs 实际碰撞检测
4. **连续性追踪** — 多 tick 状态机

**5 个检测标记**:

| 标记 | 含义 |
|------|------|
| `high_jump` | ΔY 超过跳跃高度 |
| `air_bst` | 空中上升加速度递增（违反重力） |
| `ground_spoof` | 客户端声称在地面但实际在空中 |
| `bad_jumps` | 非法跳跃模式 |
| `noclip` | 穿透检测 |

#### 4.2.2 15+ 种豁免场景

`bg.java` 中识别的合法场景标记：

`velocity`, `ladder`, `liquid`, `levitation`, `collision`, `jump`, `vine`, `bamboo_col`, `entity_collision`, `shulker`, `on_ice`, `piston`, `powder_snow`, `anvil_fix`, `farm`, `reverse_step`, `ground_pass`, `fruit_tp_fix`, `ignore_setback`

#### 4.2.3 VL 惩罚

- VL 上限: 10.0
- 惩罚: 5 HP 伤害 + 传送到地面 + 设置坠落距离
- Y < 0 时回弹到 from 位置

### 4.3 NoFall 检测

NoFall 不是独立模块，嵌入 Fly 检测的 `ground_spoof` 标记中。

**检测逻辑**: 客户端声称 onGround 且上一 tick 也声称地面，但实际 Y 位移为负、水平速度 < 0.2。

**惩罚机制**: 找到最近地面，计算高度差，设置坠落距离 = 当前 + 高度差，造成 `damage(fallDistance)` 伤害。

**地面寻找算法**: 从玩家位置向下遍历直到找到实体方块，找不到则使用 Y=-10。

### 4.4 Phase 检测

入口文件 `90.java`（678 行）。

#### 4.4.1 核心算法：碰撞箱射线检测

构建 AABB 碰撞箱（宽 ±0.3，高 1.8），沿 from→to 路径检测方块碰撞。检测到不可穿透方块时标记 Phase 违规。

#### 4.4.2 分级惩罚

| 条件 | 权重 |
|------|------|
| 距离 > 0.3 blocks | weight=4 |
| 其他 | weight=2 |

**严重违规判定**: ΔY ≥ 0.4（大幅上升）或 ΔY < -0.08 或水平位移 > 0.4 或坠落距离 > 4.0 或潜行中或 VL 过高。

#### 4.4.3 免检条件

旁观模式 / 传送/重生/世界切换冷却 / 滑翔/游泳/激流 / 非 STANDING/SNEAKING 姿势 / 梯子上 / 距离 > 16 blocks

#### 4.4.4 方块豁免

检查当前位置、下方、6 个水平方向的方块是否在配置的豁免列表中。

### 4.5 Velocity 检测

分布在两个检测类中：

- **`9T.java`**（648 行）— 纵向速度包验证：记录速度包 Y 分量，在 tick 窗口内验证玩家是否应用了纵向速度。窗口到期未消耗则标记 Anti-KB。
- **`6B.java`**（566 行）— 速度包队列验证：维护 `CopyOnWriteArrayList` 队列，收到速度包时入队，每次移动时检查是否匹配。匹配的记录被移除，超时的未匹配记录触发 flag。

---

## 5 战斗检测深度分析

### 5.1 Reach 检测

入口文件 `6e_0.java`，使用**射线-包围盒碰撞检测**而非简单欧氏距离。

#### 5.1.1 距离计算方法

1. **射线构建**: 从玩家位置构建多条射线，基于多个眼高偏移 `{1.62, 1.3, 1.54}` + 当前/历史/混合朝向
2. **包围盒碰撞**: 目标实体构建 AABB，按实体类型动态调整尺寸（玩家宽 0.41/0.31，非玩家 +0.25 padding），标准膨胀 0.001
3. **射线范围**: 0.0~10.0 blocks

#### 5.1.2 阈值体系

| 场景 | 阈值 | 说明 |
|------|------|------|
| 生存模式基础 | 3.1 blocks | 略大于 MC 标准 3.0 |
| 创造模式 | 6.1 blocks | 基础 + 3.0 offset |
| 快速拒绝 | 5.0 / 6.0 blocks | 简单欧氏距离前置检查 |
| VL 计算 | `(距离 − 阈值) × 倍率` | clamp(3, 5) |
| 惩罚模式 | setDamage(0) / 取消事件 | 由 `8b_0` 枚举控制 |

#### 5.1.3 延迟补偿

使用**位置历史队列**实现延迟补偿：

- `Queue<2B>` 保存位置历史
- 攻击时记录攻击者位置 + 目标位置到 `27` 上下文对象
- 下一个 `PlayerMoveEvent` 用最新朝向执行最终 Reach 分析
- `this.2Z` 缓冲计数器等待 2 tick 后再分析

### 5.2 KillAura 检测

核心逻辑在 `ci.java`，基于 PacketEvents 的包监听。

#### 5.2.1 检测触发

监听 `INTERACT_ENTITY` 和 `ATTACK` 包，记录每次交互的 entityId + 位置 + 动作类型。在 `PONG/WINDOW_CONFIRMATION` 包到达时批量处理积压记录。

#### 5.2.2 多目标分析

遍历 3.0 blocks 范围内的所有实体：

1. 构建每个实体的 AABB
2. 对每个实体做射线碰撞检测
3. 如果攻击目标 A，但射线与目标 B 有更近的碰撞点 → KillAura 标记

#### 5.2.3 检测条件

- 仅 SURVIVAL 和 ADVENTURE 模式
- 仅 LIVINGENTITY 和 END_CRYSTAL 目标
- 排除 NPC/假人（`2O.2()` 排除列表）
- 排除 DYING 状态实体

### 5.3 AutoClicker 检测

通过 `PlayerAnimationEvent` 中的 `ARM_SWING` 事件和 PacketEvents 的 `ANIMATION` 包监听点击。基于包到达时间间隔统计 CPS。

**推断阈值**: 正常玩家 6~12 CPS，可疑阈值推断为 >16~20 CPS。具体阈值因 DES 加密无法确认。

### 5.4 Aimbot 检测

分布在 `ug.java`（朝向向量追踪）、`el.java`（角度标志检测）、`gk.java`（极端角度检测）中。

**推断逻辑**:

1. 记录玩家 yaw/pitch 变化序列
2. 计算攻击方向与目标方向的角度偏差
3. 检查朝向变化的时间模式是否过于规律
4. 极端角度检测: Pitch > 70° 且移动速度 > 0.2

**关键碎片**: `el.java` 中 `this.7 >= 2.0 && angle` 触发，`gk.java` 中 `f > 83.0f && pitch > 70.0f` 检测。角度偏差阈值推断 >3.0°。

### 5.5 战斗检测交叉验证

Matrix 通过事件分发中心 `ym.java` 实现交叉验证：

```
ym.java（Listener）
├── EntityDamageByEntityEvent (LOW)
│   → 分发给所有 hg 子模块 → Reach 记录攻击上下文
├── PlayerAnimationEvent (HIGH)
│   → ARM_SWING → AutoClicker 检测
├── PlayerMoveEvent (LOW)
│   → 位置变化触发挂起的 Reach 分析（用最新朝向）
└── PacketReceiveEvent (via 8v_0)
    → INTERACT_ENTITY → ci.g() 记录
    → PONG → ci.1() 批量处理 KillAura 检测
```

**交叉验证机制**:

1. **位置-朝向交叉** — Reach 在攻击事件记录上下文，在移动事件中用最新朝向分析
2. **多模块协作** — 同一玩家会话持有所有检测模块实例，独立记录状态
3. **VL 独立累积** — 各检测模块独立维护 VL，通过 `ez` 枚举控制启用/禁用
4. **位置历史补偿** — Reach 使用历史队列回溯攻击时位置

---

## 6 辅助检测分析

### 6.1 World 检测

#### FastBreak / Nuker（`9i_0.java`）

滑动窗口（大小 4）跟踪 `PLAYER_DIGGING` 包时间间隔。`START_DIGGING` 到 `FINISHED_DIGGING` 的时间短于阈值时，在后续 `BlockBreakEvent` 中取消事件。

#### FastPlace（`9y_0.java` + `62.java`）

时间窗口内放置计数。超过阈值（推断 >5 次/窗口）且非 FIRE/SCAFFOLDING 材质时取消事件。

#### Scaffold — 三模块联合

Matrix 的 Scaffold 检测由三个独立模块组成：

1. **`bo.java`（射线追踪）** — 从多个眼高和角度发射射线，检查是否命中目标方块
2. **`zk.java`（角度检测）** — 基于 cursor 位置和 face 方向的射线交点距离，阈值 0.084 + 潜行 0.15
3. **`gk.java`（多模式综合，1207 行）** — 10+ 子检测：Pitch 区间、边缘检测、方向检测、180° 旋转、高速链、同材质链等

#### Tower — 搭高跳（`dm.java`）

检测玩家在下方放置方块时是否配合精确的跳跃（ΔY == 0.42 ±0.001），惩罚为传送 + 180° 旋转。

#### Nuker / 近 XRay（`wk.java`）

破坏方块时检查周围范围内不在破坏列表中的可疑方块（矿物类）。排除刷石机旁 COBBLESTONE/STONE、Silk Touch > 3。

**重要发现**: Matrix **没有独立的 X-Ray 检测模块**。不做矿石化石比例统计，这是传统反作弊的重要功能空白。

### 6.2 Inventory 检测

#### FastConsume（`9E.java`）

计算食物消耗计时比率。实际消耗时间 / 预期时间 ≤ 0.82 时标记。

#### FastBow（`64.java`）

蓄力时间与发射力度匹配检查。蓄力时间 ≤ force × 1075ms 且力度 ≥ 0.274 时标记。

#### FastHeal（`9L.java`）

自然饱和回血间隔检测。默认 495ms，困难模式 3800ms，近期受伤 -100ms。间隔低于阈值时标记并减半回血量。

#### AutoInventory / AutoClickWindow / FastSlot（`6z_0.java`）

三合一检测：快速 Shift-Click 装备、快速点击窗口、快速切换物品栏，均基于包间隔时间。

#### move.invmove（`bg.java`）

检测玩家移动时操作背包（NoSlow 变体），在移动检测中集成。

### 6.3 Timer 检测

Matrix 没有统一的 Timer 检测器，计时器检测分散在多个模块中：

| 模块 | 检测内容 | 算法 |
|------|---------|------|
| `9o_0` | 包频率/挖掘速度 | 包间隔计数，连续 ≥3 次低于阈值 |
| `65` | Plugin Message 限速 | 时间门控 |
| `9O` | 聊天频率 | 间隔 + 重复检测，7s 窗口 |
| `9L` | 回血计时 | 游戏机制时序 |
| `9k_0` | 窒息穿墙 | 时间窗口碰撞 |
| `6_` | 三叉戟飞行 | 移动计数 + 加速度 |
| `zh` | 登录限速 | ConcurrentHashMap + 时间差 |

**核心计时工具**: `2d_0` 滑动窗口（`1g_0<6o_0>` 环形缓冲区）和 `qb` 限速器提供可复用的包频率分析基础设施。

### 6.4 Packet 检测

| 模块 | 检测内容 | 算法 |
|------|---------|------|
| `9H` | 无效位置数据 | NaN/Infinity/极端值验证，重复则 kick |
| `or` | 位置包有序性 | Δ位置分析 + Entity Action 排序 |
| `ci` | 战斗交互协议 | 实体状态机 + 距离验证 |
| `zk` | 方块放置验证 | 射线相交 + 面有效性 |
| `9q_0` | 速度一致性 | 服务端速度 vs 实际移动 |
| `6p_0` | 旋转分析 | GCD + yaw/pitch 一致性（混淆严重） |

---

## 7 检测流程图

### 7.1 整体架构

```
┌─────────────────────────────────────────────────────┐
│                    事件采集层                         │
│                                                     │
│  Bukkit Events (ym.java)    PacketEvents (8v_0)     │
│  ┌──────────────────┐      ┌──────────────────┐     │
│  │ 31 种 Bukkit 事件 │      │ 网络包拦截监听    │     │
│  │ Move/Damage/     │      │ Position/Interact │     │
│  │ Break/Place/     │      │ Animation/Digging │     │
│  │ Animation/...    │      │ Velocity/Action   │     │
│  └────────┬─────────┘      └────────┬─────────┘     │
│           │                         │                │
└───────────┼─────────────────────────┼────────────────┘
            │                         │
            ▼                         ▼
┌─────────────────────────────────────────────────────┐
│                   玩家数据层                         │
│                                                     │
│  vd.java → Map<UUID, lo>                            │
│  lo = per-player 数据对象                            │
│    ├── hg[] 检测模块实例列表                         │
│    ├── Map<ez, Long> 计时状态                       │
│    └── 位置历史 / 速度包队列 / 状态追踪              │
│                                                     │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                   检测引擎层                         │
│                                                     │
│  for (hg module : lo.g().g()) {                     │
│      if (!enabled || exempted || world_blacklisted)  │
│          continue;                                  │
│      module.process(event);                         │
│  }                                                  │
│                                                     │
│  ┌─────────┐ ┌──────────┐ ┌───────────┐            │
│  │ Speed   │ │ Reach    │ │ Scaffold  │ ...        │
│  │ Fly     │ │ KillAura │ │ FastBreak │            │
│  │ Phase   │ │ Click    │ │ Timer     │            │
│  │ Velocity│ │ Aimbot   │ │ Packet    │            │
│  └────┬────┘ └────┬─────┘ └─────┬─────┘            │
│       │           │             │                    │
└───────┼───────────┼─────────────┼────────────────────┘
        │           │             │
        ▼           ▼             ▼
┌─────────────────────────────────────────────────────┐
│                   判定与处罚层                       │
│                                                     │
│  VL 累积 → 阈值判定 → 惩罚执行                      │
│                                                     │
│  惩罚方式:                                          │
│  ├── Setback（回弹到合法位置）                      │
│  ├── Cancel Event（取消违规操作）                    │
│  ├── Damage（施加伤害）                             │
│  ├── Teleport（传送到地面）                         │
│  ├── Set Fall Distance（设置坠落距离）              │
│  └── Kick（踢出服务器）                             │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 7.2 移动检测时序

```
Client              Server                    Matrix
  │                    │                         │
  ├── Position ──────→ │                         │
  │                    ├── PlayerMoveEvent ─────→│
  │                    │                         │ ├─ 免检过滤（12 项）
  │                    │                         │ ├─ 摩擦力模型计算
  │                    │                         │ ├─ Speed 加速度追踪
  │                    │                         │ ├─ Fly 状态机分析
  │                    │                         │ ├─ Phase 碰撞检测
  │                    │                         │ └─ VL 累积/判定
  │                    │                         │
  ├── Velocity ──────→ │                         │
  │   (Server→Client)  │                         │ ├─ 速度包入队 (9T/6B)
  │                    │                         │ └─ 等待移动事件验证
  │                    │                         │
  ├── Position ──────→ │                         │
  │                    ├── PlayerMoveEvent ─────→│
  │                    │                         │ └─ 验证速度包消耗
```

### 7.3 战斗检测时序

```
Client              Server                    Matrix
  │                    │                         │
  ├── ANIMATION ─────→ │                         │
  │                    ├── PlayerAnimationEvent →│ AutoClicker CPS 分析
  │                    │                         │
  ├── INTERACT ──────→ │                         │
  │   (ATTACK)         │                         │ ci.g() 记录交互
  │                    │                         │
  ├── PONG ──────────→ │                         │
  │                    │                         │ ci.1() 批量处理
  │                    │                         │ ├─ Hitbox 视线检测
  │                    │                         │ └─ 多目标 KillAura
  │                    │                         │
  │                    ├── EntityDamageByEntityEvent →
  │                    │                         │ 6e_0.8() 记录攻击上下文
  │                    │                         │ 6e_0.2() 构建 27 对象
  │                    │                         │
  ├── Position ──────→ │                         │
  │                    ├── PlayerMoveEvent ─────→│ 6e_0.g() Reach 射线分析
  │                    │                         │ └─ VL 判定 + 惩罚
```

---

## 8 反混淆与局限性说明

### 8.1 混淆技术详情

Matrix 使用多层混淆保护：

| 层级 | 技术 | 影响 |
|------|------|------|
| 命名 | 单字母/数字类名方法名 | 无法通过命名推断功能 |
| 字符串 | DES/CBC/PKCS5Padding | 日志消息、配置键名、检测类型名全部加密 |
| 整数 | DES/CBC/NoPadding + ThreadLocal | 关键阈值、倍率运行时解密 |
| 控制流 | try-catch 嵌套 + 虚假分支 | 增加逆向工程难度 |
| 动态 | LambdaMetafactory + MethodHandle | 方法调用在运行时解析 |

### 8.2 可提取的内容

- 硬编码 double 常量（0.135, 0.91, 3.1 等）
- Bukkit API 调用（`getWalkSpeed()`, `isFlying()` 等）
- PacketEvents 类型名（`INTERACT_ENTITY`, `ANIMATION` 等）
- 枚举字段名（`speed`, `height`, `margin`, `threshold` 等，来自 `b.java`）
- 算法结构（摩擦力模型、射线检测、队列验证的模式）

### 8.3 不可提取的内容

- **具体 VL 阈值** — 通过 `d("h", int, long)` 加密
- **配置参数名** — 通过 `b("q", int, long)` 加密
- **定时器/超时值** — 通过 `9T.e("y", int, long)` 加密
- **完整惩罚矩阵** — VL 累积到多少触发何种惩罚
- **多版本适配逻辑** — `y.g()` 版本判断的分支目标
- **AutoClicker CPS 阈值** — 统计逻辑可见但阈值加密

### 8.4 分析可信度总评

| 领域 | 可信度 | 说明 |
|------|--------|------|
| Speed 算法 | 高 | 数值常量直接可见，摩擦力模型逻辑清晰 |
| Fly 算法 | 高 | 多维度交叉验证，大量标记和字符串可见 |
| NoFall 算法 | 中 | 嵌入 Fly 中，逻辑可见但部分阈值加密 |
| Phase 算法 | 高 | AABB 射线检测逻辑清晰 |
| Reach 算法 | 高 | 射线-包围盒 API 调用清晰，3.1 阈值可读 |
| KillAura 算法 | 高 | PacketEvents API 调用清晰 |
| AutoClicker | 中 | ANIMATION 包监听确认，CPS 阈值加密 |
| Aimbot | 中 | 基于碎片推断，角度偏差阈值无法确认 |
| World 检测 | 高 | Scaffold/FastBreak/Tower 算法可见 |
| Timer 检测 | 高 | 分散模式清晰，具体阈值加密 |
| 交叉验证 | 高 | 事件分发架构和时序可读 |

---

## 9 与已有文档的关系声明

本文是 `docs/matrix/` 目录下的第 5 份文档，定位如下：

| 文档 | 覆盖内容 | 与本文关系 |
|------|---------|-----------|
| `anti-cheat-research.md` | 反作弊生态全景调研 | 本文不重复生态对比，聚焦 Matrix 内部实现 |
| `architecture.md` | Matrix 系统架构（Go 后端） | 本文补充 MC 端 Matrix 的检测算法细节 |
| `java-integration.md` | Java 插件对接教程 | 本文说明 Matrix 检测端的事件需求来源 |
| `tps-impact-analysis.md` | TPS 下降对检测的影响 | 本文提供 Matrix 检测的精度基准 |
| **本文** | Matrix 7.19.4 反编译检测分析 | 整合 Task 4/5/6 分析笔记，覆盖全部检测模块 |

### 关键差异对比

| 维度 | architecture.md 已知 | 本文反编译发现 |
|------|---------------------|---------------|
| Speed 算法 | 简单 `speed = Δd/Δt`, 阈值 12.0 b/s | 摩擦力模型 + 加速度追踪 + 药水补偿 + 1.4x 容差 |
| 检测维度 | 2 个 (Speed, Reach) | 25+ 种检测模块 |
| Reach 算法 | 欧氏距离 > 3.5 | 射线-包围盒碰撞检测，阈值 3.1 |
| 延迟补偿 | 无 | 位置历史队列 + tick 回溯 |
| 多目标分析 | 无 | 3.0 blocks 范围内遍历 |
| 误报规避 | 未提及 | 15+ 种豁免场景 |

---

*报告生成时间: 2026-05-27*
*分析师: 筱锋*
*数据来源: CFR 0.152 反编译输出 + 字符串常量驱动分析*
