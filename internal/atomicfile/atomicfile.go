// Package atomicfile replaces files so that a reader sees either the old
// content or the new one, never a torn write.
package atomicfile

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Windows briefly locks a file while it is being replaced, so both sides retry.
const (
	attempts = 20
	backoff  = 25 * time.Millisecond
)

// Write replaces path with data, readable by the owner only.
func Write(path string, data []byte) (err error) {
	dir, name := filepath.Split(path)
	root, err := os.OpenRoot(cmp.Or(dir, "."))
	if err != nil {
		return fmt.Errorf("open %s: %w", dir, err)
	}
	defer root.Close()

	tmp, err := os.CreateTemp(root.Name(), "."+name+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("write temp file: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("sync temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Unlike os.Rename, Root.Rename uses POSIX semantics on Windows and
	// replaces a file that readers still hold open.
	err = retry(func() error { return root.Rename(filepath.Base(tmp.Name()), name) })
	if err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}

	return nil
}

// Read reads a file that Write may be replacing at the same time.
func Read(path string) ([]byte, error) {
	var data []byte
	err := retry(func() error {
		f, err := open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		data, err = io.ReadAll(f)

		return err //nolint:wrapcheck // wrapped once below
	})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

func retry(op func() error) error {
	var err error
	for range attempts {
		if err = op(); err == nil || errors.Is(err, fs.ErrNotExist) {
			return err
		}
		time.Sleep(backoff)
	}

	return err
}
