package handler

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

// Seedance 原生接口本地魔改（LOCAL CUSTOMIZATION）
//
// 目的：让"桥豆麻衣酱酱"等写死火山方舟 Seedance 原生协议的客户端能通过本站生成即梦视频。
// 客户端固定发送：
//   POST /seedance/v3/contents/generations/tasks
//   { "model": "seedance-2.0-fast-sdols",
//     "content": [ {"type":"text","text":"提示词 --resolution 720p --duration 15 --ratio 9:16"},
//                  {"type":"image_url","image_url":{"url":"..."}} ] }
// 上游 zz 仅支持 OpenAI 兼容 /v1/videos（model/prompt/seconds/aspect_ratio）。
// 因此本 handler 把 Seedance 原生 body 翻译成 /v1/videos body，再复用现有 Videos() 转发。
//
// 模型名映射：seedance-2.0-fast-sdols -> as-sd2.0-fast（可扩展）。

var seedanceModelMap = map[string]string{
	"seedance-2.0-fast-sdols": "as-sd2.0-fast",
	"seedance-2.0-sdols":      "as-sd2.0",
}

// --resolution / --duration / --ratio / --camerafixed 等命令行式参数
var (
	seedanceDurationRe   = regexp.MustCompile(`--duration\s+(\d+)`)
	seedanceRatioRe      = regexp.MustCompile(`--ratio\s+([0-9]+:[0-9]+)`)
	seedanceResolutionRe = regexp.MustCompile(`--resolution\s+(\S+)`)
	seedanceAnyFlagRe    = regexp.MustCompile(`\s*--[a-zA-Z]+\s+\S+`)
	// zz /v1/videos 只接受具体比例（如 9:16 / 16:9 / 1:1）；adaptive/auto 不合法
	seedanceValidRatioRe = regexp.MustCompile(`^[0-9]+:[0-9]+$`)
)

// SeedanceCreate 处理 POST /seedance/v3/contents/generations/tasks
func (h *OpenAIGatewayHandler) SeedanceCreate(c *gin.Context) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	_ = c.Request.Body.Close()

	// 临时记录真实请求体，便于确认客户端格式（LOCAL DEBUG，可后续移除）
	logger.L().Info("seedance.create.raw_body",
		zap.String("path", c.Request.URL.Path),
		zap.String("body", string(raw)),
	)

	translated, ok := translateSeedanceToVideos(raw)
	if !ok {
		// 转换失败时，原样透传给 Videos()（让其按 /v1/videos 逻辑处理并返回上游错误）
		translated = raw
	}

	// LOCAL CUSTOMIZATION: 安全审计门。seedance 携带用户提示词，需经官方 prompt 审计协调器（与 /v1/images 一致）。
	// 未配置审计/内容审核时 checkSecurityAudit 为 no-op（coordinator/legacy 均 nil 直接放行）。
	if apiKey, ok := middleware2.GetAPIKeyFromContext(c); ok {
		if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
			reqLog := requestLogger(c, "handler.openai_gateway.seedance", zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID))
			model := gjson.GetBytes(translated, "model").String()
			if decision := h.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, model, translated); decision != nil && !decision.AllowNextStage {
				h.openAISecurityAuditError(c, decision)
				return
			}
		}
	}

	// 用翻译后的 body 覆盖请求，并将路径改写为 /v1/videos 复用现有转发
	c.Request.Body = io.NopCloser(bytes.NewReader(translated))
	c.Request.ContentLength = int64(len(translated))
	c.Request.Header.Set("Content-Type", "application/json")
	// 改写路径为 /v1/videos，使 Videos() 转发到上游正确创建端点
	c.Request.URL.Path = "/v1/videos"

	h.Videos(c)
}

// SeedanceQuery 处理 GET /seedance/v3/contents/generations/tasks/:taskId
// 转发到现有 Videos()（其支持 /v1/videos/{id} 查询），并记录以便确认客户端期望的响应格式。
func (h *OpenAIGatewayHandler) SeedanceQuery(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("taskId"))
	taskID = strings.TrimPrefix(taskID, "/")
	logger.L().Info("seedance.query.raw",
		zap.String("path", c.Request.URL.Path),
		zap.String("task_id", taskID),
	)
	// 改写路径为 /v1/videos/{id}，复用现有查询转发
	if taskID != "" {
		c.Request.URL.Path = "/v1/videos/" + taskID
	}
	h.Videos(c)
}

// translateSeedanceToVideos 把 Seedance 原生 body 转成 /v1/videos body。
func translateSeedanceToVideos(raw []byte) ([]byte, bool) {
	if !gjson.ValidBytes(raw) {
		return raw, false
	}

	model := strings.TrimSpace(gjson.GetBytes(raw, "model").String())
	if mapped, ok := seedanceModelMap[model]; ok {
		model = mapped
	} else if model == "" {
		model = "as-sd2.0-fast"
	}

	// 收集 content 数组里的 text 与 image_url
	var promptText string
	var firstImage string
	content := gjson.GetBytes(raw, "content")
	if content.IsArray() {
		content.ForEach(func(_, item gjson.Result) bool {
			switch item.Get("type").String() {
			case "text":
				if t := item.Get("text").String(); t != "" && promptText == "" {
					promptText = t
				}
			case "image_url":
				if u := item.Get("image_url.url").String(); u != "" && firstImage == "" {
					firstImage = u
				}
			}
			return true
		})
	}
	// 兼容：若直接给了 prompt 字段
	if promptText == "" {
		promptText = gjson.GetBytes(raw, "prompt").String()
	}

	// 秒数：优先顶层 duration（桥豆麻衣酱发数字），其次 text 内 --duration，最后 seconds 字段。
	seconds := ""
	if d := gjson.GetBytes(raw, "duration"); d.Exists() && d.Int() > 0 {
		seconds = strconv.FormatInt(d.Int(), 10)
	} else if m := seedanceDurationRe.FindStringSubmatch(promptText); len(m) == 2 {
		seconds = m[1]
	} else if s := gjson.GetBytes(raw, "seconds").String(); s != "" {
		seconds = s
	}

	// 比例：优先顶层 ratio（桥豆麻衣酱），其次 text 内 --ratio，最后 aspect_ratio。
	// 上游 zz /v1/videos 只认具体比例（如 9:16）；"adaptive"/"auto" 等非法值一律省略，交给上游用默认。
	aspect := ""
	if r := strings.TrimSpace(gjson.GetBytes(raw, "ratio").String()); r != "" {
		aspect = r
	} else if m := seedanceRatioRe.FindStringSubmatch(promptText); len(m) == 2 {
		aspect = m[1]
	} else if a := strings.TrimSpace(gjson.GetBytes(raw, "aspect_ratio").String()); a != "" {
		aspect = a
	}
	if !seedanceValidRatioRe.MatchString(aspect) {
		aspect = "" // adaptive/auto/空 -> 不下发，上游用默认
	}

	// 剥离提示词末尾的 --flag value 参数，保留纯提示词
	cleanPrompt := strings.TrimSpace(seedanceAnyFlagRe.ReplaceAllString(promptText, ""))

	out := []byte(`{}`)
	out, _ = sjson.SetBytes(out, "model", model)
	out, _ = sjson.SetBytes(out, "prompt", cleanPrompt)
	// seconds 为字符串（zz 要求）；为空时不下发，交给上游默认。
	if seconds != "" {
		out, _ = sjson.SetBytes(out, "seconds", seconds)
	}
	if aspect != "" {
		out, _ = sjson.SetBytes(out, "aspect_ratio", aspect)
	}
	if firstImage != "" {
		// 首帧图：/v1/videos 兼容字段（若上游支持 image / input_image）
		out, _ = sjson.SetBytes(out, "image", firstImage)
	}
	_ = seedanceResolutionRe // resolution 暂不透传给上游（/v1/videos 不接受）
	return out, true
}
