package prompt

import (
	"context"
	"fmt"

	"chat-agent/internal/history"
	"chat-agent/internal/llm"
)

// SimpleBuilderConfig defines how prompts are generated.
type SimpleBuilderConfig struct {
	DefaultModel string
	MaxTokens    int
	Temperature  float32
}

// SimpleBuilder composes a straightforward prompt from the provided context.
type SimpleBuilder struct {
	cfg SimpleBuilderConfig
}

// NewSimpleBuilder creates a SimpleBuilder using the supplied configuration.
func NewSimpleBuilder(cfg SimpleBuilderConfig) (*SimpleBuilder, error) {
	if cfg.DefaultModel == "" {
		return nil, fmt.Errorf("default model must be provided")
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 512
	}
	if cfg.Temperature <= 0 {
		cfg.Temperature = 0.7
	}

	return &SimpleBuilder{cfg: cfg}, nil
}

// Build converts the build context into LLM-ready messages and parameters.
func (b *SimpleBuilder) Build(_ context.Context, input BuildContext) (BuildResult, error) {
	messages := make([]llm.ChatMessage, 0, len(input.SystemMessages)+len(input.History)+2)

	for _, msg := range input.SystemMessages {
		messages = append(messages, llm.ChatMessage{Role: string(msg.Role), Content: msg.Content})
	}

	if input.Persona.Content != "" {
		messages = append(messages, llm.ChatMessage{Role: string(history.RoleSystem), Content: input.Persona.Content})
	}

	for _, msg := range input.History {
		messages = append(messages, llm.ChatMessage{Role: string(msg.Role), Content: msg.Content})
	}

	if input.CurrentInput.Content != "" {
		messages = append(messages, llm.ChatMessage{Role: string(input.CurrentInput.Role), Content: input.CurrentInput.Content})
	}

	params := llm.GenerationParams{
		Model:       b.cfg.DefaultModel,
		MaxTokens:   b.cfg.MaxTokens,
		Temperature: b.cfg.Temperature,
	}

	return BuildResult{Messages: messages, Params: params}, nil
}

var _ Builder = (*SimpleBuilder)(nil)
