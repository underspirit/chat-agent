package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAIProviderStreamsChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		lines := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" world"}}]}`,
			"data: [DONE]",
		}
		for _, line := range lines {
			_, _ = w.Write([]byte(line + "\n\n"))
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider(OpenAIConfig{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		Model:      "test-model",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewOpenAIProvider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, err := provider.StreamChat(ctx, []ChatMessage{{Role: "user", Content: "hi"}}, GenerationParams{})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var builder strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		builder.WriteString(chunk.Delta)
		if chunk.Done {
			break
		}
	}

	if builder.String() != "Hello world" {
		t.Fatalf("unexpected stream contents: %q", builder.String())
	}
}

func TestOpenAIProviderHandlesErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider(OpenAIConfig{APIKey: "test", BaseURL: server.URL, Model: "test-model", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewOpenAIProvider: %v", err)
	}

	_, err = provider.StreamChat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, GenerationParams{})
	if err == nil {
		t.Fatalf("expected error from StreamChat")
	}
}
