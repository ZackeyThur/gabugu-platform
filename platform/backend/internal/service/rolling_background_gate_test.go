//go:build unit

package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/stretchr/testify/require"
)

func configureRollingGateTest(t *testing.T) {
	t.Helper()
	t.Setenv(backgroundgate.ActivationFileEnv, filepath.Join(t.TempDir(), "activate.json"))
	t.Setenv(backgroundgate.ReleaseIDEnv, "test-candidate")
	require.NoError(t, backgroundgate.InitializeFromEnv())
	t.Cleanup(func() {
		backgroundgate.Close()
		// Restore the default manager before other tests construct services.
		_ = os.Unsetenv(backgroundgate.ActivationFileEnv)
		_ = os.Unsetenv(backgroundgate.ReleaseIDEnv)
		require.NoError(t, backgroundgate.InitializeFromEnv())
	})
}

func TestRollingOAuthFailsClosedBeforeProviderOrCredentialMutation(t *testing.T) {
	configureRollingGateTest(t)
	for _, tc := range []struct {
		name  string
		cache GeminiTokenCache
	}{
		{"redis-unavailable", &refreshAPICacheStub{lockErr: errors.New("redis unavailable")}},
		{"cache-missing", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive}
			repo := &refreshAPIAccountRepo{account: account}
			executor := &refreshAPIExecutorStub{needsRefresh: true, credentials: map[string]any{"access_token": "test-token"}}
			result, err := NewOAuthRefreshAPI(repo, tc.cache).RefreshIfNeeded(context.Background(), account, executor, time.Hour)
			require.Error(t, err)
			require.Nil(t, result)
			require.Zero(t, executor.refreshCalls)
			require.Zero(t, repo.getByIDCalls)
			require.Zero(t, repo.updateCalls)
		})
	}
}

func TestRollingOAuthRetainsHealthyDistributedRefresh(t *testing.T) {
	configureRollingGateTest(t)
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive}
	repo := &refreshAPIAccountRepo{account: account}
	cache := &refreshAPICacheStub{lockResult: true}
	executor := &refreshAPIExecutorStub{needsRefresh: true, credentials: map[string]any{"access_token": "test-token"}}
	result, err := NewOAuthRefreshAPI(repo, cache).RefreshIfNeeded(context.Background(), account, executor, time.Hour)
	require.NoError(t, err)
	require.True(t, result.Refreshed)
	require.Equal(t, 1, executor.refreshCalls)
	require.Equal(t, 1, cache.releaseCalls)
}

type rollingConcurrencyCache struct {
	ConcurrencyCache
	cleanupCalled bool
}

func (c *rollingConcurrencyCache) CleanupStaleProcessSlots(context.Context, string) error {
	c.cleanupCalled = true
	return nil
}

func TestRollingProviderDoesNotDeletePeerConcurrencySlots(t *testing.T) {
	configureRollingGateTest(t)
	cache := &rollingConcurrencyCache{}
	require.NotNil(t, ProvideConcurrencyService(cache, nil, nil))
	require.False(t, cache.cleanupCalled, "legacy startup sweep deletes live peer slots")
}

type rollingOllamaRepo struct {
	*ollamaUsageTestRepo
	dueCalls atomic.Int64
}

func (r *rollingOllamaRepo) ListDueOllamaCloudUsageAccounts(context.Context, time.Time, time.Duration, time.Duration, int) ([]Account, error) {
	r.dueCalls.Add(1)
	return nil, nil
}

func TestRollingOllamaStandbyServesRequestProbeBeforePeriodicActivation(t *testing.T) {
	configureRollingGateTest(t)
	fixedNow := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	account := ollamaUsageAccount(1)
	account.Extra[OllamaCloudUsageSessionExtraKey] = "cipher:wos-session=fixture"
	base := &ollamaUsageTestRepo{upstreamBillingProbeAccountRepo: &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{account.ID: account}}}
	repo := &rollingOllamaRepo{ollamaUsageTestRepo: base}
	settings := NewSettingService(&upstreamBillingProbeSettingRepo{values: map[string]string{SettingKeyOllamaCloudUsageSettings: `{"enabled":true,"interval_minutes":60}`}}, nil)
	resetAt := fixedNow.Add(2 * time.Hour)
	upstream := &ollamaUsageHTTPStub{body: ollamaCloudProbeUsageBody(100, resetAt.Format(time.RFC3339))}
	svc := NewOllamaCloudUsageService(repo, upstream, settings, ollamaUsageTestEncryptor{}, true)
	svc.now = func() time.Time { return fixedNow }
	svc.Start()
	t.Cleanup(svc.Stop)
	require.Zero(t, backgroundgate.Snapshot().Started)
	require.Zero(t, repo.dueCalls.Load())
	completed := make(chan time.Time, 1)
	require.True(t, svc.ScheduleOllamaCloudUsageRateLimitProbe(account.ID, func(_ int64, reset time.Time) { completed <- reset }))
	select {
	case got := <-completed:
		require.True(t, got.Equal(resetAt))
	case <-time.After(3 * time.Second):
		t.Fatal("standby request probe did not run")
	}
	require.Equal(t, int64(1), upstream.calls.Load())
	require.Zero(t, repo.dueCalls.Load(), "periodic RunDue must remain deferred")
	backgroundgate.Seal()
	marker := os.Getenv(backgroundgate.ActivationFileEnv)
	require.NoError(t, os.WriteFile(marker+".tmp", []byte(`{"release_id":"test-candidate","activate":true}`), 0600))
	require.NoError(t, os.Rename(marker+".tmp", marker))
	require.Eventually(t, func() bool { return repo.dueCalls.Load() == 1 }, 3*time.Second, 10*time.Millisecond)
	require.Equal(t, 1, backgroundgate.Snapshot().Started)
}
