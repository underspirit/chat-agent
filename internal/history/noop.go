package history

import "context"

// NoopStore is a placeholder Store implementation used during early scaffolding.
type NoopStore struct{}

// NewNoopStore constructs a NoopStore instance.
func NewNoopStore() *NoopStore {
	return &NoopStore{}
}

// GetHistory returns an empty message batch.
func (s *NoopStore) GetHistory(_ context.Context, _ ConversationKey, _ ReadOptions) (MessageBatch, error) {
	return MessageBatch{}, nil
}

// AppendMessages discards the provided messages.
func (s *NoopStore) AppendMessages(_ context.Context, _ ConversationKey, _ MessageBatch) error {
	return nil
}

// UpsertSummary is a no-op.
func (s *NoopStore) UpsertSummary(_ context.Context, _ ConversationKey, _ Message) error {
	return nil
}

// Clear is a no-op.
func (s *NoopStore) Clear(_ context.Context, _ ConversationKey) error {
	return nil
}

var _ Store = (*NoopStore)(nil)
