# 自助充值

自助充值允许管理员在系统设置中上传微信和支付宝个人收款码。用户充值时选择“微信自助”或“支付宝自助”，扫码付款后手动提交充值金额和微信/支付宝交易订单号。系统会先实时为用户增加余额，再由管理员在账单管理中每日对账审核。

## 配置入口

入口：`系统设置` -> `支付设置` -> `自助充值`。

配置项：

- `SelfServeTopUpEnabled`：自助充值总开关。
- `SelfServeAlipayEnabled`：是否启用支付宝自助充值。
- `SelfServeWechatPayEnabled`：是否启用微信自助充值。
- `SelfServeAlipayQRCode`：支付宝收款码，支持 `http/https` 图片地址或 `data:image/...;base64,...`。
- `SelfServeWechatPayQRCode`：微信收款码，支持 `http/https` 图片地址或 `data:image/...;base64,...`。
- `SelfServeTopUpSingleMaxAmount`：每人单笔自助充值金额上限，单位为人民币元。
- `SelfServeTopUpDailyMaxAmount`：每人每日自助充值累计金额上限，单位为人民币元。
- `SelfServeRejectAutoBan`：拒绝审核时的封禁策略开关，前端提供“拒绝并封禁”操作，后端审核接口也支持 `ban_user` 参数。

二维码限制：

- 前端上传限制为 `PNG/JPG/WebP` 且不超过 `300KB`。
- 后端配置校验允许空值、`http/https` 图片地址和 `data:image/png|jpeg|jpg|webp;base64,...`，最大 `512KB`。

## 风控限额

自助充值限额是配置项，没有内置默认值。管理员必须手动配置 `SelfServeTopUpSingleMaxAmount` 和 `SelfServeTopUpDailyMaxAmount` 后，用户才可以使用自助充值；每日限额必须大于或等于单笔限额。为了按当前运营要求限制用户充值，建议配置为：

- 每人单笔最高 `199.99` 元。
- 每人每日最高 `499.99` 元。

用户充值弹窗、管理员编辑弹窗和后端校验都会使用当前配置值展示和限制金额。未完整配置限额时，即使总开关和收款码已配置，前端也不会展示自助充值入口，后端也会拒绝预览、提交和编辑请求。

每日限额按用户、自然日、本地服务端时区统计，包含待审核和已通过的自助充值记录，已拒绝记录不计入每日已用额度。后端会在预览、提交和管理员编辑时强制校验，前端限制仅作为交互提示。

## 用户流程

1. 用户进入充值页，选择“支付宝自助”或“微信自助”。
2. 系统展示对应收款码、单笔/每日限额、今日剩余额度和风险提示。
3. 用户扫码付款后填写：
   - 充值金额，单位为人民币元。
   - 交易订单号，必须填写微信或支付宝账单里的交易订单号，不是商户订单号。
4. 用户确认“已完成付款，并承诺金额和交易订单号真实有效”后提交。
5. 系统立即创建一笔成功充值账单和一条待审核记录，并实时增加用户余额。

风险提示必须保留：

- 提交后余额会实时到账。
- 虚假填写、重复提交或金额不符会被拒绝、扣回余额，账户可能被封禁。
- 自助充值拒绝后概不退款。

## 到账额度计算

用户填写的是人民币充值金额。系统按当前充值价格和用户充值分组倍率换算到账额度：

```text
到账额度 = 充值金额 / Price / TopupGroupRatio(user.group) * QuotaPerUnit
```

结果向下取整。若系统价格、额度倍率或充值分组倍率配置错误，后端会拒绝提交。

## 管理员审核

入口：`账单管理`。

账单管理新增：

- 支付方式筛选：`自助充值`。
- 账单视图：`待审核自助充值`。
- 列字段：交易订单号、申报金额、审核状态。
- 待审核记录操作：
  - `通过`：确认真实到账后将审核状态改为 `approved`，并在通过时发放邀请返佣。
  - `编辑`：纠正申报金额、交易订单号和审核备注。保存后系统会按新金额重新计算到账额度，并按差额调整用户余额。
  - `拒绝`：将审核状态改为 `rejected`，扣回该订单已到账余额，创建退款/扣回记录，并把充值账单标记为已退款。
  - `拒绝并封禁`：在拒绝扣回的同时禁用用户账户并失效用户缓存/令牌缓存。

已拒绝订单再次执行“拒绝并封禁”不会重复扣款，也不会重复创建扣回记录，只会补执行封禁。

## API

用户接口：

- `POST /api/user/self-serve/preview`
  - 请求：`declared_money`
  - 返回：预计到账额度、今日已用金额、今日剩余金额、当前配置限额。
- `POST /api/user/self-serve/pay`
  - 请求：`payment_method`、`declared_money`、`transaction_no`
  - 支付方式：`alipay_self_serve`、`wxpay_self_serve`
  - 成功后立即入账，并生成待审核记录。

管理员接口：

- `POST /api/user/topup/self-serve/approve`
  - 请求：`trade_no`、`reason`
- `POST /api/user/topup/self-serve/update`
  - 请求：`trade_no`、`declared_money`、`transaction_no`、`reason`
- `POST /api/user/topup/self-serve/reject`
  - 请求：`trade_no`、`reason`、`ban_user`

账单查询：

- `payment_method=self_serve`：筛选自助充值账单。
- `audit_status=pending|approved|rejected`：筛选自助充值审核状态。

## 数据表

自助充值审核记录存储在 `self_serve_top_up_audits`：

- `top_up_id`：关联充值账单。
- `trade_no`：系统充值订单号。
- `transaction_no`：用户填写的微信/支付宝交易订单号，全局唯一。
- `payment_method`：`alipay_self_serve` 或 `wxpay_self_serve`。
- `declared_money`：用户申报金额。
- `credited_quota`：本次已实时到账额度。
- `status`：`pending`、`approved`、`rejected`。
- `admin_reason`、`auditor_id`、`reviewed_time`：管理员审核信息。

## 运营建议

- 管理员应每天至少核对一次待审核自助充值。
- 审核时以微信/支付宝实际到账记录为准，核对交易订单号和金额。
- 对金额填错但确实到账的记录，优先使用“编辑”纠正后再通过。
- 对未到账、重复提交、恶意虚假提交的记录，使用“拒绝”或“拒绝并封禁”。
