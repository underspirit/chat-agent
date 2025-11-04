package prompt

import (
	"context"

	"chat-agent/internal/llm"
)

// NoopBuilder returns an empty prompt result and leaves the heavy lifting for future iterations.
type NoopBuilder struct{}

// NewNoopBuilder constructs a NoopBuilder.
func NewNoopBuilder() *NoopBuilder {
	return &NoopBuilder{}
}

// Build returns an empty result.
func (b *NoopBuilder) Build(_ context.Context, _ BuildContext) (BuildResult, error) {
	return BuildResult{Messages: []llm.ChatMessage{}, Params: llm.GenerationParams{}}, nil
}

var _ Builder = (*NoopBuilder)(nil)
