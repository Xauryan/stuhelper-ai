package controller

import (
	"strconv"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/model"

	"github.com/gin-gonic/gin"
)

func GetChannelMonitorSummary(c *gin.Context) {
	windowSeconds, _ := strconv.ParseInt(c.Query("window_seconds"), 10, 64)
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	errorLimit, _ := strconv.Atoi(c.Query("error_limit"))

	summary, err := model.GetChannelMonitorStats(model.ChannelMonitorStatsParams{
		WindowSeconds: windowSeconds,
		Source:        c.Query("source"),
		ChannelID:     channelID,
		ModelName:     c.Query("model_name"),
		Group:         c.Query("group"),
		ErrorLimit:    errorLimit,
		IncludeNames:  c.GetInt("role") >= common.RoleAdminUser,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}
