package credentialcache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	cacheDirName = ".kube/.kubectl-gs"
	cacheSubDir  = "cache"
)

type Entry struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

func WriteWithLock(issuerURL, clientID, idToken, refreshToken string) error {
	f, err := lock(issuerURL, clientID)
	if err != nil {
		return err
	}
	defer unlock(f)

	dir := cacheDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(Entry{IDToken: idToken, RefreshToken: refreshToken}) //nolint:gosec // Field name matches pattern but contains no hardcoded credential
	if err != nil {
		return err
	}

	return os.WriteFile(filePath(issuerURL, clientID), data, 0600)
}

func ReadWithLock(issuerURL, clientID string) (Entry, error) {
	f, err := lock(issuerURL, clientID)
	if err != nil {
		return Entry{}, err
	}
	defer unlock(f)

	data, err := os.ReadFile(filePath(issuerURL, clientID))
	if err != nil {
		return Entry{}, err
	}

	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return Entry{}, err
	}

	return e, nil
}

// Lock acquires an exclusive file lock for the cache entry identified by
// issuerURL and clientID. The caller must call Unlock to release it.
// This allows callers to hold the lock across operations (e.g. token renewal)
// to prevent concurrent processes from consuming the same refresh token.
func Lock(issuerURL, clientID string) (*os.File, error) {
	return lock(issuerURL, clientID)
}

// Unlock releases a lock previously acquired with Lock.
func Unlock(f *os.File) {
	unlock(f)
}

// ReadWithoutLock reads the cache entry without acquiring a lock.
func ReadWithoutLock(issuerURL, clientID string) (Entry, error) {
	data, err := os.ReadFile(filePath(issuerURL, clientID))
	if err != nil {
		return Entry{}, err
	}

	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return Entry{}, err
	}

	return e, nil
}

// WriteWithoutLock writes the cache entry without acquiring a lock.
func WriteWithoutLock(issuerURL, clientID, idToken, refreshToken string) error {
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(Entry{IDToken: idToken, RefreshToken: refreshToken})
	if err != nil {
		return err
	}

	return os.WriteFile(filePath(issuerURL, clientID), data, 0600)
}

func lockFilePath(issuerURL, clientID string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", issuerURL, clientID)))
	return filepath.Join(cacheDir(), fmt.Sprintf("token-%x.lock", hash[:16]))
}

func filePath(issuerURL, clientID string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", issuerURL, clientID)))
	return filepath.Join(cacheDir(), fmt.Sprintf("token-%x.json", hash[:16]))
}

func cacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), cacheDirName, cacheSubDir)
	}
	return filepath.Join(homeDir, cacheDirName, cacheSubDir)
}
