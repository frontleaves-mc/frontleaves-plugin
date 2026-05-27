package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	apiDirectMessage "github.com/frontleaves-mc/frontleaves-plugin/api/direct_message"
	apiMessage "github.com/frontleaves-mc/frontleaves-plugin/api/message"
	"github.com/gin-gonic/gin"
)

// ChatSSEClient SSE 客户端连接
type ChatSSEClient struct {
	Writer  gin.ResponseWriter
	Flusher http.Flusher
	UserID  string
	mu      sync.Mutex
}

// chatSSEManager SSE 客户端管理器
var chatSSEManager struct {
	mu         sync.RWMutex
	clients    map[string]*ChatSSEClient   // client_id -> client
	userClients map[string][]*ChatSSEClient // user_id -> clients (同一用户多 tab)
	recent     []apiMessage.SSEChatMessage
}

const maxRecentMessages = 50

func init() {
	chatSSEManager.clients = make(map[string]*ChatSSEClient)
	chatSSEManager.userClients = make(map[string][]*ChatSSEClient)
	chatSSEManager.recent = make([]apiMessage.SSEChatMessage, 0, maxRecentMessages)
}

// RegisterSSEClient 注册 SSE 客户端（同一用户多 tab 使用不同 clientID）
func RegisterSSEClient(clientID string, client *ChatSSEClient) {
	chatSSEManager.mu.Lock()
	defer chatSSEManager.mu.Unlock()
	chatSSEManager.clients[clientID] = client
	chatSSEManager.userClients[client.UserID] = append(chatSSEManager.userClients[client.UserID], client)
}

// RemoveSSEClient 移除 SSE 客户端
func RemoveSSEClient(clientID string) {
	chatSSEManager.mu.Lock()
	defer chatSSEManager.mu.Unlock()

	client, ok := chatSSEManager.clients[clientID]
	if !ok {
		return
	}
	delete(chatSSEManager.clients, clientID)

	// 从 userClients 索引中移除
	userID := client.UserID
	userClientList := chatSSEManager.userClients[userID]
	for i, c := range userClientList {
		if c == client {
			chatSSEManager.userClients[userID] = append(userClientList[:i], userClientList[i+1:]...)
			break
		}
	}
	// 清理空的用户条目
	if len(chatSSEManager.userClients[userID]) == 0 {
		delete(chatSSEManager.userClients, userID)
	}
}

// BroadcastChatMessage 广播聊天消息到所有 SSE 客户端
func BroadcastChatMessage(msg apiMessage.SSEChatMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	eventData := fmt.Sprintf("event: chat\ndata: %s\n\n", data)

	// 先在锁内复制客户端列表，再释放锁后做 I/O
	chatSSEManager.mu.Lock()
	chatSSEManager.recent = append(chatSSEManager.recent, msg)
	if len(chatSSEManager.recent) > maxRecentMessages {
		chatSSEManager.recent = chatSSEManager.recent[len(chatSSEManager.recent)-maxRecentMessages:]
	}
	clients := make([]*ChatSSEClient, 0, len(chatSSEManager.clients))
	for _, c := range chatSSEManager.clients {
		clients = append(clients, c)
	}
	chatSSEManager.mu.Unlock()

	// 锁外执行 I/O，每个客户端有独立 mutex 防止并发写入
	for _, client := range clients {
		client.mu.Lock()
		if _, writeErr := fmt.Fprint(client.Writer, eventData); writeErr == nil {
			client.Flusher.Flush()
		}
		client.mu.Unlock()
	}
}

// GetRecentMessages 获取最近的消息缓存
func GetRecentMessages() []apiMessage.SSEChatMessage {
	chatSSEManager.mu.RLock()
	defer chatSSEManager.mu.RUnlock()
	result := make([]apiMessage.SSEChatMessage, len(chatSSEManager.recent))
	copy(result, chatSSEManager.recent)
	return result
}

// SendDirectMessage 向指定用户的所有 SSE 客户端推送私信
func SendDirectMessage(receiverID string, msg apiDirectMessage.SSEDirectMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	eventData := fmt.Sprintf("event: dm\ndata: %s\n\n", data)

	chatSSEManager.mu.RLock()
	userClientList := chatSSEManager.userClients[receiverID]
	clients := make([]*ChatSSEClient, len(userClientList))
	copy(clients, userClientList)
	chatSSEManager.mu.RUnlock()

	for _, client := range clients {
		client.mu.Lock()
		if _, writeErr := fmt.Fprint(client.Writer, eventData); writeErr == nil {
			client.Flusher.Flush()
		}
		client.mu.Unlock()
	}
}

// SendInitMessages 向新客户端发送初始消息
func SendInitMessages(client *ChatSSEClient, messages []apiMessage.ChatLogResponse) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(client.Writer, "event: init\ndata: %s\n\n", data); err != nil {
		return err
	}
	client.Flusher.Flush()
	return nil
}
