package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ArenaLeaderboardHandler serves the public Arena model leaderboard snapshot and admin manual sync.
type ArenaLeaderboardHandler struct {
	arenaService *service.ArenaLeaderboardService
}

// NewArenaLeaderboardHandler creates a new ArenaLeaderboardHandler.
func NewArenaLeaderboardHandler(arenaService *service.ArenaLeaderboardService) *ArenaLeaderboardHandler {
	return &ArenaLeaderboardHandler{
		arenaService: arenaService,
	}
}

// Get returns the cached Arena leaderboard snapshot.
// GET /api/v1/arena/leaderboards?category=text|code|vision|image
func (h *ArenaLeaderboardHandler) Get(c *gin.Context) {
	if h == nil || h.arenaService == nil {
		response.NotFound(c, "Arena leaderboard is not enabled")
		return
	}

	category := c.Query("category")
	snapshot, err := h.arenaService.GetSnapshot(category)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArenaDisabled):
			response.NotFound(c, "Arena leaderboard is not enabled")
		case errors.Is(err, service.ErrArenaCategoryInvalid):
			response.BadRequest(c, "Invalid leaderboard category")
		case errors.Is(err, service.ErrArenaNotReady):
			response.Error(c, http.StatusServiceUnavailable, "Arena leaderboard is synchronizing, please retry shortly")
		default:
			response.InternalError(c, "Failed to load arena leaderboard")
		}
		return
	}

	c.Header("Cache-Control", "public, max-age=300")
	response.Success(c, snapshot)
}

// AdminSync triggers an immediate upstream sync for all Arena leaderboard categories.
// POST /api/v1/admin/arena/sync
func (h *ArenaLeaderboardHandler) AdminSync(c *gin.Context) {
	if h == nil || h.arenaService == nil || !h.arenaService.Enabled() {
		response.NotFound(c, "Arena leaderboard is not enabled")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Minute)
	defer cancel()

	res, err := h.arenaService.SyncNow(ctx)
	if err != nil {
		if res != nil {
			response.ErrorWithDetails(c, http.StatusBadGateway, "Failed to synchronize arena leaderboard", "ARENA_SYNC_FAILED", map[string]string{
				"status": res.Status,
			})
			return
		}
		response.Error(c, http.StatusBadGateway, "Failed to synchronize arena leaderboard")
		return
	}

	response.Success(c, res)
}
