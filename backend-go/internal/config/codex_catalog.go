package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// reasoningEffortPreset mirrors Codex's ReasoningEffortPreset struct.
type reasoningEffortPreset struct {
	Effort      string `json:"effort"`
	Description string `json:"description"`
}

// codexModelCatalogEntry mirrors Codex's ModelInfo JSON shape.
// Every field matches the serde defaults of the Rust ModelInfo so Codex can deserialise it.
type codexModelCatalogEntry struct {
	PreferWebsockets              bool                    `json:"prefer_websockets"`
	SupportVerbosity              bool                    `json:"support_verbosity"`
	DefaultVerbosity              *string                 `json:"default_verbosity"`
	ApplyPatchToolType            *string                 `json:"apply_patch_tool_type"`
	WebSearchToolType             string                  `json:"web_search_tool_type"`
	InputModalities               []string                `json:"input_modalities"`
	SupportsImageDetailOriginal   bool                    `json:"supports_image_detail_original"`
	TruncationPolicy              map[string]any          `json:"truncation_policy"`
	SupportsParallelToolCalls     bool                    `json:"supports_parallel_tool_calls"`
	ContextWindow                 *int                    `json:"context_window"`
	MaxContextWindow              *int                    `json:"max_context_window"`
	AutoCompactTokenLimit         *int                    `json:"auto_compact_token_limit"`
	ReasoningSummaryFormat        *string                 `json:"reasoning_summary_format"`
	DefaultReasoningSummary       string                  `json:"default_reasoning_summary"`
	Slug                          string                  `json:"slug"`
	DisplayName                   string                  `json:"display_name"`
	Description                   string                  `json:"description"`
	DefaultReasoningLevel         string                  `json:"default_reasoning_level"`
	SupportedReasoningLevels      []reasoningEffortPreset `json:"supported_reasoning_levels"`
	ShellType                     string                  `json:"shell_type"`
	Visibility                    string                  `json:"visibility"`
	MinimalClientVersion          string                  `json:"minimal_client_version"`
	SupportedInAPI                bool                    `json:"supported_in_api"`
	AvailabilityNux               *availabilityNux        `json:"availability_nux"`
	Upgrade                       any                     `json:"upgrade"`
	Priority                      int                     `json:"priority"`
	IncludeSkillsUsageInstructions bool                   `json:"include_skills_usage_instructions"`
	BaseInstructions              string                  `json:"base_instructions"`
	ModelMessages                 *modelMessages          `json:"model_messages"`
	ExperimentalSupportedTools    []string                `json:"experimental_supported_tools"`
	AvailableInPlans              []string                `json:"available_in_plans"`
	SupportsSearchTool            bool                    `json:"supports_search_tool"`
	ServiceTiers                  []serviceTier           `json:"service_tiers"`
	AdditionalSpeedTiers          []string                `json:"additional_speed_tiers"`
	SupportsReasoningSummaries    bool                    `json:"supports_reasoning_summaries"`
	EffectiveContextWindowPercent int                     `json:"-"`
}

type availabilityNux struct {
	Message string `json:"message"`
}

type modelMessages struct {
	Personality    string   `json:"personality"`
	ToolCategories []string `json:"tool_categories"`
	ToolGuidance   string   `json:"tool_guidance"`
}

type serviceTier struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type codexModelCatalog struct {
	Models []codexModelCatalogEntry `json:"models"`
}

const defaultBaseInstructions = "You are a coding agent running in the Codex CLI, a terminal-based coding assistant. Codex CLI is an open source project led by OpenAI. You are expected to be precise, safe, and helpful."

var defaultSupportedReasoningLevels = []reasoningEffortPreset{
	{Effort: "low", Description: "Fast responses with lighter reasoning"},
	{Effort: "medium", Description: "Balances speed and reasoning depth for everyday tasks"},
	{Effort: "high", Description: "Greater reasoning depth for complex problems"},
	{Effort: "xhigh", Description: "Extra high reasoning depth for complex problems"},
}

func extractModelIDsFromPatterns(caps map[string]UpstreamModelCapability) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, cap := range caps {
		name := strings.TrimSpace(cap.DisplayName)
		if name == "" {
			continue
		}
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		if idx := strings.LastIndex(slug, "-"); idx > 0 {
			suffix := slug[idx+1:]
			if len(suffix) >= 6 && isAllDigits(suffix) {
				slug = slug[:idx]
			}
		}
		if seen[slug] {
			continue
		}
		seen[slug] = true
		ids = append(ids, slug)
	}
	sort.Strings(ids)
	return ids
}

var wellKnownModelIDs = []string{
	"gpt", "gpt-5.4", "gpt-5.5", "gpt-5.2", "gpt-5.3-codex", "gpt-5.4-mini",
	"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna",
	"deepseek-chat", "deepseek-reasoner", "deepseek-v4-pro", "deepseek-v4-flash", "deepseek-v3.2",
	"claude-sonnet-4-5", "claude-opus-4-5", "claude-haiku-4-5",
	"claude-sonnet-4-6", "claude-opus-4-6", "claude-opus-4-7", "claude-opus-4-8",
	"claude-sonnet-5", "claude-fable-5", "claude-mythos-5",
	"mini", "fable", "opus", "sonnet", "haiku",
	"glm-5.2", "glm-5.1", "glm-5p2", "glm-5p1",
	"minimax-m2.7", "minimax-m2.5", "minimax-m2.1",
	"qwen3.6-plus", "qwen3.6-max",
	"kimi-k2.7",
	"mimo-v2.5", "mimo-v2.5-pro", "mimo-v2-flash",
	"ernie-4.5", "baichuan-m2", "yi-34b-200k", "longcat-2.0", "step-3.7-flash",
}

func newCatalogEntry(slug string) codexModelCatalogEntry {
	return codexModelCatalogEntry{
		PreferWebsockets:                false,
		SupportVerbosity:                false,
		DefaultVerbosity:                nil,
		ApplyPatchToolType:              nil,
		WebSearchToolType:               "text",
		InputModalities:                 []string{"text"},
		SupportsImageDetailOriginal:     false,
		TruncationPolicy:                map[string]any{"mode": "bytes", "limit": float64(10000)},
		SupportsParallelToolCalls:       false,
		ContextWindow:                   nil,
		MaxContextWindow:                nil,
		AutoCompactTokenLimit:           nil,
		ReasoningSummaryFormat:          nil,
		DefaultReasoningSummary:         "auto",
		Slug:                            slug,
		DisplayName:                     slug,
		Description:                     "Model via CCX proxy",
		DefaultReasoningLevel:           "medium",
		SupportedReasoningLevels:        defaultSupportedReasoningLevels,
		ShellType:                       "default",
		Visibility:                      "list",
		MinimalClientVersion:            "0.98.0",
		SupportedInAPI:                  true,
		AvailabilityNux:                 nil,
		Upgrade:                         nil,
		Priority:                        99,
		IncludeSkillsUsageInstructions:  false,
		BaseInstructions:                defaultBaseInstructions,
		ModelMessages:                   nil,
		ExperimentalSupportedTools:      []string{},
		AvailableInPlans:                []string{},
		SupportsSearchTool:              false,
		ServiceTiers:                    []serviceTier{},
		AdditionalSpeedTiers:            []string{},
		SupportsReasoningSummaries:      false,
	}
}

func GenerateCodexModelCatalog(caps map[string]UpstreamModelCapability) codexModelCatalog {
	ids := extractModelIDsFromPatterns(caps)
	merged := make(map[string]bool)
	for _, id := range ids {
		merged[id] = true
	}
	for _, id := range wellKnownModelIDs {
		merged[id] = true
	}
	allIDs := make([]string, 0, len(merged))
	for id := range merged {
		allIDs = append(allIDs, id)
	}
	sort.Strings(allIDs)

	entries := make([]codexModelCatalogEntry, 0, len(allIDs))
	for _, slug := range allIDs {
		entries = append(entries, newCatalogEntry(slug))
	}
	return codexModelCatalog{Models: entries}
}

func WriteCodexModelCatalog(stateDir string) error {
	caps := BuiltinUpstreamModelCapabilities()
	catalog := GenerateCodexModelCatalog(caps)

	path := filepath.Join(stateDir, "model_catalog.json")
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
