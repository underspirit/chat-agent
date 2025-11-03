package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"chat-agent/internal/app"
	"chat-agent/internal/contextmgr"
	"chat-agent/internal/conversation"
	"chat-agent/internal/history"
	"chat-agent/internal/llm"
	"chat-agent/internal/logging"
	"chat-agent/internal/prompt"
	"chat-agent/internal/server"
)

func main() {
	logger := logging.NewStdLogger()

	historyDir := os.Getenv("CHAT_AGENT_HISTORY_DIR")
	if historyDir == "" {
		historyDir = filepath.Join("data", "history")
	}

	historyStore, err := history.NewFileStore(historyDir)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	promptBuilder, err := prompt.NewSimpleBuilder(prompt.SimpleBuilderConfig{
		DefaultModel: envOrDefault("CHAT_AGENT_LLM_MODEL", "gpt-4o-mini"),
		MaxTokens:    512,
		Temperature:  0.7,
	})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	ctxManager := contextmgr.NewNoopManager()

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")

	llmProvider, err := llm.NewOpenAIProvider(llm.OpenAIConfig{APIKey: apiKey, BaseURL: baseURL})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	manager := conversation.NewOrchestrator(historyStore, promptBuilder, ctxManager, llmProvider, logger)
	grpcServer := server.NewChatAgentGRPCServer(manager, logger)

	application, err := app.New(app.Config{GRPCPort: 50051}, logger, grpcServer)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := application.Run(ctx); err != nil {
		logger.Error(err.Error())
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
