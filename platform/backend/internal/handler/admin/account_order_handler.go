package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// Reorder persists the admin account list's presentation order.
func (h *AccountHandler) Reorder(c *gin.Context) {
	var request struct {
		IDs []int64 `json:"ids" binding:"required,min=1,max=10000"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid account order")
		return
	}
	if err := h.adminService.ReorderAccounts(c.Request.Context(), request.IDs); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"success": true})
}
