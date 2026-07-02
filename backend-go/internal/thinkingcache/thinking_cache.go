package thinkingcache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/utils"
)

const (
	defaultTTL        = 2 * time.Hour
	defaultMaxEntries = 512
)

type cacheEntry struct {
	Thinking  string
	ExpiresAt time.Time
	UpdatedAt time.Time
}

type cacheStore struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

var globalStore = &cacheStore{entries: make(map[string]cacheEntry)}

// ResetForTest clears the process-local cache.
func ResetForTest() {
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	globalStore.entries = make(map[string]cacheEntry)
}

// ShouldTrackClaudeThinking returns true for strict DeepSeek Claude-compatible channels.
func ShouldTrackClaudeThinking(upstream *config.UpstreamConfig, bodyBytes []byte) bool {
	return isDeepSeekClaudeTarget(upstream, bodyBytes)
}

// InjectCachedClaudeThinking prepends cached thinking blocks to assistant history
// only when the request is in Claude thinking mode and the assistant content
// fingerprint has a previous exact cache hit.
func InjectCachedClaudeThinking(bodyBytes []byte, sessionID string, upstream *config.UpstreamConfig) ([]byte, int) {
	if strings.TrimSpace(sessionID) == "" || !isDeepSeekClaudeTarget(upstream, bodyBytes) {
		return bodyBytes, 0
	}

	data, ok := decodeObject(bodyBytes)
	if !ok || !claudeThinkingRequested(data) {
		return bodyBytes, 0
	}

	messages, ok := data["messages"].([]interface{})
	if !ok {
		return bodyBytes, 0
	}

	injected := 0
	for _, rawMsg := range messages {
		msg, ok := rawMsg.(map[string]interface{})
		if !ok {
			continue
		}
		if role, _ := msg["role"].(string); role != "assistant" {
			continue
		}

		content, exists := msg["content"]
		if !exists || assistantContentHasThinking(content) {
			continue
		}

		thinking, ok := LookupClaudeThinkingForContent(sessionID, content)
		if !ok {
			continue
		}

		switch typed := content.(type) {
		case []interface{}:
			next := make([]interface{}, 0, len(typed)+1)
			next = append(next, thinkingBlock(thinking))
			next = append(next, typed...)
			msg["content"] = next
		case string:
			msg["content"] = []interface{}{
				thinkingBlock(thinking),
				map[string]interface{}{"type": "text", "text": typed},
			}
		default:
			continue
		}
		injected++
	}

	if injected == 0 {
		return bodyBytes, 0
	}

	data["messages"] = messages
	nextBytes, err := utils.MarshalJSONNoEscape(data)
	if err != nil {
		return bodyBytes, 0
	}
	return nextBytes, injected
}

func thinkingBlock(thinking string) map[string]interface{} {
	return map[string]interface{}{
		"type":     "thinking",
		"thinking": thinking,
	}
}

func claudeThinkingRequested(data map[string]interface{}) bool {
	thinking, ok := data["thinking"].(map[string]interface{})
	if !ok {
		return false
	}
	thinkingType, _ := thinking["type"].(string)
	switch strings.ToLower(strings.TrimSpace(thinkingType)) {
	case "adaptive", "enabled":
		return true
	default:
		return false
	}
}

func isDeepSeekClaudeTarget(upstream *config.UpstreamConfig, bodyBytes []byte) bool {
	if upstream == nil || upstream.ServiceType != "claude" {
		return false
	}

	parts := []string{upstream.BaseURL, upstream.GetEffectiveBaseURL(), upstream.Name, upstream.Website}
	parts = append(parts, upstream.BaseURLs...)
	if strings.Contains(strings.ToLower(strings.Join(parts, " ")), "deepseek") {
		return true
	}

	data, ok := decodeObject(bodyBytes)
	if !ok {
		return false
	}
	model, _ := data["model"].(string)
	return strings.Contains(strings.ToLower(model), "deepseek")
}

func decodeObject(bodyBytes []byte) (map[string]interface{}, bool) {
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.UseNumber()

	var data map[string]interface{}
	if err := decoder.Decode(&data); err != nil {
		return nil, false
	}
	return data, true
}

// StoreClaudeThinkingForContent stores thinking by session and assistant content fingerprint.
func StoreClaudeThinkingForContent(sessionID string, content interface{}, thinking string) bool {
	if strings.TrimSpace(sessionID) == "" || !isRealThinking(thinking) {
		return false
	}

	fingerprint := FingerprintClaudeAssistantContent(content)
	if fingerprint == "" {
		return false
	}

	globalStore.store(sessionID, fingerprint, thinking)
	return true
}

// LookupClaudeThinkingForContent returns cached thinking for the assistant content fingerprint.
func LookupClaudeThinkingForContent(sessionID string, content interface{}) (string, bool) {
	if strings.TrimSpace(sessionID) == "" {
		return "", false
	}
	fingerprint := FingerprintClaudeAssistantContent(content)
	if fingerprint == "" {
		return "", false
	}
	return globalStore.lookup(sessionID, fingerprint)
}

func (s *cacheStore) store(sessionID, fingerprint, thinking string) {
	now := time.Now()
	key := cacheKey(sessionID, fingerprint)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.evictExpiredLocked(now)
	if _, exists := s.entries[key]; !exists {
		for len(s.entries) >= defaultMaxEntries {
			s.evictOldestLocked()
		}
	}

	s.entries[key] = cacheEntry{
		Thinking:  thinking,
		ExpiresAt: now.Add(defaultTTL),
		UpdatedAt: now,
	}
}

func (s *cacheStore) lookup(sessionID, fingerprint string) (string, bool) {
	now := time.Now()
	key := cacheKey(sessionID, fingerprint)

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[key]
	if !ok {
		return "", false
	}
	if now.After(entry.ExpiresAt) {
		delete(s.entries, key)
		return "", false
	}
	return entry.Thinking, true
}

func (s *cacheStore) evictExpiredLocked(now time.Time) {
	for key, entry := range s.entries {
		if now.After(entry.ExpiresAt) {
			delete(s.entries, key)
		}
	}
}

func (s *cacheStore) evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time
	for key, entry := range s.entries {
		if oldestKey == "" || entry.UpdatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.UpdatedAt
		}
	}
	if oldestKey != "" {
		delete(s.entries, oldestKey)
	}
}

func cacheKey(sessionID, fingerprint string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(sum[:]) + ":" + fingerprint
}

func isRealThinking(thinking string) bool {
	return strings.TrimSpace(thinking) != ""
}

// FingerprintClaudeAssistantContent fingerprints assistant content after removing thinking blocks.
func FingerprintClaudeAssistantContent(content interface{}) string {
	normalized := normalizeAssistantContent(content)
	if len(normalized) == 0 {
		return ""
	}

	raw, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func normalizeAssistantContent(content interface{}) []interface{} {
	switch typed := content.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []interface{}{map[string]interface{}{"type": "text", "text": typed}}
	case []interface{}:
		normalized := make([]interface{}, 0, len(typed))
		for _, rawBlock := range typed {
			block, ok := normalizeAssistantBlock(rawBlock)
			if ok {
				normalized = append(normalized, block)
			}
		}
		return normalized
	default:
		return nil
	}
}

func normalizeAssistantBlock(rawBlock interface{}) (interface{}, bool) {
	block, ok := rawBlock.(map[string]interface{})
	if !ok {
		return nil, false
	}

	blockType, _ := block["type"].(string)
	blockType = strings.TrimSpace(blockType)
	switch blockType {
	case "", "thinking", "redacted_thinking":
		return nil, false
	case "text":
		text, _ := block["text"].(string)
		if text == "" {
			return nil, false
		}
		return map[string]interface{}{"type": "text", "text": text}, true
	case "tool_use", "server_tool_use":
		normalized := map[string]interface{}{"type": blockType}
		if id, _ := block["id"].(string); id != "" {
			normalized["id"] = id
		}
		if name, _ := block["name"].(string); name != "" {
			normalized["name"] = name
		}
		if input, exists := block["input"]; exists {
			normalized["input"] = normalizeJSONValue(input)
		}
		return normalized, true
	default:
		normalized := make(map[string]interface{}, len(block))
		keys := make([]string, 0, len(block))
		for key := range block {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if shouldSkipFingerprintField(key) {
				continue
			}
			normalized[key] = normalizeJSONValue(block[key])
		}
		if len(normalized) == 0 {
			return nil, false
		}
		return normalized, true
	}
}

func shouldSkipFingerprintField(key string) bool {
	switch key {
	case "thinking", "signature", "cache_control":
		return true
	default:
		return false
	}
}

func normalizeJSONValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			normalized[key] = normalizeJSONValue(value)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, 0, len(typed))
		for _, value := range typed {
			normalized = append(normalized, normalizeJSONValue(value))
		}
		return normalized
	default:
		return typed
	}
}

func assistantContentHasThinking(content interface{}) bool {
	blocks, ok := content.([]interface{})
	if !ok {
		return false
	}
	for _, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]interface{})
		if !ok {
			continue
		}
		blockType, _ := block["type"].(string)
		if blockType != "thinking" && blockType != "redacted_thinking" {
			continue
		}
		thinking, _ := block["thinking"].(string)
		if isRealThinking(thinking) {
			return true
		}
	}
	return false
}
