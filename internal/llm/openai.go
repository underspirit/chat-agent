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

// OpenAIConfig configures the OpenAIProvider.
type OpenAIConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// OpenAIProvider implements StreamingProvider using OpenAI's Chat Completions API.
type OpenAIProvider struct {
	client  *http.Client
	apiKey  string
	base    string
	model   string
	timeout time.Duration
}

const defaultOpenAIBaseURL = "https://api.openai.com/v1"
const defaultHTTPTimeout = 60 * time.Second

// NewOpenAIProvider constructs an OpenAI-backed provider.
func NewOpenAIProvider(cfg OpenAIConfig) (*OpenAIProvider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("openai api key must be provided")
	}
	base := cfg.BaseURL
	if base == "" {
		base = defaultOpenAIBaseURL
	}
	model := cfg.Model
	if model == "" {
		return nil, errors.New("openai model must be provided")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultHTTPTimeout
	}
	return &OpenAIProvider{client: client, apiKey: cfg.APIKey, base: strings.TrimRight(base, "/"), model: model, timeout: timeout}, nil
}

// StreamChat implements StreamingProvider by calling OpenAI's streaming chat completions endpoint.
func (p *OpenAIProvider) StreamChat(ctx context.Context, messages []ChatMessage, params GenerationParams) (<-chan PartialChunk, error) {
	if len(messages) == 0 {
		return nil, errors.New("at least one message must be provided")
	}

	reqBody := openAIChatRequest{
		Model:       coalesce(params.Model, p.model),
		Stream:      true,
		Temperature: params.Temperature,
	}
	if params.MaxTokens > 0 {
		reqBody.MaxTokens = params.MaxTokens
	}
	reqBody.Messages = make([]openAIMessage, 0, len(messages))
	for _, msg := range messages {
		reqBody.Messages = append(reqBody.Messages, openAIMessage{Role: msg.Role, Content: msg.Content})
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	ctxReq := ctx
	var cancel context.CancelFunc
	if p.timeout > 0 {
		ctxReq, cancel = context.WithTimeout(ctx, p.timeout)
	}

	req, err := http.NewRequestWithContext(ctxReq, http.MethodPost, fmt.Sprintf("%s/chat/completions", p.base), bytes.NewReader(payload))
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(req)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("perform request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	ch := make(chan PartialChunk)
	go func() {
		defer resp.Body.Close()
		if cancel != nil {
			defer cancel()
		}
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				ch <- PartialChunk{Done: true}
				return
			}
			var chunk openAIStreamResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- PartialChunk{Err: fmt.Errorf("decode chunk: %w", err), Done: true}
				return
			}
			for _, choice := range chunk.Choices {
				delta := choice.Delta.Content
				if delta != "" {
					ch <- PartialChunk{Delta: delta}
				}
				if choice.FinishReason != nil {
					ch <- PartialChunk{Done: true}
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- PartialChunk{Err: fmt.Errorf("stream read: %w", err), Done: true}
		} else {
			ch <- PartialChunk{Done: true}
		}
	}()

	return ch, nil
}

func coalesce(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature float32         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

var _ StreamingProvider = (*OpenAIProvider)(nil)
