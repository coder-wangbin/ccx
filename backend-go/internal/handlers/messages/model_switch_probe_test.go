package messages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
)

// 模拟上游：返回空 content（复现 MiMo 在 max_tokens=1 下的真实行为）。
// 若探针未被拦截，空响应会触发 Fuzzy 模式 failover，最终报错。
func newEmptyContentUpstream(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"msg_up","type":"message","role":"assistant","content":[],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":0}}`))
	}))
	t.Cleanup(func() { upstream.Close() })
	return upstream, &calls
}

func TestMessagesHandler_ModelSwitchProbeNonStream(t *testing.T) {
	upstream, calls := newEmptyContentUpstream(t)

	router := newMessagesTestRouter(t, config.UpstreamConfig{
		Name:        "mimo-probe-upstream",
		BaseURL:     upstream.URL,
		APIKeys:     []string{"sk-test"},
		ServiceType: "claude",
		Status:      "active",
	})

	w := performMessagesHandlerRequest(t, router, `{"model":"xiaomi/mimo-v2.5","max_tokens":1,"messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if got := atomic.LoadInt32(calls); got != 0 {
		t.Fatalf("upstream calls = %d, want 0 (probe should be intercepted)", got)
	}

	var resp struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v, body=%s", err, w.Body.String())
	}
	if resp.Type != "message" || resp.Role != "assistant" || resp.Model != "xiaomi/mimo-v2.5" {
		t.Fatalf("unexpected message metadata: %#v", resp)
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != "text" || resp.Content[0].Text != claudeCodeModelSwitchProbeText {
		t.Fatalf("unexpected content: %#v", resp.Content)
	}
	if resp.StopReason != "end_turn" {
		t.Fatalf("stop_reason = %q, want end_turn", resp.StopReason)
	}
}

func TestMessagesHandler_ModelSwitchProbeStream(t *testing.T) {
	upstream, calls := newEmptyContentUpstream(t)

	router := newMessagesTestRouter(t, config.UpstreamConfig{
		Name:        "mimo-probe-stream-upstream",
		BaseURL:     upstream.URL,
		APIKeys:     []string{"sk-test"},
		ServiceType: "claude",
		Status:      "active",
	})

	w := performMessagesHandlerRequest(t, router, `{"model":"xiaomi/mimo-v2.5","max_tokens":1,"stream":true,"messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if got := atomic.LoadInt32(calls); got != 0 {
		t.Fatalf("upstream calls = %d, want 0 (probe should be intercepted)", got)
	}
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
	}

	body := w.Body.String()
	for _, want := range []string{
		"event: message_start",
		"event: content_block_start",
		"event: content_block_delta",
		"event: content_block_stop",
		"event: message_delta",
		"event: message_stop",
		`"text":"` + claudeCodeModelSwitchProbeText + `"`,
		`"stop_reason":"end_turn"`,
		`"model":"xiaomi/mimo-v2.5"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("stream body missing %q:\n%s", want, body)
		}
	}
}

// TestMessagesHandler_ModelSwitchProbeDoesNotInterceptNonProbe 验证非探针请求不被误拦截。
func TestMessagesHandler_ModelSwitchProbeDoesNotInterceptNonProbe(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "max_tokens_gt_1",
			body: `{"model":"xiaomi/mimo-v2.5","max_tokens":10,"messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}]}`,
		},
		{
			name: "with_tools",
			body: `{"model":"xiaomi/mimo-v2.5","max_tokens":1,"tools":[{"name":"foo","description":"x","input_schema":{"type":"object","properties":{}}}],"messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}]}`,
		},
		{
			name: "multiple_messages",
			body: `{"model":"xiaomi/mimo-v2.5","max_tokens":1,"messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]},{"role":"user","content":[{"type":"text","text":"there"}]}]}`,
		},
		{
			name: "text_too_long",
			body: `{"model":"xiaomi/mimo-v2.5","max_tokens":1,"messages":[{"role":"user","content":[{"type":"text","text":"this text is way longer than sixteen chars"}]}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"msg_up","type":"message","role":"assistant","content":[{"type":"text","text":"from upstream"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
			}))
			defer upstream.Close()

			router := newMessagesTestRouter(t, config.UpstreamConfig{
				Name:        "probe-pass-through-" + tc.name,
				BaseURL:     upstream.URL,
				APIKeys:     []string{"sk-test"},
				ServiceType: "claude",
				Status:      "active",
			})

			w := performMessagesHandlerRequest(t, router, tc.body)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "from upstream") {
				t.Fatalf("expected upstream response, got %s", w.Body.String())
			}
		})
	}
}
