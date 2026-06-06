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
	Icon        string
}

var titleTagPattern = regexp.MustCompile(`(?is)<title>.*?</title>`)
var linkIconPattern = regexp.MustCompile(`(?is)<link\b[^>]*\brel="[^"]*\bicon\b[^"]*"[^>]*>`)
var hrefAttrPattern = regexp.MustCompile(`(?is)\bhref="[^"]*"`)

func currentSiteMeta() siteMeta {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	title := strings.TrimSpace(common.SystemName)
	if title == "" {
		title = "StuHelper AI"
	}

	icon := strings.TrimSpace(common.Logo)
	if icon == "" || icon == "/favicon.ico" {
		icon = "/logo.png"
	}

	image := strings.TrimSpace(common.SEOImage)
	if image == "" {
		image = icon
	}

	return siteMeta{
		Title:       title,
		Description: common.SEODescription,
		Keywords:    common.SEOKeywords,
		Image:       publicAssetURL(image),
		Icon:        publicAssetURL(icon),
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

func replaceIconLink(page string, href string) string {
	if strings.TrimSpace(href) == "" {
		return page
	}
	escaped := html.EscapeString(href)
	if linkIconPattern.MatchString(page) {
		return linkIconPattern.ReplaceAllStringFunc(page, func(match string) string {
			if hrefAttrPattern.MatchString(match) {
				return hrefAttrPattern.ReplaceAllString(match, `href="`+escaped+`"`)
			}
			end := strings.LastIndex(match, ">")
			if end < 0 {
				return match
			}
			return match[:end] + ` href="` + escaped + `"` + match[end:]
		})
	}

	link := `    <link rel="icon" href="` + escaped + `" />` + "\n"
	marker := `    <meta name="viewport"`
	if strings.Contains(page, marker) {
		return strings.Replace(page, marker, link+marker, 1)
	}
	return strings.Replace(page, "</head>", link+"</head>", 1)
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

	page = replaceIconLink(page, meta.Icon)
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

func serveConfiguredFavicon(c *gin.Context) {
	icon := currentSiteMeta().Icon
	if icon == "" {
		icon = "/logo.png"
	}
	c.Header("Cache-Control", "no-cache")
	if strings.HasPrefix(strings.ToLower(icon), "data:") {
		c.Status(http.StatusNoContent)
		return
	}
	c.Redirect(http.StatusFound, icon)
}

func SetWebRouter(router *gin.Engine, assets ThemeAssets) {
	classicFS := common.EmbedFolder(assets.ClassicBuildFS, "web/classic/dist")

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.GET("/favicon.ico", serveConfiguredFavicon)
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
