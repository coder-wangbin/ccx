package handlers

import (
	"encoding/json"
	"testing"
)

func TestBuildChatProbeBody_ReasoningEffortUsesProviderCompatibleValue(t *testing.T) {
	bodyBytes := buildChatProbeBody("test-model")

	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("unmarshal body failed: %v", err)
	}
	if body["reasoning_effort"] != "low" {
		t.Fatalf("reasoning_effort=%v, want low", body["reasoning_effort"])
	}
}

func TestBuildMessagesProbeBody_ClaudeOpus48UsesSystemMessage(t *testing.T) {
	bodyBytes := buildMessagesProbeBody(capabilityProbeModelClaudeOpus48)

	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("unmarshal body failed: %v", err)
	}
	messages, ok := body["messages"].([]interface{})
	if !ok {
		t.Fatalf("messages=%T, want []interface{}", body["messages"])
	}
	if len(messages) != 4 {
		t.Fatalf("messages len=%d, want 4", len(messages))
	}
	middle, ok := messages[2].(map[string]interface{})
	if !ok {
		t.Fatalf("middle message=%T, want map[string]interface{}", messages[2])
	}
	if middle["role"] != "system" {
		t.Fatalf("middle role=%v, want system", middle["role"])
	}
	if _, ok := middle["content"].(string); !ok {
		t.Fatalf("middle content=%T, want string", middle["content"])
	}
}

func TestBuildMessagesProbeBody_LegacyModelsKeepTopLevelSystem(t *testing.T) {
	bodyBytes := buildMessagesProbeBody("claude-opus-4-7")

	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("unmarshal body failed: %v", err)
	}
	messages, ok := body["messages"].([]interface{})
	if !ok {
		t.Fatalf("messages=%T, want []interface{}", body["messages"])
	}
	for _, raw := range messages {
		msg, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if msg["role"] == "system" {
			t.Fatalf("legacy probe should not include system message: %#v", messages)
		}
	}
	system, ok := body["system"].([]interface{})
	if !ok {
		t.Fatalf("system=%T, want []interface{}", body["system"])
	}
	if len(system) != 2 {
		t.Fatalf("system len=%d, want 2", len(system))
	}
}
