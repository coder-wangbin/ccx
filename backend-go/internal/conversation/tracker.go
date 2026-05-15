package conversation

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

type Conversation struct {
	ID             string    `json:"id"`
	Kind           string    `json:"kind"`
	UserID         string    `json:"userId"`
	RawUserID      string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	LastActiveAt   time.Time `json:"lastActiveAt"`
	RequestCount   int       `json:"requestCount"`
	Models         []string  `json:"models"`
	CurrentChannel int       `json:"currentChannel"`
	ChannelName    string    `json:"channelName"`
	Status         string    `json:"status"`
	LastModel      string    `json:"lastModel"`
	LastRequestID  string    `json:"lastRequestId"`
}

type ConversationTracker struct {
	mu             sync.RWMutex
	conversations  map[string]*Conversation
	sessionMapping map[string]string // sessionID → conversationID (for Responses)
	userMapping    map[string]string // kind:userID → conversationID (for Chat/Messages/Gemini)
	idleTTL        time.Duration
	expireTTL      time.Duration
	stopCh         chan struct{}
}

func NewConversationTracker(idleTTL, expireTTL time.Duration) *ConversationTracker {
	ct := &ConversationTracker{
		conversations:  make(map[string]*Conversation),
		sessionMapping: make(map[string]string),
		userMapping:    make(map[string]string),
		idleTTL:        idleTTL,
		expireTTL:      expireTTL,
		stopCh:         make(chan struct{}),
	}
	go ct.cleanupLoop()
	return ct
}

func (ct *ConversationTracker) Track(kind, userID, model string, channelIndex int, channelName, sessionID string) {
	if userID == "" {
		return
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	convID := ct.resolveConversationID(kind, userID, sessionID)
	now := time.Now()

	conv, exists := ct.conversations[convID]
	if !exists {
		conv = &Conversation{
			ID:        convID,
			Kind:      kind,
			UserID:    maskUserID(userID),
			RawUserID: userID,
			CreatedAt: now,
			Status:    "active",
			Models:    []string{},
		}
		ct.conversations[convID] = conv

		if sessionID != "" {
			ct.sessionMapping[sessionID] = convID
		}
		compositeKey := kind + ":" + userID
		ct.userMapping[compositeKey] = convID
	}

	conv.LastActiveAt = now
	conv.RequestCount++
	conv.CurrentChannel = channelIndex
	conv.ChannelName = channelName
	conv.LastModel = model
	conv.Status = "active"

	if !containsString(conv.Models, model) {
		conv.Models = append(conv.Models, model)
	}
}

func (ct *ConversationTracker) UpdateStatus(kind, userID, status string) {
	if userID == "" {
		return
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	compositeKey := kind + ":" + userID
	convID, ok := ct.userMapping[compositeKey]
	if !ok {
		return
	}
	conv, ok := ct.conversations[convID]
	if !ok {
		return
	}
	conv.Status = status
	conv.LastActiveAt = time.Now()
}

func (ct *ConversationTracker) SetLastRequestID(kind, userID, requestID string) {
	if userID == "" {
		return
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	compositeKey := kind + ":" + userID
	convID, ok := ct.userMapping[compositeKey]
	if !ok {
		return
	}
	conv, ok := ct.conversations[convID]
	if !ok {
		return
	}
	conv.LastRequestID = requestID
}

func (ct *ConversationTracker) GetActiveConversations(kindFilter string) []*Conversation {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make([]*Conversation, 0, len(ct.conversations))
	for _, conv := range ct.conversations {
		if kindFilter != "" && conv.Kind != kindFilter {
			continue
		}
		result = append(result, conv)
	}
	return result
}

func (ct *ConversationTracker) GetConversation(id string) (*Conversation, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	conv, ok := ct.conversations[id]
	return conv, ok
}

func (ct *ConversationTracker) GetConversationByUser(kind, userID string) (*Conversation, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	compositeKey := kind + ":" + userID
	convID, ok := ct.userMapping[compositeKey]
	if !ok {
		return nil, false
	}
	conv, ok := ct.conversations[convID]
	return conv, ok
}

func (ct *ConversationTracker) Stop() {
	close(ct.stopCh)
}

func (ct *ConversationTracker) resolveConversationID(kind, userID, sessionID string) string {
	if sessionID != "" {
		if convID, ok := ct.sessionMapping[sessionID]; ok {
			return convID
		}
		return sessionID
	}

	compositeKey := kind + ":" + userID
	if convID, ok := ct.userMapping[compositeKey]; ok {
		return convID
	}

	hash := sha256.Sum256([]byte(compositeKey))
	return "conv_" + hex.EncodeToString(hash[:6])
}

func (ct *ConversationTracker) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ct.stopCh:
			return
		case <-ticker.C:
			ct.cleanup()
		}
	}
}

func (ct *ConversationTracker) cleanup() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	var removed int

	for id, conv := range ct.conversations {
		idleDuration := now.Sub(conv.LastActiveAt)

		if idleDuration > ct.expireTTL {
			ct.removeConversation(id, conv)
			removed++
		} else if idleDuration > ct.idleTTL && conv.Status != "idle" {
			conv.Status = "idle"
		}
	}

	if removed > 0 {
		log.Printf("[ConversationTracker-Cleanup] 清理 %d 个过期对话, 剩余 %d", removed, len(ct.conversations))
	}
}

func (ct *ConversationTracker) removeConversation(id string, conv *Conversation) {
	delete(ct.conversations, id)

	compositeKey := conv.Kind + ":" + conv.RawUserID
	if ct.userMapping[compositeKey] == id {
		delete(ct.userMapping, compositeKey)
	}

	for sessID, convID := range ct.sessionMapping {
		if convID == id {
			delete(ct.sessionMapping, sessID)
		}
	}
}

func maskUserID(userID string) string {
	if len(userID) <= 8 {
		return userID[:1] + "***"
	}
	return userID[:4] + "***" + userID[len(userID)-4:]
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
