package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	releasesCacheTTL      = 4 * time.Hour
	releasesErrorCacheTTL = 30 * time.Minute
	releasesAPIURL        = "https://api.github.com/repos/BenedictKing/ccx/releases?per_page=10"
	releasesTimeout       = 10 * time.Second
)

// ReleaseCheckResult 前端直接消费的版本检查结果。
type ReleaseCheckResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseURL     string `json:"releaseUrl"`
	Status         string `json:"status"` // "latest" | "update-available" | "error"
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
}

// prereleasePattern 匹配 -alpha, -beta, -rc, -dev, -pre, -canary, -nightly 后缀。
var prereleasePattern = regexp.MustCompile(`-(?i)(alpha|beta|rc|dev|pre|canary|nightly)`)

// DesktopService 扩展：release 检查的缓存字段

type releaseCheckCache struct {
	mu        sync.Mutex
	result    ReleaseCheckResult
	expiresAt time.Time
}

var releaseCache releaseCheckCache

// CheckLatestRelease 查询 GitHub Releases，返回是否有新版本。
// 结果在内存中缓存 30 分钟（错误状态缓存 5 分钟），避免频繁访问 GitHub API。
// 失败不返回 error，仅将 Status 置为 "error"，让前端安静忽略。
// 传入 force=true 时绕过缓存（用户主动「立即检查」场景）。
func (s *DesktopService) CheckLatestRelease(force bool) ReleaseCheckResult {
	releaseCache.mu.Lock()
	defer releaseCache.mu.Unlock()

	if !force && time.Now().Before(releaseCache.expiresAt) && releaseCache.result.Status != "" {
		return releaseCache.result
	}

	result := s.fetchLatestRelease()
	releaseCache.result = result
	if result.Status == "error" {
		releaseCache.expiresAt = time.Now().Add(releasesErrorCacheTTL)
	} else {
		releaseCache.expiresAt = time.Now().Add(releasesCacheTTL)
	}
	return result
}

func (s *DesktopService) fetchLatestRelease() ReleaseCheckResult {
	current := s.versionInfo.Version

	result := ReleaseCheckResult{
		CurrentVersion: current,
		Status:         "error",
	}

	if current == "" {
		return result
	}

	client := &http.Client{Timeout: releasesTimeout}
	resp, err := client.Get(releasesAPIURL)
	if err != nil {
		log.Printf("[Desktop-Updater] GitHub Releases 查询失败: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Desktop-Updater] GitHub Releases 返回状态码 %d", resp.StatusCode)
		return result
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		log.Printf("[Desktop-Updater] 解析 GitHub Releases 失败: %v", err)
		return result
	}

	// 过滤预发布版本，取第一个正式版
	for _, rel := range releases {
		if rel.Prerelease || prereleasePattern.MatchString(rel.TagName) {
			continue
		}
		result.LatestVersion = rel.TagName
		result.ReleaseURL = rel.HTMLURL

		cmp := compareVersions(current, rel.TagName)
		if cmp < 0 {
			result.HasUpdate = true
			result.Status = "update-available"
		} else {
			result.Status = "latest"
		}
		return result
	}

	// 没找到正式版（全被过滤了），当作 up-to-date
	result.Status = "latest"
	return result
}

// compareVersions 比较两个语义化版本号（可带 v 前缀）。
// 返回 -1: a < b, 0: a == b, 1: a > b。
func compareVersions(a, b string) int {
	aParts := parseVersionParts(a)
	bParts := parseVersionParts(b)

	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		aVal, bVal := 0, 0
		if i < len(aParts) {
			aVal = aParts[i]
		}
		if i < len(bParts) {
			bVal = bParts[i]
		}
		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}
	}
	return 0
}

func parseVersionParts(v string) []int {
	v = strings.TrimPrefix(v, "v")
	// 截断到 x.y.z，忽略后续的 -rc1 等预发布后缀（版本数值比较不关心预发布标识）
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	segments := strings.Split(v, ".")
	parts := make([]int, 0, len(segments))
	for _, s := range segments {
		n := 0
		for _, c := range s {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		parts = append(parts, n)
	}
	return parts
}
