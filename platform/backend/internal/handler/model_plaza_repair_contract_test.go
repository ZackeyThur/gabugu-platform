//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type plazaRepairGroups struct {
	service.GroupRepository
	groups []service.Group
}

func (r plazaRepairGroups) ListActive(context.Context) ([]service.Group, error) { return r.groups, nil }

func TestModelPlazaRepairContract_DisclosesTiersAndKeepsPrivateGroupsHidden(t *testing.T) {
	ptr := func(v float64) *float64 { return &v }
	groups := []service.Group{
		{ID: 4, Name: "GPT", Platform: service.PlatformOpenAI, RateMultiplier: 0.4, LongContextPricingEnabled: true, WebSearchPricePerCall: ptr(0.5), ModelPricing: []service.ChannelModelPricing{{Models: []string{"gpt-6.1-terra", "gpt-6.1-sol"}, BillingMode: service.BillingModeToken}}},
		{ID: 7, Name: "Video", Platform: service.PlatformGrok, RateMultiplier: 1, ModelPricing: []service.ChannelModelPricing{{Models: []string{"grok-imagine-video"}, BillingMode: service.BillingModeVideo, PerRequestPrice: ptr(0.05)}}},
		{ID: 21, Name: "private-only-group", Platform: service.PlatformOpenAI, IsExclusive: true, ModelPricing: []service.ChannelModelPricing{{Models: []string{"private-only-model"}, BillingMode: service.BillingModeToken, InputPrice: ptr(1e-6), OutputPrice: ptr(5e-6)}}},
	}
	billing := service.NewBillingService(&config.Config{}, nil)
	resolver := service.NewModelPricingResolver(nil, billing)
	svc := service.NewModelPlazaService(plazaVaryChannelRepo{}, plazaRepairGroups{groups: groups}, nil, billing, resolver)
	h := NewModelPlazaHandler(svc, nil, service.NewSettingService(plazaVarySettingRepo{}, &config.Config{}))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/model-plaza", h.Get)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/model-plaza", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "private-only")
	require.Contains(t, w.Body.String(), "service_tiers")
	require.Contains(t, w.Body.String(), "additional_charges")
	var body struct {
		Data struct {
			Groups []modelPlazaGroup `json:"groups"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Groups, 2)
	for _, g := range body.Data.Groups {
		for _, m := range g.Models {
			if m.Name == "gpt-6.1-terra" {
				require.InDelta(t, 1e-6, *m.Pricing.InputPrice, 1e-12)
				require.NotEmpty(t, m.ServiceTiers)
			}
		}
	}
	// Optional artifact for local browser acceptance; no network or live data.
	if path := os.Getenv("GABUGU_PRICING_UI_FIXTURE"); path != "" {
		require.NoError(t, os.WriteFile(path, w.Body.Bytes(), 0600))
	}
}
