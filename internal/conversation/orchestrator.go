package conversation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"chat-agent/internal/contextmgr"
	"chat-agent/internal/history"
	"chat-agent/internal/llm"
	"chat-agent/internal/logging"
	"chat-agent/internal/prompt"
)

// Orchestrator wires together the Storage, Prompt Builder, Context Manager, and LLM provider.
type Orchestrator struct {
	historyStore  history.Store
	promptBuilder prompt.Builder
	ctxManager    contextmgr.Manager
	llmProvider   llm.StreamingProvider
	logger        logging.Logger
	systemPrompts []history.Message
}

// OrchestratorConfig configures request level defaults.
type OrchestratorConfig struct {
	SystemPrompts []string
}

// NewOrchestrator constructs a new Orchestrator instance with the required dependencies.
func NewOrchestrator(cfg OrchestratorConfig, store history.Store, builder prompt.Builder, manager contextmgr.Manager, provider llm.StreamingProvider, logger logging.Logger) *Orchestrator {
	systemMessages := make([]history.Message, 0, len(cfg.SystemPrompts))
	for _, prompt := range cfg.SystemPrompts {
		trimmed := strings.TrimSpace(prompt)
		if trimmed == "" {
			continue
		}
		systemMessages = append(systemMessages, history.Message{Role: history.RoleSystem, Content: trimmed})
	}

	return &Orchestrator{
		historyStore:  store,
		promptBuilder: builder,
		ctxManager:    manager,
		llmProvider:   provider,
		logger:        logger,
		systemPrompts: systemMessages,
	}
}

// HandleChat executes the full chat workflow, streaming LLM output back to the caller and persisting history.
func (o *Orchestrator) HandleChat(ctx context.Context, req ChatRequest, stream Stream) error {
	if err := o.validate(req); err != nil {
		return err
	}

	log := o.logger.With(logging.Field{Key: "player_id", Value: req.PlayerID}, logging.Field{Key: "niki_id", Value: req.NikiID})
	log.Info("received chat request")

	key := history.ConversationKey{PlayerID: req.PlayerID, NikiID: req.NikiID}
	prior, err := o.historyStore.GetHistory(ctx, key, history.ReadOptions{})
	if err != nil {
		return fmt.Errorf("fetch history: %w", err)
	}

	buildCtx := prompt.BuildContext{
		SystemMessages: append([]history.Message{}, o.systemPrompts...),
		Persona: history.Message{
			Role:    history.RoleSystem,
			Content: fmt.Sprintf("You are %s (id=%s). Maintain persona continuity while speaking with %s.", req.NikiName, req.NikiID, req.PlayerNickname),
		},
		History: prior,
		CurrentInput: history.Message{
			Role:      history.RoleUser,
			Content:   req.InputText,
			Timestamp: time.Now().Unix(),
		},
	}

	promptResult, err := o.promptBuilder.Build(ctx, buildCtx)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	managed, err := o.ctxManager.Truncate(ctx, promptResult.Messages, promptResult.Params)
	if err != nil {
		return fmt.Errorf("truncate prompt: %w", err)
	}

	chunks, err := o.llmProvider.StreamChat(ctx, managed.Messages, managed.Params)
	if err != nil {
		return fmt.Errorf("llm stream: %w", err)
	}

	var responseBuilder strings.Builder
	for chunk := range chunks {
		if chunk.Err != nil {
			return fmt.Errorf("llm chunk error: %w", chunk.Err)
		}
		if chunk.Delta != "" {
			responseBuilder.WriteString(chunk.Delta)
			if err := stream.SendChunk(ctx, ChatChunk{Text: chunk.Delta}); err != nil {
				return fmt.Errorf("send chunk: %w", err)
			}
		}
		if chunk.Done {
			break
		}
	}

	responseText := responseBuilder.String()
	appendBatch := history.MessageBatch{
		{Role: history.RoleUser, Content: req.InputText, Timestamp: time.Now().Unix()},
	}
	if strings.TrimSpace(responseText) != "" {
		appendBatch = append(appendBatch, history.Message{Role: history.RoleAssistant, Content: responseText, Timestamp: time.Now().Unix()})
	}

	if err := o.historyStore.AppendMessages(ctx, key, appendBatch); err != nil {
		return fmt.Errorf("append history: %w", err)
	}

	return nil
}

func (o *Orchestrator) validate(req ChatRequest) error {
	switch {
	case req.PlayerID == "":
		return errors.New("player_id must be provided")
	case req.PlayerNickname == "":
		return errors.New("player_nickname must be provided")
	case req.NikiID == "":
		return errors.New("niki_id must be provided")
	case req.NikiName == "":
		return errors.New("niki_name must be provided")
	case req.InputText == "":
		return errors.New("input_text must be provided")
	}
	return nil
}

// DefaultHistoryProvider adapts the history.Store to the HistoryProvider interface.
type DefaultHistoryProvider struct {
	store history.Store
}

// NewDefaultHistoryProvider creates a DefaultHistoryProvider wrapper.
func NewDefaultHistoryProvider(store history.Store) *DefaultHistoryProvider {
	return &DefaultHistoryProvider{store: store}
}

// FetchHistory retrieves the conversation history for the given key.
func (p *DefaultHistoryProvider) FetchHistory(ctx context.Context, key history.ConversationKey) (history.MessageBatch, error) {
	return p.store.GetHistory(ctx, key, history.ReadOptions{})
}

// PersistMessages appends the messages to storage.
func (p *DefaultHistoryProvider) PersistMessages(ctx context.Context, key history.ConversationKey, messages history.MessageBatch) error {
	return p.store.AppendMessages(ctx, key, messages)
}

var _ Manager = (*Orchestrator)(nil)
var _ HistoryProvider = (*DefaultHistoryProvider)(nil)
