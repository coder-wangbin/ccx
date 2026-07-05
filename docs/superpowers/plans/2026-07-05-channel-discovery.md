# Channel Discovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a transient channel discovery flow that tests baseURL/key without saving a channel, then recommends usable protocols, model mappings, and compatibility switches.

**Architecture:** Add a focused backend discovery handler that converts request JSON into a temporary `config.UpstreamConfig`, discovers models, probes text protocols, runs existing compat diagnosis, and returns a recommendation object. Add a small frontend service/type layer and wire the edit channel modal to call the endpoint and apply recommendations to the draft form only.

**Tech Stack:** Go + Gin backend, existing CCX provider/capability helpers, Vue 3 + TypeScript + Vuetify frontend, existing locale JSON.

## Global Constraints

- Always respond and document user-facing work in Simplified Chinese.
- Discovery must not write `.config/config.json`, create scheduler channels, or change circuit-breaker state.
- Discovery supports `messages`, `responses`, `chat`, and `gemini`; it does not infer capability-test support for `images` or `vectors`.
- No production code before a failing test for backend behavior.
- Do not stop or kill the project process on port 3688.
- Do not push; local commit only after verification.

---

### Task 1: Backend Discovery DTO and Recommendation Core

**Files:**
- Create: `backend-go/internal/handlers/channel_discovery.go`
- Create: `backend-go/internal/handlers/channel_discovery_test.go`
- Modify: `backend-go/main.go`

**Interfaces:**
- Produces: `ChannelDiscovery(cfgManager *config.ConfigManager) gin.HandlerFunc`
- Produces: `buildTransientDiscoveryChannel(req ChannelDiscoveryRequest) (*config.UpstreamConfig, error)`
- Produces: `selectDiscoveryModels(models []string, global map[string]config.UpstreamModelCapability) DiscoverySelectedModels`
- Produces: `buildDiscoveryMappingRecommendation(channelKind string, selected DiscoverySelectedModels, successByProtocol map[string][]string, targetClients []string) DiscoveryRecommendation`

- [ ] **Step 1: Write failing tests for transient config and mapping**

Add tests in `backend-go/internal/handlers/channel_discovery_test.go`:

```go
func TestBuildTransientDiscoveryChannelRequiresBaseURLAndKey(t *testing.T) {
	req := ChannelDiscoveryRequest{ServiceType: "openai"}
	_, err := buildTransientDiscoveryChannel(req)
	if err == nil {
		t.Fatal("expected error for missing base URL and api key")
	}
}

func TestBuildTransientDiscoveryChannelDoesNotNeedConfigManager(t *testing.T) {
	req := ChannelDiscoveryRequest{
		ChannelKind:        "responses",
		ServiceType:        "openai",
		BaseURLs:           []string{"https://api.example.com/v1"},
		APIKey:             "sk-test",
		AuthHeader:         "bearer",
		CustomHeaders:      map[string]string{"X-Test": "yes"},
		ProxyURL:           "http://127.0.0.1:8080",
		InsecureSkipVerify: true,
		ModelMapping:       map[string]string{"gpt": "actual-main"},
	}
	channel, err := buildTransientDiscoveryChannel(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel.Name != "临时发现渠道" || channel.ServiceType != "openai" {
		t.Fatalf("unexpected channel identity: %#v", channel)
	}
	if got := channel.GetAllBaseURLs(); len(got) != 1 || got[0] != "https://api.example.com/v1" {
		t.Fatalf("base urls = %#v", got)
	}
	if len(channel.APIKeys) != 1 || channel.APIKeys[0] != "sk-test" {
		t.Fatalf("api keys = %#v", channel.APIKeys)
	}
	if !channel.InsecureSkipVerify || channel.ProxyURL == "" || channel.AuthHeader != "bearer" {
		t.Fatalf("transport fields not copied: %#v", channel)
	}
	if channel.ModelMapping["gpt"] != "actual-main" {
		t.Fatalf("model mapping not copied: %#v", channel.ModelMapping)
	}
}

func TestBuildDiscoveryMappingRecommendationUsesOnlySuccessfulModels(t *testing.T) {
	selected := DiscoverySelectedModels{Strong: "actual-pro", Primary: "actual-main", Fast: "actual-mini"}
	successByProtocol := map[string][]string{"responses": {"actual-main", "actual-mini"}}
	rec := buildDiscoveryMappingRecommendation("responses", selected, successByProtocol, []string{"codex"})
	if rec.ChannelKind != "responses" {
		t.Fatalf("channelKind=%q", rec.ChannelKind)
	}
	if rec.ModelMapping["gpt"] != "actual-main" || rec.ModelMapping["mini"] != "actual-mini" {
		t.Fatalf("unexpected mapping: %#v", rec.ModelMapping)
	}
	if _, ok := rec.ModelMapping["codex"]; ok {
		t.Fatalf("codex should not map to failed actual-pro: %#v", rec.ModelMapping)
	}
}
```

- [ ] **Step 2: Verify tests fail**

Run:

```bash
cd "backend-go" && GOCACHE="/tmp/go-build" go test ./internal/handlers -run 'TestBuildTransientDiscoveryChannel|TestBuildDiscoveryMappingRecommendation' -count=1
```

Expected: FAIL because discovery types/functions are undefined.

- [ ] **Step 3: Implement DTO and pure recommendation core**

Create `channel_discovery.go` with request/response structs, validation, model tier selection, successful-model filtering, and a placeholder `ChannelDiscovery` handler that returns `501` until Task 2 fills orchestration.

- [ ] **Step 4: Verify tests pass**

Run:

```bash
cd "backend-go" && GOCACHE="/tmp/go-build" go test ./internal/handlers -run 'TestBuildTransientDiscoveryChannel|TestBuildDiscoveryMappingRecommendation' -count=1
```

Expected: PASS.

### Task 2: Backend Models Discovery and Protocol Probing

**Files:**
- Modify: `backend-go/internal/handlers/channel_discovery.go`
- Modify: `backend-go/internal/handlers/channel_discovery_test.go`
- Modify: `backend-go/main.go`

**Interfaces:**
- Consumes: `buildTransientDiscoveryChannel`
- Produces: `discoverTransientModels(ctx context.Context, channel *config.UpstreamConfig, channelKind string, apiKey string) DiscoveryModelsResult`
- Produces: `runDiscoveryProtocolProbe(ctx context.Context, channel *config.UpstreamConfig, protocol string, models []string, timeout time.Duration, cfgManager *config.ConfigManager) DiscoveryProtocolResult`
- Produces: `ChannelDiscovery(cfgManager *config.ConfigManager) gin.HandlerFunc`

- [ ] **Step 1: Write failing HTTP handler test**

Add a test with `httptest.Server` responding to `/v1/models` and `/v1/responses`, then POST `/api/channel-discovery` through a Gin router. Assert:

```go
if resp.Recommendation.ChannelKind != "responses" { t.Fatalf(...) }
if resp.Models.Source != "models_endpoint" { t.Fatalf(...) }
if resp.Recommendation.ModelMapping["gpt"] != "actual-main" { t.Fatalf(...) }
```

- [ ] **Step 2: Verify handler test fails**

Run:

```bash
cd "backend-go" && GOCACHE="/tmp/go-build" go test ./internal/handlers -run TestChannelDiscoveryHandler -count=1
```

Expected: FAIL because handler returns `501` or discovery orchestration is missing.

- [ ] **Step 3: Implement model discovery, protocol probe, and route**

Use `buildCapabilityTestURL`, `utils.SetAuthenticationHeaderWithOverride`, `httpclient.GetManager().GetStandardClient`, `executeModelTest` with empty `jobID`, nil log store, and `runCompatDiagnose` after recommendation. Register:

```go
apiGroup.POST("/channel-discovery", handlers.ChannelDiscovery(cfgManager))
```

- [ ] **Step 4: Verify backend discovery tests pass**

Run:

```bash
cd "backend-go" && GOCACHE="/tmp/go-build" go test ./internal/handlers -run 'TestChannelDiscovery|TestBuildTransientDiscoveryChannel|TestBuildDiscoveryMappingRecommendation' -count=1
```

Expected: PASS.

### Task 3: Frontend Service Types and API Method

**Files:**
- Modify: `frontend/src/services/api-types.ts`
- Modify: `frontend/src/services/api.ts`

**Interfaces:**
- Produces: `ChannelDiscoveryRequest`
- Produces: `ChannelDiscoveryResponse`
- Produces: `api.discoverChannelConfig(request: ChannelDiscoveryRequest): Promise<ChannelDiscoveryResponse>`

- [ ] **Step 1: Add TypeScript types**

Add request/response interfaces mirroring the backend response, including `recommendation.modelMapping`, `recommendation.compat`, `models.selected`, and `evidence`.

- [ ] **Step 2: Add API method**

Add:

```ts
async discoverChannelConfig(request: ChannelDiscoveryRequest): Promise<ChannelDiscoveryResponse> {
  return this.request('/channel-discovery', {
    method: 'POST',
    body: JSON.stringify(request)
  })
}
```

- [ ] **Step 3: Verify frontend type/build still compiles after Task 4**

This task is verified together with Task 4 using `cd "frontend" && bun run build`.

### Task 4: Frontend Modal Integration

**Files:**
- Modify: `frontend/src/composables/useEditChannelModal.ts`
- Modify: `frontend/src/components/EditChannelModal.vue`
- Modify: `frontend/src/locales/zh-CN.json`
- Modify: `frontend/src/locales/en.json`
- Modify: `frontend/src/locales/id.json`

**Interfaces:**
- Consumes: `apiService.discoverChannelConfig`
- Produces: `discoveringChannelConfig`, `channelDiscoveryResult`, `handleDiscoverChannelConfig`, `applyChannelDiscoveryRecommendation`

- [ ] **Step 1: Add discovery state and request builder**

Build request from current draft form: `channelType`, `serviceType`, `baseUrlsText`, first API key, `authHeader`, `customHeaders`, `proxyUrl`, `insecureSkipVerify`, `modelMapping`, and `reasoningMapping`.

- [ ] **Step 2: Apply recommendation to draft only**

Update `form.serviceType`, `baseUrlsText`, `form.supportedModels`, compatibility switches, `modelMappingRows`, and `form.reasoningMapping`. Do not call save.

- [ ] **Step 3: Add compact result panel**

Place a button near the auth/model section and show recommended protocol, selected models, model mapping count, compat count, warnings/evidence, and an “apply recommendation” button.

- [ ] **Step 4: Add locale keys**

Add `channelDiscovery.*` keys for Chinese, English, and Indonesian files.

- [ ] **Step 5: Verify frontend build**

Run:

```bash
cd "frontend" && bun run build
```

Expected: PASS.

### Task 5: Final Verification and Commit

**Files:**
- All changed files from previous tasks.

- [ ] **Step 1: Run backend focused tests**

```bash
cd "backend-go" && GOCACHE="/tmp/go-build" go test ./internal/handlers -run 'TestChannelDiscovery|TestBuildTransientDiscoveryChannel|TestBuildDiscoveryMappingRecommendation' -count=1
```

- [ ] **Step 2: Run backend package tests**

```bash
cd "backend-go" && GOCACHE="/tmp/go-build" go test ./internal/handlers ./internal/config
```

- [ ] **Step 3: Run frontend build**

```bash
cd "frontend" && bun run build
```

- [ ] **Step 4: Commit related files**

```bash
git add -- backend-go/internal/handlers/channel_discovery.go backend-go/internal/handlers/channel_discovery_test.go backend-go/main.go frontend/src/services/api-types.ts frontend/src/services/api.ts frontend/src/composables/useEditChannelModal.ts frontend/src/components/EditChannelModal.vue frontend/src/locales/zh-CN.json frontend/src/locales/en.json frontend/src/locales/id.json docs/superpowers/plans/2026-07-05-channel-discovery.md
git commit -m "feat: add transient channel discovery"
```
