package contextmgr

import (
	"context"

	"chat-agent/internal/llm"
)

// TruncationResult contains the final prompt messages and generation parameters after context window management.
type TruncationResult struct {
	Messages []llm.ChatMessage
	Params   llm.GenerationParams
}

// Manager is responsible for truncating or summarizing the prompt to fit within the target context window.
type Manager interface {
	Truncate(ctx context.Context, messages []llm.ChatMessage, params llm.GenerationParams) (TruncationResult, error)
}
