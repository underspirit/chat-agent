package contextmgr

import (
	"context"

	"chat-agent/internal/llm"
)

// NoopManager returns the original messages without modification.
type NoopManager struct{}

// NewNoopManager constructs a NoopManager instance.
func NewNoopManager() *NoopManager {
	return &NoopManager{}
}

// Truncate is a passthrough implementation.
func (m *NoopManager) Truncate(_ context.Context, messages []llm.ChatMessage, params llm.GenerationParams) (TruncationResult, error) {
	return TruncationResult{Messages: messages, Params: params}, nil
}

var _ Manager = (*NoopManager)(nil)
