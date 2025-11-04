package llm

import "context"

// ChatMessage models the message format consumed by downstream LLM providers.
type ChatMessage struct {
	Role    string
	Content string
}

// GenerationParams captures LLM generation knobs.
type GenerationParams struct {
	Model       string
	MaxTokens   int
	Temperature float32
	Stop        []string
}

// PartialChunk represents a streaming delta from the provider.
type PartialChunk struct {
	Delta string
	Done  bool
}

// StreamingProvider abstracts a streaming chat completion provider.
type StreamingProvider interface {
	StreamChat(ctx context.Context, messages []ChatMessage, params GenerationParams) (<-chan PartialChunk, error)
}
