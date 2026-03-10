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

func Write(issuerURL, clientID, idToken, refreshToken string) error {
	f, err := lock(issuerURL, clientID)
	if err != nil {
		return err
	}
	defer unlock(f)

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

func Read(issuerURL, clientID string) (Entry, error) {
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
