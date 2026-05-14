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
  跳转微信支付收银台，适用于移动端浏览器。

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

## 后端配置项

支付宝官方支付：

- `AlipayOfficialEnabled`：是否启用。
- `AlipayOfficialSandbox`：是否使用支付宝沙盒网关。
- `AlipayOfficialAppID`：支付宝开放平台应用 AppID。
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
`AlipayOfficialPrivateKeyConfigured`、`WechatPayOfficialAPIv3KeyConfigured`
和 `WechatPayOfficialPrivateKeyConfigured` 这类布尔状态，供后台判断是否已有
密钥可保留。后台保存时，敏感密钥输入框留空表示保持现有密钥不变；如果此前
没有保存过密钥，启用官方支付时必须重新填写。支付宝公钥和微信平台公钥也支持
“留空保持当前不变”，便于后续只调整价格、回调地址或开关配置。

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
- 管理员账单中的“查询”会调用 `alipay.trade.query`，若返回
  `TRADE_SUCCESS` 或 `TRADE_FINISHED`，会按回调同样的金额校验和入账流程
  补齐订单。若支付宝返回交易不存在，说明支付宝侧尚未形成可查询交易或交易
  已不存在，接口会返回本地订单状态，不再把这类可预期状态当成操作失败。
- 管理员账单中的“关闭”和超时任务会调用 V3 REST
  `POST /v3/alipay/trade/close`。如果超时关闭失败，会再调用
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
  字段，只展示退款业务详情和 IP。

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

两类支付都会校验本地订单 `PaymentProvider`，避免其他支付网关的订单被官方
支付回调误入账；订单完成使用事务和订单级锁保证幂等。

## 前端行为

classic 充值页根据浏览器环境选择支付场景：

- 电脑端支付宝：提交官方表单到支付宝电脑网站支付。
- 移动端支付宝：提交官方表单到支付宝手机网站支付。
- 电脑端微信支付：展示 Native `code_url` 二维码，用户用微信扫码支付。
- 移动端微信支付：跳转微信 H5 `h5_url`。

官方支付目前只接入“额度充值”。订阅套餐购买弹窗会过滤
`alipay_official` 和 `wxpay_official`，不会把它们当成易支付子渠道使用。
当易支付商户地址、商户号、密钥或易支付方式未完整配置时，充值页不会展示
易支付的支付宝、微信等方式；官方支付宝和微信支付按各自完整配置独立展示。
官方支付宝和微信支付实际仍以 `/api/user/topup/info` 返回的完整配置结果为准：
只有后端判定配置完整时，classic 钱包页才会展示对应支付按钮。充值数量变化或
选择预设额度时，classic 钱包页会按当前支付方式调用对应金额接口，避免官方支付
沿用易支付单价。
充值额度档位中，额度标题仍按系统额度展示类型显示；实付金额按当前选择的支付
方式币种显示，支付宝官方和微信支付官方固定显示人民币金额，并使用各自独立的
官方单价。未配置充值金额折扣时，档位卡片不会显示“节省 0.00”。
钱包管理的支付方式选择会把 `alipay_official` 简化显示为“支付宝”，避免用户侧
出现“支付宝官方支付”这种运营语义；管理员支付设置中仍保留“支付宝官方”以区分
易支付和官方直连配置。
充值账单会将 `alipay_official` 映射为本地化后的“支付宝”，避免直接向用户展示
后端枚举值。管理员查看充值账单时会额外显示用户名列，并支持按用户 ID、用户名
和订单号搜索。账单弹窗使用更宽的自适应宽度，便于显示退款、用户名和操作列。
待支付的支付宝官方订单在超过配置的订单超时时间后显示红色“已超时”。

## 外部参考 PR

用户提供的 `QuantumNous/new-api#2677` 是支付宝当面付参考，本次没有导入该
PR 的实现，也没有按当面付产品实现。本功能按支付宝电脑网站支付、支付宝手机
网站支付、微信 Native 支付、微信 H5 支付的官方企业接入路径实现。
