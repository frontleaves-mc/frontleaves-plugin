package handler

import (
	"context"
	"sync"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	statuspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/status/v1"
	"google.golang.org/grpc"
)

// eventStream 管理一个来自 Java 插件的客户端流连接
type eventStream struct {
	stream     grpc.ClientStreamingServer[statuspb.ServerEventStreamRequest, statuspb.ServerEventStreamResponse]
	serverName string
	log        *xLog.LogNamedLogger
}

// eventStreamManager 管理活跃的 ServerEventStream 客户端流
// 按 server_name 管理多条并发连接
var eventStreamManager struct {
	mu      sync.RWMutex
	streams map[string]*eventStream
}

func init() {
	eventStreamManager.streams = make(map[string]*eventStream)
}

// setEventStream 注册/替换指定服务器的客户端流
func (h *ServerStatusHandler) setEventStream(serverName string, es *eventStream) {
	eventStreamManager.mu.Lock()
	defer eventStreamManager.mu.Unlock()
	eventStreamManager.streams[serverName] = es
	h.log.Info(context.Background(), "EventStream - 流已注册: "+serverName)
}

// getEventStream 获取指定服务器的客户端流
func (h *ServerStatusHandler) getEventStream(serverName string) *eventStream {
	eventStreamManager.mu.RLock()
	defer eventStreamManager.mu.RUnlock()
	return eventStreamManager.streams[serverName]
}

// removeEventStream 移除指定服务器的客户端流
func (h *ServerStatusHandler) removeEventStream(serverName string) {
	eventStreamManager.mu.Lock()
	defer eventStreamManager.mu.Unlock()
	delete(eventStreamManager.streams, serverName)
	h.log.Info(context.Background(), "EventStream - 流已移除: "+serverName)
}
