package controller

import (
	"net/http"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/service"
	"github.com/gin-gonic/gin"
)

type rankingsAccessConfig struct {
	enabled     bool
	requireAuth bool
}

func getRankingsAccessConfig() rankingsAccessConfig {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap["HeaderNavModules"]
	common.OptionMapRWMutex.RUnlock()

	config := rankingsAccessConfig{enabled: true}
	if raw == "" {
		return config
	}

	var parsed map[string]interface{}
	if err := common.Unmarshal([]byte(raw), &parsed); err != nil {
		return config
	}
	rankings, ok := parsed["rankings"]
	if !ok {
		return config
	}
	switch v := rankings.(type) {
	case bool:
		config.enabled = v
	case map[string]interface{}:
		if enabled, ok := v["enabled"]; ok {
			if b, ok := enabled.(bool); ok {
				config.enabled = b
			}
		}
		if requireAuth, ok := v["requireAuth"]; ok {
			if b, ok := requireAuth.(bool); ok {
				config.requireAuth = b
			}
		}
	}
	return config
}

func isRankingsEnabled() bool {
	return getRankingsAccessConfig().enabled
}

func abortRankingsDisabled(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": "rankings is disabled",
	})
}

func enforceRankingsAccess(c *gin.Context) bool {
	config := getRankingsAccessConfig()
	if !config.enabled {
		abortRankingsDisabled(c)
		return false
	}
	if config.requireAuth && c.GetInt("id") <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "login required",
		})
		return false
	}
	return true
}

func GetRankings(c *gin.Context) {
	if !enforceRankingsAccess(c) {
		return
	}

	result, err := service.GetRankingsSnapshot(c.DefaultQuery("period", "week"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetUserRankings(c *gin.Context) {
	if !enforceRankingsAccess(c) {
		return
	}

	result, err := service.GetUserRankingsSnapshot(
		c.DefaultQuery("period", "week"),
		c.GetInt("id"),
		c.DefaultQuery("metric", "tokens"),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
