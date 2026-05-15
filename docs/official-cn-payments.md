# 支付宝和微信支付官方接入

本文档记录 StuHelper AI 的中国大陆官方企业支付接入。该功能面向企业主体
开通的支付宝开放平台和微信支付直连商户能力，不是易支付聚合支付，也不是
支付宝当面付。

## 支持范围

- 支付宝电脑网站支付：`alipay.trade.page.pay`，`product_code` 为
  `FAST_INSTANT_TRADE_PAY`。这是支付宝电脑网站支付官方文档中
  `biz_content.product_code` 的固定产品码，错误产品码会导致支付宝收银台
  提示订单信息无法识别或 `INVALID_PARAMETER`。
- 支付宝手机网站支付：`alipay.trade.wap.pay`，`product_code` 为
  `QUICK_WAP_WAY`。
- 微信支付 Native 支付：`POST /v3/pay/transactions/native`，前端展示
  `code_url` 二维码，适用于电脑网站扫码支付。
- 微信支付 H5 支付：`POST /v3/pay/transactions/h5`，前端使用 `h5_url`
  跳转微信支付收银台，适用于移动端浏览器。H5 是微信支付侧单独的产品权限；
  如果商户未开通 H5，后端只在微信返回 H5 无权限/产品不可用时降级为 Native
  `code_url` 扫码支付，其他证书、签名、参数错误仍按失败处理。

官方文档来源通过 Context7 查询确认：

- Alipay Open Docs：`/websites/opendocs_alipay`
- WeChat Pay API v3：`/websites/pay_weixin_qq_doc_v3`

本轮支付宝官方支付按以下开放平台文档校准：

- 电脑网站支付快速接入：
  `https://opendocs.alipay.com/open-v3/05w3qg?pathHash=e5f3724a`
- `alipay.trade.page.pay`：
  `https://opendocs.alipay.com/open-v3/2423fad5_alipay.trade.page.pay?scene=22&pathHash=86a404ff`
- 手机网站支付快速接入：
  `https://opendocs.alipay.com/open-v3/05w4kt?pathHash=51d42218`
- `alipay.trade.wap.pay`：
  `https://opendocs.alipay.com/open-v3/1a957be0_alipay.trade.wap.pay?scene=21&pathHash=9012db1f`
- 支付宝异步通知说明：
  `https://opendocs.alipay.com/open-v3/05w3qh?pathHash=78bd7a2c` 和
  `https://opendocs.alipay.com/open-v3/05w4ku?pathHash=af025e20`
- `alipay.trade.refund`：
  `https://opendocs.alipay.com/open-v3/01073208_alipay.trade.refund?scene=common&pathHash=a6d8f430`
- `alipay.trade.fastpay.refund.query`：
  `https://opendocs.alipay.com/open-v3/46bff59c_alipay.trade.fastpay.refund.query?scene=common&pathHash=cfdb9929`
- `alipay.trade.query`：
  `https://opendocs.alipay.com/open-v3/e9ce4f59_alipay.trade.query?scene=23&pathHash=6efa478d`
- `alipay.trade.close`：
  `https://opendocs.alipay.com/open-v3/429ffb46_alipay.trade.close?scene=common&pathHash=42b295c0`
- `alipay.trade.refund.depositback.completed`：
  `https://opendocs.alipay.com/open-v3/42a9ce75_alipay.trade.refund.depositback.completed?scene=common&pathHash=9c33d734`

本轮微信支付官方支付按 WeChat Pay API v3 文档校准：

- 商户订单号查询订单：`GET /v3/pay/transactions/out-trade-no/{out_trade_no}?mchid=...`
- 商户订单号关闭订单：`POST /v3/pay/transactions/out-trade-no/{out_trade_no}/close`
- 申请退款：`POST /v3/refund/domestic/refunds`
- 查询单笔退款：`GET /v3/refund/domestic/refunds/{out_refund_no}`
- 退款结果通知：`event_type=REFUND.SUCCESS`、`REFUND.ABNORMAL` 或
  `REFUND.CLOSED`，`resource.original_type=refund`

## 后端配置项

支付宝官方支付：

- `AlipayOfficialEnabled`：是否启用。
- `AlipayOfficialSandbox`：是否使用支付宝沙盒网关。
- `AlipayOfficialAppID`：支付宝开放平台应用 AppID。
- `AlipayOfficialAppAuthToken`：支付宝应用授权 Token。服务商/第三方代理代商户
  调用时必须填写；直连商户应用可留空。电脑网站支付、手机网站支付、查询、
  关闭、退款和退款查询都会携带同一个授权 Token，避免订单创建在被授权商户
  上下文中、后续查询/关闭却落到服务商应用自身上下文而返回
  `ACQ.TRADE_NOT_EXIST`。
- `AlipayOfficialPrivateKey`：应用私钥，支持 PEM 或 Base64 DER。
- `AlipayOfficialAlipayPublicKey`：支付宝公钥，支持 PEM 或 Base64 DER。
- `AlipayOfficialAppCertSN`：应用公钥证书 SN，普通公钥模式可留空。
- `AlipayOfficialRootCertSN`：支付宝根证书 SN，普通公钥模式可留空。
- `AlipayOfficialAlipayCertSN`：支付宝公钥证书 SN，普通公钥模式可留空。
- `AlipayOfficialNotifyURL`：自定义异步通知地址，留空使用默认回调。
- `AlipayOfficialReturnURL`：自定义支付返回地址，留空返回充值页。
- `AlipayOfficialUnitPrice`：站内充值单价。
- `AlipayOfficialMinTopUp`：最低充值数量。
- `AlipayOfficialOrderTimeoutMin`：支付宝官方订单超时时间，默认 `10`
  分钟。创建支付宝电脑网站/手机网站支付时会同步写入 `timeout_express`。
  超时后后台维护任务会调用 `alipay.trade.close` 关闭支付宝侧订单；只有
  支付宝明确关闭成功或查询确认 `TRADE_CLOSED` 时，才把本地待支付充值单标记为
  `expired`，classic 账单显示“已超时”。

微信支付官方支付：

- `WechatPayOfficialEnabled`：是否启用。
- `WechatPayOfficialAppID`：直连商户绑定的 AppID。
- `WechatPayOfficialMchID`：微信支付商户号。
- `WechatPayOfficialCertificateSerial`：商户 API 证书序列号，用于请求签名
  `serial_no`。
- `WechatPayOfficialAPIv3Key`：APIv3 密钥，必须为 32 字节，用于解密回调
  `resource`。
- `WechatPayOfficialPrivateKey`：商户私钥，支持 PEM 或 Base64 DER。
- `WechatPayOfficialPlatformPublicKey`：微信支付平台公钥或平台证书 PEM，
  用于校验微信支付响应和回调签名。
- `WechatPayOfficialNotifyURL`：自定义异步通知地址，留空使用默认回调。
- `WechatPayOfficialReturnURL`：微信 H5 场景信息和回跳地址，留空使用充值页。
- `WechatPayOfficialUnitPrice`：站内充值单价。
- `WechatPayOfficialMinTopUp`：最低充值数量。

后台入口位于 classic 前端的“系统设置 -> 支付设置 -> 官方支付设置”。
私钥和 APIv3 密钥不会从 `/api/option/` 回显，但接口会返回
`AlipayOfficialAppAuthTokenConfigured`、`AlipayOfficialPrivateKeyConfigured`、
`WechatPayOfficialAPIv3KeyConfigured` 和 `WechatPayOfficialPrivateKeyConfigured`
这类布尔状态，供后台判断是否已有密钥可保留。后台保存时，敏感密钥输入框
留空表示保持现有密钥不变；如果此前没有保存过密钥，启用官方支付时必须重新
填写。支付宝公钥和微信平台公钥也支持“留空保持当前不变”，便于后续只调整
价格、回调地址或开关配置。

官方支付在钱包页展示前会做完整配置检查。支付宝官方支付必须同时满足
`AlipayOfficialEnabled`、`AlipayOfficialAppID`、`AlipayOfficialPrivateKey` 和
`AlipayOfficialAlipayPublicKey` 均已配置；微信支付官方支付必须同时满足开关、
AppID、商户号、商户证书序列号、APIv3 密钥、商户私钥和平台公钥均已配置。
仅打开“启用”开关不会让钱包页显示对应支付方式。

官方支付单价支持三位小数保存，例如 `7.231`。创建订单、金额预估和向支付网关
提交实际金额时，都会按进一法保留到两位小数；例如 `7.231` 会按 `7.24` 支付，
微信支付的分单位金额为 `724`。

## 回调地址

默认回调地址由站点回调地址拼接：

- 支付宝：`/api/alipay/official/notify`
- 微信支付：`/api/wechat-pay/official/notify`

如果部署在反向代理后面，应先确认 `ServerAddress` 或 `CustomCallbackAddress`
能生成公网 HTTPS 地址；也可以在对应官方支付配置项中填写完整自定义回调地址。

## 安全校验

支付宝通知处理：

- 使用支付宝公钥按 RSA2 验签。
- 验签内容排除 `sign` 和 `sign_type`。
- 只处理 `TRADE_SUCCESS` 和 `TRADE_FINISHED`。
- 回调 `total_amount` 必须与本地充值订单金额一致。
- 成功响应支付宝要求的纯文本 `success`。
- `alipay.trade.refund.depositback.completed` 退款冲退完成通知会在同一
  回调入口验签后处理；`dback_status=S` 标记退款成功，`F` 回滚本地退款。

支付宝主动接口：

- 创建电脑网站支付使用 `alipay.trade.page.pay`，`product_code` 固定为
  `FAST_INSTANT_TRADE_PAY`，并携带 `timeout_express`。
- 创建手机网站支付使用 `alipay.trade.wap.pay`，`product_code` 固定为
  `QUICK_WAP_WAY`，并在 `biz_content` 中携带 `quit_url` 返回充值页和
  `timeout_express`。
- 如果配置了 `AlipayOfficialAppAuthToken`，创建支付表单时会把
  `app_auth_token` 作为公共请求参数签名并提交；查询、退款、退款查询也会在
  `gateway.do` 请求中携带该参数。官方文档要求服务商代调用场景必须携带
  `app_auth_token`，否则支付宝会按服务商应用自身上下文查询，可能对真实存在的
  被授权商户订单返回 `ACQ.TRADE_NOT_EXIST`。
- 管理员账单中的“查询”会调用 `alipay.trade.query`，若返回
  `TRADE_SUCCESS` 或 `TRADE_FINISHED`，会按回调同样的金额校验和入账流程
  补齐订单。若支付宝返回交易不存在，说明支付宝侧尚未形成可查询交易或交易
  已不存在，接口会返回本地订单状态，不再把这类可预期状态当成操作失败。
- 管理员账单中的“关闭”和超时任务会调用 V3 REST
  `POST /v3/alipay/trade/close`。如果配置了应用授权 Token，会按 V3 文档通过
  `alipay-app-auth-token` 请求头传给支付宝。如果超时关闭失败，会再调用
  `alipay.trade.query` 对账：已支付则入账，已关闭则标记本地订单已超时，
  其他状态保留待支付并记录日志。支付宝返回交易不存在时不会再把本地订单标记为
  `expired`，避免用户继续使用旧支付入口付款后出现资金悬挂。
- 管理员账单中的“退款”会调用 `alipay.trade.refund`。本地先创建退款请求
  号 `out_request_no` 并预留可退金额/额度，支付宝返回 `fund_change=Y` 时
  标记成功；如果 `fund_change` 不是 `Y`，会继续调用
  `alipay.trade.fastpay.refund.query`，只有 `refund_status=REFUND_SUCCESS`
  才确认成功。失败或未确认会回滚本地预留。
- 退款请求和退款查询会携带 `query_options=["deposit_back_info"]`，用于让
  支付宝在涉及银行卡退款冲退时返回/通知冲退信息。
- 支持部分退款和全额退款。部分退款后充值单状态为 `partial_refunded`，全额
  退款后状态为 `refunded`；退款会按退款金额比例扣回用户额度，并按同样比例
  冲回充值返佣。
- 支付宝官方退款成功后会写入退款日志，日志保留与充值日志一致的支付审计字段：
  订单支付方式、回调支付方式、回调调用者 IP、服务器 IP、节点名称和系统版本；
  classic 日志表中退款行不展示渠道、令牌、模型、输入、输出、花费等模型调用
  字段；详情列与充值行一致使用普通日志内容渲染，展开行展示退款业务详情和支付
  审计信息。

支付宝排障经验：

- 如果用户已经能在支付宝后台看到未支付订单，但 classic 充值账单查询或关闭仍
  返回 `ACQ.TRADE_NOT_EXIST`，不要再优先判断为前端表单没有提交成功。此时应
  优先核对查询/关闭接口与创建订单是否使用同一个支付宝上下文：生产/沙盒环境、
  `app_id`、证书模式，以及服务商代调用时的 `AlipayOfficialAppAuthToken`。
- 服务商/第三方代理代商户调用时，支付创建、查询、关闭、退款和退款查询必须
  共用同一个授权 Token。`gateway.do` 接口通过公共参数 `app_auth_token` 传递并
  参与 RSA2 签名；V3 关闭接口通过 `alipay-app-auth-token` 请求头传递。缺少该
  Token 时，支付宝会按服务商应用自身上下文查询，真实存在的被授权商户订单也
  可能返回交易不存在。
- 管理员手动关闭或超时关闭订单时，只有支付宝侧明确关闭成功，或关闭失败后
  再查询确认 `TRADE_CLOSED`，本地才可以标记 `expired`。如果关闭返回交易不存在
  且无法确认支付宝侧订单已关闭，本地应保留待支付，避免用户继续使用旧支付入口
  支付后出现资金悬挂。

微信支付处理：

- 请求微信支付 API v3 时使用商户私钥构造
  `WECHATPAY2-SHA256-RSA2048` Authorization。
- 预支付响应使用微信平台公钥或平台证书按
  `timestamp + "\n" + nonce + "\n" + body + "\n"` 验签。
- 回调先使用 `Wechatpay-Timestamp`、`Wechatpay-Nonce`、
  `Wechatpay-Signature` 和原始 body 验签。
- 验签通过后，使用 APIv3 密钥按 `AEAD_AES_256_GCM` 解密 `resource`。
- 只处理 `event_type=TRANSACTION.SUCCESS` 且 `trade_state=SUCCESS`。
- 回调 `amount.total` 必须与本地充值订单金额一致。
- 成功响应 HTTP 204，失败响应微信支付 API v3 的 `FAIL` JSON。
- 管理员账单中的“查询”会调用微信支付 V3
  `GET /v3/pay/transactions/out-trade-no/{out_trade_no}?mchid=...`，并使用
  商户订单号对账。如果微信侧 `trade_state=SUCCESS` 但本地尚未入账，会按
  支付回调同样的金额校验和幂等流程补齐额度充值或订阅订单。
- 管理员账单中的“关闭”会调用微信支付 V3
  `POST /v3/pay/transactions/out-trade-no/{out_trade_no}/close`，请求体固定
  携带当前商户号 `mchid`。只有微信侧关闭接口返回 2xx 后，本地待支付订单才会
  标记为 `expired`；关闭失败不会先改本地状态，避免用户仍可在微信侧付款。
- 管理员账单中的“退款”会调用微信支付 V3
  `POST /v3/refund/domestic/refunds`。请求使用 `out_trade_no` 和唯一
  `out_refund_no`，金额字段按微信官方分单位传入：`amount.refund` 为本次退款
  分数，`amount.total` 为原订单分数，`amount.currency` 固定 `CNY`。
- 微信退款接口返回 `SUCCESS` 时立即标记本地退款成功；返回 `PROCESSING` 时
  本地保留退款记录为 `pending`，等待管理员“退款查询”或微信退款通知确认。
  `REFUNDCLOSE`、`CLOSED` 或 `ABNORMAL` 会回滚本地预留的退款金额、额度和
  返佣冲销；其中微信回调文档中的 `REFUND.CLOSED` / `refund_status=CLOSED`
  会归一为本地失败处理。
- 管理员“退款查询”使用
  `GET /v3/refund/domestic/refunds/{out_refund_no}`，确认 `SUCCESS` 后同步本地
  退款状态；确认失败状态后回滚。退款通知同样使用微信平台签名验签和 APIv3
  密钥解密后按 `resource.original_type=refund` 分流，并处理成功、异常和关闭
  三类退款事件。

两类支付都会校验本地订单 `PaymentProvider`，避免其他支付网关的订单被官方
支付回调误入账；订单完成使用事务和订单级锁保证幂等。

## 退款规则

官方支付宝和微信支付的额度充值、订阅订单都支持退款和部分退款。系统会先计算
一个“当前最多可退金额”，普通用户只能在该金额内提交退款申请，管理员审批后才
会调用官方支付退款 API；管理员也可以在账单里直接退款，并可选择全额退款。

额度充值的最多可退金额按订单未退金额和用户当前余额共同约束：系统先计算该
充值订单尚未退回的额度，再与用户当前余额取较小值，按原订单人民币金额比例
折算为可退人民币并向下保留两位小数。例如用户原有余额 100，又充值 100 后只
使用 50，当前余额仍覆盖这笔订单的 100 额度，则预设可全额退款；如果之后总共
使用 150，当前余额只剩 50，则这笔充值预设最多退款 50 对应的人民币。

订阅订单的最多可退金额按时间和额度两条线取更严格的已使用比例：`已使用比例 =
max(已过时间 / 总有效期, 已用额度 / 总额度)`，预设退款金额为订单金额乘以
`1 - 已使用比例`，并向下保留两位小数。订阅部分退款不会扣用户钱包余额；全额
退款会取消对应订阅实例并按订阅分组规则回退用户分组。

用户在自己的充值账单中可以对官方支付成功或部分退款订单发起退款申请，必须
填写退款原因。申请记录会挂到同一张充值账单上，管理员账单中会显示“审批退款”
和“拒绝”操作，并展示用户申请金额和原因。审批通过后才会调用支付宝或微信官方
退款接口；拒绝只更新申请状态，不调用支付网关。

退款会同步冲回邀请返佣。额度充值退款按 `topup` 来源和订单金额比例冲回；订阅
退款按 `subscription` 来源和订阅订单 ID 冲回。退款失败或微信退款关闭/异常时，
系统会回滚本地退款预留和返佣冲销。普通额度充值全额退款后，如果用户没有其他
有效成功付款记录，会撤销由首笔付款解锁的一次性邀请人奖励；订阅全额退款也会
触发同样的有效付款状态复核。

## 前端行为

classic 充值页根据浏览器环境选择支付场景：

- 电脑端支付宝：提交官方表单到支付宝电脑网站支付。
- 移动端支付宝：提交官方表单到支付宝手机网站支付。
- 电脑端微信支付：展示 Native `code_url` 二维码，用户用微信扫码支付。
- 移动端微信支付：优先跳转微信 H5 `h5_url`；未开通 H5 产品权限时返回
  Native `code_url` 并展示扫码二维码。
- classic 的微信 Native 二维码弹窗会按订单号轮询本地充值账单状态；支付回调
  入账后自动关闭弹窗并刷新用户额度或订阅状态，避免用户扫码完成后页面停留在
  未刷新状态。轮询会调用 `POST /api/user/wechat-pay/official/status`，后端按
  微信官方 `GET /v3/pay/transactions/out-trade-no/{out_trade_no}` 查询订单；
  如果微信侧已支付但回调尚未落库，也会主动完成入账。

classic 充值页同时存在“额度充值”和“订阅套餐”时，默认进入“额度充值”，并把
“额度充值”放在“订阅套餐”左侧，避免用户进入钱包后被自动切到订阅购买流程。
订阅套餐卡片的“推荐”标签和紫色高亮边框不再默认给第一个套餐，而是由 classic
后台“订阅管理”中的“推荐”开关手动控制。

official 支付与其他支付方式在 classic 订阅购买页走同一套选择流程：用户先在
套餐列表中选中一个套餐，再在套餐列表下方选择当前可用于订阅的支付方式，页面
即时展示该支付方式对应的实付金额，最后点击“立即订阅”进入确认弹窗。确认弹窗
只展示套餐、支付方式和应付金额，不再在弹窗内二次选择支付方式。

订阅金额展示和实际下单金额都按支付方式通用规则处理：如果支付方式带有
`unit_price`，则用套餐美元金额乘以该单价，并按进一法保留两位小数后显示为
人民币；如果支付方式没有独立单价，则按当前前端货币配置显示套餐金额。官方
支付宝订阅使用 `AlipayOfficialUnitPrice`，微信支付官方订阅使用
`WechatPayOfficialUnitPrice`，易支付订阅使用通用 `Price`，避免前端显示一个
金额、支付网关实际收取另一个金额。

官方支付宝同时接入“额度充值”和“订阅套餐购买”。如果只配置了支付宝官方支付、
没有配置易支付，订阅套餐仍会显示“支付宝”支付入口。订阅订单通过
`POST /api/subscription/alipay-official/pay` 创建，电脑端使用
`alipay.trade.page.pay`，移动端使用 `alipay.trade.wap.pay`。支付宝异步通知
仍使用 `/api/alipay/official/notify`，回调处理会先识别官方支付宝订阅订单并
完成订阅；如果不是订阅订单，再按普通额度充值处理。
微信支付官方同时接入“额度充值”和“订阅套餐购买”。如果只配置了微信支付官方、
没有配置易支付，订阅套餐仍会显示“微信”支付入口。订阅订单通过
`POST /api/subscription/wechat-pay-official/pay` 创建，电脑端使用微信 Native
支付并在 classic 弹窗中展示 `code_url` 二维码，移动端优先使用微信 H5 支付并
跳转 `h5_url`；未开通 H5 产品权限时降级为 Native 二维码。微信支付异步通知
仍使用 `/api/wechat-pay/official/notify`，回调处理会先识别微信支付官方订阅
订单并完成订阅；如果不是订阅订单，再按普通额度充值处理。
当易支付商户地址、商户号、密钥或易支付方式未完整配置时，充值页不会展示
易支付的支付宝、微信等方式；官方支付宝和微信支付按各自完整配置独立展示。
官方支付宝和微信支付实际仍以 `/api/user/topup/info` 返回的完整配置结果为准：
只有后端判定配置完整时，classic 钱包页才会展示对应支付按钮。充值数量变化或
选择预设额度时，classic 钱包页会按当前支付方式调用对应金额接口，避免官方支付
沿用易支付单价。
充值额度档位中，额度标题仍按系统额度展示类型显示；实付金额按当前选择的支付
方式币种显示，支付宝官方和微信支付官方固定显示人民币金额，并使用各自独立的
官方单价。未配置充值金额折扣时，档位卡片不会显示“节省 0.00”。
钱包管理的支付方式选择会把 `alipay_official` 简化显示为“支付宝”，把
`wxpay_official` 简化显示为“微信”，避免用户侧出现“官方支付”这种运营语义；
管理员支付设置页也使用同样的短名称作为 tab 标题，具体字段名仍保留
“支付宝 AppID”“微信支付商户号”等接入含义。
充值账单会将 `alipay_official` 映射为本地化后的“支付宝”，将
`wxpay_official` 映射为本地化后的“微信”，避免直接向用户展示后端枚举值。
订阅订单创建时会同步写入同一张充值账单，状态为待支付，金额为对应支付方式的
实付金额，充值额度固定为 0；支付成功、拉起失败或超时关闭时会同步更新同一条
账单状态。因此用户和管理员在支付回调前也可以按订单号查到订阅待支付账单。
classic 会把 `SUB`、`ALIPAYSUB`、`WXSUB` 前缀且充值额度为 0 的账单识别为
“订阅套餐”，避免官方订阅账单显示成普通 0 额度充值。
管理员查看充值账单时会额外显示用户名列，并支持按用户 ID、用户名和订单号搜索。
账单弹窗使用更宽的自适应宽度，便于显示退款、用户名和操作列。
待支付的支付宝官方订单在超过配置的订单超时时间后显示红色“已超时”。
用户账单中的官方支付成功订单会显示“申请退款”，已有待审核申请时显示“退款
审核中”。管理员账单对待审核申请显示“审批退款”和“拒绝”；直接退款仍保留在
没有待审核申请的官方支付订单上。退款确认弹窗会展示预设可退金额、订阅未使用
比例、用户申请金额和申请原因。

## 外部参考 PR

用户提供的 `QuantumNous/new-api#2677` 是支付宝当面付参考，本次没有导入该
PR 的实现，也没有按当面付产品实现。本功能按支付宝电脑网站支付、支付宝手机
网站支付、微信 Native 支付、微信 H5 支付的官方企业接入路径实现。
