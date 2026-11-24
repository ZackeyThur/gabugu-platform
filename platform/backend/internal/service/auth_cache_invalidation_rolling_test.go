//go:build unit

package service

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/dgraph-io/ristretto"
	"github.com/stretchr/testify/require"
)

type rollingAuthInvalidationRepo struct {
	*authInvalidationRepoStub
	claims atomic.Int64
}

func (r *rollingAuthInvalidationRepo) Claim(context.Context, string, int, time.Duration) ([]AuthCacheInvalidationEvent, error) {
	r.claims.Add(1)
	return nil, nil
}

func TestRollingAuthInvalidationDefersOutboxButStartsLocalSubscriber(t *testing.T) {
	configureRollingGateTest(t)
	repo := &rollingAuthInvalidationRepo{authInvalidationRepoStub: &authInvalidationRepoStub{}}
	connected := make(chan struct{})
	cache := &authInvalidationCacheStub{subscribeFn: func(ctx context.Context, _ func(string)) error {
		NotifyAuthCacheSubscriptionReady(ctx)
		close(connected)
		<-ctx.Done()
		return ctx.Err()
	}}
	svc := NewAPIKeyService(nil, nil, nil, nil, nil, cache, nil)
	localCache, err := ristretto.NewCache(&ristretto.Config{NumCounters: 10, MaxCost: 1, BufferItems: 64})
	require.NoError(t, err)
	defer localCache.Close()
	svc.authNegativeCacheL1 = localCache
	svc.StartAuthCacheInvalidationSubscriber(context.Background())
	defer svc.StopAuthCacheInvalidationSubscriber()
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("standby local cache subscriber did not start")
	}
	worker := ProvideAuthCacheInvalidationWorker(repo, cache, svc)
	worker.Start()
	backgroundgate.Seal()
	state := backgroundgate.Snapshot()
	require.Equal(t, []string{authInvalidationWorkerTask}, state.Tasks)
	require.Equal(t, 1, state.Registered)
	require.Zero(t, state.Started)
	require.False(t, worker.Health(context.Background()).Running)
	require.Never(t, func() bool { return repo.claims.Load() != 0 }, 600*time.Millisecond, 10*time.Millisecond)
	worker.Stop()
	worker.Stop()
	// Stop before activation forbids a late marker from starting the consumer.
	require.NoError(t, os.WriteFile(os.Getenv(backgroundgate.ActivationFileEnv), []byte(`{"release_id":"test-candidate","activate":true}`), 0600))
	require.Eventually(t, func() bool { return backgroundgate.Snapshot().Active }, 3*time.Second, 10*time.Millisecond)
	require.Zero(t, repo.claims.Load())
	require.False(t, worker.Health(context.Background()).Running)
	require.Empty(t, repo.deleted)
	require.Empty(t, repo.scheduled)
	require.Empty(t, repo.retried)
	require.Empty(t, cache.deleted)
	require.Empty(t, cache.published)
}

func TestRollingAuthInvalidationActivationAndStopStayIdempotent(t *testing.T) {
	configureRollingGateTest(t)
	repo := &rollingAuthInvalidationRepo{authInvalidationRepoStub: &authInvalidationRepoStub{}}
	worker := ProvideAuthCacheInvalidationWorker(repo, &authInvalidationCacheStub{}, nil)
	worker.Start()
	backgroundgate.Seal()
	require.Zero(t, repo.claims.Load())
	require.NoError(t, os.WriteFile(os.Getenv(backgroundgate.ActivationFileEnv), []byte(`{"release_id":"test-candidate","activate":true}`), 0600))
	require.Eventually(t, func() bool { return repo.claims.Load() > 0 }, 3*time.Second, 10*time.Millisecond)
	state := backgroundgate.Snapshot()
	require.True(t, state.Ready)
	require.Equal(t, 1, state.Registered)
	require.Equal(t, 1, state.Started)
	worker.Stop()
	worker.Stop()
	claims := repo.claims.Load()
	worker.Start()
	require.Never(t, func() bool { return repo.claims.Load() != claims }, 600*time.Millisecond, 10*time.Millisecond)
	require.False(t, worker.Health(context.Background()).Running)
}
