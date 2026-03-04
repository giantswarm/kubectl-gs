package credentialcache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	cacheDirName = ".kube/.kubectl-gs"
	cacheSubDir  = "cache"
)

type Entry struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

func Write(issuerURL, clientID, idToken, refreshToken string) error {
	cacheDir := cacheDir()
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(Entry{IDToken: idToken, RefreshToken: refreshToken})
	if err != nil {
		return err
	}

	return os.WriteFile(filePath(issuerURL, clientID), data, 0600)
}

func Read(issuerURL, clientID string) (Entry, error) {
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

func FilePath(issuerURL, clientID string) string {
	return filePath(issuerURL, clientID)
}

func LockFilePath(issuerURL, clientID string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", issuerURL, clientID)))
	return filepath.Join(cacheDir(), fmt.Sprintf("token-%x.lock", hash[:16]))
}

// Lock acquires an exclusive advisory lock for the given issuer and client ID.
// The caller must call Unlock when done.
func Lock(issuerURL, clientID string) (*os.File, error) {
	lockPath := LockFilePath(issuerURL, clientID)

	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}

	return f, nil
}

// Unlock releases the lock acquired by Lock.
func Unlock(f *os.File) {
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
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
