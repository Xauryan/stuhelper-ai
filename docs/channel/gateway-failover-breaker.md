# 渠道 failover 与熔断维护规则

本页记录 relay 自动 failover、流式异常上报和渠道熔断的本地维护规则。

## 默认重试次数

`common.RetryTimes` 的代码默认值保持 `0`。部署方可以通过系统配置启用普通同分组重试；代码层不要把默认值硬改为非零，避免在未显式配置时增加上游请求次数和成本。

`auto` 分组的跨分组重试是独立保护：即使 `RetryTimes=0`，只要令牌开启 `cross_group_retry`，当前真实分组返回全局可重试错误时仍可以推进到后续真实分组。

`auto` 跨真实分组另有一条网关例外：HTTP `502`、`503` 和 Cloudflare `524` 都可以作为切到下一个真实分组的候选。`524` 仍是普通同分组 retry 的 always-skip 状态码；该例外不得绕过显式 `skip-retry`、固定渠道请求或渠道亲和的失败不切换保护。

## 错误分类

relay 错误重试、`auto` 跨分组推进、渠道亲和失败回退、自动禁用判断和熔断计分都必须复用 `service.ClassifyRelayError`。不要在各调用点重新拼一套状态码和错误码判断，否则 `skip-retry`、`always-skip`、渠道侧失败和临时失败会再次出现策略漂移。

分类约定：

- `channel`：明确的渠道侧错误，包含 `types.IsChannelError`、自动禁用状态码和自动禁用关键词。用于强制重试、亲和 de-pin、自动禁用和熔断 fatal 计分。
- `transient`：临时上游错误，例如全局可重试状态码或非法 upstream status。用于普通 retry 和熔断 transient 计分。
- `skip`：显式 `skip-retry`、全局 always-skip 错误码/状态码，以及异常的 2xx 错误。不得触发 retry、亲和回退或熔断惩罚。
- `client`：其余业务/请求错误。不得惩罚渠道。

调用方如果需要处理 `auto` 跨真实分组的网关例外，应先复用 `service.ClassifyRelayError`，再只针对 HTTP `502`、`503`、`524` 做窄口径补充判断；不要把 `524` 移出全局 always-skip 列表，也不要让该例外影响渠道亲和、熔断或自动禁用计分。

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

渠道可用性 telemetry 使用 `CHANNEL_AVAILABILITY_WINDOW_SECONDS` 控制最近统计窗口，默认 `600` 秒。该配置同样必须在 `.env` 加载和 `common.InitEnv()` 后通过 `service.InitChannelAvailabilityConfig()` 初始化。

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

同一接口还会附加只读字段 `availability`，用于展示最近窗口内的可用性统计：

- `window_seconds`：统计窗口长度。
- `success` / `channel_failures` / `transient_failures` / `ignored`：最近窗口内的样本计数。
- `success_rate`：成功率，仅按成功、渠道失败和临时失败计算分母。
- `last_success_at` / `last_failure_at`：最近一次成功或失败的时间戳。
- `last_error` / `last_class`：最近一次记录到的错误摘要和分类。

该字段同样是进程本地 telemetry，不入库，不作为跨实例一致性依据。

classic 渠道列表只展示该 telemetry 的摘要标签和 tooltip。它用于人工排查“最近是否稳定”，不参与渠道选择、熔断决策、自动禁用或计费。

## 渠道可用性监控

经典后台新增独立的可用性监控聚合接口 `GET /api/channel/monitor/summary`，用于给 classic 的模型广场和仪表盘展示持久化可用率。该接口读取既有 `logs` 表，不新增监控表，也不参与渠道选择、熔断、自动禁用或计费。

查询参数：

- `window_seconds`：统计窗口，默认 `7d`。
- `source`：`all` / `log` / `probe`，用于切换真实请求、主动探测或综合视图。
- `channel_id`、`model_name`、`group`：按渠道、模型或分组过滤。
- `error_limit`：最近错误条数，默认 `20`，最大 `100`。

统计口径：

- `success`：成功日志。
- `channel_failures`：明确的渠道侧失败，例如 401/403、key revoked、quota、无可用 key、`channel:*` 错误等。
- `transient_failures`：429、5xx、timeout、连接中断、临时 upstream 异常等。
- `ignored`：客户端 400/422、invalid request、prompt blocked 或本地不可测试等不会惩罚渠道的错误。
- `sla`：仅按 `success + channel_failures + transient_failures` 计算分母，`ignored` 不计入分母。

主动探测通过通道测试写入的日志识别，优先使用 `Other.monitor_source = probe` 或 `Other.probe = true`，并兼容历史的“模型测试”日志。成功探测会计入 `probe` 视图；失败探测会写入 `probe_status=failed`；本地不可测试、参数构造失败或流程未真正发起上游请求的情况会写入 `probe_status=local_error`，并从 SLA 分母剔除。

classic 监控面板显示三块指标卡：

- `真实请求 SLA`：只看真实 relay 日志。
- `主动探测 SLA`：只看主动探测日志。
- `综合 SLA`：真实请求与主动探测合并后的整体视图。

最近错误列表会独立展示来源、时间、渠道、模型、状态码、错误码、错误类型、请求路径和报错内容，便于区分真实流量、主动探测、渠道错误、本地错误和被排除出 SLA 分母的错误。

## 流式异常

`StreamScannerHandler` 会在 `RelayInfo.StreamStatus` 中记录流结束原因。timeout、scanner error、panic、ping fail 等异常结束必须通过 `helper.StreamInterruptionError` 转为 `channel:stream_interrupted`，让上层可以记录渠道失败、阻止静默成功，并按已提交响应状态决定是否重试。

如果响应已经向客户端写出，relay 不得切换到其它渠道重试；错误只能通过协议内事件写回。新增或修改流式 handler 时，读取结束后必须检查 `StreamInterruptionError`，不要只处理 JSON 解析错误。

OpenAI Images 流式响应中的 `event: upstream_error` / error payload 也必须视为上游流中断，而不是正常 EOF。已经向客户端写出 partial image 时，应继续把错误事件写回客户端，然后返回 `channel:stream_interrupted`，确保上层记录渠道失败并阻止静默成功。

普通 OpenAI SSE 响应即使 HTTP 状态码是 `200`，只要数据帧中出现 `{"error":...}`、`{"type":"error"}` 或 `{"type":"upstream_error"}`，也必须返回真实上游错误，不得等到 EOF 后按成功结算。

`StreamStatus` 的结束原因和软错误计数可能被 scanner、handler、ping goroutine 和日志生成同时访问。新增代码应通过 `SetEndReason()`、`End()`、`EndReasonIs()`、`Snapshot()`、`HasErrors()` 和 `TotalErrorCount()` 访问，不要直接读写 `EndReason`、`EndError`、`Errors` 或 `ErrorCount`。日志生成和响应后处理应优先使用 `Snapshot()`，避免竞态。

`StreamScannerHandler` 的停止信号使用 close-once 广播语义。新增 goroutine 需要监听同一个停止信号或调用同一个 stop 函数，不能恢复成向 stop channel 发送 bool 的模式，否则 close/send 会重新引入竞态。

流式 scanner 单行缓冲默认 `128MB`，可通过 `STREAM_SCANNER_MAX_BUFFER_MB` 覆盖。同步上游时继续保持该默认值，避免大 JSON/SSE 帧回退到过小 scanner buffer。

## Relay 中间过滤器

本地新增了一个可配置的 relay filter worker，用来在请求上游之前修改请求体，或在上游响应返回后修改响应体。它覆盖的是 classic 和 relay 的公共转发链路，不是单独某个渠道的特例。

环境变量：

- `RELAY_FILTER_WORKER_ENABLED`：总开关，默认开启。
- `RELAY_FILTER_WORKER_CONFIG`：JSON 配置，支持 `request`、`response` 和 `stream_response` 三类规则。
- `RELAY_FILTER_WORKER_MAX_RESPONSE_MB`：响应过滤时最多读入多少 MB，默认 `16`，超过后不再尝试全量缓冲过滤。

规则格式沿用现有 param override 的 `operations` 引擎，支持单条对象或对象数组，支持 `name`、`enabled`、`conditions`、`logic`、`operations`、`override`。请求阶段在 channel 转换和 param override 之后执行；响应阶段在 `runResponsePipeline` 中、adaptor 解析响应体之前执行；SSE 的 `data:` JSON 帧也会在被 adaptor 读取前进行同样的规则处理。

三个阶段分别是：

- `request`：修改发往上游的 JSON 请求体。
- `response`：修改普通 JSON 响应体。
- `stream_response`：修改 SSE `data:` JSON 帧内容。

过滤器命中时会把审计轨迹写入 `RelayInfo.RelayFilterAudit`，并在日志的 `Other.relay_filter_worker` 里回显命中的规则名，方便排查哪些规则真的生效了。请求体过滤后的 `UpstreamRequestBodySize` 会按过滤后的 payload 重新记录，避免日志和实际出站体积不一致。

示例：

```json
{
  "enabled": true,
  "request": [
    {
      "name": "trim-model",
      "operations": [
        { "path": "model", "mode": "trim_prefix", "value": "openai/" }
      ]
    }
  ],
  "response": [
    {
      "name": "mask-secret",
      "operations": [
        { "path": "data.secret", "mode": "set", "value": "redacted" }
      ]
    }
  ],
  "stream_response": [
    {
      "name": "mask-stream",
      "operations": [
        { "path": "delta.secret", "mode": "set", "value": "redacted" }
      ]
    }
  ]
}
```

如果过滤规则本身报错，request 阶段会按参数覆写非法错误返回，response 阶段会按坏响应体错误返回，避免把过滤失败伪装成成功响应。

## 响应流水线

主 HTTP relay 响应路径应通过 `runResponsePipeline` 做统一收口：识别 `text/event-stream` 并更新 `RelayInfo.IsStream`、处理非 200 上游响应、应用渠道状态码映射、以及在 adaptor `DoResponse` 返回错误时再次应用状态码映射。

Replicate 图片接口的 `201 Created` 是兼容例外，只有 image helper 以 `allowCreated` 显式允许时才可视为成功。其它 provider 不应绕过统一非 200 处理。

WebSocket relay 不是普通 HTTP response-body 语义，暂不纳入该 helper；如未来改造，需要单独设计协议内错误和关闭码映射。

## 异步任务 relay

任务提交 relay 也必须把每次渠道尝试上报给 `service.ReportRelayResult`：

- 提交成功：记录成功样本，用于 breaker 半开探测恢复和 availability 成功率。
- 上游/渠道错误：转换为统一 `StuHelperAIError` 后复用 `ClassifyRelayError`，再记录 breaker/availability，并把失败渠道加入当前 retry exclude set。
- 本地请求错误：不记录渠道失败，不污染 breaker/availability。

这保证文本、图片、音频和异步任务的渠道健康统计使用同一套分类和熔断规则。

## 上游请求取消

主 relay 路径里的 API、form 和 task 提交请求必须继承 `gin.Context.Request.Context()`。客户端断开、网关取消或服务端请求超时时，上游 HTTP 请求应一起取消，避免后台继续占用连接、渠道额度和上游排队资源。

不要在主 relay 请求构造中退回 `http.NewRequest` + `context.Background()`。若某个特殊 adaptor 必须自行构造 HTTP 请求，也应优先使用 `http.NewRequestWithContext(c.Request.Context(), ...)`，并在 `channel.DoRequest` 兜底入口保留 context 继承保护。

`RELAY_TIMEOUT` 采用分层语义：

- 非流式 relay：继续作为完整上游请求超时。
- 流式 relay：不再让 `http.Client.Timeout` 覆盖整个响应体读取，只用 transport `ResponseHeaderTimeout` 限制连接和响应头等待；响应体长流空闲由 `STREAMING_TIMEOUT` 处理。
- AWS Bedrock 非流式/Nova 调用继续叠加 `RELAY_TIMEOUT`；AWS 流式调用继承请求取消，响应体空闲交给 stream scanner。

Midjourney 透传请求也必须继承客户端请求 context，并在原有 Midjourney timeout 上叠加取消保护；GET 请求不得构造 `null` body。
