package handler

import (
	"fmt"
	"sync"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	essentialspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/essentials/v1"
	"google.golang.org/grpc"
)

// essentialsQueryStream 管理一条到 MC 服务器的双向流连接
type essentialsQueryStream struct {
	stream     grpc.BidiStreamingServer[essentialspb.PlayerQueryRequest, essentialspb.PlayerQueryResponse]
	serverName string
	pending    sync.Map // request_id -> chan *essentialspb.PlayerQueryRequest
	log        *xLog.LogNamedLogger
}

// essentialsQueryStreamManager 管理活跃的 Essentials 查询流
// 按 server_name 管理多条并发连接
var essentialsQueryStreamManager struct {
	mu      sync.RWMutex
	streams map[string]*essentialsQueryStream // server_name -> stream
}

func init() {
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
}

// registerStream 注册一条新的查询流
func registerEssentialsQueryStream(serverName string, qs *essentialsQueryStream) {
	essentialsQueryStreamManager.mu.Lock()
	defer essentialsQueryStreamManager.mu.Unlock()
	essentialsQueryStreamManager.streams[serverName] = qs
}

// removeStream 移除指定服务器的查询流
func removeEssentialsQueryStream(serverName string) {
	essentialsQueryStreamManager.mu.Lock()
	defer essentialsQueryStreamManager.mu.Unlock()
	delete(essentialsQueryStreamManager.streams, serverName)
}

// getEssentialsQueryStream 获取指定服务器的查询流
func getEssentialsQueryStream(serverName string) *essentialsQueryStream {
	essentialsQueryStreamManager.mu.RLock()
	defer essentialsQueryStreamManager.mu.RUnlock()
	return essentialsQueryStreamManager.streams[serverName]
}

// getAnyEssentialsQueryStream 获取任意一条活跃的查询流（用于无特定服务器目标的查询）
func getAnyEssentialsQueryStream() *essentialsQueryStream {
	essentialsQueryStreamManager.mu.RLock()
	defer essentialsQueryStreamManager.mu.RUnlock()
	for _, qs := range essentialsQueryStreamManager.streams {
		return qs
	}
	return nil
}

// generateEssentialsRequestID 生成请求追踪标识
func generateEssentialsRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
