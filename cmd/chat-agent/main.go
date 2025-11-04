package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
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

	config := loadConfig()

	historyStore, err := history.NewFileStore(history.FileStoreConfig{BaseDir: config.DataDir})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	promptBuilder, err := prompt.NewSimpleBuilder(prompt.SimpleConfig{
		DefaultSystem: []string{
			"Respond in the tone of a friendly, thoughtful AI companion.",
			"Ensure replies comply with safety policies and remain concise unless the player requests detail.",
		},
		Model:       config.Model,
		Temperature: 0.7,
		MaxTokens:   1024,
	})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	ctxManager := contextmgr.NewNoopManager()

	llmProvider, err := llm.NewOpenAIProvider(llm.OpenAIConfig{
		APIKey:  config.OpenAIAPIKey,
		BaseURL: config.OpenAIBaseURL,
		Model:   config.Model,
	})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	manager := conversation.NewOrchestrator(conversation.OrchestratorConfig{SystemPrompts: config.SystemPrompts}, historyStore, promptBuilder, ctxManager, llmProvider, logger)
	grpcServer := server.NewChatAgentGRPCServer(manager, logger)

	application, err := app.New(app.Config{GRPCPort: config.GRPCPort}, logger, grpcServer)
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

type serviceConfig struct {
	GRPCPort      int
	DataDir       string
	Model         string
	OpenAIAPIKey  string
	OpenAIBaseURL string
	SystemPrompts []string
}

func loadConfig() serviceConfig {
	port := 50051
	if val := os.Getenv("CHAT_AGENT_GRPC_PORT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			port = parsed
		}
	}

	dataDir := os.Getenv("CHAT_AGENT_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	prompts := []string{
		"Follow platform safety guidelines and respond in the player's preferred language when possible.",
	}

	return serviceConfig{
		GRPCPort:      port,
		DataDir:       dataDir,
		Model:         model,
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL: os.Getenv("OPENAI_BASE_URL"),
		SystemPrompts: prompts,
	}
}
