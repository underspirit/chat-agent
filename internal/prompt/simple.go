package prompt

import (
	"context"
	"fmt"
	"strings"

	"chat-agent/internal/history"
	"chat-agent/internal/llm"
)

// SimpleConfig configures the SimpleBuilder.
type SimpleConfig struct {
	DefaultSystem []string
	Model         string
	Temperature   float32
	MaxTokens     int
}

// SimpleBuilder assembles prompts using a straightforward concatenation strategy.
type SimpleBuilder struct {
	cfg SimpleConfig
}

// NewSimpleBuilder returns a SimpleBuilder instance.
func NewSimpleBuilder(cfg SimpleConfig) (*SimpleBuilder, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model must be provided")
	}
	return &SimpleBuilder{cfg: cfg}, nil
}

// Build converts the provided context into OpenAI-style chat messages.
func (b *SimpleBuilder) Build(_ context.Context, input BuildContext) (BuildResult, error) {
	messages := make([]llm.ChatMessage, 0, len(input.SystemMessages)+len(input.History)+2)

	for _, sys := range b.cfg.DefaultSystem {
		if trimmed := strings.TrimSpace(sys); trimmed != "" {
			messages = append(messages, llm.ChatMessage{Role: string(history.RoleSystem), Content: trimmed})
		}
	}

	for _, sys := range input.SystemMessages {
		if trimmed := strings.TrimSpace(sys.Content); trimmed != "" {
			messages = append(messages, llm.ChatMessage{Role: string(history.RoleSystem), Content: trimmed})
		}
	}

	if trimmed := strings.TrimSpace(input.Persona.Content); trimmed != "" {
		messages = append(messages, llm.ChatMessage{Role: string(history.RoleSystem), Content: trimmed})
	}

	for _, msg := range input.History {
		messages = append(messages, llm.ChatMessage{Role: string(msg.Role), Content: msg.Content})
	}

	messages = append(messages, llm.ChatMessage{Role: string(input.CurrentInput.Role), Content: input.CurrentInput.Content})

	params := llm.GenerationParams{
		Model:       b.cfg.Model,
		MaxTokens:   b.cfg.MaxTokens,
		Temperature: b.cfg.Temperature,
	}
	return BuildResult{Messages: messages, Params: params}, nil
}

var _ Builder = (*SimpleBuilder)(nil)
