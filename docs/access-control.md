# 访问限制策略

StuHelper AI 支持在服务端按请求来源国家/地区、访问身份和资源层级拦截官网 Web 与 API。
所有策略默认关闭，必须由超级管理员在 classic 后台 `系统设置 -> 访问限制` 中显式启用。

## 配置项

| Option Key                                                 | 用途                                                                                                                                                                                       | 默认值  |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------- |
| `access_control.web_policy_enabled`                        | 官网 Web 访问限制总开关。开启后，服务端 Web 路由和 classic 前端客户端路由都会按下方矩阵判断；关闭时 Web 页面不执行这些访问限制。                                                           | `false` |
| `access_control.api_policy_enabled`                        | API 访问限制总开关。开启后覆盖管理后台 `/api/*`、relay `/v1`/`/v1beta`/`/mj`/`/suno` 等 token API，以及旧 dashboard API；资源级 `model_api` 只匹配大模型 relay 服务，不等于全部 `/api/*`。 | `false` |
| `access_control.block_china_mainland`                      | 兼容旧配置：禁止识别为中国大陆的 IP 访问全部资源，国家代码为 `CN`。这是全局地域封禁，会影响游客、普通用户和管理员。                                                                        | `false` |
| `access_control.block_european_union`                      | 兼容旧配置：禁止识别为欧盟成员国的 IP 访问全部资源。                                                                                                                                       | `false` |
| `access_control.block_china_mainland_homepage`             | 兼容旧配置：仅禁止中国大陆 IP 的游客和普通用户访问官网主页资源。classic 后台会把它映射到 `source_resource_rules.china_mainland.home.guest/user=true`。                                     | `false` |
| `access_control.block_china_mainland_user_sensitive_pages` | 兼容旧配置：禁止中国大陆普通用户访问令牌、钱包和账单相关 Web 页面与 API。classic 后台会把它映射到 `source_resource_rules.china_mainland.token/wallet/billing.user=true`。                  | `false` |
| `access_control.block_guests`                              | 兼容旧配置：禁止游客访问。对 Web 表示无登录 session；对 API 表示无可识别认证凭据的请求。                                                                                                   | `false` |
| `access_control.block_users`                               | 兼容旧配置：禁止普通用户访问。API token 请求按 token 所属用户角色判断。                                                                                                                    | `false` |
| `access_control.block_admins`                              | 兼容旧配置：禁止管理员访问。包含审计管理员、管理员和超级管理员。                                                                                                                           | `false` |
| `access_control.geoip_database_path`                       | 本地 MaxMind 兼容 MMDB 国家库路径。留空时只使用可信代理注入的国家代码请求头。                                                                                                              | 空      |
| `access_control.role_geo_rules`                            | 兼容旧配置：来源 × 角色的全局限制。classic 后台不再单独展示；读取时迁移到 `source_resource_rules[source].all[role]=true`，保存时写回 `{}`。                                                | `{}`    |
| `access_control.source_resource_rules`                     | 访问限制矩阵主规则。按来源 key、资源 key、身份字段三层配置是否限制访问指定资源；字段为 `true` 表示限制，字段缺失表示不因该来源、资源和角色组合拦截。                                       | `{}`    |
| `access_control.resource_rules`                            | 兼容旧配置：不区分来源的资源拒绝规则。classic 后台不再单独展示；读取时把 `resource_rules[resource][role]=false` 迁移到 `source_resource_rules.all[resource][role]=true`，保存时写回 `{}`。 | `{}`    |

## 身份层级

访问限制矩阵和兼容旧字段都使用五类身份字段：

| 字段          | 角色                                                                                                                               | 后端角色值 |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| `guest`       | 游客，未登录且无已认证 API token 的请求。                                                                                          | `0`        |
| `user`        | 普通用户。                                                                                                                         | `1`        |
| `audit_admin` | 审计管理员，只读管理角色；可查看日志和部分管理列表，但不能访问渠道管理，也不能查看渠道名称、密钥、Base URL、标签等可识别渠道信息。 | `5`        |
| `admin`       | 管理员。                                                                                                                           | `10`       |
| `root`        | 超级管理员。                                                                                                                       | `100`      |

字段没有继承关系，必须按身份单独配置。例如只写 `"admin": true` 只会匹配管理员，
不会自动匹配超级管理员；如需超级管理员也限制，必须同时写 `"root": true`。

## 执行顺序

服务端访问限制的执行顺序固定如下：

1. 旧全局地域开关：`block_china_mainland`、`block_european_union`。
2. 兼容来源角色全局规则：`role_geo_rules[source][role]=true`。
3. 访问限制矩阵主规则：`source_resource_rules[source][resource][role]=true`。
4. 旧中国大陆细粒度开关：`block_china_mainland_homepage`、`block_china_mainland_user_sensitive_pages`。
5. 兼容资源全局拒绝规则：`resource_rules[resource][role]=false`。
6. 旧全局身份开关：`block_guests`、`block_users`、`block_admins`。

上面任一层命中都会直接拒绝请求。API 请求如果已经携带 `Authorization`、`api-key`、
`mj-api-secret`、`x-api-key`、`x-goog-api-key`、WebSocket `Sec-WebSocket-Protocol`
中的 OpenAI realtime key，或 `key` query 参数，前置中间件不会把它误判为游客；它会等认证完成后按真实用户角色再次判断。

classic 后台读取设置时会把旧 JSON 字段和旧布尔开关合并到 `source_resource_rules` 主矩阵；保存时继续写入 `source_resource_rules`，
并把 `role_geo_rules` 与 `resource_rules` 写回 `{}`。因此正常后台操作后主要由第 3 层主规则执行；
如果旧脚本或数据库手工写入兼容字段，后端仍会按上面的顺序执行这些兼容规则。

## 访问限制矩阵

classic 后台的 `系统设置 -> 访问限制 -> 访问限制矩阵` 是唯一的细粒度配置入口。
界面先通过“限制来源”下拉菜单选择来源 IP/地区，再展示该来源下的资源 × 角色矩阵；不同来源不再并排展开。

矩阵行是官网、令牌、钱包、账单、API、日志、管理资源等资源，列是游客、普通用户、审计管理员、管理员和超级管理员。
勾选表示限制该角色从当前来源访问该资源；未勾选表示不因这个来源、资源和角色组合拦截。
分组标题行上的复选框可以批量限制或放开当前来源下某一资源组的某类角色。

后台保存时写入 `access_control.source_resource_rules`。底层配置是 JSON 对象，第一层 key 是来源，
第二层 key 是资源，第三层 key 是身份字段：

- 没有配置某个来源：默认不因该来源拦截。
- 配置了来源但缺少某个资源：默认不因该来源和资源组合拦截。
- 配置了资源但缺少某个身份字段：默认不因该来源、资源和身份组合拦截。
- 某身份字段为 `true`：拒绝该身份从该来源访问该资源。

内置来源 key：

| 来源 key          | 覆盖范围                                       |
| ----------------- | ---------------------------------------------- |
| `all`             | 全部来源，不区分 IP 或国家地区。               |
| `china_mainland`  | 识别为中国大陆的请求，国家代码为 `CN`。        |
| `european_union`  | 识别为欧盟成员国的请求。                       |
| `unknown_country` | 无法通过可信代理头或 MMDB 识别国家代码的请求。 |

`european_union` 只匹配下方列出的欧盟成员国代码；例如 `RU` 不属于欧盟成员国，不会因为欧盟来源规则或
`block_european_union` 被拦截。`RU` 请求如果被拒绝，应检查 `all` 来源、具体资源、全局身份开关或
其他非地区规则是否命中；拒绝页会显示具体命中的来源、资源、角色和策略范围。

资源 key `all` 是矩阵内置的“全部资源”伪资源，只在 `source_resource_rules` 中有效，表示“当前来源下全部 Web 与 API 资源”。
`all` 不是实际路由资源，不会出现在下方普通资源 key 表中。

- `source_resource_rules.china_mainland.all.user=true`：限制中国大陆 IP 普通用户访问所有 Web 与 API 资源。
- `source_resource_rules.all.all.guest=true`：不区分来源，限制游客访问所有 Web 与 API 资源。
- `source_resource_rules.all.token.user=true`：不区分来源，限制普通用户访问令牌资源。

这使主矩阵同时覆盖旧“来源角色全局限制矩阵”和旧“高级资源全局限制”的能力：

- 旧来源角色全局限制：使用某个来源下的“全部资源”行。
- 旧不区分来源的资源全局限制：选择“全部来源”，再勾选具体资源行。

示例：禁止中国大陆 IP 的游客和普通用户访问官网首页，同时禁止中国大陆普通用户访问令牌、钱包和账单，并且不区分来源禁止游客访问模型 API：

```json
{
  "china_mainland": {
    "home": {
      "guest": true,
      "user": true
    },
    "token": {
      "user": true
    },
    "wallet": {
      "user": true
    },
    "billing": {
      "user": true
    }
  },
  "all": {
    "model_api": {
      "guest": true
    }
  }
}
```

旧字段与主矩阵的读取迁移关系：

- `role_geo_rules[source][role]=true` 迁移为 `source_resource_rules[source].all[role]=true`。
- `resource_rules[resource][role]=false` 迁移为 `source_resource_rules.all[resource][role]=true`。
- `block_china_mainland=true` 迁移为 `china_mainland.all` 下五类身份全部限制。
- `block_european_union=true` 迁移为 `european_union.all` 下五类身份全部限制。
- `block_guests=true` 迁移为 `all.all.guest=true`。
- `block_users=true` 迁移为 `all.all.user=true`。
- `block_admins=true` 迁移为 `all.all.audit_admin=true`、`all.all.admin=true`、`all.all.root=true`。
- `block_china_mainland_homepage=true` 映射为 `china_mainland.home.guest=true` 和 `china_mainland.home.user=true`。
- `block_china_mainland_user_sensitive_pages=true` 映射为 `china_mainland.token.user=true`、`china_mainland.wallet.user=true`、`china_mainland.billing.user=true`。

旧字段不再作为 classic 后台的独立菜单或快捷开关展示。保存时后台会从主矩阵反写旧布尔开关，
以保持旧脚本和旧部署兼容；`role_geo_rules` 与 `resource_rules` 则写回 `{}`，避免同一语义在多个菜单重复维护。

要让这些规则完整生效，需要同时开启对应作用域总开关：

- `access_control.web_policy_enabled`：拦截 Web 页面并让 classic 前端隐藏菜单。
- `access_control.api_policy_enabled`：拦截相关后端接口和模型 API 服务。

如果数据库中已经存在自定义资源 key，classic 会在当前来源的“自定义资源”分组中显示并保留这些拒绝规则；
当前内置资源矩阵覆盖下方“资源 key”表列出的资源。

### 资源 key

| 资源 key             | 覆盖范围                                                                                                                                                                                   |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `web`                | 官网普通 Web 页面，例如 `/`、`/pricing`、`/rankings`、`/about`、协议和隐私页；未知 SPA fallback 路径也会归入 `web`。登录、注册、重置密码、OAuth 回调、setup、静态资源和 API 路径不会归入。 |
| `home`               | 官网首页 `/`，以及未知 SPA fallback 路径，例如 `/1`、`/foo`。登录、注册、重置密码、OAuth 回调、setup、静态资源、`/console/*` 和 API 路径不会归入。                                         |
| `model_api`          | 大模型 API / relay 服务，例如 `/v1`、`/v1beta`、`/mj`、`/:mode/mj`、`/suno`、`/kling/v1`、`/jimeng`、`/pg`，以及带 `relay` 路由标签的模型服务。它不是所有后端 `/api/*` 管理接口。          |
| `token`              | 令牌管理页面 `/console/token` 和 `/api/token/*`。                                                                                                                                          |
| `wallet`             | 钱包/充值/订阅购买页面 `/console/topup`，以及充值、支付、余额订阅购买、支付方式询价、返佣转余额等接口。                                                                                    |
| `billing`            | 账单页面 `/console/billing`、用户账单列表、管理员账单列表和旧 dashboard billing API。                                                                                                      |
| `usage_log`          | 使用日志页面 `/console/log`，日志、用量和数据统计接口。                                                                                                                                    |
| `dashboard`          | 数据看板 `/console`。                                                                                                                                                                      |
| `playground`         | 操练场 `/console/playground` 和 `/pg/*`。                                                                                                                                                  |
| `chat`               | 聊天页 `/console/chat/*` 和 `/chat2link`。                                                                                                                                                 |
| `personal`           | 个人设置 `/console/personal`，以及 2FA、Passkey、OAuth 绑定、签到和用户设置接口。                                                                                                          |
| `drawing_log`        | 绘图日志 `/console/midjourney` 和 `/api/mj/*`。                                                                                                                                            |
| `task_log`           | 任务日志 `/console/task` 和 `/api/task/*`。                                                                                                                                                |
| `admin_channel`      | 渠道、分组、预填分组和厂商管理。该资源只能由管理员及以上访问；审计管理员即使资源规则缺省允许，也会被 `/api/channel/*` 后端认证和 classic 路由拒绝。                                        |
| `admin_subscription` | 订阅管理。                                                                                                                                                                                 |
| `admin_model`        | 模型管理。                                                                                                                                                                                 |
| `admin_redemption`   | 兑换码管理。                                                                                                                                                                               |
| `admin_user`         | 用户管理。                                                                                                                                                                                 |
| `admin_referral`     | 邀请管理。                                                                                                                                                                                 |
| `admin_setting`      | 系统设置、性能、同步比例和自定义 OAuth Provider 管理。                                                                                                                                     |

未知 SPA fallback 是指会由 Web NoRoute 返回 classic 首页的路径，但排除这些入口：
`/setup`、`/login`、`/register`、`/reset`、`/user/reset`、`/forbidden`、`/favicon.ico`、
`/api*`、`/v1*`、`/v1beta*`、`/mj*`、`/suno*`、`/kling*`、`/jimeng*`、`/pg*`、
`/assets*`、`/static*`、`/oauth*`、包含 `.` 的静态文件路径，以及 `/console*`。
因此中国大陆游客访问 `/1` 这类未知路径时，会和访问 `/` 一样命中 `home` 资源限制，不会绕过主页封禁看到首页内容和公告。

classic 前端会从 `/api/status` 读取 `access_control.source_resource_rules`、`access_control.resource_rules`
和当前请求来源识别结果，并按同一资源 key 隐藏侧边栏菜单项或在客户端路由显示 403。
菜单隐藏只是体验层，直接访问 URL 或接口仍由服务端中间件拦截。

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
3. 如果无法识别国家代码，`block_china_mainland`、`block_european_union`、`china_mainland`
   和 `european_union` 来源策略都不会命中；如需限制这类请求，可在访问限制矩阵中选择 `unknown_country` 来源。

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
令牌管理、钱包管理、账单管理，管理员不受影响”，推荐启用：

- `access_control.web_policy_enabled`
- `access_control.api_policy_enabled`
- 访问限制矩阵中的 `china_mainland.home.guest=true`
- 访问限制矩阵中的 `china_mainland.home.user=true`
- 访问限制矩阵中的 `china_mainland.token.user=true`
- 访问限制矩阵中的 `china_mainland.wallet.user=true`
- 访问限制矩阵中的 `china_mainland.billing.user=true`

不要为该需求启用 `access_control.block_china_mainland`，也不要在访问限制矩阵中把
`china_mainland.all` 五类身份全部勾选。两者都是全局地域封禁语义，会先于细粒度策略生效，并且会影响管理员。

旧兼容开关不再作为后台菜单展示；如果历史配置或脚本仍写入它们，后台读取时会迁移为上述矩阵规则：

- `block_china_mainland_homepage`：迁移为限制中国大陆游客和普通用户访问 `home` 资源。
- `block_china_mainland_user_sensitive_pages`：迁移为限制中国大陆普通用户访问 `token`、`wallet`、`billing` 资源。

这组策略的 Web 受限路径包括：

- `/`
- `/1` 等未知 SPA fallback 路径
- `/console/token`
- `/console/topup`
- `/console/billing`

这组策略的普通用户 API 受限路径包括：

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

`/api/status` 会返回当前请求的 `access_control` 识别结果，包括兼容字段 `role_geo_rules`、
主规则 `source_resource_rules`、兼容字段 `resource_rules`、兼容旧布尔开关、当前角色、请求 IP、
请求国家代码、IP 归属地展示值、是否中国大陆来源和是否欧盟来源。
classic 前端据此隐藏受限菜单项并在客户端路由显示 403；真正的访问控制仍在服务端中间件执行。

## 生效路径

- Web 策略挂在 classic Web 路由层，影响内置 classic 前端页面、未知 SPA fallback 页面和外置前端
  `FRONTEND_BASE_URL` 重定向路径；登录、注册、重置密码、OAuth 回调、setup 和静态资源会被排除在主页资源之外，避免锁死入口。
- API 策略分两段执行：
  - `/api/*`、relay、视频和旧 dashboard API 路由会先执行地域限制、兼容来源角色限制、访问限制矩阵、兼容资源全局限制和游客限制。
    带 API 凭据的请求会延后到认证完成后再按真实角色判断。
  - `UserAuth`、`TokenAuth`、`TokenAuthReadOnly` 等认证完成后，会再次按真实用户角色执行兼容来源角色、访问限制矩阵、用户、管理员和兼容资源限制。
- Web 拒绝响应返回自包含的 403 HTML 页面，并把浏览器地址显示为
  `/forbidden?access_limited=1`。该页面不依赖前端 JS/CSS 资源，标题为“访问请求已被策略拦截”，并展示具体命中原因、
  命中来源、命中资源、命中角色、策略范围、您当前 IP 和 IP 归属地。归属地展示值由服务端生成：中国大陆显示为“中国大陆”，
  欧盟国家显示为“欧盟地区（国家代码）”，其他已知地区显示国家代码，无法识别时显示“未知”。非地区策略命中时不会再误写成
  “本站不对您所在的地区开放”。
- 管理后台 API 的拒绝响应保持 JSON 语义，包含 `{ "success": false, "message": "访问受限" }`，并额外返回
  `reason_text` 和结构化 `reason` 字段，说明命中的来源、资源、角色和策略范围。relay API 的拒绝响应保持 OpenAI 风格错误体，
  便于客户端按 API 语义处理。

## 运维注意事项

- 如果站点在 CDN 或反向代理后运行，必须确保 `TrustedProxies` / 部署层真实客户端 IP
  配置正确，否则 MMDB 查询可能拿到代理 IP。
- `CF-IPCountry` 等请求头只有在入口代理可信时才应使用。不要允许外部客户端绕过代理直连
  后端并伪造这些请求头。
- 在访问限制矩阵中限制 `all.all.user`、`all.all.audit_admin`、`all.all.admin` 或 `all.all.root` 前，应确认仍有
  不受该策略影响的管理入口，例如临时保留 Web 策略关闭、API 策略关闭，或通过部署层临时
  改回 options 表。
- 访问限制矩阵的 `all` 资源会限制当前来源下所有 Web 与 API 资源；如果只想限制官网、令牌、钱包、账单或模型 API，应使用具体资源 key。
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
