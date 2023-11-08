package renewtoken

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"

	"github.com/giantswarm/kubectl-gs/v2/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v2/test/kubeconfig"
)

const (
	Issuer       = "idp-issuer-url"
	IDToken      = "id-token"
	RefreshToken = "refresh-token"
)

func Test_RenewTokenMiddleware(t *testing.T) {

	testCases := []struct {
		name           string
		idTokenCreator func(issuer string, key *rsa.PrivateKey) (string, error)
		renewToken     string
		expectRenew    bool
	}{
		{
			name: "case 0: Valid up-to-date id token should not be renewed",
			idTokenCreator: func(issuer string, key *rsa.PrivateKey) (string, error) {
				return createTestIdToken(issuer, time.Now().Add(15*time.Minute), key)
			},
			renewToken:  createTestRefreshToken(),
			expectRenew: false,
		},
		{
			name: "case 1: Valid outdated id token should be renewed",
			idTokenCreator: func(issuer string, key *rsa.PrivateKey) (string, error) {
				return createTestIdToken(issuer, time.Now().Add(-15*time.Minute), key)
			},
			renewToken:  createTestRefreshToken(),
			expectRenew: true,
		},
		{
			name: "case 2: Invalid id token should not be renewed",
			idTokenCreator: func(issuer string, key *rsa.PrivateKey) (string, error) {
				return "", nil
			},
			expectRenew: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "renewTokenTest")
			if err != nil {
				t.Fatal(err)
			}

			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}

			authServer := mockAuthServer(key)
			defer authServer.Close()

			idToken, err := tc.idTokenCreator(authServer.URL, key)
			if err != nil {
				t.Fatal(err)
			}

			cf := genericclioptions.NewConfigFlags(true)
			cf.KubeConfig = ptr.To[string](fmt.Sprintf("%s/config.yaml", configDir))
			k8sConfigAccess := cf.ToRawKubeConfigLoader().ConfigAccess()
			err = clientcmd.ModifyConfig(k8sConfigAccess, *createValidTestConfig(authServer.URL, idToken, tc.renewToken), false)
			if err != nil {
				t.Fatal(err)
			}

			cmd := &cobra.Command{
				PreRunE: middleware.Compose(
					Middleware(cf),
				),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}

			err = cmd.ExecuteContext(context.TODO())
			if err != nil {
				t.Fatal(err)
			}

			newCf := genericclioptions.NewConfigFlags(true)
			newCf.KubeConfig = ptr.To[string](fmt.Sprintf("%s/config.yaml", configDir))
			newk8sConfigAccess := newCf.ToRawKubeConfigLoader().ConfigAccess()

			newConfig, err := newk8sConfigAccess.GetStartingConfig()
			if err != nil {
				t.Fatal(err)
			}

			if newConfig == nil {
				t.Fatal("New config should not be nil")
				return
			}

			newAuthInfo, ok := newConfig.AuthInfos["clean"]
			if !ok {
				t.Fatal("New AuthInfo is missing")
			}

			newIdToken, ok := newAuthInfo.AuthProvider.Config[idTokenKey]
			if !ok {
				t.Fatal("New token is missing")
			}

			if idToken == newIdToken && tc.expectRenew {
				t.Fatal("Token not renewed. Expected a new ID token, got the same one")
			} else if idToken != newIdToken && !tc.expectRenew {
				t.Fatalf("Unexpected token renewal. Expected %s, got %s", idToken, newIdToken)
			}
		})
	}

}

func mockAuthServer(key *rsa.PrivateKey) *httptest.Server {
	var issuer string
	hf := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, getIssuerData(issuer))
		case "/token":
			token, err := getToken(issuer, key)
			if err != nil {
				w.WriteHeader(500)
				_, _ = w.Write([]byte{})
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, token)
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte{})
		}
	}

	s := httptest.NewServer(http.HandlerFunc(hf))
	issuer = s.URL
	return s
}

func createValidTestConfig(issuer string, idToken string, refreshToken string) *clientcmdapi.Config {
	config := kubeconfig.CreateValidTestConfig()
	config.AuthInfos[config.CurrentContext] = &clientcmdapi.AuthInfo{
		AuthProvider: &clientcmdapi.AuthProviderConfig{
			Config: map[string]string{
				Issuer:       issuer,
				IDToken:      idToken,
				RefreshToken: refreshToken,
			},
		},
	}
	return config
}

func createTestIdToken(issuer string, expiration time.Time, key *rsa.PrivateKey) (string, error) {

	header := map[string]string{
		"alg": "RS256",
		"kid": "key-id",
	}

	headerJson, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"iss":                issuer,
		"sub":                "subject",
		"aud":                []string{"dex-k8s-authenticator", "authorized-party-id"},
		"exp":                expiration.Unix(),
		"iat":                expiration.Add(-15 * time.Minute),
		"azp":                "authorized-party-id",
		"at_hash":            "access-token-hash",
		"c_hash":             "authorization-code-hash",
		"email":              "user@email.com",
		"email_verified":     true,
		"groups":             []string{"access:group:1"},
		"name":               "user",
		"prederred_username": "user",
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerBase64 := base64.RawURLEncoding.EncodeToString(headerJson)
	payloadBase64 := base64.RawURLEncoding.EncodeToString(payloadJson)

	content := []byte(fmt.Sprintf("%s.%s", headerBase64, payloadBase64))

	contentHash := sha512.New()
	_, err = contentHash.Write(content)
	if err != nil {
		return "", err
	}
	contentHashSum := contentHash.Sum(nil)

	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA512, contentHashSum, nil)
	if err != nil {
		return "", err
	}

	signatureBase64 := base64.RawURLEncoding.EncodeToString(signature)

	return fmt.Sprintf("%s.%s", content, signatureBase64), nil
}

func createTestRefreshToken() string {
	return "refresh-token"
}

func getIssuerData(issuer string) string {
	return fmt.Sprintf(`{
		"issuer": "%[1]s",
		"authorization_endpoint": "%[1]s/auth",
		"token_endpoint": "%[1]s/token",
		"jwks_uri": "%[1]s/keys",
		"userinfo_endpoint": "%[1]s/userinfo",
		"response_types_supported": [
		  "code"
		],
		"subject_types_supported": [
		  "public"
		],
		"id_token_signing_alg_values_supported": [
		  "RS256"
		],
		"scopes_supported": [
		  "openid",
		  "email",
		  "groups",
		  "profile",
		  "offline_access"
		],
		"token_endpoint_auth_methods_supported": [
		  "client_secret_basic"
		],
		"claims_supported": [
		  "aud",
		  "email",
		  "email_verified",
		  "exp",
		  "iat",
		  "iss",
		  "locale",
		  "name",
		  "sub"
		]
	}`, issuer)

}

func getToken(issuer string, key *rsa.PrivateKey) (string, error) {
	idToken, err := createTestIdToken(issuer, time.Now(), key)

	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("id_token", idToken)
	params.Add("access_token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9")
	params.Add("refresh_token", createTestRefreshToken())
	return params.Encode(), nil
}
