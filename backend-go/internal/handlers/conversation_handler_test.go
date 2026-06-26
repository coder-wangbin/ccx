package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/conversation"
	"github.com/gin-gonic/gin"
)

func TestSetConversationOverride_SubagentOnlyAndClear(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := conversation.NewConversationTracker(1*time.Hour, 24*time.Hour, "")
	defer tracker.Stop()

	tracker.Track("chat", "user1", "gpt-test", 0, "primary", "", "提交", 1, "")
	conv, ok := tracker.GetConversationByUser("chat", "user1")
	if !ok {
		t.Fatal("expected conversation to be tracked")
	}

	overrideManager := conversation.NewOverrideManager(30 * time.Minute)
	defer overrideManager.Stop()

	deps := &ConversationHandlerDeps{
		Tracker:         tracker,
		OverrideManager: overrideManager,
	}

	postOverride := func(body map[string]interface{}) *httptest.ResponseRecorder {
		reqJSON, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request failed: %v", err)
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: conv.ID}}
		c.Request = httptest.NewRequest("POST", "/api/conversations/"+conv.ID+"/override", bytes.NewReader(reqJSON))
		c.Request.Header.Set("Content-Type", "application/json")
		SetConversationOverride(deps)(c)
		return w
	}

	w := postOverride(map[string]interface{}{
		"sequence": []interface{}{},
		"subagentSequence": []map[string]interface{}{
			{"channelIndex": 2, "channelName": "subagent"},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected subagent-only override status 200, got %d: %s", w.Code, w.Body.String())
	}

	if _, ok := overrideManager.GetOverrideForUser("chat", "user1"); ok {
		t.Fatal("expected subagent-only override not to affect main conversation")
	}
	if _, ok := overrideManager.GetOverrideForUserWithRole("chat", "user1", "subagent"); !ok {
		t.Fatal("expected subagent role override to exist")
	}

	w = postOverride(map[string]interface{}{
		"sequence":              []interface{}{},
		"clearSubagentSequence": true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected clear subagent override status 200, got %d: %s", w.Code, w.Body.String())
	}

	if _, ok := overrideManager.GetOverrideForUserWithRole("chat", "user1", "subagent"); ok {
		t.Fatal("expected subagent override to be cleared")
	}
	if _, ok := overrideManager.GetOverride(conv.ID); ok {
		t.Fatal("expected subagent-only override snapshot to be removed")
	}
}
