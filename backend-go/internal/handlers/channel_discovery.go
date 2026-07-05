package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/httpclient"
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/gin-gonic/gin"
)

const discoveryProbeTimeout = 10 * time.Second

type ChannelDiscoveryRequest struct {
	ChannelKind        string            `json:"channelKind"`
	ServiceType        string            `json:"serviceType"`
	BaseURL            string            `json:"baseUrl"`
	BaseURLs           []string          `json:"baseUrls"`
	APIKey             string            `json:"apiKey"`
	AuthHeader         string            `json:"authHeader"`
	CustomHeaders      map[string]string `json:"customHeaders"`
	ProxyURL           string            `json:"proxyUrl"`
	InsecureSkipVerify bool              `json:"insecureSkipVerify"`
	ModelMapping       map[string]string `json:"modelMapping"`
	ReasoningMapping   map[string]string `json:"reasoningMapping"`
	TargetClients      []string          `json:"targetClients"`
}

type DiscoverySelectedModels struct {
	Strong  string `json:"strong,omitempty"`
	Primary string `json:"primary,omitempty"`
	Fast    string `json:"fast,omitempty"`
}

type DiscoveryModelsResult struct {
	Source     string                  `json:"source"`
	URL        string                  `json:"url,omitempty"`
	StatusCode int                     `json:"statusCode,omitempty"`
	Items      []string                `json:"items"`
	Selected   DiscoverySelectedModels `json:"selected"`
	Warnings   []string                `json:"warnings,omitempty"`
}

type DiscoveryProtocolResult struct {
	Protocol      string   `json:"protocol"`
	Success       bool     `json:"success"`
	SuccessModels []string `json:"successModels,omitempty"`
	FailedModels  []string `json:"failedModels,omitempty"`
	LatencyMs     int64    `json:"latencyMs,omitempty"`
	Error         string   `json:"error,omitempty"`
}

type DiscoveryEvidence struct {
	Type    string `json:"type"`
	Key     string `json:"key,omitempty"`
	Message string `json:"message"`
}

type DiscoveryRecommendation struct {
	ChannelKind       string                 `json:"channelKind"`
	ServiceType       string                 `json:"serviceType"`
	BaseURLs          []string               `json:"baseUrls,omitempty"`
	ModelMapping      map[string]string      `json:"modelMapping"`
	ReasoningMapping  map[string]string      `json:"reasoningMapping,omitempty"`
	SupportedModels   []string               `json:"supportedModels,omitempty"`
	Compat            map[string]bool        `json:"compat,omitempty"`
	URLRecommendation *URLRecommendation     `json:"urlRecommendation,omitempty"`
	Evidence          []DiscoveryEvidence    `json:"evidence,omitempty"`
	Alternatives      []DiscoveryAlternative `json:"alternatives,omitempty"`
}

type DiscoveryAlternative struct {
	ChannelKind string `json:"channelKind"`
	Reason      string `json:"reason"`
}

type ChannelDiscoveryResponse struct {
	Models         DiscoveryModelsResult     `json:"models"`
	Protocols      []DiscoveryProtocolResult `json:"protocols"`
	Recommendation DiscoveryRecommendation   `json:"recommendation"`
	Evidence       []DiscoveryEvidence       `json:"evidence,omitempty"`
}

func ChannelDiscovery(cfgManager *config.ConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ChannelDiscoveryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		channel, err := buildTransientDiscoveryChannel(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		globalCapabilities := map[string]config.UpstreamModelCapability(nil)
		if cfgManager != nil {
			globalCapabilities = cfgManager.GetConfig().UpstreamModelCapabilities
		}

		models := discoverTransientModels(c.Request.Context(), channel, normalizeDiscoveryChannelKind(req.ChannelKind), channel.APIKeys[0])
		if len(models.Items) == 0 {
			models.Items = fallbackDiscoveryProbeModels(req.ChannelKind, channel.ServiceType)
			models.Source = "fallback"
			models.Warnings = append(models.Warnings, "models endpoint unavailable; used built-in probe candidates")
		}
		models.Selected = selectDiscoveryModels(models.Items, globalCapabilities)

		probeModels := discoveryProbeModels(models.Selected, models.Items)
		protocols := runDiscoveryProtocolProbes(c.Request.Context(), channel, probeModels, cfgManager)
		successByProtocol := discoverySuccessModelsByProtocol(protocols)
		recommendedKind := recommendDiscoveryChannelKind(req.ChannelKind, req.TargetClients, protocols)

		recommendation := buildDiscoveryMappingRecommendation(recommendedKind, models.Selected, successByProtocol, req.TargetClients)
		recommendation.ServiceType = channel.ServiceType
		recommendation.BaseURLs = append([]string(nil), channel.BaseURLs...)
		if recommendation.ChannelKind != "" {
			compatModel := discoveryCompatProbeModel(recommendation.ChannelKind, models.Selected, successByProtocol)
			compat := runCompatDiagnoseWithProbeModel(channel, recommendation.ChannelKind, channel.APIKeys[0], capabilityTestBaseURL(channel), compatModel)
			recommendation.Compat = compat.Recommendations
			recommendation.URLRecommendation = compat.URLRecommendations
			for key, message := range compat.Evidence {
				recommendation.Evidence = append(recommendation.Evidence, DiscoveryEvidence{Type: "compat", Key: key, Message: message})
			}
		}

		evidence := buildDiscoveryEvidence(models, protocols, recommendation)
		c.JSON(http.StatusOK, ChannelDiscoveryResponse{
			Models:         models,
			Protocols:      protocols,
			Recommendation: recommendation,
			Evidence:       evidence,
		})
	}
}

func buildTransientDiscoveryChannel(req ChannelDiscoveryRequest) (*config.UpstreamConfig, error) {
	baseURLs := normalizeDiscoveryBaseURLs(req.BaseURL, req.BaseURLs)
	if len(baseURLs) == 0 {
		return nil, errors.New("baseUrl is required")
	}
	for _, baseURL := range baseURLs {
		if err := utils.ValidateBaseURL(baseURL); err != nil {
			return nil, err
		}
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return nil, errors.New("apiKey is required")
	}
	serviceType := strings.TrimSpace(req.ServiceType)
	if serviceType == "" {
		return nil, errors.New("serviceType is required")
	}

	return &config.UpstreamConfig{
		Name:               "临时发现渠道",
		ServiceType:        serviceType,
		BaseURL:            baseURLs[0],
		BaseURLs:           baseURLs,
		APIKeys:            []string{apiKey},
		AuthHeader:         strings.TrimSpace(req.AuthHeader),
		CustomHeaders:      cloneStringMap(req.CustomHeaders),
		ProxyURL:           strings.TrimSpace(req.ProxyURL),
		InsecureSkipVerify: req.InsecureSkipVerify,
		ModelMapping:       cloneStringMap(req.ModelMapping),
		ReasoningMapping:   cloneStringMap(req.ReasoningMapping),
	}, nil
}

func normalizeDiscoveryBaseURLs(baseURL string, baseURLs []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(baseURLs)+1)
	add := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	add(baseURL)
	for _, value := range baseURLs {
		add(value)
	}
	return result
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		dst[trimmedKey] = trimmedValue
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func selectDiscoveryModels(models []string, global map[string]config.UpstreamModelCapability) DiscoverySelectedModels {
	unique := uniqueDiscoveryModels(models)
	if len(unique) == 0 {
		return DiscoverySelectedModels{}
	}

	selected := DiscoverySelectedModels{
		Strong:  bestDiscoveryModel(unique, global, []string{"opus", "pro", "max", "ultra", "codex"}),
		Primary: bestDiscoveryModel(unique, global, []string{"sonnet", "gpt", "chat", "main"}),
		Fast:    bestDiscoveryModel(unique, global, []string{"haiku", "mini", "flash", "lite", "fast"}),
	}
	if selected.Primary == "" {
		selected.Primary = unique[0]
	}
	if selected.Strong == "" {
		selected.Strong = selected.Primary
	}
	if selected.Fast == "" {
		selected.Fast = selected.Primary
	}
	return selected
}

func uniqueDiscoveryModels(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	result := make([]string, 0, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func bestDiscoveryModel(models []string, global map[string]config.UpstreamModelCapability, keywords []string) string {
	best := ""
	bestScore := -1
	for _, model := range models {
		score := discoveryModelKeywordScore(model, keywords)
		if resolved := config.ResolveUpstreamCapability(model, nil, global); resolved.Capability.ContextWindowTokens > 0 {
			score += resolved.Capability.ContextWindowTokens / 100000
		}
		if score > bestScore {
			best = model
			bestScore = score
		}
	}
	if bestScore <= 0 {
		return ""
	}
	return best
}

func discoveryModelKeywordScore(model string, keywords []string) int {
	lower := strings.ToLower(model)
	score := 0
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			score += 10
		}
	}
	return score
}

func buildDiscoveryMappingRecommendation(
	channelKind string,
	selected DiscoverySelectedModels,
	successByProtocol map[string][]string,
	targetClients []string,
) DiscoveryRecommendation {
	successful := make(map[string]struct{})
	for _, model := range successByProtocol[channelKind] {
		successful[model] = struct{}{}
	}

	modelMapping := make(map[string]string)
	add := func(alias, model string) {
		if model == "" {
			return
		}
		if _, ok := successful[model]; !ok {
			return
		}
		modelMapping[alias] = model
	}

	switch channelKind {
	case "messages":
		add("opus", selected.Strong)
		add("sonnet", selected.Primary)
		add("haiku", selected.Fast)
		add("fable", selected.Strong)
	case "responses", "chat":
		add("gpt", selected.Primary)
		add("mini", selected.Fast)
		add("codex", firstSuccessfulDiscoveryModel(successful, selected.Strong, selected.Primary))
	case "gemini":
		add("gemini", selected.Primary)
		add("pro", selected.Strong)
		add("flash", selected.Fast)
	}

	return DiscoveryRecommendation{
		ChannelKind:     channelKind,
		ModelMapping:    modelMapping,
		SupportedModels: discoverySupportedModelPatterns(modelMapping, targetClients),
	}
}

func firstSuccessfulDiscoveryModel(successful map[string]struct{}, models ...string) string {
	for _, model := range models {
		if _, ok := successful[model]; ok {
			return model
		}
	}
	return ""
}

func discoverySupportedModelPatterns(modelMapping map[string]string, targetClients []string) []string {
	patterns := make([]string, 0, len(modelMapping))
	for alias := range modelMapping {
		patterns = append(patterns, alias+"*")
	}
	sort.Strings(patterns)
	return patterns
}

func discoverTransientModels(ctx context.Context, channel *config.UpstreamConfig, channelKind string, apiKey string) DiscoveryModelsResult {
	baseURL := capabilityTestBaseURL(channel)
	if baseURL == "" {
		return DiscoveryModelsResult{Source: "none", Warnings: []string{"base URL is empty"}}
	}

	modelsURL := discoveryModelsURL(baseURL, channelKind, channel.ServiceType)
	client := httpclient.GetManager().GetStandardClient(10*time.Second, channel.InsecureSkipVerify, channel.ProxyURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return DiscoveryModelsResult{Source: "models_endpoint", URL: modelsURL, Warnings: []string{err.Error()}}
	}
	utils.SetAuthenticationHeaderWithOverride(req.Header, apiKey, channel.AuthHeader)
	utils.ApplyCustomHeaders(req.Header, channel.CustomHeaders)

	resp, err := client.Do(req)
	if err != nil {
		return DiscoveryModelsResult{Source: "models_endpoint", URL: modelsURL, Warnings: []string{err.Error()}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return DiscoveryModelsResult{Source: "models_endpoint", URL: modelsURL, StatusCode: resp.StatusCode, Warnings: []string{err.Error()}}
	}

	result := DiscoveryModelsResult{
		Source:     "models_endpoint",
		URL:        modelsURL,
		StatusCode: resp.StatusCode,
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Warnings = []string{fmt.Sprintf("models endpoint returned HTTP %d", resp.StatusCode)}
		return result
	}
	result.Items = parseDiscoveryModels(body)
	if len(result.Items) == 0 {
		result.Warnings = []string{"models endpoint returned no parseable models"}
	}
	return result
}

func discoveryModelsURL(baseURL, channelKind, serviceType string) string {
	if channelKind == "gemini" || serviceType == "gemini" {
		return buildCapabilityTestURL(baseURL, "/v1beta", "/models")
	}
	if serviceType == "copilot" {
		return strings.TrimRight(strings.TrimSuffix(baseURL, "#"), "/") + "/models"
	}
	return buildCapabilityTestURL(baseURL, "/v1", "/models")
}

func parseDiscoveryModels(body []byte) []string {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}
	models := make([]string, 0)
	appendModel := func(value interface{}) {
		switch typed := value.(type) {
		case string:
			model := strings.TrimPrefix(strings.TrimSpace(typed), "models/")
			if model != "" {
				models = append(models, model)
			}
		case map[string]interface{}:
			for _, key := range []string{"id", "name", "model"} {
				if model, ok := typed[key].(string); ok {
					model = strings.TrimPrefix(strings.TrimSpace(model), "models/")
					if model != "" {
						models = append(models, model)
						return
					}
				}
			}
		}
	}
	if data, ok := raw["data"].([]interface{}); ok {
		for _, item := range data {
			appendModel(item)
		}
	}
	if data, ok := raw["models"].([]interface{}); ok {
		for _, item := range data {
			appendModel(item)
		}
	}
	return uniqueDiscoveryModels(models)
}

func fallbackDiscoveryProbeModels(channelKind, serviceType string) []string {
	if channelKind != "" {
		if models, err := getCapabilityProbeModels(channelKind); err == nil {
			return models
		}
	}
	if protocol, ok := normalizeServiceTypeToProtocol(serviceType); ok {
		if models, err := getCapabilityProbeModels(string(protocol)); err == nil {
			return models
		}
	}
	return []string{"gpt-5.4", "claude-sonnet-4-6", "gemini-3.5-flash"}
}

func discoveryProbeModels(selected DiscoverySelectedModels, all []string) []string {
	candidates := []string{selected.Strong, selected.Primary, selected.Fast}
	for _, model := range all {
		candidates = append(candidates, model)
		if len(candidates) >= 6 {
			break
		}
	}
	return uniqueDiscoveryModels(candidates)
}

func runDiscoveryProtocolProbes(ctx context.Context, channel *config.UpstreamConfig, models []string, cfgManager *config.ConfigManager) []DiscoveryProtocolResult {
	protocols := []string{"messages", "responses", "chat", "gemini"}
	results := make([]DiscoveryProtocolResult, 0, len(protocols))
	for _, protocol := range protocols {
		results = append(results, runDiscoveryProtocolProbe(ctx, channel, protocol, models, discoveryProbeTimeout, cfgManager))
	}
	return results
}

func runDiscoveryProtocolProbe(ctx context.Context, channel *config.UpstreamConfig, protocol string, models []string, timeout time.Duration, cfgManager *config.ConfigManager) DiscoveryProtocolResult {
	result := DiscoveryProtocolResult{Protocol: protocol}
	for _, model := range models {
		modelResult := executeModelTest(ctx, channel, protocol, model, timeout, "", cfgManager, -1, protocol, channel.APIKeys[0], nil)
		result.LatencyMs += modelResult.Latency
		if modelResult.Success {
			result.Success = true
			result.SuccessModels = append(result.SuccessModels, model)
		} else {
			result.FailedModels = append(result.FailedModels, model)
			if modelResult.Error != nil && result.Error == "" {
				result.Error = *modelResult.Error
			}
		}
	}
	if len(models) > 0 {
		result.LatencyMs = result.LatencyMs / int64(len(models))
	}
	return result
}

func discoverySuccessModelsByProtocol(protocols []DiscoveryProtocolResult) map[string][]string {
	result := make(map[string][]string, len(protocols))
	for _, protocol := range protocols {
		result[protocol.Protocol] = append([]string(nil), protocol.SuccessModels...)
	}
	return result
}

func discoveryCompatProbeModel(channelKind string, selected DiscoverySelectedModels, successByProtocol map[string][]string) string {
	successful := make(map[string]struct{})
	for _, model := range successByProtocol[channelKind] {
		if strings.TrimSpace(model) != "" {
			successful[model] = struct{}{}
		}
	}
	candidates := []string{selected.Primary, selected.Fast, selected.Strong}
	for _, model := range candidates {
		if _, ok := successful[model]; ok {
			return model
		}
	}
	if models := successByProtocol[channelKind]; len(models) > 0 {
		return models[0]
	}
	for _, model := range candidates {
		if strings.TrimSpace(model) != "" {
			return model
		}
	}
	return ""
}

func recommendDiscoveryChannelKind(requested string, targetClients []string, protocols []DiscoveryProtocolResult) string {
	success := make(map[string]bool, len(protocols))
	for _, protocol := range protocols {
		success[protocol.Protocol] = protocol.Success
	}
	if requested != "" && success[requested] {
		return requested
	}
	targetSet := make(map[string]bool, len(targetClients))
	for _, target := range targetClients {
		targetSet[strings.ToLower(strings.TrimSpace(target))] = true
	}
	if targetSet["codex"] {
		if success["responses"] {
			return "responses"
		}
		if success["chat"] {
			return "chat"
		}
	}
	if targetSet["claude-code"] || targetSet["claude"] {
		if success["messages"] {
			return "messages"
		}
	}
	for _, protocol := range []string{"responses", "messages", "chat", "gemini"} {
		if success[protocol] {
			return protocol
		}
	}
	return normalizeDiscoveryChannelKind(requested)
}

func normalizeDiscoveryChannelKind(channelKind string) string {
	switch strings.TrimSpace(channelKind) {
	case "messages", "responses", "chat", "gemini":
		return strings.TrimSpace(channelKind)
	default:
		return ""
	}
}

func buildDiscoveryEvidence(models DiscoveryModelsResult, protocols []DiscoveryProtocolResult, recommendation DiscoveryRecommendation) []DiscoveryEvidence {
	evidence := make([]DiscoveryEvidence, 0, len(protocols)+len(recommendation.Evidence)+1)
	if len(models.Items) > 0 {
		evidence = append(evidence, DiscoveryEvidence{Type: "models", Message: fmt.Sprintf("%s returned %d models", models.Source, len(models.Items))})
	}
	for _, warning := range models.Warnings {
		evidence = append(evidence, DiscoveryEvidence{Type: "models", Message: warning})
	}
	for _, protocol := range protocols {
		if protocol.Success {
			evidence = append(evidence, DiscoveryEvidence{Type: "protocol", Key: protocol.Protocol, Message: fmt.Sprintf("%d models passed", len(protocol.SuccessModels))})
		}
	}
	evidence = append(evidence, recommendation.Evidence...)
	return evidence
}
