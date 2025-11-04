package chatagent

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"chat-agent/pkg/llm"
	"chat-agent/pkg/storage"
	chatpb "chat-agent/proto"
)

type Option func(*Service)

type Service struct {
	chatpb.UnimplementedChatAgentServiceServer
	store          storage.ConversationStore
	llmClient      llm.Client
	systemMessages []llm.Message
}

func NewService(store storage.ConversationStore, client llm.Client, opts ...Option) (*Service, error) {
	if store == nil {
		return nil, errors.New("chatagent: store is required")
	}
	if client == nil {
		return nil, errors.New("chatagent: llm client is required")
	}
	svc := &Service{store: store, llmClient: client}
	for _, opt := range opts {
		opt(svc)
	}
	return svc, nil
}

func WithSystemMessage(content string) Option {
	return func(s *Service) {
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			return
		}
		s.systemMessages = append(s.systemMessages, llm.Message{Role: llm.RoleSystem, Content: trimmed})
	}
}

func (s *Service) Chat(req *chatpb.PlayerChatRequest, stream chatpb.ChatAgentService_ChatServer) error {
	if req == nil {
		return errors.New("chatagent: request is nil")
	}
	if stream == nil {
		return errors.New("chatagent: stream is nil")
	}
	ctx := stream.Context()

	history, err := s.store.Load(ctx, req.PlayerId, req.NikiId)
	if err != nil {
		return fmt.Errorf("chatagent: load conversation: %w", err)
	}

	userMessage := storage.Message{
		Role:      llm.RoleUser,
		Content:   strings.TrimSpace(req.InputText),
		Timestamp: time.Now().UTC().Unix(),
	}
	if userMessage.Content == "" {
		return errors.New("chatagent: input text is empty")
	}
	history = append(history, userMessage)

	llmMessages := make([]llm.Message, 0, len(s.systemMessages)+len(history))
	llmMessages = append(llmMessages, s.systemMessages...)
	for _, msg := range history {
		llmMessages = append(llmMessages, llm.Message{Role: msg.Role, Content: msg.Content})
	}

	llmStream, err := s.llmClient.CreateChatCompletionStream(ctx, llmMessages)
	if err != nil {
		return fmt.Errorf("chatagent: create llm stream: %w", err)
	}
	defer llmStream.Close()

	var builder strings.Builder

	for {
		chunk, err := llmStream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("chatagent: receive llm chunk: %w", err)
		}
		if chunk == "" {
			continue
		}
		builder.WriteString(chunk)
		if err := stream.Send(&chatpb.PlayerChatResponse{GeneratedText: chunk}); err != nil {
			return fmt.Errorf("chatagent: send stream chunk: %w", err)
		}
	}

	assistantMessage := storage.Message{
		Role:      llm.RoleAssistant,
		Content:   builder.String(),
		Timestamp: time.Now().UTC().Unix(),
	}
	history = append(history, assistantMessage)

	if err := s.store.Save(ctx, req.PlayerId, req.NikiId, history); err != nil {
		return fmt.Errorf("chatagent: save conversation: %w", err)
	}

	return nil
}
