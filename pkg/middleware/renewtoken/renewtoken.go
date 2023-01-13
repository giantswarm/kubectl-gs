package renewtoken

import (
	"encoding/base64"
	"encoding/json"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
	"time"

	"github.com/giantswarm/kubectl-gs/v2/pkg/kubeconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v2/pkg/oidc"
)

const (
	refreshTokenKey = "refresh-token"
	idTokenKey      = "id-token"
	idpIssuerUrlKey = "idp-issuer-url"
	clientIdKey     = "client-id"
)

// Middleware will attempt to renew the current context's auth info token.
// If the renewal fails, this middleware will not fail.
func Middleware(config genericclioptions.RESTClientGetter) middleware.Middleware {
	return func(cmd *cobra.Command, args []string) error {
		k8sConfigAccess := config.ToRawKubeConfigLoader().ConfigAccess()
		ctx := cmd.Context()

		config, err := k8sConfigAccess.GetStartingConfig()
		if err != nil {
			return nil
		}

		authProvider, exists := kubeconfig.GetAuthProvider(config, config.CurrentContext)
		if !exists {
			return nil
		}

		if isValidIdToken(authProvider.Config["id-token"]) {
			return nil
		}

		var auther *oidc.Authenticator
		{
			oidcConfig := oidc.Config{
				Issuer:   authProvider.Config[idpIssuerUrlKey],
				ClientID: authProvider.Config[clientIdKey],
			}
			auther, err = oidc.New(ctx, oidcConfig)
			if err != nil {
				return nil
			}
		}

		idToken, rToken, err := auther.RenewToken(ctx, authProvider.Config[refreshTokenKey])
		if err != nil {
			return nil
		}

		// Update the config only in case there are actual changes
		if authProvider.Config[refreshTokenKey] != rToken || authProvider.Config[idTokenKey] != idToken {
			authProvider.Config[refreshTokenKey] = rToken
			authProvider.Config[idTokenKey] = idToken
			_ = clientcmd.ModifyConfig(k8sConfigAccess, *config, true)
		}

		return nil
	}
}

func isValidIdToken(idToken string) bool {
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

	return expTime.After(time.Now())
}
