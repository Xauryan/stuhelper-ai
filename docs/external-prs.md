# 外部 PR 和补丁记录

本文档记录不是通过常规上游 release 同步引入的 PR 或补丁。

## 官方支付订阅购买与 classic 订阅展示控制

- 来源：StuHelper AI 本地补丁。
- 相关参考：
  - 支付宝官方电脑网站支付 / 手机网站支付文档。
  - 微信支付 API v3 Native 支付 / H5 支付 / 支付成功通知文档。
  - `docs/official-cn-payments.md` 中记录的官方支付宝和微信支付接入约束。
- 本地导入日期：2026-05-14
- 更新日期：2026-05-15，补齐微信支付官方订阅购买；继续补齐微信支付官方查询、
  关闭、退款、退款查询，以及用户退款申请/管理员审批。
- 导入方式：按本地官方支付架构手工实现；没有套用支付宝当面付或易支付实现。
- 本地涉及文件：
  - `controller/subscription_payment_alipay_official.go`
  - `controller/subscription.go`
  - `controller/subscription_test.go`
  - `controller/topup_official.go`
  - `controller/topup_test.go`
  - `controller/topup.go`
  - `model/subscription.go`
  - `model/main.go`
  - `model/payment_method_guard_test.go`
  - `model/topup.go`
  - `router/api-router.go`
  - `web/classic/src/components/topup/RechargeCard.jsx`
  - `web/classic/src/components/topup/SubscriptionPlansCard.jsx`
  - `web/classic/src/components/topup/modals/SubscriptionPurchaseModal.jsx`
  - `web/classic/src/components/topup/modals/TopupBillingTable.jsx`
  - `web/classic/src/components/topup/modals/TopupHistoryModal.jsx`
  - `web/classic/src/components/topup/modals/topupHistoryUtils.mjs`
  - `web/classic/src/components/topup/rechargeTabs.js`
  - `web/classic/src/components/topup/subscriptionPaymentMethods.js`
  - `web/classic/src/components/topup/subscriptionPlanDisplay.js`
  - `web/classic/src/pages/Billing/index.jsx`
  - `web/classic/src/components/table/subscriptions/**`
  - `docs/official-cn-payments.md`
  - `docs/fork-maintenance.md`

### 原因

官方支付宝和微信支付官方此前都先覆盖额度充值，订阅购买流程需要补齐官方支付
能力。用户在只启用官方支付、不启用易支付时，订阅套餐购买流程不应提示未开启
在线支付；同时 classic 钱包页默认切到订阅，订阅卡片把第一个套餐默认标记为
“推荐”，不利于后台手动运营。

### 本地行为

- 新增 `POST /api/subscription/alipay-official/pay`，用于订阅套餐官方支付宝支付。
- 新增 `POST /api/subscription/wechat-pay-official/pay`，用于订阅套餐微信支付官方
  支付。
- 支付宝订阅订单使用官方电脑网站支付或手机网站支付，移动端使用
  `alipay.trade.wap.pay`，电脑端使用 `alipay.trade.page.pay`。
- 微信支付官方订阅订单复用微信支付 API v3，电脑端使用
  `/v3/pay/transactions/native` 并展示扫码二维码，移动端使用
  `/v3/pay/transactions/h5` 并跳转 `h5_url`；如果微信返回 H5 无权限或产品
  不可用，则降级为 Native `code_url` 二维码，其他签名、证书、参数错误仍按
  失败处理。
- classic 钱包页和订阅购买页展示微信 Native 二维码后会轮询本地充值账单状态，
  等待微信支付成功通知完成入账后自动关闭弹窗并刷新用户额度或订阅状态；轮询
  使用 `POST /api/user/wechat-pay/official/status`，后端可按微信订单查询结果
  主动补齐成功入账。
- 订阅价格按套餐美元金额乘以对应官方支付单价换算成人民币，并按进一法保留两位
  小数提交给支付平台；支付宝使用 `AlipayOfficialUnitPrice`，微信支付官方使用
  `WechatPayOfficialUnitPrice`。
- 如果站点以服务商/第三方代理身份代商户调用支付宝接口，`AlipayOfficialAppAuthToken`
  会被电脑网站支付、手机网站支付、查询、关闭、退款和退款查询共用；该授权
  Token 作为敏感配置保存，不从 `/api/option/` 回显，只暴露
  `AlipayOfficialAppAuthTokenConfigured`。
- 支付宝官方异步通知会先识别订阅订单；如果命中订阅订单，则完成订阅并写入与
  充值账单兼容的支付记录；如果不是订阅订单，再走普通额度充值完成逻辑。
- 微信支付官方异步通知会先识别订阅订单；如果命中订阅订单，则校验微信支付回调
  金额与订阅订单金额一致后完成订阅；如果不是订阅订单，再走普通额度充值完成逻辑。
- 订阅订单创建时同步写入充值账单待支付占位记录，支付成功或过期时更新同一条
  账单；官方订阅待回调订单也能在充值账单中按订单号查到。
- 订阅写入的充值账单必须保留 `PaymentProvider=alipay_official`，并执行支付
  方式和支付提供方一致性保护；微信支付官方订阅保留
  `PaymentProvider=wxpay_official`，避免不同支付网关订单串用。
- classic 充值账单把 `SUB`、`ALIPAYSUB`、`WXSUB` 前缀且充值额度为 0 的账单
  识别为“订阅套餐”，官方订阅不会显示成普通 0 额度充值。
- classic 订阅购买页会在官方支付宝或微信支付官方完整配置时展示“支付宝”或
  “微信”支付方式，不再依赖易支付 `enable_online_topup` 开关，也不会把官方
  支付混入易支付子渠道。
- classic 订阅购买流程改为先选套餐、再选支付方式，页面即时展示该方式换算后的
  实付人民币，确认弹窗只展示套餐、支付方式和应付金额。
- classic 钱包页同时存在“额度充值”和“订阅套餐”时，默认进入“额度充值”，且
  “额度充值”位于“订阅套餐”左侧。
- 订阅套餐新增 `recommended` 字段。classic 后台“订阅管理”中可手动开关
  “推荐”，用户侧订阅卡片只有在该字段为 `true` 时才显示“推荐”标签和高亮边框；
  不再默认把第一个套餐标记为推荐。
- 微信支付官方额度充值和订阅账单补齐管理员查询、关闭、退款和退款查询：
  查询使用 `GET /v3/pay/transactions/out-trade-no/{out_trade_no}?mchid=...`，
  关闭使用
  `POST /v3/pay/transactions/out-trade-no/{out_trade_no}/close`，退款使用
  `POST /v3/refund/domestic/refunds`，退款查询使用
  `GET /v3/refund/domestic/refunds/{out_refund_no}`。只有微信侧关闭 2xx 后才把
  本地待支付订单标记为已超时。
- 微信退款返回 `PROCESSING` 时本地退款记录保持 `pending`，等待退款查询或
  微信退款通知确认；`SUCCESS` 才同步本地退款成功和订阅订单状态，
  `REFUNDCLOSE`、`CLOSED` / `REFUND.CLOSED`、`ABNORMAL` / `REFUND.ABNORMAL`
  会回滚本地预留。
- 用户可在自己的 classic 充值账单中对官方支付宝/微信支付成功或部分退款订单
  提交退款申请，申请时必须填写原因。后端按“未使用部分”计算当前最多可退金额：
  额度充值按当前用户余额和该订单剩余额度取较小值折算人民币，订阅按已过时间和
  已用额度中更严格的比例计算未使用金额。管理员可审批通过后调用官方退款 API，
  也可拒绝申请；管理员直接退款仍可选择全额退款。
- 订阅退款不会扣用户钱包余额。部分退款只同步订阅订单为 `partial_refunded`；
  全额退款会取消订阅实例、按分组规则回退用户分组，并按 `subscription` 来源和
  订阅订单 ID 冲回返佣。额度充值退款仍按 `topup` 来源冲回返佣。
- classic 新增独立账单页 `/console/billing`，侧边栏位置在“钱包管理”和“个人设置”
  之间。普通用户只看自己的充值、订阅和退款记录；管理员在同一页面看全平台账单
  并处理查询、关闭、退款、审批和拒绝退款。
- 钱包页“充值账单”弹窗和独立账单页共用 `TopupBillingTable`，避免两套表格逻辑
  分叉。独立账单页提供“全部账单”和“待处理退款”两个视图。
- 独立账单页视觉结构对齐 classic 使用日志：页面外层保留与 `/console/log`
  相同的顶部间距，外层使用 `CardPro type2`，表格使用 `CardTable`，筛选、
  视图选择、紧凑列表切换和底部分页位置与使用日志一致；钱包弹窗保留轻量
  内嵌表格模式。
- 待处理退款视图通过 `pending_refund=true` 传给 `/api/user/topup/self` 或
  `/api/user/topup`，后端使用待审核退款申请子查询过滤账单，分页总数和搜索结果
  都来自数据库过滤结果，不在前端做客户端过滤。

### 验证

```powershell
go test ./controller ./model ./service -count=1
bun web/classic/src/components/topup/subscriptionPaymentMethods.test.mjs
bun web/classic/src/components/topup/subscriptionPaymentDisplay.test.mjs
bun web/classic/src/components/topup/rechargeAmountDisplay.test.mjs
bun web/classic/src/components/topup/subscriptionPlanDisplay.test.mjs
bun web/classic/src/components/topup/rechargeTabs.test.mjs
bun web/classic/src/components/topup/modals/topupHistoryUtils.test.mjs
go test ./service -run "WechatPayOfficial.*(Query|Close|Refund)|DecodeWechatPayOfficial" -count=1
go test ./controller -run "WechatPayOfficial|AlipayOfficialRefund|AlipayOfficialQuery|AlipayOfficialClose" -count=1
go test ./model -run "TopUpQueryOptions|SearchAllTopUps|CreateTopUpRefundRequest" -count=1
Set-Location web/classic; bun run build
git diff --check
```

### 未来上游同步检查点

每次同步上游 release 时：

- 不要把官方支付宝或微信支付官方订阅支付回退成易支付子渠道，也不要把支付宝
  订阅回退成支付宝当面付。
- 如果上游新增订阅支付能力，需要比较订单字段、回调幂等、支付提供方保护和金额
  换算规则，确保本地官方支付路径仍独立且可用。
- 保留 classic 钱包页默认“额度充值”的 tab 顺序，以及订阅推荐由后台手动控制的
  运营能力。
- 如果上游新增订阅套餐展示字段，确认不会覆盖或忽略本地 `recommended` 字段。
- 不要把微信退款 `PROCESSING` 当成最终成功；只能在退款查询或通知确认
  `SUCCESS` 后同步订阅退款状态。退款查询/通知确认 `CLOSED`、`REFUND.CLOSED`
  或 `ABNORMAL` 时，要回滚本地预留和订阅退款状态。
- 保留用户退款申请和管理员审批的两段式流程，普通用户不能直接触发官方支付
  退款 API。
- 保留 `/console/billing` 独立账单页、侧边栏顺序和 `pending_refund=true`
  后端过滤，不能把待处理退款改回前端分页后的客户端过滤。

## QuantumNous/new-api#3288 - 邀请充值返佣

- 来源 PR：https://github.com/QuantumNous/new-api/pull/3288
- 关联 issue：
  - https://github.com/QuantumNous/new-api/issues/128
  - https://github.com/QuantumNous/new-api/issues/187
  - https://github.com/QuantumNous/new-api/issues/1852
- 审查时的上游状态：open，尚未合并。
- 本地导入日期：2026-05-13
- 导入方式：阅读 PR 描述、关联 issue 和 CodeRabbit 评审后手工重写核心行为；
  没有直接套用上游 diff，且前端配置只落在 classic 前端。
- 本地涉及文件：
  - `common/constants.go`
  - `model/referral_commission.go`
  - `model/referral_commission_test.go`
  - `model/user.go`
  - `model/topup.go`
  - `model/subscription.go`
  - `model/main.go`
  - `model/option.go`
  - `model/payment_method_guard_test.go`
  - `model/task_cas_test.go`
  - `controller/topup.go`
  - `controller/user.go`
  - `router/api-router.go`
  - `web/classic/src/components/settings/OperationSetting.jsx`
  - `web/classic/src/pages/Setting/Operation/SettingsCreditLimit.jsx`
  - `web/classic/src/components/table/users/modals/EditUserModal.jsx`
  - `web/classic/src/components/topup/InvitationCard.jsx`
  - `web/classic/src/i18n/locales/*.json`

### 原因

现有邀请奖励主要是注册时一次性发放额度，容易被批量小号注册利用，也缺少
对持续邀请高质量用户的激励。关联 issue 希望把奖励延后到被邀请用户实际充值
或购买订阅后，按支付金额比例返佣，并可限制前 N 次充值。

### 审查记录

- PR #3288 的总体思路可取：新增全局返佣配置、邀请人单用户比例覆盖、支付完成
  时写返佣记录、邀请人获得可划转的 `aff_quota`。
- 未照搬上游代码，主要原因是当前 StuHelper AI 已有本地支付网关、订阅、
  classic 前端和 fork 维护规则，需要按本地结构重写。
- 采纳 CodeRabbit 对 #3288 的有效反馈：
  - 支付完成后的邀请人一次性奖励解锁和充值返佣不能作为 best-effort 写入；
    本地在各充值和订阅完成事务内调用 `CreditInviteRewardsAfterPaymentTx`，
    失败会回滚支付完成写库。
  - 返佣幂等键不能只依赖 `top_up_id`，否则订阅场景会出错；本地使用
    `source_type + source_id + invitee_id + payment_method`，其中订阅使用
    `subscription_orders.id` 作为 `source_id`。
  - 返佣历史必须分页查询；本地 `GET /api/user/aff/commissions` 使用通用分页。
  - `ReferralCommissionPercent` 和 `ReferralCommissionMaxRecharges` 必须先校验
    再保存，避免数据库保存值和运行时生效值不一致。
  - 只有真实新增返佣记录时才写邀请人返佣日志。

### 本地行为

- 新增 `referral_commissions` 表，记录邀请人、被邀请人、来源类型、来源 ID、
  支付方式、支付金额、返佣额度、返佣比例和创建时间。
- 新增全局选项：
  - `ReferralCommissionEnabled`
  - `ReferralCommissionPercent`
  - `ReferralCommissionMaxRecharges`
- `QuotaForInvitee` 控制新用户使用邀请码奖励；只要大于 0，被邀请用户注册后
  实时到账，不受返佣开关影响。
- `QuotaForInviter` 控制邀请人一次性奖励；默认注册后实时进入邀请人的
  `aff_quota`，开启 `InviterRewardAfterPaymentEnabled` 后改为被邀请用户首次
  充值或购买订阅成功时解锁到账；解锁额度使用注册时写入的
  `inviter_reward_quota` 快照，不受后续全局配置调整影响。
- `ReferralCommissionEnabled=true` 时，充值返佣与一次性邀请奖励独立叠加；
  返佣不会替代或关闭 `QuotaForInviter` / `QuotaForInvitee`。
- `users` 表新增 `invitee_reward_quota`、`inviter_reward_quota`、
  `inviter_reward_unlocked` 和 `inviter_reward_unlocked_by_payment`，用于展示
  被邀请用户实际获得的邀请码奖励、延迟邀请人奖励幂等，以及区分一次性邀请人
  奖励是否由首充/订阅支付解锁；首次新增延迟奖励状态字段时，已有邀请关系会
  标记为已处理，避免老用户首充后被补发，后续启动不会清除新产生的待解锁奖励。
  字段上线前的历史邀请关系没有被邀请人奖励快照，管理页中可能显示为 0。
- 被邀请关系的唯一来源是 `users.inviter_id`。密码注册、统一 OAuth、旧 GitHub
  和 LinuxDO 注册都必须把解析出的邀请人 ID 写入新用户记录；只增加邀请人的
  `aff_count` / `aff_quota` 不能代表关系已持久化。2026-05-15 修复了
  `Insert` / `InsertWithTx` 忽略传入 `inviterId` 的问题，并让后台邀请管理在邀请人
  用户行缺失时仍按被邀请用户的 `inviter_id` 展示审计关系。受旧缺陷影响的生产
  历史数据若 `users.inviter_id = 0`，需要从注册请求、系统日志或邀请奖励日志中
  人工回填后，才能出现在邀请管理并参与后续返佣。
- 返佣覆盖的支付完成路径包括 Stripe、Creem、Epay、Waffo、Waffo Pancake、
  支付宝官方、微信支付官方、管理员补单和订阅订单完成。
- 返佣额度按 `recharge_amount * QuotaPerUnit * rate / 100` 计算，向下取整；
  邀请人单用户覆盖比例优先于全局比例。新返佣记录写入 `recharge_sequence`
  用于审计和最大返佣次数判断；历史记录没有序号时按已有记录数量兜底，避免
  旧库迁移时因默认序号重复而失败。
- 官方支付退款会记录 `refunded_recharge_amount` 和 `refunded_commission_quota`
  作为返佣冲销字段；用户侧返佣记录和 classic 邀请管理页展示净支付金额与净返佣。
  退款失败会恢复冲销字段和邀请人收益；充值全额退款会回滚由首充触发的一次性
  邀请人奖励解锁，即时发放的一次性邀请奖励不会被退款撤回，并且全额退款订单
  不再计为邀请管理中的有效首充。
- classic 运维设置页新增返佣全局开关、比例、最大次数和邀请人一次性奖励
  首充后到账开关；classic 用户编辑页新增单用户返佣比例覆盖；classic 充值页
  邀请卡展示分页返佣记录。
- classic 管理员菜单新增“邀请管理”，通过 `GET /api/user/referrals` 分页展示所有
  邀请关系、被邀请用户奖励、被邀请用户是否已首充或购买订阅、邀请人一次性奖励
  解锁状态和返佣汇总。该接口仅管理员可访问，支持按邀请人/被邀请用户关键词
  搜索，并支持筛选邀请人一次性奖励的已解锁/待首充状态；返佣明细在弹窗打开时
  通过 `GET /api/user/referrals/:invitee_id/commissions` 按被邀请用户分页加载。

### 验证

```powershell
go test ./model -run "ReferralCommission|CompleteEpayTopUp|PaymentGuard|SubscriptionOrder|ManualCompleteTopUp|Refund|Redemption" -count=1
go test ./controller -run "TopUp|Epay|Stripe|Creem|Official|Option|User" -count=1
Set-Location web/classic; bun run build
```

### 未来上游同步检查点

每次同步上游 release 时：

- 检查上游是否已经合并 PR #3288 或等价邀请返佣实现。
- 如果上游合并等价实现，比较字段名、幂等键、事务边界和 classic 前端配置位置，
  不要把本地官方支付、订阅和 classic 默认前端行为冲掉。
- 保留本地回归测试，除非上游已有等价覆盖并且本地支付路径全部确认兼容。

## QuantumNous/new-api#4564 - 订阅套餐模型限制

- 来源 PR：https://github.com/QuantumNous/new-api/pull/4564
- 相关参考 PR：https://github.com/QuantumNous/new-api/pull/4246
- 审查时的上游状态：open，尚未合并。
- 本地导入日期：2026-05-13
- 导入方式：手工移植核心行为，并按 StuHelper AI 当前后端与 classic 前端结构重写。
- 本地涉及文件：
  - `model/subscription.go`
  - `model/main.go`
  - `controller/subscription.go`
  - `service/billing_session.go`
  - `model/subscription_model_limits_test.go`
  - `controller/subscription_test.go`
  - `service/task_billing_test.go`
  - `model/task_cas_test.go`
  - `web/classic/src/components/table/subscriptions/**`
  - `web/classic/src/components/topup/SubscriptionPlansCard.jsx`
  - `web/classic/src/helpers/subscription.js`
  - `web/classic/src/i18n/locales/*.json`

### 原因

订阅套餐此前只能限制额度、周期、购买次数和升级分组，不能限制该套餐可用于
哪些模型。管理员如果希望某个订阅仅覆盖低成本或指定模型，只能依赖外部约定，
实际预扣费链路仍会优先消耗任何可用订阅额度。

### 审查记录

- PR #4564 的原始实现覆盖了后端字段、预扣费过滤和 classic 前端配置入口，但
  本地没有直接套用 diff，而是按当前 StuHelper AI 结构补齐 classic 前端和
  本地测试。
- 采纳 CodeRabbit 对 #4564 的有效反馈：
  - 保存前统一 trim、去空值、去重，避免 UI 数量和后端实际限制不一致。
  - 编辑表单只有在 `model_limits_enabled=true` 时回填模型列表，避免残留 CSV
    在再次保存时被意外重新启用。
  - 前端模型选项使用管理员可见的 `/api/channel/models_enabled`，不使用用户侧
    `/api/user/models`。
  - 模型列表加载失败时阻止提交，不能把失败静默当成“空列表、不限制”保存。
- PR #4246 是同类“订阅套餐允许分组”限制，本次只借鉴其预扣费过滤和错误分类
  模式，没有扩大实现允许分组功能。

### 本地行为

- 订阅套餐新增 `model_limits_enabled` 和 `model_limits` 字段。
- 管理员创建或更新套餐时，模型限制会被规范化为去重 CSV；空列表会保存为
  `model_limits_enabled=false` 和空字符串。
- 订阅预扣费时：
  - 未设置模型限制的套餐继续支持所有模型。
  - 设置模型限制的套餐只在请求模型命中限制列表时参与扣费。
  - 如果存在活跃订阅但没有任何套餐允许该模型，返回
    `no subscription allows model <model>`，并在服务层归类为额度不足 / 不可用，
    使 `subscription_first` 能回退钱包，`subscription_only` 返回 403。
- classic 管理界面新增模型限制配置和列表展示；用户购买视图展示
  该套餐的可用模型数量。

### 验证

本节记录实现时应持续验证的命令：

```powershell
go test ./model ./controller ./service -count=1
Set-Location web/classic; bun run build
```

### 未来上游同步检查点

每次同步上游 release 时：

- 检查上游是否已经合并 PR #4564 或等价实现。
- 如果上游合并 #4246 的允许分组功能，单独评估是否需要移植；不要把本地模型
  限制补丁和分组限制补丁混在同一次同步里。
- 对比上游错误消息、字段名和前端模型来源，确保本地保留加载失败阻止保存、
  规范化保存和 classic 展示行为。

## QuantumNous/new-api#2787 - 密码登录和密码注册开关前台生效

- 来源 PR：https://github.com/QuantumNous/new-api/pull/2787
- 审查时的上游状态：closed，未合并。
- 本地导入日期：2026-05-13
- 导入方式：手工移植并适配 StuHelper AI 当前 classic 前端结构。
- 本地涉及文件：
  - `controller/misc.go`
  - `controller/misc_test.go`
  - `web/classic/src/components/auth/LoginForm.jsx`
  - `web/classic/src/components/auth/RegisterForm.jsx`
  - `web/classic/src/i18n/locales/*.json`

### 原因

现有后端已经有 `PasswordLoginEnabled` 和 `PasswordRegisterEnabled` 管理项，
并在登录、注册接口上执行最终拦截；后台设置页也能修改这两个选项。但
`/api/status` 未向前台暴露对应开关，classic 前台登录 / 注册页
也没有根据开关隐藏密码表单入口。

### 审查记录

- PR #2787 的原始 diff 面向旧 `web/src` 前端路径，不能直接套用到当前
  StuHelper AI 的 `web/classic` 前端结构。
- 本地保留现有后端接口拦截作为最终安全边界，只新增状态字段和前端展示逻辑。
- 状态接口使用 `password_login` / `password_register` 字段；前端对缺失字段按
  启用处理，以兼容旧缓存或旧服务端状态。
- classic 前端在没有任何可见登录 / 注册方式时显示联系管理员文案，避免空白
  认证页。

### 本地行为

- `PasswordLoginEnabled=false` 时：
  - `/api/status` 返回 `password_login: false`。
  - classic 登录页隐藏密码登录表单和密码登录入口。
  - Passkey、OAuth、WeChat、Telegram 等登录方式继续按各自开关显示。
- `PasswordRegisterEnabled=false` 时：
  - `/api/status` 返回 `password_register: false`。
  - classic 注册页隐藏用户名 / 密码注册表单和密码注册入口。
  - 第三方注册入口继续按各自开关显示。

### 验证

```powershell
go test ./controller -run TestGetStatusIncludesPasswordAuthSwitches -count=1
Set-Location web/classic; bun run build
```

### 未来上游同步检查点

每次同步上游 release 时：

- 检查上游是否已经合并 PR #2787 或等价修复。
- 如果上游吸收了等价行为，确认字段名和前端语义是否一致，再协调本地补丁。
- 保留 `controller/misc_test.go` 中的状态字段回归覆盖，除非上游已有等价测试。

## QuantumNous/new-api#4719 - 修复 auto 分组 token 返回 403

- 来源 PR：https://github.com/QuantumNous/new-api/pull/4719
- 审查时的上游状态：open，尚未合并。
- 本地导入日期：2026-05-12
- 导入方式：手工移植最终有效的后端行为。
- 本地涉及文件：
  - `middleware/auth.go`
  - `middleware/auth_test.go`

### 原因

`TokenAuth` 会在请求到达下游 auto 分组解析之前，拒绝配置为 `auto` 分组的
API token。失败信息为：

```text
无权访问 auto 分组
```

`auto` 是一个伪分组。它应该通过中间件中的 token 分组校验，然后在后续流程中
通过基于用户可用分组的 auto 分组选择逻辑解析。

### 审查记录

- PR 描述指出，过早执行的 `UserUsableGroups` 检查是 403 的来源。
- CodeRabbit 对最终 diff 没有给出可执行的 inline comment。
- CodeRabbit 曾在中间版本中指出前端表单重置问题；该问题来自引入
  `firstGroup` 的中间实现，而该前端行为后来已在 PR 历史中回滚。
- 最终 PR 没有保留实质性的前端行为变更，所以本地只导入后端鉴权行为，并
  添加测试。
- 后端安全边界保持不变：非 `auto` 的显式 token 分组仍必须存在于该用户的
  可用分组中。

### 本地行为

- `token.Group == "auto"` 时，跳过中间件中的可用分组成员检查。
- `token.Group != "auto"` 时，仍要求该分组存在于
  `service.GetUserUsableGroups(userGroup)`。
- 下游渠道和模型选择使用 `service.GetUserAutoGroup(userGroup)` 解析 `auto`，
  该函数会按用户真实可用分组过滤。

### 验证

```powershell
go test ./middleware -run TestTokenAuthAllowsAutoPseudoGroup -count=1
go test ./middleware -count=1
```

已观察到目标回归测试在修复前因 `auto` 返回 403 而失败，并在修复后通过。

### 未来上游同步检查点

每次同步上游 release 时：

- 检查上游是否已经合并 PR #4719 或等价修复。
- 如果上游已经吸收该行为，则将本地补丁与上游实现协调。
- 除非上游加入等价覆盖，否则保留本地回归测试。
