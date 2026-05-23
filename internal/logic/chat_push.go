package logic

import (
	"context"
	"sync"
)

// globalPushChatFunc 全局推送聊天消息到 MC 插件的回调函数
var (
	globalPushChatFunc   func(ctx context.Context, senderName, message string) error
	globalPushChatFuncMu sync.RWMutex
)

// SetGlobalPushChatFunc 设置全局推送函数（在 gRPC 注册时调用）
func SetGlobalPushChatFunc(fn func(ctx context.Context, senderName, message string) error) {
	globalPushChatFuncMu.Lock()
	defer globalPushChatFuncMu.Unlock()
	globalPushChatFunc = fn
}

// PushChatToGame 推送聊天消息到 MC 插件
func PushChatToGame(ctx context.Context, senderName, message string) error {
	globalPushChatFuncMu.RLock()
	fn := globalPushChatFunc
	globalPushChatFuncMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(ctx, senderName, message)
}
