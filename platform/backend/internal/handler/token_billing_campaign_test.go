package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type campaignPublicSettingsRepo struct {
	service.SettingRepository
	raw string
}

func (r *campaignPublicSettingsRepo) GetValue(context.Context, string) (string, error) {
	if r.raw == "" {
		return "", service.ErrSettingNotFound
	}
	return r.raw, nil
}

func TestBillingCampaignPublicHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, raw := range []string{"", `{"schema_version":999,"secret":"never-return-this"}`} {
		router := gin.New()
		h := NewSettingHandler(service.NewSettingService(&campaignPublicSettingsRepo{raw: raw}, nil), "")
		router.GET("/api/v1/billing-campaign", h.GetBillingCampaign)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/billing-campaign", nil))
		require.Equal(t, http.StatusOK, response.Code)
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		require.NotContains(t, response.Body.String(), "never-return-this")
		var result struct {
			Data service.PublicBillingCampaignResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		require.Nil(t, result.Data.Campaign)
		require.WithinDuration(t, time.Now(), result.Data.ServerNow, time.Minute)
		if raw == "" {
			require.Equal(t, "disabled", result.Data.Status)
		} else {
			require.Equal(t, "invalid", result.Data.Status)
		}
	}
}
