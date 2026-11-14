package sessionstore

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gotd/td/session"

	"github.com/chaindead/telegram-mcp/internal/secret"
)

func TestLoadMissingIsNotNewSession(t *testing.T) {
	s := Open(filepath.Join(t.TempDir(), "session.json"), nil)
	_, err := s.LoadSession(t.Context())
	if !errors.Is(err, fs.ErrNotExist) || errors.Is(err, session.ErrNotFound) {
		t.Fatalf("LoadSession() error = %v, want fs.ErrNotExist and not session.ErrNotFound", err)
	}
}

func TestEncrypted(t *testing.T) {
	key, err := secret.Generate()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "1.session.age")
	if err := Open(path, key).StoreSession(t.Context(), []byte(`{"Version":1}`)); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), "-----BEGIN AGE ENCRYPTED FILE-----") {
		t.Errorf("stored session is not age armor: %q", raw)
	}

	got, err := Open(path, key).LoadSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"Version":1}` {
		t.Errorf("LoadSession() = %q", got)
	}
}

func TestConcurrentClientsNeverSeeTornSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	payloads := [][]byte{bytes.Repeat([]byte("a"), 64<<10), bytes.Repeat([]byte("b"), 64<<10)}
	if err := Open(path, nil).StoreSession(t.Context(), payloads[0]); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := range 8 {
		s := Open(path, nil)
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
