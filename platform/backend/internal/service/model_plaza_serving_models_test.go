//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type plazaServingModelStub map[int64][]string

func (m plazaServingModelStub) GetAvailableModels(_ context.Context, groupID *int64, _ string) []string {
	return m[*groupID]
}

func TestPlazaServingModelsFillPricedModelsWithoutChannels(t *testing.T) {
	groups := []Group{{ID: 1, Platform: PlatformOpenAI, RateMultiplier: 0.4}, {ID: 2, Platform: PlatformAnthropic}}
	svc := newPlazaService(nil, groups, nil)
	svc.billingService = NewBillingService(&config.Config{}, nil)
	svc.resolver = NewModelPricingResolver(nil, svc.billingService)
	svc.modelLister = plazaServingModelStub{1: {"gpt-6.1-sol", "gpt-6-astra", "gpt-*", "unpriced-model"}, 2: {"claude-opus-5-5"}}
	out, err := svc.ListGroups(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 2)
	byID := map[int64]PlazaGroup{}
	for _, g := range out {
		byID[g.ID] = g
	}
	require.Len(t, byID[1].Models, 2)
	require.Equal(t, "gpt-6-astra", byID[1].Models[0].Name)
	require.Equal(t, "gpt-6.1-sol", byID[1].Models[1].Name)
	require.InDelta(t, 2e-6, *byID[1].Models[1].Pricing.InputPrice, 1e-12)
	require.Equal(t, "claude-opus-5-5", byID[2].Models[0].Name)
}

func TestPlazaServingModelsRespectAllowlistAndDoNotPublishMappingTargets(t *testing.T) {
	group := Group{ID: 1, Platform: PlatformOpenAI, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"gpt-6.1-sol"}}}
	svc := newPlazaService([]Channel{plazaPricedChannel(1, "ch", []int64{1}, PlatformOpenAI, "gpt-6-astra")}, []Group{group}, nil)
	svc.billingService = NewBillingService(&config.Config{}, nil)
	svc.resolver = NewModelPricingResolver(nil, svc.billingService)
	svc.modelLister = plazaServingModelStub{1: {"gpt-6.1-sol", "gpt-6-astra", "internal-upstream-target"}}
	out, err := svc.ListGroups(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].Models, 1)
	require.Equal(t, "gpt-6.1-sol", out[0].Models[0].Name)
}

func TestPlazaServiceTierPricesUseBillingScheduleAndPreserveFreeFast(t *testing.T) {
	for _, freeFast := range []bool{false, true} {
		group := Group{ID: 1, Platform: PlatformOpenAI, LongContextPricingEnabled: true, FreeOpenAIFast: freeFast, WebSearchPricePerCall: testPtrFloat64(0.5)}
		svc := newPlazaService(nil, []Group{group}, nil)
		svc.billingService = NewBillingService(&config.Config{}, nil)
		svc.resolver = NewModelPricingResolver(nil, svc.billingService)
		svc.modelLister = plazaServingModelStub{1: {"gpt-6.1-sol"}}
		out, err := svc.ListGroups(context.Background())
		require.NoError(t, err)
		require.Len(t, out, 1)
		require.Equal(t, 0.5, out[0].AdditionalCharges.WebSearchPerCall)
		tiers := map[string]*ChannelModelPricing{}
		for _, tier := range out[0].Models[0].ServiceTiers {
			tiers[tier.Tier] = tier.Pricing
		}
		require.Contains(t, tiers, "priority")
		require.Contains(t, tiers, "flex")
		want := 4e-6
		if freeFast {
			want = 2e-6
		}
		require.InDelta(t, want, *tiers["priority"].InputPrice, 1e-12)
		require.InDelta(t, 1e-6, *tiers["flex"].InputPrice, 1e-12)
		require.Len(t, tiers["priority"].Intervals, 2)
		require.Nil(t, tiers["priority"].CacheWrite1hPrice, "flat cache pricing must not invent a one-hour retention SKU")
	}
}

func TestPlazaPublicAliasUsesChannelBillingModelWithoutExposingTarget(t *testing.T) {
	channel := plazaPricedChannel(1, "mapped", []int64{1}, PlatformOpenAI, "gpt-6.1-sol")
	channel.ModelMapping = map[string]map[string]string{PlatformOpenAI: {"gpt-6-luna": "gpt-6.1-sol"}}
	channel.RestrictModels = true
	channel.BillingModelSource = BillingModelSourceChannelMapped
	channel.ModelPricing[0].InputPrice = testPtrFloat64(2e-6)
	channel.ModelPricing[0].OutputPrice = testPtrFloat64(10e-6)
	group := Group{ID: 1, Platform: PlatformOpenAI, LongContextPricingEnabled: true, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"gpt-6-luna"}}}
	svc := newPlazaServiceWithBilling([]Channel{channel}, []Group{group}, map[int64]string{1: PlatformOpenAI}, nil)
	out, err := svc.ListGroups(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].Models, 1)
	m := out[0].Models[0]
	require.Equal(t, "gpt-6-luna", m.Name)
	require.InDelta(t, 2e-6, *m.Pricing.InputPrice, 1e-12, "must price the channel target rather than the cheaper public alias")
	require.NotEmpty(t, m.ServiceTiers)
}
