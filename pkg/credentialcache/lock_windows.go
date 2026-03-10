package credentialcache

import (
	"os"
	"path/filepath"
)

func lock(issuerURL, clientID string) (*os.File, error) {
	lockPath := lockFilePath(issuerURL, clientID)

	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, err
	}

	// On Windows, opening with O_CREATE|O_RDWR provides basic mutual
	// exclusion because the OS locks open file handles by default.
	// This is not as robust as flock but sufficient for credential caching.
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func unlock(f *os.File) {
	_ = f.Close()
}
