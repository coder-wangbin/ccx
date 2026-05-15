package conversation

import (
	"testing"
	"time"
)

func TestConversationTracker_Track(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user123", "claude-sonnet-4-20250514", 0, "primary", "")

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}

	conv := convs[0]
	if conv.Kind != "chat" {
		t.Errorf("expected kind=chat, got %s", conv.Kind)
	}
	if conv.RequestCount != 1 {
		t.Errorf("expected requestCount=1, got %d", conv.RequestCount)
	}
	if conv.ChannelName != "primary" {
		t.Errorf("expected channelName=primary, got %s", conv.ChannelName)
	}
	if conv.LastModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected lastModel=claude-sonnet-4-20250514, got %s", conv.LastModel)
	}
}

func TestConversationTracker_TrackMultipleRequests(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user123", "claude-sonnet-4-20250514", 0, "primary", "")
	ct.Track("chat", "user123", "claude-opus-4-20250514", 1, "backup", "")

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation (same user), got %d", len(convs))
	}

	conv := convs[0]
	if conv.RequestCount != 2 {
		t.Errorf("expected requestCount=2, got %d", conv.RequestCount)
	}
	if len(conv.Models) != 2 {
		t.Errorf("expected 2 models, got %d", len(conv.Models))
	}
	if conv.CurrentChannel != 1 {
		t.Errorf("expected currentChannel=1, got %d", conv.CurrentChannel)
	}
}

func TestConversationTracker_DifferentUsers(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user1", "model-a", 0, "ch1", "")
	ct.Track("chat", "user2", "model-b", 1, "ch2", "")

	convs := ct.GetActiveConversations("")
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}
}

func TestConversationTracker_KindFilter(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user1", "model-a", 0, "ch1", "")
	ct.Track("messages", "user2", "model-b", 1, "ch2", "")

	chatConvs := ct.GetActiveConversations("chat")
	if len(chatConvs) != 1 {
		t.Errorf("expected 1 chat conversation, got %d", len(chatConvs))
	}

	msgConvs := ct.GetActiveConversations("messages")
	if len(msgConvs) != 1 {
		t.Errorf("expected 1 messages conversation, got %d", len(msgConvs))
	}
}

func TestConversationTracker_SessionID(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("responses", "user1", "model-a", 0, "ch1", "sess_abc123")
	ct.Track("responses", "user1", "model-a", 0, "ch1", "sess_abc123")

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	if convs[0].ID != "sess_abc123" {
		t.Errorf("expected ID=sess_abc123, got %s", convs[0].ID)
	}
	if convs[0].RequestCount != 2 {
		t.Errorf("expected requestCount=2, got %d", convs[0].RequestCount)
	}
}

func TestConversationTracker_EmptyUserID(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "", "model-a", 0, "ch1", "")

	convs := ct.GetActiveConversations("")
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations for empty userID, got %d", len(convs))
	}
}

func TestConversationTracker_MaskUserID(t *testing.T) {
	result := maskUserID("short")
	if result != "s***" {
		t.Errorf("expected s***, got %s", result)
	}

	result = maskUserID("longUserIdentifier")
	if result != "long***fier" {
		t.Errorf("expected long***fier, got %s", result)
	}
}
