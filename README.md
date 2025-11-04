# Chat Agent

该项目提供了一个简单的分层 Chat Agent 实现，遵循 `proto/chat.proto` 中的接口约定，包含以下层次：

- **gRPC 接口层**：`internal/chatagent.Service` 实现了 `Chat` 接口逻辑，可直接复用到 gRPC Server 中。
- **历史存储层**：通过 `pkg/storage` 定义统一接口，当前实现了基于 JSON 文件的版本。
- **LLM 推理层**：`pkg/llm` 定义统一的推理接口，`pkg/llm/openai` 提供了基于 OpenAI Chat Completions API 的流式实现。

## 目录结构

```
proto/                 # gRPC 协议及对应的 Go 类型
internal/chatagent/    # Chat 服务实现与测试
pkg/storage/           # 对话历史存储接口与文件实现
pkg/llm/               # LLM 客户端接口及 OpenAI 实现
```

## 运行测试

```bash
go test ./...
```

测试用例会通过内置的文件存储和假的 LLM 客户端跑通一次完整的流式对话流程。

## 运行示例

1. 准备一个存储目录，例如 `./data`。
2. 设置 OpenAI API Key：

   ```bash
   export OPENAI_API_KEY=sk-xxxx
   ```

3. 在你的 `main.go` 中组合各个层：

   ```go
   package main

   import (
       "log"
       "net"
       "os"

       "google.golang.org/grpc"

       "chat-agent/internal/chatagent"
       "chat-agent/pkg/llm/openai"
       "chat-agent/pkg/storage"
       chatpb "chat-agent/proto"
   )

   func main() {
       store, err := storage.NewFileStore("./data")
       if err != nil {
           log.Fatal(err)
       }

       client, err := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
       if err != nil {
           log.Fatal(err)
       }

       svc, err := chatagent.NewService(store, client)
       if err != nil {
           log.Fatal(err)
       }

       // 伪代码：此处将 svc 注册到 gRPC Server 中，然后监听端口。
       grpcServer := grpc.NewServer()
       // TODO: 使用 protoc 生成的注册代码（chatpb.RegisterChatAgentServiceServer）将 svc 注册到 gRPC Server。
       // 目前仓库中提供的是最小可用的接口定义，如需完整 gRPC 集成请使用 protoc 重新生成 pb 代码。
       lis, err := net.Listen("tcp", ":8080")
       if err != nil {
           log.Fatal(err)
       }
       log.Println("chat agent serving on :8080")
       if err := grpcServer.Serve(lis); err != nil {
           log.Fatal(err)
       }
   }
   ```

   完整的 gRPC Server 启动代码需要在本地拉取 `google.golang.org/grpc` 依赖后补充。此示例展示了如何组合存储与 LLM 层。

## 注意事项

- `pkg/llm/openai` 仅在设置了有效的 OpenAI API Key 并允许外网访问时才能正常工作。
- 文件存储会在指定目录下以 `<player_id>__<niki_id>.json` 的方式保存对话历史，后续可根据接口替换为 Redis 或数据库实现。
