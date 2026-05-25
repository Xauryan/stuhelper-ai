package router

import (
	"embed"
	"html"
	"net/http"
	"regexp"
	"strings"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/controller"
	"github.com/Xauryan/stuhelper-ai/middleware"
	"github.com/Xauryan/stuhelper-ai/setting/system_setting"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// ThemeAssets holds the embedded classic frontend assets.
type ThemeAssets struct {
	ClassicBuildFS   embed.FS
	ClassicIndexPage []byte
}

type siteMeta struct {
	Title       string
	Description string
	Keywords    string
	Image       string
}

var titleTagPattern = regexp.MustCompile(`(?is)<title>.*?</title>`)

func currentSiteMeta() siteMeta {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	title := strings.TrimSpace(common.SystemName)
	if title == "" {
		title = "StuHelper AI"
	}

	image := strings.TrimSpace(common.SEOImage)
	if image == "" {
		image = strings.TrimSpace(common.Logo)
	}
	if image == "" {
		image = "/logo.png"
	}

	return siteMeta{
		Title:       title,
		Description: common.SEODescription,
		Keywords:    common.SEOKeywords,
		Image:       publicAssetURL(image),
	}
}

func publicAssetURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") {
		return value
	}
	if strings.HasPrefix(value, "/") && system_setting.ServerAddress != "" {
		return strings.TrimRight(system_setting.ServerAddress, "/") + value
	}
	return value
}

func replaceMetaContent(page string, attr string, content string) string {
	pattern := regexp.MustCompile(`(?is)<meta\s+` + regexp.QuoteMeta(attr) + `\s+content="[^"]*"\s*/?>`)
	escaped := html.EscapeString(content)
	return pattern.ReplaceAllStringFunc(page, func(match string) string {
		start := strings.Index(match, `content="`)
		if start < 0 {
			return match
		}
		start += len(`content="`)
		end := strings.Index(match[start:], `"`)
		if end < 0 {
			return match
		}
		return match[:start] + escaped + match[start+end:]
	})
}

func replaceTitle(page string, title string) string {
	escaped := html.EscapeString(title)
	if titleTagPattern.MatchString(page) {
		return titleTagPattern.ReplaceAllStringFunc(page, func(string) string {
			return "<title>" + escaped + "</title>"
		})
	}
	return strings.Replace(page, "</head>", "<title>"+escaped+"</title>\n</head>", 1)
}

func upsertKeywordsMeta(page string, keywords string) string {
	if strings.Contains(page, `name="keywords"`) {
		return replaceMetaContent(page, `name="keywords"`, keywords)
	}
	if strings.TrimSpace(keywords) == "" {
		return page
	}
	meta := `    <meta name="keywords" content="` + html.EscapeString(keywords) + `" />` + "\n"
	marker := `    <meta property="og:type" content="website" />`
	if strings.Contains(page, marker) {
		return strings.Replace(page, marker, meta+marker, 1)
	}
	return strings.Replace(page, "</head>", meta+"</head>", 1)
}

func renderClassicIndexPage(indexPage []byte) []byte {
	page := string(indexPage)
	meta := currentSiteMeta()

	page = replaceMetaContent(page, `name="application-name"`, meta.Title)
	page = replaceMetaContent(page, `name="apple-mobile-web-app-title"`, meta.Title)
	page = replaceMetaContent(page, `name="description"`, meta.Description)
	page = upsertKeywordsMeta(page, meta.Keywords)
	page = replaceMetaContent(page, `property="og:site_name"`, meta.Title)
	page = replaceMetaContent(page, `property="og:title"`, meta.Title)
	page = replaceMetaContent(page, `property="og:description"`, meta.Description)
	page = replaceMetaContent(page, `property="og:image"`, meta.Image)
	page = replaceMetaContent(page, `name="twitter:title"`, meta.Title)
	page = replaceMetaContent(page, `name="twitter:description"`, meta.Description)
	page = replaceMetaContent(page, `name="twitter:image"`, meta.Image)
	page = replaceTitle(page, meta.Title)

	return []byte(page)
}

func SetWebRouter(router *gin.Engine, assets ThemeAssets) {
	classicFS := common.EmbedFolder(assets.ClassicBuildFS, "web/classic/dist")

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", classicFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", renderClassicIndexPage(assets.ClassicIndexPage))
	})
}
