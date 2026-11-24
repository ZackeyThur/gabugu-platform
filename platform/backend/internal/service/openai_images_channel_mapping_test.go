//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSelectAccountForImages_ChannelMappingAndPricingRestrictions(t *testing.T) {
	const publicModel = "gemini-3-pro-image"
	const channelModel = "monkey-image-pro"
	const upstreamModel = "monkey-image-pro-upstream"
	const groupID int64 = 701110

	for _, loadBatch := range []bool{false, true} {
		mode := "legacy_single"
		if loadBatch {
			mode = "legacy_load_batch"
		}
		t.Run(mode, func(t *testing.T) {
			for _, tc := range []struct {
				name         string
				source       string
				restrict     bool
				pricedModels []string
				allowed      bool
			}{
				{name: "canonical_selects_alias_only_account", source: BillingModelSourceRequested, allowed: true},
				{name: "requested_allows_public_without_alias", source: BillingModelSourceRequested, restrict: true, pricedModels: []string{publicModel}, allowed: true},
				{name: "requested_rejects_alias_without_public", source: BillingModelSourceRequested, restrict: true, pricedModels: []string{channelModel, upstreamModel}},
				{name: "channel_mapped_allows_alias", source: BillingModelSourceChannelMapped, restrict: true, pricedModels: []string{channelModel}, allowed: true},
				{name: "channel_mapped_rejects_public_only", source: BillingModelSourceChannelMapped, restrict: true, pricedModels: []string{publicModel}},
				{name: "upstream_allows_final_account_model", source: BillingModelSourceUpstream, restrict: true, pricedModels: []string{upstreamModel}, allowed: true},
				{name: "upstream_rejects_channel_alias_only", source: BillingModelSourceUpstream, restrict: true, pricedModels: []string{channelModel}},
				{name: "upstream_rejects_public_only", source: BillingModelSourceUpstream, restrict: true, pricedModels: []string{publicModel}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					resetOpenAIAdvancedSchedulerSettingCacheForTest()
					t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)
					channel := Channel{
						ID: 1, Status: StatusActive, GroupIDs: []int64{groupID},
						BillingModelSource: tc.source, RestrictModels: tc.restrict,
						ModelMapping: map[string]map[string]string{PlatformOpenAI: {publicModel: channelModel}},
					}
					if len(tc.pricedModels) > 0 {
						channel.ModelPricing = []ChannelModelPricing{{Platform: PlatformOpenAI, Models: tc.pricedModels}}
					}
					account := Account{
						ID: 65, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
						Status: StatusActive, Schedulable: true, Concurrency: 1,
						Credentials: map[string]any{"model_mapping": map[string]any{channelModel: upstreamModel}},
					}
					cfg := &config.Config{}
					cfg.Gateway.Scheduling.LoadBatchEnabled = loadBatch
					svc := &OpenAIGatewayService{
						accountRepo:    schedulerTestOpenAIAccountRepo{accounts: []Account{account}},
						channelService: newTestChannelService(makeStandardRepo(channel, map[int64]string{groupID: PlatformOpenAI})),
						cache:          &schedulerTestGatewayCache{}, cfg: cfg,
						concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
					}
					gid := groupID
					selection, _, err := svc.SelectAccountWithSchedulerForImages(
						context.Background(), &gid, "", publicModel, nil, OpenAIImagesCapabilityNative,
					)
					if selection != nil && selection.ReleaseFunc != nil {
						t.Cleanup(selection.ReleaseFunc)
					}
					if !tc.allowed {
						require.ErrorIs(t, err, ErrNoAvailableAccounts)
						require.Nil(t, selection, "pricing restrictions must prevent account selection")
						return
					}
					require.NoError(t, err)
					require.NotNil(t, selection)
					require.NotNil(t, selection.Account)
					require.Equal(t, account.ID, selection.Account.ID)
					require.Equal(t, upstreamModel, selection.Account.GetMappedModel(channelModel))
					// The account declares only the private alias: selecting it for the
					// public name requires the channel mapping, not a permissive account.
					require.NotContains(t, selection.Account.Credentials["model_mapping"], publicModel)
				})
			}
		})
	}
}
