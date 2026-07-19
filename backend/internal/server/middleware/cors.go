package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

var corsWarningOnce sync.Once

// CORS 跨域中间件
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	allowedOrigins := normalizeOrigins(cfg.AllowedOrigins)
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}
	wildcardWithSpecific := allowAll && len(allowedOrigins) > 1
	if wildcardWithSpecific {
		allowedOrigins = []string{"*"}
	}
	allowCredentials := cfg.AllowCredentials

	corsWarningOnce.Do(func() {
		if len(allowedOrigins) == 0 {
			log.Println("Warning: CORS allowed_origins not configured; cross-origin requests will be rejected.")
		}
		if wildcardWithSpecific {
			log.Println("Warning: CORS allowed_origins includes '*'; wildcard will take precedence over explicit origins.")
		}
		if allowAll && allowCredentials {
			log.Println("Warning: CORS allowed_origins set to '*', disabling allow_credentials.")
		}
	})
	if allowAll && allowCredentials {
		allowCredentials = false
	}

	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "" || origin == "*" {
			continue
		}
		allowedSet[origin] = struct{}{}
	}
	allowHeaders := []string{
		"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization",
		"accept", "origin", "Cache-Control", "X-Requested-With", "X-API-Key", "X-Admin-UI-Request", "X-User-UI-Request",
	}
	// OpenAI Node SDK 会发送 x-stainless-* 请求头，需在 CORS 中显式放行。
	openAIProperties := []string{
		"lang", "package-version", "os", "arch", "retry-count", "runtime",
		"runtime-version", "async", "helper-method", "poll-helper", "custom-poll-interval", "timeout",
	}
	for _, prop := range openAIProperties {
		allowHeaders = append(allowHeaders, "x-stainless-"+prop)
	}
	allowHeadersValue := strings.Join(allowHeaders, ", ")

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		originAllowed := allowAll
		if origin != "" && !allowAll {
			_, originAllowed = allowedSet[origin]
		}
		// LOCAL CUSTOMIZATION: Seedance 原生接口（桥豆麻衣酱等浏览器客户端）需无条件放行跨域，
		// 否则浏览器 OPTIONS 预检被 403 拦截，真正的 POST 无法发出。
		if !originAllowed && strings.HasPrefix(c.Request.URL.Path, "/seedance/") {
			originAllowed = true
		}
		// LOCAL CUSTOMIZATION: OpenAI 兼容端点（/v1/*，如 chat/completions、responses、images、models）
		// 同样需无条件放行浏览器/WebView 客户端（桥豆麻衣酱等）的跨域预检，
		// 否则 OPTIONS 预检被 403 拦截，前端报 "Failed to fetch"，真正的 POST 无法发出。
		// 仅放行 CORS 预检与响应头，不改变后续鉴权/计费/转发逻辑（POST 仍照常校验 API key 并计费）。
		if !originAllowed && (strings.HasPrefix(c.Request.URL.Path, "/v1/") ||
			strings.HasPrefix(c.Request.URL.Path, "/videos") ||
			strings.HasPrefix(c.Request.URL.Path, "/jimeng/")) {
			originAllowed = true
		}

		if originAllowed {
			if allowAll {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Add("Vary", "Origin")
			}
			if allowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeadersValue)
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "ETag, Server-Timing")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}
		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			if originAllowed {
				c.AbortWithStatus(http.StatusNoContent)
			} else {
				c.AbortWithStatus(http.StatusForbidden)
			}
			return
		}

		c.Next()
	}
}

func normalizeOrigins(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
