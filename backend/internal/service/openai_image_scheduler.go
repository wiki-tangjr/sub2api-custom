package service

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	openAIImageLatencySchedulerSettingKey = "openai_image_latency_scheduler_enabled"
	openAIImageLatencySchedulerCacheTTL   = 5 * time.Second
	openAIImageHealthStatsCacheTTL        = 60 * time.Second
	openAIImageHealthStatsLookback        = 7 * 24 * time.Hour
	openAIImageHealthStatsDBTimeout       = 2 * time.Second
	openAIImageSlowCooldown               = 30 * time.Minute
	openAIImageRateLimitCooldown          = 15 * time.Second
	openAIImageSlowEditsThresholdMs       = int64(180000)
	openAIImageSlowGenerationsThresholdMs = int64(220000)
	openAIImageVerySlowThresholdMs        = int64(240000)
	openAIImageHealthPenalty              = 2.5
	openAIImageHealthBonus                = 1.0
	openAIImageMaxConcurrency             = 1
)

type cachedOpenAIImageLatencySchedulerSetting struct {
	enabled   bool
	expiresAt int64
}

type openAIImageAccountHealth struct {
	applicable      bool
	slow            bool
	recentP95Ms     int64
	scoreAdjustment float64
	cooldownUntil   time.Time
	reason          string
}

type OpenAIImageAccountLatencyStats struct {
	AccountID     int64
	RequestCount  int
	AvgMs         int64
	P50Ms         int64
	P90Ms         int64
	P95Ms         int64
	MaxMs         int64
	SlowCount     int
	VerySlowCount int
}

type openAIImageLatencyStatsRepository interface {
	GetOpenAIImageAccountLatencyStats(ctx context.Context, endpoint string, model string, since time.Time) (map[int64]OpenAIImageAccountLatencyStats, error)
}

type openAIImageAccountCooldown struct {
	until  time.Time
	reason string
}

type openAIImageHealthStatsCacheEntry struct {
	stats     map[int64]OpenAIImageAccountLatencyStats
	expiresAt time.Time
}

type openAIImageHealthState struct {
	cooldowns sync.Map // key: endpoint|model|account_id, value: openAIImageAccountCooldown
	cache     sync.Map // key: endpoint|model, value: *openAIImageHealthStatsCacheEntry
	sf        singleflight.Group
}

var openAIImageLatencySchedulerSettingCache atomic.Value // *cachedOpenAIImageLatencySchedulerSetting
var openAIImageLatencySchedulerSettingSF singleflight.Group

func newOpenAIImageHealthState() *openAIImageHealthState {
	return &openAIImageHealthState{}
}

func resetOpenAIImageLatencySchedulerSettingCacheForTest() {
	openAIImageLatencySchedulerSettingCache = atomic.Value{}
	openAIImageLatencySchedulerSettingSF = singleflight.Group{}
}

func (s *OpenAIGatewayService) isOpenAIImageLatencySchedulerEnabled(ctx context.Context) bool {
	if cached, ok := openAIImageLatencySchedulerSettingCache.Load().(*cachedOpenAIImageLatencySchedulerSetting); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.enabled
		}
	}

	result, _, _ := openAIImageLatencySchedulerSettingSF.Do(openAIImageLatencySchedulerSettingKey, func() (any, error) {
		if cached, ok := openAIImageLatencySchedulerSettingCache.Load().(*cachedOpenAIImageLatencySchedulerSetting); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.enabled, nil
			}
		}

		enabled := false
		repo := (*SettingService)(nil)
		if s != nil {
			repo = s.settingService
			if repo == nil && s.rateLimitService != nil {
				repo = s.rateLimitService.settingService
			}
		}
		if repo != nil && repo.settingRepo != nil {
			dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIImageHealthStatsDBTimeout)
			defer cancel()

			value, err := repo.settingRepo.GetValue(dbCtx, openAIImageLatencySchedulerSettingKey)
			if err == nil {
				enabled = strings.EqualFold(strings.TrimSpace(value), "true")
			}
		}

		openAIImageLatencySchedulerSettingCache.Store(&cachedOpenAIImageLatencySchedulerSetting{
			enabled:   enabled,
			expiresAt: time.Now().Add(openAIImageLatencySchedulerCacheTTL).UnixNano(),
		})
		return enabled, nil
	})

	enabled, _ := result.(bool)
	return enabled
}

func openAIImageCapabilityFromEndpoint(endpoint string) OpenAIImagesCapability {
	if normalizeOpenAIImageEndpoint(endpoint) == "" {
		return ""
	}
	return OpenAIImagesCapabilityNative
}

func normalizeOpenAIImageEndpoint(endpoint string) string {
	switch strings.TrimSpace(endpoint) {
	case "/v1/images/edits", "/v1/images/generations":
		return strings.TrimSpace(endpoint)
	default:
		return ""
	}
}

func normalizeOpenAIImageModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "gpt-image-2"
	}
	return model
}

func openAIImageHealthKey(endpoint string, model string) string {
	endpoint = normalizeOpenAIImageEndpoint(endpoint)
	if endpoint == "" {
		return ""
	}
	return endpoint + "|" + normalizeOpenAIImageModel(model)
}

func openAIImageAccountHealthKey(endpoint string, model string, accountID int64) string {
	key := openAIImageHealthKey(endpoint, model)
	if key == "" || accountID <= 0 {
		return ""
	}
	return key + "|" + strconv.FormatInt(accountID, 10)
}

func openAIImageSlowThresholdMs(endpoint string) int64 {
	switch normalizeOpenAIImageEndpoint(endpoint) {
	case "/v1/images/edits":
		return openAIImageSlowEditsThresholdMs
	case "/v1/images/generations":
		return openAIImageSlowGenerationsThresholdMs
	default:
		return openAIImageVerySlowThresholdMs
	}
}

func (s *OpenAIGatewayService) openAIImageAccountHealth(req OpenAIAccountScheduleRequest, accountID int64) openAIImageAccountHealth {
	if s == nil || accountID <= 0 || normalizeOpenAIImageEndpoint(req.ImageEndpoint) == "" {
		return openAIImageAccountHealth{}
	}
	if !s.isOpenAIImageLatencySchedulerEnabled(context.Background()) {
		return openAIImageAccountHealth{}
	}

	health := openAIImageAccountHealth{applicable: true}
	if cooldown, ok := s.openAIImageAccountCooldown(req, accountID); ok {
		health.slow = true
		health.cooldownUntil = cooldown.until
		health.reason = cooldown.reason
		health.scoreAdjustment -= openAIImageHealthPenalty * 2
		return health
	}

	stats := s.openAIImageLatencyStats(context.Background(), req.ImageEndpoint, req.RequestedModel)
	stat, ok := stats[accountID]
	if !ok || stat.RequestCount <= 0 {
		return health
	}
	health.recentP95Ms = stat.P95Ms
	slowThreshold := openAIImageSlowThresholdMs(req.ImageEndpoint)
	if stat.P95Ms >= slowThreshold || stat.VerySlowCount > 0 {
		health.slow = true
		health.reason = "recent_image_latency"
		health.scoreAdjustment -= openAIImageHealthPenalty
	} else if stat.P95Ms > 0 && stat.P95Ms < slowThreshold {
		health.scoreAdjustment += openAIImageHealthBonus * (1 - clamp01(float64(stat.P95Ms)/float64(slowThreshold)))
	}
	return health
}

func (s *OpenAIGatewayService) isOpenAIImageAccountCooling(req OpenAIAccountScheduleRequest, accountID int64) bool {
	health := s.openAIImageAccountHealth(req, accountID)
	return health.applicable && health.slow && !health.cooldownUntil.IsZero() && time.Now().Before(health.cooldownUntil)
}

func (s *OpenAIGatewayService) shouldBypassOpenAIImageSticky(req OpenAIAccountScheduleRequest, accountID int64) bool {
	health := s.openAIImageAccountHealth(req, accountID)
	return health.applicable && health.slow
}

func (s *OpenAIGatewayService) openAIImageAccountCooldown(req OpenAIAccountScheduleRequest, accountID int64) (openAIImageAccountCooldown, bool) {
	if s == nil || s.openaiImageHealth == nil {
		return openAIImageAccountCooldown{}, false
	}
	key := openAIImageAccountHealthKey(req.ImageEndpoint, req.RequestedModel, accountID)
	if key == "" {
		return openAIImageAccountCooldown{}, false
	}
	value, ok := s.openaiImageHealth.cooldowns.Load(key)
	if !ok {
		return openAIImageAccountCooldown{}, false
	}
	cooldown, ok := value.(openAIImageAccountCooldown)
	if !ok || cooldown.until.IsZero() {
		s.openaiImageHealth.cooldowns.Delete(key)
		return openAIImageAccountCooldown{}, false
	}
	if time.Now().After(cooldown.until) {
		s.openaiImageHealth.cooldowns.Delete(key)
		return openAIImageAccountCooldown{}, false
	}
	return cooldown, true
}

func (s *OpenAIGatewayService) openAIImageLatencyStats(ctx context.Context, endpoint string, model string) map[int64]OpenAIImageAccountLatencyStats {
	if s == nil || s.openaiImageHealth == nil {
		return nil
	}
	key := openAIImageHealthKey(endpoint, model)
	if key == "" {
		return nil
	}
	now := time.Now()
	if value, ok := s.openaiImageHealth.cache.Load(key); ok {
		entry, _ := value.(*openAIImageHealthStatsCacheEntry)
		if entry != nil && now.Before(entry.expiresAt) {
			return entry.stats
		}
	}

	result, _, _ := s.openaiImageHealth.sf.Do(key, func() (any, error) {
		if value, ok := s.openaiImageHealth.cache.Load(key); ok {
			entry, _ := value.(*openAIImageHealthStatsCacheEntry)
			if entry != nil && time.Now().Before(entry.expiresAt) {
				return entry.stats, nil
			}
		}
		repo, ok := s.usageLogRepo.(openAIImageLatencyStatsRepository)
		if !ok || repo == nil {
			return map[int64]OpenAIImageAccountLatencyStats{}, nil
		}
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIImageHealthStatsDBTimeout)
		defer cancel()

		stats, err := repo.GetOpenAIImageAccountLatencyStats(dbCtx, normalizeOpenAIImageEndpoint(endpoint), normalizeOpenAIImageModel(model), time.Now().Add(-openAIImageHealthStatsLookback))
		if err != nil {
			slog.Warn("openai.image_scheduler.latency_stats_failed", "endpoint", endpoint, "model", model, "error", err)
			stats = map[int64]OpenAIImageAccountLatencyStats{}
		}
		s.openaiImageHealth.cache.Store(key, &openAIImageHealthStatsCacheEntry{
			stats:     stats,
			expiresAt: time.Now().Add(openAIImageHealthStatsCacheTTL),
		})
		return stats, nil
	})

	stats, _ := result.(map[int64]OpenAIImageAccountLatencyStats)
	return stats
}

func (s *OpenAIGatewayService) sortOpenAIImageAccountsByHealth(req OpenAIAccountScheduleRequest, accounts []*Account) {
	if s == nil || len(accounts) <= 1 || normalizeOpenAIImageEndpoint(req.ImageEndpoint) == "" {
		return
	}
	if !s.isOpenAIImageLatencySchedulerEnabled(context.Background()) {
		return
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		a := s.openAIImageAccountHealth(req, accounts[i].ID)
		b := s.openAIImageAccountHealth(req, accounts[j].ID)
		if a.slow != b.slow {
			return !a.slow
		}
		if a.recentP95Ms != b.recentP95Ms {
			if a.recentP95Ms <= 0 {
				return false
			}
			if b.recentP95Ms <= 0 {
				return true
			}
			return a.recentP95Ms < b.recentP95Ms
		}
		return false
	})
}

func (s *OpenAIGatewayService) ReportOpenAIImageScheduleResult(accountID int64, endpoint string, model string, durationMs int64, success bool) {
	if s == nil || s.openaiImageHealth == nil || accountID <= 0 || normalizeOpenAIImageEndpoint(endpoint) == "" || durationMs <= 0 {
		return
	}
	if !s.isOpenAIImageLatencySchedulerEnabled(context.Background()) {
		return
	}
	threshold := openAIImageSlowThresholdMs(endpoint)
	if success && durationMs < threshold {
		return
	}
	key := openAIImageAccountHealthKey(endpoint, model, accountID)
	if key == "" {
		return
	}
	reason := "image_slow"
	if !success {
		reason = "image_error"
	}
	until := time.Now().Add(openAIImageSlowCooldown)
	s.openaiImageHealth.cooldowns.Store(key, openAIImageAccountCooldown{until: until, reason: reason})
	s.openaiImageHealth.cache.Delete(openAIImageHealthKey(endpoint, model))
	slog.Warn("openai.image_scheduler.account_cooling",
		"account_id", accountID,
		"endpoint", endpoint,
		"model", normalizeOpenAIImageModel(model),
		"duration_ms", durationMs,
		"threshold_ms", threshold,
		"until", until,
		"reason", reason,
	)
}

func (s *OpenAIGatewayService) ReportOpenAIImageRateLimit(accountID int64, endpoint string, model string) {
	if s == nil || s.openaiImageHealth == nil || accountID <= 0 || normalizeOpenAIImageEndpoint(endpoint) == "" {
		return
	}
	if !s.isOpenAIImageLatencySchedulerEnabled(context.Background()) {
		return
	}
	key := openAIImageAccountHealthKey(endpoint, model, accountID)
	if key == "" {
		return
	}
	until := time.Now().Add(openAIImageRateLimitCooldown)
	s.openaiImageHealth.cooldowns.Store(key, openAIImageAccountCooldown{until: until, reason: "image_rate_limit"})
	s.openaiImageHealth.cache.Delete(openAIImageHealthKey(endpoint, model))
	slog.Warn("openai.image_scheduler.account_cooling",
		"account_id", accountID,
		"endpoint", endpoint,
		"model", normalizeOpenAIImageModel(model),
		"duration_ms", 0,
		"threshold_ms", 0,
		"until", until,
		"reason", "image_rate_limit",
	)
}

func (s *OpenAIGatewayService) openAIImageEffectiveLoadConcurrency(req OpenAIAccountScheduleRequest, account *Account) int {
	if account == nil {
		return openAIImageMaxConcurrency
	}
	if normalizeOpenAIImageEndpoint(req.ImageEndpoint) == "" || !s.isOpenAIImageLatencySchedulerEnabled(context.Background()) {
		return account.EffectiveLoadFactor()
	}
	if openAIImageMaxConcurrency > 0 && openAIImageMaxConcurrency < account.EffectiveLoadFactor() {
		return openAIImageMaxConcurrency
	}
	return account.EffectiveLoadFactor()
}

func (s *OpenAIGatewayService) openAIImageEffectiveSlotConcurrency(req OpenAIAccountScheduleRequest, account *Account) int {
	if account == nil {
		return openAIImageMaxConcurrency
	}
	if normalizeOpenAIImageEndpoint(req.ImageEndpoint) == "" || !s.isOpenAIImageLatencySchedulerEnabled(context.Background()) {
		return account.Concurrency
	}
	if openAIImageMaxConcurrency > 0 && openAIImageMaxConcurrency < account.Concurrency {
		return openAIImageMaxConcurrency
	}
	return account.Concurrency
}
