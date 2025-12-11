package credentialplugin

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	clientauthv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"

	"github.com/giantswarm/kubectl-gs/v5/pkg/oidc"
)

const (
	envIssuerURL    = "KUBECTL_GS_OIDC_ISSUER_URL"
	envClientID     = "KUBECTL_GS_OIDC_CLIENT_ID"
	envRefreshToken = "KUBECTL_GS_OIDC_REFRESH_TOKEN"

	// Cache file will be stored in user's home directory under .kube/.kubectl-gs/cache/
	cacheDirName = ".kube/.kubectl-gs"
	cacheSubDir  = "cache"
)

type runner struct {
	stderr io.Writer
	stdout io.Writer
	stdin  io.Reader
}

// execCredentialResponse represents the response structure for the credential plugin
type execCredentialResponse struct {
	APIVersion string                                  `json:"apiVersion"`
	Kind       string                                  `json:"kind"`
	Status     *clientauthv1beta1.ExecCredentialStatus `json:"status"`
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Read ExecCredential request from stdin
	// Check if stdin is a terminal - if so, skip reading
	if file, ok := r.stdin.(*os.File); ok {
		stat, err := file.Stat()
		if err == nil {
			// Check if stdin is a character device (terminal)
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				// It's a terminal, don't try to read from it
			} else {
				// It's a pipe or file, try to read (kubectl may send ExecCredential request)
				// Use a limited reader to avoid reading too much
				limitedReader := io.LimitReader(r.stdin, 4096)
				_, _ = io.ReadAll(limitedReader)
			}
		}
	} else {
		// Not a file, try to read with a limit (kubectl may send ExecCredential request)
		limitedReader := io.LimitReader(r.stdin, 4096)
		_, _ = io.ReadAll(limitedReader)
	}

	// Get OIDC configuration from environment variables
	issuerURL := os.Getenv(envIssuerURL)
	clientID := os.Getenv(envClientID)
	refreshToken := os.Getenv(envRefreshToken)

	if issuerURL == "" || clientID == "" || refreshToken == "" {
		return microerror.Maskf(credentialPluginError, "missing required environment variables: %s, %s, %s", envIssuerURL, envClientID, envRefreshToken)
	}

	// Check if we have a cached valid token before renewing
	cachedToken, err := r.getCachedToken(issuerURL, clientID)
	if err == nil && isValidIdToken(cachedToken) {
		// Use cached token if it's still valid
		idToken := cachedToken
		execCredential := execCredentialResponse{
			APIVersion: clientauthv1beta1.SchemeGroupVersion.String(),
			Kind:       "ExecCredential",
			Status: &clientauthv1beta1.ExecCredentialStatus{
				Token: idToken,
			},
		}

		encoder := json.NewEncoder(r.stdout)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(execCredential)
		if err != nil {
			return microerror.Maskf(credentialPluginError, "failed to encode ExecCredential: %v", err)
		}
		return nil
	}

	// Create OIDC authenticator
	var auther *oidc.Authenticator
	{
		oidcConfig := oidc.Config{
			ClientID: clientID,
			Issuer:   issuerURL,
		}

		var err error
		auther, err = oidc.New(ctx, oidcConfig)
		if err != nil {
			return microerror.Maskf(credentialPluginError, "failed to create OIDC authenticator: %v", err)
		}
	}

	// Renew token using refresh token
	idToken, newRefreshToken, err := auther.RenewToken(ctx, refreshToken)
	if err != nil {
		return microerror.Maskf(credentialPluginError, "failed to renew token: %v", err)
	}

	// Cache the new token
	_ = r.cacheToken(issuerURL, clientID, idToken, newRefreshToken)

	// Create ExecCredential response
	execCredential := execCredentialResponse{
		APIVersion: clientauthv1beta1.SchemeGroupVersion.String(),
		Kind:       "ExecCredential",
		Status: &clientauthv1beta1.ExecCredentialStatus{
			Token: idToken,
		},
	}

	// Output the ExecCredential response as JSON
	encoder := json.NewEncoder(r.stdout)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(execCredential)
	if err != nil {
		return microerror.Maskf(credentialPluginError, "failed to encode ExecCredential: %v", err)
	}

	return nil
}

// isValidIdToken checks if a JWT token is still valid (not expired)
func isValidIdToken(idToken string) bool {
	if idToken == "" {
		return false
	}

	idTokenParts := strings.Split(idToken, ".")
	if len(idTokenParts) != 3 {
		return false
	}

	rawIdTokenBody, err := base64.RawStdEncoding.DecodeString(idTokenParts[1])
	if err != nil {
		return false
	}

	var bodyMap map[string]json.RawMessage
	err = json.Unmarshal(rawIdTokenBody, &bodyMap)
	if err != nil {
		return false
	}

	var expTimestamp int64
	err = json.Unmarshal(bodyMap["exp"], &expTimestamp)
	if err != nil {
		return false
	}

	if expTimestamp == 0 {
		return false
	}

	expTime := time.Unix(expTimestamp, 0)
	// Add a 5 minute buffer to avoid using tokens that are about to expire
	return expTime.After(time.Now().Add(5 * time.Minute))
}

// getCachedToken retrieves a cached token from disk
func (r *runner) getCachedToken(issuerURL, clientID string) (string, error) {
	cacheFile := r.getCacheFilePath(issuerURL, clientID)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return "", err
	}

	var cache struct {
		IDToken string `json:"id_token"`
	}

	err = json.Unmarshal(data, &cache)
	if err != nil {
		return "", err
	}

	return cache.IDToken, nil
}

// cacheToken stores a token in a cache file
func (r *runner) cacheToken(issuerURL, clientID, idToken, refreshToken string) error {
	cacheDir := r.getCacheDir()
	err := os.MkdirAll(cacheDir, 0700)
	if err != nil {
		return err
	}

	cacheFile := r.getCacheFilePath(issuerURL, clientID)

	cache := struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		IDToken:      idToken,
		RefreshToken: refreshToken,
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0600)
}

// getCacheDir returns the cache directory path
func (r *runner) getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to temp directory if home directory is not available
		return filepath.Join(os.TempDir(), cacheDirName, cacheSubDir)
	}
	return filepath.Join(homeDir, cacheDirName, cacheSubDir)
}

// getCacheFilePath returns the cache file path for a given issuer and client ID
func (r *runner) getCacheFilePath(issuerURL, clientID string) string {
	// Create a hash of issuer+clientID to create a unique filename
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", issuerURL, clientID)))
	filename := fmt.Sprintf("token-%x.json", hash[:16]) // Use first 16 bytes for filename
	return filepath.Join(r.getCacheDir(), filename)
}
