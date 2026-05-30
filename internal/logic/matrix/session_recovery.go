package matrix

import (
	"context"
	"fmt"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/google/uuid"
)

// ServerQuerier 定义服务端状态查询能力（由 gRPC handler 层实现）
type ServerQuerier interface {
	// GetAllConnectedServers 返回所有已连接的服务器名称列表
	GetAllConnectedServers() []string
	// QueryServerStatus 查询指定服务器的在线玩家列表
	QueryServerStatus(ctx context.Context, serverName string) (
		players []PlayerStatusInfo, err error,
	)
}

// PlayerStatusInfo 玩家状态信息（从 protobuf 解耦）
type PlayerStatusInfo struct {
	PlayerUUID string
	PlayerName string
	WorldName  string
}

// sessionRecoveryStartupDelay 恢复前的等待延迟（测试中可覆盖）
var sessionRecoveryStartupDelay = 10 * time.Second

// RecoverSessions 恢复所有已连接服务器上的玩家会话
//
// 设计为在 goroutine 中调用（fire-and-forget），
// 函数内部会先等待 startupDelay 让 gRPC 流连接建立完毕，
// 再遍历所有已连接服务器查询在线玩家并恢复会话。
//
// 任意单台服务器或单个玩家的失败不会中断整体恢复流程。
func RecoverSessions(ctx context.Context, querier ServerQuerier) {
	log := xLog.WithName(xLog.NamedINIT, "SessionRecovery")
	startTime := time.Now()

	log.Info(ctx, "RecoverSessions - 等待 gRPC 流连接建立...")

	// 等待 MC 插件侧建立 gRPC 双向流连接
	select {
	case <-time.After(sessionRecoveryStartupDelay):
		log.Info(ctx, "RecoverSessions - 启动延迟结束，开始恢复会话")
	case <-ctx.Done():
		log.Warn(ctx, "RecoverSessions - 上下文已取消，跳过会话恢复")
		return
	}

	// 获取所有已连接的服务器
	servers := querier.GetAllConnectedServers()
	if len(servers) == 0 {
		log.Info(ctx, "RecoverSessions - 无已连接的服务器，跳过恢复")
		return
	}

	log.Info(ctx, fmt.Sprintf("RecoverSessions - 发现 %d 台已连接服务器，开始遍历恢复", len(servers)))

	manager := GetGlobalMatrixSessionManager()
	if manager == nil {
		log.Warn(ctx, "RecoverSessions - MatrixSessionManager 未初始化，无法恢复会话")
		return
	}

	var (
		totalServers  int
		totalPlayers  int
		failedServers int
	)

	for _, serverName := range servers {
		players, err := querier.QueryServerStatus(ctx, serverName)
		if err != nil {
			log.Warn(ctx, fmt.Sprintf("RecoverSessions - 查询服务器 [%s] 状态失败: %v，跳过", serverName, err))
			failedServers++
			continue
		}

		totalServers++
		recovered := 0

		for _, player := range players {
			playerUUID, parseErr := uuid.Parse(player.PlayerUUID)
			if parseErr != nil {
				log.Warn(ctx, fmt.Sprintf(
					"RecoverSessions - 服务器 [%s] 玩家 UUID 解析失败 [%s]: %v，跳过",
					serverName, player.PlayerUUID, parseErr,
				))
				continue
			}

			manager.GetOrCreate(ctx, serverName, playerUUID, player.PlayerName)
			recovered++
		}

		totalPlayers += recovered
		log.Info(ctx, fmt.Sprintf(
			"RecoverSessions - 服务器 [%s] 恢复完成，已恢复 %d 名玩家会话",
			serverName, recovered,
		))
	}

	elapsed := time.Since(startTime)
	log.Info(ctx, fmt.Sprintf(
		"RecoverSessions - 恢复完成：处理 %d/%d 台服务器，恢复 %d 名玩家会话，耗时 %v",
		totalServers, len(servers), totalPlayers, elapsed,
	))
	if failedServers > 0 {
		log.Warn(ctx, fmt.Sprintf(
			"RecoverSessions - %d 台服务器查询失败（已跳过）", failedServers,
		))
	}
}
