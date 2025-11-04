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
}

// NewOrchestrator constructs a new Orchestrator instance with the required dependencies.
func NewOrchestrator(store history.Store, builder prompt.Builder, manager contextmgr.Manager, provider llm.StreamingProvider, logger logging.Logger) *Orchestrator {
	return &Orchestrator{
		historyStore:  store,
		promptBuilder: builder,
		ctxManager:    manager,
		llmProvider:   provider,
		logger:        logger,
	}
}

func (o *Orchestrator) HandleChat(ctx context.Context, req ChatRequest, stream Stream) error {
	if err := o.validate(req); err != nil {
		return err
	}

	log := o.logger.With(logging.Field{Key: "player_id", Value: req.PlayerID}, logging.Field{Key: "niki_id", Value: req.NikiID})
	log.Info("handling chat request")

	key := history.ConversationKey{PlayerID: req.PlayerID, NikiID: req.NikiID}

	historyBatch, err := o.historyStore.GetHistory(ctx, key, history.ReadOptions{})
	if err != nil {
		log.Error(fmt.Sprintf("failed to load history: %v", err))
		return err
	}

	now := time.Now().Unix()
	userMessage := history.Message{Role: history.RoleUser, Content: req.InputText, Timestamp: now}
	personaMessage := history.Message{Role: history.RoleSystem, Content: fmt.Sprintf("You are %s (persona %s).", req.NikiName, req.NikiID), Timestamp: now}
	systemMessage := history.Message{Role: history.RoleSystem, Content: fmt.Sprintf("You are chatting with %s (%s). Provide helpful, concise answers.", req.PlayerNickname, req.PlayerID), Timestamp: now}

	buildCtx := prompt.BuildContext{
		SystemMessages: []history.Message{systemMessage},
		Persona:        personaMessage,
		History:        historyBatch,
		CurrentInput:   userMessage,
	}

	promptResult, err := o.promptBuilder.Build(ctx, buildCtx)
	if err != nil {
		log.Error(fmt.Sprintf("failed to build prompt: %v", err))
		return err
	}

	truncation, err := o.ctxManager.Truncate(ctx, promptResult.Messages, promptResult.Params)
	if err != nil {
		log.Error(fmt.Sprintf("context truncation failed: %v", err))
		return err
	}

	streamCh, err := o.llmProvider.StreamChat(ctx, truncation.Messages, truncation.Params)
	if err != nil {
		log.Error(fmt.Sprintf("llm stream call failed: %v", err))
		return err
	}

	var responseBuilder strings.Builder

	for chunk := range streamCh {
		if chunk.Delta != "" {
			if err := stream.SendChunk(ctx, ChatChunk{Text: chunk.Delta, Done: chunk.Done}); err != nil {
				log.Error(fmt.Sprintf("failed to send chunk: %v", err))
				return err
			}
			responseBuilder.WriteString(chunk.Delta)
		}
	}

	assistantMessage := history.Message{Role: history.RoleAssistant, Content: responseBuilder.String(), Timestamp: time.Now().Unix()}

	if err := o.historyStore.AppendMessages(ctx, key, history.MessageBatch{userMessage, assistantMessage}); err != nil {
		log.Error(fmt.Sprintf("failed to persist history: %v", err))
		return err
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
