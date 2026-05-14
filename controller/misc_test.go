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

func TestGetStatusIncludesFooterTemplateSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalOptionMap := common.OptionMap
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"FooterTemplateCopyrightYear":        "2025-2026",
		"FooterTemplateCopyrightOwner":       "StuHelper AI.",
		"FooterTemplateIcpBeianNumber":       "京ICP备2025154912号",
		"FooterTemplateIcpBeianUrl":          "https://beian.miit.gov.cn/",
		"FooterTemplateTelecomLicenseNumber": "京B2-20253869",
		"FooterTemplateTelecomLicenseUrl":    "https://tsm.miit.gov.cn/",
		"FooterTemplateTelecomLicenseTypes":  "ICP,EDI",
	}
	common.OptionMapRWMutex.Unlock()

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
	require.Equal(t, "2025-2026", response.Data["footer_template_copyright_year"])
	require.Equal(t, "StuHelper AI.", response.Data["footer_template_copyright_owner"])
	require.Equal(t, "京ICP备2025154912号", response.Data["footer_template_icp_beian_number"])
	require.Equal(t, "https://beian.miit.gov.cn/", response.Data["footer_template_icp_beian_url"])
	require.Equal(t, "京B2-20253869", response.Data["footer_template_telecom_license_number"])
	require.Equal(t, "https://tsm.miit.gov.cn/", response.Data["footer_template_telecom_license_url"])
	require.Equal(t, "ICP,EDI", response.Data["footer_template_telecom_license_types"])
}
