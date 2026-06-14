# 渠道 failover 与熔断维护规则

本页记录 relay 自动 failover、流式异常上报和渠道熔断的本地维护规则。

## 默认重试次数

`common.RetryTimes` 的代码默认值保持 `0`。部署方可以通过系统配置启用普通同分组重试；代码层不要把默认值硬改为非零，避免在未显式配置时增加上游请求次数和成本。

`auto` 分组的跨分组重试是独立保护：即使 `RetryTimes=0`，只要令牌开启 `cross_group_retry`，当前真实分组返回全局可重试错误时仍可以推进到后续真实分组。

## 熔断配置

渠道熔断配置使用 `CHANNEL_BREAKER_*` 环境变量：

- `CHANNEL_BREAKER_ENABLED`
- `CHANNEL_BREAKER_WINDOW_SECONDS`
- `CHANNEL_BREAKER_MIN_SAMPLES`
- `CHANNEL_BREAKER_TRIP_SCORE_PCT`
- `CHANNEL_BREAKER_CONSECUTIVE_FATAL`
- `CHANNEL_BREAKER_COOLDOWN_SECONDS`
- `CHANNEL_BREAKER_MAX_COOLDOWN_SECONDS`
- `CHANNEL_BREAKER_HALFOPEN_PROBES`

这些配置必须在 `.env` 加载和 `common.InitEnv()` 后通过 `service.InitChannelBreakerConfig()` 初始化，不能在包级变量初始化时直接读取环境变量，否则 `.env` 中的配置不会生效。

数值型配置必须保留防御式校验：非正值回退到默认值，百分比限制在 `1..100`，`max_cooldown` 不小于 `cooldown`。这可以避免 bucket 时间片除零、永久 open 或异常频繁探测。

## 渠道恢复

手动启用渠道、自动测试重新启用渠道，以及按 tag 批量启用渠道后，都必须清理该渠道的 in-memory breaker 状态。DB 状态恢复为 enabled 后，选择层不应继续因为旧 breaker open 状态排除该渠道。

## 流式异常

`StreamScannerHandler` 会在 `RelayInfo.StreamStatus` 中记录流结束原因。timeout、scanner error、panic、ping fail 等异常结束必须通过 `helper.StreamInterruptionError` 转为 `channel:stream_interrupted`，让上层可以记录渠道失败、阻止静默成功，并按已提交响应状态决定是否重试。

如果响应已经向客户端写出，relay 不得切换到其它渠道重试；错误只能通过协议内事件写回。新增或修改流式 handler 时，读取结束后必须检查 `StreamInterruptionError`，不要只处理 JSON 解析错误。
