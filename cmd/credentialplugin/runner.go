package credentialplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	clientauthv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"

	"github.com/giantswarm/kubectl-gs/v5/pkg/credentialcache"
	"github.com/giantswarm/kubectl-gs/v5/pkg/oidc"
)

const (
	envIssuerURL    = "KUBECTL_GS_OIDC_ISSUER_URL"
	envClientID     = "KUBECTL_GS_OIDC_CLIENT_ID"
	envRefreshToken = "KUBECTL_GS_OIDC_REFRESH_TOKEN"
	envIDToken      = "KUBECTL_GS_OIDC_ID_TOKEN"
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

	// Get OIDC configuration from environment variables
	issuerURL := os.Getenv(envIssuerURL)
	clientID := os.Getenv(envClientID)
	refreshToken := os.Getenv(envRefreshToken)

	if issuerURL == "" || clientID == "" || refreshToken == "" {
		return microerror.Maskf(credentialPluginError, "missing required environment variables: %s, %s, %s", envIssuerURL, envClientID, envRefreshToken)
	}

	// Check if the id_token passed from login is still valid
	if idTokenFromEnv := os.Getenv(envIDToken); oidc.IsValidIDToken(idTokenFromEnv) {
		return r.outputExecCredential(idTokenFromEnv)
	}

	// Fast path: return a cached token if still valid (no lock needed).
	if cached, err := credentialcache.Read(issuerURL, clientID); err == nil && oidc.IsValidIDToken(cached.IDToken) {
		return r.outputExecCredential(cached.IDToken)
	}

	// Create the OIDC authenticator before acquiring the lock to keep lock
	// duration short (provider discovery requires an HTTP round-trip).
	var auther *oidc.Authenticator
	{
		oidcConfig := oidc.Config{
			ClientID: clientID,
			Issuer:   issuerURL,
		}

		var err error
		auther, err = oidc.New(ctx, oidcConfig)
		if err != nil {
			return microerror.Maskf(credentialPluginError, "failed to create OIDC authenticator for %s: %v", issuerURL, err)
		}
	}

	// Acquire an exclusive lock to prevent concurrent processes from
	// simultaneously consuming the same refresh token.
	tokenLock, lockErr := credentialcache.Lock(issuerURL, clientID)
	if lockErr == nil {
		defer credentialcache.Unlock(tokenLock)
	}

	tokenSource := "kubeconfig"
	if lockErr == nil {
		// Under lock: re-check cache in case another process renewed while we waited.
		if cached, err := credentialcache.ReadLocked(issuerURL, clientID); err == nil {
			if oidc.IsValidIDToken(cached.IDToken) {
				return r.outputExecCredential(cached.IDToken)
			}
			// Use the cached refresh token (may be newer after prior rotation).
			if cached.RefreshToken != "" {
				refreshToken = cached.RefreshToken
				tokenSource = "cache"
			}
		}
	} else {
		// Locking failed: best-effort fallback without coordination.
		_, _ = fmt.Fprintf(r.stderr, "kubectl-gs: warning: failed to acquire token cache lock: %v\n", lockErr)
		if cached, err := credentialcache.Read(issuerURL, clientID); err == nil && cached.RefreshToken != "" {
			refreshToken = cached.RefreshToken
			tokenSource = "cache"
		}
	}

	_, _ = fmt.Fprintf(r.stderr, "kubectl-gs: renewing OIDC token (issuer: %s, client: %s, refresh token source: %s)\n", issuerURL, clientID, tokenSource)

	idToken, newRefreshToken, err := auther.RenewToken(ctx, refreshToken)
	if err != nil {
		return microerror.Maskf(credentialPluginError, "failed to renew token (issuer: %s, client: %s, source: %s): %v", issuerURL, clientID, tokenSource, err)
	}

	if lockErr == nil {
		if err := credentialcache.WriteLocked(issuerURL, clientID, idToken, newRefreshToken); err != nil {
			return microerror.Maskf(credentialPluginError, "failed to cache token: %v", err)
		}
	} else {
		if err := credentialcache.Write(issuerURL, clientID, idToken, newRefreshToken); err != nil {
			return microerror.Maskf(credentialPluginError, "failed to cache token: %v", err)
		}
	}

	return r.outputExecCredential(idToken)
}

func (r *runner) outputExecCredential(idToken string) error {
	execCredential := execCredentialResponse{
		APIVersion: clientauthv1beta1.SchemeGroupVersion.String(),
		Kind:       "ExecCredential",
		Status: &clientauthv1beta1.ExecCredentialStatus{
			Token: idToken,
		},
	}

	encoder := json.NewEncoder(r.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(execCredential); err != nil {
		return microerror.Maskf(credentialPluginError, "failed to encode ExecCredential: %v", err)
	}
	return nil
}
