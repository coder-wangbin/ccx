package converters

import (
	"context"
	"testing"

	"encoding/json"
	"github.com/tidwall/gjson"
)

func TestConvertOpenAIChatToResponsesNonStream(t *testing.T) {
	ctx := context.Background()

	// 模拟 OpenAI Chat Completions 非流式响应
	chatResponse := `{
		"id": "chatcmpl-789",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello! How can I help you today?"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 8,
			"total_tokens": 18
		}
	}`

	originalReq := []byte(`{"model":"gpt-4o","input":"Hi","instructions":"Be helpful"}`)

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-4o", originalReq, nil, []byte(chatResponse), nil)

	// 解析结果
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// 验证基本字段
	if resp["object"] != "response" {
		t.Errorf("object should be response, got %v", resp["object"])
	}
	if resp["status"] != "completed" {
		t.Errorf("status should be completed, got %v", resp["status"])
	}

	// 验证 output
	output, ok := resp["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("output should have items")
	}

	msgItem := output[0].(map[string]interface{})
	if msgItem["type"] != "message" {
		t.Errorf("first output item should be message, got %v", msgItem["type"])
	}

	// 验证 usage
	usage, ok := resp["usage"].(map[string]interface{})
	if !ok {
		t.Fatal("usage should exist")
	}
	if usage["input_tokens"].(float64) != 10 {
		t.Errorf("input_tokens should be 10, got %v", usage["input_tokens"])
	}
	if usage["output_tokens"].(float64) != 8 {
		t.Errorf("output_tokens should be 8, got %v", usage["output_tokens"])
	}
}

func TestConvertOpenAIChatToResponsesNonStream_VLLMReasoning(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id": "chatcmpl-vllm",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "glm-5.2",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"reasoning": "vllm reasoning",
				"content": "final answer"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 8,
			"total_tokens": 18
		}
	}`

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "glm-5.2", []byte(`{"model":"glm-5.2","input":"Hi"}`), nil, []byte(chatResponse), nil)
	parsed := gjson.Parse(result)

	if got := parsed.Get(`output.#(type=="reasoning").summary.0.text`).String(); got != "vllm reasoning" {
		t.Fatalf("reasoning summary = %q, want vllm reasoning; result=%s", got, result)
	}
	if got := parsed.Get(`output.#(type=="message").content.0.text`).String(); got != "final answer" {
		t.Fatalf("message text = %q, want final answer; result=%s", got, result)
	}
}

func TestConvertOpenAIChatToResponsesNonStream_ToolCalls(t *testing.T) {
	ctx := context.Background()

	// 模拟带 tool_calls 的响应
	chatResponse := `{
		"id": "chatcmpl-tool",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [
					{
						"id": "call_xyz",
						"type": "function",
						"function": {
							"name": "search",
							"arguments": "{\"query\": \"test\"}"
						}
					}
				]
			},
			"finish_reason": "tool_calls"
		}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}
	}`

	originalReq := []byte(`{"model":"gpt-4o","input":"Search for test"}`)

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-4o", originalReq, nil, []byte(chatResponse), nil)

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	output, ok := resp["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("output should have items")
	}

	// 查找 function_call item
	var funcItem map[string]interface{}
	for _, item := range output {
		itemMap := item.(map[string]interface{})
		if itemMap["type"] == "function_call" {
			funcItem = itemMap
			break
		}
	}

	if funcItem == nil {
		t.Fatal("should have function_call item")
	}

	if funcItem["name"] != "search" {
		t.Errorf("function name should be search, got %v", funcItem["name"])
	}
	if funcItem["call_id"] != "call_xyz" {
		t.Errorf("call_id should be call_xyz, got %v", funcItem["call_id"])
	}
}

func TestConvertOpenAIChatToResponsesNonStream_CustomToolCall(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id": "chatcmpl-custom",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [{
					"id": "call_patch",
					"type": "function",
					"function": {
						"name": "apply_patch_add_file",
						"arguments": "{\"path\":\"docs/test.md\",\"content\":\"# Test\\n\"}"
					}
				}]
			},
			"finish_reason": "tool_calls"
		}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}
	}`
	originalReq := []byte(`{"model":"gpt-4o","input":"edit","tools":[{"type":"custom","name":"apply_patch"}],"transformer_metadata":{"codex_tool_compat_enabled":true}}`)

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-4o", originalReq, nil, []byte(chatResponse), nil)

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatal(err)
	}
	output := resp["output"].([]interface{})
	item := output[0].(map[string]interface{})
	if item["type"] != "custom_tool_call" {
		t.Fatalf("item type = %v, result = %s", item["type"], result)
	}
	if item["name"] != "apply_patch" {
		t.Fatalf("name = %v", item["name"])
	}
	if _, ok := item["output"]; ok {
		t.Fatalf("custom_tool_call should not contain output: %#v", item)
	}
	if got := item["input"].(string); got != "*** Begin Patch\n*** Add File: docs/test.md\n+# Test\n*** End Patch" {
		t.Fatalf("input = %q", got)
	}
}

func TestConvertOpenAIChatToResponsesNonStream_ToolSearchCall(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id": "chatcmpl-tool-search",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": null,
				"tool_calls": [{
					"id": "call_search",
					"type": "function",
					"function": {
						"name": "tool_search",
						"arguments": "{\"query\":\"sub-agent\"}"
					}
				}]
			},
			"finish_reason": "tool_calls"
		}]
	}`
	originalReq := []byte(`{"model":"gpt-4o","input":"find tools","tools":[{"type":"tool_search","execution":"client","parameters":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}}],"transformer_metadata":{"codex_tool_compat_enabled":true}}`)

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-4o", originalReq, nil, []byte(chatResponse), nil)

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatal(err)
	}
	output := resp["output"].([]interface{})
	item := output[0].(map[string]interface{})
	if item["type"] != "tool_search_call" {
		t.Fatalf("item type = %v, result = %s", item["type"], result)
	}
	if _, ok := item["name"]; ok {
		t.Fatalf("tool_search_call should not carry custom tool name: %#v", item)
	}
	if item["execution"] != "client" {
		t.Fatalf("execution = %v", item["execution"])
	}
	args, ok := item["arguments"].(map[string]interface{})
	if !ok {
		t.Fatalf("arguments should be object: %#v", item["arguments"])
	}
	if args["query"] != "sub-agent" {
		t.Fatalf("arguments = %#v", args)
	}
}

func TestConvertOpenAIChatToResponsesNonStream_ClaudeCacheTTLTotalFallback(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id":"chatcmpl-claude-cache",
		"object":"chat.completion",
		"created":1234567890,
		"model":"claude",
		"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{
			"input_tokens":100,
			"output_tokens":20,
			"cache_read_input_tokens":30,
			"cache_creation_5m_input_tokens":7,
			"cache_creation_1h_input_tokens":3
		}
	}`

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "claude", []byte(`{"model":"claude","input":"hi"}`), nil, []byte(chatResponse), nil)
	if got := gjson.Get(result, "usage.total_tokens").Int(); got != 160 {
		t.Fatalf("usage.total_tokens = %d, want 160", got)
	}
}

func TestConvertOpenAIChatToResponsesNonStream_OpenAICacheDetailsNormalizesInput(t *testing.T) {
	ctx := context.Background()
	chatResponse := `{
		"id":"chatcmpl-openai-cache",
		"object":"chat.completion",
		"created":1234567890,
		"model":"gpt-5.5",
		"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{
			"prompt_tokens":38451,
			"completion_tokens":1275,
			"total_tokens":39726,
			"prompt_tokens_details":{"cached_tokens":36608}
		}
	}`

	result := ConvertOpenAIChatToResponsesNonStream(ctx, "gpt-5.5", []byte(`{"model":"gpt-5.5","input":"hi"}`), nil, []byte(chatResponse), nil)
	if got := gjson.Get(result, "usage.input_tokens").Int(); got != 1843 {
		t.Fatalf("usage.input_tokens = %d, want 1843", got)
	}
	if got := gjson.Get(result, "usage.total_tokens").Int(); got != 39726 {
		t.Fatalf("usage.total_tokens = %d, want 39726", got)
	}
	if gjson.Get(result, "usage.cache_read_input_tokens").Exists() {
		t.Fatalf("usage.cache_read_input_tokens should not be emitted: %s", result)
	}
	if got := gjson.Get(result, "usage.input_tokens_details.cached_tokens").Int(); got != 36608 {
		t.Fatalf("usage.input_tokens_details.cached_tokens = %d, want 36608", got)
	}
}

// ============== <think> 标签提取测试 ==============
