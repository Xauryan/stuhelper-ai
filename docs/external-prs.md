# 外部 PR 和补丁记录

本文档记录不是通过常规上游 release 同步引入的 PR 或补丁。

## 官方支付宝订阅支付与 classic 订阅展示控制

- 来源：StuHelper AI 本地补丁。
- 相关参考：
  - 支付宝官方电脑网站支付 / 手机网站支付文档。
  - `docs/official-cn-payments.md` 中记录的官方支付宝和微信支付接入约束。
- 本地导入日期：2026-05-14
- 导入方式：按本地官方支付架构手工实现；没有套用支付宝当面付或易支付实现。
- 本地涉及文件：
  - `controller/subscription_payment_alipay_official.go`
  - `controller/subscription.go`
  - `controller/subscription_test.go`
  - `controller/topup_official.go`
  - `controller/topup_test.go`
  - `model/subscription.go`
  - `model/main.go`
  - `model/payment_method_guard_test.go`
  - `router/api-router.go`
  - `web/classic/src/components/topup/RechargeCard.jsx`
  - `web/classic/src/components/topup/SubscriptionPlansCard.jsx`
  - `web/classic/src/components/topup/modals/SubscriptionPurchaseModal.jsx`
  - `web/classic/src/components/topup/rechargeTabs.js`
  - `web/classic/src/components/topup/subscriptionPaymentMethods.js`
  - `web/classic/src/components/topup/subscriptionPlanDisplay.js`
  - `web/classic/src/components/table/subscriptions/**`
  - `docs/official-cn-payments.md`
  - `docs/fork-maintenance.md`

### 原因

官方支付宝此前只覆盖额度充值。用户在只启用官方支付宝、不启用易支付时，
订阅套餐购买流程仍会提示未开启在线支付；同时 classic 钱包页默认切到订阅，
订阅卡片把第一个套餐默认标记为“推荐”，不利于后台手动运营。

### 本地行为

- 新增 `POST /api/subscription/alipay-official/pay`，用于订阅套餐官方支付宝支付。
- 订阅订单使用官方支付宝电脑网站支付或手机网站支付，移动端使用
  `alipay.trade.wap.pay`，电脑端使用 `alipay.trade.page.pay`。
- 订阅价格按套餐美元金额乘以 `AlipayOfficialUnitPrice` 换算成人民币，并按
  进一法保留两位小数提交给支付宝。
- 支付宝官方异步通知会先识别订阅订单；如果命中订阅订单，则完成订阅并写入与
  充值账单兼容的支付记录；如果不是订阅订单，再走普通额度充值完成逻辑。
- 订阅完成写入的充值账单必须保留 `PaymentProvider=alipay_official`，并执行
  支付方式和支付提供方一致性保护，避免不同支付网关订单串用。
- classic 订阅购买弹窗会在官方支付宝完整配置时展示“支付宝”支付方式，不再依赖
  易支付 `enable_online_topup` 开关。
- classic 钱包页同时存在“额度充值”和“订阅套餐”时，默认进入“额度充值”，且
  “额度充值”位于“订阅套餐”左侧。
- 订阅套餐新增 `recommended` 字段。classic 后台“订阅管理”中可手动开关
  “推荐”，用户侧订阅卡片只有在该字段为 `true` 时才显示“推荐”标签和高亮边框；
  不再默认把第一个套餐标记为推荐。

### 验证

```powershell
go test ./controller ./model ./service -count=1
bun web/classic/src/components/topup/subscriptionPaymentMethods.test.mjs
bun web/classic/src/components/topup/rechargeAmountDisplay.test.mjs
bun web/classic/src/components/topup/subscriptionPlanDisplay.test.mjs
bun web/classic/src/components/topup/rechargeTabs.test.mjs
bun web/classic/src/components/topup/modals/topupHistoryUtils.test.mjs
Set-Location web/classic; bun run build
git diff --check
```

### 未来上游同步检查点

每次同步上游 release 时：

- 不要把官方支付宝订阅支付回退成易支付子渠道或支付宝当面付。
- 如果上游新增订阅支付能力，需要比较订单字段、回调幂等、支付提供方保护和金额
  换算规则，确保本地官方支付宝路径仍独立且可用。
- 保留 classic 钱包页默认“额度充值”的 tab 顺序，以及订阅推荐由后台手动控制的
  运营能力。
- 如果上游新增订阅套餐展示字段，确认不会覆盖或忽略本地 `recommended` 字段。

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
  classic/default 双前端和 fork 维护规则，需要按本地结构重写。
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
- `users` 表新增 `inviter_reward_quota` 和 `inviter_reward_unlocked`，用于延迟
  邀请人奖励幂等；首次新增这两个字段时，已有邀请关系会标记为已处理，避免
  老用户首充后被补发，后续启动不会清除新产生的待解锁奖励。
- 返佣覆盖的支付完成路径包括 Stripe、Creem、Epay、Waffo、Waffo Pancake、
  支付宝官方、微信支付官方、管理员补单和订阅订单完成。
- 返佣额度按 `recharge_amount * QuotaPerUnit * rate / 100` 计算，向下取整；
  邀请人单用户覆盖比例优先于全局比例。
- classic 运维设置页新增返佣全局开关、比例、最大次数和邀请人一次性奖励
  首充后到账开关；classic 用户编辑页新增单用户返佣比例覆盖；classic 充值页
  邀请卡展示分页返佣记录。

### 验证

```powershell
go test ./model -run "ReferralCommission|CompleteEpayTopUp|PaymentGuard|SubscriptionOrder" -count=1
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
- 导入方式：手工移植核心行为，并按 StuHelper AI 当前后端与 default / classic 双前端结构重写。
- 本地涉及文件：
  - `model/subscription.go`
  - `model/main.go`
  - `controller/subscription.go`
  - `service/billing_session.go`
  - `model/subscription_model_limits_test.go`
  - `controller/subscription_test.go`
  - `service/task_billing_test.go`
  - `model/task_cas_test.go`
  - `web/default/src/features/subscriptions/**`
  - `web/default/src/features/wallet/components/subscription-plans-card.tsx`
  - `web/default/src/i18n/locales/*.json`
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
  本地没有直接套用 diff，而是按当前 StuHelper AI 结构补齐 default / classic
  双前端和本地测试。
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
- default 与 classic 管理界面新增模型限制配置和列表展示；用户购买视图展示
  该套餐的可用模型数量。

### 验证

本节记录实现时应持续验证的命令：

```powershell
go test ./model ./controller ./service -count=1
Set-Location web/default; bun run typecheck
Set-Location web/default; bun run i18n:sync
Set-Location web/classic; bun run build
```

### 未来上游同步检查点

每次同步上游 release 时：

- 检查上游是否已经合并 PR #4564 或等价实现。
- 如果上游合并 #4246 的允许分组功能，单独评估是否需要移植；不要把本地模型
  限制补丁和分组限制补丁混在同一次同步里。
- 对比上游错误消息、字段名和前端模型来源，确保本地保留加载失败阻止保存、
  规范化保存和双前端展示行为。

## QuantumNous/new-api#2787 - 密码登录和密码注册开关前台生效

- 来源 PR：https://github.com/QuantumNous/new-api/pull/2787
- 审查时的上游状态：closed，未合并。
- 本地导入日期：2026-05-13
- 导入方式：手工移植并适配 StuHelper AI 当前 default / classic 双前端结构。
- 本地涉及文件：
  - `controller/misc.go`
  - `controller/misc_test.go`
  - `web/default/src/features/auth/components/oauth-providers.tsx`
  - `web/default/src/features/auth/sign-in/components/user-auth-form.tsx`
  - `web/default/src/features/auth/sign-up/components/sign-up-form.tsx`
  - `web/default/src/features/auth/types.ts`
  - `web/default/src/i18n/locales/*.json`
  - `web/classic/src/components/auth/LoginForm.jsx`
  - `web/classic/src/components/auth/RegisterForm.jsx`
  - `web/classic/src/i18n/locales/*.json`

### 原因

现有后端已经有 `PasswordLoginEnabled` 和 `PasswordRegisterEnabled` 管理项，
并在登录、注册接口上执行最终拦截；后台设置页也能修改这两个选项。但
`/api/status` 未向前台暴露对应开关，default 和 classic 前台登录 / 注册页
也没有根据开关隐藏密码表单入口。

### 审查记录

- PR #2787 的原始 diff 面向旧 `web/src` 前端路径，不能直接套用到当前
  StuHelper AI 的 `web/default` 和 `web/classic` 双前端结构。
- 本地保留现有后端接口拦截作为最终安全边界，只新增状态字段和前端展示逻辑。
- 状态接口使用 `password_login` / `password_register` 字段；前端对缺失字段按
  启用处理，以兼容旧缓存或旧服务端状态。
- default 前端在没有任何可见登录 / 注册方式时显示空状态文案；classic 前端
  在相同场景显示联系管理员文案，避免空白认证页。

### 本地行为

- `PasswordLoginEnabled=false` 时：
  - `/api/status` 返回 `password_login: false`。
  - default 与 classic 登录页隐藏密码登录表单和密码登录入口。
  - Passkey、OAuth、WeChat、Telegram 等登录方式继续按各自开关显示。
- `PasswordRegisterEnabled=false` 时：
  - `/api/status` 返回 `password_register: false`。
  - default 与 classic 注册页隐藏用户名 / 密码注册表单和密码注册入口。
  - 第三方注册入口继续按各自开关显示。

### 验证

```powershell
go test ./controller -run TestGetStatusIncludesPasswordAuthSwitches -count=1
Set-Location web/default; bun run typecheck
Set-Location web/default; bun run build
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
