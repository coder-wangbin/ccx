package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
	if channel.BaseURL != "https://api.example.com/v1" {
		t.Fatalf("baseUrl = %q", channel.BaseURL)
	}
	if got := channel.GetAllBaseURLs(); len(got) != 1 || got[0] != "https://api.example.com" {
		t.Fatalf("canonical base urls = %#v", got)
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
	if rec.ModelMapping["codex"] == "actual-pro" {
		t.Fatalf("codex should not map to failed actual-pro: %#v", rec.ModelMapping)
	}
}

func TestChannelDiscoveryHandlerDiscoversTransientResponsesChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"actual-main"},{"id":"actual-mini"}]}`))
		case "/v1/responses":
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
				t.Fatalf("Authorization header = %q", got)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
			_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))

	body := []byte(`{
		"channelKind":"responses",
		"serviceType":"openai",
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test",
		"targetClients":["codex"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var resp ChannelDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Models.Source != "models_endpoint" {
		t.Fatalf("models source=%q", resp.Models.Source)
	}
	if resp.Recommendation.ChannelKind != "responses" {
		t.Fatalf("recommended channelKind=%q", resp.Recommendation.ChannelKind)
	}
	if resp.Recommendation.ModelMapping["gpt"] != "actual-main" {
		t.Fatalf("modelMapping=%#v", resp.Recommendation.ModelMapping)
	}
	var responsesOK bool
	for _, protocol := range resp.Protocols {
		if protocol.Protocol == "responses" && protocol.Success {
			responsesOK = true
			break
		}
	}
	if !responsesOK {
		t.Fatalf("protocols=%#v", resp.Protocols)
	}
}

func TestChannelDiscoveryCompatUsesDiscoveredActualModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var sawDefaultCompatModel bool
	var sawActualCompatToolProbe bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"actual-main"}]}`))
		case "/v1/responses":
			body, _ := io.ReadAll(r.Body)
			if bytes.Contains(body, []byte("gpt-5.4-mini")) {
				sawDefaultCompatModel = true
			}
			if bytes.Contains(body, []byte("image_generation")) && bytes.Contains(body, []byte("actual-main")) {
				sawActualCompatToolProbe = true
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
			_, _ = w.Write([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	router := gin.New()
	router.POST("/api/channel-discovery", ChannelDiscovery(nil))

	body := []byte(`{
		"channelKind":"responses",
		"serviceType":"responses",
		"baseUrls":["` + upstream.URL + `"],
		"apiKey":"sk-test",
		"targetClients":["codex"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/channel-discovery", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if sawDefaultCompatModel {
		t.Fatal("compat probe used default gpt-5.4-mini instead of discovered actual model")
	}
	if !sawActualCompatToolProbe {
		t.Fatal("expected image_generation compat probe to use discovered actual model")
	}
}
