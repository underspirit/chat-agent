package conversation

import (
	"context"

	"chat-agent/internal/history"
)

// ChatRequest represents a normalized request for a chat interaction.
type ChatRequest struct {
	PlayerID       string
	PlayerNickname string
	NikiID         string
	NikiName       string
	InputText      string
}

// Stream abstracts the streaming response writer.
type Stream interface {
	SendChunk(ctx context.Context, chunk ChatChunk) error
}

// ChatChunk represents a streaming chunk returned to the caller.
type ChatChunk struct {
	Text string
	Done bool
}

// Manager orchestrates the chat workflow for a single request.
type Manager interface {
	HandleChat(ctx context.Context, req ChatRequest, stream Stream) error
}

// HistoryProvider exposes read/write operations for conversation history.
type HistoryProvider interface {
	FetchHistory(ctx context.Context, key history.ConversationKey) (history.MessageBatch, error)
	PersistMessages(ctx context.Context, key history.ConversationKey, messages history.MessageBatch) error
}
