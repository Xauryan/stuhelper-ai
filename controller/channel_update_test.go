package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"
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
