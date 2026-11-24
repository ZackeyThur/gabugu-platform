//go:build unit

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

var (
	gemini4BeforeListPrice = time.Date(2026, time.December, 1, 7, 59, 59, 0, time.UTC) // 2026-11-30 23:59:59 PST
	gemini4AtListPrice     = time.Date(2026, time.December, 1, 8, 0, 0, 0, time.UTC)   // 2026-12-01 00:00 PST
)

func TestGemini4CarbonFallbackPricingIsBillable(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	for _, model := range []string{
		"gemini-4-carbon",
		"gemini-4-carbon-high",
		"gemini-4-carbon-low",
		"gemini-4-carbon-medium",
		"models/gemini-4-carbon",
	} {
		t.Run(model, func(t *testing.T) {
			p, err := svc.getModelPricingAt(model, gemini4BeforeListPrice)
			require.NoError(t, err)
			require.InDelta(t, 2e-6, p.InputPricePerToken, 1e-15)
			require.InDelta(t, 10e-6, p.OutputPricePerToken, 1e-15)
			require.InDelta(t, 0.1e-6, p.CacheReadPricePerToken, 1e-15)
			require.InDelta(t, 3.6e-6, p.InputPricePerTokenPriority, 1e-15)
		})
	}
}

func TestGemini4PromoEndsOnDecemberFirst(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	for _, model := range []string{"gemini-4-argon", "gemini-4-argon-high", "gemini-4-carbon", "gemini-4-carbon-medium"} {
		t.Run(model, func(t *testing.T) {
			before, err := svc.getModelPricingAt(model, gemini4BeforeListPrice)
			require.NoError(t, err)
			require.InDelta(t, 2e-6, before.InputPricePerToken, 1e-15)

			after, err := svc.getModelPricingAt(model, gemini4AtListPrice)
			require.NoError(t, err)
			require.InDelta(t, 4e-6, after.InputPricePerToken, 1e-15)
			require.InDelta(t, 20e-6, after.OutputPricePerToken, 1e-15)
			require.InDelta(t, 0.2e-6, after.CacheReadPricePerToken, 1e-15)
			require.InDelta(t, 7.2e-6, after.InputPricePerTokenPriority, 1e-15)
			require.InDelta(t, 36e-6, after.OutputPricePerTokenPriority, 1e-15)
			require.InDelta(t, 0.36e-6, after.CacheReadPricePerTokenPriority, 1e-15)
		})
	}
	// The shared fallback card must not be mutated by the date switch.
	again, err := svc.getModelPricingAt("gemini-4-carbon", gemini4BeforeListPrice)
	require.NoError(t, err)
	require.InDelta(t, 2e-6, again.InputPricePerToken, 1e-15)
}

func TestGemini4PromoSwitchSkipsCyberAndListPricedCards(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	cyber, err := svc.getModelPricingAt("gemini-4-argon-cyber-permissive", gemini4AtListPrice)
	require.NoError(t, err)
	require.InDelta(t, 4e-6, cyber.InputPricePerToken, 1e-15)

	listPriced := &ModelPricing{InputPricePerToken: 4e-6, OutputPricePerToken: 20e-6, CacheReadPricePerToken: 0.2e-6}
	require.Same(t, listPriced, applyGemini4PromoPricing("gemini-4-carbon", listPriced, gemini4AtListPrice))

	other := &ModelPricing{InputPricePerToken: 2e-6}
	require.Same(t, other, applyGemini4PromoPricing("gemini-3.1-pro-preview", other, gemini4AtListPrice))
}

func TestGemini4CarbonCatalogEntryAndDateSwitch(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "resources", "model-pricing", "model_prices_and_context_window.json"))
	require.NoError(t, err)
	ps := &PricingService{}
	pd, err := ps.parsePricingData(data)
	require.NoError(t, err)
	ps.pricingData = pd
	svc := NewBillingService(&config.Config{}, ps)

	require.NotNil(t, ps.GetModelPricing("gemini-4-carbon-high"))
	tokens := UsageTokens{InputTokens: 1_000_000, OutputTokens: 1_000_000, CacheReadTokens: 1_000_000}
	resolver := NewModelPricingResolver(nil, svc)
	group := &Group{ID: 24, Platform: "gemini"}
	gid := group.ID
	for _, tc := range []struct {
		at    time.Time
		total float64
	}{
		{gemini4BeforeListPrice, 2 + 10 + 0.1},
		{gemini4AtListPrice, 4 + 20 + 0.2},
	} {
		cost, err := svc.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: "gemini-4-carbon-high", Group: group, GroupID: &gid,
			Tokens: tokens, RequestCount: 1, RateMultiplier: 1, PricingAt: tc.at, Resolver: resolver,
		})
		require.NoError(t, err)
		require.InDelta(t, tc.total, cost.TotalCost, 1e-9, "at %s", tc.at)

		discounted, err := svc.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: "gemini-4-carbon", Group: group, GroupID: &gid,
			Tokens: tokens, RequestCount: 1, RateMultiplier: 0.6, PricingAt: tc.at, Resolver: resolver,
		})
		require.NoError(t, err)
		require.InDelta(t, tc.total*0.6, discounted.ActualCost, 1e-9, "group multiplier at %s", tc.at)
	}
}

func TestGemini4CarbonPassesDispatchPricingPreflight(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	resolver := NewModelPricingResolver(nil, svc)
	group := &Group{ID: 24, Platform: "gemini"}
	gid := group.ID
	for _, model := range []string{"gemini-4-carbon", "gemini-4-carbon-high"} {
		require.NoError(t, CheckDispatchPricing(context.Background(), PricingPreflightInput{
			Resolver: resolver, Billing: svc, Group: group, GroupID: &gid,
			Model: model, Kind: PricingPreflightToken, Candidates: []string{model},
		}), model)
	}
}
