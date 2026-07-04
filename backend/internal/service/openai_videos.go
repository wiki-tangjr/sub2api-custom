package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	openAIVideosGenerationsEndpoint = "/v1/videos/generations"
	openAIVideosEndpoint            = "/v1/videos"
	openAIVideosEndpointPrefix      = "/v1/videos"
	openAIJimengEndpointPrefix      = "/v1/jimeng"
	openAIVideosGenerationsURL      = "https://api.openai.com/v1/videos/generations"
)

type OpenAIVideosRequest struct {
	Endpoint    string
	Model       string
	Stream      bool
	ContentType string
}

func ParseOpenAIVideoModel(body []byte) string {
	return strings.TrimSpace(gjson.GetBytes(body, "model").String())
}

func (s *OpenAIGatewayService) ParseOpenAIVideosRequest(c *gin.Context, body []byte) (*OpenAIVideosRequest, error) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return nil, errors.New("request context is required")
	}
	endpoint := normalizeOpenAIVideoEndpoint(c.Request.URL.Path)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported video endpoint: %s", c.Request.URL.Path)
	}
	model := ParseOpenAIVideoModel(body)
	if model == "" {
		model = strings.TrimSpace(c.Query("model"))
	}
	if model == "" && strings.EqualFold(c.Request.Method, http.MethodPost) && strings.Contains(endpoint, "/generations") {
		return nil, errors.New("model is required")
	}
	return &OpenAIVideosRequest{
		Endpoint:    endpoint,
		Model:       model,
		Stream:      gjson.GetBytes(body, "stream").Bool(),
		ContentType: strings.TrimSpace(c.GetHeader("Content-Type")),
	}, nil
}

func normalizeOpenAIVideoEndpoint(path string) string {
	path = "/" + strings.TrimLeft(strings.TrimSpace(path), "/")
	path = strings.TrimRight(path, "/")
	for _, prefix := range []string{"/backend-api/openai", "/openai"} {
		if path == prefix+openAIVideosEndpoint || strings.HasPrefix(path, prefix+"/v1/") {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	if path == openAIVideosEndpoint || strings.HasPrefix(path, openAIVideosEndpointPrefix+"/") || strings.HasPrefix(path, openAIJimengEndpointPrefix) {
		return path
	}
	return ""
}

func (s *OpenAIGatewayService) ForwardVideos(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	parsed *OpenAIVideosRequest,
	channelMappedModel string,
) (*OpenAIForwardResult, error) {
	if parsed == nil {
		return nil, fmt.Errorf("parsed videos request is required")
	}
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("unsupported video account type: %s", account.Type)
	}
	return s.forwardOpenAIVideosAPIKey(ctx, c, account, body, parsed, channelMappedModel)
}

func (s *OpenAIGatewayService) forwardOpenAIVideosAPIKey(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	parsed *OpenAIVideosRequest,
	channelMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		requestModel = mapped
	}
	upstreamModel := requestModel
	if requestModel != "" {
		upstreamModel = account.GetMappedModel(requestModel)
	}
	if strings.TrimSpace(upstreamModel) == "" {
		upstreamModel = requestModel
	}
	forwardEndpoint := openAIVideoUpstreamEndpoint(parsed.Endpoint, requestModel)
	forwardBody, forwardContentType, err := rewriteOpenAIVideoJSONBody(body, parsed.ContentType, upstreamModel, forwardEndpoint)
	if err != nil {
		return nil, err
	}
	upstreamCtx, releaseUpstreamCtx := detachStreamUpstreamContext(ctx, parsed.Stream)
	defer releaseUpstreamCtx()

	token, _, err := s.GetAccessToken(upstreamCtx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideosRequest(upstreamCtx, c, account, forwardBody, forwardContentType, token, forwardEndpoint)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, UpstreamURL: safeUpstreamURL(upstreamReq.URL.String()), Kind: "request_error", Message: safeErr})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), UpstreamURL: safeUpstreamURL(upstreamReq.URL.String()), Kind: "failover", Message: upstreamMsg})
			s.handleFailoverSideEffects(upstreamCtx, resp, account, respBody, upstreamModel)
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode)}
		}
		return s.handleOpenAIVideosErrorResponse(resp, c, account, upstreamModel)
	}
	defer func() { _ = resp.Body.Close() }()

	if parsed.Stream && isEventStreamResponse(resp.Header) {
		usage, firstTokenMs, err := s.handleOpenAIVideosStreamingResponse(resp, c, startTime)
		if err != nil {
			return nil, err
		}
		return &OpenAIForwardResult{RequestID: resp.Header.Get("x-request-id"), Usage: usage, Model: requestModel, UpstreamModel: upstreamModel, Stream: true, ResponseHeaders: resp.Header.Clone(), Duration: time.Since(startTime), FirstTokenMs: firstTokenMs}, nil
	}
	usage, err := s.handleOpenAIVideosNonStreamingResponse(resp, c)
	if err != nil {
		return nil, err
	}
	return &OpenAIForwardResult{RequestID: resp.Header.Get("x-request-id"), Usage: usage, Model: requestModel, UpstreamModel: upstreamModel, Stream: false, ResponseHeaders: resp.Header.Clone(), Duration: time.Since(startTime)}, nil
}

func (s *OpenAIGatewayService) buildOpenAIVideosRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, contentType string, token string, endpoint string) (*http.Request, error) {
	targetURL := buildOpenAIVideosURL("https://api.openai.com", endpoint)
	if endpoint == openAIVideosGenerationsEndpoint {
		targetURL = openAIVideosGenerationsURL
	}
	if baseURL := account.GetOpenAIBaseURL(); baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = buildOpenAIVideosURL(validatedURL, endpoint)
	}
	if c != nil && c.Request != nil && c.Request.URL != nil && c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(ctx, c.Request.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+token)
	for key, values := range c.Request.Header {
		if !openaiPassthroughAllowedHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func buildOpenAIVideosURL(base string, endpoint string) string {
	return buildOpenAIEndpointURL(base, endpoint)
}

func openAIVideoUpstreamEndpoint(endpoint string, model string) string {
	if isJimengVideoModel(model) && (endpoint == openAIVideosEndpoint || endpoint == openAIVideosGenerationsEndpoint || endpoint == openAIJimengEndpointPrefix+"/videos/generations") {
		return openAIVideosEndpoint
	}
	return endpoint
}

func isJimengVideoModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "video-ds-2.0", "video-ds-2.0-fast":
		return true
	default:
		return false
	}
}

func rewriteOpenAIVideoJSONBody(body []byte, contentType string, model string, endpoint string) ([]byte, string, error) {
	if !json.Valid(body) {
		return body, contentType, nil
	}
	rewritten := body
	model = strings.TrimSpace(model)
	if model != "" {
		var err error
		rewritten, err = sjson.SetBytes(rewritten, "model", model)
		if err != nil {
			return nil, "", fmt.Errorf("rewrite video request model: %w", err)
		}
	}
	if endpoint == openAIVideosEndpoint && isJimengVideoModel(model) {
		patched, err := patchJimengVideoCreateBody(rewritten)
		if err != nil {
			return nil, "", err
		}
		rewritten = patched
	}
	return rewritten, contentType, nil
}

func patchJimengVideoCreateBody(body []byte) ([]byte, error) {
	out := body
	if !gjson.GetBytes(out, "seconds").Exists() {
		for _, field := range []string{"duration", "second"} {
			value := strings.TrimSpace(gjson.GetBytes(out, field).String())
			if value == "" {
				continue
			}
			var err error
			out, err = sjson.SetBytes(out, "seconds", value)
			if err != nil {
				return nil, fmt.Errorf("rewrite jimeng video seconds: %w", err)
			}
			break
		}
	}
	for _, field := range []string{"duration", "second", "width", "height", "size", "mode", "model_name", "req_key"} {
		if !gjson.GetBytes(out, field).Exists() {
			continue
		}
		var err error
		out, err = sjson.DeleteBytes(out, field)
		if err != nil {
			return nil, fmt.Errorf("drop unsupported jimeng video field %s: %w", field, err)
		}
	}
	return out, nil
}

func (s *OpenAIGatewayService) handleOpenAIVideosErrorResponse(resp *http.Response, c *gin.Context, account *Account, upstreamModel string) (*OpenAIForwardResult, error) {
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, resp.Header.Get("x-request-id"))
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), UpstreamURL: safeUpstreamURL(resp.Request.URL.String()), Kind: "upstream_error", Message: upstreamMsg})
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)
	return nil, fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, upstreamMsg)
}

func (s *OpenAIGatewayService) handleOpenAIVideosNonStreamingResponse(resp *http.Response, c *gin.Context) (OpenAIUsage, error) {
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return OpenAIUsage{}, err
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)
	usage, _ := extractOpenAIUsageFromJSONBytes(body)
	return usage, nil
}

func (s *OpenAIGatewayService) handleOpenAIVideosStreamingResponse(resp *http.Response, c *gin.Context, startTime time.Time) (OpenAIUsage, *int, error) {
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "text/event-stream"
	}
	c.Status(resp.StatusCode)
	c.Header("Content-Type", contentType)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return OpenAIUsage{}, nil, fmt.Errorf("streaming is not supported by response writer")
	}
	usage := OpenAIUsage{}
	var firstTokenMs *int
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := strings.TrimSpace(string(line))
			if firstTokenMs == nil && trimmed != "" {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			if strings.HasPrefix(trimmed, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				if payload != "" && payload != "[DONE]" {
					mergeOpenAIUsage(&usage, []byte(payload))
				}
			} else if json.Valid(bytes.TrimSpace(line)) {
				mergeOpenAIUsage(&usage, bytes.TrimSpace(line))
			}
			if _, writeErr := c.Writer.Write(line); writeErr != nil {
				return usage, firstTokenMs, writeErr
			}
			flusher.Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return usage, firstTokenMs, err
		}
	}
	return usage, firstTokenMs, nil
}
