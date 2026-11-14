//go:build !windows

package atomicfile

import "os"

func open(path string) (*os.File, error) {
	return os.Open(path) //nolint:gosec,wrapcheck // the caller picks the path and wraps the error
}
