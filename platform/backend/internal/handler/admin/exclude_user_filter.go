package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

var errExcludeSelfUnauthorized = errors.New("unauthorized for exclude_self")

// parseExcludeUserFilter 解析管理端使用记录/运维查询中的排除用户参数：
//   - exclude_user_id: 显式排除指定正整数用户 ID（优先级更高）
//   - exclude_self:    布尔开关，为 true 时从认证上下文取当前管理员 UserID
//
// 当显式指定 exclude_self=true 但认证上下文缺失有效 UserID 时返回 errExcludeSelfUnauthorized，
// 防止退化为静默失效或误排除其他账号。
func parseExcludeUserFilter(c *gin.Context) (int64, error) {
	var explicitID int64
	if rawID := strings.TrimSpace(c.Query("exclude_user_id")); rawID != "" {
		parsed, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, errors.New("invalid exclude_user_id")
		}
		explicitID = parsed
	}

	var excludeSelf bool
	if rawSelf := strings.TrimSpace(c.Query("exclude_self")); rawSelf != "" {
		parsed, err := strconv.ParseBool(rawSelf)
		if err != nil {
			return 0, errors.New("invalid exclude_self value, use true or false")
		}
		excludeSelf = parsed
	}

	if explicitID > 0 {
		return explicitID, nil
	}
	if excludeSelf {
		subject, ok := middleware.GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			return 0, errExcludeSelfUnauthorized
		}
		return subject.UserID, nil
	}
	return 0, nil
}

func respondExcludeUserFilterError(c *gin.Context, err error) {
	if errors.Is(err, errExcludeSelfUnauthorized) {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}
	response.BadRequest(c, err.Error())
}
