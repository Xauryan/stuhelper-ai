# 访问限制策略

StuHelper AI 支持在服务端按请求来源国家/地区、访问身份和资源层级拦截官网 Web 与 API。
所有策略默认关闭，必须由超级管理员在 classic 后台
`系统设置 -> 访问限制` 中显式启用。

## 配置项

| Option Key                                                 | 用途                                                                                                                                                                                                          | 默认值  |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `access_control.web_policy_enabled`                        | 是否对官网 Web 路由启用访问限制。包含内置 classic 前端资源、首页兜底和 `FRONTEND_BASE_URL` 重定向。                                                                                                           | `false` |
| `access_control.api_policy_enabled`                        | 是否对 API 路由启用访问限制中间件。该总开关会覆盖管理后台 `/api/*`、relay `/v1`/`/v1beta`/`/mj`/`/suno` 等 token API，以及旧 dashboard API；资源级 `model_api` 只匹配大模型 relay 服务，不等于全部 `/api/*`。 | `false` |
| `access_control.block_china_mainland`                      | 禁止识别为中国大陆的 IP 访问，国家代码为 `CN`。这是全局地域封禁，会影响游客、普通用户和管理员。                                                                                                               | `false` |
| `access_control.block_european_union`                      | 禁止识别为欧盟成员国的 IP 访问。                                                                                                                                                                              | `false` |
| `access_control.block_china_mainland_homepage`             | 仅禁止中国大陆 IP 访问官网主页 `/`，审计管理员、管理员和超级管理员放行。                                                                                                                                      | `false` |
| `access_control.block_china_mainland_user_sensitive_pages` | 禁止中国大陆普通用户访问令牌、钱包和账单相关 Web 页面与 API，审计管理员、管理员和超级管理员放行。                                                                                                             | `false` |
| `access_control.block_guests`                              | 禁止游客访问。对 Web 表示无登录 session；对 API 表示无可识别认证凭据的请求。                                                                                                                                  | `false` |
| `access_control.block_users`                               | 禁止普通用户访问。API token 请求按 token 所属用户角色判断。                                                                                                                                                   | `false` |
| `access_control.block_admins`                              | 禁止管理员访问。包含审计管理员、管理员和超级管理员。                                                                                                                                                          | `false` |
| `access_control.geoip_database_path`                       | 本地 MaxMind 兼容 MMDB 国家库路径。留空时只使用代理注入的国家代码请求头。                                                                                                                                     | 空      |
| `access_control.resource_rules`                            | 资源级访问矩阵。按资源 key 配置 `guest`、`user`、`audit_admin`、`admin`、`root` 五类身份是否允许访问；字段缺失表示默认允许。                                                                                  | `{}`    |

## 身份层级

资源级规则使用五类身份字段：

| 字段          | 角色                                                                                                                               | 后端角色值 |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| `guest`       | 游客，未登录且无已认证 API token 的请求。                                                                                          | `0`        |
| `user`        | 普通用户。                                                                                                                         | `1`        |
| `audit_admin` | 审计管理员，只读管理角色；可查看日志和部分管理列表，但不能访问渠道管理，也不能查看渠道名称、密钥、Base URL、标签等可识别渠道信息。 | `5`        |
| `admin`       | 管理员。                                                                                                                           | `10`       |
| `root`        | 超级管理员。                                                                                                                       | `100`      |

字段没有继承关系，必须按身份单独配置。例如只写 `"admin": false` 只会禁止管理员，
不会自动禁止超级管理员；如需超级管理员也禁止，必须同时写 `"root": false`。

## 资源级规则

classic 后台的 `系统设置 -> 访问限制 -> 资源访问矩阵` 提供按资源分组的权限勾选界面：
行是资源，列是游客、普通用户、审计管理员、管理员和超级管理员。勾选表示允许访问，
取消勾选表示拒绝访问；每个分组标题行上的复选框可以按身份批量切换该分组下全部资源。
“全部允许”会清空资源级覆盖规则，“应用常用限制”会写入首页、令牌、钱包和账单的常见限制模板。

该界面保存时仍写入 `access_control.resource_rules`。底层配置是 JSON 对象，第一层 key 是资源，
第二层 key 是身份字段。语义如下：

- 没有配置某个资源：默认允许。
- 配置了资源但缺少某个身份字段：该身份默认允许。
- 某身份字段为 `false`：拒绝该身份访问该资源。
- 某身份字段为 `true`：允许该身份访问该资源。

示例：游客不能访问官网首页，普通用户、审计管理员和管理员不能访问令牌、钱包和账单，
只有超级管理员可以访问这些敏感资源：

```json
{
  "home": {
    "guest": false,
    "user": true,
    "audit_admin": true,
    "admin": true,
    "root": true
  },
  "token": {
    "guest": false,
    "user": false,
    "audit_admin": false,
    "admin": false,
    "root": true
  },
  "wallet": {
    "guest": false,
    "user": false,
    "audit_admin": false,
    "admin": false,
    "root": true
  },
  "billing": {
    "guest": false,
    "user": false,
    "audit_admin": false,
    "admin": false,
    "root": true
  }
}
```

要让这组规则完整生效，需要同时开启：

- `access_control.web_policy_enabled`：拦截 Web 页面并让 classic 前端隐藏菜单。
- `access_control.api_policy_enabled`：拦截相关后端接口和模型 API 服务。

如果数据库中已经存在自定义资源 key，classic 会在“自定义资源”分组中显示并保留这些规则；
当前内置资源矩阵覆盖下方“资源 key”表列出的资源。

### 资源 key

| 资源 key             | 覆盖范围                                                                                                                                                                          |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `web`                | 官网普通 Web 页面，例如 `/`、`/pricing`、`/rankings`、`/about`、协议和隐私页。登录、注册、重置密码、OAuth 回调和 setup 页面不映射到 `web`，避免锁死入口。                         |
| `home`               | 官网首页 `/`。                                                                                                                                                                    |
| `model_api`          | 大模型 API / relay 服务，例如 `/v1`、`/v1beta`、`/mj`、`/:mode/mj`、`/suno`、`/kling/v1`、`/jimeng`、`/pg`，以及带 `relay` 路由标签的模型服务。它不是所有后端 `/api/*` 管理接口。 |
| `token`              | 令牌管理页面 `/console/token` 和 `/api/token/*`。                                                                                                                                 |
| `wallet`             | 钱包/充值/订阅购买页面 `/console/topup`，以及充值、支付、余额订阅购买、支付方式询价、返佣转余额等接口。                                                                           |
| `billing`            | 账单页面 `/console/billing`、用户账单列表、管理员账单列表和旧 dashboard billing API。                                                                                             |
| `usage_log`          | 使用日志页面 `/console/log`，日志、用量和数据统计接口。                                                                                                                           |
| `dashboard`          | 数据看板 `/console`。                                                                                                                                                             |
| `playground`         | 操练场 `/console/playground` 和 `/pg/*`。                                                                                                                                         |
| `chat`               | 聊天页 `/console/chat/*` 和 `/chat2link`。                                                                                                                                        |
| `personal`           | 个人设置 `/console/personal`，以及 2FA、Passkey、OAuth 绑定、签到和用户设置接口。                                                                                                 |
| `drawing_log`        | 绘图日志 `/console/midjourney` 和 `/api/mj/*`。                                                                                                                                   |
| `task_log`           | 任务日志 `/console/task` 和 `/api/task/*`。                                                                                                                                       |
| `admin_channel`      | 渠道、分组、预填分组和厂商管理。该资源只能由管理员及以上访问；审计管理员即使资源规则缺省允许，也会被 `/api/channel/*` 后端认证和 classic 路由拒绝。                               |
| `admin_subscription` | 订阅管理。                                                                                                                                                                        |
| `admin_model`        | 模型管理。                                                                                                                                                                        |
| `admin_redemption`   | 兑换码管理。                                                                                                                                                                      |
| `admin_user`         | 用户管理。                                                                                                                                                                        |
| `admin_referral`     | 邀请管理。                                                                                                                                                                        |
| `admin_setting`      | 系统设置、性能、同步比例和自定义 OAuth Provider 管理。                                                                                                                            |

classic 前端会从 `/api/status` 读取 `access_control.resource_rules` 和
`access_control.resource_access`，并按同一资源 key 隐藏侧边栏菜单项；例如 `token.user=false`
时普通用户看不到“令牌管理”。菜单隐藏只是体验层，直接访问 URL 或接口仍由服务端中间件拦截。

### 审计管理员与渠道信息

审计管理员用于查看审计、日志和只读管理状态，不用于排查或维护具体上游渠道。为避免向审计管理员泄露渠道配置和供应商识别信息：

- `/api/channel/*` 渠道管理接口要求管理员及以上，审计管理员不能访问渠道列表、搜索、模型列表、批量测试、渠道详情或渠道管理页下的监控兼容入口。
- classic 的 `/console/channel` 路由和侧栏“渠道管理”入口要求管理员及以上。
- `/api/prefill_group/*` 预填组接口也要求管理员及以上，避免审计管理员通过渠道标签、端点模板或渠道配置辅助数据反推出具体渠道信息。
- 审计管理员可只读查看模型管理列表，但模型的“已绑定渠道”只返回和展示渠道数字 ID；接口不会向审计管理员返回绑定渠道名称或渠道类型，前端 tooltip 也只显示 `#渠道ID`。
- 审计管理员查看使用日志时，后端不回填 `channel_name`，并会清洗 `Other` 中的 `channel_name`、渠道亲和详情和 `channel.*` 审计参数里的名称、标签、Base URL 等可识别字段；前端渠道列、展开详情和鼠标悬停只显示渠道数字 ID，例如 `#12`，不显示渠道名。
- 主页/模型页的渠道可用性监控走 `/api/log/channel_monitor/summary` 审计只读入口；审计管理员可看健康度和最近错误，但表格只展示渠道数字 ID，后端也不会把日志或渠道表中的 `channel_name` 回填给审计管理员。
- 管理审计日志中的 `channel.*` 操作参数对审计管理员脱敏，只保留渠道数字 ID、数量、变更字段和执行状态等非识别字段，不保留渠道类型、名称、标签或上游地址。

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
