package llm

import "context"

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

type Message struct {
	Role    string
	Content string
}

type Stream interface {
	Recv() (string, error)
	Close() error
}

type Client interface {
	CreateChatCompletionStream(ctx context.Context, messages []Message) (Stream, error)
}
