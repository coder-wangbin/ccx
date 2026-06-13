package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/conversation"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/gin-gonic/gin"
)

func TestConversationSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建临时配置目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 写入初始配置
	initialConfig := map[string]interface{}{
		"upstream":          []interface{}{},
		"responsesUpstream": []interface{}{},
		"geminiUpstream":    []interface{}{},
		"chatUpstream":      []interface{}{},
		"imagesUpstream":    []interface{}{},
	}
	configJSON, _ := json.MarshalIndent(initialConfig, "", "  ")
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 初始化 ConfigManager
	backupDir := filepath.Join(tmpDir, "backups")
	cfgManager, err := config.NewConfigManager(configPath, backupDir)
	if err != nil {
		t.Fatalf("初始化 ConfigManager 失败: %v", err)
	}
	defer cfgManager.Close()

	// 初始化依赖
	tracker := conversation.NewConversationTracker(1*time.Hour, 24*time.Hour, "")
	overrideManager := conversation.NewOverrideManager(30 * time.Minute)
	channelScheduler := scheduler.NewChannelScheduler(cfgManager, nil, nil, nil, nil, nil, nil, nil)

	deps := &ConversationHandlerDeps{
		Tracker:          tracker,
		OverrideManager:  overrideManager,
		ChannelScheduler: channelScheduler,
		ConfigManager:    cfgManager,
	}

	// 测试 GET /api/conversations/settings
	t.Run("GetSettings", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/conversations/settings", nil)

		GetConversationSettings(deps)(c)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		// 初始值应该是 0（使用环境变量默认值）
		if ttl, ok := resp["overrideTtlMinutes"].(float64); !ok || ttl != 0 {
			t.Errorf("期望 overrideTtlMinutes 为 0，得到 %v", resp["overrideTtlMinutes"])
		}
	})

	// 测试 PUT /api/conversations/settings
	t.Run("UpdateSettings", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"overrideTtlMinutes": 60,
		}
		reqJSON, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/conversations/settings", bytes.NewReader(reqJSON))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateConversationSettings(deps)(c)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
		}

		// 验证配置已更新
		cfg := cfgManager.GetConfig()
		if cfg.OverrideTTLMinutes != 60 {
			t.Errorf("期望配置中 OverrideTTLMinutes 为 60，得到 %d", cfg.OverrideTTLMinutes)
		}
	})

	// 测试边界验证
	t.Run("UpdateSettings_InvalidValue_TooLarge", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"overrideTtlMinutes": 2000, // 超出最大值 1440
		}
		reqJSON, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/conversations/settings", bytes.NewReader(reqJSON))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateConversationSettings(deps)(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("期望状态码 400，得到 %d", w.Code)
		}
	})

	t.Run("UpdateSettings_NeverExpire", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"overrideTtlMinutes": -1, // -1 表示永不恢复
		}
		reqJSON, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/conversations/settings", bytes.NewReader(reqJSON))
		c.Request.Header.Set("Content-Type", "application/json")

		UpdateConversationSettings(deps)(c)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200（-1 允许作为永不恢复），得到 %d: %s", w.Code, w.Body.String())
		}

		// 验证配置已更新为 -1
		cfg := cfgManager.GetConfig()
		if cfg.OverrideTTLMinutes != -1 {
			t.Errorf("期望配置中 OverrideTTLMinutes 为 -1，得到 %d", cfg.OverrideTTLMinutes)
		}
	})

	// 测试读取已更新的值
	t.Run("GetSettings_AfterUpdate", func(t *testing.T) {
		// 先重新设置为 60
		reqBody := map[string]interface{}{
			"overrideTtlMinutes": 60,
		}
		reqJSON, _ := json.Marshal(reqBody)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("PUT", "/api/conversations/settings", bytes.NewReader(reqJSON))
		c.Request.Header.Set("Content-Type", "application/json")
		UpdateConversationSettings(deps)(c)

		// 然后读取验证
		w = httptest.NewRecorder()
		c, _ = gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/conversations/settings", nil)

		GetConversationSettings(deps)(c)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		// 应该返回之前设置的 60
		if ttl, ok := resp["overrideTtlMinutes"].(float64); !ok || ttl != 60 {
			t.Errorf("期望 overrideTtlMinutes 为 60，得到 %v", resp["overrideTtlMinutes"])
		}
	})
}
