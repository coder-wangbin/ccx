package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/desktop/internal/backend"
	"github.com/BenedictKing/ccx/desktop/internal/channelpreset"
	"github.com/BenedictKing/ccx/desktop/internal/configservice"
	"github.com/BenedictKing/ccx/desktop/internal/editor"
	"github.com/BenedictKing/ccx/desktop/internal/updater"
	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type DesktopService struct {
	manager       *backend.Manager
	configService *configservice.Service
	app           *application.App
	mainWindow    application.Window
	updater       *updater.Updater
	versionInfo   VersionInfo
	notifications *notifications.NotificationService
}

type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
	GitCommit string `json:"gitCommit"`
}

type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Notes          string `json:"notes"`
	DownloadURL    string `json:"downloadUrl"`
	Sha256URL      string `json:"sha256Url"`
	Size           int64  `json:"size"`
}

type EnvFileState struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Exists  bool   `json:"exists"`
}

func NewDesktopService(manager *backend.Manager) *DesktopService {
	configService, err := configservice.New(manager.DataDir())
	if err != nil {
		log.Printf("[Desktop-Init] Agent 配置服务初始化失败: %v", err)
	}
	return &DesktopService{manager: manager, configService: configService}
}

func (s *DesktopService) setApp(app *application.App) {
	s.app = app
}

func (s *DesktopService) setMainWindow(window application.Window) {
	s.mainWindow = window
}

func (s *DesktopService) setVersion(v VersionInfo) {
	s.versionInfo = v
	s.updater = updater.New(v.Version)
}

func (s *DesktopService) setNotifications(svc *notifications.NotificationService) {
	s.notifications = svc
}

// Notify 推送一条系统通知。失败仅记录日志，不向上抛错。
//
//wails:ignore
func (s *DesktopService) Notify(title, body string) {
	if s.notifications == nil {
		return
	}
	id := fmt.Sprintf("ccx-%d", time.Now().UnixNano())
	err := s.notifications.SendNotification(notifications.NotificationOptions{
		ID:    id,
		Title: title,
		Body:  body,
	})
	if err != nil {
		log.Printf("[Desktop-Notify] 推送通知失败: %v", err)
	}
}

// CopyText 把文本写入系统剪贴板。
func (s *DesktopService) CopyText(text string) error {
	if s.app == nil {
		return fmt.Errorf("应用未初始化")
	}
	if !s.app.Clipboard.SetText(text) {
		return fmt.Errorf("写入剪贴板失败")
	}
	return nil
}

// WebURL 返回当前网关 Web UI 的访问地址（即使服务未启动，也基于配置端口拼接）。
func (s *DesktopService) WebURL() string {
	return s.manager.WebURL()
}

// GetVersion 返回构建时注入的版本信息。
func (s *DesktopService) GetVersion() VersionInfo {
	return s.versionInfo
}

func (s *DesktopService) GetStatus() backend.Status {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	return s.manager.Status(ctx)
}

func (s *DesktopService) GetProxyAccessKey() (string, error) {
	return s.manager.EnsureProxyAccessKey()
}

func (s *DesktopService) GetEnvFile() (EnvFileState, error) {
	path := filepath.Join(s.manager.DataDir(), ".env")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return EnvFileState{Path: path, Exists: false}, nil
		}
		return EnvFileState{}, err
	}
	return EnvFileState{Path: path, Content: string(content), Exists: true}, nil
}

func (s *DesktopService) SaveEnvFile(content string) error {
	path := filepath.Join(s.manager.DataDir(), ".env")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

// DetectEditors 返回系统上可用的文本编辑器列表。
func (s *DesktopService) DetectEditors() []editor.Editor {
	return editor.Detect()
}

// OpenEnvFileInEditor 使用指定编辑器打开 .env 文件。
func (s *DesktopService) OpenEnvFileInEditor(editorPath string) error {
	path := filepath.Join(s.manager.DataDir(), ".env")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return editor.Open(editorPath, path)
}

func (s *DesktopService) StartService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return s.manager.Start(ctx)
}

func (s *DesktopService) StopService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return s.manager.Stop(ctx)
}

func (s *DesktopService) RestartService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.manager.Restart(ctx)
}

func (s *DesktopService) GetLogs() []string {
	return s.manager.Logs()
}

func (s *DesktopService) GetAgentConfigStatus(platform string) (configservice.AgentConfigStatus, error) {
	if s.configService == nil {
		return configservice.AgentConfigStatus{}, fmt.Errorf("配置服务未初始化")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	status := s.manager.Status(ctx)
	return s.configService.GetStatus(platform, status.Port)
}

func (s *DesktopService) ApplyAgentConfig(req configservice.ApplyAgentConfigRequest) error {
	if s.configService == nil {
		return fmt.Errorf("配置服务未初始化")
	}
	platform := req.Platform
	if platform == "" {
		return fmt.Errorf("agent 平台不能为空")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	status := s.manager.Status(ctx)
	var key string
	if platform == configservice.PlatformCodex || (platform == configservice.PlatformClaude && (req.Provider == "" || req.Provider == configservice.ProviderCCX)) {
		if !status.Running {
			return fmt.Errorf("请先启动 CCX 服务")
		}
		var err error
		key, err = s.manager.EnsureProxyAccessKey()
		if err != nil {
			return err
		}
	}
	return s.configService.Apply(req, status.Port, key)
}

func (s *DesktopService) RestoreAgentConfig(platform string) error {
	if s.configService == nil {
		return fmt.Errorf("配置服务未初始化")
	}
	return s.configService.Restore(platform)
}

func (s *DesktopService) GetSavedProviderKeys() map[string]string {
	if s.configService == nil {
		return map[string]string{}
	}
	return s.configService.GetSavedProviderKeys()
}

func (s *DesktopService) GetProviderPresets(target string) []channelpreset.ProviderPreset {
	presets := channelpreset.Presets()
	target = strings.TrimSpace(target)
	if target == "" {
		return presets
	}
	for i := range presets {
		presets[i].Plans = channelpreset.FilterPlansForTarget(presets[i].Plans, target)
	}
	return presets
}

func (s *DesktopService) GetProviderKeyAssets() []configservice.ProviderKeyAsset {
	if s.configService == nil {
		return []configservice.ProviderKeyAsset{}
	}
	return s.configService.GetProviderKeyAssets()
}

func (s *DesktopService) CreateCCXChannelFromPreset(req channelpreset.CreateChannelRequest) (channelpreset.CreateChannelResult, error) {
	if s.configService == nil {
		return channelpreset.CreateChannelResult{}, fmt.Errorf("配置服务未初始化")
	}
	if preset, ok := channelpreset.FindPreset(req.Provider); ok {
		req.Provider = preset.ID
		if strings.TrimSpace(req.Target) == "" {
			req.Target = preset.DefaultTarget
		}
	}
	if strings.TrimSpace(req.APIKey) == "" {
		planID := strings.TrimSpace(req.PlanID)
		var fallbackKey string
		for _, asset := range s.configService.GetProviderKeyAssets() {
			if asset.Provider != strings.TrimSpace(req.Provider) || asset.APIKey == "" {
				continue
			}
			if planID != "" && asset.PlanID != planID {
				continue
			}
			if asset.PlanID == "" {
				fallbackKey = asset.APIKey
				break
			}
			if fallbackKey == "" {
				fallbackKey = asset.APIKey
			}
		}
		req.APIKey = fallbackKey
	}
	payload, err := channelpreset.BuildPayload(req)
	if err != nil {
		return channelpreset.CreateChannelResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.manager.Start(ctx); err != nil {
		return channelpreset.CreateChannelResult{}, err
	}
	if err := s.manager.WaitHealthy(ctx, 15*time.Second); err != nil {
		return channelpreset.CreateChannelResult{}, err
	}
	adminKey, err := s.adminAccessKey()
	if err != nil {
		return channelpreset.CreateChannelResult{}, err
	}
	updated, err := s.createChannel(ctx, req.Target, payload, adminKey)
	if err != nil {
		return channelpreset.CreateChannelResult{}, err
	}
	if err := s.configService.SaveProviderKeyAsset(configservice.ProviderKeyAsset{
		Provider: req.Provider,
		APIKey:   payload.APIKeys[0],
		BaseURL:  payload.BaseURL,
		PlanID:   req.PlanID,
		Usages:   []string{req.Target + "-channel"},
	}); err != nil {
		return channelpreset.CreateChannelResult{}, err
	}
	message := "渠道已添加到 CCX"
	if updated {
		message = "同名渠道配置已覆盖更新"
	}
	return channelpreset.CreateChannelResult{
		Provider: req.Provider,
		Target:   req.Target,
		Name:     payload.Name,
		BaseURL:  payload.BaseURL,
		Message:  message,
	}, nil
}

func (s *DesktopService) adminAccessKey() (string, error) {
	env, err := s.GetEnvFile()
	if err != nil {
		return "", err
	}
	values := parseEnvContent(env.Content)
	if key := strings.TrimSpace(values["ADMIN_ACCESS_KEY"]); key != "" {
		return key, nil
	}
	if key := strings.TrimSpace(values["PROXY_ACCESS_KEY"]); key != "" {
		return key, nil
	}
	return s.manager.EnsureProxyAccessKey()
}

// createChannel 创建或更新渠道，返回 (是否更新, error)。
func (s *DesktopService) createChannel(ctx context.Context, target string, payload channelpreset.ChannelPayload, adminKey string) (bool, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	baseURL := s.manager.WebURL()
	path := "/api/" + strings.TrimSpace(target) + "/channels"
	client := &http.Client{Timeout: 15 * time.Second}

	// 查找同名渠道，存在则覆盖更新
	if existingID, err := s.findChannelIndexByName(ctx, client, baseURL, path, payload.Name, adminKey); err == nil && existingID >= 0 {
		return true, s.putChannel(ctx, client, baseURL, fmt.Sprintf("%s/%d", path, existingID), body, adminKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", adminKey)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.TrimSpace(string(data))
	if message == "" {
		message = resp.Status
	}
	return false, fmt.Errorf("创建 CCX %s 渠道失败: %s", target, message)
}

func (s *DesktopService) findChannelIndexByName(ctx context.Context, client *http.Client, baseURL, path, name, adminKey string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return -1, err
	}
	req.Header.Set("x-api-key", adminKey)
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return -1, fmt.Errorf("查询渠道列表失败: %s", resp.Status)
	}
	var result struct {
		Channels []struct {
			Index int    `json:"index"`
			Name  string `json:"name"`
		} `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return -1, err
	}
	for _, ch := range result.Channels {
		if ch.Name == name {
			return ch.Index, nil
		}
	}
	return -1, nil
}

func (s *DesktopService) putChannel(ctx context.Context, client *http.Client, baseURL, path string, body []byte, adminKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", adminKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.TrimSpace(string(data))
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("更新 CCX 渠道失败: %s", message)
}

func parseEnvContent(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if rest, ok := strings.CutPrefix(key, "export "); ok {
			key = strings.TrimSpace(rest)
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		values[key] = value
	}
	return values
}

func (s *DesktopService) ShowStatusTab() error {
	s.showWindow()
	if s.app != nil {
		s.app.Event.Emit("desktop:show-tab", "status")
	}
	return nil
}

func (s *DesktopService) ShowAgentTab() error {
	s.showWindow()
	if s.app != nil {
		s.app.Event.Emit("desktop:show-tab", "agent")
	}
	return nil
}

func (s *DesktopService) ShowWebUITab() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.manager.Start(ctx); err != nil {
		return err
	}
	if err := s.manager.WaitHealthy(ctx, 15*time.Second); err != nil {
		return err
	}
	s.showWindow()
	if s.app != nil {
		s.app.Event.Emit("desktop:show-tab", "web")
	}
	return nil
}

func (s *DesktopService) OpenWebUIInBrowser() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.manager.Start(ctx); err != nil {
		return err
	}
	if err := s.manager.WaitHealthy(ctx, 15*time.Second); err != nil {
		return err
	}
	return browser.OpenURL(s.manager.WebURL())
}

func (s *DesktopService) GetAutostartStatus() (bool, error) {
	if s.app == nil {
		return false, fmt.Errorf("应用未初始化")
	}
	return s.app.Autostart.IsEnabled()
}

func (s *DesktopService) SetAutostart(enabled bool) error {
	if s.app == nil {
		return fmt.Errorf("应用未初始化")
	}
	if enabled {
		return s.app.Autostart.Enable()
	}
	return s.app.Autostart.Disable()
}

func (s *DesktopService) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = s.manager.Stop(ctx)
}

// CheckUpdate 查询是否有新版本可用。
func (s *DesktopService) CheckUpdate() (UpdateInfo, error) {
	info := UpdateInfo{CurrentVersion: s.versionInfo.Version}
	if s.updater == nil {
		return info, fmt.Errorf("updater 未初始化")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	release, err := s.updater.Check(ctx)
	if err != nil {
		return info, err
	}
	if release == nil {
		return info, nil
	}
	info.Available = true
	info.LatestVersion = release.Version
	info.Notes = release.Notes
	info.DownloadURL = release.DownloadURL
	info.Sha256URL = release.Sha256URL
	info.Size = release.Size
	if s.app != nil {
		s.app.Event.Emit("update:available", info)
	}
	return info, nil
}

// DownloadAndInstall 下载、校验并触发安装。整个流程通过 update:progress 事件推送进度。
func (s *DesktopService) DownloadAndInstall(info UpdateInfo) error {
	if s.updater == nil {
		return fmt.Errorf("updater 未初始化")
	}
	if info.DownloadURL == "" {
		return fmt.Errorf("缺少下载地址")
	}
	release := &updater.Release{
		Version:     info.LatestVersion,
		DownloadURL: info.DownloadURL,
		Sha256URL:   info.Sha256URL,
		Size:        info.Size,
	}
	go s.runUpdate(release)
	return nil
}

func (s *DesktopService) runUpdate(release *updater.Release) {
	// 订阅进度并转发到前端
	if s.app != nil {
		go func() {
			for p := range s.updater.Subscribe() {
				s.app.Event.Emit("update:progress", p)
				if p.Phase == updater.PhaseDone || p.Phase == updater.PhaseError {
					return
				}
			}
		}()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	localPath, err := s.updater.Download(ctx, release)
	if err != nil {
		if s.app != nil {
			s.app.Event.Emit("update:progress", updater.Progress{
				Phase: updater.PhaseError,
				Error: err.Error(),
			})
		}
		return
	}

	if err := s.updater.Verify(ctx, localPath, release.Sha256URL); err != nil {
		if s.app != nil {
			s.app.Event.Emit("update:progress", updater.Progress{
				Phase: updater.PhaseError,
				Error: err.Error(),
			})
		}
		return
	}

	if err := s.updater.Install(localPath); err != nil {
		if s.app != nil {
			s.app.Event.Emit("update:progress", updater.Progress{
				Phase: updater.PhaseError,
				Error: err.Error(),
			})
		}
	}
}

// CancelUpdate 中止正在进行的下载/安装。
func (s *DesktopService) CancelUpdate() error {
	if s.updater != nil {
		s.updater.Cancel()
	}
	return nil
}

func (s *DesktopService) showWindow() {
	if s.mainWindow == nil {
		return
	}
	if s.mainWindow.IsMinimised() {
		s.mainWindow.UnMinimise()
	}
	s.mainWindow.Show()
	s.mainWindow.Focus()
}
