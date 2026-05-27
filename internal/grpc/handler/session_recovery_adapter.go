package handler

import (
	"context"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix"
)

// serverQuerierAdapter 实现 matrix.ServerQuerier 接口
// 桥接 gRPC handler 层的查询能力到 logic 层的恢复流程
type serverQuerierAdapter struct {
	queryHandler *EssentialsPlayerQueryHandler
}

var _ matrix.ServerQuerier = (*serverQuerierAdapter)(nil)

// NewServerQuerierAdapter 创建适配器实例
func NewServerQuerierAdapter(queryHandler *EssentialsPlayerQueryHandler) matrix.ServerQuerier {
	return &serverQuerierAdapter{queryHandler: queryHandler}
}

// GetAllConnectedServers 返回所有已连接的服务器名称
func (a *serverQuerierAdapter) GetAllConnectedServers() []string {
	essentialsQueryStreamManager.mu.RLock()
	defer essentialsQueryStreamManager.mu.RUnlock()

	servers := make([]string, 0, len(essentialsQueryStreamManager.streams))
	for name := range essentialsQueryStreamManager.streams {
		servers = append(servers, name)
	}
	return servers
}

// QueryServerStatus 查询指定服务器的在线玩家列表
func (a *serverQuerierAdapter) QueryServerStatus(ctx context.Context, serverName string) ([]matrix.PlayerStatusInfo, error) {
	players, _, _, _, err := a.queryHandler.QueryServerStatus(ctx, serverName)
	if err != nil {
		return nil, err
	}

	result := make([]matrix.PlayerStatusInfo, 0, len(players))
	for _, p := range players {
		result = append(result, matrix.PlayerStatusInfo{
			PlayerUUID: p.GetPlayerUuid(),
			PlayerName: p.GetPlayerName(),
			WorldName:  p.GetWorldName(),
		})
	}
	return result, nil
}