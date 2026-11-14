package sessionstore

import (
	"bytes"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gotd/td/session"
)

func TestLoadMissing(t *testing.T) {
	s := Open(filepath.Join(t.TempDir(), "session.json"))
	if _, err := s.LoadSession(t.Context()); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("LoadSession() error = %v, want ErrNotFound", err)
	}
}

func TestConcurrentClientsNeverSeeTornSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	payloads := [][]byte{bytes.Repeat([]byte("a"), 64<<10), bytes.Repeat([]byte("b"), 64<<10)}
	if err := Open(path).StoreSession(t.Context(), payloads[0]); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := range 8 {
		s := Open(path)
		wg.Go(func() {
			for range 20 {
				if err := s.StoreSession(t.Context(), payloads[i%2]); err != nil {
					t.Error(err)

					return
				}
				got, err := s.LoadSession(t.Context())
				if err != nil {
					t.Error(err)

					return
				}
				if !bytes.Equal(got, payloads[0]) && !bytes.Equal(got, payloads[1]) {
					t.Errorf("read %d bytes that match neither write", len(got))

					return
				}
			}
		})
	}
	wg.Wait()
}
