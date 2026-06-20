package config

import "testing"

func TestResolveAgentModelProfile_CodexBuiltins(t *testing.T) {
	profile := ResolveAgentModelProfile("gpt-5.4", nil)
	if !profile.Known {
		t.Fatal("expected built-in gpt-5.4 profile")
	}
	if profile.Profile.ContextWindowTokens != 272000 {
		t.Fatalf("ContextWindowTokens = %d, want 272000", profile.Profile.ContextWindowTokens)
	}
	if profile.Profile.MaxContextWindowTokens != 1000000 {
		t.Fatalf("MaxContextWindowTokens = %d, want 1000000", profile.Profile.MaxContextWindowTokens)
	}
	if profile.Profile.TruncationMode != "tokens" {
		t.Fatalf("TruncationMode = %q, want tokens", profile.Profile.TruncationMode)
	}
}

func TestResolveAgentModelProfile_ClaudeBuiltins(t *testing.T) {
	profile := ResolveAgentModelProfile("claude-sonnet-4-6", nil)
	if !profile.Known {
		t.Fatal("expected built-in claude-sonnet-4-6 profile")
	}
	if profile.Profile.ContextWindowTokens != 1000000 {
		t.Fatalf("ContextWindowTokens = %d, want 1000000", profile.Profile.ContextWindowTokens)
	}
	if profile.Profile.MaxOutputTokens != 64000 {
		t.Fatalf("MaxOutputTokens = %d, want 64000", profile.Profile.MaxOutputTokens)
	}

	alias := ResolveAgentModelProfile("sonnet", nil)
	if !alias.Known {
		t.Fatal("expected built-in sonnet alias profile")
	}
	if alias.Profile.ContextWindowTokens != 1000000 {
		t.Fatalf("alias ContextWindowTokens = %d, want 1000000", alias.Profile.ContextWindowTokens)
	}
}

func TestResolveAgentModelProfile_GlobalOverrideWins(t *testing.T) {
	profile := ResolveAgentModelProfile("gpt-5.4", map[string]AgentModelProfile{
		"gpt-5.4": {ContextWindowTokens: 512000},
	})
	if !profile.Known || profile.Source != "global" {
		t.Fatalf("profile source = %q known=%v, want global known", profile.Source, profile.Known)
	}
	if profile.Profile.ContextWindowTokens != 512000 {
		t.Fatalf("ContextWindowTokens = %d, want 512000", profile.Profile.ContextWindowTokens)
	}
}

func TestResolveUpstreamCapability_UsesActualModelAfterMapping(t *testing.T) {
	upstream := &UpstreamConfig{
		ModelMapping: map[string]string{
			"agent": "claude-sonnet-4-6",
		},
	}

	resolved := ResolveUpstreamCapability("agent", upstream, nil)
	if !resolved.Known {
		t.Fatal("expected built-in upstream capability")
	}
	if resolved.ActualModel != "claude-sonnet-4-6" {
		t.Fatalf("ActualModel = %q, want claude-sonnet-4-6", resolved.ActualModel)
	}
	if resolved.Capability.ContextWindowTokens != 1000000 {
		t.Fatalf("ContextWindowTokens = %d, want 1000000", resolved.Capability.ContextWindowTokens)
	}
	if resolved.Capability.MaxOutputTokens != 64000 {
		t.Fatalf("MaxOutputTokens = %d, want 64000", resolved.Capability.MaxOutputTokens)
	}
}

func TestResolveUpstreamCapability_ChannelOverrideWins(t *testing.T) {
	upstream := &UpstreamConfig{
		ModelMapping: map[string]string{"agent": "claude-sonnet-4-6"},
		ModelCapabilities: map[string]UpstreamModelCapability{
			"claude-sonnet-4-6": {ContextWindowTokens: 200000, MaxOutputTokens: 32000},
		},
	}

	resolved := ResolveUpstreamCapability("agent", upstream, map[string]UpstreamModelCapability{
		"claude-sonnet-4-6": {ContextWindowTokens: 500000},
	})
	if resolved.Source != "channel" {
		t.Fatalf("source = %q, want channel", resolved.Source)
	}
	if resolved.Capability.ContextWindowTokens != 200000 {
		t.Fatalf("ContextWindowTokens = %d, want 200000", resolved.Capability.ContextWindowTokens)
	}
}

func TestResolveUpstreamCapability_KimiK27Builtin(t *testing.T) {
	upstream := &UpstreamConfig{
		ModelMapping: map[string]string{
			"agent": "Kimi-K2.7-Code-HighSpeed",
		},
	}

	resolved := ResolveUpstreamCapability("agent", upstream, nil)
	if !resolved.Known || resolved.Source != "builtin" {
		t.Fatalf("source = %q known=%v, want builtin known", resolved.Source, resolved.Known)
	}
	if resolved.Capability.ContextWindowTokens != 262144 {
		t.Fatalf("ContextWindowTokens = %d, want 262144", resolved.Capability.ContextWindowTokens)
	}
	if resolved.Capability.MaxOutputTokens != 262144 {
		t.Fatalf("MaxOutputTokens = %d, want 262144", resolved.Capability.MaxOutputTokens)
	}
	if resolved.Capability.DefaultOutputTokens != 32768 {
		t.Fatalf("DefaultOutputTokens = %d, want 32768", resolved.Capability.DefaultOutputTokens)
	}
	if resolved.Capability.RecommendedOutputTokens != 262144 {
		t.Fatalf("RecommendedOutputTokens = %d, want 262144", resolved.Capability.RecommendedOutputTokens)
	}
	if resolved.Capability.Pricing == nil || resolved.Capability.Pricing.OutputPrice == nil || *resolved.Capability.Pricing.OutputPrice != 4 {
		t.Fatalf("Pricing.OutputPrice = %#v, want 8", resolved.Capability.Pricing)
	}
}

func TestResolveUpstreamCapability_RequestModelFallback(t *testing.T) {
	upstream := &UpstreamConfig{
		ModelMapping: map[string]string{"agent-1m": "vendor-hidden-model"},
	}

	resolved := ResolveUpstreamCapability("agent-1m", upstream, map[string]UpstreamModelCapability{
		"agent-*": {ContextWindowTokens: 1000000},
	})
	if !resolved.Known || resolved.Source != "global" {
		t.Fatalf("source = %q known=%v, want global known", resolved.Source, resolved.Known)
	}
	if resolved.MatchedPattern != "agent-*" {
		t.Fatalf("MatchedPattern = %q, want agent-*", resolved.MatchedPattern)
	}
	if resolved.Capability.ContextWindowTokens != 1000000 {
		t.Fatalf("ContextWindowTokens = %d, want 1000000", resolved.Capability.ContextWindowTokens)
	}
}
