package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIProviderStreamChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if !payload.Stream {
			t.Fatalf("expected streaming request")
		}
		if payload.Model != "gpt-test" {
			t.Fatalf("unexpected model: %s", payload.Model)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"},\"finish_reason\":\"stop\"}]}\n\n"))
		flusher.Flush()
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider(OpenAIConfig{APIKey: "test", BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, err := provider.StreamChat(ctx, []ChatMessage{{Role: "user", Content: "hi"}}, GenerationParams{Model: "gpt-test"})
	if err != nil {
		t.Fatalf("stream chat failed: %v", err)
	}

	var deltas []PartialChunk
	for chunk := range ch {
		deltas = append(deltas, chunk)
	}

	if len(deltas) != 3 {
		t.Fatalf("expected 3 chunks including done, got %d", len(deltas))
	}
	if deltas[0].Delta != "Hello" {
		t.Fatalf("unexpected first delta: %#v", deltas[0])
	}
	if deltas[1].Delta != " world" || !deltas[1].Done {
		t.Fatalf("unexpected second delta: %#v", deltas[1])
	}
	if !deltas[2].Done || deltas[2].Delta != "" {
		t.Fatalf("expected final done chunk: %#v", deltas[2])
	}
}
