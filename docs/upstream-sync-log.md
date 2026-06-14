# 上游同步日志

本文档记录从 `QuantumNous/new-api` 同步到 StuHelper AI 分叉仓库的过程。

## 提交处理台账

处理方式说明：

- `合入`：本轮已把上游有效改动移植到本地代码。
- `语义覆盖`：未直接合入该 commit，但其效果已由其他本地移植或历史同步覆盖。
- `忽略`：明确不移植，通常为 `web/default` only、上游运营策略不适配，或用户要求跳过。
- `部分合入`：只移植该 commit 中适配本地的后端/共享逻辑，明确跳过 default-only、Waffo、或与本地产品策略冲突的部分。
- `待决策`：已核对现状，但等待后续决定是否移植。
- `历史已处理`：此前同步日志已记录处理结果，本轮不重复评估。
- `本地提交`：本地为覆盖上游同步、保留分叉覆盖层或实现本地需求产生的 commit。
- `origin 合入`：推送前合入本仓库 `origin/main` 的依赖或维护提交。
- `发布提交`：本地同步结果已经形成 commit/tag 并推送到 `origin`。
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
| `b397c58ba` | 2026-06-06 | 合入 | 2026-05-19 16:14:34 +0800 | fix(auth): expose register_enabled in /api/status and gate sign-up link (#4871) | 手工移植到本地 classic 语义。`/api/status` 新增 `register_enabled` 与 `password_register_enabled`，保留既有 `password_register`；classic 登录页和顶栏注册按钮同时受 `register_enabled` 与 `self_use_mode_enabled` 控制，注册页内部继续用 `password_register` 控制用户名/密码注册方式。 |
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
| `e272ad0e1` | 2026-05-20 | 发布提交 | 2026-05-20 23:14:39 +0800 | chore: sync upstream rc7 updates | 本地 rc7 同步落地提交已推送到 `origin/main`。发布 tag 改用无后缀 `v1.0.0-rc.7`；删除 `v1.0.0-rc.7-stuhelper.1`，并将 `v1.0.0-rc.7` 指向本地最新 `main` 后推送到 `origin`。 |
| `58ba867d` | 2026-05-25 | 忽略 | 2026-05-21 11:09:51 +0800 | fix: improve channel test failure details UX (#4988) | default-only 渠道测试弹窗与 i18n 体验优化，本地 classic 管理端不移植。 |
| `6f11d198` | 2026-05-25 | 忽略 | 2026-05-21 11:10:22 +0800 | fix: normalize model pricing display drift (#4985) | default-only 模型价格编辑显示精度修复；classic 如后续出现同类显示问题再单独按本地组件实现，不直接移植 default 文件。 |
| `006e8016` | 2026-05-25 | 合入 | 2026-05-21 11:16:17 +0800 | fix: resolve model owned_by from active channels (#4416) | 手工移植 `/v1/models` 的 `owned_by` 解析。现在根据当前 token/user group、auto 分组上下文、能力优先级、权重与启用渠道类型选择 owner；同时保留本地用户自定义 auto 分组优先级，高于系统默认 auto 分组。新增 controller/model 测试与 model owner 查询测试。 |
| `ae6a0336` | 2026-05-25 | 合入 | 2026-05-22 10:32:11 +0800 | perf: optimize request metadata extraction and disabled field filtering (#5009) | 手工移植后端性能优化：JSON 分发阶段用 `gjson` 只读取 `model/group` 并复位请求体；OpenAI stream token 统计边读边处理，不再缓存整个 stream item 列表；禁用字段过滤先快速判断是否存在可移除字段，避免无效整包 unmarshal。 |
| `e13d6734` | 2026-05-25 | 忽略 | 2026-05-22 10:36:50 +0800 | fix: update default frontend hardcoded route links (#5016) | default-only 路由链接修复，本地 classic 不移植。 |
| `8e5e89bb` | 2026-05-25 | 忽略 | 2026-05-22 10:39:24 +0800 | 修复 切换新版前端Turnstile 开启后注册页未显示验证的问题 (#5011) | default-only 注册页 Turnstile 修复；classic 注册页为本地独立实现，本轮不移植。 |
| `19f1821f` | 2026-05-25 | 忽略 | 2026-05-22 11:00:58 +0800 | [Feature Request] Waffo Pancake gateway — full integration with subscription support + admin catalog binding flow (#4935) | 用户确认本项目不用 Waffo/Waffo Pancake 支付。该提交新增 Waffo Pancake 充值、订阅、catalog/store/product 绑定、SDK 与 UI，不适配本地支付策略，明确不合入。 |
| `f2c7647e` | 2026-05-25 | 忽略 | 2026-05-22 11:48:32 +0800 | fix: enforce Waffo subscription compliance and product ID update (#5038) | Waffo Pancake 订阅合规与产品 ID 修复，依赖上一个 Waffo Pancake 集成；本项目不用该支付，明确不合入。 |
| `b9bc6f0e` | 2026-05-25 | 忽略 | 2026-05-22 16:19:54 +0800 | Revert "fix: correct usage logs filtering (#4883)" | 上游回滚 `554defe4f` 的日志过滤语义。本地保留已移植的 `LIKE ... ESCAPE '!'` 安全过滤，避免 `%`、`_` 被误当通配符；不跟随回滚。 |
| `fddf54cc` | 2026-05-25 | 合入 | 2026-05-22 19:08:38 +0800 | perf: reduce heap residency for large base64 relay requests | 手工移植大请求内存优化：`UnmarshalBodyReusable` 对磁盘缓存 JSON 走流式 decode；新增出站 `BodyStorage` 包装并传播 `ContentLength`；多个 relay handler 在转换后释放原始 `jsonData`；参数覆写改为 `[]byte` 热路径；Gemini inline media 响应改用 `strings.Builder`，减少 base64 大字符串中间分配。 |
| `ebbe3155` | 2026-05-25 | 合入 | 2026-05-23 13:24:56 +0800 | 🐛 fix(channel): evict auto-disabled multi-key channels from cache (#4983) | 手工移植多 key 渠道缓存修复。现在按实际 key 状态判断是否全部不可用，找不到 using key 时记录并跳过错误更新，key 恢复启用时可恢复渠道状态，缓存状态变化时同步更新，减少 auto 路由反复选中无可用 key 渠道的问题。 |
| `0354c38b` | 2026-05-25 | 忽略 | 2026-05-24 16:19:27 +0800 | [BugFix] fix webhook process (#5047) | Waffo Pancake webhook 订单映射修复；本项目不用 Waffo/Waffo Pancake 支付，明确不合入。 |
| `49bc3a11` | 2026-05-25 | 忽略 | 2026-05-24 16:37:43 +0800 | fix(payment): hide classic Waffo Pancake settings (#5085) | Waffo Pancake checkout 参数校验与 classic 设置入口移除；本项目不用 Waffo/Waffo Pancake 支付，且移除 classic 设置不适合本地后台策略，明确不合入。 |
| `92a09594` | 2026-05-25 | 忽略 | 2026-05-24 22:09:05 +0800 | ✨ refactor(web/default): adopt drill-in sidebar pattern for System Settings | default-only 系统设置导航重构，本地 classic 不移植。 |
| `b08febaa` | 2026-05-25 | 忽略 | 2026-05-25 00:34:26 +0800 | ✨ refactor: system settings UI for consistent, compact layouts | default-only 系统设置 UI 与 default i18n 同步，本地 classic 不移植。 |
| `88437a18` | 2026-05-25 | 忽略 | 2026-05-25 01:06:42 +0800 | ⬆️ chore(deps): Upgrade default frontend dependencies | default 前端依赖升级，本地 classic 依赖不受影响，不移植。 |
| `b302be30` | 2026-05-25 | 部分合入 | 2026-05-25 02:42:22 +0800 | 🛠️ fix: v1 interface feedback regressions | 只移植适配本地的后端小修：复制渠道改用 `clone.Insert()`，确保克隆后能力与新 ID 绑定；用户搜索接口新增 `role/status` 服务端过滤。default 前端缓存、表格、认证、Playground、依赖等改动忽略；`password_login_enabled` 别名仍不单独加入，因为 classic 现有调用只使用 `password_login`，`password_register_enabled` 已随 `b397c58ba` 注册状态字段补齐。 |
| `583da452` | 2026-05-25 | 忽略 | 2026-05-25 05:35:44 +0800 | ✨ refactor(ui): Improve usage log filter responsiveness and mobile UX | default-only 使用日志筛选与移动端 UI 优化，本地 classic 使用日志为独立实现，不移植。 |
| `2a528d46` | 2026-05-29 | 合入 | 2026-05-25 22:57:02 +0800 | fix(relay): correct image quality parameter handling (#5103) | 手工移植后端图片 relay 修复。`relay/image_handler.go` 的消费日志/计费上下文现在保留客户端传入的 `quality`，只在空值时默认 `standard`，避免 `gpt-image-1` 的 `low`、`medium`、`high`、`auto` 等质量被误记为 `standard`。 |
| `51ca897c` | 2026-05-29 | 忽略 | 2026-05-25 23:10:10 +0800 | ✨ refactor(home): redesign hero section to dual-column layout with compliant copywriting | default-only 首页 hero 与 default i18n 文案重构。本地已删除 `web/default` 并保留 classic-only 产品线，不合入。 |
| `12880281` | 2026-05-29 | 合入 | 2026-05-25 23:10:30 +0800 | fix: truncate oversized upstream error logs (#5083) | 手工移植后端日志安全修复。新增 `common.LocalLogPreview`，relay 错误日志、渠道禁用原因和本地错误日志只记录截断预览，debug 模式保留完整内容；补充 `RelayErrorHandler` 截断行为测试，避免上游超大错误体写爆本地日志。 |
| `ff06067a` | 2026-05-29 | 合入 | 2026-05-25 23:13:06 +0800 | fix: 移除 fcIdx -1 偏移，修复并发工具调用撞键问题 (#5095) | 手工移植 Claude 流式 tool_use 转 OpenAI tool_calls 修复。`StreamResponseClaude2OpenAI` 现在直接使用 Claude content block index，不再减 1，避免并发工具调用撞到同一 tool call 槽位。 |
| `465c5eda` | 2026-05-29 | 合入 | 2026-05-25 23:14:01 +0800 | fix:gemini to claude tool_use err (#5041) | 手工移植 Gemini stream 转 Claude 格式的 tool_use 结束语义修复。tool call chunk 会设置 `tool_calls` finish reason；Claude 格式下清空 choice finish reason，并避免额外 stop chunk；最终 usage 响应在 Claude 转换未完成时改发 stop 响应并携带 usage。 |
| `349d5429` | 2026-05-29 | 忽略 | 2026-05-25 23:15:59 +0800 | fix: handle paginated API key search response (#5014) | default-only API key 搜索分页响应处理。本地 classic 的令牌搜索已通过 `syncPageData(data)` 处理分页对象，后端 `/api/token/search` 也返回 `PageInfo`；不移植 default 文件。 |
| `3d850d38` | 2026-05-29 | 忽略 | 2026-05-26 01:22:49 +0800 | ♻️ refactor(channels): rebuild channel create/edit drawer with modular sections and improved form UX | default-only 渠道创建/编辑抽屉重构，本地 classic 独立实现且 `web/default` 已删除；不合入。 |
| `b37b6d80` | 2026-05-29 | merge-only | 2026-05-26 01:22:56 +0800 | Merge remote-tracking branch 'origin/main' | 上游 merge commit；`--remerge-diff` 未发现独立冲突解决内容，按父提交分别处理，不单独移植。 |
| `33608826` | 2026-05-29 | 忽略 | 2026-05-26 01:55:27 +0800 | ♻️ refactor(channels): rebuild channel editor UX with modular sections and Base UI multi-select | default-only 渠道编辑器与首页片段调整，本地 classic 不移植。 |
| `a64f26d1` | 2026-05-29 | 忽略 | 2026-05-26 04:31:13 +0800 | 🎨 feat(web/default): add Anthropic theme preset and configurable serif typography | default-only 主题预设、字体与 default 依赖变更，本地 classic 不移植。 |
| `ad224ecf` | 2026-05-29 | 忽略 | 2026-05-26 10:20:54 +0800 | fix: prevent duplicate channel action toasts (#5015) | default-only 渠道操作 toast 去重与 default API 封装调整。本地 classic 使用 Semi/本地 helpers，未复用该前端栈；不直接移植。 |
| `bc8110ce` | 2026-05-29 | 忽略 | 2026-05-26 11:20:38 +0800 | 🎨 refactor(badge): restore status-badge sizes and classic color scheme | default-only badge 样式调整，本地 classic 不移植。 |
| `10119349` | 2026-05-29 | 忽略 | 2026-05-26 11:29:38 +0800 | 🎨 fix(theme): default theme font preset falls back to Sans instead of Serif | default-only 主题字体回退修复，本地 classic 不移植。 |
| `6b6c9904` | 2026-05-29 | 合入 | 2026-05-26 12:03:02 +0800 | feat(subscription): support balance purchases | 手工移植余额购买订阅并按本地产品策略适配。新增 `/api/subscription/balance/pay`，同一事务内锁定用户、校验余额、条件扣减 `quota`、创建订阅实例、创建成功订阅订单并同步账单记录；余额购买不计入邀请返佣，避免用既有钱包余额触发返佣套利。上游 default UI 改为本地 classic 充值/订阅页实现，并补齐 8 个 locale 文案。 |
| `a8b7c92e` | 2026-05-29 | 忽略 | 2026-05-26 12:03:43 +0800 | 🎨 fix(logs): restore timing background badges and optimize model/token spacing | default-only 使用日志表格样式调整，本地 classic 不移植。 |
| `9e283ab1` | 2026-05-29 | 忽略 | 2026-05-26 12:16:26 +0800 | 🎨 fix(logs): remove hardcoded font-mono to support global theme font inheritance | default-only 使用日志字体继承修复，本地 classic 不移植。 |
| `f223db93` | 2026-05-29 | 忽略 | 2026-05-26 12:30:13 +0800 | 🎨 fix(charts): improve dark mode chart readability | default-only 图表暗色模式样式修复，本地 classic 不移植。 |
| `c91ba0c4` | 2026-05-29 | 忽略 | 2026-05-26 12:32:05 +0800 | fix: consolidate Waffo payment settings save flow (#5110) | default-only Waffo/Waffo Pancake 支付设置保存流程修复；本项目已明确不用 Waffo/Waffo Pancake，且 default 已删除，不合入。 |
| `30025aeb` | 2026-05-29 | 合入 | 2026-05-26 12:32:20 +0800 | fix: use actual user id for channel tests (#5109) | 手工移植后端渠道测试修复。单渠道测试使用当前请求用户 ID 获取缓存、分组和记录消费日志；自动批量测试没有请求上下文时回退 root 用户，并补充内部测试覆盖请求用户优先逻辑。 |
| `5bc4c748` | 2026-05-29 | 忽略 | 2026-05-26 12:40:39 +0800 | 🎨 fix(logs): tune usage table typography | default-only 使用日志排版修复；该提交是上游 `v1.0.0-rc.9` tag commit，本地 classic 不移植。 |
| `65f8afe9` | 2026-05-29 | 忽略 | 2026-05-26 15:43:56 +0800 | 🐛 fix(system-settings): resolve save detection and number input NaN issues | default-only 系统设置保存检测和数字输入 NaN 修复。本地 classic 设置页为独立实现，不直接移植；如 classic 出现同类 NaN 问题再按本地组件单独修。 |
| `f8add4ca` | 2026-05-29 | 忽略 | 2026-05-26 18:35:51 +0800 | feat(theme): add simple-large preset, xl scale and clean up channel badge dots | default-only 主题预设和日志/渠道样式调整，本地 classic 不移植。 |
| `dc245ae7` | 2026-05-29 | 忽略 | 2026-05-26 20:28:28 +0800 | fix(web): improve channel and usage log UI | default-only 渠道和使用日志 UI 改进，本地 classic 不移植。 |
| `1d320373` | 2026-05-29 | 合入 | 2026-05-26 21:00:32 +0800 | fix: keep usage log filters exact unless wildcard is explicit (#5097) | 手工移植后端日志筛选语义调整。usage/admin 日志的 model、username 筛选在未显式包含 `%` 时改为精确匹配；包含 `%` 时复用 `sanitizeLikePattern` 并用 `LIKE ... ESCAPE '!'` 进行受控通配匹配。default 表格 manual filtering 不适用于本地 classic，未移植。 |
| `74985fa8` | 2026-05-29 | 合入 | 2026-05-26 21:17:25 +0800 | fix: keep token log filters exact | 手工移植 token name 日志筛选修正。令牌名保持精确匹配，不允许 `%` 通配，避免用户输入令牌名时被误当模糊条件；该提交是上游 `v1.0.0-rc.10` tag commit，本地未拉取/保留上游 tag。 |
| `5b86ce0d` | 2026-05-29 | 合入 | 2026-05-27 13:01:13 +0800 | fix: optimize batch update process | 手工移植后端批量更新优化。批处理先快照各类累加 map，再把同一用户的 `quota`、`used_quota`、`request_count` 合并成一次 GORM `Updates`，减少结算写库次数；表达式保持 MySQL/PostgreSQL/SQLite 通用。 |
| `63ead2bf` | 2026-05-29 | 合入 | 2026-05-28 15:02:00 +0800 | chore(repo): ignore playwright mcp artifacts | 手工合入仓库维护项，在 `.gitignore` 增加 `.playwright-mcp`，避免 Playwright MCP 临时产物进入工作区。 |
| `e79cee1e` | 2026-05-29 | 忽略 | 2026-05-28 15:10:17 +0800 | perf(form): focus first validation error on submit | default-only 表单校验失败自动聚焦实现。本地 classic 使用 Semi Design 表单体系，不移植 default 组件。 |
| `e8c836d7` | 2026-05-29 | merge-only | 2026-05-28 23:34:02 +0800 | fix(web): improve form validation error focus #5163 | 上游 PR merge commit，内容为 `e79cee1e` 和 `.gitignore` 变更；按对应提交处理，不单独移植。 |
| `38bf2d8d` | 2026-06-05 | 忽略 | 2026-05-29 12:18:52 +0800 | feat(keys/cc-switch-dialog): 修复自定义cc-switch名称失焦后重置问题 (#5170) | default-only API key / cc-switch dialog 前端修复；本地 classic 令牌页面未复用该组件，不移植。 |
| `15880270` | 2026-06-05 | 合入 | 2026-05-29 12:54:00 +0800 | feat: add subscription balance redemption toggle (#3071) | 手工合入套餐级余额兑换开关。后端新增 `SubscriptionPlan.allow_balance_pay`、迁移默认值和余额购买校验；classic 订阅管理增加“允许余额支付”，用户订阅购买页按套餐隐藏余额支付，后台列表展示余额支付配置，并补齐多语言文案。 |
| `afb470e4` | 2026-06-05 | 合入 | 2026-05-30 19:54:02 +0800 | fix(model): correct idx_created_at_id index column order to (created_at, id) (#5191) | 合入 `logs.created_at,id` 复合索引顺序修正，使后续按 `created_at desc, id desc` 排序能命中索引。 |
| `230a3592` | 2026-06-05 | 合入 | 2026-05-30 20:00:02 +0800 | perf: order admin logs by created_at to use composite index (#5116) | 合入管理员日志排序优化。`GetAllLogs` 改为按 `created_at desc, id desc` 排序，避免只按自增 ID 时无法充分利用时间索引。 |
| `b2e25b7d` | 2026-06-05 | 合入 | 2026-05-31 13:49:50 +0800 | chore(deps): bump axios from 1.15.2 to 1.16.0 in /web/classic (#5185) | 合入 classic axios 依赖更新；当时刷新 classic 锁文件，后续 Rsbuild workspace 迁移已统一到 `web/bun.lock`。 |
| `0c7aceb8` | 2026-06-05 | 合入 | 2026-05-31 13:50:52 +0800 | feat: add claude opus 4.8 support (#5177) | 合入 Claude Opus 4.8 支持。补充 Claude/AWS/Vertex 模型映射、倍率与缓存倍率，并把 4.8 系列纳入 adaptive thinking 后缀处理和测试。 |
| `08604465` | 2026-06-05 | 忽略 | 2026-06-01 17:58:02 +0800 | fix(pricing): sync custom model icons | default-only 定价图标同步。本地 classic 价格展示未复用 default 图标映射，不移植。 |
| `45d54c16` | 2026-06-05 | merge-only | 2026-06-01 18:17:58 +0800 | fix(pricing): sync custom model icons #5224 | 上游 PR merge commit，实际内容由 `08604465` 处理；不单独移植。 |
| `b596de73` | 2026-06-06 | 合入 | 2026-06-01 19:12:39 +0800 | chore(web): centralize shared frontend dependency versions | 手工合入前端 workspace 依赖集中管理。本地没有 `web/default`，因此 `web/package.json` 只声明 `classic` workspace，锁文件迁移为 `web/bun.lock`，workspace 包名保持 StuHelper AI 品牌。 |
| `9a2e60df` | 2026-06-06 | merge-only | 2026-06-01 19:19:13 +0800 | chore(web): centralize shared frontend dependency versions #5227 | 上游 PR merge commit，实际内容由 `b596de73` 处理；本地按 classic-only workspace 适配。 |
| `1e9ff8a0` | 2026-06-06 | 合入 | 2026-06-02 00:32:16 +0800 | feat(web): support classic Rsbuild dev and build | 手工合入 classic Rsbuild dev/build。删除 Vite 配置，新增 `web/classic/rsbuild.config.ts`，React 依赖升至 19，并把 Dockerfile、makefile 和开发文档调整为从 `web` 根 workspace 安装依赖。 |
| `0bbcaa89` | 2026-06-06 | 合入 | 2026-06-02 00:50:29 +0800 | fix(classic): inject Semi React 19 adapter | 合入 classic 入口 `@douyinfe/semi-ui/react19-adapter` 注入，配合 React 19 运行。 |
| `0ff9c35e` | 2026-06-06 | merge-only | 2026-06-02 11:33:33 +0800 | feat(web): support classic Rsbuild dev and build | 上游 PR merge commit，实际内容由 `1e9ff8a0` 和 `0bbcaa89` 处理；本地已按 classic-only 适配。 |
| `4d20e053` | 2026-06-05 | 忽略 | 2026-06-02 12:09:47 +0800 | fix(channels): reveal advanced validation errors | default-only 渠道表单校验展示优化。本地 classic 渠道表单为独立实现，不直接移植 default 组件。 |
| `cb5c0453` | 2026-06-05 | 忽略 | 2026-06-02 12:31:32 +0800 | fix(channels): avoid expanding advanced settings for model mapping | default-only 渠道高级设置展开逻辑修复。本地 classic 不移植。 |
| `7791b784` | 2026-06-05 | 忽略 | 2026-06-02 14:28:35 +0800 | chore(fd): delete the test file | 上游前端临时测试文件清理，本地没有对应 default/Rsbuild 测试文件，不移植。 |
| `7aaa5332` | 2026-06-05 | merge-only | 2026-06-02 14:30:20 +0800 | fix(channels): reveal advanced validation errors #5239 | 上游 PR merge commit，实际内容由 `4d20e053`、`cb5c0453`、`7791b784` 处理；不单独移植。 |
| `d17b566b` | 2026-06-05 | 忽略 | 2026-06-03 12:04:40 +0800 | docs: refine issue templates (#5271) | 上游 GitHub issue template 流程不适配本地 StuHelper AI/Xauryan 维护策略，不移植。 |
| `b0ac0429` | 2026-06-05 | 忽略 | 2026-06-03 12:37:36 +0800 | fix(web): resolve TypeScript errors in usage logs mobile card | default-only usage logs TypeScript 修复。本地 classic 不使用 default usage logs mobile card。 |
| `580ad97c` | 2026-06-05 | 合入 | 2026-06-03 22:23:12 +0800 | fix: convert usd amount by exchange rate in classic quota display | 合入 classic 额度金额展示修复。`renderQuotaWithAmount` 现在复用 `getCurrencyConfig()` 并按汇率换算金额，避免只切换货币符号不换算数值。 |
| `00d23abf` | 2026-06-05 | merge-only | 2026-06-04 02:55:23 +0800 | fix: 修复余额显示时只切换了单位未切换数值 #5296 | 上游 PR merge commit，实际内容由 `580ad97c` 处理；不单独移植。 |
| `3aa113b5` | 2026-06-05 | 合入 | 2026-06-04 18:21:35 +0800 | fix(dify): initialize file pointer before remote-image field assignment (#5134) | 合入 Dify 远程图片 nil pointer 修复，远程图片文件对象先初始化再设置类型、传输方式和 URL。 |
| `87cc22d7` | 2026-06-05 | 合入 | 2026-06-04 18:48:30 +0800 | fix(distributor): resolve model for GET /v1/video/generations/:task_id (#5133) | 合入视频任务 GET 分发模型解析修复。`/v1/videos` 与 `/v1/video/generations` 查询任务时从任务记录恢复原始模型名，避免 token 模型限制场景选路错误。 |
| `933ea0cd` | 2026-06-05 | 合入 | 2026-06-05 11:30:08 +0800 | fix: add relay idle connection timeout config (#5309) | 合入 `RELAY_IDLE_CONN_TIMEOUT` 配置，默认 90 秒，并补充 `.env.example`、Docker Compose 和 README 说明。 |
| `4a188dee` | 2026-06-05 | 合入 | 2026-06-05 11:30:29 +0800 | feat: 支持配置渠道被禁用后是否清空渠道粘性 (#5306) | 合入渠道亲和缓存清理配置。默认在亲和渠道禁用或不适用于当前分组/模型时清除当前粘性并重新选路；新增 `keep_on_channel_disabled` 后端配置、classic 设置开关、测试和多语言文案。 |
| `83068d11` | 2026-06-05 | 合入 | 2026-06-05 11:31:20 +0800 | fix(relay): fix Anthropic-compatible compatibility for GLM (avoid chunked encoding) (#5307) | 合入 Claude/Anthropic 兼容修复，透传上游请求体时记录 `UpstreamRequestBodySize`，避免 GLM 等兼容服务端因 chunked encoding 处理不兼容出错。 |
| `d2f7f9ee` | 2026-06-05 | 合入 | 2026-06-05 11:39:29 +0800 | fix: limit anonymous request body (#5244) | 合入匿名公开接口请求体限制，新增 `ANONYMOUS_REQUEST_BODY_LIMIT_KB`，对注册、登录、找回、OAuth 绑定、支付回调等未认证 POST 路由施加默认 512KB 限制；本地官方支付宝/微信回调也纳入同一中间件。 |
| `189913b7` | 2026-06-05 | 合入 | 2026-06-05 11:54:57 +0800 | fix(i18n): clarify thinking adapter copy (#5242) | 合入 classic 全局模型设置文案调整，将“禁用思考处理的模型列表”改为“不自动处理思考后缀的模型列表”，并补齐多语言翻译。 |
| `01c2128e` | 2026-06-05 | 合入 | 2026-06-05 12:12:45 +0800 | fix: 收窄 OpenAI o 系列模型适配范围 (#5293) | 合入 OpenAI reasoning 模型识别收窄。只将 `o1/o3/o4` 系列按 reasoning 模型处理，避免 `omni-*` 等以 `o` 开头的非 reasoning 模型被错误改 system role 或温度参数。 |
| `32805849` | 2026-06-05 | 合入 | 2026-06-05 12:18:57 +0800 | fix: reuse stream scanner buffer in channel handlers (#5225) | 合入 stream scanner buffer 复用。新增 `helper.NewStreamScanner` 并切换 Cloudflare、Cohere、Coze、Ollama、Tencent、Zhipu 等流式适配器，统一使用可配置的大行缓冲，补充大行 scanner 测试。 |
| `b5331936` | 2026-06-06 | 忽略 | 2026-06-05 06:34:22 +0800 | feat(web): show user id on profile page | default-only profile header 展示用户 ID。本地已删除 `web/default`，classic 用户资料页不复用该组件，不移植。 |
| `adc390c5` | 2026-06-06 | merge-only | 2026-06-06 00:45:13 +0800 | feat(web): show user id on profile page (#5317) | 上游 PR merge commit，实际内容由 `b5331936` 处理；default-only，不单独移植。 |
| `979aeceb` | 2026-06-13 | 忽略 | 2026-05-28 19:17:47 +0800 | fix: align mobile usage log cost badge | default-only 使用日志移动端消费徽标样式修复。本地已删除 `web/default`，classic 使用日志移动端卡片未复用该组件，不移植。 |
| `abad0d3c` | 2026-06-13 | 忽略 | 2026-06-03 14:49:08 +0800 | fix(model-pricing): detect visual pricing draft changes on save | default-only 模型价格视觉编辑器保存检测修复。已按规则复核 `pkg/billingexpr/expr.md`；该提交不改后端表达式、存储、结算或 classic 编辑器，本轮不移植。 |
| `9e59ffc3` | 2026-06-14 | 部分合入 | 2026-06-03 18:27:07 +0800 | fix(model-pricing): align pricing mode editor spacing | default-only 模型价格编辑器布局不直接移植；按 classic 组件兼容吸收价格输入提示、搜索防抖和 IME 组合输入保护，避免中文输入时列表过早过滤。 |
| `39e05118` | 2026-06-14 | 部分合入 | 2026-06-03 18:27:40 +0800 | fix(model-pricing): align pricing mode editor spacing | default-only 分层价格编辑器布局不直接移植；classic `TieredPricingEditor` 兼容扩大原始表达式编辑区并使用等宽排版，提高长表达式可读性。 |
| `77d31575` | 2026-06-13 | 忽略 | 2026-06-04 17:22:50 +0800 | fix(model-pricing): commit visual pricing drafts on save | default-only 模型价格视觉草稿保存语义修复。本地 classic 的表达式编辑器和保存路径独立，且该提交不改 `pkg/billingexpr` 后端语义，本轮不移植。 |
| `6e5a3591` | 2026-06-13 | 忽略 | 2026-06-05 00:06:41 +0800 | refactor(model-pricing): split visual pricing editor modules | default-only 模型价格视觉编辑器拆分重构。已复核计费表达式设计文档；本地 classic 组件树不同，不移植 default 模块拆分。 |
| `5681c92b` | 2026-06-14 | 部分合入 | 2026-06-05 01:04:47 +0800 | perf(model-pricing): refine visual editor actions | default-only 视觉编辑器动作不直接移植；classic 兼容吸收模型价格搜索防抖、IME 组合输入保护与小数输入提示，未改后端表达式计费、存储、预扣或结算语义。 |
| `c8d37680` | 2026-06-13 | 忽略 | 2026-06-05 14:02:26 +0800 | fix: respect theme for multiselect combobox popover | default-only combobox 主题修复，本地 classic 使用 Semi 组件，不移植。 |
| `e2dbd02c` | 2026-06-13 | merge-only | 2026-06-05 14:11:55 +0800 | Merge remote-tracking branch 'upstream/main' into fix/mobile-usage-log-cost-alignment | 上游 merge commit，不单独移植；按其父提交实际内容分别处理。 |
| `e8c36762` | 2026-06-14 | 部分合入 | 2026-06-05 17:24:33 +0800 | fix: support six-decimal steps in model pricing editor | 上游 default 分层价格输入步进从 `0.01` 改为 `0.000001`。classic 原本未受 step 限制，本轮补充 `inputMode='decimal'` 与“支持最多 6 位小数”提示，便于管理员明确输入精度。 |
| `81d3dc08` | 2026-06-13 | 忽略 | 2026-06-06 14:15:44 +0800 | perf(model-pricing): reduce duplicate model name display | default-only 模型价格编辑器展示优化，本地 classic 不移植。 |
| `75c05bb4` | 2026-06-14 | 部分合入 | 2026-06-06 14:36:21 +0800 | perf(model-pricing): improve JSON pricing editor layout | default-only JSON 价格编辑器不直接移植；classic 原始表达式 TextArea 兼容扩大高度并使用等宽/换行样式，保留现有保存路径。 |
| `0f043ae4` | 2026-06-13 | 忽略 | 2026-06-06 15:14:26 +0800 | feat(json-editor): add reusable JSON code editor | default-only JSON code editor 组件与 default 模型价格编辑器集成；本地 classic 不复用该组件，本轮不移植。 |
| `4dd68bad` | 2026-06-13 | 忽略 | 2026-06-06 15:26:53 +0800 | perf(model-pricing): move pricing tabs into page title | default-only 系统设置/价格编辑器布局调整，本地 classic 不移植。 |
| `f5753a2b` | 2026-06-13 | 忽略 | 2026-06-06 15:49:38 +0800 | perf(web): simplify public page hero copy | default-only public page 文案调整；本地 classic/站点文案不复用 default 页面，不移植。 |
| `7a5348ca` | 2026-06-13 | 忽略 | 2026-06-06 18:47:10 +0800 | feat(web): add shared dialog wrapper | default-only dialog wrapper 新增，本地 classic 使用 Semi Modal/自有封装，不移植。 |
| `2eaa943d` | 2026-06-13 | 忽略 | 2026-06-06 21:49:33 +0800 | perf(web): improve dialog sizing and footer layout | default-only 大量 dialog 尺寸和 footer 布局调整，本地 classic 不移植。 |
| `1e6f31b2` | 2026-06-14 | merge-only | 2026-06-06 23:14:18 +0800 | perf(model-pricing): improve model pricing editor UX (#5275) | 上游 PR merge commit，不单独移植；实际内容按子提交处理。classic 已兼容吸收适合本地的搜索输入、六位小数提示和原始表达式编辑布局，未搬 default 组件拆分和 JSON code editor。 |
| `a1c82841` | 2026-06-13 | merge-only | 2026-06-06 23:15:05 +0800 | chore(web): simplify public page hero copy (#5339) | 上游 PR merge commit，实际内容由 `f5753a2b` 处理；default-only，不单独移植。 |
| `15ff8e02` | 2026-06-13 | merge-only | 2026-06-06 23:16:53 +0800 | chore(web): improve frontend dialog layout and sizing (#5346) | 上游 PR merge commit，实际内容由 `7a5348ca`、`2eaa943d` 处理；default-only，不单独移植。 |
| `19153448` | 2026-06-13 | merge-only | 2026-06-06 23:18:04 +0800 | fix: respect theme for multiselect combobox popover (#5328) | 上游 PR merge commit，实际内容由 `c8d37680` 处理；default-only，不单独移植。 |
| `16dd7237` | 2026-06-13 | merge-only | 2026-06-06 23:19:07 +0800 | fix: align mobile usage log cost badge (#5161) | 上游 PR merge commit，实际内容由 `979aeceb` 处理；default-only，不单独移植。 |
| `4ca47ee2` | 2026-06-13 | merge-only | 2026-06-06 23:22:37 +0800 | fix: support six-decimal steps in model pricing editor (#5332) | 上游 PR merge commit，实际内容由 `e8c36762` 处理；本地 classic 语义已覆盖，不单独移植。 |
| `d2576ddc` | 2026-06-13 | 部分合入 | 2026-06-08 18:36:17 +0800 | fix(openai): support streaming image relay and image edit for images API  (#4608) | 手工移植 OpenAI Images API 后端核心行为：`ImageRequest.Stream` 改为 `*bool` 以保留显式 `false`，multipart image edit 通过 `common.ParseMultipartFormReusable` 保持 body 可复用并解析 `stream`，OpenAI 图片响应按 stream/非 stream 分流处理，JSON 图片结果可包装为 SSE。未引入 default 前端内容。 |
| `30d3a3a5` | 2026-06-13 | 忽略 | 2026-06-10 17:18:51 +0800 | perf(web): add debounce channel search and skip during IME composition (#5393) | default-only 渠道表格搜索防抖与 IME 组合输入处理。本地 classic 渠道表格未复用 default data-table，不移植。 |
| `867d8acf` | 2026-06-13 | 合入 | 2026-06-10 17:19:57 +0800 | fix: normalize kimi k2.6 temperature (#5390) | 手工移植 Moonshot `kimi-k2.6` temperature 归一化。对上游模型名为 `kimi-k2.6` 且客户端显式传入非 `1.0` temperature 的请求改写为 `1.0`；未传 temperature 时保持省略，其他 Moonshot 模型不受影响，并补充单元测试。 |
| `59a93cf5` | 2026-06-14 | 部分合入 | 2026-06-10 17:47:37 +0800 | fix(openai): align image streaming relay governance | 在本地拆出 `relay/channel/openai/relay_image.go`，补齐图片 usage 归一化、SSE error event 记录、非流 JSON 到 SSE 的 fallback、XAI 图片 handler 分流；补同步接受默认 API/Web rate limit 提升到 360/120 和默认 scanner buffer 128MB。仍保留本地 `StreamStatus` 非空不覆盖保护，不接受上游无条件重建流状态。 |
| `6f415428` | 2026-06-13 | 忽略 | 2026-06-11 02:36:41 +0800 | perf(web): improve frontend table rendering and pinned columns/UI table (#5405) | default-only data-table 大重构、列固定、静态表格和多页面迁移。本地 `web/default` 已删除，classic 不移植。 |
| `27b2b2c4` | 2026-06-13 | 忽略 | 2026-06-12 23:18:22 +0800 | perf(data-table): improve data table layout and badge display (#5460) | default-only data-table 布局与 badge 单元格优化，本地 classic 不移植。 |
| `d0c4305a` | 2026-06-14 | 部分合入 | 2026-06-12 23:40:40 +0800 | feat(audit): add localized security audit logs (#5462) | 按 classic 兼容路径移植审计基础：新增 `controller/audit.go`、`middleware/audit.go`、`LogTypeLogin`、`Other.op/admin_info/audit_info`、登录日志、用户/渠道/设置/兑换码/Passkey/2FA 等高价值手动埋点，以及管理写操作兜底审计。classic 使用日志已支持登录类型、审计动作/参数、路由/结果/admin_info 展开；普通用户查询会剥离 `admin_info`/`audit_info`。本地差异：写权限仍由 `RequireAdminRole`/`RootAuth` 控制，审计管理员只读；登录 IP 遵循用户 `record_ip_log` 设置。 |
| `1292b8b2` | 2026-06-14 | 部分合入 | 2026-06-12 23:45:15 +0800 | chore: update Codex channel (#5461) | 接受渠道命名为 `ChatGPT Subscription (Codex)`，同步后端渠道名与 classic 渠道选项/文案；不接受删除本地 Codex OAuth start/complete API、`CodexOAuthModal` 和授权按钮的行为收缩。classic 现在明确支持手填 JSON、OAuth 生成和编辑时刷新凭证三种路径。 |
| `15072292` | 2026-06-13 | 部分合入 | 2026-06-13 00:03:28 +0800 | fix: add deleted user status filter (#5464) | 手工移植后端用户搜索过滤。`SearchUsers` 在 `status=-1` 时查询软删除用户，普通 enabled/disabled 状态显式排除 `deleted_at IS NOT NULL`；新增测试覆盖 deleted/enabled/disabled 三种状态。上游 default 用户表筛选 UI 不移植。 |
| `51475c80` | 2026-06-13 | 忽略 | 2026-06-13 00:10:26 +0800 | fix: only fetch deployment settings on deployments page (#5466) | default-only 模型/部署页面请求优化，本地 classic 不移植。 |
| `1ac0f580` | 2026-06-14 | 合入 | 2026-06-13 16:12:22 +0800 | feat(audit): add authentication method tracking in audit logs | 随审计体系合入。`admin_info.auth_method` 记录 `session` / `access_token`，classic 使用日志详情展示为“会话”或“访问令牌”；不引入 default 使用日志组件。 |

## 2026-06-06 Rsbuild 迁移后核对

- 上游状态：执行 `git fetch upstream main --no-tags --prune` 后，`upstream/main` 仍为 `adc390c5`，没有比上表已记录的 `b5331936` / `adc390c5` 更新的提交。
- 本地修复：迁移到 Rsbuild / React 19 / Semi adapter 后，补齐 classic 前端 lint 闭环。`web/classic/.prettierignore` 排除 `node_modules`、`dist`、`build`、`coverage`，避免 Prettier 扫描 Rsbuild 构建产物；同时格式化 `web/classic/src/components/topup/subscriptionPaymentDisplay.test.mjs` 和 `web/classic/src/hooks/usage-logs/useUsageLogsData.jsx`。
- 验证结果：`cd web && bun run --cwd classic lint`、`cd web && bun run build`、`cd web && bun run --cwd classic i18n:status`、`git diff --check` 均通过。

## 2026-06-06 rc10 Docker 镜像构建修复

- 现象：`v1.0.0-rc.10` tag 已触发 `Publish Docker image` workflow，但镜像没有产出；失败点在 Dockerfile 前端 builder 阶段：`cd classic && VITE_REACT_APP_VERSION=$(cat /build/VERSION) bun run build`，Rspack 报错链路落到 `@lobehub/icons/es/features/ProviderIcon/DefaultIcon.js`。
- 原因：本地 classic-only workspace 在同步上游 `b596de73` 依赖集中管理时继承了 `@lobehub/icons ^5.10.0`。该版本面向上游 default 前端依赖图，peer dependency 要求 `@lobehub/ui ^5` 与 `antd ^6`；StuHelper AI classic 当前实际是 `@lobehub/ui 2.x` / `antd 5.x`。本机旧依赖目录可掩盖问题，clean Docker build 使用冻结锁文件重新安装后暴露该 peer graph 不匹配。
- 处理：`web/package.json` catalog 将 `@lobehub/icons` 固定回 classic 兼容的 `^2.1.0`，并刷新 `web/bun.lock`，使图标库、`@lobehub/ui` 与 `antd-style` 回到同一代 `antd 5` 依赖线。同步上游前端 workspace 时继续保留 classic-only 和该依赖约束，除非未来完整升级 `@lobehub/ui` / `antd` 并通过容器构建验证。
- 验证结果：`cd web && bun install --frozen-lockfile`、`cd web && bun run --cwd classic lint`、`cd web && bun run build`、`docker build --target builder --progress=plain -t stuhelper-ai-frontend-builder-test .` 均通过。

## 2026-06-06 上游同步后代码质量审查

- 审查范围：核对本轮上游同步、classic Rsbuild 迁移、站点元信息注入和支付通道相关覆盖层。重点检查是否存在已在同步台账中判定“不合入”的支付路径仍被运行时暴露，以及维护文档中未替换的覆盖层占位。
- Waffo Pancake：此前台账已明确上游 Waffo Pancake 支付提交不适配本地支付策略。审查发现 `/api/user/topup/info`、classic 支付设置和充值页仍存在可见或半可见分支；本轮移除 classic Waffo Pancake 设置组件、充值页创建/询价分支和 stale 注释路由，并让后端充值信息过滤历史 `PayMethods` 中的 `waffo_pancake`。保留底层历史支付方式识别，避免旧账单展示丢失，但不再提供新订单入口。
- 站点元信息：复查 `/favicon.ico` 动态处理后，将默认 favicon 选择集中到单一 helper，避免服务端 meta 注入和 favicon handler 出现重复兜底。
- 注册入口状态：复查 `b397c58ba` 的待决策项后补齐本地语义。`/api/status` 暴露全局注册开关和密码注册别名，classic 登录页与顶栏注册按钮不再只看自用模式，避免后台关闭新用户注册后仍展示注册入口。
- 维护文档：清理 `docs/local-overlays.md` 中前序提交留下的 `uncommitted` / `本次兼容` / `本次排序修复` 占位，并记录 Waffo Pancake 不暴露为本地在线支付通道、注册入口状态字段的同步保留点。

## 2026-06-13 上游同步初审记录

- 上游状态：执行 `git fetch upstream` 后，`upstream/main` 为 `1ac0f5807a8a538fa9a1ec1d86e692210243ed6a`。本轮以上次记录的 `adc390c5` 为同步基线，核对范围为 `adc390c5..1ac0f580` 共 34 个提交；继续采用选择性手工移植，不整体 merge 上游。
- 本轮合入：移植 `867d8acf` 的 Moonshot `kimi-k2.6` temperature 归一化；移植 `d2576ddc` / `59a93cf5` 中适配本地的 OpenAI Images API streaming、image edit multipart body 复用、图片 usage 归一化和 XAI 图片响应处理；移植 `15072292` 的用户搜索 deleted 状态过滤后端语义。
- 本轮保留本地差异：未引入 `web/default`；初审阶段暂未接受上游 `59a93cf5` 的默认 rate limit 提升、默认 128MB stream scanner buffer、以及无条件重建 `StreamStatus` 的行为。后续 2026-06-14 兼容补同步已统一接受 128MB 默认 stream scanner buffer 和 rate limit 提升，但仍保留本地既有 `StreamStatus` 非空不覆盖保护。
- 本轮跳过：大量 `web/default` only 的 model-pricing、dialog、data-table、public page、channel search、deployment settings 提交不适配本地 classic-only 前端；其中 model-pricing 相关提交已按规则复核 `pkg/billingexpr/expr.md`，确认未触碰后端表达式计费、存储、预扣或结算语义。
- 待决策：`d0c4305a` / `1ac0f580` 是完整安全审计体系和认证方式追踪，需要先决定 classic 日志展示、本地化、日志字段兼容和审计范围；`1292b8b2` 会删除现有 Codex OAuth 授权入口，只保留手填 JSON/刷新凭证，需确认是否接受该产品行为收缩。
- 验证结果：`go test ./relay/channel/moonshot ./relay/channel/openai ./relay/helper ./model -count=1` 通过；`git diff --check` 通过。

## 2026-06-14 兼容补同步记录

- 上游状态：再次执行 `git fetch upstream` 后，`upstream/main` 仍为 `1ac0f5807a8a538fa9a1ec1d86e692210243ed6a`，没有新增提交；继续不整体 merge 上游。
- 审计日志：按当前 classic 前端兼容合入 `d0c4305a` / `1ac0f580`。后端新增审计 helpers、写操作兜底、登录日志类型、`Other.op/admin_info/audit_info` 和 `auth_method`；classic 使用日志增加登录筛选、审计动作/参数、登录方式、User-Agent、认证方式、审计路由、HTTP 状态、操作结果和路由参数展示。普通用户日志仍剥离 `admin_info` / `audit_info`；审计管理员保持只读，写操作继续由 `RequireAdminRole` / `RootAuth` 拦截；登录 IP 遵循用户 `record_ip_log` 设置。
- Codex 渠道：按兼容路径处理 `1292b8b2`。接受渠道命名 `ChatGPT Subscription (Codex)`，同步后端、classic 渠道选项、密钥输入提示和授权弹窗标题；保留本地 OAuth start/complete API、`CodexOAuthModal` 和授权按钮，不接受上游删除 OAuth 引导的产品行为收缩。classic 现在明确支持手填 JSON、OAuth 生成和编辑时刷新凭证。
- OpenAI 图片流治理：补同步 `59a93cf5` 此前未接受项。接受 `GLOBAL_API_RATE_LIMIT=360`、`GLOBAL_WEB_RATE_LIMIT=120` 和默认 stream scanner buffer `128MB`；仍保留本地 `StreamStatus` 非空不覆盖保护，避免上游无条件重建流状态覆盖已有流式状态信息。
- default 前端复核：重新检查 `web/default` 相关提交。data-table、dialog wrapper、public page、deployment settings、combobox 主题等仍绑定 default 架构，classic 不直接移植；model-pricing 只吸收可落到 classic 的 UX 点：六位小数提示、价格输入 `inputMode='decimal'`、搜索 200ms 防抖与 IME 组合输入保护、原始表达式编辑区增高和等宽换行。按规则已复核 `pkg/billingexpr/expr.md`，这些改动只影响 classic 视觉编辑体验，不改后端表达式计费、存储、预扣或结算语义。
