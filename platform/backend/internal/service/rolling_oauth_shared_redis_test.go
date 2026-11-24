//go:build unit

package service_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// This opt-in test exercises the real repository Redis lock protocol. The
// credential repository and provider executor are synthetic; no provider runs.
// It does not claim that a native peer's Redis-error fallback is safe.
func TestRollingOAuthSharedRedisProtocol(t *testing.T) {
	addr := strings.TrimSpace(os.Getenv("GABUGU_TEST_SHARED_REDIS_ADDR"))
	if addr == "" {
		t.Skip("opt-in real Redis test: set GABUGU_TEST_SHARED_REDIS_ADDR and GABUGU_TEST_SHARED_REDIS_DB=9")
	}
	host, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	portNumber, err := strconv.Atoi(port)
	require.NoError(t, err)
	require.True(t, portNumber > 0 && portNumber <= 65535, "invalid test Redis port")
	require.True(t, (host == "rolling-redis" && port == "6379") || (host == "127.0.0.1" && port != "6379"),
		"only rolling-redis:6379 or an explicitly owned non-default loopback port is allowed")
	require.Equal(t, "9", os.Getenv("GABUGU_TEST_SHARED_REDIS_DB"), "only isolated Redis DB 9 is allowed")
	require.Empty(t, os.Getenv(backgroundgate.ActivationFileEnv), "refuse to replace an existing rolling configuration")
	require.Empty(t, os.Getenv(backgroundgate.ReleaseIDEnv), "refuse to replace an existing release configuration")
	t.Setenv(backgroundgate.ActivationFileEnv, filepath.Join(t.TempDir(), "activate.json"))
	t.Setenv(backgroundgate.ReleaseIDEnv, "oauth-shared-redis-protocol-test")
	require.NoError(t, backgroundgate.InitializeFromEnv())
	t.Cleanup(func() {
		backgroundgate.Close()
		_ = os.Unsetenv(backgroundgate.ActivationFileEnv)
		_ = os.Unsetenv(backgroundgate.ReleaseIDEnv)
		if err := backgroundgate.InitializeFromEnv(); err != nil {
			t.Errorf("restore default gate: %v", err)
		}
	})

	t.Run("healthy-shared-lease-excludes-second-instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		firstClient := redis.NewClient(sharedOAuthRedisOptions(addr))
		defer firstClient.Close()
		secondClient := redis.NewClient(sharedOAuthRedisOptions(addr))
		defer secondClient.Close()
		require.NoError(t, firstClient.Ping(ctx).Err())
		info, err := firstClient.Info(ctx, "server").Result()
		require.NoError(t, err)
		for _, line := range strings.Split(info, "\n") {
			if strings.HasPrefix(line, "redis_version:") {
				t.Logf("real Redis %s; isolated DB 9", strings.TrimSpace(line))
			}
		}
		key := sharedOAuthRedisKey(t)
		lockKey := "oauth:refresh_lock:" + key // repository's actual lock key
		defer func() {
			cleanupCtx, done := context.WithTimeout(context.Background(), time.Second)
			defer done()
			// Never FLUSHDB or remove another participant's keys.
			if err := firstClient.Del(cleanupCtx, lockKey, "oauth:token:"+key).Err(); err != nil {
				t.Errorf("clean owned Redis keys: %v", err)
			}
		}()
		repo := newSharedOAuthAccountRepo()
		entered, release := make(chan struct{}), make(chan struct{})
		firstExecutor := &sharedOAuthExecutor{key: key, entered: entered, release: release}
		secondExecutor := &sharedOAuthExecutor{key: key}
		// Each API owns an independent localLocks map and Redis client.
		firstAPI := service.NewOAuthRefreshAPI(repo, repository.NewGeminiTokenCache(firstClient))
		secondAPI := service.NewOAuthRefreshAPI(repo, repository.NewGeminiTokenCache(secondClient))
		type outcome struct {
			result *service.OAuthRefreshResult
			err    error
		}
		firstResult, firstDone := make(chan outcome, 1), make(chan struct{})
		go func() {
			result, err := firstAPI.RefreshIfNeeded(ctx, repo.snapshot(), firstExecutor, time.Minute)
			firstResult <- outcome{result, err}
			close(firstDone)
		}()
		var releaseOnce sync.Once
		defer func() {
			releaseOnce.Do(func() { close(release) })
			cancel()
			select {
			case <-firstDone:
			case <-time.After(time.Second):
				t.Error("first refresh did not terminate")
			}
		}()
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("first executor did not acquire the real Redis lease")
		}
		ttl, err := secondClient.PTTL(ctx, lockKey).Result()
		require.NoError(t, err)
		require.Greater(t, ttl, time.Duration(0))
		require.LessOrEqual(t, ttl, time.Minute)
		secondResult, err := secondAPI.RefreshIfNeeded(ctx, repo.snapshot(), secondExecutor, time.Minute)
		require.NoError(t, err)
		require.NotNil(t, secondResult)
		require.True(t, secondResult.LockHeld)
		require.False(t, secondResult.Refreshed)
		require.Equal(t, int64(1), repo.reads.Load(), "lock-held contender must not reread or mutate the account")
		require.Zero(t, repo.writes.Load())
		require.Zero(t, secondExecutor.calls.Load())
		releaseOnce.Do(func() { close(release) })
		first := <-firstResult
		require.NoError(t, first.err)
		require.NotNil(t, first.result)
		require.True(t, first.result.Refreshed)
		require.Equal(t, int64(1), firstExecutor.calls.Load())
		require.Equal(t, int64(1), repo.writes.Load())
		exists, err := secondClient.Exists(ctx, lockKey).Result()
		require.NoError(t, err)
		require.Zero(t, exists, "completed refresh must release its Redis lease")
		// Once the lock is free, the second instance rereads the fresh account.
		after, err := secondAPI.RefreshIfNeeded(ctx, repo.snapshot(), secondExecutor, time.Minute)
		require.NoError(t, err)
		require.False(t, after.Refreshed)
		require.False(t, after.LockHeld)
		require.Zero(t, secondExecutor.calls.Load())
		require.Equal(t, int64(1), repo.writes.Load())
		t.Logf("healthy shared lease: refreshes=1 account_mutations=1 contender_lock_held=true ttl_observed=%s", ttl)
	})

	t.Run("real-connection-failure-fails-closed", func(t *testing.T) {
		// Allocate our own loopback port, then close it to produce a real TCP
		// connection refusal without stopping or changing the shared Redis.
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		closedAddr := listener.Addr().String()
		require.NoError(t, listener.Close())
		client := redis.NewClient(sharedOAuthRedisOptions(closedAddr))
		defer client.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		require.Error(t, client.Ping(ctx).Err(), "must exercise a real failed connection")
		require.NoError(t, ctx.Err(), "the API must receive a live context")
		repo := newSharedOAuthAccountRepo()
		executor := &sharedOAuthExecutor{key: sharedOAuthRedisKey(t)}
		api := service.NewOAuthRefreshAPI(repo, repository.NewGeminiTokenCache(client))
		result, err := api.RefreshIfNeeded(ctx, repo.snapshot(), executor, time.Minute)
		require.Error(t, err)
		require.Contains(t, err.Error(), "OAuth refresh account state is unavailable")
		require.Nil(t, result)
		require.Zero(t, repo.reads.Load())
		require.Zero(t, repo.writes.Load())
		require.Zero(t, executor.calls.Load())
		t.Log("real Redis connection failure: refreshes=0 account_mutations=0; native-peer failure behavior is outside this test")
	})
}

func sharedOAuthRedisOptions(addr string) *redis.Options {
	return &redis.Options{Addr: addr, DB: 9, MaxRetries: -1, DialTimeout: 250 * time.Millisecond,
		ReadTimeout: 250 * time.Millisecond, WriteTimeout: 250 * time.Millisecond, PoolTimeout: time.Second,
		ContextTimeoutEnabled: true}
}

func sharedOAuthRedisKey(t *testing.T) string {
	t.Helper()
	var nonce [16]byte
	_, err := rand.Read(nonce[:])
	require.NoError(t, err)
	return "rolling-oauth-shared-redis-test:" + hex.EncodeToString(nonce[:])
}

type sharedOAuthAccountRepo struct {
	service.AccountRepository
	mu      sync.Mutex
	account service.Account
	reads   atomic.Int64
	writes  atomic.Int64
}

func newSharedOAuthAccountRepo() *sharedOAuthAccountRepo {
	return &sharedOAuthAccountRepo{account: service.Account{ID: 9911, Platform: service.PlatformAnthropic,
		Type: service.AccountTypeOAuth, Status: service.StatusActive,
		Credentials: map[string]any{"access_token": "synthetic-expired"}}}
}

func (r *sharedOAuthAccountRepo) snapshot() *service.Account {
	r.mu.Lock()
	defer r.mu.Unlock()
	account := r.account
	account.Credentials = maps.Clone(account.Credentials)
	return &account
}

func (r *sharedOAuthAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	r.reads.Add(1)
	account := r.snapshot()
	if id != account.ID {
		return nil, fmt.Errorf("unknown synthetic account")
	}
	return account, nil
}

func (r *sharedOAuthAccountRepo) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if id != r.account.ID {
		return fmt.Errorf("unknown synthetic account")
	}
	r.account.Credentials = maps.Clone(credentials)
	r.writes.Add(1)
	return nil
}

type sharedOAuthExecutor struct {
	key     string
	calls   atomic.Int64
	entered chan struct{}
	release <-chan struct{}
}

func (e *sharedOAuthExecutor) CacheKey(*service.Account) string { return e.key }
func (e *sharedOAuthExecutor) CanRefresh(*service.Account) bool { return true }
func (e *sharedOAuthExecutor) NeedsRefresh(account *service.Account, _ time.Duration) bool {
	return account.Credentials["access_token"] == "synthetic-expired"
}
func (e *sharedOAuthExecutor) Refresh(ctx context.Context, _ *service.Account) (map[string]any, error) {
	e.calls.Add(1)
	if e.entered != nil {
		close(e.entered)
	}
	if e.release != nil {
		select {
		case <-e.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return map[string]any{"access_token": "synthetic-fresh"}, nil
}
