package service

import (
	"strings"
)

// 本地魔改 #5 配套：Gemini / Veo 原生上游路径护栏。
//
// 官方 upstream_path_guard.go 的 sanitizedUpstreamPathSuffix 采用闭集允许清单
// （只放行 [A-Za-z0-9_.-]），这对 Responses 子路径够用，但 Gemini AI Studio 原生
// 接口的路径天然带两样它不允许的东西：
//
//  1. 冒号动作后缀：/v1beta/files/{file}:download、/v1beta/{operation}:cancel、
//     /v1beta/models/{model}:predictLongRunning
//  2. 查询串：?alt=media、?alt=json
//
// 因此这里提供一个 Gemini 专用版本，保持官方「默认拒绝」的思路不变，只在两点上放宽：
//   - 先把 query 从 path 上摘下来单独处理，护栏只校验 path 部分（query 位于 '?' 之后，
//     不可能再改变路径结构）；
//   - 每个路径片段最多允许一个 ':'，冒号两侧各自仍须通过官方的片段允许清单校验。
//
// 不放宽的部分与官方一致：仍然拒绝 ".."、空片段、纯点片段、超长片段、片段数超限，
// 以及允许清单外的任何字符（含控制字符与非 ASCII）。
// 只校验、不改写：不合规输入一律拒绝，不做静默修正。

// isSafeGeminiUpstreamPathSegment 在官方片段规则之上，额外允许「一个」冒号动作后缀。
func isSafeGeminiUpstreamPathSegment(segment string) bool {
	if segment == "" || len(segment) > maxUpstreamPathSegmentLen {
		return false
	}
	parts := strings.Split(segment, ":")
	if len(parts) > 2 {
		return false
	}
	for _, part := range parts {
		if !isSafeUpstreamPathSegment(part) {
			return false
		}
	}
	return true
}

// sanitizedGeminiUpstreamPath 校验 "/v1beta/..." 形态的上游路径（可带 ?query）。
// 返回值是可安全拼接到上游 base URL 之后的完整后缀（含原样保留的 query）。
// ok=false 表示必须拒绝该请求，调用方不得降级为空路径。
func sanitizedGeminiUpstreamPath(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	pathPart := trimmed
	queryPart := ""
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		pathPart = trimmed[:idx]
		queryPart = trimmed[idx:]
	}

	// query 里不允许再出现 '#'，避免把 fragment 带进上游请求。
	if strings.Contains(queryPart, "#") {
		return "", false
	}
	if strings.Contains(pathPart, "#") {
		return "", false
	}

	if !strings.HasPrefix(pathPart, "/") {
		return "", false
	}
	segments := strings.Split(strings.TrimPrefix(pathPart, "/"), "/")
	if len(segments) > maxUpstreamPathSegments {
		return "", false
	}
	for _, segment := range segments {
		if !isSafeGeminiUpstreamPathSegment(segment) {
			return "", false
		}
	}

	return pathPart + queryPart, true
}
