# 访问限制策略

StuHelper AI 支持在服务端按请求来源国家/地区和访问身份拦截官网 Web 与 API。
所有策略默认关闭，必须由超级管理员在 classic 后台
`系统设置 -> 访问限制` 中显式启用。

## 配置项

| Option Key | 用途 | 默认值 |
| --- | --- | --- |
| `access_control.web_policy_enabled` | 是否对官网 Web 路由启用访问限制。包含内置 classic 前端资源、首页兜底和 `FRONTEND_BASE_URL` 重定向。 | `false` |
| `access_control.api_policy_enabled` | 是否对 API 路由启用访问限制。包含管理后台 `/api/*`、relay `/v1`/`/v1beta`/`/mj`/`/suno` 等 token API，以及旧 dashboard API。 | `false` |
| `access_control.block_china_mainland` | 禁止识别为中国大陆的 IP 访问，国家代码为 `CN`。 | `false` |
| `access_control.block_european_union` | 禁止识别为欧盟成员国的 IP 访问。 | `false` |
| `access_control.block_guests` | 禁止游客访问。对 Web 表示无登录 session；对 API 表示无可识别认证凭据的请求。 | `false` |
| `access_control.block_users` | 禁止普通用户访问。API token 请求按 token 所属用户角色判断。 | `false` |
| `access_control.block_admins` | 禁止管理员访问。包含审计管理员、管理员和超级管理员。 | `false` |
| `access_control.geoip_database_path` | 本地 MaxMind 兼容 MMDB 国家库路径。留空时只使用代理注入的国家代码请求头。 | 空 |

## 国家/地区识别

识别顺序：

1. 优先读取可信反向代理或 CDN 注入的国家代码请求头：
   `CF-IPCountry`、`CloudFront-Viewer-Country`、`X-Vercel-IP-Country`、
   `X-Country-Code`、`X-Geo-Country`。
2. 如果没有可用请求头，并且配置了 `access_control.geoip_database_path`，后端会用
   `gin.Context.ClientIP()` 得到客户端 IP，再查询本地 MMDB 文件。
3. 如果无法识别国家代码，地域策略不拦截该请求，只继续执行身份策略。

欧盟成员国代码按 27 个 EU 成员国维护：
`AT`、`BE`、`BG`、`HR`、`CY`、`CZ`、`DK`、`EE`、`FI`、`FR`、`DE`、`GR`、`HU`、
`IE`、`IT`、`LV`、`LT`、`LU`、`MT`、`NL`、`PL`、`PT`、`RO`、`SK`、`SI`、`ES`、`SE`。

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
- MMDB 文件不会写入数据库；只保存服务端本地路径。替换文件后，路径不变时需要重启服务才
  能强制重新打开 reader；修改路径会触发热加载。
