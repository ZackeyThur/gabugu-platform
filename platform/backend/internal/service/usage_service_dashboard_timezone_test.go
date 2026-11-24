package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

type legacyDashboardRepoStub struct {
	UsageLogRepository
	userID int64
}

func (r *legacyDashboardRepoStub) GetUserDashboardStats(_ context.Context, userID int64) (*usagestats.UserDashboardStats, error) {
	r.userID = userID
	return &usagestats.UserDashboardStats{TodayActualCost: 0.125}, nil
}

func TestGetUserDashboardStatsLegacyRepositoryCompatibility(t *testing.T) {
	for _, tz := range [][]string{nil, {""}, {"Asia/Shanghai"}} {
		repo := &legacyDashboardRepoStub{}
		svc := NewUsageService(repo, nil, nil, nil)
		stats, err := svc.GetUserDashboardStats(context.Background(), 42, tz...)
		require.NoError(t, err)
		require.Equal(t, int64(42), repo.userID)
		require.Equal(t, 0.125, stats.TodayActualCost)
	}
}
