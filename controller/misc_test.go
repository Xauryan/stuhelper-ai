package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetStatusIncludesPasswordAuthSwitches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalLoginEnabled := common.PasswordLoginEnabled
	originalRegisterEnabled := common.PasswordRegisterEnabled
	common.PasswordLoginEnabled = false
	common.PasswordRegisterEnabled = false
	t.Cleanup(func() {
		common.PasswordLoginEnabled = originalLoginEnabled
		common.PasswordRegisterEnabled = originalRegisterEnabled
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)

	GetStatus(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, false, response.Data["password_login"])
	require.Equal(t, false, response.Data["password_register"])
}
