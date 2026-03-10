package credentialplugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	clientauthv1beta1 "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"

	"github.com/giantswarm/kubectl-gs/v6/pkg/credentialcache"
	"github.com/giantswarm/kubectl-gs/v6/pkg/oidc"
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
	if idTokenFromEnv := os.Getenv(envIDToken); isValidIdToken(idTokenFromEnv) {
		return r.outputExecCredential(idTokenFromEnv)
	}

	// Check if we have a cached valid token before renewing
	cached, err := credentialcache.Read(issuerURL, clientID)
	if err == nil && isValidIdToken(cached.IDToken) {
		return r.outputExecCredential(cached.IDToken)
	}

	// Prefer cached refresh token over env var (it may be newer after rotation)
	tokenSource := "kubeconfig"
	if cached.RefreshToken != "" {
		refreshToken = cached.RefreshToken
		tokenSource = "cache"
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
			return microerror.Maskf(credentialPluginError, "failed to create OIDC authenticator for %s: %v", issuerURL, err)
		}
	}

	_, _ = fmt.Fprintf(r.stderr, "kubectl-gs: renewing OIDC token (issuer: %s, client: %s, refresh token source: %s)\n", issuerURL, clientID, tokenSource)

	idToken, newRefreshToken, err := auther.RenewToken(ctx, refreshToken)
	if err != nil {
		return microerror.Maskf(credentialPluginError, "failed to renew token (issuer: %s, client: %s, source: %s): %v", issuerURL, clientID, tokenSource, err)
	}

	if err := credentialcache.Write(issuerURL, clientID, idToken, newRefreshToken); err != nil {
		return microerror.Maskf(credentialPluginError, "failed to cache token: %v", err)
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
