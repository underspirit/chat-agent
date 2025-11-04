package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIConfig controls how the OpenAIProvider communicates with the upstream API.
type OpenAIConfig struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// OpenAIProvider implements StreamingProvider using OpenAI compatible chat completions.
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider creates a provider that talks to an OpenAI compatible endpoint.
func NewOpenAIProvider(cfg OpenAIConfig) (*OpenAIProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("openai api key must be provided")
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	return &OpenAIProvider{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		httpClient: client,
	}, nil
}

// StreamChat performs a streaming chat completion request against the configured endpoint.
func (p *OpenAIProvider) StreamChat(ctx context.Context, messages []ChatMessage, params GenerationParams) (<-chan PartialChunk, error) {
	if params.Model == "" {
		return nil, errors.New("model must be provided for OpenAI chat completion")
	}

	payload := openAIChatRequest{
		Model:    params.Model,
		Stream:   true,
		Messages: make([]openAIChatMessage, 0, len(messages)),
	}

	if params.MaxTokens > 0 {
		payload.MaxTokens = params.MaxTokens
	}
	if params.Temperature > 0 {
		payload.Temperature = params.Temperature
	}
	if len(params.Stop) > 0 {
		payload.Stop = params.Stop
	}

	for _, msg := range messages {
		payload.Messages = append(payload.Messages, openAIChatMessage{Role: msg.Role, Content: msg.Content})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform openai request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("openai request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	chunks := make(chan PartialChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		sentDone := false
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return
				}
				line = strings.TrimSpace(line)
				if line == "" {
					return
				}
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				sentDone = true
				chunks <- PartialChunk{Done: true}
				return
			}

			var event openAIStreamResponse
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				continue
			}

			for _, choice := range event.Choices {
				delta := choice.Delta.Content
				if delta == "" {
					continue
				}
				chunks <- PartialChunk{Delta: delta, Done: choice.FinishReason == "stop"}
				if choice.FinishReason == "stop" {
					sentDone = true
				}
			}
		}

		if !sentDone {
			chunks <- PartialChunk{Done: true}
		}
	}()

	return chunks, nil
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float32             `json:"temperature,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

var _ StreamingProvider = (*OpenAIProvider)(nil)
