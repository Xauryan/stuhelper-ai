# Relay 自引用循环保护

StuHelper AI 支持把另一个兼容 OpenAI API 的网关配置为渠道上游。如果渠道的上游地址指向本站，并且渠道 key 使用本站生成的 API Key，普通 relay 或渠道测试请求会再次进入本站的令牌鉴权与渠道选择流程。

当这个本站 API Key 属于 `auto` 分组时，auto 分组选择可能再次选中同一个自引用渠道，从而形成递归 relay。递归链路通常表现为请求长时间无响应，最终由前置网关返回 `524`，日志中出现大量 `status_code=524, bad response status code 524`。

## 保护策略

后端在每次向上游转发时都会附加内部 relay 路径头：

- `StuHelper-AI-Relay-Path`
- `StuHelper-AI-Relay-Signature`

签名使用本实例的 `CRYPTO_SECRET` 生成，只有签名有效时才会被识别为内部 relay 链路。客户端伪造或未签名的同名 header 会被忽略。

当带签名的 relay 请求回到本站时：

- auto 和普通分组选择会排除路径中已经出现过的渠道，优先尝试其他可用渠道；
- 如果没有其他渠道可用，或者选中的渠道已经在当前链路中出现过，请求会以 `508 Loop Detected` 中止；
- 错误码为 `channel:relay_loop`；
- 该错误标记为不重试、不写入普通错误日志，避免递归风暴继续放大。

`channel:relay_loop` 只用于签名 relay 路径确认的自引用循环。普通 failover 中因为本请求已排除失败渠道、渠道熔断或 setup 阶段无可用 key 而没有剩余候选时，不应返回 `channel:relay_loop`，而应保留原始渠道错误或普通“无可用渠道”错误，便于按真实原因排查。

## 配置建议

不要把同一个站点生成的 `auto` API Key 填回同一个站点的渠道 key，除非该 auto 分组中还有其他真实上游渠道可供兜底。更稳定的配置方式是：

- 同站点自引用渠道使用非 `auto` 的专用分组，并确保该分组不会再选回同一个渠道；
- 或者把上游地址配置到另一个独立 StuHelper AI 实例；
- 渠道测试失败时如果看到 `channel:relay_loop`，优先检查该渠道的 base URL 是否指向本站，以及 key 是否为本站 API Key。
