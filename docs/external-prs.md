# 外部 PR 和补丁记录

本文档记录不是通过常规上游 release 同步引入的 PR 或补丁。

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
