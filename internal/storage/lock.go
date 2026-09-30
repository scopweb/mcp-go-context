package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Lock exclusively locks dir/store.lock until the returned function is called.
// The operating system releases the lock if the process exits.
// The lock is not reentrant within the same process.
func Lock(dir string) (func() error, error) {
	if dir == "" {
		return nil, fmt.Errorf("storage directory is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "store.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open store lock: %w", err)
	}
	held, err := lockOS(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() error {
		return errors.Join(held.unlock(), f.Close())
	}, nil
}
