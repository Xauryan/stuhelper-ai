# 渠道 failover 与熔断维护规则

本页记录 relay 自动 failover、流式异常上报和渠道熔断的本地维护规则。

## 默认重试次数

`common.RetryTimes` 的代码默认值保持 `0`。部署方可以通过系统配置启用普通同分组重试；代码层不要把默认值硬改为非零，避免在未显式配置时增加上游请求次数和成本。

`auto` 分组的跨分组重试是独立保护：即使 `RetryTimes=0`，只要令牌开启 `cross_group_retry`，当前真实分组返回全局可重试错误时仍可以推进到后续真实分组。

## 错误分类

relay 错误重试、`auto` 跨分组推进、渠道亲和失败回退、自动禁用判断和熔断计分都必须复用 `service.ClassifyRelayError`。不要在各调用点重新拼一套状态码和错误码判断，否则 `skip-retry`、`always-skip`、渠道侧失败和临时失败会再次出现策略漂移。

分类约定：

- `channel`：明确的渠道侧错误，包含 `types.IsChannelError`、自动禁用状态码和自动禁用关键词。用于强制重试、亲和 de-pin、自动禁用和熔断 fatal 计分。
- `transient`：临时上游错误，例如全局可重试状态码或非法 upstream status。用于普通 retry 和熔断 transient 计分。
- `skip`：显式 `skip-retry`、全局 always-skip 错误码/状态码，以及异常的 2xx 错误。不得触发 retry、亲和回退或熔断惩罚。
- `client`：其余业务/请求错误。不得惩罚渠道。

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

multi-key 渠道按 tag 批量启用时必须逐个调用 `UpdateChannelStatus`，不能只批量更新 `channels.status`，否则 `handlerMultiKeyUpdate` 不会清理单 key 禁用状态，渠道会出现 DB 已启用但可用 key 仍被禁用的假恢复。

## 可见性

后台渠道列表和搜索接口会在返回的 `Channel` 上附加只读字段 `breaker_state`。该字段不入库，取值来自本进程内存中的熔断器状态：

- `closed`：正常状态，classic 列表默认不展示额外标签。
- `open`：熔断冷却中，classic 状态列展示“熔断中”。
- `half_open`：恢复探测中，classic 状态列展示“探测中”。
- `disabled`：熔断功能被 `CHANNEL_BREAKER_ENABLED=false` 关闭，classic 状态列展示“熔断关闭”。

`breaker_state` 是进程本地状态，不代表数据库禁用状态，也不应作为跨实例强一致监控来源。多实例部署需要外部可用性监控时，应另建聚合指标或健康检查，不要把该字段写回数据库。

## 流式异常

`StreamScannerHandler` 会在 `RelayInfo.StreamStatus` 中记录流结束原因。timeout、scanner error、panic、ping fail 等异常结束必须通过 `helper.StreamInterruptionError` 转为 `channel:stream_interrupted`，让上层可以记录渠道失败、阻止静默成功，并按已提交响应状态决定是否重试。

如果响应已经向客户端写出，relay 不得切换到其它渠道重试；错误只能通过协议内事件写回。新增或修改流式 handler 时，读取结束后必须检查 `StreamInterruptionError`，不要只处理 JSON 解析错误。

OpenAI Images 流式响应中的 `event: upstream_error` / error payload 也必须视为上游流中断，而不是正常 EOF。已经向客户端写出 partial image 时，应继续把错误事件写回客户端，然后返回 `channel:stream_interrupted`，确保上层记录渠道失败并阻止静默成功。

## 上游请求取消

主 relay 路径里的 API、form 和 task 提交请求必须继承 `gin.Context.Request.Context()`。客户端断开、网关取消或服务端请求超时时，上游 HTTP 请求应一起取消，避免后台继续占用连接、渠道额度和上游排队资源。

不要在主 relay 请求构造中退回 `http.NewRequest` + `context.Background()`。若某个特殊 adaptor 必须自行构造 HTTP 请求，也应优先使用 `http.NewRequestWithContext(c.Request.Context(), ...)`，并在 `channel.DoRequest` 兜底入口保留 context 继承保护。
