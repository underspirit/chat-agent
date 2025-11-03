package history

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStoreConfig configures the behaviour of FileStore.
type FileStoreConfig struct {
	// BaseDir determines where conversation directories are written.
	BaseDir string
	// FileMode controls the permissions of newly created files. Defaults to 0o600 when zero.
	FileMode os.FileMode
	// DirMode controls the permissions of newly created directories. Defaults to 0o750 when zero.
	DirMode os.FileMode
}

// FileStore persists conversation history using JSONL files on disk.
type FileStore struct {
	cfg FileStoreConfig
	mu  sync.Mutex
}

// NewFileStore constructs a FileStore rooted at cfg.BaseDir.
func NewFileStore(cfg FileStoreConfig) (*FileStore, error) {
	if cfg.BaseDir == "" {
		return nil, errors.New("base directory must be provided")
	}
	if cfg.DirMode == 0 {
		cfg.DirMode = 0o750
	}
	if cfg.FileMode == 0 {
		cfg.FileMode = 0o600
	}
	if err := os.MkdirAll(cfg.BaseDir, cfg.DirMode); err != nil {
		return nil, fmt.Errorf("create base directory: %w", err)
	}
	return &FileStore{cfg: cfg}, nil
}

func (s *FileStore) historyPath(key ConversationKey) string {
	return filepath.Join(s.cfg.BaseDir, key.PlayerID, key.NikiID, "history.jsonl")
}

func (s *FileStore) summaryPath(key ConversationKey) string {
	return filepath.Join(s.cfg.BaseDir, key.PlayerID, key.NikiID, "summary.json")
}

// GetHistory returns the stored conversation history ordered by insertion time.
func (s *FileStore) GetHistory(ctx context.Context, key ConversationKey, opts ReadOptions) (MessageBatch, error) {
	path := s.historyPath(key)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MessageBatch{}, nil
		}
		return nil, fmt.Errorf("open history file: %w", err)
	}
	defer file.Close()

	var messages MessageBatch
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			return nil, fmt.Errorf("decode history entry: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("scan history: %w", err)
		}
	}

	if opts.LimitMessages > 0 && len(messages) > opts.LimitMessages {
		messages = append(MessageBatch{}, messages[len(messages)-opts.LimitMessages:]...)
	}

	return messages, nil
}

// AppendMessages appends the provided messages to the JSONL history file.
func (s *FileStore) AppendMessages(ctx context.Context, key ConversationKey, messages MessageBatch) error {
	if len(messages) == 0 {
		return nil
	}

	path := s.historyPath(key)
	dir := filepath.Dir(path)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(dir, s.cfg.DirMode); err != nil {
		return fmt.Errorf("create conversation directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, s.cfg.FileMode)
	if err != nil {
		return fmt.Errorf("open history file for append: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, msg := range messages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if msg.Timestamp == 0 {
			msg.Timestamp = time.Now().Unix()
		}
		if err := encoder.Encode(msg); err != nil {
			return fmt.Errorf("encode history message: %w", err)
		}
	}
	return nil
}

// UpsertSummary stores or replaces a summary file for the conversation.
func (s *FileStore) UpsertSummary(ctx context.Context, key ConversationKey, summary Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path := s.summaryPath(key)
	dir := filepath.Dir(path)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(dir, s.cfg.DirMode); err != nil {
		return fmt.Errorf("create summary directory: %w", err)
	}

	if summary.Timestamp == 0 {
		summary.Timestamp = time.Now().Unix()
	}

	data, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), s.cfg.FileMode); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}

// Clear removes the stored history and summary for the conversation.
func (s *FileStore) Clear(ctx context.Context, key ConversationKey) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	dir := filepath.Dir(s.historyPath(key))

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.RemoveAll(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove conversation directory: %w", err)
	}
	return nil
}

var _ Store = (*FileStore)(nil)
