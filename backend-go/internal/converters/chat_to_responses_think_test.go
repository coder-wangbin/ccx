package converters

import (
	"context"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// TestThinkTag_StreamCrossChunkBoundary 验证跨 chunk 边界的 <think>...</think> 能被正确切分。
func TestThinkTag_StreamCrossChunkBoundary(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<thi"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"nk>思考"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"内容</thi"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"nk>正文"}}]}`,
		`data: {"id":"cc-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "思考内容" {
		t.Errorf("reasoning = %q, want %q", reasoning, "思考内容")
	}
	if text != "正文" {
		t.Errorf("text = %q, want %q", text, "正文")
	}
	joined := strings.Join(events, "\n")
	if strings.Contains(joined, "<think>") || strings.Contains(joined, "</think>") {
		t.Errorf("events should not contain raw think tags: %s", joined)
	}
}

// TestThinkTag_StreamSingleChunk 验证单个 chunk 内含完整 <think>...</think>正文的拆分。
func TestThinkTag_StreamSingleChunk(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<think>full-think</think>answer"}}]}`,
		`data: {"id":"cc-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "full-think" {
		t.Errorf("reasoning = %q, want %q", reasoning, "full-think")
	}
	if text != "answer" {
		t.Errorf("text = %q, want %q", text, "answer")
	}
}

// TestThinkTag_StreamUnclosedFallback 验证未闭合 <think> 会被视为推理内容兜底刷出。
func TestThinkTag_StreamUnclosedFallback(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-3","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<think>仅有思考"}}]}`,
		`data: {"id":"cc-3","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "仅有思考" {
		t.Errorf("reasoning = %q, want %q", reasoning, "仅有思考")
	}
	if text != "" {
		t.Errorf("text should be empty, got %q", text)
	}
}

// TestThinkTag_StreamPlainTextWithLiteralTag 验证正文中误用 <think> 不应被剥离。
func TestThinkTag_StreamPlainTextWithLiteralTag(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-4","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello <think>not-reasoning</think>"}}]}`,
		`data: {"id":"cc-4","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "" {
		t.Errorf("reasoning should be empty when <think> is not at the start, got %q", reasoning)
	}
	if text != "Hello <think>not-reasoning</think>" {
		t.Errorf("text = %q, want %q", text, "Hello <think>not-reasoning</think>")
	}
}

// TestThinkTag_NonStreamSplit 验证非流式接口能从 content 头部提取 <think>。
func TestThinkTag_NonStreamSplit(t *testing.T) {
	ctx := context.Background()
	body := `{
		"id":"cc-ns-1",
		"object":"chat.completion",
		"created":1,
		"model":"MiniMax-M2.7",
		"choices":[{"index":0,"message":{"role":"assistant","content":"<think>think-here</think>answer-here"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}
	}`
	result := ConvertOpenAIChatToResponsesNonStream(ctx, "MiniMax-M2.7", []byte(`{"model":"MiniMax-M2.7","input":"hi"}`), nil, []byte(body), nil)

	outputs := gjson.Get(result, "output").Array()
	if len(outputs) < 2 {
		t.Fatalf("expected at least 2 outputs (reasoning + message), got %d: %s", len(outputs), result)
	}
	if outputs[0].Get("type").String() != "reasoning" {
		t.Errorf("first output type = %q, want reasoning", outputs[0].Get("type").String())
	}
	if got := outputs[0].Get("summary.0.text").String(); got != "think-here" {
		t.Errorf("reasoning text = %q, want %q", got, "think-here")
	}
	if outputs[1].Get("type").String() != "message" {
		t.Errorf("second output type = %q, want message", outputs[1].Get("type").String())
	}
	if got := outputs[1].Get("content.0.text").String(); got != "answer-here" {
		t.Errorf("message text = %q, want %q", got, "answer-here")
	}
}

// TestThinkTag_StreamThenToolCalls 验证 <think>...</think> 后紧跟 tool_calls 的 SSE：
//
//   - reasoning 内容应正确进入 reasoning_summary_text.delta
//   - function_call 的 output_index 计算应正确（reasoning 占位后的 fc 索引应 +1）
//   - 不应有任何 response.output_text.delta（content 全部被 think 吸收）
//   - 不应留下空的 message item
func TestThinkTag_StreamThenToolCalls(t *testing.T) {
	chunks := []string{
		// 第一帧：role + 完整 <think>...</think>
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"<think>let me call a tool</think>"}}]}`,
		// 第二帧：tool_calls 的开头（id + name）
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_tt","type":"function","function":{"name":"lookup","arguments":""}}]}}]}`,
		// 第三帧：tool_calls 的 arguments 分片
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":\"x\"}"}}]}}]}`,
		// 第四帧：finish
		`data: {"id":"cc-tt","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "let me call a tool" {
		t.Errorf("reasoning = %q, want %q", reasoning, "let me call a tool")
	}
	if text != "" {
		t.Errorf("text should be empty when content is entirely <think>, got %q", text)
	}

	// 解析关键事件
	var (
		reasoningAddedIdx     = -1
		funcCallAddedIdx      = -1
		funcCallArgsDelta     string
		funcCallOutputIndex   = -1
		funcCallItemID        string
		funcCallArgsOutputIdx = -1
		sawOutputTextDelta    = false
		sawEmptyMessage       = false
	)

	for _, ev := range events {
		for _, line := range strings.Split(ev, "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			typ := gjson.Get(payload, "type").String()
			switch typ {
			case "response.output_item.added":
				itemType := gjson.Get(payload, "item.type").String()
				outIdx := int(gjson.Get(payload, "output_index").Int())
				if itemType == "reasoning" && reasoningAddedIdx < 0 {
					reasoningAddedIdx = outIdx
				}
				if itemType == "function_call" && funcCallAddedIdx < 0 {
					funcCallAddedIdx = outIdx
					funcCallItemID = gjson.Get(payload, "item.id").String()
				}
				if itemType == "message" {
					// 状态机吸收完 <think> 后正文为空 → 不应发射 message item
					sawEmptyMessage = true
				}
			case "response.output_text.delta":
				sawOutputTextDelta = true
			case "response.function_call_arguments.delta":
				funcCallArgsDelta += gjson.Get(payload, "delta").String()
				funcCallArgsOutputIdx = int(gjson.Get(payload, "output_index").Int())
			case "response.output_item.done":
				if gjson.Get(payload, "item.type").String() == "function_call" && funcCallOutputIndex < 0 {
					funcCallOutputIndex = int(gjson.Get(payload, "output_index").Int())
				}
			}
		}
	}

	if reasoningAddedIdx != 0 {
		t.Errorf("reasoning output_index = %d, want 0", reasoningAddedIdx)
	}
	if funcCallAddedIdx != 1 {
		t.Errorf("function_call output_index = %d, want 1 (reasoning + fc)", funcCallAddedIdx)
	}
	if funcCallArgsOutputIdx != 1 {
		t.Errorf("function_call_arguments.delta output_index = %d, want 1", funcCallArgsOutputIdx)
	}
	if funcCallArgsDelta != `{"q":"x"}` {
		t.Errorf("function arguments = %q, want %q", funcCallArgsDelta, `{"q":"x"}`)
	}
	if funcCallItemID == "" {
		t.Error("function_call item.id should not be empty")
	}
	if sawOutputTextDelta {
		t.Error("should not emit response.output_text.delta when content is fully absorbed by <think>")
	}
	if sawEmptyMessage {
		t.Error("should not emit empty message item when content is fully absorbed by <think>")
	}
}

// TestThinkTag_StreamWithLeadingWhitespace 验证流式开头有空白字符时，<think> 仍能被正确提取。
func TestThinkTag_StreamWithLeadingWhitespace(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-ws-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"  \n  <thi"}}]}`,
		`data: {"id":"cc-ws-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"nk>思考内容</think>正文"}}]}`,
		`data: {"id":"cc-ws-1","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "思考内容" {
		t.Errorf("reasoning = %q, want %q", reasoning, "思考内容")
	}
	if text != "正文" {
		t.Errorf("text = %q, want %q", text, "正文")
	}
}

// TestThinkTag_StreamWithLeadingWhitespaceNoThink 验证流式开头有空白字符但最终没有 <think> 时，空白字符能被正确作为正文输出。
func TestThinkTag_StreamWithLeadingWhitespaceNoThink(t *testing.T) {
	chunks := []string{
		`data: {"id":"cc-ws-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"role":"assistant","content":"  \n  "}}]}`,
		`data: {"id":"cc-ws-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{"content":"正文"}}]}`,
		`data: {"id":"cc-ws-2","object":"chat.completion.chunk","created":1,"model":"MiniMax-M2.7","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}

	events := runThinkStream(t, chunks)
	reasoning, text := collectThinkStreamText(events)

	if reasoning != "" {
		t.Errorf("reasoning should be empty, got %q", reasoning)
	}
	if text != "  \n  正文" {
		t.Errorf("text = %q, want %q", text, "  \n  正文")
	}
}

// TestExtractThinkTag_TableDriven 覆盖共享函数 extractThinkTag 的核心分支。
func TestExtractThinkTag_TableDriven(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantText  string
		wantThink string
		wantHas   bool
	}{
		{"no think", "plain answer", "plain answer", "", false},
		{"think at start", "<think>t</think>a", "a", "t", true},
		{"think with leading whitespace", "  \n<think>t</think>a", "a", "t", true},
		{"unclosed think", "<think>only thinking", "", "only thinking", true},
		{"think in middle", "head <think>x</think>tail", "head <think>x</think>tail", "", false},
		{"empty input", "", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotText, gotThink, gotHas := extractThinkTag(tc.input)
			if gotText != tc.wantText || gotThink != tc.wantThink || gotHas != tc.wantHas {
				t.Errorf("extractThinkTag(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.input, gotText, gotThink, gotHas, tc.wantText, tc.wantThink, tc.wantHas)
			}
		})
	}
}
