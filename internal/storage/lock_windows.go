//go:build windows

package storage

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const lockfileExclusiveLock = 0x00000002

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
	procMoveFileExW  = kernel32.NewProc("MoveFileExW")
)

type osLock struct {
	f  *os.File
	ol syscall.Overlapped
}

func lockOS(f *os.File) (*osLock, error) {
	held := &osLock{f: f}
	r, _, err := procLockFileEx.Call(
		f.Fd(),
		uintptr(lockfileExclusiveLock),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&held.ol)),
	)
	if r == 0 {
		if err == nil {
			err = syscall.EINVAL
		}
		return nil, fmt.Errorf("lock storage: %w", err)
	}
	return held, nil
}

func (l *osLock) unlock() error {
	r, _, err := procUnlockFileEx.Call(
		l.f.Fd(),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&l.ol)),
	)
	if r == 0 {
		if err == nil {
			err = syscall.EINVAL
		}
		return fmt.Errorf("unlock storage: %w", err)
	}
	return nil
}

func moveReplace(src, dst string) error {
	from, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	to, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	const replaceExisting = 1
	const writeThrough = 8
	r, _, callErr := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(from)),
		uintptr(unsafe.Pointer(to)),
		uintptr(replaceExisting|writeThrough),
	)
	if r == 0 {
		if callErr == nil {
			callErr = syscall.EINVAL
		}
		return callErr
	}
	return nil
}
