package server

import (
	"context"
	"io"
	"testing"

	"chat-agent/internal/contextmgr"
	"chat-agent/internal/conversation"
	"chat-agent/internal/history"
	"chat-agent/internal/llm"
	"chat-agent/internal/logging"
	"chat-agent/internal/prompt"
	chatpb "chat-agent/proto"
)

type fakeStream struct {
	ctx       context.Context
	responses []string
}

func (f *fakeStream) Send(resp *chatpb.PlayerChatResponse) error {
	f.responses = append(f.responses, resp.GeneratedText)
	return nil
}

func (f *fakeStream) SendMsg(m interface{}) error {
	if msg, ok := m.(*chatpb.PlayerChatResponse); ok {
		return f.Send(msg)
	}
	return nil
}

func (f *fakeStream) RecvMsg(interface{}) error {
	return io.EOF
}

func (f *fakeStream) Context() context.Context {
	return f.ctx
}

type mockProvider struct {
	chunks []llm.PartialChunk
}

func (m *mockProvider) StreamChat(ctx context.Context, _ []llm.ChatMessage, _ llm.GenerationParams) (<-chan llm.PartialChunk, error) {
	ch := make(chan llm.PartialChunk)
	go func() {
		defer close(ch)
		for _, chunk := range m.chunks {
			select {
			case <-ctx.Done():
				return
			case ch <- chunk:
			}
		}
	}()
	return ch, nil
}

func TestChatAgentServerStreams(t *testing.T) {
	store, err := history.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	builder, err := prompt.NewSimpleBuilder(prompt.SimpleBuilderConfig{DefaultModel: "test-model", MaxTokens: 128, Temperature: 0.6})
	if err != nil {
		t.Fatalf("failed to create builder: %v", err)
	}

	provider := &mockProvider{chunks: []llm.PartialChunk{{Delta: "hello "}, {Delta: "world"}}}
	logger := logging.NewStdLoggerWithWriter(io.Discard)
	orchestrator := conversation.NewOrchestrator(store, builder, contextmgr.NewNoopManager(), provider, logger)
	svc := NewChatAgentGRPCServer(orchestrator, logger)

	stream := &fakeStream{ctx: context.Background()}
	req := &chatpb.PlayerChatRequest{PlayerId: "p1", PlayerNickname: "Player", NikiId: "n1", NikiName: "Niki", InputText: "hello"}

	if err := svc.Chat(req, stream); err != nil {
		t.Fatalf("chat call failed: %v", err)
	}

	if len(stream.responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(stream.responses))
	}

	historyBatch, err := store.GetHistory(context.Background(), history.ConversationKey{PlayerID: "p1", NikiID: "n1"}, history.ReadOptions{})
	if err != nil {
		t.Fatalf("failed to read history: %v", err)
	}

	if len(historyBatch) != 2 {
		t.Fatalf("expected 2 messages persisted, got %d", len(historyBatch))
	}
	if historyBatch[0].Role != history.RoleUser || historyBatch[0].Content != "hello" {
		t.Fatalf("unexpected user message: %#v", historyBatch[0])
	}
	if historyBatch[1].Role != history.RoleAssistant || historyBatch[1].Content != "hello world" {
		t.Fatalf("unexpected assistant message: %#v", historyBatch[1])
	}
}
