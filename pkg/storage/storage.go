package storage

import "context"

type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

type ConversationStore interface {
	Load(ctx context.Context, playerID, nikiID string) ([]Message, error)
	Save(ctx context.Context, playerID, nikiID string, messages []Message) error
}
