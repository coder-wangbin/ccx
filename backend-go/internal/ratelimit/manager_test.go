package ratelimit

import (
	"testing"
	"time"
)

func TestManager_GetOrCreate_New(t *testing.T) {
	m := NewManager()
	l := m.GetOrCreate("messages", 0, Config{RPM: 60})
	if l == nil {
		t.Fatal("expected non-nil limiter")
	}
	s := l.Status(time.Now())
	if s.MaxRequests != 60 {
		t.Fatalf("maxRequests = %v, want 60", s.MaxRequests)
	}
}

func TestManager_GetOrCreate_Existing(t *testing.T) {
	m := NewManager()
	l1 := m.GetOrCreate("messages", 0, Config{RPM: 60})
	l2 := m.GetOrCreate("messages", 0, Config{RPM: 120})
	if l1 != l2 {
		t.Fatal("expected same limiter instance for same key")
	}
	// Verify updated config
	s := l2.Status(time.Now())
	if s.MaxRequests != 120 {
		t.Fatalf("maxRequests = %v, want 120", s.MaxRequests)
	}
}

func TestManager_Get(t *testing.T) {
	m := NewManager()
	if m.Get("messages", 0) != nil {
		t.Fatal("expected nil for non-existent key")
	}
	m.GetOrCreate("messages", 0, Config{RPM: 60})
	if m.Get("messages", 0) == nil {
		t.Fatal("expected non-nil after create")
	}
}

func TestManager_Remove(t *testing.T) {
	m := NewManager()
	m.GetOrCreate("messages", 0, Config{RPM: 60})
	m.Remove("messages", 0)
	if m.Get("messages", 0) != nil {
		t.Fatal("expected nil after remove")
	}
}

func TestManager_UpdateAll(t *testing.T) {
	m := NewManager()
	m.GetOrCreate("messages", 0, Config{RPM: 60})
	m.GetOrCreate("chat", 1, Config{RPM: 30})

	m.UpdateAll(func(apiType string, channelIndex int) (Config, bool) {
		if apiType == "messages" {
			return Config{RPM: 120}, true
		}
		return Config{}, false
	})

	l0 := m.Get("messages", 0)
	if l0 == nil {
		t.Fatal("messages limiter disappeared")
	}
	if s := l0.Status(time.Now()); s.MaxRequests != 120 {
		t.Fatalf("messages maxRequests = %v, want 120", s.MaxRequests)
	}

	// chat unchanged
	l1 := m.Get("chat", 1)
	if l1 == nil {
		t.Fatal("chat limiter disappeared")
	}
	if s := l1.Status(time.Now()); s.MaxRequests != 30 {
		t.Fatalf("chat maxRequests = %v, want 30", s.MaxRequests)
	}
}

func TestManager_GetStatus(t *testing.T) {
	m := NewManager()
	m.GetOrCreate("messages", 0, Config{RPM: 60, MaxConcurrent: 5})
	m.GetOrCreate("chat", 1, Config{RPM: 30})

	statuses := m.GetStatus(time.Now())
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
}

func TestManager_DifferentChannelTypes(t *testing.T) {
	m := NewManager()

	kinds := []struct {
		apiType      string
		channelIndex int
	}{
		{"messages", 0},
		{"chat", 0},
		{"responses", 0},
		{"gemini", 0},
		{"images", 0},
	}

	for _, k := range kinds {
		m.GetOrCreate(k.apiType, k.channelIndex, Config{RPM: 60})
	}

	for _, k := range kinds {
		if m.Get(k.apiType, k.channelIndex) == nil {
			t.Fatalf("missing limiter for %s[%d]", k.apiType, k.channelIndex)
		}
	}
}

func TestManager_MultipleChannelsSameType(t *testing.T) {
	m := NewManager()
	m.GetOrCreate("messages", 0, Config{RPM: 60})
	m.GetOrCreate("messages", 1, Config{RPM: 120})
	m.GetOrCreate("messages", 2, Config{RPM: 30})

	if m.Get("messages", 0) == m.Get("messages", 1) {
		t.Fatal("different indices should have different limiters")
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		key      string
		wantType string
		wantIdx  int
	}{
		{"messages:0", "messages", 0},
		{"chat:3", "chat", 3},
		{"responses:10", "responses", 10},
		{"unknown", "unknown", 0},
	}
	for _, tt := range tests {
		apiType, idx := parseKey(tt.key)
		if apiType != tt.wantType || idx != tt.wantIdx {
			t.Errorf("parseKey(%q) = (%q, %d), want (%q, %d)",
				tt.key, apiType, idx, tt.wantType, tt.wantIdx)
		}
	}
}
