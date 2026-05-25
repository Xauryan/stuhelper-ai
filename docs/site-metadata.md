# 站点标题与 SEO 元信息配置

StuHelper AI 的站点标题、首页副标题和 SEO/分享卡片元信息由后台
`系统设置 -> 其他设置 -> 个性化设置` 管理。相关配置项保存在
`options` 表，并通过 `/api/status` 下发给前端。

## 配置项

| Option Key | `/api/status` 字段 | 用途 | 默认值 |
| --- | --- | --- | --- |
| `SystemName` | `system_name` | 站点标题、浏览器标题、`application-name`、`og:title`、`twitter:title` | `StuHelper AI` |
| `SystemSubtitle` | `system_subtitle` | 默认首页主标题下方的站点副标题 | `统一的大模型 API 网关` |
| `SEODescription` | `seo_description` | `description`、`og:description`、`twitter:description` | `StuHelper AI 是 Xauryan 部署的统一 AI 模型聚合与分发网关，提供高性价比的集中式模型管理与网关服务。` |
| `SEOKeywords` | `seo_keywords` | `keywords` meta；可为空，多个关键词用英文逗号分隔 | 空 |
| `SEOImage` | `seo_image` | `og:image`、`twitter:image`；为空时使用 `Logo`，再为空时使用 `/logo.png` | 空 |
| `Logo` | `logo` | 页头 Logo、favicon，也作为 SEO 分享图兜底 | 空 |

## 生效路径

- 服务端在返回 classic `index.html` 时，会根据当前配置替换 `<title>`、
  description、keywords、Open Graph 和 Twitter Card meta。这样不执行
  JavaScript 的搜索引擎或社交分享抓取也能拿到配置后的内容。
- 前端加载 `/api/status` 后，会把相同字段写入 `localStorage`，并再次同步
  `document.title`、favicon 和 meta，保证客户端切换配置后即时生效。
- 如果 `SEOImage` 或 `Logo` 是以 `/` 开头的相对路径，并且配置了
  `ServerAddress`，服务端会在首屏 HTML 中转换为绝对地址，便于分享平台抓取。

## 维护要求

- 新增或重命名站点元信息字段时，要同时更新后端 `OptionMap` 默认值、
  `/api/status`、服务端 `index.html` 注入、前端 `applySiteMeta` 和本文件。
- 项目身份和代码仓库仍保持 `StuHelper AI` / `Xauryan`；这些配置只用于运行中
  的站点展示和分享元信息，不用于修改 Go module、仓库、镜像或源码归属。
