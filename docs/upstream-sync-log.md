# 上游同步日志

本文档记录从 `QuantumNous/new-api` 同步到 StuHelper AI 分叉仓库的过程。

## 提交处理台账

处理方式说明：

- `合入`：本轮已把上游有效改动移植到本地代码。
- `语义覆盖`：未直接合入该 commit，但其效果已由其他本地移植或历史同步覆盖。
- `忽略`：明确不移植，通常为 `web/default` only、上游运营策略不适配，或用户要求跳过。
- `待决策`：已核对现状，但等待后续决定是否移植。
- `历史已处理`：此前同步日志已记录处理结果，本轮不重复评估。
- `本地提交`：本地为覆盖上游同步、保留分叉覆盖层或实现本地需求产生的 commit。
- `origin 合入`：推送前合入本仓库 `origin/main` 的依赖或维护提交。
- `merge-only`：合并提交本身不单独移植，按其实际内容所在提交处理。

| Commit | 处理时间 | 处理方式 | 提交时间 | 内容 | 处理说明 |
| --- | --- | --- | --- | --- | --- |
| `faa0f1425` | 2026-05-20 | 合入 | 2026-05-07 05:58:57 -0700 | fix: qualify column names in PerfMetric upsert to avoid ambiguity | 手工移植到 `model/perf_metric.go`。`perf_metrics` upsert 累加表达式显式引用 `perf_metrics.<column>`，避免 PostgreSQL `ON CONFLICT DO UPDATE` 列名歧义；按上游说明兼容 MySQL 和 SQLite。 |
| `19fc384e6` | 2026-05-13 | 历史已处理 | 2026-05-12 16:04:15 +0800 | feat(performance): update performance metrics handling and UI components | 2026-05-13 同步记录已语义覆盖。 |
| `03d537328` | 2026-05-13 | 历史已处理 | 2026-05-12 16:13:14 +0800 | fix(default): improve performance health panel layout | 2026-05-13 同步记录已语义覆盖。 |
| `3057f04a1` | 2026-05-13 | 历史已处理 | 2026-05-12 16:23:04 +0800 | fix(wallet): read topup gateway flags from topupInfo instead of status (#4599) | 2026-05-13 同步记录已覆盖；该上游提交本身相对父提交无文件 diff。 |
| `7fe896d2f` | 2026-05-13 | 历史已处理 | 2026-05-12 16:23:14 +0800 | fix: use getUserGroups for ratio display to respect GroupGroupRatio (#4772) | 2026-05-13 同步记录已语义覆盖。 |
| `2b89989f6` | 2026-05-13 | 历史已处理 | 2026-05-12 16:23:24 +0800 | fix(default): support DropdownMenuItem onSelect (#4787) | 2026-05-13 同步记录已语义覆盖。 |
| `fde2cac9d` | 2026-05-13 | 历史已处理 | 2026-05-12 16:23:33 +0800 | fix(web/default): guard playground messages against legacy classic shape (#4650) | 2026-05-13 同步记录已语义覆盖。 |
| `a720064d9` | 2026-05-13 | merge-only | 2026-05-12 16:24:00 +0800 | Merge branch 'main' of github.com:QuantumNous/new-api | 上游 merge commit，2026-05-13 记录说明未直接合并。 |
| `469d3747a` | 2026-05-13 | 历史已处理 | 2026-05-12 16:47:02 +0800 | fix: defaut ui triage (#4802) | 2026-05-13 同步记录已按语义拆分覆盖。 |
| `3856b9d2c` | 2026-05-13 | 历史已处理 | 2026-05-12 16:54:30 +0800 | chore(deps): bump axios from 1.15.0 to 1.15.2 in /web/classic (#4634) | 2026-05-13 同步记录已覆盖，2026-05-14 记录也说明本地已满足依赖状态。 |
| `428e3d91f` | 2026-05-13 | 历史已处理 | 2026-05-12 21:50:50 +0800 | chore: refresh related resources | 2026-05-13 同步记录已手工刷新，未覆盖 StuHelper AI/Xauryan 身份。 |
| `aa56667b8` | 2026-05-13 | 历史已处理 | 2026-05-12 21:53:37 +0800 | feat: track upstream request ID and prevent response header override | 2026-05-13 同步记录已语义覆盖。 |
| `53e318398` | 2026-05-13 | 本地提交 | 2026-05-13 01:09:54 +0800 | feat: track upstream request ids | 2026-05-13 同步结果提交，覆盖上游 `aa56667b8` 的 request id 语义；记录 upstream request id，并避免响应头覆盖。 |
| `49c8c933c` | 2026-05-13 | 本地提交 | 2026-05-13 01:15:35 +0800 | fix: make payment return paths theme-aware | 本地支付回跳路径修复。充值与订阅支付返回路径统一按当前主题生成，保留 classic/default 分流与本地支付覆盖层。 |
| `3d2f70a86` | 2026-05-13 | 本地提交 | 2026-05-13 01:19:12 +0800 | fix(default): port upstream UI compatibility fixes | 2026-05-13 同步结果提交，移植当时选定的 default UI 兼容修复，包括 dropdown onSelect、playground 旧 classic 消息结构兼容和 API key UI 调整。 |
| `fce87074c` | 2026-05-13 | 本地提交 | 2026-05-13 01:25:24 +0800 | fix(default): port route guards and ranking access checks | 2026-05-13 同步结果提交，覆盖上游路由守卫与排行榜访问校验；同时保留本地 classic 排行榜页面、`/api/rankings/users`、顶栏入口和后台管理开关。 |
| `8bfbaa289` | 2026-05-13 | 本地提交 | 2026-05-13 01:31:27 +0800 | feat(default): refresh performance dashboard | 2026-05-13 同步结果提交，覆盖性能指标 API、`pkg/perf_metrics` 类型调整和当时选定的 default dashboard/performance UI 刷新。 |
| `d4d328223` | 2026-05-13 | 本地提交 | 2026-05-13 01:35:16 +0800 | chore: refresh upstream related resources | 2026-05-13 同步结果提交，刷新相关资源、页脚与 i18n 元数据；明确保留 StuHelper AI/Xauryan 身份、Go module/import path、workflow、Docker、classic 默认前端和 GHCR release-only 发布策略。 |
| `d8bf61f4c` | 2026-05-13 | origin 合入 | 2026-05-11 19:20:25 +0000 | chore(deps): bump the npm_and_yarn group across 2 directories with 4 updates | 推送前合入 `origin/main` 依赖更新；保留 classic `axios` `1.15.2`，同时纳入 classic Vite 依赖与 electron 间接依赖更新。 |
| `1b18758a5` | 2026-05-13 | merge-only | 2026-05-12 17:25:03 +0800 | Merge pull request #1 from Xauryan/dependabot/npm_and_yarn/web/classic/npm_and_yarn-7f8752592c | `origin/main` PR merge commit；本地采用普通 merge 合入，不 rebase、不强推，实际依赖内容见 `d8bf61f4c`。 |
| `0526a2264` | 2026-05-14 | 忽略 | 2026-05-13 22:18:46 +0800 | feat: require compliance confirmation for paid features | 2026-05-14 记录已明确跳过；后续不要重复评估，除非产品决定引入本地合规确认流程。 |
| `3e588b4d4` | 2026-05-14 | 历史已处理 | 2026-05-13 22:21:03 +0800 | chore(deps-dev): bump ip-address from 10.1.0 to 10.2.0 in /electron (#4811) | 2026-05-14 记录说明本地已满足目标状态。 |
| `51b5cbe1b` | 2026-05-14 | 忽略 | 2026-05-13 22:21:24 +0800 | fix: prevent combobox from over-filtering options on focus (#4829) | default-only 修复，2026-05-14 已明确跳过。 |
| `18282e610` | 2026-05-14 | 历史已处理 | 2026-05-13 22:23:45 +0800 | chore(deps): update axios from 1.15.0 to 1.15.2 | 2026-05-14 记录说明本地已满足目标状态。 |
| `ff462faaa` | 2026-05-18 | 本地提交 | 2026-05-18 22:07:43 +0800 | feat(classic): redesign user consumption rankings with 3 metrics + me row | 本地 classic 排行榜扩展。`/api/rankings/users` 向后兼容新增 `total_tokens`、`request_count` 与可选 `me`；classic 页面改为 Token 用量、消费额度、调用次数三指标视图，充值榜数据继续保留，`web/default` 排行榜未改动。 |
| `abe6a63ea` | 2026-05-18 | 本地提交 | 2026-05-18 23:12:42 +0800 | fix(classic/rankings): recharge tab columns + me display name | 本地 classic 排行榜后续修正，调整充值榜保留字段与 `me.display` 展示名回退逻辑；该提交对应本地 tag `v1.0.0-rc.6-stuhelper.1`。 |
| `3caa6e467` | 2026-05-20 | 忽略 | 2026-05-16 14:48:49 +0800 | fix(web/default): batch fix new UI issues #4880 #4893 #4817 #4877 #4898 | default-only，按本轮要求不移植。 |
| `8f9ee9ba8` | 2026-05-20 | 忽略 | 2026-05-16 14:54:18 +0800 | fix: allow clearing channel remark (#4886) | default-only，按本轮要求不移植。 |
| `554defe4f` | 2026-05-20 | 合入 | 2026-05-16 14:54:23 +0800 | fix: correct usage logs filtering (#4883) | 移植 `model/log.go` 后端过滤修复。日志查询中的 model、username、token name 搜索改为转义后的 `LIKE ... ESCAPE '!'`，避免 `%`、`_` 等字符被当成通配符；default usage logs table 改动忽略。 |
| `8a10dedb7` | 2026-05-20 | 忽略 | 2026-05-16 14:54:35 +0800 | fix(web): handle unlimited API key quota validation (#4881) | default-only，按本轮要求不移植。 |
| `6f8668e4c` | 2026-05-20 | 合入 | 2026-05-16 14:54:47 +0800 | fix: enforce header nav access control for public modules (#4889) | 按本地需求移植“模型广场/定价”后端直达访问控制：`/api/pricing` 在模块关闭时返回 403，`/api/perf-metrics` 在 pricing 关闭或要求登录时走登录校验。排行榜继续保留本地 controller 级访问策略；default 路由/UI 改动忽略。 |
| `132d7b9f9` | 2026-05-20 | 语义覆盖 | 2026-05-16 14:54:50 +0800 | fix: GetAllChannels ignores group filter parameter (#4847) | 已由 `2d968c3ea` 的频道列表 group filter 重构覆盖。 |
| `cb7a61466` | 2026-05-20 | merge-only | 2026-05-16 22:11:38 +0800 | Merge pull request #4684 from SAY-5/fix/perf-metric-ambiguous-column | merge-only，内容由 `faa0f1425` 覆盖；不单独合入。 |
| `2d968c3ea` | 2026-05-20 | 合入 | 2026-05-17 11:44:07 +0800 | fix: apply group filter to channel list queries (#4885) | 手工移植非 default 后端部分到 `controller/channel.go` 与 `model/channel.go`。频道列表、标签列表、type counts 统一复用 group/status 查询条件；保留本地 `includeSensitive` 搜索能力和 `audit_admin` 脱敏/隐藏敏感字段策略。 |
| `68830e609` | 2026-05-20 | 合入 | 2026-05-17 11:44:38 +0800 | feat: support request_header key source (#4903) | 合入 `request_header` channel affinity key source，覆盖 service、测试、operation setting 注释和 classic 设置页选项；default 设置页改动忽略。 |
| `f69ceb696` | 2026-05-20 | 忽略 | 2026-05-17 11:45:27 +0800 | fix: 修复新 UI 语言与文案显示问题 (#4876) | 按用户决策忽略。default i18n/UI 改动不进入同步；唯一非 default 的 makefile dev/reset/rebuild 目标本轮也不移植。 |
| `5dd0d3bcb` | 2026-05-20 | 忽略 | 2026-05-17 18:54:39 +0800 | fix: add analytics placeholder (#4928) | default-only，按本轮要求不移植。 |
| `ee9736bbc` | 2026-05-20 | 忽略 | 2026-05-19 01:14:03 -0700 | fix: add type="submit" to forgot password form button (#4910) | default-only，按本轮要求不移植。 |
| `0936e2504` | 2026-05-20 | 合入 | 2026-05-19 12:11:24 +0800 | perf: avoid eager formatting in debug log calls (#4929) | 移植有价值的 debug 日志延迟格式化和无条件调试输出清理。未移植上游权限语义调整和 `model.User.AccessToken` JSON tag 改动，因为本地已有更严格的角色管理、`audit_admin` 权限边界和敏感字段脱敏逻辑。 |
| `04b4483d7` | 2026-05-20 | 忽略 | 2026-05-19 16:14:08 +0800 | fix(web): normalize model detail tabs layout (#4938) | default-only，按本轮要求不移植。 |
| `8ae095c3b` | 2026-05-20 | 合入 | 2026-05-19 16:14:11 +0800 | fix user create and delete handling (#4818) | 移植后端 `DeleteUser` 错误处理修复。硬删除失败时返回 `common.ApiError`，成功时才返回 success；default users drawer 改动忽略。 |
| `b397c58ba` | 2026-05-20 | 待决策 | 2026-05-19 16:14:34 +0800 | fix(auth): expose register_enabled in /api/status and gate sign-up link (#4871) | 已检查但未合入。本地已有 `password_login` / `password_register` 状态字段，classic 注册页也会根据 `password_register` 禁用密码注册；但 `/api/status` 仍未暴露全局 `register_enabled` / `password_register_enabled`，classic 登录页“注册”入口仍只按 `self_use_mode_enabled` 判断。因此本地只部分修复，和上游提交不完全等价。 |
| `fc08c133e` | 2026-05-20 | 忽略 | 2026-05-19 16:14:37 +0800 | fix(web/default): update pagination button labels in ModelCardGrid (#4675) | default-only，按本轮要求不移植。 |
| `cb9270ed2` | 2026-05-20 | 忽略 | 2026-05-19 01:14:49 -0700 | fix(auth): localize reset password confirmation (#4769) | default-only，按本轮要求不移植。 |
| `8db32213e` | 2026-05-20 | 忽略 | 2026-05-19 16:14:56 +0800 | fix(web/default/wallet): make recharge preset selection visible in dark mode (#4897) | default-only，按本轮要求不移植。 |
| `c78573ce0` | 2026-05-20 | 忽略 | 2026-05-19 16:15:02 +0800 | fix(web/default): api-info color dot shows wrong color due to semantic token mismatch (#4824) | default-only，按本轮要求不移植。 |
| `032993ed4` | 2026-05-20 | 合入 | 2026-05-19 16:15:13 +0800 | fix: check save result in handleSaveAll and add slate to validColors (#4823) | 移植后端 `validColors` 中的 `slate` 颜色允许项；default save result 处理忽略。 |
| `0cd9a3a06` | 2026-05-20 | 忽略 | 2026-05-19 16:39:42 +0800 | fix(auth): use aff_code field name in registration payload (#4945) (#4965) | default-only，按本轮要求不移植。 |
| `5e88f97ac` | 2026-05-20 | 忽略 | 2026-05-19 16:39:57 +0800 | fix(data-table): make faceted filter popover width adaptive (#4905) (#4966) | default-only，按本轮要求不移植。 |
| `146dd77b8` | 2026-05-20 | 忽略 | 2026-05-19 16:40:11 +0800 | fix(keys): call submit handler directly to avoid stale form linkage (#4858) (#4967) | default-only，按本轮要求不移植；该提交是 `v1.0.0-rc.7` tag commit。 |
| `0d4b25795` | 2026-05-20 | 合入 | 2026-05-19 18:28:03 +0800 | fix: expose param override audits for sensitive message fields (#4974) | 移植 `relay/common/override.go` 和测试。参数覆写审计扩展到 `messages`、`input`、`instructions`、`system`、Gemini contents / systemInstruction 等敏感正文路径，并按字段边界匹配；default log details 展示忽略，本地 `audit_admin` 脱敏逻辑保留。 |
| `2d1ca1538` | 2026-05-20 | 忽略 | 2026-05-19 18:46:21 +0800 | fix: respect dashboard content visibility settings (#4975) | default-only，按本轮要求不移植。 |
| `20d3e7373` | 2026-05-20 | 合入 | 2026-05-20 11:38:09 +0800 | fix: filter perf metrics summary by active groups (#4976) | 移植性能指标 summary 的 active groups 过滤。数据库聚合和内存 hot bucket 均只统计当前分组倍率中存在的分组以及 `auto`，并补充测试迁移 `PerfMetric` 表。 |
