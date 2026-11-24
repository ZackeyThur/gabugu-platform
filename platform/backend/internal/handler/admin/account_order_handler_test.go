package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type reorderAccountAdminStub struct {
	*stubAdminService
	ids []int64
	err error
}

func (s *reorderAccountAdminStub) ReorderAccounts(_ context.Context, ids []int64) error {
	if err := service.ValidateAccountOrderIDs(ids); err != nil {
		return err
	}
	s.ids = append([]int64(nil), ids...)
	return s.err
}

func TestAccountHandlerReorder(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		err        error
		status     int
	}{
		{name: "success", body: `{"ids":[8,3,5]}`, status: http.StatusOK},
		{name: "empty", body: `{"ids":[]}`, status: http.StatusBadRequest},
		{name: "missing", body: `{}`, status: http.StatusBadRequest},
		{name: "duplicate", body: `{"ids":[3,3]}`, status: http.StatusBadRequest},
		{name: "nonpositive", body: `{"ids":[0]}`, status: http.StatusBadRequest},
		{name: "negative", body: `{"ids":[-1]}`, status: http.StatusBadRequest},
		{name: "fraction", body: `{"ids":[1.5]}`, status: http.StatusBadRequest},
		{name: "overflow", body: `{"ids":[9223372036854775808]}`, status: http.StatusBadRequest},
		{name: "too_many", body: `{"ids":[` + strings.Repeat("1,", 10000) + `1]}`, status: http.StatusBadRequest},
		{name: "deleted", body: `{"ids":[8,3,5]}`, err: service.ErrAccountOrderConflict, status: http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			svc := &reorderAccountAdminStub{stubAdminService: newStubAdminService(), err: tc.err}
			h := NewAccountHandler(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			router := gin.New()
			router.PUT("/api/v1/admin/accounts/reorder", h.Reorder)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/reorder", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			if tc.status == http.StatusOK || tc.status == http.StatusConflict {
				require.Equal(t, []int64{8, 3, 5}, svc.ids)
			}
		})
	}
}
