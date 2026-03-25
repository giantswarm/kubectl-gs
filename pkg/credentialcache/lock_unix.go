//go:build !windows

package credentialcache

import (
	"os"
	"path/filepath"
	"syscall"
)

func lock(issuerURL, clientID string) (*os.File, error) {
	lockPath := lockFilePath(issuerURL, clientID)

	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600) //nolint:gosec // Path is constructed internally, not from user input
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil { //nolint:gosec // Fd() returns a small file descriptor
		_ = f.Close()
		return nil, err
	}

	return f, nil
}

func unlock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:gosec // Fd() returns a small file descriptor
	_ = f.Close()
}
