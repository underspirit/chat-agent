package prompt

import (
	"context"

	"chat-agent/internal/history"
	"chat-agent/internal/llm"
)

// BuildContext captures the information needed to construct an LLM-ready prompt.
type BuildContext struct {
	SystemMessages []history.Message
	Persona        history.Message
	History        history.MessageBatch
	CurrentInput   history.Message
}

// BuildResult contains the prompt messages and generation parameters ready for the LLM.
type BuildResult struct {
	Messages []llm.ChatMessage
	Params   llm.GenerationParams
}

// Builder assembles messages into the format required by the LLM provider.
type Builder interface {
	Build(ctx context.Context, input BuildContext) (BuildResult, error)
}
