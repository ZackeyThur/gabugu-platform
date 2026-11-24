package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterArenaLeaderboardRoutes registers the public Arena model leaderboard routes.
//
// PublicIP rate-limiting protects against scraping bursts. BackendModeUserGuard keeps
// the public surface consistent with Model Plaza when backend-only mode is active.
func RegisterArenaLeaderboardRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	optionalJWT middleware.OptionalJWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h == nil || h.ArenaLeaderboard == nil {
		return
	}
	arena := v1.Group("/arena")
	arena.Use(panelRateLimiter.PublicIP())
	arena.Use(gin.HandlerFunc(optionalJWT))
	arena.Use(middleware.BackendModeUserGuard(settingService))
	{
		arena.GET("/leaderboards", h.ArenaLeaderboard.Get)
	}
}
