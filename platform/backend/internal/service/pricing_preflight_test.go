//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func preflightContext(t *testing.T, group *Group) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set("api_key", &APIKey{Group: group})
	return c
}

func TestDispatchPricingCNDoesNotBorrowClaudeFallback(t *testing.T) {
	billing := NewBillingService(&config.Config{}, nil)
	svc := &OpenAIGatewayService{billingService: billing, resolver: NewModelPricingResolver(nil, billing)}
	account := &Account{Platform: PlatformOpenCodeGo, Credentials: map[string]any{"model_mapping": map[string]any{"claude-sonnet-4-5": "qwen3.8-max"}}}
	c := preflightContext(t, &Group{ID: 1, Platform: PlatformOpenCodeGo})
	err := svc.checkOpenAIDispatchPricing(context.Background(), c, account, "claude-sonnet-4-5")
	require.ErrorIs(t, err, ErrDispatchPricingUnavailable)
	require.Equal(t, http.StatusServiceUnavailable, c.Writer.Status())
}

func TestDispatchPricingSimpleModeChecksOptInKeyLimits(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		cfg := &config.Config{RunMode: config.RunModeSimple, SimpleModeKeyRateLimitEnabled: enabled}
		billing := NewBillingService(cfg, nil)
		c := preflightContext(t, &Group{ID: 1, Platform: PlatformOpenAI})
		key, _ := c.Get("api_key")
		key.(*APIKey).RateLimit5h = 1
		err := CheckDispatchPricingForRequest(context.Background(), c, nil, "unknown-unpriced-model", PricingPreflightToken, NewModelPricingResolver(nil, billing), billing)
		if enabled {
			require.ErrorIs(t, err, ErrDispatchPricingUnavailable)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestForwardRejectsMissingPriceBeforeNetwork(t *testing.T) {
	upstream := &anthropicHTTPUpstreamRecorder{}
	svc := newForwardPartialUsageServiceForTest(upstream)
	svc.billingService = NewBillingService(svc.cfg, nil)
	svc.resolver = NewModelPricingResolver(nil, svc.billingService)
	c := preflightContext(t, &Group{ID: 1, Platform: PlatformAnthropic})
	result, err := svc.Forward(context.Background(), c, newAnthropicAPIKeyAccountForTest(), &ParsedRequest{Model: "unknown-unpriced-model"})
	require.ErrorIs(t, err, ErrDispatchPricingUnavailable)
	require.Nil(t, result)
	require.Nil(t, upstream.lastReq, "missing-price guard must run before transport")
	require.Equal(t, http.StatusServiceUnavailable, c.Writer.Status())
}

func TestDispatchPricingUsesCompleteChannelAndAccountMappingChain(t *testing.T) {
	channel := Channel{ID: 1, Name: "mapped", Status: StatusActive, GroupIDs: []int64{1}, BillingModelSource: BillingModelSourceUpstream, ModelMapping: map[string]map[string]string{PlatformOpenAI: {"public-alias": "account-alias"}}}
	group := Group{ID: 1, Platform: PlatformOpenAI}
	svc := newPlazaServiceWithBilling([]Channel{channel}, []Group{group}, map[int64]string{1: PlatformOpenAI}, nil)
	account := &Account{Credentials: map[string]any{"model_mapping": map[string]any{"account-alias": "gpt-6.1-terra"}}}
	id := int64(1)
	key := &APIKey{GroupID: &id, Group: &group}
	candidates := dispatchBillingCandidates(context.Background(), svc.resolver, key, account, "public-alias")
	require.Equal(t, "gpt-6.1-terra", candidates[0])
	require.NoError(t, CheckDispatchPricing(context.Background(), PricingPreflightInput{Model: "public-alias", Group: &group, GroupID: &id, Resolver: svc.resolver, Billing: svc.billingService, Account: account, Candidates: candidates}))
}

func TestDispatchPricingWritesSSEErrorAfterKeepalive(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set("api_key", &APIKey{Group: &Group{ID: 1, Platform: PlatformOpenAI}})
	c.Header("Content-Type", "text/event-stream")
	_, _ = c.Writer.WriteString(": keepalive\n\n")
	c.Writer.Flush()
	billing := NewBillingService(&config.Config{}, nil)
	err := CheckDispatchPricingForRequest(context.Background(), c, nil, "unknown-unpriced-model", PricingPreflightToken, NewModelPricingResolver(nil, billing), billing)
	require.ErrorIs(t, err, ErrDispatchPricingUnavailable)
	require.Equal(t, http.StatusOK, recorder.Code, "headers already sent must remain unchanged")
	require.Contains(t, recorder.Body.String(), "event: error\n")
	require.Contains(t, recorder.Body.String(), "pricing_unavailable")
}

func TestWSPricingUsesCurrentAPIKeySnapshot(t *testing.T) {
	billing := NewBillingService(&config.Config{}, nil)
	svc := &OpenAIGatewayService{billingService: billing, resolver: NewModelPricingResolver(nil, billing)}
	input, output := 1e-6, 5e-6
	groupID := int64(1)
	old := &APIKey{GroupID: &groupID, Group: &Group{ID: 1, Platform: PlatformOpenAI, ModelPricing: []ChannelModelPricing{{Models: []string{"custom-priced-only"}, BillingMode: BillingModeToken, InputPrice: &input, OutputPrice: &output}}}}
	fresh := &APIKey{GroupID: &groupID, Group: &Group{ID: 1, Platform: PlatformOpenAI}}
	require.NoError(t, svc.CheckDispatchPricingForAPIKey(context.Background(), old, nil, "custom-priced-only"))
	require.ErrorIs(t, svc.CheckDispatchPricingForAPIKey(context.Background(), fresh, nil, "custom-priced-only"), ErrDispatchPricingUnavailable, "a refreshed turn must not borrow its connection's retired price card")
}

func TestCheckDispatchPricing_AllowsExplicitFreeRequestTier(t *testing.T) {
	zero := float64(0)
	resolver := NewModelPricingResolver(nil, nil)
	// The resolver is not needed for this pure helper test; validate the tier
	// distinction directly through the same predicate used by dispatch checks.
	require.True(t, requestPricingConfigured(&ResolvedPricing{
		Mode:         BillingModePerRequest,
		RequestTiers: []PricingInterval{{TierLabel: "free", PerRequestPrice: &zero}},
	}))
	_ = resolver
}

func TestDispatchPricingCandidatesIncludesMappedModel(t *testing.T) {
	a := &Account{Credentials: map[string]any{"model_mapping": map[string]any{"alias": "real-model"}}}
	require.Equal(t, []string{"alias", "real-model"}, dispatchPricingCandidates(a, "alias"))
}

func TestCheckDispatchPricingUnknownModelFailsClosed(t *testing.T) {
	billing := NewBillingService(nil, nil)
	err := CheckDispatchPricing(context.Background(), PricingPreflightInput{
		Billing: billing,
		Model:   "unknown-unpriced-model",
		Kind:    PricingPreflightToken,
	})
	require.ErrorIs(t, err, ErrDispatchPricingUnavailable)
}

func TestCheckDispatchPricingSubscriptionStillChecksPrice(t *testing.T) {
	billing := NewBillingService(nil, nil)
	err := CheckDispatchPricing(context.Background(), PricingPreflightInput{
		Billing: billing,
		Group:   &Group{SubscriptionType: SubscriptionTypeSubscription},
		Model:   "unknown-unpriced-model",
		Kind:    PricingPreflightToken,
	})
	require.ErrorIs(t, err, ErrDispatchPricingUnavailable)
}
