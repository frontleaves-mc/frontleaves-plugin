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
	announcementpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/announcement/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// announcementStream 管理一个到 MC 服务器的 Server-Stream 连接
type announcementStream struct {
	stream     grpc.ServerStreamingServer[announcementpb.AnnouncementPushResponse]
	serverName string
	log        *xLog.LogNamedLogger
}

// announcementStreamManager 管理活跃的公告推送流
// 按 server_name 管理多条并发连接
var announcementStreamManager struct {
	mu      sync.RWMutex
	streams map[string]*announcementStream
}

func init() {
	announcementStreamManager.streams = make(map[string]*announcementStream)
}

// AnnouncementHandler 公告推送 gRPC Handler
type AnnouncementHandler struct {
	grpcHandler
	announcementpb.UnimplementedAnnouncementServiceServer
}

// NewAnnouncementHandler 构造 AnnouncementHandler
func NewAnnouncementHandler(ctx context.Context, server grpc.ServiceRegistrar) *AnnouncementHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "AnnouncementHandler")
	h := &AnnouncementHandler{grpcHandler: *base}

	announcementpb.RegisterAnnouncementServiceServer(server, h)
	xGrpcMiddle.UseStream(announcementpb.AnnouncementService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

// AnnouncementStream 公告推送服务端流
//
// MC 服务器建立连接后，此方法阻塞直到连接断开。
// 连接期间，服务端通过 PushAnnouncement 向所有活跃流推送公告。
func (h *AnnouncementHandler) AnnouncementStream(
	_ *emptypb.Empty,
	stream grpc.ServerStreamingServer[announcementpb.AnnouncementPushResponse],
) error {
	ctx := stream.Context()

	// 从 metadata 提取插件名称作为 serverName
	serverName, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
	if xErr != nil {
		h.log.Warn(ctx, "AnnouncementStream - 无法获取 plugin-name: "+xErr.Error())
		return xErr
	}

	h.log.Info(ctx, "AnnouncementStream - 新连接: "+serverName)

	// 注册流
	as := &announcementStream{
		stream:     stream,
		serverName: serverName,
		log:        h.log,
	}
	h.setAnnouncementStream(serverName, as)

	// 阻塞直到客户端断开
	<-ctx.Done()

	// 清理
	h.removeAnnouncementStream(serverName)
	h.log.Info(ctx, "AnnouncementStream - 连接关闭: "+serverName)
	return nil
}

// PushAnnouncement 向所有活跃的 MC 服务器流推送公告
//
// fire-and-forget 模式：单条流发送失败仅记录日志，不影响其他流。
func (h *AnnouncementHandler) PushAnnouncement(
	ctx context.Context,
	announcementID, title, content string,
	annType int32,
) error {
	streams := h.getAnnouncementStreams()
	h.log.Info(ctx, "PushAnnouncement - 推送公告 "+announcementID+"，活跃流数: "+fmt.Sprintf("%d", len(streams)))

	resp := &announcementpb.AnnouncementPushResponse{
		BaseResponse:   &xGrpcGenerate.BaseResponse{Output: "Success"},
		AnnouncementId: announcementID,
		Title:          title,
		Content:        content,
		Type:           annType,
	}

	for _, as := range streams {
		if err := as.stream.Send(resp); err != nil {
			h.log.Warn(ctx, "PushAnnouncement - 发送失败 ["+as.serverName+"]: "+err.Error())
		}
	}

	return nil
}

// setAnnouncementStream 注册/替换指定服务器的公告推送流
func (h *AnnouncementHandler) setAnnouncementStream(serverName string, as *announcementStream) {
	announcementStreamManager.mu.Lock()
	defer announcementStreamManager.mu.Unlock()
	if _, exists := announcementStreamManager.streams[serverName]; exists {
		h.log.Warn(context.Background(), "AnnouncementStream - 替换已存在的流: "+serverName)
	}
	announcementStreamManager.streams[serverName] = as
	h.log.Info(context.Background(), "AnnouncementStream - 流已注册: "+serverName)
}

// getAnnouncementStreams 获取所有活跃的公告推送流
func (h *AnnouncementHandler) getAnnouncementStreams() []*announcementStream {
	announcementStreamManager.mu.RLock()
	defer announcementStreamManager.mu.RUnlock()
	result := make([]*announcementStream, 0, len(announcementStreamManager.streams))
	for _, as := range announcementStreamManager.streams {
		result = append(result, as)
	}
	return result
}

// removeAnnouncementStream 移除指定服务器的公告推送流
func (h *AnnouncementHandler) removeAnnouncementStream(serverName string) {
	announcementStreamManager.mu.Lock()
	defer announcementStreamManager.mu.Unlock()
	delete(announcementStreamManager.streams, serverName)
	h.log.Info(context.Background(), "AnnouncementStream - 流已移除: "+serverName)
}
