// Package sessionstore keeps a gotd session in a file that every client of
// the process shares.
package sessionstore

import (
	"context"
	"fmt"
	"sync"

	"github.com/chaindead/telegram-mcp/internal/atomicfile"
	"github.com/chaindead/telegram-mcp/internal/secret"
)

// Store implements telegram.SessionStorage. Share one Store per file: tool
// calls run concurrently, each with its own client.
//
// A missing file is an error rather than session.ErrNotFound, which would
// make gotd create a new key that is not logged in and save it over the
// session: only auth starts sessions.
type Store struct {
	path string
	key  *secret.Key
	mu   sync.Mutex
}

// Open keeps the session encrypted with key, or in plain JSON if key is nil.
func Open(path string, key *secret.Key) *Store {
	return &Store{path: path, key: key}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) LoadSession(context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := atomicfile.Read(s.path)
	if err != nil {
		return nil, fmt.Errorf("read session (telegram-mcp auth creates it): %w", err)
	}
	if s.key == nil {
		return data, nil
	}

	plain, err := s.key.Decrypt(secret.Armored(data))
	if err != nil {
		return nil, fmt.Errorf("session %s: %w", s.path, err)
	}

	return plain, nil
}

func (s *Store) StoreSession(_ context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.key != nil {
		text, err := s.key.Encrypt(data)
		if err != nil {
			return fmt.Errorf("encrypt session: %w", err)
		}
		data = []byte(text)
	}

	if err := atomicfile.Write(s.path, data); err != nil {
		return fmt.Errorf("store session: %w", err)
	}

	return nil
}
