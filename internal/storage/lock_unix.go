//go:build !windows

package storage

import (
	"fmt"
	"os"
	"syscall"
)

type osLock struct {
	f *os.File
}

func lockOS(f *os.File) (*osLock, error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return nil, fmt.Errorf("lock storage: %w", err)
	}
	return &osLock{f: f}, nil
}

func (l *osLock) unlock() error {
	if err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock storage: %w", err)
	}
	return nil
}
