# 外部 PR 和补丁记录

本文档记录不是通过常规上游 release 同步引入的 PR 或补丁。

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
