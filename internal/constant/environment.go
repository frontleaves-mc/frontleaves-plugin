package bConst

import xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"

const (
	EnvGrpcSecretKey     xEnv.EnvKey = "YGGLEAF_GRPC_SECRET_KEY"      // gRPC 服务间调用的共享密钥
	EnvYggleafGrpcHost   xEnv.EnvKey = "YGGLEAF_GRPC_HOST"            // Yggleaf gRPC 服务地址
	EnvYggleafGrpcPort   xEnv.EnvKey = "YGGLEAF_GRPC_PORT"            // Yggleaf gRPC 服务端口

	// gRPC Keepalive
	EnvGrpcKeepaliveTime    xEnv.EnvKey = "GRPC_KEEPALIVE_TIME"    // gRPC 服务端 keepalive PING 间隔，默认值为 30s，格式为 Go duration string
	EnvGrpcKeepaliveTimeout xEnv.EnvKey = "GRPC_KEEPALIVE_TIMEOUT" // gRPC PING 响应超时时间，默认值为 10s，格式为 Go duration string
	EnvGrpcKeepaliveMinTime xEnv.EnvKey = "GRPC_KEEPALIVE_MIN_TIME" // gRPC 允许的客户端最小 PING 间隔，默认值为 5s，格式为 Go duration string
	EnvGrpcKeepaliveMaxIdle xEnv.EnvKey = "GRPC_KEEPALIVE_MAX_IDLE" // gRPC 连接最长空闲时间，默认值为 5m，格式为 Go duration string

	EnvMatrixManageInterval xEnv.EnvKey = "MATRIX_MANAGE_INTERVAL"    // Matrix 管理任务执行间隔，默认值为 5s，格式为 Go duration string（如 "5s"、"2s"、"500ms"）

	// Matrix Anti-Cheat
	EnvMatrixAcEnabled                  xEnv.EnvKey = "MATRIX_AC_ENABLED"                    // 是否启用 Matrix 反作弊检测，默认为 true
	EnvMatrixAcSpeedBaseSpeed           xEnv.EnvKey = "MATRIX_AC_SPEED_BASE_SPEED"           // 基础移动速度阈值（blocks/tick），默认为 4.317
	EnvMatrixAcSpeedTolerance           xEnv.EnvKey = "MATRIX_AC_SPEED_TOLERANCE"           // 速度容忍度浮动范围，默认为 0.15
	EnvMatrixAcSpeedVlThreshold         xEnv.EnvKey = "MATRIX_AC_SPEED_VL_THRESHOLD"         // 速度违规 Violation Level 阈值，默认为 5.0
	EnvMatrixAcReachMaxReach            xEnv.EnvKey = "MATRIX_AC_REACH_MAX_REACH"            // 最大攻击距离（blocks），默认为 3.0
	EnvMatrixAcReachExpand              xEnv.EnvKey = "MATRIX_AC_REACH_EXPAND"              // 扩展攻击距离（blocks），默认为 0.1
	EnvMatrixAcReachVlThreshold         xEnv.EnvKey = "MATRIX_AC_REACH_VL_THRESHOLD"         // 攻击距离违规 Violation Level 阈值，默认为 3.0
	EnvMatrixAcTimerClockDrift          xEnv.EnvKey = "MATRIX_AC_TIMER_CLOCK_DRIFT"          // 时钟漂移容忍度（ms），默认为 150
	EnvMatrixAcTimerVlThreshold         xEnv.EnvKey = "MATRIX_AC_TIMER_VL_THRESHOLD"         // Timer 违规 Violation Level 阈值，默认为 5.0
	EnvMatrixAcFlyVlThreshold           xEnv.EnvKey = "MATRIX_AC_FLY_VL_THRESHOLD"           // 飞行违规 Violation Level 阈值，默认为 5.0
	EnvMatrixAcFlyHoverThreshold        xEnv.EnvKey = "MATRIX_AC_FLY_HOVER_THRESHOLD"        // 悬停违规判定阈值（ticks），默认为 40
	EnvMatrixAcXrayWindowSize           xEnv.EnvKey = "MATRIX_AC_XRAY_WINDOW_SIZE"           // Xray 检测窗口大小（样本数），默认为 100
	EnvMatrixAcXrayDiamondRatio         xEnv.EnvKey = "MATRIX_AC_XRAY_DIAMOND_RATIO"         // 钻石采集异常阈值，默认为 0.05
	EnvMatrixAcXrayEmeraldRatio         xEnv.EnvKey = "MATRIX_AC_XRAY_EMERALD_RATIO"         // 绿宝石采集异常阈值，默认为 0.03
	EnvMatrixAcXrayAncientRatio         xEnv.EnvKey = "MATRIX_AC_XRAY_ANCIENT_RATIO"         // 远古遗迹采集异常阈值，默认为 0.01
	EnvMatrixAcKillauraWindowMs         xEnv.EnvKey = "MATRIX_AC_KILLAURA_WINDOW_MS"         // KillAura 检测时间窗口（ms），默认为 3000
	EnvMatrixAcKillauraMaxSwitch        xEnv.EnvKey = "MATRIX_AC_KILLAURA_MAX_SWITCH"        // Killaura 最大切换目标次数，默认为 3
	EnvMatrixAcKillauraAngleThreshold   xEnv.EnvKey = "MATRIX_AC_KILLAURA_ANGLE_THRESHOLD"   // Killaura 角度异常阈值（度），默认为 90.0
	EnvMatrixAcAimbotSampleWindow       xEnv.EnvKey = "MATRIX_AC_AIMBOT_SAMPLE_WINDOW"       // Aimbot 采样窗口大小（样本数），默认为 30
	EnvMatrixAcAimbotSensitivityDelta   xEnv.EnvKey = "MATRIX_AC_AIMBOT_SENSITIVITY_DELTA"   // Aimbot 灵敏度变化阈值，默认为 0.15
	EnvMatrixAcAutoclickerMaxCps        xEnv.EnvKey = "MATRIX_AC_AUTOCLICKER_MAX_CPS"        // AutoClicker 最大 CPS 阈值，默认为 16
	EnvMatrixAcAutoclickerWindowMs      xEnv.EnvKey = "MATRIX_AC_AUTOCLICKER_WINDOW_MS"      // AutoClicker 检测时间窗口（ms），默认为 1000
	EnvMatrixAcAutoclickerMinStddev     xEnv.EnvKey = "MATRIX_AC_AUTOCLICKER_MIN_STDDEV"     // AutoClicker 最小标准差阈值，默认为 15.0
	EnvMatrixAcFastbreakWindowSize      xEnv.EnvKey = "MATRIX_AC_FASTBREAK_WINDOW_SIZE"      // Fastbreak 检测窗口大小（样本数），默认为 20
	EnvMatrixAcFastbreakRatioThreshold  xEnv.EnvKey = "MATRIX_AC_FASTBREAK_RATIO_THRESHOLD"  // Fastbreak 破坏比例阈值，默认为 0.5
	EnvMatrixAcNofallFallSpeed          xEnv.EnvKey = "MATRIX_AC_NOFALL_FALL_SPEED"          // NoFall 下落速度阈值（blocks/tick），默认为 0.1
	EnvMatrixAcNofallConsecutiveTicks   xEnv.EnvKey = "MATRIX_AC_NOFALL_CONSECUTIVE_TICKS"   // NoFall 连续无伤害下落判定阈值（ticks），默认为 3

	// 邮件服务
	EnvEmailHost     xEnv.EnvKey = "EMAIL_HOST"                       // SMTP 服务器地址
	EnvEmailPort     xEnv.EnvKey = "EMAIL_PORT"                       // SMTP 服务器端口
	EnvEmailUser     xEnv.EnvKey = "EMAIL_USERNAME"                   // SMTP 登录用户名
	EnvEmailPass     xEnv.EnvKey = "EMAIL_PASSWORD"                   // SMTP 登录密码
	EnvEmailFromName xEnv.EnvKey = "EMAIL_FROM_NAME"                  // 邮件发件人名称
	EnvEmailFrom     xEnv.EnvKey = "EMAIL_FROM_ADDRESS"               // 邮件发件人地址
)
