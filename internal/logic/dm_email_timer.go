package logic

import (
	"context"
	"fmt"
	"sync"
	"time"

	xAsync "github.com/bamboo-services/bamboo-base-go/plugins/async"
	xEmail "github.com/bamboo-services/bamboo-base-go/plugins/email"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/google/uuid"
)

// dmMessageSnapshot captures a message for email batching
type dmMessageSnapshot struct {
	SenderName string
	Message    string
	SentAt     time.Time
}

// dmTimerEntry holds the timer and accumulated messages for one (sender, receiver) pair
type dmTimerEntry struct {
	timer    *time.Timer
	messages []dmMessageSnapshot
}

// DmEmailTimer manages delayed email notifications for offline DMs.
// When a DM is sent to an offline user, a 2-minute timer starts (or resets).
// All messages accumulated within that window are batched into a single email.
type DmEmailTimer struct {
	mu     sync.Mutex
	timers map[string]*dmTimerEntry // key: "senderID:receiverID"
	log    *xLog.LogNamedLogger
}

const (
	dmTimerDelay       = 2 * time.Minute
	dmTimerMaxMessages = 10000
)

// NewDmEmailTimer creates a new DmEmailTimer instance
func NewDmEmailTimer() *DmEmailTimer {
	return &DmEmailTimer{
		timers: make(map[string]*dmTimerEntry),
		log:    xLog.WithName(xLog.NamedLOGC, "DmEmailTimer"),
	}
}

// Schedule adds a message to the batch for the given (sender, receiver) pair
// and starts (or resets) the 2-minute accumulation timer.
func (t *DmEmailTimer) Schedule(ctx context.Context, senderID, receiverID uuid.UUID, senderName string, msg dmMessageSnapshot) {
	key := fmt.Sprintf("%s:%s", senderID, receiverID)

	t.mu.Lock()
	defer t.mu.Unlock()

	entry, exists := t.timers[key]
	if exists {
		// Stop existing timer (drain the channel to prevent stale fires)
		entry.timer.Stop()
		// Append message with cap guard
		if len(entry.messages) < dmTimerMaxMessages {
			entry.messages = append(entry.messages, msg)
		}
	} else {
		// Create new entry
		entry = &dmTimerEntry{
			messages: []dmMessageSnapshot{msg},
		}
		t.timers[key] = entry
	}

	// (Re)start the timer — when it fires, send the accumulated batch
	entry.timer = time.AfterFunc(dmTimerDelay, func() {
		// Snapshot messages under lock, then release
		t.mu.Lock()
		snapshot := make([]dmMessageSnapshot, len(entry.messages))
		copy(snapshot, entry.messages)
		delete(t.timers, key)
		t.mu.Unlock()

		// Use a separate entry reference for the async call
		batchEntry := &dmTimerEntry{messages: snapshot}
		t.sendEmail(ctx, senderID, receiverID, senderName, batchEntry)
	})
}

// Cancel cleans up timer entries for BOTH directions between sender and receiver.
// This should be called when the receiver comes online (no more need for email notifications).
func (t *DmEmailTimer) Cancel(senderID, receiverID uuid.UUID) {
	keyForward := fmt.Sprintf("%s:%s", senderID, receiverID)
	keyReverse := fmt.Sprintf("%s:%s", receiverID, senderID)

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, key := range []string{keyForward, keyReverse} {
		if entry, ok := t.timers[key]; ok {
			entry.timer.Stop()
			delete(t.timers, key)
		}
	}
}

// sendEmail asynchronously sends the accumulated DM batch as a single email.
// This is called from the timer callback — all work is offloaded via xAsync.
func (t *DmEmailTimer) sendEmail(ctx context.Context, senderID, receiverID uuid.UUID, senderName string, entry *dmTimerEntry) {
	if len(entry.messages) == 0 {
		return
	}

	xAsync.Async(
		ctx,
		func(asyncCtx context.Context) {
			emailClient, ok := t.getEmailClient(asyncCtx)
			if !ok || emailClient == nil {
				t.log.SugarWarn(asyncCtx, "邮件客户端不可用，跳离线私信通知",
					"sender", senderName,
					"receiver_id", receiverID,
					"message_count", len(entry.messages),
				)
				return
			}

			msg := &xEmail.Message{
				To:       nil, // TODO: 由上层调用方填充收件人邮箱地址
				Subject:  fmt.Sprintf("你收到了来自 %s 的离线私信", senderName),
				Template: "private_message",
				TemplateData: map[string]any{
					"SenderName":   senderName,
					"Messages":     entry.messages,
					"ReceiverName": "", // TODO: 由上层调用方填充
				},
			}

			if err := emailClient.SendTemplate(asyncCtx, msg); err != nil {
				t.log.SugarError(asyncCtx, "发送离线私信邮件失败",
					"error", err,
					"sender", senderName,
					"receiver_id", receiverID,
				)
				return
			}

			t.log.SugarInfo(asyncCtx, "离线私信邮件发送成功",
				"sender", senderName,
				"receiver_id", receiverID,
				"message_count", len(entry.messages),
			)
		},
		xAsync.WithName("DmEmailNotification"),
		xAsync.WithDebug(),
	)
}

// getEmailClient retrieves the email client from the context DI container.
func (t *DmEmailTimer) getEmailClient(ctx context.Context) (*xEmail.EmailClient, bool) {
	client, err := xCtxUtil.Get[*xEmail.EmailClient](ctx, xCtx.EmailClientKey)
	if err != nil {
		return nil, false
	}
	return client, true
}
