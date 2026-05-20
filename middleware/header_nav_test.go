package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func withHeaderNavModules(t *testing.T, raw string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	previous, hadPrevious := common.OptionMap["HeaderNavModules"]
	common.OptionMap["HeaderNavModules"] = raw
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadPrevious {
			common.OptionMap["HeaderNavModules"] = previous
			return
		}
		delete(common.OptionMap, "HeaderNavModules")
	})
}

func performHeaderNavRequest(handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessions.Sessions("session", cookie.NewStore([]byte("header-nav-test"))))
	router.GET("/api/test", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestHeaderNavModuleAuthAllowsDefaultPublicAccess(t *testing.T) {
	withHeaderNavModules(t, "")

	recorder := performHeaderNavRequest(HeaderNavModuleAuth("pricing"))

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
}

func TestHeaderNavModuleAuthRejectsDisabledPricing(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":{"enabled":false,"requireAuth":false}}`)

	recorder := performHeaderNavRequest(HeaderNavModuleAuth("pricing"))

	require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
}

func TestHeaderNavModuleAuthRequiresLoginForPricing(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":{"enabled":true,"requireAuth":true}}`)

	recorder := performHeaderNavRequest(HeaderNavModuleAuth("pricing"))

	require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
}

func TestHeaderNavModulePublicOrUserAuthAllowsDefaultPublicAccess(t *testing.T) {
	withHeaderNavModules(t, "")

	recorder := performHeaderNavRequest(HeaderNavModulePublicOrUserAuth("pricing"))

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
}

func TestHeaderNavModulePublicOrUserAuthRequiresLoginWhenDisabled(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":{"enabled":false,"requireAuth":false}}`)

	recorder := performHeaderNavRequest(HeaderNavModulePublicOrUserAuth("pricing"))

	require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
}

func TestHeaderNavModulePublicOrUserAuthRequiresLoginWhenRequireAuth(t *testing.T) {
	withHeaderNavModules(t, `{"pricing":{"enabled":true,"requireAuth":true}}`)

	recorder := performHeaderNavRequest(HeaderNavModulePublicOrUserAuth("pricing"))

	require.Equal(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
}
