# 订阅计费维护规则

本页记录订阅额度预扣、成功后结算、钱包/订阅回退和缓存一致性的本地维护规则。

## 预扣与结算边界

订阅计费分为两个阶段：

1. 预扣阶段：请求发往上游前执行，用于准入控制。`model.PreConsumeUserSubscription` 和 `model.PostConsumeUserSubscriptionDelta` 必须保持严格限制，不能让 `amount_used` 超过 `amount_total`。如果剩余额度不足，应拒绝本次请求，或按用户计费偏好回退到钱包。
2. 成功后结算阶段：请求或异步任务已经成功，真实用量才可得。`model.SettleUserSubscriptionDelta` 允许正向 delta 把 `amount_used` 写到 `amount_total` 以上，确保已发生的用量完整记账。后续预扣会看到该订阅没有剩余额度，不能继续使用。

不要把成功后结算改回严格拒绝超总额。否则当预估用量低于真实用量、订阅在预扣后刚好耗尽时，`BillingSession.Settle` 或异步任务 `RecalculateTaskQuota` 会补扣失败并只记录错误，形成少扣费。

## BillingSession

`service.BillingSession` 是同步 relay 的统一计费生命周期：

- `PreConsumeBilling` 创建 session 并写入 `relayInfo.Billing`。
- 钱包计费在预扣和结算时调整用户钱包额度。
- 订阅计费在预扣时调用严格的订阅预扣路径，结算时通过 `SubscriptionFunding.Settle` 调用 `model.SettleUserSubscriptionDelta`。
- 令牌额度仍跟随实际预扣和结算 delta 调整；资金来源已结算但令牌调整失败时，只记录错误，不能再触发退款。

异步任务完成后的 `RecalculateTaskQuota` 也属于成功后结算，订阅来源必须使用 `model.SettleUserSubscriptionDelta`，而不是预扣阶段的严格扣减接口。

## 展示与日志

订阅用量允许在最终结算后超过总额，但对外展示剩余额度必须夹到 0：

- 消费日志 `subscription_remain` 不显示负数。
- classic 订阅卡片的使用百分比封顶为 100%。
- 管理端仍可以看到原始 `amount_used / amount_total`，用于审计超额结算是否来自真实请求。

## Redis quota 缓存

用户钱包额度的热路径使用 Redis `HINCRBY` / `HDECRBY` 类原子增减。`model.GetUserQuota(id, fromDB=true)` 从数据库 fallback 读到的是快照，不能异步写回 Redis，否则可能覆盖并发请求刚刚写入的额度 delta。

DB fallback 成功后只允许异步失效用户缓存，让下一次读取重新加载；不要恢复旧的 `updateUserQuotaCache(id, quota)` 回写逻辑。

Redis 未启用时，`IncreaseUserQuota` / `DecreaseUserQuota` 不应派发异步 quota cache 增减任务；Redis 启用状态必须在同步路径判断，避免后台 goroutine 在测试清理或运行时配置恢复阶段再读取全局开关。

## 同步注意事项

同步上游计费、订阅或任务轮询逻辑时必须保留：

- 订阅预扣严格限制、成功后结算允许记录超额使用的两阶段语义。
- `BillingSession` 和异步任务重算都走 settlement 专用订阅 delta。
- Redis quota fallback 只失效缓存，不用 DB 快照覆盖 Redis。
- Redis 未启用时不要派发 quota cache 增减 goroutine；Redis 启用时仍必须使用原子增减。
- classic 订阅展示对剩余额度和百分比做非负/封顶处理。
