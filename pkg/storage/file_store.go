package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileStore struct {
	baseDir string
	mu      sync.Mutex
}

func NewFileStore(baseDir string) (*FileStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, errors.New("base directory is required")
	}
	return &FileStore{baseDir: baseDir}, nil
}

func (s *FileStore) Load(ctx context.Context, playerID, nikiID string) ([]Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := s.filePath(playerID, nikiID)
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read conversation file: %w", err)
	}
	var messages []Message
	if len(data) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("decode conversation: %w", err)
	}
	return messages, nil
}

func (s *FileStore) Save(ctx context.Context, playerID, nikiID string, messages []Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return fmt.Errorf("create base directory: %w", err)
	}

	payload, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("encode conversation: %w", err)
	}

	path := s.filePath(playerID, nikiID)
	tmpFile := path + ".tmp"

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.WriteFile(tmpFile, payload, 0o644); err != nil {
		return fmt.Errorf("write temp conversation: %w", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		return fmt.Errorf("replace conversation file: %w", err)
	}
	return nil
}

func (s *FileStore) filePath(playerID, nikiID string) string {
	file := fmt.Sprintf("%s__%s.json", sanitizeSegment(playerID), sanitizeSegment(nikiID))
	return filepath.Join(s.baseDir, file)
}

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	cleaned := make([]rune, 0, len(value))
	for _, r := range value {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			cleaned = append(cleaned, '_')
		default:
			cleaned = append(cleaned, r)
		}
	}
	return string(cleaned)
}
