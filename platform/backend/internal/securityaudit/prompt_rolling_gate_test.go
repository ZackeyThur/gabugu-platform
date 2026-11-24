package securityaudit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/backgroundgate"
	"github.com/stretchr/testify/require"
)

type promptRollingRepository struct {
	*fakeJobRepository
	claims   atomic.Int64
	reclaims atomic.Int64
}

func (r *promptRollingRepository) ClaimNextJob(context.Context, time.Time) (*Job, bool, error) {
	r.claims.Add(1)
	return nil, false, nil
}

func (r *promptRollingRepository) ReclaimStale(context.Context, time.Time, time.Time, int) (int64, error) {
	r.reclaims.Add(1)
	return 0, nil
}

type promptRollingPayload struct {
	*fakePayloadStore
	pings atomic.Int64
}

func (s *promptRollingPayload) Ping(context.Context) error {
	s.pings.Add(1)
	return nil
}

type promptRollingSettings struct {
	staticSettingRepository
	reads  atomic.Int64
	writes atomic.Int64
}

func (s *promptRollingSettings) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	s.reads.Add(1)
	return s.staticSettingRepository.GetMultiple(ctx, keys)
}

func (s *promptRollingSettings) Set(context.Context, string, string) error {
	s.writes.Add(1)
	return nil
}

func (s *promptRollingSettings) SetMultiple(context.Context, map[string]string) error {
	s.writes.Add(1)
	return nil
}

func (s *promptRollingSettings) Delete(context.Context, string) error {
	s.writes.Add(1)
	return nil
}

func promptRollingGate(t *testing.T) (string, func()) {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "activate.json")
	t.Setenv(backgroundgate.ActivationFileEnv, marker)
	t.Setenv(backgroundgate.ReleaseIDEnv, "prompt-audit-test")
	require.NoError(t, backgroundgate.InitializeFromEnv())
	return marker, func() {
		backgroundgate.Close()
		t.Setenv(backgroundgate.ActivationFileEnv, "")
		t.Setenv(backgroundgate.ReleaseIDEnv, "")
		require.NoError(t, backgroundgate.InitializeFromEnv())
	}
}

func promptRollingRunner(config ConfigStore) (*Runner, *promptRollingRepository, *promptRollingPayload) {
	repo := &promptRollingRepository{fakeJobRepository: &fakeJobRepository{}}
	payload := &promptRollingPayload{fakePayloadStore: &fakePayloadStore{}}
	scanner := PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
		return &NormalizedResult{Decision: EventPass}, nil
	})
	return NewRunner(config, repo, payload, scanner, NewAtomicMetrics()), repo, payload
}

func TestPromptAuditRollingStandbyKeepsBlockingPolicyWithoutSharedWrites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		marker, restore := promptRollingGate(t)
		defer restore()
		// Valid explicit blocking intent with unusable worker config must remain
		// fail-closed even though shared processing is deferred.
		settings := &promptRollingSettings{staticSettingRepository: staticSettingRepository{values: map[string]string{
			SettingKeyRiskControl:       "true",
			SettingKeyPromptAuditConfig: `{"enabled":true,"blocking_enabled":true,"config_version":3,"worker_count":"invalid"}`,
		}}}
		config := NewConfigManager(nil, settings, nil, prefixEncryptor{}, testTotpKeyConfig())
		runner, repo, payload := promptRollingRunner(config)
		svc := &PromptService{config: config, runner: runner, evaluator: NewGuardEvaluator(nil, nil, nil)}
		require.Error(t, svc.Start(context.Background()))
		backgroundgate.Seal()
		require.Positive(t, settings.reads.Load(), "local policy initialization must not be deferred")
		require.Equal(t, ModeBlocking, svc.EffectiveMode())
		decision, err := svc.Evaluate(context.Background(), Request{Protocol: "openai_chat_completions", Body: []byte(`{"messages":[{"role":"user","content":"fixture"}]}`)})
		require.Error(t, err)
		require.Nil(t, decision)
		var guardErr *GuardError
		require.ErrorAs(t, err, &guardErr)
		require.Equal(t, ErrorCodeUnavailable, guardErr.Code)
		state := backgroundgate.Snapshot()
		require.Equal(t, 1, state.Registered)
		require.Zero(t, state.Started)
		require.Equal(t, []string{promptAuditRunnerTask}, state.Tasks)
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		require.Zero(t, repo.claims.Load())
		require.Zero(t, repo.reclaims.Load(), "standby must not reclaim shared jobs even with audit disabled/degraded")
		require.Zero(t, settings.writes.Load())
		require.Zero(t, payload.pings.Load(), "deferred runner must not touch its payload backend")
		require.Nil(t, runner.cancel)
		require.NoError(t, svc.Shutdown(context.Background()))
		// Stopping an unstarted task must also prevent a later marker from starting it.
		require.NoError(t, os.WriteFile(marker, []byte(`{"release_id":"prompt-audit-test","activate":true}`), 0600))
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		require.Zero(t, repo.claims.Load())
		require.Zero(t, repo.reclaims.Load())
		require.Zero(t, payload.pings.Load())
	})
}

func TestPromptAuditRunnerNormalModeStillClaimsAndReclaims(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Setenv(backgroundgate.ActivationFileEnv, "")
		t.Setenv(backgroundgate.ReleaseIDEnv, "")
		require.NoError(t, backgroundgate.InitializeFromEnv())
		defer backgroundgate.Close()
		runner, repo, payload := promptRollingRunner(&fakeConfigStore{active: true, cfg: asyncConfig()})
		require.NoError(t, runner.Start(context.Background()))
		require.NoError(t, runner.Start(context.Background()))
		time.Sleep(61 * time.Second)
		synctest.Wait()
		require.EqualValues(t, 1, payload.pings.Load(), "normal Start remains idempotent")
		require.Positive(t, repo.claims.Load())
		require.Positive(t, repo.reclaims.Load())
		require.NoError(t, runner.Shutdown(context.Background()))
		require.NoError(t, runner.Shutdown(context.Background()))
		claims, reclaims := repo.claims.Load(), repo.reclaims.Load()
		time.Sleep(61 * time.Second)
		synctest.Wait()
		require.Equal(t, claims, repo.claims.Load())
		require.Equal(t, reclaims, repo.reclaims.Load())
		require.Nil(t, runner.cancel)
	})
}

func TestPromptAuditRunnerActivationStartsSharedWorkersOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		marker, restore := promptRollingGate(t)
		defer restore()
		runner, repo, payload := promptRollingRunner(&fakeConfigStore{active: true, cfg: asyncConfig()})
		require.NoError(t, runner.Start(context.Background()))
		require.NoError(t, runner.Start(context.Background()))
		backgroundgate.Seal()
		require.Zero(t, payload.pings.Load())
		require.NoError(t, os.WriteFile(marker, []byte(`{"release_id":"prompt-audit-test","activate":true}`), 0600))
		time.Sleep(61 * time.Second)
		synctest.Wait()
		state := backgroundgate.Snapshot()
		require.True(t, state.Ready, strings.Join(state.Tasks, ","))
		require.Equal(t, 1, state.Started)
		require.EqualValues(t, 1, payload.pings.Load())
		require.Positive(t, repo.claims.Load())
		require.Positive(t, repo.reclaims.Load())
		require.NoError(t, runner.Shutdown(context.Background()))
	})
}
