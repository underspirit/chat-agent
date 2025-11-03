# ChatAgent 架构设计文档

> 版本：v1.0
> 作者：—
> 日期：2025-11-03（Asia/Tokyo）

---

## 1. 背景与目标

根据给定的 `proto3` 协议：

```proto
syntax = "proto3";

package chat;

service ChatAgentService {
  rpc Chat (PlayerChatRequest) returns (stream PlayerChatResponse);
}

message PlayerChatRequest {
  string player_id = 1;
  string player_nickname = 2;
  string niki_id = 3;
  string niki_name = 4;
  string input_text = 5;
}

message PlayerChatResponse {
  string generated_text = 1;
}
```

目标：设计一个分层、可扩展的 ChatAgent，满足：

1. gRPC 流式对话；2) 会话历史按 `(player_id, niki_id)` 维度持久化并在每次请求前拼接；3) 存储与推理后端可插拔（内存/Redis/文件/数据库；OpenAI/vLLM/本地等）。

非目标：

* 不限定具体编程语言与框架实现；
* 不在本版本定义业务策略（如风控词库）的细节实现。

---

## 2. 总体架构

### 2.1 高层组件

```
+--------------------------------------------------------------+
|                        ChatAgent Service                     |
|                    (gRPC streaming server)                   |
+-----------------------+--------------------+-----------------+
                        |                    |
                        |                    |
                +-------v-------+    +------v-------+
                | Conversation  |    |  LLM Orchestrator |
                |   Manager     |    | (Provider Abstraction)
                +-------+-------+    +------+-------+
                        |                    |
                        |                    |
         +--------------v----+      +-------v----------------+
         | History Store API |      | Prompt Builder &       |
         | (Storage Adapter) |      | Context Window Manager |
         +----+----+----+----+      +-----------+------------+
              |    |    |                       |
          +---+  +--+  +---+                 +--v--------------------+
          |Mem|  |DB|  |Redis|               | Tokenizer/Truncation  |
          +---+  +--+  +----+                +-----------------------+
                                               |
                              +----------------v----------------+
                              | Observability (Logs/Metrics/Traces)
                              +-----------------------------------+
```

### 2.2 分层设计原则

* **接口分离**：对外 `gRPC` 接口稳定；对内通过 **Storage SPI** 和 **LLM SPI** 解耦；
* **无状态服务**：ChatAgent 实例保持无状态，可水平扩展；会话状态保存在可替换的 Store；
* **流式处理**：上游 gRPC 流与下游 LLM 流打通，边生成边回传，支持背压；
* **可观测性**：全链路追踪（trace id 贯穿）、关键指标（QPS、P50/95、TOKENS/s、上下文长度、失败率）。

---

## 3. 接口设计

### 3.1 外部接口（gRPC）

* **服务**：`chat.ChatAgentService/Chat`
* **入参**：`PlayerChatRequest{player_id, player_nickname, niki_id, niki_name, input_text}`
* **出参**：`stream PlayerChatResponse{generated_text}`
* **语义**：单向流，服务端将生成文本分片持续写入；
* **幂等性**：可选地支持 `x-request-id`（通过 metadata）保证客户端重试安全（见 §7.2）。

### 3.2 存储 SPI（Storage Service Provider Interface）

抽象接口（伪代码）：

```ts
interface HistoryStore {
  /**
   * 读取对话历史，按时间升序返回，支持分页 & 限长（token 限制）
   */
  getHistory(conversationKey: ConversationKey, opts: {limitTokens?: number, limitMessages?: number, since?: string, until?: string}): Promise<MessageBatch>;

  /** 追加消息（系统/用户/助手），原子写入 */
  appendMessages(conversationKey: ConversationKey, messages: ChatMessage[]): Promise<void>;

  /** 可选：对长会话做摘要，减少上下文 */
  upsertSummary(conversationKey: ConversationKey, summary: ChatMessage): Promise<void>;

  /** 清理或归档 */
  clear(conversationKey: ConversationKey): Promise<void>;
}

type ConversationKey = { playerId: string; nikiId: string };

// 标准化消息结构，避免与具体 LLM SDK 耦合
interface ChatMessage {
  role: 'system' | 'user' | 'assistant' | 'tool';
  content: string;          // 纯文本或富文本（多模待扩展）
  timestamp: number;        // 服务器时间戳
  meta?: Record<string,any>;// 如 token count, model, latency 等
}
```

> 参考实现：`MemoryStore`、`RedisStore`、`FileStore`、`SqlStore`；通过 DI/配置切换。

### 3.3 LLM SPI（LLM Provider Abstraction）

```ts
interface LlmProvider {
  /**
   * 流式生成：输入标准化消息，返回异步迭代器/stream
   */
  streamChat(
    messages: ChatMessage[],
    params: GenerationParams,
    abortSignal?: AbortSignal
  ): AsyncIterable<PartialChunk>;
}

type GenerationParams = {
  model: string;             // 模型标识
  temperature?: number;
  top_p?: number;
  max_tokens?: number;
  stop?: string[];
  // 供应商私有参数（如 presence_penalty, logit_bias）以扩展字段承载
  vendorOptions?: Record<string, any>;
};

type PartialChunk = {
  delta: string;             // 本次增量文本
  finish_reason?: 'stop' | 'length' | 'error';
  usage?: { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number };
};
```

> 参考实现：`OpenAIProvider`、`VllmProvider`（兼容 OpenAI 风格） 、`LocalProvider`（本地推理服务）。

---

## 4. 核心流程

### 4.1 请求处理顺序

```
Client --> gRPC Chat --> ChatServer
  1. 验证入参（player_id/niki_id/input_text）
  2. 组装 ConversationKey = (player_id, niki_id)
  3. 从 HistoryStore 读取历史（含可选 summary）
  4. PromptBuilder 拼接：System 指令 + Niki Persona + History + 当前用户输入
  5. Context Window Manager 估算 token；必要时摘要/裁剪（优先保留最近轮）
  6. 调用 LLMProvider.streamChat()
  7. 边读边写：将增量 delta 写入 gRPC 流（PlayerChatResponse.generated_text）
  8. 完成后将用户输入与完整助手回复（合并后的文本）追加回 HistoryStore
  9. 写入 metrics / traces
```

### 4.2 时序图（简化）

```
Client        ChatServer     HistoryStore    Prompt/Context     LLMProvider
  |   Chat()     |               |               |                 |
  |------------->|               |               |                 |
  |              |  getHistory   |               |                 |
  |              |-------------> |               |                 |
  |              |   messages    |               |                 |
  |              |<------------- |               |                 |
  |              | buildPrompt & truncate (tokens)                |
  |              |------------------------------->|               |
  |              | prompt                         |               |
  |              |<-------------------------------|               |
  |              | streamChat(prompt)                             |
  |              |----------------------------------------------->|
  |              |                     delta1                     |
  |              |<-----------------------------------------------|
  |   delta1     |                                               |
  |<-------------|                                               |
  |   ...        |                                               |
  |   deltaN     |                                               |
  |<-------------|                                               |
  |              | appendMessages(user + assistant full)         |
  |              |------------->|                                 |
  |              |   ok        |                                 |
  |              |<-------------                                 |
  |   [stream end]
```

### 4.3 Prompt 结构建议

* **System**：平台通用规则（安全/风格/输出格式），可随模型/业务版本化；
* **Niki Persona**：从 `niki_id/niki_name` 载入角色设定（可放 Store 或配置中心）；
* **History**：最近 N 轮（或摘要 + 最近若干完整轮）；
* **Current**：本次用户输入（`player_nickname` 可用于称呼）。

---

## 5. 上下文与摘要策略（Context Window Manager）

* **Token 估算**：根据目标模型的 tokenizer 粗估历史 token；
* **优先级**：`System > Persona > 最近轮对话 > 更早历史摘要`；
* **裁剪策略**：

  1. 若超窗：先保留摘要，再保留最近 K 轮完整对话；
  2. 仍超窗：对用户长文本做片段摘要；
  3. 最终兜底：限制 `max_tokens` 与 `stop`；
* **自动摘要**：当对话累计轮次 > T 时触发，生成“会话摘要消息（role=system/tool）”，并存入 `upsertSummary`。

---

## 6. 存储适配与数据模型

### 6.1 会话键

* `conversation_key = hash(player_id + ":" + niki_id)` （或复合主键）。

### 6.2 典型表/结构（示例）

**SQL 表**：

```
conversations(
  id PK,
  player_id,
  niki_id,
  created_at,
  updated_at
)

messages(
  id PK,
  conversation_id FK,
  role ENUM('system','user','assistant','tool'),
  content TEXT,
  ts TIMESTAMP,
  meta JSON
)

summaries(
  conversation_id PK,
  content TEXT,
  ts TIMESTAMP,
  meta JSON
)
```

**Redis**：

* List：`conv:{player_id}:{niki_id}:msgs` 追加 `ChatMessage(JSON)`；
* Hash：`conv:{player_id}:{niki_id}:summary`。

**文件**：

* 目录：`/data/{player_id}/{niki_id}/history.jsonl`；

**内存**：

* 进程级 `Map<ConversationKey, Message[]>`（仅适合开发/单机）。

### 6.3 存储接口与一致性

* 追加写保证单会话内 **顺序一致**；
* 多实例部署下使用存储的原子操作（Redis `RPUSH`、DB 事务）；
* **写后读**：append 后无需同步读取，除非回显；
* 清理策略：LRU + TTL；归档到冷存储。

---

## 7. 可靠性与弹性

### 7.1 流式稳定性

* **背压与 flush**：设置 gRPC send 阈值与定期 flush；
* **心跳**：长生成时发送小空格/keepalive 以维持连接（可配置）；
* **超时**：整体请求超时（如 60s~120s），LLM 流分片超时（如 10s）。

### 7.2 重试与幂等

* 客户端可带 `x-request-id`；服务端可在 `appendMessages` 前后记录 **完成标志**，避免双写；
* LLM 请求失败：**短重试**（指数退避），超过阈值降级到替补模型；
* 网络中断：允许客户端重连并用同 `x-request-id` 重放（可选特性）。

### 7.3 熔断与降级

* 基于供应商错误率/延迟的 **断路器**；
* 降级策略：切换到更小/本地模型；降低 `max_tokens`；关闭工具调用等高成本特性。

---

## 8. 安全与合规

* **鉴权**：gRPC metadata 携带服务令牌/用户签名；服务端校验租户与调用权限；
* **配额与限流**：按 `player_id`、`niki_id`、来源 IP 维度限流；
* **隐私**：PII 脱敏日志；按租户隔离会话数据；
* **内容安全**：预/后置审核（黑白名单、阈值分类器），对违规输出截断并返回安全提示；
* **审计**：关键操作审计日志（读取历史、清理、模型切换）。

---

## 9. 可观测性

* **日志**：结构化（requestId、playerId、nikiId、model、latency、tokens）；
* **指标**：

  * QPS / 并发 / 错误率；
  * Prompt/Completion tokens、TOKENS/s；
  * 平均上下文长度、摘要命中率；
* **Tracing**：gRPC 入口 span → HistoryStore → PromptBuild → LLMProvider；
* **采样**：对长会话按比例采样保存完整 prompt 以便复现。

---

## 10. 配置与部署

### 10.1 配置（示例 YAML）

```yaml
server:
  port: 8080
  tls: false

historyStore:
  type: redis # memory|file|sql|redis
  redis:
    addr: 127.0.0.1:6379
    db: 0

llm:
  provider: openai # openai|vllm|local
  openai:
    baseUrl: https://api.openai.com/v1
    model: gpt-4o-mini
    timeoutMs: 60000
    maxTokens: 1024
    temperature: 0.7

context:
  hardTokenLimit: 8192
  reserveForCompletion: 1024
  maxTurns: 12
  summarization:
    enabled: true
    triggerTurns: 20
    summaryModel: gpt-4o-mini

observability:
  logLevel: info
  metrics: prometheus
  tracing: otlp

security:
  authRequired: true
  rateLimit:
    perPlayerQps: 2
    burst: 5
```

### 10.2 部署拓扑

* **无状态多副本**：Kubernetes `Deployment`，前置 gRPC Ingress；
* **配置中心**：ConfigMap/Secret 注入；
* **水平扩展**：HPA 基于 QPS/CPU/自定义 TOKENS/s；
* **蓝绿/金丝雀**：按 `niki_id` 或百分比路由不同模型版本；
* **就绪/存活探针**：检查 Store/LLM 依赖可用性。

---

## 11. 参考实现要点（伪代码）

### 11.1 gRPC 处理器

```ts
async function Chat(call: ServerWritableStream<PlayerChatRequest, PlayerChatResponse>) {
  const req = call.request;
  validate(req);
  const key = { playerId: req.player_id, nikiId: req.niki_id };

  // 1) 读历史
  const history = await historyStore.getHistory(key, { limitTokens: ctx.hardLimit - ctx.reserve });

  // 2) 构建 Prompt
  const promptMsgs = promptBuilder.build({
    system: systemRules(),
    persona: loadPersona(req.niki_id, req.niki_name),
    history,
    current: { role: 'user', content: req.input_text }
  });

  // 3) token 控制
  const { finalMsgs, genParams } = contextManager.truncate(promptMsgs, defaultGenParams);

  // 4) 下游流式推理
  let fullText = '';
  for await (const chunk of llm.streamChat(finalMsgs, genParams, call.cancelledSignal)) {
    if (chunk.delta) {
      fullText += chunk.delta;
      call.write({ generated_text: chunk.delta });
    }
  }

  // 5) 落库（用户输入 + 助手完整输出）
  await historyStore.appendMessages(key, [
    { role: 'user', content: req.input_text, timestamp: Date.now() },
    { role: 'assistant', content: fullText, timestamp: Date.now() }
  ]);

  call.end();
}
```

### 11.2 Storage Adapter（示例：Redis）

```ts
class RedisHistoryStore implements HistoryStore {
  constructor(private client: Redis) {}
  async getHistory(key: ConversationKey, opts: {limitTokens?: number, limitMessages?: number}): Promise<MessageBatch> {
    const listKey = `conv:${key.playerId}:${key.nikiId}:msgs`;
    const raw = await this.client.lrange(listKey, 0, -1);
    const messages = raw.map(JSON.parse);
    return sliceByTokens(messages, opts);
  }
  async appendMessages(key: ConversationKey, messages: ChatMessage[]) {
    const listKey = `conv:${key.playerId}:${key.nikiId}:msgs`;
    await this.client.rpush(listKey, messages.map(m => JSON.stringify(m)));
  }
  async upsertSummary(key: ConversationKey, summary: ChatMessage) {
    await this.client.hset(`conv:${key.playerId}:${key.nikiId}:summary`, { content: summary.content, ts: summary.timestamp });
  }
  async clear(key: ConversationKey) { await this.client.del(`conv:${key.playerId}:${key.nikiId}:msgs`); }
}
```

### 11.3 LLM Provider（示例：OpenAI 风格）

```ts
class OpenAIProvider implements LlmProvider {
  constructor(private cfg: OpenAIConfig) {}
  async *streamChat(messages: ChatMessage[], params: GenerationParams): AsyncIterable<PartialChunk> {
    const req = toOpenAI(messages, params);
    const resp = await fetch(`${this.cfg.baseUrl}/chat/completions`, { method: 'POST', headers: {...}, body: JSON.stringify(req) });
    for await (const sse of readServerSentEvents(resp.body)) {
      yield { delta: sse.choices[0]?.delta?.content ?? '' };
    }
  }
}
```

---

## 12. 测试与验收

* **单元测试**：PromptBuilder、ContextManager、Storage/LLM 适配器；
* **集成测试**：起一个内存 Store + Mock LLM 的 gRPC 端到端流；
* **回归测试**：固定输入/输出快照；
* **压测**：并发 100~1k，测 tokens/s、P95、错误率；
* **故障演练**：断网、LLM 500、Redis 慢查询、限流命中。

---

## 13. 迭代路线

* v1：内存/Redis Store，OpenAI Provider；
* v1.1：摘要与上下文自适应；
* v1.2：多供应商路由（按 niki_id 或 AB）；
* v1.3：工具/函数调用与 RAG（可在 LLM SPI 上层增加 Orchestrator）；
* v2.0：多模态（图片/音频）、多轮工具执行、会话迁移/跨端同步。

---

## 14. 风险与权衡

* **上下文成本**：历史越长费率越高 → 摘要/裁剪；
* **一致性**：多副本竞争写入 → 单会话分区/哈希一致性；
* **供应商波动**：加熔断与降级；
* **数据合规**：日志脱敏、数据过期策略；
* **开发效率 vs 灵活性**：通过 SPI 保持灵活，同时提供默认实现与模板工程。

---

## 15. 附：典型错误处理清单

| 场景        | 处理                  | 返回策略                     |
| --------- | ------------------- | ------------------------ |
| 入参缺失      | 400/InvalidArgument | 立即结束流并返回错误状态             |
| Store 不可用 | 快速失败 + 熔断           | 返回提示“系统繁忙，请稍后重试”         |
| LLM 超时    | 短重试 + 降级模型          | 若仍失败，返回部分内容或提示           |
| 超上下文      | 自动摘要/裁剪             | 记录裁剪比率指标                 |
| 客户端断开     | 停止下游推理              | 取消 token 消耗，回滚写入（或标注未完成） |

---

**结论**：本方案通过 Storage/LLM 双抽象、上下文管理与流式处理，构建了可演进的 ChatAgent。后续可在不改变整体架构的前提下替换存储介质与推理后端，并逐步引入工具调用、RAG、多模态等能力。
