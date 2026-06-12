package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type openAIImageLatencyStatsRepoStub struct {
	UsageLogRepository
	stats map[int64]OpenAIImageAccountLatencyStats
}

func (s *openAIImageLatencyStatsRepoStub) GetOpenAIImageAccountLatencyStats(ctx context.Context, endpoint string, model string, since time.Time) (map[int64]OpenAIImageAccountLatencyStats, error) {
	return s.stats, nil
}

func TestOpenAIImageAccountHealthRanksRecentSlowAccount(t *testing.T) {
	resetOpenAIImageLatencySchedulerSettingCacheForTest()
	svc := &OpenAIGatewayService{
		usageLogRepo: &openAIImageLatencyStatsRepoStub{stats: map[int64]OpenAIImageAccountLatencyStats{
			1: {AccountID: 1, RequestCount: 5, P95Ms: 90000},
			2: {AccountID: 2, RequestCount: 5, P95Ms: 260000, VerySlowCount: 1},
		}},
		openaiImageHealth: newOpenAIImageHealthState(),
	}
	openAIImageLatencySchedulerSettingCache.Store(&cachedOpenAIImageLatencySchedulerSetting{
		enabled:   true,
		expiresAt: time.Now().Add(time.Hour).UnixNano(),
	})

	req := OpenAIAccountScheduleRequest{
		ImageEndpoint:  "/v1/images/edits",
		RequestedModel: "gpt-image-2",
	}

	fast := svc.openAIImageAccountHealth(req, 1)
	slow := svc.openAIImageAccountHealth(req, 2)

	require.True(t, fast.applicable)
	require.False(t, fast.slow)
	require.True(t, slow.slow)
	require.Less(t, slow.scoreAdjustment, fast.scoreAdjustment)
}

func TestReportOpenAIImageScheduleResultCoolsSlowAccount(t *testing.T) {
	resetOpenAIImageLatencySchedulerSettingCacheForTest()
	svc := &OpenAIGatewayService{openaiImageHealth: newOpenAIImageHealthState()}
	openAIImageLatencySchedulerSettingCache.Store(&cachedOpenAIImageLatencySchedulerSetting{
		enabled:   true,
		expiresAt: time.Now().Add(time.Hour).UnixNano(),
	})

	req := OpenAIAccountScheduleRequest{
		ImageEndpoint:  "/v1/images/edits",
		RequestedModel: "gpt-image-2",
	}
	svc.ReportOpenAIImageScheduleResult(21, req.ImageEndpoint, req.RequestedModel, 244000, true)

	require.True(t, svc.isOpenAIImageAccountCooling(req, 21))
	require.False(t, svc.isOpenAIImageAccountCooling(req, 2))
}
