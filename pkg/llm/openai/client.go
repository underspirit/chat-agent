package openai

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

	"chat-agent/pkg/llm"
)

const (
	defaultBaseURL = "https://api.openai.com/v1/chat/completions"
	defaultModel   = "gpt-3.5-turbo"
)

var ErrMissingAPIKey = errors.New("openai: api key is required")

type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

type Option func(*Client)

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = baseURL
		}
	}
}

func WithModel(model string) Option {
	return func(c *Client) {
		if strings.TrimSpace(model) != "" {
			c.model = model
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrMissingAPIKey
	}
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		model:   defaultModel,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Temperature *float32      `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type openAIStream struct {
	reader *bufio.Reader
	body   io.ReadCloser
}

func (s *openAIStream) Recv() (string, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", io.EOF
			}
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			return "", io.EOF
		}
		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return "", fmt.Errorf("openai: decode stream chunk: %w", err)
		}
		var builder strings.Builder
		for _, choice := range chunk.Choices {
			builder.WriteString(choice.Delta.Content)
		}
		text := builder.String()
		if text == "" {
			continue
		}
		return text, nil
	}
}

func (s *openAIStream) Close() error {
	if s.body != nil {
		return s.body.Close()
	}
	return nil
}

func (c *Client) CreateChatCompletionStream(ctx context.Context, messages []llm.Message) (llm.Stream, error) {
	if c == nil {
		return nil, errors.New("openai: client is nil")
	}
	if c.apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	reqPayload := chatRequest{
		Model:  c.model,
		Stream: true,
	}
	for _, msg := range messages {
		if strings.TrimSpace(msg.Role) == "" || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		reqPayload.Messages = append(reqPayload.Messages, chatMessage{Role: msg.Role, Content: msg.Content})
	}
	if len(reqPayload.Messages) == 0 {
		return nil, errors.New("openai: no messages to send")
	}

	rawBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("openai: encode request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("openai: send request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()
		limited := io.LimitReader(response.Body, 4096)
		body, _ := io.ReadAll(limited)
		return nil, fmt.Errorf("openai: unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	return &openAIStream{
		reader: bufio.NewReader(response.Body),
		body:   response.Body,
	}, nil
}
