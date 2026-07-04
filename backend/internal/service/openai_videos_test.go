//go:build unit

package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

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
