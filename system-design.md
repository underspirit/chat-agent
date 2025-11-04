syntax = "proto3";

package chat;

// 服务定义
service ChatAgentService {
  // 单向流：客户端发送请求 → 服务端返回流式文本回复
  rpc Chat (PlayerChatRequest) returns (stream PlayerChatResponse);
}

// 请求消息：包含玩家和 Niki 的信息及输入文本
message PlayerChatRequest {
  string player_id = 1;      // 玩家ID（必填）
  string player_nickname = 2; // 玩家昵称（必填）
  string niki_id = 3;        // Niki的ID（必填）
  string niki_name = 4;      // Niki名称（必填）
  string input_text = 5;     // 玩家输入文本（必填）
}

// 流式响应消息：服务端逐步返回生成的文本块
message PlayerChatResponse {
  string generated_text = 1; // 生成的文本片段
}

接口说明: 外部服务每次调用Chat在接口指定了player_id 与niki_id, 以及玩家的输入(input_text), Chat接口返回的PlayerChatResponse.generated_text 为niki给玩家的对话回复

按照上面的ChatAgent的grpc协议, 帮我写一个Chat-agent模块, 提供Chat 接口用于对话
要求:
0 . 使用Go语言开发
1. 对话的Chat接口符合proto协议的定义, 进行与大模型的流式对话
2. ChatAgent需要保存player_id 与niki_id的对话历史, 接收到新的Chat请求是, 需要先获取对话历史, 进行拼接, 在去请求大模型获取对话返回
3. 我是Go语言新手, 所以不需要太复杂的设计, 我的想法是ChatAgent需要分层设计, 后期如果修改可以灵活改动, 而不需要修改架构;
4. 对话历史的存储层, 真实的存储介质可以使内存/redis/文件/数据库, 接口统一, 当前先实现基于文件的版本, 后续可以灵活切换; 
5. LLM的推理层, 可以接不同的LLM推理服务, 比如本地推理服务/Openai api/vllm等, 当前先使用最简单的Openai chat api

注意: 不要太复杂的设计, 架构清晰明了即可, 我是Go新手; 并给出运行步骤
