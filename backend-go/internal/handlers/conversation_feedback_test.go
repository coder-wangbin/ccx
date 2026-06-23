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

func TestAddConversationFeedback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := conversation.NewConversationTracker(time.Hour, 24*time.Hour, "")
	tracker.Track("responses", "session-1", "gpt-5.5", 7, "codex-main", "session-1", "build the feature", 1, "main")

	convs := tracker.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected one conversation, got %d", len(convs))
	}

	deps := &ConversationHandlerDeps{Tracker: tracker}
	body, _ := json.Marshal(map[string]string{"message": "prefer low latency channel for subagents"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: convs[0].ID}}
	c.Request = httptest.NewRequest("POST", "/api/conversations/"+convs[0].ID+"/feedback", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	AddConversationFeedback(deps)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	updated, ok := tracker.GetConversation(convs[0].ID)
	if !ok {
		t.Fatal("conversation missing after feedback")
	}
	if updated.LatestFeedback != "prefer low latency channel for subagents" {
		t.Fatalf("LatestFeedback = %q", updated.LatestFeedback)
	}
	if updated.LatestFeedbackAt == nil || updated.LatestFeedbackAt.IsZero() {
		t.Fatal("LatestFeedbackAt should be set")
	}
}
