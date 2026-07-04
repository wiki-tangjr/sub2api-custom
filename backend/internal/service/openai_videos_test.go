//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardOpenAIVideosJimengUsesVideosRootAndPatchedBody(t *testing.T) {
	t.Setenv("SUB2API_ALLOW_UNSAFE_URL_OVERRIDES", "true")
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"video-ds-2.0-fast","prompt":"cat","duration":5,"aspect_ratio":"16:9","width":1280}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	account := &Account{
		ID:          34,
		Name:        "jimeng",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key",
			"base_url": "https://zz1cc.cc.cd/v1",
		},
	}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(`{"id":"task_123","status":"queued"}`)),
	}}
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}
	parsed := &OpenAIVideosRequest{Endpoint: "/v1/videos/generations", Model: "video-ds-2.0-fast", ContentType: "application/json"}

	_, err := svc.ForwardVideos(context.Background(), c, account, body, parsed, "video-ds-2.0-fast")
	require.NoError(t, err)
	require.Equal(t, "https://zz1cc.cc.cd/v1/videos", upstream.lastReq.URL.String())
	require.Equal(t, http.MethodPost, upstream.lastReq.Method)
	require.True(t, json.Valid(upstream.lastBody))
	require.Equal(t, "video-ds-2.0-fast", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "cat", gjson.GetBytes(upstream.lastBody, "prompt").String())
	require.Equal(t, "5", gjson.GetBytes(upstream.lastBody, "seconds").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "duration").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, "width").Exists())
}

func TestNormalizeOpenAIVideoEndpointAcceptsVideosRoot(t *testing.T) {
	t.Parallel()

	require.Equal(t, "/v1/videos", normalizeOpenAIVideoEndpoint("/v1/videos"))
	require.Equal(t, "/v1/videos", normalizeOpenAIVideoEndpoint("/v1/videos/"))
	require.Equal(t, "/v1/videos", normalizeOpenAIVideoEndpoint("/openai/v1/videos"))
	require.Equal(t, "/v1/videos/task_123/content", normalizeOpenAIVideoEndpoint("/v1/videos/task_123/content"))
}

func TestOpenAIVideoUpstreamEndpointMapsJimengGenerationsToVideosRoot(t *testing.T) {
	t.Parallel()

	require.Equal(t, "/v1/videos", openAIVideoUpstreamEndpoint("/v1/videos/generations", "video-ds-2.0-fast"))
	require.Equal(t, "/v1/videos", openAIVideoUpstreamEndpoint("/v1/jimeng/videos/generations", "video-ds-2.0"))
	require.Equal(t, "/v1/videos/generations", openAIVideoUpstreamEndpoint("/v1/videos/generations", "grok-imagine-video-1.5"))
}

func TestRewriteOpenAIVideoJSONBodyPatchesJimengCreateBody(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"video-ds-2.0-fast",
		"prompt":"cat",
		"duration":15,
		"aspect_ratio":"9:16",
		"width":720,
		"height":1280,
		"size":"720x1280",
		"model_name":"bad"
	}`)

	patched, contentType, err := rewriteOpenAIVideoJSONBody(body, "application/json", "video-ds-2.0-fast", "/v1/videos")
	require.NoError(t, err)
	require.Equal(t, "application/json", contentType)
	require.True(t, json.Valid(patched))
	require.Equal(t, "video-ds-2.0-fast", gjson.GetBytes(patched, "model").String())
	require.Equal(t, "cat", gjson.GetBytes(patched, "prompt").String())
	require.Equal(t, "15", gjson.GetBytes(patched, "seconds").String())
	require.Equal(t, "9:16", gjson.GetBytes(patched, "aspect_ratio").String())
	require.False(t, gjson.GetBytes(patched, "duration").Exists())
	require.False(t, gjson.GetBytes(patched, "width").Exists())
	require.False(t, gjson.GetBytes(patched, "height").Exists())
	require.False(t, gjson.GetBytes(patched, "size").Exists())
	require.False(t, gjson.GetBytes(patched, "model_name").Exists())
}
