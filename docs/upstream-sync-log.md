# 上游同步日志

本文档记录从 `QuantumNous/new-api` 同步到 StuHelper AI 分叉仓库的过程。

## 2026-05-13 - 待同步到 v1.0.0-rc.5

- 状态：待处理，尚未合并。
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

当前本地线尚未合并上面列出的 `v1.0.0-rc.5` 或更新的 `upstream/main` 提交。
执行待处理的上游同步时，必须明确检查并保留本地排行榜功能、classic 默认
前端策略、仅面向 release 的 GHCR workflow，以及身份清理改动。

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

- 合并策略：TBD
- 结果提交：TBD
- 冲突：TBD
- 验证：TBD
- 备注：本条记录只是同步前的规划基线。实际同步分支合并后，需要更新本节。
