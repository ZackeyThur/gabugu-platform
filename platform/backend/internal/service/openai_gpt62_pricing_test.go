//go:build unit

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGPT62FallbackKey(t *testing.T) {
	for model, want := range map[string]string{
		"gpt-6.2-sol":                "gpt-6.2-sol",
		"GPT-6.2-Sol":                "gpt-6.2-sol",
		"openai/gpt-6.2-sol":         "gpt-6.2-sol",
		"gpt-6.2-sol-high":           "gpt-6.2-sol",
		"gpt-6.2-sol-xhigh":          "gpt-6.2-sol",
		"gpt-6.2-sol-openai-compact": "gpt-6.2-sol",
		"gpt-6.2-sol-20261112":       "gpt-6.2-sol",
		"gpt-6.2-astra":              "gpt-6.2-astra",
		"openai/gpt-6.2-astra-low":   "gpt-6.2-astra",
		"gpt_6.2_astra":              "gpt-6.2-astra",
	} {
		require.Equal(t, want, openAIGPT62FallbackKey(model), model)
	}
	for _, model := range []string{
		"", "gpt-6.2", "gpt-6.2-terra", "gpt-6.2-luna", "gpt-6.2-sol-foo", "gpt-6.2-sol-2026",
		"gpt-6.3-sol", "gpt-6.1-sol", "gpt-6-astra", "gpt-6", "gpt-6.2-astra-law", "claude-opus-5-5",
	} {
		require.Equal(t, "", openAIGPT62FallbackKey(model), model)
	}
}

func TestGPT62FallbackPricingIsBillable(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	for _, tc := range []struct {
		models             []string
		in, out, cacheRead float64
		cacheWrite         float64
	}{
		{[]string{"gpt-6.2-sol", "openai/gpt-6.2-sol", "gpt-6.2-sol-high", "gpt-6.2-sol-20261112"}, 2e-6, 10e-6, 0.1e-6, 2.5e-6},
		{[]string{"gpt-6.2-astra", "openai/gpt-6.2-astra", "gpt-6.2-astra-xhigh", "gpt-6.2-astra-openai-compact"}, 8e-6, 40e-6, 0.4e-6, 10e-6},
	} {
		for _, model := range tc.models {
			t.Run(model, func(t *testing.T) {
				p, err := svc.GetModelPricing(model)
				require.NoError(t, err)
				require.InDelta(t, tc.in, p.InputPricePerToken, 1e-15)
				require.InDelta(t, tc.out, p.OutputPricePerToken, 1e-15)
				require.InDelta(t, tc.cacheRead, p.CacheReadPricePerToken, 1e-15)
				require.InDelta(t, tc.cacheWrite, p.CacheCreationPricePerToken, 1e-15)
				// Fast tier is 2x Standard.
				require.InDelta(t, tc.in*2, p.InputPricePerTokenPriority, 1e-15)
				require.InDelta(t, tc.out*2, p.OutputPricePerTokenPriority, 1e-15)
			})
		}
	}
}

// Anything that is not an exact GPT-6.2 Sol/Astra must keep failing closed; the
// GPT-6.2 cards must never become a catch-all for neighbouring names.
func TestGPT62NeighboursStillFailClosed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "resources", "model-pricing", "model_prices_and_context_window.json"))
	require.NoError(t, err)
	catalog := &PricingService{}
	catalog.pricingData, err = catalog.parsePricingData(data)
	require.NoError(t, err)

	for name, svc := range map[string]*BillingService{
		"no-catalog":      NewBillingService(&config.Config{}, nil),
		"bundled-catalog": NewBillingService(&config.Config{}, catalog),
	} {
		for _, model := range []string{"gpt-6.2", "gpt-6.2-terra", "gpt-6.2-luna", "gpt-6.3-sol", "gpt-6.2-sol-foo", "openai/gpt-6.2"} {
			t.Run(name+"/"+model, func(t *testing.T) {
				_, err := svc.GetModelPricing(model)
				require.ErrorIs(t, err, ErrModelPricingUnavailable)
			})
		}
		// The bundled catalog does not know GPT-6.2, so the fallback card must be used.
		for _, model := range []string{"gpt-6.2-sol", "gpt-6.2-astra"} {
			_, err := svc.GetModelPricing(model)
			require.NoError(t, err, name+"/"+model)
		}
	}
}

func TestGPT62CostAndServiceTiers(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	// 100k in + 100k out + 100k cache read = 200k input-side tokens: below the
	// 272k long-context threshold, so only the service tier changes the price.
	// Expected values below are USD per 1M tokens of each kind, scaled by 0.1.
	tokens := UsageTokens{InputTokens: 100_000, OutputTokens: 100_000, CacheReadTokens: 100_000}
	const scale = 0.1

	for _, tc := range []struct {
		model                   string
		standard, fast, flex, u float64
	}{
		{"gpt-6.2-sol", 2 + 10 + 0.1, 2 * (2 + 10 + 0.1), 0.5 * (2 + 10 + 0.1), openAIUltrafastMultiplier("gpt-6.2-sol", timezone.Now()) * (2 + 10 + 0.1)},
		{"gpt-6.2-astra", 8 + 40 + 0.4, 2 * (8 + 40 + 0.4), 0.5 * (8 + 40 + 0.4), openAIUltrafastMultiplier("gpt-6.2-astra", timezone.Now()) * (8 + 40 + 0.4)},
	} {
		t.Run(tc.model, func(t *testing.T) {
			for tier, want := range map[string]float64{"": tc.standard, "priority": tc.fast, "fast": tc.fast, "flex": tc.flex, "ultrafast": tc.u} {
				cost, err := svc.CalculateCostWithServiceTier(tc.model, tokens, 1, tier)
				require.NoError(t, err, tier)
				require.InDelta(t, want*scale, cost.TotalCost, 1e-9, "tier %q", tier)
			}
		})
	}

	// Ultra Fast follows the official schedule: 3x before 2027-03-01 00:00 Pacific, then 6x.
	sol, err := svc.GetModelPricing("gpt-6.2-sol")
	require.NoError(t, err)
	require.Equal(t, openAIUltrafastMultiplier("gpt-6.2-sol", timezone.Now()), configuredServiceTierMultiplier("ultrafast", sol))
	astra, err := svc.GetModelPricing("gpt-6.2-astra")
	require.NoError(t, err)
	require.Equal(t, openAIUltrafastMultiplier("gpt-6.2-astra", timezone.Now()), configuredServiceTierMultiplier("ultrafast", astra))
}

func TestGPT62LongContextSurcharge(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	for _, n := range []int{271_999, 272_000, 272_001} {
		cost, err := svc.CalculateCostWithServiceTier("gpt-6.2-sol", UsageTokens{InputTokens: n, OutputTokens: 1000}, 1, "")
		require.NoError(t, err)
		im, om := 1.0, 1.0
		if n > 272_000 {
			im, om = 2, 1.5
		}
		require.InDelta(t, float64(n)*2e-6*im, cost.InputCost, 1e-9, "input at %d", n)
		require.InDelta(t, 1000*10e-6*om, cost.OutputCost, 1e-9, "output at %d", n)
	}
}

// A catalog entry for GPT-6.2 (when upstream publishes one) must win over the
// provisional fallback card.
func TestGPT62CatalogEntryWinsOverFallback(t *testing.T) {
	ps := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gpt-6.2-sol": {InputCostPerToken: 3e-6, OutputCostPerToken: 15e-6, CacheReadInputTokenCost: 0.3e-6, LiteLLMProvider: "openai", Mode: "chat"},
	}}
	svc := NewBillingService(&config.Config{}, ps)
	p, err := svc.GetModelPricing("gpt-6.2-sol")
	require.NoError(t, err)
	require.InDelta(t, 3e-6, p.InputPricePerToken, 1e-15)
	require.InDelta(t, 15e-6, p.OutputPricePerToken, 1e-15)
	// Astra has no catalog entry here, so it still uses its own fallback card.
	a, err := svc.GetModelPricing("gpt-6.2-astra")
	require.NoError(t, err)
	require.InDelta(t, 8e-6, a.InputPricePerToken, 1e-15)
}

// Group 38 ("AZ GPT") scenario: openai group, multiplier 1.0, account mapping
// gpt-6.2-* -> gpt-6.2-*. Before this change both models returned 503.
func TestGPT62PassesDispatchPricingPreflightAndGroupMultiplier(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	resolver := NewModelPricingResolver(nil, svc)
	group := &Group{ID: 38, Platform: "openai"}
	gid := group.ID
	tokens := UsageTokens{InputTokens: 100_000, OutputTokens: 100_000, CacheReadTokens: 100_000}
	const scale = 0.1 // expected totals are USD per 1M tokens of each kind

	for _, tc := range []struct {
		model string
		total float64
	}{
		{"gpt-6.2-sol", 2 + 10 + 0.1},
		{"gpt-6.2-astra", 8 + 40 + 0.4},
		{"gpt-6.2-astra-high", 8 + 40 + 0.4},
	} {
		t.Run(tc.model, func(t *testing.T) {
			require.NoError(t, CheckDispatchPricing(context.Background(), PricingPreflightInput{
				Resolver: resolver, Billing: svc, Group: group, GroupID: &gid,
				Model: tc.model, Kind: PricingPreflightToken, Candidates: []string{tc.model},
			}))
			for _, rate := range []float64{1.0, 0.8} {
				cost, err := svc.CalculateCostUnified(CostInput{
					Ctx: context.Background(), Model: tc.model, Group: group, GroupID: &gid,
					Tokens: tokens, RequestCount: 1, RateMultiplier: rate, Resolver: resolver,
				})
				require.NoError(t, err)
				require.InDelta(t, tc.total*scale, cost.TotalCost, 1e-9)
				require.InDelta(t, tc.total*scale*rate, cost.ActualCost, 1e-9, "rate %v", rate)
			}
		})
	}

	// Neighbours are still rejected by the same preflight.
	for _, model := range []string{"gpt-6.2", "gpt-6.2-terra"} {
		require.ErrorIs(t, CheckDispatchPricing(context.Background(), PricingPreflightInput{
			Resolver: resolver, Billing: svc, Group: group, GroupID: &gid,
			Model: model, Kind: PricingPreflightToken, Candidates: []string{model},
		}), ErrDispatchPricingUnavailable, model)
	}
}
