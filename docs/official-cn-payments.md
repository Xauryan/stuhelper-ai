# 支付宝和微信支付官方接入

本文档记录 StuHelper AI 的中国大陆官方企业支付接入。该功能面向企业主体
开通的支付宝开放平台和微信支付直连商户能力，不是易支付聚合支付，也不是
支付宝当面付。

## 支持范围

- 支付宝电脑网站支付：`alipay.trade.page.pay`，`product_code` 为
  `FAST_INSTANT_PAY_PAY`。
- 支付宝手机网站支付：`alipay.trade.wap.pay`，`product_code` 为
  `QUICK_WAP_PAY`。
- 微信支付 Native 支付：`POST /v3/pay/transactions/native`，前端展示
  `code_url` 二维码，适用于电脑网站扫码支付。
- 微信支付 H5 支付：`POST /v3/pay/transactions/h5`，前端使用 `h5_url`
  跳转微信支付收银台，适用于移动端浏览器。

官方文档来源通过 Context7 查询确认：

- Alipay Open Docs：`/websites/opendocs_alipay`
- WeChat Pay API v3：`/websites/pay_weixin_qq_doc_v3`

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

## 外部参考 PR

用户提供的 `QuantumNous/new-api#2677` 是支付宝当面付参考，本次没有导入该
PR 的实现，也没有按当面付产品实现。本功能按支付宝电脑网站支付、支付宝手机
网站支付、微信 Native 支付、微信 H5 支付的官方企业接入路径实现。
