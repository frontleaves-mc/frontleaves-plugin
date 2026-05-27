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

// globalPushPrivateMessageFunc 全局推送私聊消息到 MC 插件的回调函数
var (
	globalPushPrivateMessageFunc   func(ctx context.Context, senderName, senderUUID, message string) error
	globalPushPrivateMessageFuncMu sync.RWMutex
)

// SetGlobalPushPrivateMessageFunc 设置全局私聊推送函数（在 gRPC 注册时调用）
func SetGlobalPushPrivateMessageFunc(fn func(ctx context.Context, senderName, senderUUID, message string) error) {
	globalPushPrivateMessageFuncMu.Lock()
	defer globalPushPrivateMessageFuncMu.Unlock()
	globalPushPrivateMessageFunc = fn
}

// PushPrivateMessageToGame 推送私聊消息到 MC 插件
func PushPrivateMessageToGame(ctx context.Context, senderName, senderUUID, message string) error {
	globalPushPrivateMessageFuncMu.RLock()
	fn := globalPushPrivateMessageFunc
	globalPushPrivateMessageFuncMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(ctx, senderName, senderUUID, message)
}
