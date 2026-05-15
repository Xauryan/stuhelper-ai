# 分叉仓库维护规范

本文档定义 StuHelper AI 分叉仓库如何长期跟进和维护
`QuantumNous/new-api` 上游仓库。

## 仓库信息

- 上游仓库：https://github.com/QuantumNous/new-api
- 上游远程名：`upstream`
- 本项目仓库：`git@github.com:Xauryan/stuhelper-ai.git`
- 本项目远程名：`origin`
- 主分支：`main`
- 项目身份：`StuHelper AI`
- 组织和作者身份：`Xauryan`

## 改动来源

本分叉仓库包含三类改动：

1. StuHelper AI 本地二开。
2. 从 `QuantumNous/new-api` 同步的上游发版更新。
3. 未被上游合并、但本项目需要引入的外部 PR 或补丁。

这些来源必须在提交、分支命名和维护日志中保持可区分。如果某个改动来自
第三方 PR，即使是手工移植，也必须记录到 `docs/external-prs.md`。

## StuHelper AI 本地基线

以下行为属于 StuHelper AI 产品基线。同步上游时必须保留，除非某个明确的
StuHelper AI 本地任务决定修改它们：

- 默认前端是 classic 前端。后端默认值、系统设置默认值、管理员 UI 默认值
  都应继续选择 `classic`，而不是新版 default 前端；除非管理员显式修改设置。
- classic 顶部导航包含排行榜入口，位置在模型广场之后、文档之前。该入口由
  管理员顶部导航设置管理，包括启用开关和可选的登录后可见开关。
- `audit_admin` 是 StuHelper AI 本地只读审计管理员角色，角色值为 `5`。该角色
  可进入 classic 管理区中的用户管理、兑换码管理、模型管理、渠道管理、订阅管理
  和邀请管理，并可查看全量使用日志、绘图日志、任务日志及日志统计，但只能查看
  列表、审计明细和执行查询。它不得看到系统设置、模型部署、充值订单管理、用户详情、渠道详情、兑换码
  详情、模型详情、供应商详情、日志清理或任何新增、编辑、删除、启用、禁用、测试、
  复制、同步、批量操作、订阅绑定、重置 Passkey、重置 2FA、注销等写操作入口。
  后端必须使用 `AuditAdminAuth` 只开放列表/搜索/统计接口，并使用
  `RequireAdminRole` 保护所有详情和写接口；用户列表允许展示邮箱用于审计查询，
  但必须剥离 OAuth 标识、访问 token、Stripe 客户号、用户设置、备注和单用户
  返佣覆盖等敏感字段；渠道、兑换码等资源列表必须剥离密钥、Base URL、覆写
  配置、备注等敏感字段。审计管理员的渠道搜索和标签搜索不得按密钥或 Base URL
  命中，避免通过搜索结果推断隐藏渠道信息。新增用户只能创建普通用户；
  审计管理员身份必须由超级管理员在 classic 用户管理中通过“提升/降级”做单级
  身份调整：普通用户提升为审计管理员，审计管理员提升为管理员，管理员降级为
  审计管理员，审计管理员降级为普通用户。普通管理员不得通过新增用户、编辑用户
  或管理接口创建、提升、降级审计管理员身份，也不得查看审计管理员详情、禁用、
  删除、调整额度、清理绑定、重置 Passkey/2FA 或管理其订阅。Web session 鉴权
  必须在请求时重读当前用户角色和状态，避免角色降级或账号禁用后旧 cookie 继续
  保留管理员权限。
- classic 个人设置页的账户绑定区域只展示当前 `/api/status` 中已启用的第三方
  绑定入口。GitHub、Discord、OIDC、微信、Telegram、LinuxDO 和自定义 OAuth
  等入口必须跟随对应状态开关隐藏或显示，不能把禁用入口作为“未启用”卡片展示给
  普通用户。
- 用户排行榜包含消耗排行和充值排行，并支持总榜、近一月、近一周、近一天
  统计周期。为避免反推出本站总消耗或总充值，用户排行榜接口和 classic 页面
  不得暴露全站周期总额或每个上榜用户的占比。充值排行包括成功在线充值、
  兑换码兑换、管理员手动增加余额；管理员手动增加余额既要统计新日志中的
  `logs.quota`，也要兼容历史日志内容中记录的额度。
- GHCR 镜像工作流是面向发布版本的。它只在版本 tag 或针对既有版本 tag 的
  显式手动触发时构建，只发布版本 tag 镜像和 `latest`，不发布 `main` 或
  commit-SHA 镜像 tag。`latest` 应指向最新发布版本镜像。
- classic 额度充值支持支付宝和微信支付官方企业接入。支付宝使用电脑网站支付
  和手机网站支付；微信支付使用 Native 扫码和 H5 跳转。该功能是 StuHelper AI
  本地基线，维护细节见 `docs/official-cn-payments.md`，同步上游或导入外部
  PR 时不得替换为易支付或支付宝当面付实现。
- classic 钱包页同时存在“额度充值”和“订阅套餐”时，必须默认进入“额度充值”，
  且“额度充值”位于“订阅套餐”左侧。订阅套餐卡片的“推荐”标签和高亮边框必须由
  classic 后台“订阅管理”的“推荐”开关控制，不能再默认给第一个套餐；同步上游或
  调整订阅 UI 时必须保留这个手动运营能力。
- 充值页支付方式必须按实际网关配置展示：易支付未完整配置时不得展示易支付的
  支付宝、微信方式；官方支付宝和官方微信必须在开关和必填密钥、公钥、商户信息
  均完整时才展示，不能只按开关展示。官方支付敏感密钥不从 `/api/option/` 回显，
  只能通过 `*Configured` 状态判断是否已有密钥可保留。充值单价支持三位小数保存，
  实际支付金额必须按进一法保留到两位小数或分单位。
- 邀请奖励支持一次性奖励和充值返佣独立叠加。`QuotaForInvitee` 大于 0 时，
  被邀请用户注册后实时获得邀请码奖励；`QuotaForInviter` 大于 0 时，邀请人获得
  一次性邀请奖励，可通过 classic 运维设置中的
  `InviterRewardAfterPaymentEnabled` 延迟到被邀请用户首次充值或购买订阅成功后
  解锁到账，解锁额度使用注册时写入的 `inviter_reward_quota` 快照。已有邀请
  关系仅在首次新增延迟奖励状态字段的迁移中标记为已处理，不对历史用户补发。
  `ReferralCommissionEnabled`
  仅控制按被邀请用户充值和订阅支付金额额外返佣，不替代一次性邀请奖励。全局
  返佣比例和最大返佣次数在 classic 运维设置中配置；管理员可在 classic 用户
  编辑页为单个邀请人设置 `referral_commission_percent` 覆盖比例。支付完成后的
  邀请人奖励解锁和返佣必须随支付完成事务一起写入，返佣通过
  `source_type + source_id + invitee_id + payment_method` 幂等，避免重复 webhook
  或订阅订单重复回调造成重复入账。支付退款必须同步冲销充值返佣的净额统计；
  充值全额退款时，若该支付触发了一次性邀请人奖励解锁，必须回滚为待解锁状态；
  注册时即时发放的一次性邀请奖励不随退款撤回。不再把全额退款订单计为有效首充。
  所有注册入口，包括密码注册、统一 OAuth、旧 GitHub 和 LinuxDO 登录，都必须在
  创建被邀请用户时持久化 `users.inviter_id`；否则邀请管理、用户列表邀请人展示、
  后续首充解锁和返佣都会失去关系来源。邀请管理查询以被邀请用户的
  `inviter_id > 0` 为准，并使用左连接读取邀请人展示信息，避免邀请人账号被删除后
  丢失审计记录。
  classic 管理员菜单必须保留“邀请管理”，用于
  分页审计所有邀请关系、被邀请用户奖励快照、被邀请用户首充/订阅状态、邀请人
  一次性奖励解锁状态，以及每条邀请关系下的返佣记录。
- 项目身份必须保持为 `StuHelper AI`；组织、作者、联系方式、包名、Docker、
  workflow 和元数据身份必须保持为 `Xauryan`。
- 对外分享元数据必须保持 StuHelper AI 身份。classic 和 default 前端的页面
  `<title>`、`description`、Open Graph 和 Twitter Card 标签不得重新出现旧项目
  名称或旧产品描述，避免 Telegram、QQ 等聊天工具抓取旧分享卡片。
  当前默认分享描述为：`StuHelper AI 是 StuHelper 团队部署的统一 AI 模型聚合与分发网关，提供高性价比的集中式模型管理与网关服务。`
- classic 页脚必须保留“设计与开发由 StuHelper AI”右侧署名。系统设置 -> 其他设置
  中的 `Footer` 继续作为自定义 HTML 页脚；当 `Footer` 留空时，才使用默认页脚
  模板字段生成 ICP 备案、增值电信业务经营许可证和版权信息。备案号、许可证号
  或版权字段为空时不显示对应项；移动端必须允许这些内容换行，避免长备案号或
  许可证号挤压右侧署名。
- classic 默认页脚模板中的许可证类型只把 `ICP` / `EDI` 作为内部选项值，页面
  和页脚展示必须使用中文业务名称：`ICP` 显示为“互联网信息服务业务经营许可
  证”，`EDI` 显示为“增值电信业务经营许可证—在线数据处理与交易处理业务”。
  备案号和许可证号输入框不得放置示例号码作为默认提示。
- classic 前端在 Vite 8 下仍有自定义 `treat-js-files-as-jsx` 插件调用
  `transformWithEsbuild`。Docker/GitHub Actions 干净环境必须安装显式
  `esbuild` devDependency，否则 classic 构建会因为 Vite 无法解析 `esbuild`
  而失败；后续升级到 Oxc 转换前不要移除该依赖。
- 前端构建目录必须使用 `go:embed all:` 嵌入。Vite/Rolldown 会生成
  `_arrayReduce-*`、`_baseSlice-*` 等以下划线开头的 chunk；普通目录
  `go:embed` 会排除这些文件，导致生产环境 `index.html` 引用的 `/assets/_*.js`
  返回 404。

只有在标识上游来源、上游 release 或导入的上游 PR 时，才允许引用原上游仓库。
这些引用不得重新作为本分叉仓库的产品品牌、包身份、镜像名称、可见 UI 文案
或归属信息出现。

## 分支策略

上游同步和外部补丁导入使用短生命周期分支：

- `main`：StuHelper AI 稳定主线。
- `sync/upstream-vX.Y.Z`：同步某个上游 release tag 的分支。
- `patch/upstream-pr-NNNN`：导入某个上游 PR 或外部补丁的分支。
- `feature/<topic>`：StuHelper AI 本地功能开发分支。

不要在脏的 `main` 工作树中直接同步上游。同步前应先提交、stash，或把无关
本地改动移动到单独分支。

## 上游同步策略

优先同步上游 release tag，而不是随意同步某个上游提交。只有在需要紧急修复，
或明确决定跟进 release 之后的上游工作时，才使用 `upstream/main`。

StuHelper AI 当前只使用 classic 前端作为产品前端。后续同步上游时，纯
`web/default` 新版前端的 UI、布局、交互和组件修复默认跳过，不需要反复评估；
只有当这些改动同时影响后端接口契约、安全边界、共享构建链路、共享类型或
classic 前端实际用户路径时，才进入同步候选范围。

上游运营策略类功能也应先判断是否符合 StuHelper AI 产品基线。已经明确跳过的
策略改动应记录到 `docs/upstream-sync-log.md`，后续同步时按记录处理，不重复
展开分析。例如，上游 `0526a226` 的付费功能合规确认总门禁会在管理员确认前
锁住充值、兑换码、订阅和邀请奖励能力；该策略不属于当前 StuHelper AI 刚需，
且会干扰本地官方支付、订阅和邀请返佣路径，默认不引入。

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

如果某个 release 冲突太多，应按领域审查上游提交，并考虑分组 cherry-pick。
实际采用的策略必须记录到 `docs/upstream-sync-log.md`。

## 冲突处理策略

解决上游冲突时：

- 保留 `StuHelper AI` 项目品牌。
- 保留 `Xauryan` 组织、作者、包、服务和元数据身份。
- 不得重新引入旧项目名、旧服务名、旧 Docker 镜像名、旧 Go module path、
  旧前端标题、旧页脚文案或旧版权联系方式。
- 保留本地计费、订阅、Codex、OAuth、排行榜、支付、仪表盘、i18n 和部署行为，
  除非明确选择用上游改动替换它们。
- 生成文件与源文件分开处理。如果项目工具链要求生成，应重新生成，不要手工
  大规模编辑生成产物。
- 数据库相关改动必须同时保持 SQLite、MySQL 和 PostgreSQL 兼容。

如果某个冲突无法明确判断，应在同步日志中留下说明，而不是静默选择某一边。

## 外部 PR 策略

对于上游尚未合并的 PR：

1. 阅读 PR 描述、diff、提交历史和 review comments。
2. 判断应该 cherry-pick 还是手工移植。
3. 如果行为发生变化，添加聚焦测试。
4. 在 `docs/external-prs.md` 记录导入的补丁。
5. 后续每次同步上游时，检查上游是否已经吸收该补丁。

如果上游后来合并了等价修复，应在下一次同步时协调本地补丁，并更新外部 PR
记录。

## 验证策略

根据触及文件选择验证命令。常用命令：

```powershell
go test ./...
go test ./middleware -count=1
Set-Location web/default; bun run typecheck
Set-Location web/default; bun run build
Set-Location web/default; bun run i18n:sync
```

如果全量命令因为仓库已有状态失败，应记录失败原因，以及能够证明本次变更
范围的更窄验证命令。

## 本分叉仓库的发布说明

每个 StuHelper AI 本地发布都应该可以追溯到：

- 对应的上游 release tag 或上游提交范围。
- 包含的 StuHelper AI 本地提交。
- 包含的外部 PR。
- 验证命令和已知失败。

当某个本地版本基于上游 tag 加本地分叉改动发布时，可使用类似
`stuhelper-v1.0.0-rc.5-sync.1` 的 tag。
