package history

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreAppendAndGetHistory(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	key := ConversationKey{PlayerID: "player-1", NikiID: "niki-42"}
	now := time.Now().Unix()
	ctx := context.Background()

	batch := MessageBatch{
		{Role: RoleUser, Content: "hi", Timestamp: now},
		{Role: RoleAssistant, Content: "hello", Timestamp: now + 1},
	}

	if err := store.AppendMessages(ctx, key, batch); err != nil {
		t.Fatalf("append messages failed: %v", err)
	}

	history, err := store.GetHistory(ctx, key, ReadOptions{})
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}

	limited, err := store.GetHistory(ctx, key, ReadOptions{LimitMessages: 1})
	if err != nil {
		t.Fatalf("get limited history failed: %v", err)
	}
	if len(limited) != 1 || limited[0].Content != "hello" {
		t.Fatalf("unexpected limited history: %#v", limited)
	}

	summary := Message{Role: RoleSystem, Content: "summary", Timestamp: now + 2}
	if err := store.UpsertSummary(ctx, key, summary); err != nil {
		t.Fatalf("upsert summary failed: %v", err)
	}

	summaryPath := filepath.Join(dir, "player-1__niki-42.summary.json")
	if _, err := os.Stat(summaryPath); err != nil {
		t.Fatalf("expected summary file: %v", err)
	}

	if err := store.Clear(ctx, key); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	cleared, err := store.GetHistory(ctx, key, ReadOptions{})
	if err != nil {
		t.Fatalf("get history after clear failed: %v", err)
	}
	if len(cleared) != 0 {
		t.Fatalf("expected empty history after clear, got %d", len(cleared))
	}

	if _, err := os.Stat(summaryPath); !os.IsNotExist(err) {
		t.Fatalf("expected summary file removed, stat err=%v", err)
	}
}
