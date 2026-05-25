package router

import (
	"testing"

	"github.com/Xauryan/stuhelper-ai/common"
	"github.com/Xauryan/stuhelper-ai/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func TestRenderClassicIndexPageInjectsConfiguredSiteMeta(t *testing.T) {
	originalSystemName := common.SystemName
	originalSystemSubtitle := common.SystemSubtitle
	originalSEODescription := common.SEODescription
	originalSEOKeywords := common.SEOKeywords
	originalSEOImage := common.SEOImage
	originalLogo := common.Logo
	originalServerAddress := system_setting.ServerAddress
	t.Cleanup(func() {
		common.SystemName = originalSystemName
		common.SystemSubtitle = originalSystemSubtitle
		common.SEODescription = originalSEODescription
		common.SEOKeywords = originalSEOKeywords
		common.SEOImage = originalSEOImage
		common.Logo = originalLogo
		system_setting.ServerAddress = originalServerAddress
	})

	common.SystemName = `Xauryan "Gateway"`
	common.SystemSubtitle = "Configured subtitle"
	common.SEODescription = `AI gateway & billing`
	common.SEOKeywords = "ai,gateway,billing"
	common.SEOImage = ""
	common.Logo = "/custom-logo.png"
	system_setting.ServerAddress = "https://example.com"

	index := []byte(`<!doctype html>
<html>
  <head>
    <meta name="application-name" content="StuHelper AI" />
    <meta name="apple-mobile-web-app-title" content="StuHelper AI" />
    <meta
      name="description"
      content="old description"
    />
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="StuHelper AI" />
    <meta property="og:title" content="StuHelper AI" />
    <meta property="og:description" content="old description" />
    <meta property="og:image" content="/logo.png" />
    <meta name="twitter:title" content="StuHelper AI" />
    <meta name="twitter:description" content="old description" />
    <meta name="twitter:image" content="/logo.png" />
    <title>StuHelper AI</title>
  </head>
</html>`)

	rendered := string(renderClassicIndexPage(index))

	require.Contains(t, rendered, `<title>Xauryan &#34;Gateway&#34;</title>`)
	require.Contains(t, rendered, `name="application-name" content="Xauryan &#34;Gateway&#34;"`)
	require.Contains(t, rendered, `name="description"`+"\n"+`      content="AI gateway &amp; billing"`)
	require.Contains(t, rendered, `<meta name="keywords" content="ai,gateway,billing" />`)
	require.Contains(t, rendered, `property="og:title" content="Xauryan &#34;Gateway&#34;"`)
	require.Contains(t, rendered, `property="og:description" content="AI gateway &amp; billing"`)
	require.Contains(t, rendered, `property="og:image" content="https://example.com/custom-logo.png"`)
	require.Contains(t, rendered, `name="twitter:image" content="https://example.com/custom-logo.png"`)
}
