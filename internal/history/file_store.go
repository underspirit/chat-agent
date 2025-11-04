package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileStore persists conversation history to JSON files on disk.
type FileStore struct {
	baseDir string
	mu      sync.Mutex
}

// NewFileStore creates a file-backed history store rooted at the provided directory.
func NewFileStore(baseDir string) (*FileStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, errors.New("base directory must be provided")
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}

	return &FileStore{baseDir: baseDir}, nil
}

// GetHistory returns the stored conversation history, optionally respecting the read options.
func (s *FileStore) GetHistory(_ context.Context, key ConversationKey, opts ReadOptions) (MessageBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, err := s.readMessages(key)
	if err != nil {
		return nil, err
	}

	if opts.LimitMessages > 0 && len(messages) > opts.LimitMessages {
		messages = append(MessageBatch{}, messages[len(messages)-opts.LimitMessages:]...)
	} else {
		messages = append(MessageBatch{}, messages...)
	}

	return messages, nil
}

// AppendMessages appends the provided messages to the persisted history.
func (s *FileStore) AppendMessages(_ context.Context, key ConversationKey, messages MessageBatch) error {
	if len(messages) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.readMessages(key)
	if err != nil {
		return err
	}

	combined := append(existing, messages...)
	return s.writeMessages(key, combined)
}

// UpsertSummary stores the latest summary alongside the history file.
func (s *FileStore) UpsertSummary(_ context.Context, key ConversationKey, summary Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.summaryPath(key)
	tmp, err := os.CreateTemp(s.baseDir, "summary-*.json")
	if err != nil {
		return fmt.Errorf("create temp summary file: %w", err)
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("encode summary: %w", err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("close summary temp file: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("persist summary: %w", err)
	}

	return nil
}

// Clear removes the history and summary files for the conversation key.
func (s *FileStore) Clear(_ context.Context, key ConversationKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.historyPath(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Remove(s.summaryPath(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

func (s *FileStore) readMessages(key ConversationKey) (MessageBatch, error) {
	path := s.historyPath(key)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return MessageBatch{}, nil
		}
		return nil, fmt.Errorf("open history file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var messages MessageBatch
	if err := decoder.Decode(&messages); err != nil {
		if errors.Is(err, io.EOF) {
			return MessageBatch{}, nil
		}
		return nil, fmt.Errorf("decode history: %w", err)
	}

	return messages, nil
}

func (s *FileStore) writeMessages(key ConversationKey, messages MessageBatch) error {
	path := s.historyPath(key)
	tmp, err := os.CreateTemp(s.baseDir, "history-*.json")
	if err != nil {
		return fmt.Errorf("create temp history file: %w", err)
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(messages); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("encode history: %w", err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("close history temp file: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("persist history: %w", err)
	}

	return nil
}

func (s *FileStore) historyPath(key ConversationKey) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s__%s.json", sanitizeKey(key.PlayerID), sanitizeKey(key.NikiID)))
}

func (s *FileStore) summaryPath(key ConversationKey) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s__%s.summary.json", sanitizeKey(key.PlayerID), sanitizeKey(key.NikiID)))
}

func sanitizeKey(value string) string {
	if value == "" {
		return "_"
	}

	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	return builder.String()
}

var _ Store = (*FileStore)(nil)
