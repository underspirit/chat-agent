package chatagent

import (
	"context"
	"io"
	"testing"

	"chat-agent/pkg/llm"
	"chat-agent/pkg/storage"
	chatpb "chat-agent/proto"
)

type stubLLMClient struct {
	chunks    []string
	received  []llm.Message
	streamErr error
}

func (s *stubLLMClient) CreateChatCompletionStream(ctx context.Context, messages []llm.Message) (llm.Stream, error) {
	s.received = append([]llm.Message(nil), messages...)
	if s.streamErr != nil {
		return nil, s.streamErr
	}
	return &stubStream{chunks: append([]string(nil), s.chunks...)}, nil
}

type stubStream struct {
	chunks []string
	index  int
}

func (s *stubStream) Recv() (string, error) {
	if s.index >= len(s.chunks) {
		return "", io.EOF
	}
	chunk := s.chunks[s.index]
	s.index++
	return chunk, nil
}

func (s *stubStream) Close() error { return nil }

type memoryStream struct {
	responses []string
	ctx       context.Context
}

func newMemoryStream() *memoryStream {
	return &memoryStream{ctx: context.Background()}
}

func (m *memoryStream) Send(resp *chatpb.PlayerChatResponse) error {
	m.responses = append(m.responses, resp.GeneratedText)
	return nil
}

func (m *memoryStream) Context() context.Context { return m.ctx }

func TestServiceChat(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatalf("create file store: %v", err)
	}

	llmClient := &stubLLMClient{chunks: []string{"hello ", "world"}}
	svc, err := NewService(store, llmClient)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	stream := newMemoryStream()
	req := &chatpb.PlayerChatRequest{
		PlayerId:       "player-1",
		PlayerNickname: "Tester",
		NikiId:         "niki-1",
		NikiName:       "Niki",
		InputText:      "Hi there!",
	}

	if err := svc.Chat(req, stream); err != nil {
		t.Fatalf("chat: %v", err)
	}

	expectedChunks := []string{"hello ", "world"}
	if len(stream.responses) != len(expectedChunks) {
		t.Fatalf("expected %d responses, got %d", len(expectedChunks), len(stream.responses))
	}
	for i, chunk := range expectedChunks {
		if stream.responses[i] != chunk {
			t.Fatalf("chunk %d: expected %q, got %q", i, chunk, stream.responses[i])
		}
	}

	history, err := store.Load(context.Background(), "player-1", "niki-1")
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages in history, got %d", len(history))
	}
	if history[0].Role != llm.RoleUser {
		t.Fatalf("first message role: expected %q, got %q", llm.RoleUser, history[0].Role)
	}
	if history[0].Content != "Hi there!" {
		t.Fatalf("first message content: expected %q, got %q", "Hi there!", history[0].Content)
	}
	if history[1].Role != llm.RoleAssistant {
		t.Fatalf("second message role: expected %q, got %q", llm.RoleAssistant, history[1].Role)
	}
	if history[1].Content != "hello world" {
		t.Fatalf("assistant content: expected %q, got %q", "hello world", history[1].Content)
	}

	if len(llmClient.received) == 0 {
		t.Fatalf("llm did not receive messages")
	}
	// The last message sent to the LLM should be the assistant reply appended to the accumulated history.
	if llmClient.received[len(llmClient.received)-1].Role != llm.RoleUser {
		t.Fatalf("expected last message before response to be user role, got %q", llmClient.received[len(llmClient.received)-1].Role)
	}
}
