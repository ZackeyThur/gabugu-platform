//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type tokenCampaignSettingsStub struct {
	SettingRepository
	raw   string
	err   error
	reads int
}

func (r *tokenCampaignSettingsStub) GetValue(context.Context, string) (string, error) {
	r.reads++
	return r.raw, r.err
}
func campaignTestConfig(at time.Time) TokenBillingCampaign {
	return TokenBillingCampaign{SchemaVersion: 1, ID: "blg-20261115", Revision: 1, Enabled: true, StartsAt: at, Phases: []TokenBillingCampaignPhase{
		{ID: "championship-day", StartOffsetSeconds: 0, EndOffsetSeconds: 86400, GroupMultipliers: map[string]float64{"4": .2, "5": .5, "23": .5, "24": .5}},
		{ID: "celebration-week", StartOffsetSeconds: 86400, EndOffsetSeconds: 691200, GroupMultipliers: map[string]float64{"4": .9, "5": .9, "23": .9, "24": .9}},
	}}
}
func campaignTestSettings(t *testing.T, at time.Time) (*SettingService, *tokenCampaignSettingsStub) {
	t.Helper()
	raw, err := json.Marshal(campaignTestConfig(at))
	require.NoError(t, err)
	repo := &tokenCampaignSettingsStub{raw: string(raw)}
	return NewSettingService(repo, nil), repo
}
func TestTokenCampaignBoundariesAndGroupScope(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	settings, _ := campaignTestSettings(t, start)
	for _, tc := range []struct {
		offset time.Duration
		group  int64
		factor float64
	}{
		{-time.Nanosecond, 4, 1}, {0, 4, .2}, {24*time.Hour - time.Nanosecond, 4, .2}, {24 * time.Hour, 4, .9}, {192*time.Hour - time.Nanosecond, 4, .9}, {192 * time.Hour, 4, 1},
		{0, 5, .5}, {0, 23, .5}, {0, 24, .5}, {0, 6, 1}, {24 * time.Hour, 6, 1},
	} {
		ctx := settings.freezeTokenCampaign(context.Background(), &tc.group, start.Add(tc.offset))
		cost := &CostBreakdown{TotalCost: 10, ActualCost: 3.7, tokenCampaignEligibleActualCost: 3.7}
		snapshot, err := applyTokenCampaign(ctx, &APIKey{GroupID: &tc.group, Group: &Group{ID: tc.group}}, false, cost)
		require.NoError(t, err)
		require.InDelta(t, 3.7*tc.factor, cost.ActualCost, 1e-12)
		require.Equal(t, 10.0, cost.TotalCost)
		if tc.factor < 1 {
			require.NotNil(t, snapshot)
			require.Equal(t, tc.factor, snapshot.Discount)
		} else {
			require.Nil(t, snapshot)
		}
	}
}
func TestTokenCampaignFrozenConfigurationAndNonStacking(t *testing.T) {
	at := time.Now().Add(-time.Hour)
	settings, repo := campaignTestSettings(t, at)
	group := int64(4)
	ctx := settings.freezeTokenCampaign(context.Background(), &group, at.Add(time.Hour))
	repo.raw = `{"broken":true}`
	detached := CopyTokenCampaignPricingContext(ctx, context.Background())
	for i := 0; i < 2; i++ {
		cost := &CostBreakdown{TotalCost: 10, ActualCost: 2.1, tokenCampaignEligibleActualCost: 1.5}
		snapshot, err := applyTokenCampaign(detached, &APIKey{GroupID: &group, Group: &Group{ID: group}}, false, cost)
		require.NoError(t, err)
		require.InDelta(t, .9, cost.ActualCost, 1e-12)
		require.Equal(t, 2.1, snapshot.OriginalActualCost)
		require.InDelta(t, 1.2, snapshot.DiscountAmount, 1e-12)
	}
	require.Equal(t, 1, repo.reads, "settlement must never reload configuration")
}
func TestTokenCampaignExclusionsAndInvalidFailClosed(t *testing.T) {
	settings, repo := campaignTestSettings(t, time.Now().Add(-time.Hour))
	group := int64(4)
	ctx := settings.freezeTokenCampaign(context.Background(), &group, time.Now())
	for _, tc := range []struct {
		sub      bool
		eligible float64
	}{{true, 5}, {false, 0}} {
		cost := &CostBreakdown{TotalCost: 10, ActualCost: 5, tokenCampaignEligibleActualCost: tc.eligible}
		snap, err := applyTokenCampaign(ctx, &APIKey{GroupID: &group, Group: &Group{ID: group}}, tc.sub, cost)
		require.NoError(t, err)
		require.Nil(t, snap)
		require.Equal(t, 5.0, cost.ActualCost)
	}
	repo.raw = `{"schema_version":2,"enabled":true}`
	settings.InvalidateTokenBillingCampaignCache()
	invalid := settings.freezeTokenCampaign(context.Background(), &group, time.Now())
	require.Error(t, tokenCampaignPricingError(invalid))
	other := int64(6)
	unrelated := settings.freezeTokenCampaign(context.Background(), &other, time.Now())
	require.NoError(t, tokenCampaignPricingError(unrelated))
	require.Equal(t, "invalid", settings.GetPublicBillingCampaign(context.Background(), time.Now()).Status)
}
func TestTokenCampaignRejectsUnexpectedConfiguration(t *testing.T) {
	start := time.Now()
	config := campaignTestConfig(start)
	for _, mutate := range []func(*TokenBillingCampaign){
		func(c *TokenBillingCampaign) { c.Phases[0].GroupMultipliers["6"] = .1 },
		func(c *TokenBillingCampaign) { c.Phases[1].StartOffsetSeconds = 86399 },
		func(c *TokenBillingCampaign) { c.Phases[0].GroupMultipliers["4"] = 0 },
		func(c *TokenBillingCampaign) { c.Revision = 0 },
	} {
		c := config
		c.Phases = append([]TokenBillingCampaignPhase(nil), config.Phases...)
		c.Phases[0].GroupMultipliers = map[string]float64{"4": .2, "5": .5, "23": .5, "24": .5}
		mutate(&c)
		raw, err := json.Marshal(c)
		require.NoError(t, err)
		_, err = parseTokenBillingCampaign(string(raw))
		require.Error(t, err)
	}
}

func TestTokenCampaignRecordUsageBothGateways(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	for _, openAI := range []bool{false, true} {
		for _, gid := range []int64{4, 5, 23, 24, 6} {
			t.Run(fmt.Sprintf("openai=%v/group=%d", openAI, gid), func(t *testing.T) {
				settings, repo := campaignTestSettings(t, start)
				usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
				userRepo := &openAIRecordUsageUserRepoStub{}
				rate := .31
				key := &APIKey{ID: 101, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: .37}}
				ctx := settings.freezeTokenCampaign(context.Background(), &gid, start.Add(time.Hour))
				// A settlement after both phase and configuration change must retain
				// the accepted request's price and its user's exclusive base rate.
				repo.raw = `{"invalid":true}`
				var baseline *UsageLog
				if openAI {
					svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{}, &openAIUserGroupRateRepoStub{rate: &rate})
					input := &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "campaign-openai", Model: "gpt-5.1", Usage: OpenAIUsage{InputTokens: 1000, OutputTokens: 300, CacheReadInputTokens: 200, CacheCreationInputTokens: 100}, SearchCount: 2}, APIKey: key, User: &User{ID: 201}, Account: &Account{ID: 301}, PricingAt: start.Add(time.Hour)}
					require.NoError(t, svc.RecordUsage(context.Background(), input))
					baseline = usageRepo.lastLog
					require.NoError(t, svc.RecordUsage(ctx, input))
				} else {
					svc := newGatewayRecordUsageServiceForTest(usageRepo, userRepo, &openAIRecordUsageSubRepoStub{})
					key.Group.RateMultiplier = rate
					input := &RecordUsageInput{Result: &ForwardResult{RequestID: "campaign-gateway", Model: "claude-sonnet-4-5", Usage: ClaudeUsage{InputTokens: 1000, OutputTokens: 300, CacheCreationInputTokens: 100, CacheReadInputTokens: 200}, SearchCount: 2}, APIKey: key, User: &User{ID: 201}, Account: &Account{ID: 301}, PricingAt: start.Add(time.Hour)}
					require.NoError(t, svc.RecordUsage(context.Background(), input))
					baseline = usageRepo.lastLog
					require.NoError(t, svc.RecordUsage(ctx, input))
				}
				got := usageRepo.lastLog
				require.Equal(t, baseline.TotalCost, got.TotalCost)
				require.Equal(t, baseline.AccountStatsCost, got.AccountStatsCost)
				require.Equal(t, baseline.AccountRateMultiplier, got.AccountRateMultiplier)
				require.Equal(t, rate, got.RateMultiplier)
				require.Equal(t, got.ActualCost, userRepo.lastAmount)
				if gid == 6 {
					require.Nil(t, got.TokenCampaign)
					require.Equal(t, baseline.ActualCost, got.ActualCost)
					return
				}
				require.NotNil(t, got.TokenCampaign)
				factor := .5
				if gid == 4 {
					factor = .2
				}
				// Additive search is unchanged; every input/output/cache bucket is discounted.
				tokenCost := (got.InputCost + got.OutputCost + got.CacheCreationCost + got.CacheReadCost) * rate
				require.InDelta(t, baseline.ActualCost-tokenCost*(1-factor), got.ActualCost, 1e-12)
				require.Equal(t, baseline.ActualCost, got.TokenCampaign.OriginalActualCost)
				require.Equal(t, "championship-day", got.TokenCampaign.Phase)
			})
		}
	}
}

func TestTokenCampaignTurnRefreshAndMediaBuckets(t *testing.T) {
	settings, repo := campaignTestSettings(t, time.Now().Add(-time.Hour))
	svc := &OpenAIGatewayService{settingService: settings}
	gid := int64(4)
	ctx, _ := svc.WithOpenAIRequestPricingContext(context.Background(), &gid)
	cost := &CostBreakdown{InputCost: 2, OutputCost: 3, CacheCreationCost: 1, CacheReadCost: 1, ImageInputCost: 2, ImageOutputCost: 1, TotalCost: 10, ActualCost: 5}
	captureTokenCampaignEligibleCost(cost)
	snap, err := applyTokenCampaign(ctx, &APIKey{GroupID: &gid}, false, cost)
	require.NoError(t, err)
	require.InDelta(t, 2.2, cost.ActualCost, 1e-12)
	require.Equal(t, .2, snap.Discount)
	cfg := campaignTestConfig(time.Now().Add(-25 * time.Hour))
	cfg.Revision = 2
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	repo.raw = string(raw)
	settings.InvalidateTokenBillingCampaignCache()
	turn, _ := svc.WithOpenAITurnPricingContext(ctx, &gid)
	cost = &CostBreakdown{TotalCost: 1, ActualCost: 1, tokenCampaignEligibleActualCost: 1}
	snap, err = applyTokenCampaign(turn, &APIKey{GroupID: &gid}, false, cost)
	require.NoError(t, err)
	require.Equal(t, .9, cost.ActualCost)
	require.Equal(t, 2, snap.Revision)
	require.Equal(t, "celebration-week", snap.Phase)
	// Refreshing a turn must not mutate a queued prior turn's frozen snapshot.
	cost = &CostBreakdown{TotalCost: 1, ActualCost: 1, tokenCampaignEligibleActualCost: 1}
	snap, err = applyTokenCampaign(ctx, &APIKey{GroupID: &gid}, false, cost)
	require.NoError(t, err)
	require.Equal(t, 1, snap.Revision)
	require.InDelta(t, .2, cost.ActualCost, 1e-12)
}

func TestTokenCampaignRetryPreservesBillingFingerprintAndUpstreamQuota(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	settings, repo := campaignTestSettings(t, start)
	gid := int64(4)
	ctx := settings.freezeTokenCampaign(context.Background(), &gid, start.Add(time.Minute))
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{}
	users := &openAIRecordUsageUserRepoStub{}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, users, &openAIRecordUsageSubRepoStub{}, nil)
	input := &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "campaign-idempotency", Model: "gpt-5.1", Usage: OpenAIUsage{InputTokens: 10000, OutputTokens: 1000}}, APIKey: &APIKey{ID: 102, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: .37}}, User: &User{ID: 202}, Account: &Account{ID: 302}, PricingAt: start.Add(time.Minute)}
	input.Account.Type = AccountTypeAPIKey
	input.Account.Extra = map[string]any{"quota_limit": 100.0}
	require.NoError(t, svc.RecordUsage(ctx, input))
	first := *billingRepo.lastCmd
	require.InDelta(t, QuantizeUsageBillingAmount(usageRepo.lastLog.TotalCost*.37*.2), first.BalanceCost, 1e-12)
	require.InDelta(t, QuantizeUsageBillingAmount(usageRepo.lastLog.TotalCost), first.AccountQuotaCost, 1e-12)
	repo.raw = `{"invalid":true}`
	billingRepo.result = &UsageBillingApplyResult{Applied: false}
	require.NoError(t, svc.RecordUsage(CopyTokenCampaignPricingContext(ctx, context.Background()), input))
	require.Equal(t, first.RequestFingerprint, billingRepo.lastCmd.RequestFingerprint)
	require.Equal(t, first.BalanceCost, billingRepo.lastCmd.BalanceCost)
	require.Equal(t, first.AccountQuotaCost, billingRepo.lastCmd.AccountQuotaCost)
	require.Equal(t, 0, users.deductCalls, "dedup repository owns charging; no legacy double debit")
}

func TestTokenCampaignMixedImageGenerationExcluded(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	settings, _ := campaignTestSettings(t, start)
	gid := int64(4)
	ctx := settings.freezeTokenCampaign(context.Background(), &gid, start.Add(time.Minute))
	for _, openAI := range []bool{false, true} {
		for _, imageCount := range []int{0, 1} {
			t.Run(fmt.Sprintf("openai=%v/count=%d", openAI, imageCount), func(t *testing.T) {
				logs := &openAIRecordUsageLogRepoStub{inserted: true}
				users := &openAIRecordUsageUserRepoStub{}
				key := &APIKey{ID: 105, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: .37}}
				if openAI {
					svc := newOpenAIRecordUsageServiceForTest(logs, users, &openAIRecordUsageSubRepoStub{}, nil)
					input := &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "mixed-media", Model: "gpt-5.1", Usage: OpenAIUsage{InputTokens: 1000, OutputTokens: 100, ImageOutputTokens: 20}, ImageCount: imageCount}, APIKey: key, User: &User{ID: 205}, Account: &Account{ID: 305}, PricingAt: start.Add(time.Minute)}
					require.NoError(t, svc.RecordUsage(context.Background(), input))
					expected := logs.lastLog.ActualCost
					require.NoError(t, svc.RecordUsage(ctx, input))
					require.Equal(t, expected, logs.lastLog.ActualCost)
				} else {
					svc := newGatewayRecordUsageServiceForTest(logs, users, &openAIRecordUsageSubRepoStub{})
					input := &RecordUsageInput{Result: &ForwardResult{RequestID: "mixed-media", Model: "claude-sonnet-4-5", Usage: ClaudeUsage{InputTokens: 1000, OutputTokens: 100, ImageOutputTokens: 20}, ImageCount: imageCount}, APIKey: key, User: &User{ID: 205}, Account: &Account{ID: 305}, PricingAt: start.Add(time.Minute)}
					require.NoError(t, svc.RecordUsage(context.Background(), input))
					expected := logs.lastLog.ActualCost
					require.NoError(t, svc.RecordUsage(ctx, input))
					require.Equal(t, expected, logs.lastLog.ActualCost)
				}
				require.Nil(t, logs.lastLog.TokenCampaign)
			})
		}
	}
}

func TestTokenCampaignPublicStateMatchesPricingAndNeverChangesRates(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	settings, _ := campaignTestSettings(t, start)
	for _, tc := range []struct {
		offset   time.Duration
		status   string
		discount float64
	}{{-time.Second, "scheduled", 1}, {0, "championship-day", .2}, {24 * time.Hour, "celebration-week", .9}, {192 * time.Hour, "ended", 1}} {
		public := settings.GetPublicBillingCampaign(context.Background(), start.Add(tc.offset))
		require.Equal(t, tc.status, public.Status)
		require.NotNil(t, public.Campaign)
		require.Equal(t, "site_standard", public.Campaign.PricingBasis)
		require.Len(t, public.Campaign.Groups, 4)
		require.Equal(t, start.Add(24*time.Hour), public.Campaign.DayEndsAt)
		require.Equal(t, start.Add(192*time.Hour), public.Campaign.EndsAt)
		require.Equal(t, tc.discount, public.TokenDiscount(4, "standard"))
		require.Equal(t, 1.0, public.TokenDiscount(4, "subscription"))
		require.Equal(t, 1.0, public.TokenDiscount(6, "standard"))
	}
}

func TestTokenCampaignDoesNotExpandAcrossFallbackBillingGroups(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	settings, _ := campaignTestSettings(t, start)
	for _, pair := range [][2]int64{{4, 5}, {4, 6}, {6, 4}, {99, 4}} {
		original, settled := pair[0], pair[1]
		ctx := settings.freezeTokenCampaign(context.Background(), &original, start.Add(time.Minute))
		cost := &CostBreakdown{TotalCost: 10, ActualCost: 3.7, tokenCampaignEligibleActualCost: 3.7}
		snapshot, err := applyTokenCampaign(ctx, &APIKey{GroupID: &settled, Group: &Group{ID: settled}}, false, cost)
		require.NoError(t, err)
		require.Nil(t, snapshot)
		require.Equal(t, 3.7, cost.ActualCost)
	}
}

func TestTokenCampaignInvalidBlocksOpenAIDispatch(t *testing.T) {
	settings, repo := campaignTestSettings(t, time.Now().Add(-time.Hour))
	svc := &OpenAIGatewayService{settingService: settings}
	gid := int64(4)
	ctx := settings.freezeTokenCampaign(context.Background(), &gid, time.Now())
	repo.raw = `{"invalid":true}`
	settings.InvalidateTokenBillingCampaignCache()
	// The accepted context remains valid after a settings edit; a newly
	// admitted HTTP/WS request freezes the invalid state and is rejected before
	// model pricing or upstream selection.
	ctx = svc.WithTokenCampaignPricingAt(ctx, &gid, time.Now())
	err := svc.CheckDispatchPricingForAPIKey(ctx, &APIKey{GroupID: &gid, Group: &Group{ID: gid}}, &Account{ID: 1}, "gpt-5.1")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDispatchPricingUnavailable)
}

func TestTokenCampaignQuoteMatchesRecordWithStorageAtListPrice(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	settings, _ := campaignTestSettings(t, start)
	logs := &openAIRecordUsageLogRepoStub{inserted: true}
	users := &openAIRecordUsageUserRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(logs, users, &openAIRecordUsageSubRepoStub{})
	svc.settingService = settings
	gid := int64(4)
	key := &APIKey{ID: 107, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: .37}}
	input := &RecordUsageInput{Result: &ForwardResult{RequestID: "campaign-quote-storage", Model: "claude-sonnet-4-5", Usage: ClaudeUsage{InputTokens: 1000}, CacheStorageTokenHours: 100}, APIKey: key, User: &User{ID: 207}, Account: &Account{ID: 307}, PricingAt: start.Add(time.Hour)}
	accepted := svc.WithTokenCampaignPricingAt(context.Background(), &gid, start.Add(time.Hour))
	quote, err := svc.QuoteUsageCostWithError(accepted, input)
	require.NoError(t, err)
	require.NoError(t, svc.RecordUsage(accepted, input))
	require.NotNil(t, logs.lastLog)
	require.InDelta(t, quote, logs.lastLog.ActualCost, 1e-12)
	require.NotNil(t, logs.lastLog.TokenCampaign)
	// The snapshot discounts only the token bucket; explicit storage remains at
	// its original actual price in TotalCost/account statistics.
	require.Greater(t, logs.lastLog.TokenCampaign.OriginalActualCost, logs.lastLog.TokenCampaign.DiscountAmount)
}

type blockingTokenCampaignSettingsStub struct {
	SettingRepository
	mu    sync.Mutex
	raw   string
	err   error
	delay time.Duration
	reads atomic.Int64
}

func (r *blockingTokenCampaignSettingsStub) GetValue(context.Context, string) (string, error) {
	r.reads.Add(1)
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.raw, r.err
}

func TestTokenCampaignCacheTTLAndSingleflight(t *testing.T) {
	start := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	raw, err := json.Marshal(campaignTestConfig(start))
	require.NoError(t, err)
	repo := &blockingTokenCampaignSettingsStub{raw: string(raw), delay: 20 * time.Millisecond}
	settings := NewSettingService(repo, nil)

	nowMu := sync.Mutex{}
	now := start.Add(time.Hour)
	settings.tokenBillingCampaignTTL = 15 * time.Second
	settings.tokenBillingCampaignNow = func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return now
	}

	// Concurrent cold misses coalesce into a single DB lookup via singleflight.
	const workers = 12
	gid := int64(4)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			ctx := settings.freezeTokenCampaign(context.Background(), &gid, start.Add(time.Hour))
			require.NoError(t, tokenCampaignPricingError(ctx))
		}()
	}
	wg.Wait()
	require.Equal(t, int64(1), repo.reads.Load(), "concurrent requests must coalesce")

	// Subsequent reads within TTL hit in-process cache without querying DB.
	for i := 0; i < 5; i++ {
		resp := settings.GetPublicBillingCampaign(context.Background(), start.Add(time.Hour))
		require.Equal(t, "championship-day", resp.Status)
	}
	require.Equal(t, int64(1), repo.reads.Load(), "reads within TTL must hit cache")

	// After TTL expires, next read reloads from DB.
	nowMu.Lock()
	now = now.Add(16 * time.Second)
	nowMu.Unlock()
	resp := settings.GetPublicBillingCampaign(context.Background(), start.Add(time.Hour))
	require.Equal(t, "championship-day", resp.Status)
	require.Equal(t, int64(2), repo.reads.Load(), "expired cache must reload from DB")

	// Errors are never cached: once DB recovers, the next call succeeds immediately.
	repo.mu.Lock()
	repo.err = errors.New("temporary db hiccup")
	repo.mu.Unlock()
	settings.InvalidateTokenBillingCampaignCache()
	failedCtx := settings.freezeTokenCampaign(context.Background(), &gid, start.Add(time.Hour))
	require.ErrorIs(t, tokenCampaignPricingError(failedCtx), ErrDispatchPricingUnavailable)

	repo.mu.Lock()
	repo.err = nil
	repo.mu.Unlock()
	recoveredCtx := settings.freezeTokenCampaign(context.Background(), &gid, start.Add(time.Hour))
	require.NoError(t, tokenCampaignPricingError(recoveredCtx))
}
