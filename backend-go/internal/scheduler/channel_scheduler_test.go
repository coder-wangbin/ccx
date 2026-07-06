package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/BenedictKing/ccx/internal/ratelimit"
	"github.com/BenedictKing/ccx/internal/session"
	"github.com/BenedictKing/ccx/internal/warmup"
)

// createTestConfigManager 创建测试用配置管理器
func createTestConfigManager(t *testing.T, cfg config.Config) (*config.ConfigManager, func()) {
	t.Helper()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "scheduler-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建临时配置文件
	configFile := filepath.Join(tmpDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 创建配置管理器
	cfgManager, err := config.NewConfigManager(configFile, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("创建配置管理器失败: %v", err)
	}

	cleanup := func() {
		cfgManager.Close()
		os.RemoveAll(tmpDir)
	}

	return cfgManager, cleanup
}

// createTestScheduler 创建测试用调度器
func createTestScheduler(t *testing.T, cfg config.Config) (*ChannelScheduler, func()) {
	t.Helper()

	cfgManager, cleanup := createTestConfigManager(t, cfg)
	messagesMetrics := metrics.NewMetricsManager()
	responsesMetrics := metrics.NewMetricsManager()
	geminiMetrics := metrics.NewMetricsManager()
	chatMetrics := metrics.NewMetricsManager()
	imagesMetrics := metrics.NewMetricsManager()
	traceAffinity := session.NewTraceAffinityManager()
	urlManager := warmup.NewURLManager(30*time.Second, 3)

	scheduler := NewChannelScheduler(cfgManager, messagesMetrics, responsesMetrics, geminiMetrics, chatMetrics, imagesMetrics, traceAffinity, urlManager)
	scheduler.SetRateLimitManager(ratelimit.NewManager())

	return scheduler, func() {
		messagesMetrics.Stop()
		responsesMetrics.Stop()
		chatMetrics.Stop()
		geminiMetrics.Stop()
		imagesMetrics.Stop()
		cleanup()
	}
}

func TestSelectChannelDryRunDoesNotRecordLastSelectedChannel(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "first",
				BaseURL:  "https://first.example.com",
				APIKeys:  []string{"sk-first"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "second",
				BaseURL:  "https://second.example.com",
				APIKeys:  []string{"sk-second"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	if got := scheduler.GetCurrentChannelIndex(ChannelKindMessages); got != 0 {
		t.Fatalf("初始 current channel = %d, want 0", got)
	}

	result, err := scheduler.SelectChannelWithOptions(context.Background(), SelectionOptions{
		Kind:           ChannelKindMessages,
		FailedChannels: map[int]bool{0: true},
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("dry-run 选择失败: %v", err)
	}
	if result.ChannelIndex != 1 {
		t.Fatalf("dry-run result channel = %d, want 1", result.ChannelIndex)
	}
	if got := scheduler.GetCurrentChannelIndex(ChannelKindMessages); got != 0 {
		t.Fatalf("dry-run 不应更新 current channel，got %d want 0", got)
	}
}

// TestPromotedChannelBypassesHealthCheck 测试促销渠道绕过健康检查
func TestPromotedChannelBypassesHealthCheck(t *testing.T) {
	// 设置促销截止时间为 5 分钟后
	promotionUntil := time.Now().Add(5 * time.Minute)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "normal-channel",
				BaseURL:  "https://normal.example.com",
				APIKeys:  []string{"sk-normal-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "promoted-channel",
				BaseURL:        "https://promoted.example.com",
				APIKeys:        []string{"sk-promoted-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟促销渠道之前有高失败率（使其不健康）
	metricsManager := scheduler.messagesMetricsManager
	for i := 0; i < 10; i++ {
		metricsManager.RecordFailure("https://promoted.example.com", "sk-promoted-key", "claude")
	}

	// 验证促销渠道确实不健康
	isHealthy := metricsManager.IsChannelHealthyWithKeys("https://promoted.example.com", []string{"sk-promoted-key"}, "claude")
	if isHealthy {
		t.Fatal("促销渠道应该被标记为不健康")
	}

	// 选择渠道 - 促销渠道应该被选中，即使它不健康
	result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 1 {
		t.Errorf("期望选择促销渠道 (index=1)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Reason != "promotion_priority" {
		t.Errorf("期望选择原因为 promotion_priority，实际为 %s", result.Reason)
	}

	if result.Upstream.Name != "promoted-channel" {
		t.Errorf("期望选择 promoted-channel，实际选择了 %s", result.Upstream.Name)
	}

	if got := scheduler.GetCurrentChannelIndex(ChannelKindMessages); got != 1 {
		t.Errorf("运行态当前渠道 = %d, want 1", got)
	}
}

// TestPromotedChannelSkippedAfterFailure 测试促销渠道在本次请求失败后被跳过
func TestPromotedChannelSkippedAfterFailure(t *testing.T) {
	promotionUntil := time.Now().Add(5 * time.Minute)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "normal-channel",
				BaseURL:  "https://normal.example.com",
				APIKeys:  []string{"sk-normal-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "promoted-channel",
				BaseURL:        "https://promoted.example.com",
				APIKeys:        []string{"sk-promoted-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟促销渠道在本次请求中已经失败
	failedChannels := map[int]bool{
		1: true, // 促销渠道已失败
	}

	// 选择渠道 - 应该跳过促销渠道，选择正常渠道
	result, err := scheduler.SelectChannel(context.Background(), "test-user", failedChannels, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 0 {
		t.Errorf("期望选择正常渠道 (index=0)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Upstream.Name != "normal-channel" {
		t.Errorf("期望选择 normal-channel，实际选择了 %s", result.Upstream.Name)
	}

	if got := scheduler.GetCurrentChannelIndex(ChannelKindMessages); got != 0 {
		t.Errorf("运行态当前渠道 = %d, want 0", got)
	}
}

func TestPromotedChannelSkippedInRuntimeCooldown(t *testing.T) {
	promotionUntil := time.Now().Add(5 * time.Minute)
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "normal-channel",
				BaseURL:  "https://normal.example.com",
				APIKeys:  []string{"sk-normal-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "promoted-channel",
				BaseURL:        "https://promoted.example.com",
				APIKeys:        []string{"sk-promoted-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()
	scheduler.MarkChannelCooldown(ChannelKindMessages, 1, time.Minute)

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}
	if result.ChannelIndex != 0 {
		t.Fatalf("期望跳过 cooldown 促销渠道并选择 index=0，实际为 index=%d", result.ChannelIndex)
	}
	if result.Reason != "priority_order" {
		t.Fatalf("期望选择原因为 priority_order，实际为 %s", result.Reason)
	}
}

// TestNonPromotedChannelStillChecksHealth 测试非促销渠道仍然检查健康状态
func TestNonPromotedChannelStillChecksHealth(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "unhealthy-channel",
				BaseURL:  "https://unhealthy.example.com",
				APIKeys:  []string{"sk-unhealthy-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "healthy-channel",
				BaseURL:  "https://healthy.example.com",
				APIKeys:  []string{"sk-healthy-key"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟第一个渠道不健康
	metricsManager := scheduler.messagesMetricsManager
	for i := 0; i < 10; i++ {
		metricsManager.RecordFailure("https://unhealthy.example.com", "sk-unhealthy-key", "claude")
	}

	// 选择渠道 - 应该跳过不健康的渠道，选择健康的渠道
	result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 1 {
		t.Errorf("期望选择健康渠道 (index=1)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Upstream.Name != "healthy-channel" {
		t.Errorf("期望选择 healthy-channel，实际选择了 %s", result.Upstream.Name)
	}
}

func TestSelectChannelTraceRecordsSelectionAndSkips(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "suspended-channel",
				BaseURL:  "https://suspended.example.com",
				APIKeys:  []string{"sk-suspended-key"},
				Status:   "suspended",
				Priority: 1,
			},
			{
				Name:     "selected-channel",
				BaseURL:  "https://selected.example.com",
				APIKeys:  []string{"sk-selected-key"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("SelectChannel() error = %v", err)
	}
	if result.Trace == nil {
		t.Fatal("SelectionResult.Trace = nil, want trace")
	}
	if result.Trace.Kind != ChannelKindMessages {
		t.Fatalf("trace kind = %s, want %s", result.Trace.Kind, ChannelKindMessages)
	}
	if result.Trace.Selected == nil {
		t.Fatal("trace selected = nil")
	}
	if result.Trace.Selected.ChannelIndex != 1 || result.Trace.Selected.ChannelName != "selected-channel" || result.Trace.Selected.Reason != "priority_order" {
		t.Fatalf("trace selected = %+v, want selected-channel priority_order", result.Trace.Selected)
	}
	if !traceHasStage(result.Trace, "active_model_filter", 2) {
		t.Fatalf("trace stages = %+v, want active_model_filter count 2", result.Trace.Stages)
	}
	if !traceHasCandidateSkip(result.Trace, 0, "priority_order", "inactive_status") {
		t.Fatalf("trace candidates = %+v, want inactive_status skip for channel 0", result.Trace.Candidates)
	}
}

func traceHasStage(trace *SelectionTrace, name string, count int) bool {
	if trace == nil {
		return false
	}
	for _, stage := range trace.Stages {
		if stage.Name == name && stage.Count == count {
			return true
		}
	}
	return false
}

func traceHasCandidateSkip(trace *SelectionTrace, channelIndex int, stage, reason string) bool {
	if trace == nil {
		return false
	}
	for _, candidate := range trace.Candidates {
		if candidate.ChannelIndex == channelIndex && candidate.Stage == stage && candidate.Reason == reason {
			return true
		}
	}
	return false
}

func TestFormatSelectionTraceSummaryLimitsSkips(t *testing.T) {
	trace := &SelectionTrace{
		Kind: ChannelKindMessages,
		Stages: []SelectionTraceStage{
			{Name: "active_model_filter", Count: 3},
			{Name: "context_filter", Count: 2},
		},
		Candidates: []SelectionTraceCandidate{
			{ChannelIndex: 0, ChannelName: "a", Stage: "priority_order", Reason: "inactive_status"},
			{ChannelIndex: 1, ChannelName: "b", Stage: "priority_order", Reason: "runtime_cooldown"},
			{ChannelIndex: 2, ChannelName: "c", Stage: "priority_order", Reason: "rate_limit_pressure"},
		},
		Selected: &SelectionTraceSelection{ChannelIndex: 3, ChannelName: "d", Reason: "fallback"},
	}

	summary := FormatSelectionTraceSummary(trace, 2)
	for _, want := range []string{
		"stages=active_model_filter:3,context_filter:2",
		"skipped=0:a@priority_order/inactive_status,1:b@priority_order/runtime_cooldown,+1",
		"selected=3:d/fallback",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary = %q, want contains %q", summary, want)
		}
	}
}

// TestExpiredPromotionNotBypassHealthCheck 测试过期的促销不绕过健康检查
func TestExpiredPromotionNotBypassHealthCheck(t *testing.T) {
	// 设置促销截止时间为过去
	promotionUntil := time.Now().Add(-5 * time.Minute)

	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "healthy-channel",
				BaseURL:  "https://healthy.example.com",
				APIKeys:  []string{"sk-healthy-key"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:           "expired-promoted-channel",
				BaseURL:        "https://expired.example.com",
				APIKeys:        []string{"sk-expired-key"},
				Status:         "active",
				Priority:       2,
				PromotionUntil: &promotionUntil, // 已过期
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	// 模拟过期促销渠道不健康
	metricsManager := scheduler.messagesMetricsManager
	for i := 0; i < 10; i++ {
		metricsManager.RecordFailure("https://expired.example.com", "sk-expired-key", "claude")
	}

	// 选择渠道 - 过期促销渠道不应该被优先选择，应该选择健康的渠道
	result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 0 {
		t.Errorf("期望选择健康渠道 (index=0)，实际选择了 index=%d", result.ChannelIndex)
	}

	if result.Upstream.Name != "healthy-channel" {
		t.Errorf("期望选择 healthy-channel，实际选择了 %s", result.Upstream.Name)
	}
}

func TestSelectChannel_DefaultRouteRejectsPrefixedOnlyChannels(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:        "kimi-only",
				BaseURL:     "https://kimi.example.com",
				APIKeys:     []string{"sk-kimi"},
				Status:      "active",
				Priority:    1,
				RoutePrefix: "kimi",
			},
			{
				Name:        "deepseek-only",
				BaseURL:     "https://deepseek.example.com",
				APIKeys:     []string{"sk-deepseek"},
				Status:      "active",
				Priority:    2,
				RoutePrefix: "deepseek",
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	_, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
	if err == nil {
		t.Fatal("SelectChannel() error = nil, want default route rejection")
	}
}

// TestDeleteChannelMetrics_SharedMetricsKeyPreserved 测试删除渠道时共享的 metricsKey 被保留

func TestSelectChannelFiltersSupportedModels(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:            "image-excluded",
				BaseURL:         "https://excluded.example.com",
				APIKeys:         []string{"sk-excluded"},
				Status:          "active",
				Priority:        1,
				SupportedModels: []string{"gpt-4*", "!*image*"},
			},
			{
				Name:            "image-allowed",
				BaseURL:         "https://allowed.example.com",
				APIKeys:         []string{"sk-allowed"},
				Status:          "active",
				Priority:        2,
				SupportedModels: []string{"gpt-4*"},
			},
			{
				Name:            "invalid-pattern-fallback",
				BaseURL:         "https://invalid.example.com",
				APIKeys:         []string{"sk-invalid"},
				Status:          "active",
				Priority:        3,
				SupportedModels: []string{"foo*bar", "claude-*"},
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	t.Run("命中排除规则时跳过高优先级渠道", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "gpt-4-image-preview", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("期望选择 index=1，实际为 %d", result.ChannelIndex)
		}
	})

	t.Run("模型过滤跳过渠道时记录命中排除规则", func(t *testing.T) {
		var buf bytes.Buffer
		oldOutput := log.Writer()
		log.SetOutput(&buf)
		defer log.SetOutput(oldOutput)

		result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "gpt-4-image-preview", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("期望选择 index=1，实际为 %d", result.ChannelIndex)
		}

		logOutput := buf.String()
		if !strings.Contains(logOutput, "[Scheduler-ModelFilter] 跳过渠道 [0] image-excluded") {
			t.Fatalf("期望记录模型过滤跳过日志，实际日志: %s", logOutput)
		}
		if !strings.Contains(logOutput, "模型 \"gpt-4-image-preview\" 不被 supportedModels 支持 (命中排除规则 !*image*)") {
			t.Fatalf("期望记录命中的排除规则，实际日志: %s", logOutput)
		}
	})

	t.Run("非法规则被跳过且不影响合法规则", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "claude-3-7-sonnet", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 2 {
			t.Fatalf("期望选择 index=2，实际为 %d", result.ChannelIndex)
		}
	})

	t.Run("所有活跃渠道都不支持模型时返回明确错误", func(t *testing.T) {
		_, err := scheduler.SelectChannel(context.Background(), "test-user", make(map[int]bool), ChannelKindMessages, "gemini-2.5-pro", "", "")
		if err == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		if err.Error() != "没有 Messages 渠道支持模型 \"gemini-2.5-pro\"，请检查渠道的 supportedModels 配置" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSelectChannelTraceAffinityStillRespectsSupportedModels(t *testing.T) {
	cfg := config.Config{
		ResponsesUpstream: []config.UpstreamConfig{
			{
				Name:            "LocalHostClaude",
				BaseURL:         "http://127.0.0.1:3699",
				APIKeys:         []string{"sk-local"},
				Status:          "active",
				Priority:        1,
				SupportedModels: []string{"claude-*"},
				ServiceType:     "responses",
			},
			{
				Name:            "LocalHostOpenAIChat",
				BaseURL:         "http://127.0.0.1:3699",
				APIKeys:         []string{"sk-local"},
				Status:          "active",
				Priority:        2,
				SupportedModels: []string{"gpt-5.5", "gpt-5.4"},
				ServiceType:     "responses",
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	scheduler.traceAffinity.SetPreferredChannel(string(ChannelKindResponses)+":test-user", 0)

	result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindResponses, "gpt-5.5", "", "")
	if err != nil {
		t.Fatalf("选择渠道失败: %v", err)
	}

	if result.ChannelIndex != 1 {
		t.Fatalf("期望跳过不支持模型的亲和渠道并选择 index=1，实际为 index=%d", result.ChannelIndex)
	}
	if result.Reason != "priority_order" {
		t.Fatalf("期望回退到 priority_order，实际为 %s", result.Reason)
	}
}

func tcServiceType(kind ChannelKind) string {
	switch kind {
	case ChannelKindGemini:
		return "gemini"
	case ChannelKindResponses:
		return "responses"
	case ChannelKindChat:
		return "openai"
	default:
		return "claude"
	}
}

func TestNormalizedMetricsServiceType(t *testing.T) {
	tests := []struct {
		name       string
		kind       ChannelKind
		configured string
		want       string
	}{
		{name: "messages default", kind: ChannelKindMessages, want: "claude"},
		{name: "responses default", kind: ChannelKindResponses, want: "responses"},
		{name: "gemini default", kind: ChannelKindGemini, want: "gemini"},
		{name: "chat default", kind: ChannelKindChat, want: "openai"},
		{name: "configured wins", kind: ChannelKindChat, configured: "responses", want: "responses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizedMetricsServiceType(tt.kind, tt.configured); got != tt.want {
				t.Fatalf("NormalizedMetricsServiceType(%q, %q)=%q, want=%q", tt.kind, tt.configured, got, tt.want)
			}
		})
	}
}

func TestSelectChannelByName(t *testing.T) {
	cfg := config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:     "channel-a",
				BaseURL:  "https://a.example.com",
				APIKeys:  []string{"sk-a"},
				Status:   "active",
				Priority: 1,
			},
			{
				Name:     "channel-b",
				BaseURL:  "https://b.example.com",
				APIKeys:  []string{"sk-b"},
				Status:   "active",
				Priority: 2,
			},
		},
	}

	scheduler, cleanup := createTestScheduler(t, cfg)
	defer cleanup()

	t.Run("指定渠道名直接定位", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "channel-b")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("期望选择 index=1 (channel-b)，实际为 %d", result.ChannelIndex)
		}
		if result.Reason != "channel_pin" {
			t.Fatalf("期望原因为 channel_pin，实际为 %s", result.Reason)
		}
		if result.Upstream.Name != "channel-b" {
			t.Fatalf("期望选择 channel-b，实际为 %s", result.Upstream.Name)
		}
	})

	t.Run("指定渠道名跳过更高优先级渠道", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "channel-b")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 1 {
			t.Fatalf("应跳过高优先级 channel-a，实际选择 index=%d", result.ChannelIndex)
		}
	})

	t.Run("指定不存在的渠道名返回错误", func(t *testing.T) {
		_, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "nonexistent")
		if err == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		if !strings.Contains(err.Error(), "nonexistent") {
			t.Fatalf("错误信息应包含渠道名，实际: %v", err)
		}
	})

	t.Run("指定已失败的渠道名返回错误", func(t *testing.T) {
		failed := map[int]bool{1: true}
		_, err := scheduler.SelectChannel(context.Background(), "test-user", failed, ChannelKindMessages, "", "", "channel-b")
		if err == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		if !strings.Contains(err.Error(), "已失败") {
			t.Fatalf("错误信息应提示已失败，实际: %v", err)
		}
	})

	t.Run("空渠道名走正常选择逻辑", func(t *testing.T) {
		result, err := scheduler.SelectChannel(context.Background(), "test-user", map[int]bool{}, ChannelKindMessages, "", "", "")
		if err != nil {
			t.Fatalf("选择渠道失败: %v", err)
		}
		if result.ChannelIndex != 0 {
			t.Fatalf("正常逻辑应选高优先级 index=0，实际为 %d", result.ChannelIndex)
		}
		if result.Reason == "channel_pin" {
			t.Fatal("空渠道名不应触发 channel_pin")
		}
	})
}
