package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecordUsage_GeminiImagesZeroTokensPreservesPublicIdentityAndBillsActual4K(t *testing.T) {
	const publicModel = "gemini-3-pro-image"
	const channelModel = "monkey-image-pro"
	const upstreamModel = "monkey-image-pro"
	const groupID int64 = 701111
	const price4K = 0.371

	for _, tc := range []struct {
		name        string
		independent bool
		multiplier  float64
	}{
		{name: "shared_group_multiplier", multiplier: 0.8},
		{name: "independent_image_multiplier", independent: true, multiplier: 1.25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			billingRepo := &openAIRecordUsageBillingRepoStub{}
			svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(
				usageRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil,
			)
			groupPrice := price4K
			group := &Group{
				ID: groupID, Platform: PlatformOpenAI, Status: StatusActive,
				AllowImageGeneration: true, ImageCountBilling: true,
				RateMultiplier: 0.8, ImageRateIndependent: tc.independent,
				ImageRateMultiplier: tc.multiplier, ImagePrice4K: &groupPrice,
			}
			mapping := ChannelMappingResult{
				ChannelID: 1, Mapped: true, MappedModel: channelModel,
				BillingModelSource: BillingModelSourceRequested,
			}
			fields := mapping.ToUsageFields(publicModel, upstreamModel)
			result := &OpenAIForwardResult{
				RequestID: "gemini-images-zero-token-4k", Model: publicModel, UpstreamModel: upstreamModel,
				Usage: OpenAIUsage{}, Duration: time.Second,
				ImageEndpoint: true, ImageCount: 1,
				// Actual output must override a lower requested size, including
				// providers that omit usageMetadata entirely.
				ImageSize: ImageBillingSize2K, ImageInputSize: "2K",
				ImageOutputSizes: []string{"3072x5504"},
			}
			err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
				Result: result,
				APIKey: &APIKey{ID: 71, GroupID: i64p(groupID), Group: group},
				User:   &User{ID: 72}, Account: &Account{ID: 65, Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
				InboundEndpoint: "/v1/images/edits", UpstreamEndpoint: "/v1beta/models/monkey-image-pro:generateContent",
				RequestPayloadHash: "gemini-images-edit-payload", ChannelUsageFields: fields,
			})
			require.NoError(t, err)
			require.Equal(t, 1, usageRepo.calls)
			log := usageRepo.lastLog
			require.NotNil(t, log)
			require.Equal(t, publicModel, log.RequestedModel)
			require.Equal(t, publicModel, log.Model)
			require.Equal(t, upstreamModel, optionalStringValue(log.UpstreamModel))
			require.Equal(t, publicModel+"→"+channelModel, optionalStringValue(log.ModelMappingChain))
			require.Equal(t, channelModel, fields.ChannelMappedModel)
			require.Equal(t, BillingModelSourceRequested, fields.BillingModelSource)
			require.Equal(t, 1, log.ImageCount)
			require.Zero(t, log.TotalTokens())
			require.Equal(t, ImageBillingSize4K, optionalStringValue(log.ImageSize))
			require.Equal(t, "2K", optionalStringValue(log.ImageInputSize))
			require.Equal(t, "3072x5504", optionalStringValue(log.ImageOutputSize))
			require.Equal(t, ImageSizeSourceOutput, optionalStringValue(log.ImageSizeSource))
			require.Equal(t, map[string]int{ImageBillingSize4K: 1}, log.ImageSizeBreakdown)
			require.Equal(t, string(BillingModeImage), optionalStringValue(log.BillingMode))
			require.InDelta(t, price4K, log.TotalCost, 1e-12)
			require.InDelta(t, price4K*tc.multiplier, log.ActualCost, 1e-12)
			require.InDelta(t, tc.multiplier, log.RateMultiplier, 1e-12)
			require.Equal(t, 1, billingRepo.calls, "image-only usage must enter the normal durable billing path")
			require.NotNil(t, billingRepo.lastCmd)
			require.Equal(t, 1, billingRepo.lastCmd.ImageCount)
			require.InDelta(t, price4K*tc.multiplier, billingRepo.lastCmd.BalanceCost, 1e-8)
			require.NotEmpty(t, billingRepo.lastCmd.RequestFingerprint)
			require.Equal(t, "gemini-images-edit-payload", billingRepo.lastCmd.RequestPayloadHash)
		})
	}
}
