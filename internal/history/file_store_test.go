package history

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreAppendGetAndClear(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(FileStoreConfig{BaseDir: dir})
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := ConversationKey{PlayerID: "player", NikiID: "niki"}
	ctx := context.Background()

	err = store.AppendMessages(ctx, key, MessageBatch{
		{Role: RoleSystem, Content: "system"},
		{Role: RoleUser, Content: "user"},
	})
	if err != nil {
		t.Fatalf("AppendMessages: %v", err)
	}

	history, err := store.GetHistory(ctx, key, ReadOptions{})
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Content != "system" || history[1].Content != "user" {
		t.Fatalf("unexpected history contents: %+v", history)
	}

	limited, err := store.GetHistory(ctx, key, ReadOptions{LimitMessages: 1})
	if err != nil {
		t.Fatalf("GetHistory with limit: %v", err)
	}
	if len(limited) != 1 || limited[0].Content != "user" {
		t.Fatalf("expected last message, got %+v", limited)
	}

	summary := Message{Role: RoleSystem, Content: "summary"}
	if err := store.UpsertSummary(ctx, key, summary); err != nil {
		t.Fatalf("UpsertSummary: %v", err)
	}
	summaryPath := filepath.Join(dir, "player", "niki", "summary.json")
	if _, err := os.Stat(summaryPath); err != nil {
		t.Fatalf("expected summary file: %v", err)
	}

	if err := store.Clear(ctx, key); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "player", "niki")); !os.IsNotExist(err) {
		t.Fatalf("expected directory removed, got err=%v", err)
	}
}
