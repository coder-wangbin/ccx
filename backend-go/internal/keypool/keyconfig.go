package keypool

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/ratelimit"
)

type Candidate struct {
	APIKey     string
	Config     config.APIKeyConfig
	Index      int
	Scope      string
	QuotaGroup string
}

type Selection struct {
	APIKey         string
	CredentialID   string
	CredentialName string
	QuotaGroup     string
	LimiterScope   string
	Config         config.APIKeyConfig
}

func HasEffectiveConfig(upstream *config.UpstreamConfig) bool {
	if upstream == nil {
		return false
	}
	for _, cfg := range upstream.APIKeyConfigs {
		if strings.TrimSpace(cfg.Name) != "" || cfg.Enabled != nil || strings.TrimSpace(cfg.QuotaGroup) != "" ||
			cfg.RateLimitRPM > 0 || cfg.RateLimitWindowMinutes > 0 || cfg.RateLimitMaxConcurrent > 0 ||
			cfg.RateLimitAutoFromHeaders != nil || cfg.Weight > 0 || len(cfg.Models) > 0 {
			return true
		}
	}
	return false
}

func Candidates(upstream *config.UpstreamConfig, failedKeys map[string]bool) []Candidate {
	if upstream == nil || len(upstream.APIKeys) == 0 {
		return nil
	}

	configs := config.NormalizeAPIKeyConfigsForView(*upstream)
	byKey := make(map[string]config.APIKeyConfig, len(configs))
	for _, cfg := range configs {
		byKey[cfg.Key] = cfg
	}

	out := make([]Candidate, 0, len(upstream.APIKeys))
	for i, key := range upstream.APIKeys {
		key = strings.TrimSpace(key)
		if key == "" || failedKeys[key] {
			continue
		}
		cfg := byKey[key]
		if cfg.Key == "" {
			cfg.Key = key
		}
		if cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		quotaGroup := strings.TrimSpace(cfg.QuotaGroup)
		scope := "key:" + stableKeyID(key)
		if quotaGroup != "" {
			scope = "quota:" + stableKeyID("quota:"+quotaGroup)
		}
		out = append(out, Candidate{
			APIKey:     key,
			Config:     cfg,
			Index:      i,
			Scope:      scope,
			QuotaGroup: quotaGroup,
		})
	}
	return out
}

func ConfigForCandidate(channel config.UpstreamConfig, cfg config.APIKeyConfig) ratelimit.Config {
	rpm := cfg.RateLimitRPM
	if rpm <= 0 {
		rpm = channel.RateLimitRPM
	}
	windowMinutes := cfg.RateLimitWindowMinutes
	if windowMinutes <= 0 {
		windowMinutes = channel.RateLimitWindowMinutes
	}
	maxConcurrent := cfg.RateLimitMaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = channel.RateLimitMaxConcurrent
	}
	autoFromHeaders := channel.IsRateLimitAutoFromHeadersEnabled()
	if cfg.RateLimitAutoFromHeaders != nil {
		autoFromHeaders = *cfg.RateLimitAutoFromHeaders
	}
	return ratelimit.Config{
		RPM:             rpm,
		WindowSeconds:   windowMinutes * 60,
		MaxConcurrent:   maxConcurrent,
		AutoFromHeaders: autoFromHeaders,
	}
}

func stableKeyID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:16]
}
