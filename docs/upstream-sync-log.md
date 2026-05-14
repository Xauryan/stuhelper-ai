# 上游同步日志

本文档记录从 `QuantumNous/new-api` 同步到 StuHelper AI 分叉仓库的过程。

## 2026-05-14 - 检查 upstream/main `18282e61`，无需合入

- 状态：已检查上次同步点 `aa56667b` 之后的上游提交；本次不引入代码改动。
- 当前 `upstream/main`：`18282e61`
- 上游 tag：`v1.0.0-rc.6`
- 当前本地 `HEAD`：`0e3546ba`
- 检查范围：`aa56667b..18282e61`

### 本次明确跳过的上游提交

- `0526a226` feat: require compliance confirmation for paid features
  - 结论：不引入。
  - 原因：该提交新增的是上游运营策略性质的付费功能合规确认总门禁，
    不是安全漏洞修复、支付正确性修复或 StuHelper AI 当前产品刚需。
    它会在 root 管理员确认前锁住或隐藏充值、兑换码、订阅购买和邀请奖励
    等付费相关能力，影响本地已经维护的官方支付宝/微信、订阅模型限制、
    邀请返佣、首充后奖励解锁和 classic 充值体验，侵入面与维护成本过高。
  - 后续处理：后续同步上游时不要重复评估该提交，除非 StuHelper AI 明确决定
    引入类似的本地运营合规确认流程。
- `51b5cbe1` fix: prevent combobox from over-filtering options on focus (#4829)
  - 结论：不引入。
  - 原因：该提交只修复新版 `web/default` 前端的 combobox 聚焦过滤行为；
    StuHelper AI 当前产品基线是只使用 classic 前端，default-only UI 细节
    不作为同步目标。
  - 后续处理：后续同步上游时，default-only 前端修复默认跳过，除非同时影响
    后端接口契约、安全边界、共享构建链路，或 classic 前端实际用户路径。

### 本地已满足目标状态的上游提交

- `3e588b4d` chore(deps-dev): bump ip-address from 10.1.0 to 10.2.0 in /electron (#4811)
  - 本地 `electron/package-lock.json` 已经包含 `ip-address` `10.2.0`。
  - 不应按上游整文件覆盖，因为本地 lockfile 还保留 StuHelper AI 项目身份和
    后续依赖更新。
- `18282e61` chore(deps): update axios from 1.15.0 to 1.15.2
  - 本地 `web/classic/package.json` 和 `web/classic/bun.lock` 已经包含
    `axios` `1.15.2`。
  - 不应按上游整文件覆盖，因为本地 classic 前端保留 Vite 8、显式
    `esbuild` devDependency、Bun lockfile 镜像源记录和 StuHelper AI 构建基线。

### 本次检查命令

```powershell
git fetch --prune --tags https://github.com/QuantumNous/new-api.git +refs/heads/main:refs/remotes/upstream/main
git rev-parse --short refs/remotes/upstream/main
git log --oneline --reverse aa56667b..refs/remotes/upstream/main
rg -n "Compliance|compliance|RiskAcknowledgement|payment_compliance" controller model setting router web docs i18n -S
rg -n "axios|ip-address|10\.1\.0|10\.2\.0|1\.15\.0|1\.15\.2" web/classic/package.json web/classic/bun.lock electron/package-lock.json -S
```

备注：本次检查只更新维护记录，不修改业务代码；无需运行构建或测试。

## 2026-05-13 - 已同步到 upstream/main `aa56667b`

- 状态：已按语义分批 cherry-pick / 手工移植，已提交。
- 上游 release：https://github.com/QuantumNous/new-api/releases/tag/v1.0.0-rc.5
- 上游 release 日期：2026-05-12
- 上游 tag commit：`469d3747`
- 当前本地 `HEAD`：`ba474393`
- 当前 `origin/main`：`ba474393`
- 当前 `upstream/main`：`aa56667b`
- 相对 `v1.0.0-rc.5` 落后提交数：`8`
- 相对 `upstream/main` 落后提交数：`11`
- 推荐同步分支：`sync/upstream-v1.0.0-rc.5`
- 推荐同步目标：`v1.0.0-rc.5`

### v1.0.0-rc.5 中本地 HEAD 尚未包含的上游提交

- `19fc384e` feat(performance): update performance metrics handling and UI components
- `03d53732` fix(default): improve performance health panel layout
- `3057f04a` fix(wallet): read topup gateway flags from topupInfo instead of status (#4599)
- `7fe896d2` fix: use getUserGroups for ratio display to respect GroupGroupRatio (#4772)
- `2b89989f` fix(default): support DropdownMenuItem onSelect (#4787)
- `fde2cac9` fix(web/default): guard playground messages against legacy classic shape (#4650)
- `a720064d` Merge branch 'main' of github.com:QuantumNous/new-api
- `469d3747` fix: defaut ui triage (#4802)

### v1.0.0-rc.5 之后的上游提交

以下提交位于 release tag 之后的 `upstream/main`，不属于推荐的 release tag
同步范围，除非明确选择纳入。

- `3856b9d2` chore(deps): bump axios from 1.15.0 to 1.15.2 in /web/classic (#4634)
- `428e3d91` chore: refresh related resources
- `aa56667b` feat: track upstream request ID and prevent response header override

### 必须保留的本地改动

- StuHelper AI 项目身份。
- Xauryan 组织和作者身份。
- 本地 Go module/import path 改动。
- Docker、service、workflow 和部署命名改动。
- 仅面向 release 的 GHCR 镜像发布策略：发布版本 tag 和 `latest`，不发布
  `main` tag，不发布 commit-SHA tag。
- classic 前端作为后端默认值、系统设置默认值和管理员 UI 默认值中的默认前端。
- classic 前端排行榜页面，以及位于模型广场之后、文档之前的顶部导航入口。
- 排行榜的管理员顶部导航管理能力，包括启用开关、登录后可见开关，以及对
  旧布尔配置的兼容。
- 用户消耗和充值排行榜聚合逻辑，其中充值总额来源包括成功充值、兑换码兑换
  和管理员手动增加余额。
- 本地计费、订阅、支付、Codex、OAuth、排行榜、仪表盘和 i18n 改动。
- `docs/external-prs.md` 中列出的外部补丁。

### 同步前的本地覆盖层状态

本次同步执行前，本地线尚未合并上面列出的 `v1.0.0-rc.5` 或更新的
`upstream/main` 提交。同步时已明确保留本地排行榜功能、classic 默认前端
策略、仅面向 release 的 GHCR workflow，以及身份清理改动。

### 生成本条记录时使用的预同步命令

```powershell
git fetch upstream --tags --prune
git rev-parse --short HEAD
git rev-parse --short upstream/main
git rev-parse --short 'v1.0.0-rc.5^{commit}'
git rev-list --left-right --count HEAD...v1.0.0-rc.5
git rev-list --left-right --count HEAD...upstream/main
git log --oneline HEAD..v1.0.0-rc.5
git log --oneline v1.0.0-rc.5..upstream/main
```

### 同步结果

- 合并策略：按语义分批 cherry-pick / 手工移植；不直接合并
  `upstream/main`，避免恢复上游 README、workflow、Go module/import path、
  项目身份和 classic/default 前端策略。
- 结果提交：
  - `53e31839` feat: track upstream request ids
  - `49c8c933` fix: make payment return paths theme-aware
  - `3d2f70a8` fix(default): port upstream UI compatibility fixes
  - `fce87074` fix(default): port route guards and ranking access checks
  - `8bfbaa28` feat(default): refresh performance dashboard
  - `d4d32822` chore: refresh upstream related resources
- 覆盖的上游提交：
  - `19fc384e` feat(performance): update performance metrics handling and UI components
  - `03d53732` fix(default): improve performance health panel layout
  - `3057f04a` fix(wallet): read topup gateway flags from topupInfo instead of status (#4599)
  - `7fe896d2` fix: use getUserGroups for ratio display to respect GroupGroupRatio (#4772)
  - `2b89989f` fix(default): support DropdownMenuItem onSelect (#4787)
  - `fde2cac9` fix(web/default): guard playground messages against legacy classic shape (#4650)
  - `469d3747` fix: defaut ui triage (#4802)
  - `3856b9d2` chore(deps): bump axios from 1.15.0 to 1.15.2 in /web/classic (#4634)
  - `428e3d91` chore: refresh related resources
  - `aa56667b` feat: track upstream request ID and prevent response header override
- 未直接合并：
  - `a720064d` 只是上游 merge commit，本次按其有效内容所在提交移植。
  - 上游多语言 README 恢复/改写没有移植；本分叉当前只跟踪本地
    `README.md`，且必须保持 StuHelper AI/Xauryan 身份。
  - 上游 CI/workflow、Docker 镜像发布策略、Go module/import path 和大范围
    品牌信息没有移植；这些会覆盖本地二开基线。
- 排行榜说明：上游本次排行榜相关改动主要是 `web/default` 新版前端路由守卫
  和 `/api/rankings` 后端开关校验。本地 classic 前端排行榜页面、
  `/api/rankings/users`、classic 顶栏入口和后台管理开关已保留。
- 验证：
  - `go test ./common ./model ./service ./relay/channel ./relay/channel/minimax ./relay/channel/openai ./controller -run TestDoesNotExist -count=1`
  - `go test ./common ./service ./controller -run TestDoesNotExist -count=1`
  - `go test ./pkg/perf_metrics ./controller -run TestDoesNotExist -count=1`
  - `bun test src/components/ui/dropdown-menu.test.tsx`
  - `bun run typecheck`（`web/default`）
  - `node` 解析 `web/default/src/i18n/locales/{en,fr,ja,ru,vi,zh}.json`
  - `git diff --check`
- 备注：由于本地品牌、module path、classic 默认前端、classic 排行榜和 GHCR
  发布策略与上游不同，`git log --cherry-pick --right-only HEAD...upstream/main`
  仍可能列出上游提交；判断同步状态应以本节的语义覆盖记录为准。

### 推送前合入 origin 依赖更新

- 背景：推送到 `origin/main` 前，本地 `main` 比 `origin/main` 落后 2 个提交。
- 合入方式：普通 merge `origin/main`，不 rebase，不强推，保留本地同步提交历史。
- 合入提交：
  - `d8bf61f4` chore(deps): bump the npm_and_yarn group across 2 directories with 4 updates
  - `1b18758a` Merge pull request #1 from Xauryan/dependabot/npm_and_yarn/web/classic/npm_and_yarn-7f8752592c
- 影响范围：
  - `web/classic/package.json`：保留 `axios` `1.15.2`，合入 classic 前端 `vite` 依赖更新。
  - `web/classic/bun.lock`：补齐 dependabot 提交未更新的 Bun lockfile，使其与
    `package.json` 保持一致。
  - `web/classic/vite.config.js`：将 `manualChunks` 改为函数形式，保持原有分包
    归类，同时兼容 Vite 8/Rolldown 的输出配置校验；并将 `roughjs` 解析到
    ESM 入口，避免 Vite 8 选择 browser UMD 入口后缺少默认导出导致 classic
    构建失败。
  - `electron/package-lock.json`：合入 dependabot 间接依赖更新。
- 本地覆盖层：未改动 StuHelper AI/Xauryan 身份、classic 默认前端、classic 排行榜、
  GHCR release-only 发布策略和上游同步移植记录。
