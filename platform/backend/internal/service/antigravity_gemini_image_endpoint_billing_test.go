//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAntigravityForwardGemini_ImageCountBillingSwitchOnlyForDedicatedEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const model = "gemini-3.1-flash-image"
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"draw"}]}],"generationConfig":{"responseModalities":["TEXT","IMAGE"]}}`)
	c, _ := newAntigravityCompatContext(http.MethodPost, "/v1beta/models/"+model+":generateContent", body)
	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"inlineData\":{\"mimeType\":\"image/png\",\"data\":\"aGVsbG8=\"}}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":10,\"candidatesTokenCount\":5}}}\n\n")),
	}}}
	svc := newAntigravityCompatService(config.GatewayConfig{MaxLineSize: defaultMaxLineSize}, upstream)
	account := newAntigravityCompatAccount(AccountTypeOAuth)
	account.Credentials["model_mapping"] = map[string]any{model: model}
	result, err := svc.ForwardGemini(context.Background(), c, account, model, "generateContent", true, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, "2K", result.ImageSize)

	groupID := int64(907)
	resolver := newOpenAITokenImageChannelPricingResolverForTest(t, groupID, model)
	billing := &GatewayService{billingService: resolver.billingService, resolver: resolver}
	group := &Group{ID: groupID, Platform: PlatformComposite, Hydrated: true, AllowImageGeneration: true, ImagePrice2K: f64p(0.4)}
	key := &APIKey{GroupID: &groupID, Group: group}
	before := billing.calculateRecordUsageCost(context.Background(), result, key, model, 1, 1, time.Time{})
	group.ImageCountBilling = true
	after := billing.calculateRecordUsageCost(context.Background(), result, key, model, 1, 1, time.Time{})
	require.Equal(t, string(BillingModeToken), after.BillingMode)
	require.Equal(t, before.TotalCost, after.TotalCost)
	require.Greater(t, after.TotalCost, 0.0)
	require.False(t, result.ImageEndpoint)

}
