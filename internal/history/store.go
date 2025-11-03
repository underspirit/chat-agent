package history

import "context"

// ConversationKey uniquely identifies a conversation between a player and a Niki persona.
type ConversationKey struct {
	PlayerID string
	NikiID   string
}

// MessageRole enumerates supported message roles.
type MessageRole string

const (
	// RoleSystem denotes a system-level message.
	RoleSystem MessageRole = "system"
	// RoleUser denotes a user/player message.
	RoleUser MessageRole = "user"
	// RoleAssistant denotes the assistant generated reply.
	RoleAssistant MessageRole = "assistant"
	// RoleTool denotes tool invocation results.
	RoleTool MessageRole = "tool"
)

// Message models a single conversation turn persisted in the history store.
type Message struct {
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	Timestamp int64       `json:"timestamp"`
}

// MessageBatch is a simple slice alias that allows future extension.
type MessageBatch []Message

// ReadOptions configures GetHistory calls.
type ReadOptions struct {
	LimitTokens   int
	LimitMessages int
}

// Store defines the abstract interface for conversation history persistence.
type Store interface {
	GetHistory(ctx context.Context, key ConversationKey, opts ReadOptions) (MessageBatch, error)
	AppendMessages(ctx context.Context, key ConversationKey, messages MessageBatch) error
	UpsertSummary(ctx context.Context, key ConversationKey, summary Message) error
	Clear(ctx context.Context, key ConversationKey) error
}
