# 分叉仓库维护规范

本文档只保留 StuHelper AI 分叉仓库的维护流程。具体本地覆盖层不要继续写在
本文档里，统一记录到 `docs/local-overlays.md`，并按 commit hash 追溯。

## 仓库信息

- 上游仓库：https://github.com/QuantumNous/new-api
- 上游远程名：`upstream`
- 本项目仓库：`git@github.com:Xauryan/stuhelper-ai.git`
- 本项目远程名：`origin`
- 主分支：`main`
- 项目身份：`StuHelper AI`
- 组织和作者身份：`Xauryan`

## 文档分层

维护信息按来源分开记录：

- `docs/local-overlays.md`：StuHelper AI 本地二开和长期覆盖层，使用
  commit hash + 变更内容记录。未提交改动先写 `uncommitted`，提交后替换为
  真实 hash。
- `docs/external-prs.md`：手工移植的第三方 PR、上游未合并 PR 或外部补丁。
- `docs/upstream-sync-log.md`：每次上游 release/main 同步、跳过原因、冲突和
  验证结果。
- 领域文档：支付、计费、部署等专题行为写入对应专题文档，例如
  `docs/official-cn-payments.md`。

原则：不要把长篇功能细节塞回本文档；本文档只回答“怎么维护”，具体行为从
commit 和分层文档追溯。

## 改动来源

本仓库改动分为三类：

1. StuHelper AI 本地二开，记录到 `docs/local-overlays.md`。
2. 从 `QuantumNous/new-api` 同步的上游 release/main，记录到
   `docs/upstream-sync-log.md`。
3. 未被上游合并、但本项目需要引入的外部 PR 或补丁，记录到
   `docs/external-prs.md`。

如果某个改动来自第三方 PR，即使是手工重写，也必须记录到
`docs/external-prs.md`，不要混入本地覆盖层索引。

## 分支策略

- `main`：StuHelper AI 稳定主线。
- `sync/upstream-vX.Y.Z`：同步某个上游 release tag 的短生命周期分支。
- `patch/upstream-pr-NNNN`：导入某个上游 PR 或外部补丁的短生命周期分支。
- `feature/<topic>`：StuHelper AI 本地功能开发分支。

不要在脏的 `main` 工作树中直接同步上游。同步前应先提交、stash，或把无关
本地改动移动到单独分支。

## 上游同步策略

优先同步上游 release tag。只有在需要紧急修复，或明确决定跟进 release 之后的
上游工作时，才使用 `upstream/main`。

同步前执行：

```powershell
git fetch upstream --tags --prune
git status --short --branch
git rev-list --left-right --count HEAD...<upstream-tag>
git log --oneline HEAD..<upstream-tag>
```

创建独立同步分支：

```powershell
git switch -c sync/upstream-<upstream-tag>
git merge <upstream-tag>
```

同步时按以下顺序判断：

1. 先查 `docs/local-overlays.md`，确认本地覆盖层是否受影响。
2. 再查 `docs/external-prs.md`，确认外部补丁是否已被上游吸收或冲突。
3. `web/default` 已从 StuHelper AI 中移除；涉及该目录的上游前端改动默认跳过。
   如果同时影响后端接口、安全边界、共享构建链路或 classic 实际用户路径，再
   单独评估并按 classic 结构重写。
4. 上游运营策略类改动必须先判断是否符合 StuHelper AI 产品基线；跳过原因记录到
   `docs/upstream-sync-log.md`，避免后续重复评估。

如果某个 release 冲突太多，应按领域审查上游提交，并考虑分组 cherry-pick。
实际采用的策略必须记录到 `docs/upstream-sync-log.md`。

## 冲突处理清单

解决上游冲突时必须保留：

- `StuHelper AI` 项目品牌。
- `Xauryan` 组织、作者、包、服务和元数据身份。
- `docs/local-overlays.md` 中列出的本地覆盖层，除非明确决定替换或删除。
- SQLite、MySQL 和 PostgreSQL 同时兼容的数据库行为。

不得重新引入旧项目名、旧服务名、旧 Docker 镜像名、旧 Go module path、旧前端
标题、旧页脚文案或旧版权联系方式。

生成文件与源文件分开处理。如果项目工具链要求生成，应重新生成，不要手工大规模
编辑生成产物。如果某个冲突无法明确判断，应在同步日志中留下说明，而不是静默
选择某一边。

## 外部 PR 策略

对于上游尚未合并的 PR：

1. 阅读 PR 描述、diff、提交历史和 review comments。
2. 判断应该 cherry-pick 还是手工移植。
3. 如果行为发生变化，添加聚焦测试。
4. 在 `docs/external-prs.md` 记录导入补丁、涉及文件、验证和未来同步检查点。
5. 后续每次同步上游时，检查上游是否已经吸收该补丁。

如果上游后来合并了等价修复，应在下一次同步时协调本地补丁，并更新外部 PR
记录。

## 验证策略

根据触及文件选择验证命令。常用命令：

```powershell
go test ./...
go test ./middleware -count=1
Set-Location web
bun install --frozen-lockfile
Set-Location classic
bun run lint
bun run build
bun run i18n:status
Set-Location ../..
docker build --target builder --progress=plain -t stuhelper-ai-frontend-builder-test .
```

涉及发布 tag、Dockerfile、前端 workspace 依赖或构建链路时，必须额外执行上面的
`docker build --target builder`，确认 clean Docker 前端构建和本机
`node_modules` 状态无关。

如果全量命令因为仓库已有状态失败，应记录失败原因，以及能够证明本次变更范围的
更窄验证命令。

## 发布说明

每个 StuHelper AI 本地发布都应该可以追溯到：

- 对应的上游 release tag 或上游提交范围。
- 包含的 StuHelper AI 本地提交。
- 包含的外部 PR。
- 验证命令和已知失败。

当某个本地版本基于上游 tag 加本地分叉改动发布时，可使用类似
`stuhelper-v1.0.0-rc.5-sync.1` 的 tag。
