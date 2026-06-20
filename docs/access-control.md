# 访问限制策略

StuHelper AI 支持在服务端按请求来源国家/地区和访问身份拦截官网 Web 与 API。
所有策略默认关闭，必须由超级管理员在 classic 后台
`系统设置 -> 访问限制` 中显式启用。

## 配置项

| Option Key | 用途 | 默认值 |
| --- | --- | --- |
| `access_control.web_policy_enabled` | 是否对官网 Web 路由启用访问限制。包含内置 classic 前端资源、首页兜底和 `FRONTEND_BASE_URL` 重定向。 | `false` |
| `access_control.api_policy_enabled` | 是否对 API 路由启用访问限制。包含管理后台 `/api/*`、relay `/v1`/`/v1beta`/`/mj`/`/suno` 等 token API，以及旧 dashboard API。 | `false` |
| `access_control.block_china_mainland` | 禁止识别为中国大陆的 IP 访问，国家代码为 `CN`。这是全局地域封禁，会影响游客、普通用户和管理员。 | `false` |
| `access_control.block_european_union` | 禁止识别为欧盟成员国的 IP 访问。 | `false` |
| `access_control.block_china_mainland_homepage` | 仅禁止中国大陆 IP 访问官网主页 `/`，审计管理员、管理员和超级管理员放行。 | `false` |
| `access_control.block_china_mainland_user_sensitive_pages` | 禁止中国大陆普通用户访问令牌、钱包和账单相关 Web 页面与 API，审计管理员、管理员和超级管理员放行。 | `false` |
| `access_control.block_guests` | 禁止游客访问。对 Web 表示无登录 session；对 API 表示无可识别认证凭据的请求。 | `false` |
| `access_control.block_users` | 禁止普通用户访问。API token 请求按 token 所属用户角色判断。 | `false` |
| `access_control.block_admins` | 禁止管理员访问。包含审计管理员、管理员和超级管理员。 | `false` |
| `access_control.geoip_database_path` | 本地 MaxMind 兼容 MMDB 国家库路径。留空时只使用代理注入的国家代码请求头。 | 空 |

## 国家/地区识别

识别顺序：

1. 优先读取可信反向代理或 CDN 注入的国家代码请求头：
   `EO-Client-IPCountry`、`CF-IPCountry`、`CloudFront-Viewer-Country`、
   `X-Vercel-IP-Country`、`X-Country-Code`、`X-Geo-Country`。
2. 如果没有可用请求头，并且配置了 `access_control.geoip_database_path`，后端会用
   `gin.Context.ClientIP()` 得到客户端 IP，再查询本地 MMDB 文件。
3. 如果无法识别国家代码，地域策略不拦截该请求，只继续执行身份策略。

腾讯云 EdgeOne 推荐在 EO 控制台开启“客户端 IP 地理位置头部”回源，或通过规则引擎修改
HTTP 回源请求头，把客户端 IP 所在国家/地区写入请求头。头部名称推荐使用
`EO-Client-IPCountry`；如果现有规则已经使用自定义头，也可使用本项目已识别的
`X-Country-Code`。头部值应为 ISO 3166-1 alpha-2 两位国家/地区代码，例如中国大陆为
`CN`。腾讯云文档说明 EdgeOne 的客户端 IP 地理位置头部按国家/地区维度显示，值采用
ISO 3166-1 alpha-2；如需自定义回源头，可参考腾讯云 EdgeOne 的
[携带客户端 IP 地理位置头部回源](https://cloud.tencent.com/document/product/1552/80978)
和 [修改 HTTP 回源请求头](https://cloud.tencent.com/document/product/1552/71012) 文档。

欧盟成员国代码按 27 个 EU 成员国维护：
`AT`、`BE`、`BG`、`HR`、`CY`、`CZ`、`DK`、`EE`、`FI`、`FR`、`DE`、`GR`、`HU`、
`IE`、`IT`、`LV`、`LT`、`LU`、`MT`、`NL`、`PL`、`PT`、`RO`、`SK`、`SI`、`ES`、`SE`。

## 中国大陆细粒度策略

如果需求是“官网主页封禁中国大陆 IP，已登录用户仍可访问日志等普通页面，但普通用户不能访问
令牌管理、钱包管理、账单管理，管理员不受影响”，应启用：

- `access_control.web_policy_enabled`
- `access_control.api_policy_enabled`
- `access_control.block_china_mainland_homepage`
- `access_control.block_china_mainland_user_sensitive_pages`

不要为该需求启用 `access_control.block_china_mainland`。该旧开关是全局地域封禁，会先于细粒度
策略生效，并且会影响管理员。

细粒度策略的 Web 受限路径：

- `/`
- `/console/token`
- `/console/topup`
- `/console/billing`

细粒度策略的普通用户 API 受限路径：

- `/api/token` 及其子路径
- `/api/subscription/self` 及其子路径
- `/api/subscription/*/pay`
- `/api/user/topup` 及其子路径
- `/api/user/pay`
- `/api/user/amount`
- `/api/user/stripe/*`
- `/api/user/creem/*`
- `/api/user/waffo/*`
- `/api/user/alipay/official/*`
- `/api/user/wechat-pay/official/*`
- `/api/user/self-serve/*`
- `/api/user/aff`
- `/api/user/aff/commissions`
- `/api/user/aff_transfer`

细粒度策略不会限制 `/console/log`、`/api/log/self` 等日志页面和日志接口。支付平台回调接口
例如 `/api/subscription/epay/notify` 也不会因为该策略被拦截。

`/api/status` 会返回当前请求的 `access_control` 识别结果，classic 前端据此隐藏受限菜单项并在
客户端路由显示 403；真正的访问控制仍在服务端中间件执行。

## 生效路径

- Web 策略挂在 classic Web 路由层，影响静态资源、favicon、首页兜底页面和外置前端
  重定向路径。
- API 策略分两段执行：
  - `/api/*`、relay、视频和旧 dashboard API 路由会先执行地域限制和游客限制。
    前置游客判断会识别 `Authorization`、`api-key`、`mj-api-secret`、`x-api-key`、
    `x-goog-api-key`、WebSocket `Sec-WebSocket-Protocol` 中的 OpenAI realtime key，以及
    `key` query 参数；带这些凭据的请求会延后到认证完成后再按真实角色判断。
  - `UserAuth`、`TokenAuth`、`TokenAuthReadOnly` 等认证完成后，会再次按真实用户角色执行
    用户/管理员限制。
- Relay API 的拒绝响应保持 OpenAI 风格错误体；管理后台 API 和 Web 返回
  `{ "success": false, "message": "访问受限" }`。

## 运维注意事项

- 如果站点在 CDN 或反向代理后运行，必须确保 `TrustedProxies` / 部署层真实客户端 IP
  配置正确，否则 MMDB 查询可能拿到代理 IP。
- `CF-IPCountry` 等请求头只有在入口代理可信时才应使用。不要允许外部客户端绕过代理直连
  后端并伪造这些请求头。
- 启用 `block_users` 或 `block_admins` 前，应确认仍有不受该策略影响的管理入口，例如临时
  保留 Web 策略关闭、API 策略关闭，或通过部署层临时改回 options 表。
- 使用腾讯云 EdgeOne 时，优先使用 EO 注入的国家/地区请求头，应用本身不需要实时下载 IP
  数据库，也不应该在每次请求时联网查询 IP 归属地。
- 如果没有可信代理头，可使用本地 MaxMind 兼容 MMDB 国家库作为兜底，例如 GeoLite2 Country
  或 GeoIP2 Country。MaxMind GeoLite2 免费库需要 MaxMind 账号和 license key；MaxMind 推荐用
  GeoIP Update 程序自动更新数据库，也可以直接下载但不推荐作为常规方式。参考 MaxMind 的
  [GeoLite2 免费库](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data/) 和
  [更新数据库](https://dev.maxmind.com/geoip/updating-databases/) 文档。
- MMDB 文件不会写入数据库；只保存服务端本地路径。替换文件后，路径不变时需要重启服务才
  能强制重新打开 reader；修改路径会触发热加载。建议用 `geoipupdate` 定时更新本地文件，
  例如每周两次；更新同一路径文件后安排应用重启。
