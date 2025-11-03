package llm

import (
	"context"
)

// NoopProvider is a placeholder StreamingProvider that returns an empty stream.
type NoopProvider struct{}

// NewNoopProvider creates a new NoopProvider instance.
func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

// StreamChat returns an already-closed channel to satisfy the interface.
func (p *NoopProvider) StreamChat(_ context.Context, _ []ChatMessage, _ GenerationParams) (<-chan PartialChunk, error) {
	ch := make(chan PartialChunk)
	close(ch)
	return ch, nil
}

var _ StreamingProvider = (*NoopProvider)(nil)
