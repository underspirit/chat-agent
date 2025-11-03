package conversation

import (
	"context"
	"io"
	"testing"

	"chat-agent/internal/contextmgr"
	"chat-agent/internal/history"
	"chat-agent/internal/llm"
	"chat-agent/internal/logging"
	"chat-agent/internal/prompt"
)

type stubStream struct {
	chunks []ChatChunk
}

func (s *stubStream) SendChunk(_ context.Context, chunk ChatChunk) error {
	s.chunks = append(s.chunks, chunk)
	return nil
}

type stubLLMProvider struct {
	chunks []llm.PartialChunk
	calls  int
}

func (p *stubLLMProvider) StreamChat(_ context.Context, messages []llm.ChatMessage, params llm.GenerationParams) (<-chan llm.PartialChunk, error) {
	p.calls++
	if params.Model == "" {
		return nil, testingErr{"missing model"}
	}
	ch := make(chan llm.PartialChunk, len(p.chunks))
	for _, chunk := range p.chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

type testingErr struct{ msg string }

func (e testingErr) Error() string { return e.msg }

func TestOrchestratorHandleChatStreamsAndPersists(t *testing.T) {
	store, err := history.NewFileStore(history.FileStoreConfig{BaseDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	builder, err := prompt.NewSimpleBuilder(prompt.SimpleConfig{Model: "test-model"})
	if err != nil {
		t.Fatalf("NewSimpleBuilder: %v", err)
	}

	provider := &stubLLMProvider{chunks: []llm.PartialChunk{{Delta: "Hello"}, {Delta: " world"}, {Done: true}}}
	logger := logging.NewStdLoggerWithWriter(io.Discard)
	orchestrator := NewOrchestrator(OrchestratorConfig{SystemPrompts: []string{"Be kind."}}, store, builder, contextmgr.NewNoopManager(), provider, logger)

	req := ChatRequest{PlayerID: "player", PlayerNickname: "Nick", NikiID: "niki", NikiName: "Niki", InputText: "How are you?"}
	stream := &stubStream{}

	if err := orchestrator.HandleChat(context.Background(), req, stream); err != nil {
		t.Fatalf("HandleChat: %v", err)
	}

	if len(stream.chunks) != 2 {
		t.Fatalf("expected 2 streamed chunks, got %d", len(stream.chunks))
	}
	if stream.chunks[0].Text != "Hello" || stream.chunks[1].Text != " world" {
		t.Fatalf("unexpected chunk contents: %+v", stream.chunks)
	}

	stored, err := store.GetHistory(context.Background(), history.ConversationKey{PlayerID: "player", NikiID: "niki"}, history.ReadOptions{})
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected 2 messages in history, got %d", len(stored))
	}
	if stored[1].Role != history.RoleAssistant || stored[1].Content != "Hello world" {
		t.Fatalf("unexpected stored assistant message: %+v", stored[1])
	}
}

func TestOrchestratorValidate(t *testing.T) {
	store, _ := history.NewFileStore(history.FileStoreConfig{BaseDir: t.TempDir()})
	builder, _ := prompt.NewSimpleBuilder(prompt.SimpleConfig{Model: "test-model"})
	logger := logging.NewStdLoggerWithWriter(io.Discard)
	orchestrator := NewOrchestrator(OrchestratorConfig{}, store, builder, contextmgr.NewNoopManager(), &stubLLMProvider{}, logger)

	err := orchestrator.HandleChat(context.Background(), ChatRequest{}, &stubStream{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}
