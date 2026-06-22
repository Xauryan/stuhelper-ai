package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/Xauryan/stuhelper-ai/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func requireEmptyStringPtr(t *testing.T, value *string) {
	t.Helper()
	require.NotNil(t, value)
	require.Empty(t, *value)
}

func TestUpdateChannelAllowsExplicitEmptyModels(t *testing.T) {
	db := setupChannelAuditAdminControllerTestDB(t)
	require.NoError(t, db.Create(&model.Channel{
		Id:     7101,
		Type:   1,
		Key:    "sk-test",
		Name:   "clear-models-channel",
		Status: common.ChannelStatusEnabled,
		Models: "gpt-4o,gpt-4o-mini",
		Group:  "default",
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/channel/",
		strings.NewReader(`{"id":7101,"models":""}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateChannel(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])

	var channel model.Channel
	require.NoError(t, db.First(&channel, 7101).Error)
	require.Empty(t, channel.Models)
}

func TestUpdateChannelAllowsExplicitEmptyNullableTextFields(t *testing.T) {
	db := setupChannelAuditAdminControllerTestDB(t)
	baseURL := "https://api.example.com"
	openAIOrganization := "org-test"
	testModel := "gpt-4o"
	tag := "paid"
	remark := "internal remark"
	modelMapping := `{"gpt-4o":"gpt-4o-mini"}`
	statusCodeMapping := `{"429":"500"}`
	paramOverride := `{"temperature":0}`
	headerOverride := `{"X-Test":"1"}`
	require.NoError(t, db.Create(&model.Channel{
		Id:                 7102,
		Type:               1,
		Key:                "sk-test",
		Name:               "clear-text-channel",
		Status:             common.ChannelStatusEnabled,
		Models:             "gpt-4o",
		Group:              "default",
		BaseURL:            &baseURL,
		OpenAIOrganization: &openAIOrganization,
		TestModel:          &testModel,
		Tag:                &tag,
		Remark:             &remark,
		ModelMapping:       &modelMapping,
		StatusCodeMapping:  &statusCodeMapping,
		ParamOverride:      &paramOverride,
		HeaderOverride:     &headerOverride,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/channel/",
		strings.NewReader(`{
			"id":7102,
			"models":"gpt-4o",
			"base_url":"",
			"openai_organization":"",
			"test_model":"",
			"tag":"",
			"remark":"",
			"model_mapping":"",
			"status_code_mapping":"",
			"param_override":"",
			"header_override":""
		}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	UpdateChannel(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])

	var channel model.Channel
	require.NoError(t, db.First(&channel, 7102).Error)
	requireEmptyStringPtr(t, channel.BaseURL)
	requireEmptyStringPtr(t, channel.OpenAIOrganization)
	requireEmptyStringPtr(t, channel.TestModel)
	requireEmptyStringPtr(t, channel.Tag)
	requireEmptyStringPtr(t, channel.Remark)
	requireEmptyStringPtr(t, channel.ModelMapping)
	requireEmptyStringPtr(t, channel.StatusCodeMapping)
	requireEmptyStringPtr(t, channel.ParamOverride)
	requireEmptyStringPtr(t, channel.HeaderOverride)
}

func TestResetChannelBreakerClearsRuntimeStateOnly(t *testing.T) {
	db := setupChannelAuditAdminControllerTestDB(t)
	t.Cleanup(func() {
		service.ResetChannelBreaker(7201)
		service.InitChannelBreakerConfig()
	})
	t.Setenv("CHANNEL_BREAKER_ENABLED", "true")
	t.Setenv("CHANNEL_BREAKER_CONSECUTIVE_FATAL", "3")
	t.Setenv("CHANNEL_BREAKER_MIN_SAMPLES", "10")
	service.InitChannelBreakerConfig()
	require.NoError(t, db.Create(&model.User{
		Id:          123,
		Username:    "breaker-reset-admin",
		Password:    "password",
		DisplayName: "Breaker Reset Admin",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Id:     7201,
		Type:   1,
		Key:    "sk-test",
		Name:   "breaker-reset-channel",
		Status: common.ChannelStatusManuallyDisabled,
		Models: "gpt-4o",
		Group:  "default",
	}).Error)

	channelErr := types.NewErrorWithStatusCode(
		errors.New("invalid api key"),
		types.ErrorCodeChannelInvalidKey,
		http.StatusUnauthorized,
	)
	for i := 0; i < 20; i++ {
		service.ReportRelayResult(7201, channelErr)
	}
	require.Equal(t, "open", service.BreakerStateName(7201))
	require.Greater(t, service.ChannelAvailabilitySnapshot(7201).Total, int64(0))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/7201/breaker/reset", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "7201"}}
	ctx.Set("id", 123)
	ctx.Set("username", "breaker-reset-admin")
	ctx.Set("role", common.RoleAdminUser)

	ResetChannelBreaker(ctx)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			ID            int                                `json:"id"`
			BreakerState  string                             `json:"breaker_state"`
			PreviousState string                             `json:"previous_state"`
			Availability  *types.ChannelAvailabilitySnapshot `json:"availability"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 7201, response.Data.ID)
	require.Equal(t, "open", response.Data.PreviousState)
	require.Equal(t, "closed", response.Data.BreakerState)
	require.NotNil(t, response.Data.Availability)
	require.Equal(t, int64(0), response.Data.Availability.Total)

	var channel model.Channel
	require.NoError(t, db.First(&channel, 7201).Error)
	require.Equal(t, common.ChannelStatusManuallyDisabled, channel.Status)
	require.Equal(t, "closed", service.BreakerStateName(7201))
	require.Equal(t, int64(0), service.ChannelAvailabilitySnapshot(7201).Total)
}
