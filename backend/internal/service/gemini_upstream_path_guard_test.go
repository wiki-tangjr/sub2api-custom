package service

import "testing"

// 本地魔改 #5 配套护栏测试：既要放行 Gemini/Veo 原生路径，也要挡住注入类输入。
func TestSanitizedGeminiUpstreamPath(t *testing.T) {
	allowed := []string{
		"/v1beta/files/video-1:download?alt=media",
		"/v1beta/operations/op-123?alt=json",
		"/v1beta/models/veo-3.0-generate-preview:predictLongRunning",
		"/v1beta/operations/op-123:cancel",
		"/v1beta/operations/op-123:wait",
		"/v1beta/models",
	}
	for _, raw := range allowed {
		got, ok := sanitizedGeminiUpstreamPath(raw)
		if !ok {
			t.Fatalf("expected path to be allowed: %q", raw)
		}
		if got != raw {
			t.Fatalf("expected path to be preserved verbatim: got %q want %q", got, raw)
		}
	}

	rejected := []string{
		"",
		"   ",
		"v1beta/models",                      // 缺少前导斜杠
		"/v1beta/../secrets",                 // 路径穿越
		"/v1beta//models",                    // 空片段
		"/v1beta/files/a:b:c",                // 多个冒号
		"/v1beta/files/:download",            // 冒号左侧为空
		"/v1beta/files/video-1:",             // 冒号右侧为空
		"/v1beta/files/vi deo",               // 空格
		"/v1beta/files/video#frag",           // fragment
		"/v1beta/files/video?alt=media#frag", // query 内 fragment
		"/v1beta/files/../../etc/passwd",     // 穿越
		"/v1beta/models/mod\x00el",           // 控制字符
		"/a/b/c/d/e/f/g/h/i",                 // 片段数超限
	}
	for _, raw := range rejected {
		if _, ok := sanitizedGeminiUpstreamPath(raw); ok {
			t.Fatalf("expected path to be rejected: %q", raw)
		}
	}
}
