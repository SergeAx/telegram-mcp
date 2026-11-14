// Package sessionstore keeps a gotd session in a file that every client of
// the process shares.
package sessionstore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"github.com/gotd/td/session"

	"github.com/chaindead/telegram-mcp/internal/atomicfile"
)

// Store implements telegram.SessionStorage. Share one Store per file: tool
// calls run concurrently, each with its own client.
type Store struct {
	path string
	mu   sync.Mutex
}

func Open(path string) *Store {
	return &Store{path: path}
}

func (s *Store) LoadSession(context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := atomicfile.Read(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}

	return data, nil
}

func (s *Store) StoreSession(_ context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := atomicfile.Write(s.path, data); err != nil {
		return fmt.Errorf("store session: %w", err)
	}

	return nil
}
