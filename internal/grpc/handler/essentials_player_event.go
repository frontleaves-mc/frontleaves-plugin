package handler

import (
	"context"
	"strconv"
	"time"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/utility"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	essentialspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/essentials/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	apiMessage "github.com/frontleaves-mc/frontleaves-plugin/api/message"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/sse"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const playerEventTTL = 5 * time.Minute

// EssentialsPlayerEventHandler 处理玩家事件客户端流式 RPC
type EssentialsPlayerEventHandler struct {
	grpcHandler
	*essentialsService
	essentialspb.UnimplementedPlayerEventServiceServer
}

// NewEssentialsPlayerEventHandler 创建 EssentialsPlayerEventHandler 实例并注册 gRPC 服务
func NewEssentialsPlayerEventHandler(ctx context.Context, server grpc.ServiceRegistrar) *EssentialsPlayerEventHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "EssentialsPlayerEventHandler")
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *base,
		essentialsService: newEssentialsService(ctx),
	}

	essentialspb.RegisterPlayerEventServiceServer(server, h)
	xGrpcMiddle.UseStream(essentialspb.PlayerEventService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

// PlayerEventStream 实现玩家事件客户端流式 RPC
//
// 接收客户端持续发送的 PlayerEventStreamRequest 消息，
// 根据 event_type 分发到对应的处理逻辑。
func (h *EssentialsPlayerEventHandler) PlayerEventStream(
	stream grpc.ClientStreamingServer[essentialspb.PlayerEventStreamRequest, essentialspb.PlayerEventStreamResponse],
) error {
	ctx := stream.Context()
	if h.log != nil {
		h.log.Info(ctx, "PlayerEventStream - 新的客户端流连接")
	}

	for {
		req, err := stream.Recv()
		if err != nil {
			if status.Code(err) == codes.Canceled || status.Code(err) == codes.OK {
				if h.log != nil {
					h.log.Info(ctx, "PlayerEventStream - 流正常关闭")
				}
				return nil
			}
			if h.log != nil {
				h.log.Warn(ctx, "PlayerEventStream - 流读取错误: "+err.Error())
			}
			return err
		}

		h.dispatchPlayerEvent(ctx, req)
	}
}

// dispatchPlayerEvent 根据事件类型分发到对应处理逻辑
func (h *EssentialsPlayerEventHandler) dispatchPlayerEvent(
	ctx context.Context,
	req *essentialspb.PlayerEventStreamRequest,
) {
	switch req.EventType {
	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_JOIN:
		h.handlePlayerJoin(ctx, req.GetPlayerJoinEvent())

	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_QUIT:
		h.handlePlayerQuit(ctx, req.GetPlayerQuitEvent())

	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_CHAT:
		h.handlePlayerChat(ctx, req.GetPlayerChatEvent())

	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_KICK:
		h.handlePlayerKick(ctx, req.GetPlayerKickEvent())

	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_DEATH:
		h.handlePlayerDeath(ctx, req.GetPlayerDeathEvent())

	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_GROUP_CHANGE:
		h.handlePlayerGroupChange(ctx, req.GetPlayerGroupChangeEvent())

	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_COMMAND:
		h.handlePlayerCommand(ctx, req.GetPlayerCommandEvent())

	case essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PRIVATE_CHAT:
		h.handlePrivateChat(ctx, req.GetPrivateChatEvent())

	default:
		pluginName, _ := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
		if h.log != nil {
			h.log.Warn(ctx, "dispatchPlayerEvent - 收到未知事件类型, plugin="+pluginName)
		}
	}
}

// handlePlayerJoin 处理玩家加入事件
//
// Redis 操作：更新玩家在线状态 + 服务器在线玩家集合 + 服务器在线人数。
// DB 同步：UpsertGameProfile + PlayerJoined。
func (h *EssentialsPlayerEventHandler) handlePlayerJoin(
	ctx context.Context,
	join *essentialspb.PlayerJoinEvent,
) {
	if join == nil {
		return
	}

	playerUUID := join.GetPlayerUuid()
	playerName := join.GetPlayerName()
	serverName := join.GetServerName()
	worldName := join.GetWorldName()

	if playerUUID == "" || playerName == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerJoinEvent - 玩家 UUID 或名称为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerJoinEvent - 玩家加入: "+playerName)
	}

	// Redis: 更新玩家在线状态
	if h.rdb != nil {
		playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
		h.rdb.HSet(ctx, playerKey, map[string]any{
			"server_name": serverName,
			"world_name":  worldName,
			"player_name": playerName,
			"online":      "true",
			"last_seen":   strconv.FormatInt(time.Now().UnixMilli(), 10),
		})
		h.rdb.Expire(ctx, playerKey, playerEventTTL)

		serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
		h.rdb.SAdd(ctx, serverPlayersKey, playerUUID)
		h.rdb.Expire(ctx, serverPlayersKey, playerEventTTL)

		serverKey := string(bConst.CacheStatusServer.Get(serverName))
		onlineCount := h.rdb.SCard(ctx, serverPlayersKey).Val()
		h.rdb.HSet(ctx, serverKey, "online_players", strconv.FormatInt(onlineCount, 10))
	}

	parsedUUID, err := uuid.Parse(playerUUID)
	if err != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerJoinEvent - 玩家 UUID 格式无效: "+playerUUID)
		}
		return
	}

	groupName := join.GetGroupName()
	if groupName == "" {
		groupName = "PLAYER"
	}
	if h.gameProfileLogic != nil {
		if syncErr := h.gameProfileLogic.Upsert(ctx, 0, parsedUUID, playerName, groupName); syncErr != nil {
			if h.log != nil {
				h.log.Warn(ctx, "PlayerJoinEvent - 同步 GameProfile 失败: "+syncErr.Error())
			}
		}
	}

	if h.serverLogic != nil && h.serverPlayerLogic != nil {
		if server, xErr := h.serverLogic.GetOrCreateByName(ctx, serverName); xErr == nil && server != nil {
			if err := h.serverPlayerLogic.PlayerJoined(ctx, server.ID, parsedUUID, playerName, worldName); err != nil {
				if h.log != nil {
					h.log.Warn(ctx, "PlayerJoinEvent - DB 同步玩家加入失败: "+err.Error())
				}
			}
		}
	}
}

// handlePlayerQuit 处理玩家离开事件
//
// Redis 操作：更新玩家离线状态 + 从服务器在线玩家集合移除 + 更新在线人数。
// DB 同步：PlayerLeft。
func (h *EssentialsPlayerEventHandler) handlePlayerQuit(
	ctx context.Context,
	quit *essentialspb.PlayerQuitEvent,
) {
	if quit == nil {
		return
	}

	playerUUID := quit.GetPlayerUuid()
	serverName := quit.GetServerName()

	if playerUUID == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerQuitEvent - 玩家 UUID 为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerQuitEvent - 玩家离开: "+quit.GetPlayerName())
	}

	if h.rdb != nil {
		playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
		h.rdb.HSet(ctx, playerKey, map[string]any{
			"online":    "false",
			"last_seen": strconv.FormatInt(time.Now().UnixMilli(), 10),
		})
		h.rdb.Expire(ctx, playerKey, playerEventTTL)

		serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
		h.rdb.SRem(ctx, serverPlayersKey, playerUUID)

		serverKey := string(bConst.CacheStatusServer.Get(serverName))
		onlineCount := h.rdb.SCard(ctx, serverPlayersKey).Val()
		h.rdb.HSet(ctx, serverKey, "online_players", strconv.FormatInt(onlineCount, 10))
	}

	parsedUUID, parseErr := uuid.Parse(playerUUID)
	if parseErr != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerQuitEvent - 玩家 UUID 格式无效: "+playerUUID)
		}
		return
	}

	if h.serverLogic != nil && h.serverPlayerLogic != nil {
		if server, xErr := h.serverLogic.GetOrCreateByName(ctx, serverName); xErr == nil && server != nil {
			if err := h.serverPlayerLogic.PlayerLeft(ctx, server.ID, parsedUUID); err != nil {
				if h.log != nil {
					h.log.Warn(ctx, "PlayerQuitEvent - DB 同步玩家离开失败: "+err.Error())
				}
			}
		}
	}
}

// handlePlayerChat 处理玩家聊天事件
func (h *EssentialsPlayerEventHandler) handlePlayerChat(
	ctx context.Context,
	chat *essentialspb.PlayerChatEvent,
) {
	if chat == nil {
		return
	}

	if chat.GetPlayerUuid() == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerChatEvent - player_uuid 为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerChatEvent - 玩家聊天: "+chat.GetPlayerName())
	}

	parsedUUID, err := uuid.Parse(chat.GetPlayerUuid())
	if err != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerChatEvent - player_uuid 格式无效: "+err.Error())
		}
		return
	}

	if h.playerChatLogic != nil {
		// 反查关联 UserID
		var userID *xSnowflake.SnowflakeID
		if h.gameProfileLogic != nil {
			userID, _ = h.gameProfileLogic.GetUserIDByUUID(ctx, parsedUUID)
		}
		if xErr := h.playerChatLogic.RecordChat(ctx, parsedUUID, chat.GetPlayerName(),
			chat.GetServerName(), chat.GetWorldName(), chat.GetMessage(), userID); xErr != nil {
			if h.log != nil {
				h.log.Warn(ctx, "PlayerChatEvent - 记录聊天消息失败: "+xErr.Error())
			}
		}
	}

	// 广播到 SSE 客户端
	sse.BroadcastChatMessage(apiMessage.SSEChatMessage{
		PlayerName: chat.GetPlayerName(),
		ServerName: chat.GetServerName(),
		Message:    chat.GetMessage(),
		Source:     1,
	})
}

// handlePlayerKick 处理玩家被踢出事件
func (h *EssentialsPlayerEventHandler) handlePlayerKick(
	ctx context.Context,
	kick *essentialspb.PlayerKickEvent,
) {
	if kick == nil {
		return
	}

	if kick.GetPlayerUuid() == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerKickEvent - player_uuid 为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerKickEvent - 玩家被踢出: "+kick.GetPlayerName())
	}

	parsedUUID, err := uuid.Parse(kick.GetPlayerUuid())
	if err != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerKickEvent - player_uuid 格式无效: "+err.Error())
		}
		return
	}

	if h.playerEventLogic != nil {
		if xErr := h.playerEventLogic.RecordEvent(ctx, parsedUUID, kick.GetPlayerName(),
			kick.GetServerName(), kick.GetWorldName(), bConst.PlayerEventKick, kick.GetReason()); xErr != nil {
			if h.log != nil {
				h.log.Warn(ctx, "PlayerKickEvent - 记录踢出事件失败: "+xErr.Error())
			}
		}
	}
}

// handlePlayerDeath 处理玩家死亡事件
func (h *EssentialsPlayerEventHandler) handlePlayerDeath(
	ctx context.Context,
	death *essentialspb.PlayerDeathEvent,
) {
	if death == nil {
		return
	}

	if death.GetPlayerUuid() == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerDeathEvent - player_uuid 为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerDeathEvent - 玩家死亡: "+death.GetPlayerName())
	}

	parsedUUID, err := uuid.Parse(death.GetPlayerUuid())
	if err != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerDeathEvent - player_uuid 格式无效: "+err.Error())
		}
		return
	}

	if h.playerEventLogic != nil {
		if xErr := h.playerEventLogic.RecordEvent(ctx, parsedUUID, death.GetPlayerName(),
			death.GetServerName(), death.GetWorldName(), bConst.PlayerEventDeath, death.GetDeathMessage()); xErr != nil {
			if h.log != nil {
				h.log.Warn(ctx, "PlayerDeathEvent - 记录死亡事件失败: "+xErr.Error())
			}
		}
	}
}

// handlePlayerGroupChange 处理玩家权限组变更事件
func (h *EssentialsPlayerEventHandler) handlePlayerGroupChange(
	ctx context.Context,
	gc *essentialspb.PlayerGroupChangeEvent,
) {
	if gc == nil {
		return
	}

	if gc.GetPlayerUuid() == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerGroupChangeEvent - player_uuid 为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerGroupChangeEvent - 权限组变更: "+gc.GetPlayerName()+" "+gc.GetOldGroupName()+" → "+gc.GetGroupName())
	}

	parsedUUID, err := uuid.Parse(gc.GetPlayerUuid())
	if err != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerGroupChangeEvent - player_uuid 格式无效: "+err.Error())
		}
		return
	}

	if h.gameProfileLogic != nil {
		if updateErr := h.gameProfileLogic.UpdateGroupName(ctx, parsedUUID, gc.GetGroupName()); updateErr != nil {
			if h.log != nil {
				h.log.Warn(ctx, "PlayerGroupChangeEvent - 更新 GameProfile 权限组失败: "+updateErr.Error())
			}
		}
	}
}

// handlePlayerCommand 处理玩家执行指令事件
func (h *EssentialsPlayerEventHandler) handlePlayerCommand(
	ctx context.Context,
	cmd *essentialspb.PlayerCommandEvent,
) {
	if cmd == nil {
		return
	}

	if cmd.GetPlayerUuid() == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerCommandEvent - player_uuid 为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerCommandEvent - 玩家指令: "+cmd.GetPlayerName()+" → "+cmd.GetCommand())
	}

	parsedUUID, err := uuid.Parse(cmd.GetPlayerUuid())
	if err != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerCommandEvent - player_uuid 格式无效: "+err.Error())
		}
		return
	}

	if h.playerCommandLogic != nil {
		// 反查关联 UserID
		var userID *xSnowflake.SnowflakeID
		if h.gameProfileLogic != nil {
			userID, _ = h.gameProfileLogic.GetUserIDByUUID(ctx, parsedUUID)
		}
		if xErr := h.playerCommandLogic.RecordCommand(ctx, parsedUUID, cmd.GetPlayerName(),
			cmd.GetServerName(), cmd.GetWorldName(), cmd.GetCommand(), userID); xErr != nil {
			if h.log != nil {
				h.log.Warn(ctx, "PlayerCommandEvent - 记录指令日志失败: "+xErr.Error())
			}
		}
	}
}

// handlePrivateChat 处理玩家私聊事件
func (h *EssentialsPlayerEventHandler) handlePrivateChat(
	ctx context.Context,
	event *essentialspb.PlayerPrivateChatEvent,
) {
	if event == nil {
		return
	}

	if event.GetSenderUuid() == "" {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerPrivateChatEvent - sender_uuid 为空，跳过")
		}
		return
	}
	if h.log != nil {
		h.log.Info(ctx, "PlayerPrivateChatEvent - 玩家私聊: "+event.GetSenderName()+" → "+event.GetReceiverName())
	}

	// 校验发送者 UUID 格式
	senderUUID, err := uuid.Parse(event.GetSenderUuid())
	if err != nil {
		if h.log != nil {
			h.log.Warn(ctx, "PlayerPrivateChatEvent - sender_uuid 格式无效: "+err.Error())
		}
		return
	}

	// 记录私信：UUID 作为 senderUserID，source=1 表示 Game 来源
	if h.directMessageLogic != nil {
		if xErr := h.directMessageLogic.RecordDirectMessage(ctx, senderUUID.String(),
			event.GetSenderName(), event.GetReceiverName(), event.GetMessage(), 1); xErr != nil {
			if h.log != nil {
				h.log.Warn(ctx, "PlayerPrivateChatEvent - 记录私信失败: "+xErr.Error())
			}
		}
	}
}
