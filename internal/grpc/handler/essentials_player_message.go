package handler

import (
	"context"
	"fmt"
	"sync"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xGrpcGenerate "github.com/bamboo-services/bamboo-base-go/plugins/grpc/generate"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/utility"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	essentialspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/essentials/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// essentialsPlayerMessageStream 管理一条到 MC 服务器的 Server-Stream 连接
type essentialsPlayerMessageStream struct {
	stream     grpc.ServerStreamingServer[essentialspb.PlayerMessagePushResponse]
	serverName string
	log        *xLog.LogNamedLogger
	mu         sync.Mutex
}

// essentialsPlayerMessageStreamManager 管理活跃的消息推送流
var essentialsPlayerMessageStreamManager struct {
	mu      sync.RWMutex
	streams map[string]*essentialsPlayerMessageStream
}

func init() {
	essentialsPlayerMessageStreamManager.streams = make(map[string]*essentialsPlayerMessageStream)
}

// EssentialsPlayerMessageHandler 消息推送 gRPC Handler
type EssentialsPlayerMessageHandler struct {
	grpcHandler
	essentialspb.UnimplementedPlayerMessageServiceServer
}

// NewEssentialsPlayerMessageHandler 创建 EssentialsPlayerMessageHandler 实例并注册 gRPC 服务
func NewEssentialsPlayerMessageHandler(ctx context.Context, server grpc.ServiceRegistrar) *EssentialsPlayerMessageHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "EssentialsPlayerMessageHandler")
	h := &EssentialsPlayerMessageHandler{grpcHandler: *base}

	essentialspb.RegisterPlayerMessageServiceServer(server, h)
	xGrpcMiddle.UseStream(essentialspb.PlayerMessageService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

// PlayerMessageStream 消息推送服务端流
//
// MC 服务器（Essentials 插件）建立连接后，此方法阻塞直到连接断开。
// 连接期间，服务端通过 PushChatMessage / PushSystemMessage 向活跃流推送消息。
func (h *EssentialsPlayerMessageHandler) PlayerMessageStream(
	_ *emptypb.Empty,
	stream grpc.ServerStreamingServer[essentialspb.PlayerMessagePushResponse],
) error {
	ctx := stream.Context()

	serverName, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
	if xErr != nil {
		h.log.Warn(ctx, "PlayerMessageStream - 无法获取 plugin-name: "+xErr.Error())
		return xErr
	}

	h.log.Info(ctx, "PlayerMessageStream - 新连接: "+serverName)

	ms := &essentialsPlayerMessageStream{
		stream:     stream,
		serverName: serverName,
		log:        h.log,
	}
	h.setMessageStream(ctx, serverName, ms)

	<-ctx.Done()

	h.removeMessageStream(serverName)
	h.log.Info(ctx, "PlayerMessageStream - 连接关闭: "+serverName)
	return nil
}

// PushChatMessage 向所有活跃的 MC 服务器流推送聊天消息
func (h *EssentialsPlayerMessageHandler) PushChatMessage(ctx context.Context, senderName, message string) error {
	streams := h.getMessageStreams()
	h.log.Info(ctx, "PushChatMessage - 推送聊天消息，活跃流数: "+fmt.Sprintf("%d", len(streams)))

	resp := &essentialspb.PlayerMessagePushResponse{
		BaseResponse: &xGrpcGenerate.BaseResponse{Output: "Success"},
		PushType:     essentialspb.PlayerMessagePushType_PLAYER_MESSAGE_PUSH_TYPE_CHAT,
		Payload: &essentialspb.PlayerMessagePushResponse_ChatPush{
			ChatPush: &essentialspb.PlayerChatPush{
				SenderName: senderName,
				Message:    message,
				Source:     "web",
			},
		},
	}

	for _, ms := range streams {
		ms.mu.Lock()
		if err := ms.stream.Send(resp); err != nil {
			h.log.Warn(ctx, "PushChatMessage - 发送失败 ["+ms.serverName+"]: "+err.Error())
		}
		ms.mu.Unlock()
	}

	return nil
}

// PushSystemMessage 向所有活跃的 MC 服务器流推送系统消息
func (h *EssentialsPlayerMessageHandler) PushSystemMessage(ctx context.Context, title, content string) error {
	streams := h.getMessageStreams()
	h.log.Info(ctx, "PushSystemMessage - 推送系统消息，活跃流数: "+fmt.Sprintf("%d", len(streams)))

	resp := &essentialspb.PlayerMessagePushResponse{
		BaseResponse: &xGrpcGenerate.BaseResponse{Output: "Success"},
		PushType:     essentialspb.PlayerMessagePushType_PLAYER_MESSAGE_PUSH_TYPE_SYSTEM,
		Payload: &essentialspb.PlayerMessagePushResponse_SystemPush{
			SystemPush: &essentialspb.SystemMessagePush{
				Title:   title,
				Content: content,
			},
		},
	}

	for _, ms := range streams {
		ms.mu.Lock()
		if err := ms.stream.Send(resp); err != nil {
			h.log.Warn(ctx, "PushSystemMessage - 发送失败 ["+ms.serverName+"]: "+err.Error())
		}
		ms.mu.Unlock()
	}

	return nil
}

// PushPrivateMessage 向目标玩家所在的服务器推送私聊消息（定向推送）
func (h *EssentialsPlayerMessageHandler) PushPrivateMessage(ctx context.Context, senderName, senderUUID, message string) error {
	log := xLog.WithName(xLog.NamedGRPC)

	playerKey := string(bConst.CacheStatusPlayer.Get(senderUUID))
	playerData, err := h.rdb.HGetAll(ctx, playerKey).Result()
	if err != nil || len(playerData) == 0 {
		log.Info(ctx, "PushPrivateMessage - 玩家不在线或缓存不存在: "+senderUUID)
		return nil
	}

	serverName := playerData["server_name"]
	if serverName == "" {
		log.Info(ctx, "PushPrivateMessage - 玩家缓存中无 server_name: "+senderUUID)
		return nil
	}

	ms := h.getMessageStream(serverName)
	if ms == nil {
		log.Warn(ctx, "PushPrivateMessage - 无活跃流: "+serverName)
		return nil
	}

	resp := &essentialspb.PlayerMessagePushResponse{
		BaseResponse: &xGrpcGenerate.BaseResponse{Output: "Success"},
		PushType:     essentialspb.PlayerMessagePushType_PLAYER_MESSAGE_PUSH_TYPE_PRIVATE,
		Payload: &essentialspb.PlayerMessagePushResponse_PrivatePush{
			PrivatePush: &essentialspb.PrivateMessagePush{
				SenderName: senderName,
				Message:    message,
				SenderUuid: senderUUID,
			},
		},
	}

	ms.mu.Lock()
	if err := ms.stream.Send(resp); err != nil {
		log.Warn(ctx, "PushPrivateMessage - 发送失败 ["+serverName+"]: "+err.Error())
	}
	ms.mu.Unlock()

	return nil
}

func (h *EssentialsPlayerMessageHandler) setMessageStream(ctx context.Context, serverName string, ms *essentialsPlayerMessageStream) {
	essentialsPlayerMessageStreamManager.mu.Lock()
	defer essentialsPlayerMessageStreamManager.mu.Unlock()
	if _, exists := essentialsPlayerMessageStreamManager.streams[serverName]; exists {
		h.log.Warn(ctx, "PlayerMessageStream - 替换已存在的流: "+serverName)
	}
	essentialsPlayerMessageStreamManager.streams[serverName] = ms
}

func (h *EssentialsPlayerMessageHandler) getMessageStreams() []*essentialsPlayerMessageStream {
	essentialsPlayerMessageStreamManager.mu.RLock()
	defer essentialsPlayerMessageStreamManager.mu.RUnlock()
	result := make([]*essentialsPlayerMessageStream, 0, len(essentialsPlayerMessageStreamManager.streams))
	for _, ms := range essentialsPlayerMessageStreamManager.streams {
		result = append(result, ms)
	}
	return result
}

func (h *EssentialsPlayerMessageHandler) getMessageStream(serverName string) *essentialsPlayerMessageStream {
	essentialsPlayerMessageStreamManager.mu.RLock()
	defer essentialsPlayerMessageStreamManager.mu.RUnlock()
	return essentialsPlayerMessageStreamManager.streams[serverName]
}

func (h *EssentialsPlayerMessageHandler) removeMessageStream(serverName string) {
	essentialsPlayerMessageStreamManager.mu.Lock()
	defer essentialsPlayerMessageStreamManager.mu.Unlock()
	delete(essentialsPlayerMessageStreamManager.streams, serverName)
}
