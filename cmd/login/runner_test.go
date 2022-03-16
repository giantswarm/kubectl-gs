package login

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/kubectl-gs/pkg/installation"
)

func TestLogin(t *testing.T) {
	testCases := []struct {
		name        string
		startConfig *clientcmdapi.Config
		mcArg       []string
		expectError *microerror.Error
	}{
		// Empty starting config, logging into MC using codename
		{
			name:        "case 0",
			startConfig: &clientcmdapi.Config{},
			mcArg:       []string{"codename"},
			expectError: contextDoesNotExistError,
		},
		// Empty starting config, logging into MC using context
		{
			name:        "case 1",
			startConfig: &clientcmdapi.Config{},
			mcArg:       []string{"gs-codename"},
			expectError: contextDoesNotExistError,
		},
		// Valid starting config, logging into MC using codename
		{
			name:        "case 2",
			startConfig: createValidTestConfigMC(),
			mcArg:       []string{"gs-codename"},
		},
		{
			name:        "case 3",
			startConfig: createValidTestConfigMC(),
			mcArg:       []string{"gs-othercodename"},
			expectError: contextDoesNotExistError,
		},
		// Valid starting config, logging into MC using context
		{
			name:        "case 4",
			startConfig: createValidTestConfigMC(),
			mcArg:       []string{"codename"},
		},
		{
			name:        "case 5",
			startConfig: createValidTestConfigMC(),
			mcArg:       []string{"othercodename"},
			expectError: contextDoesNotExistError,
		},
		// Valid starting config, logging into MC using URL
		{
			name:        "case 6",
			startConfig: createValidTestConfigMC(),
			mcArg:       []string{"https://anything.com"},
			expectError: unknownUrlError,
		},
		// Valid starting config, Try to reuse existing context
		{
			name:        "case 7",
			startConfig: createValidTestConfigMC(),
		},
		{
			name:        "case 8",
			startConfig: createValidTestConfigWC(),
		},
		// Empty starting config, Try to reuse existing context
		{
			name:        "case 9",
			startConfig: &clientcmdapi.Config{},
			expectError: selectedContextNonCompatibleError,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "loginTest")
			if err != nil {
				t.Fatal(err)
			}

			r := runner{
				k8sConfigAccess: &clientcmd.ClientConfigLoadingRules{
					ExplicitPath: configDir + "/config.yaml",
				},
				stdout: new(bytes.Buffer),
				flag: &flag{
					WCCertTTL: "8h",
				},
			}
			err = clientcmd.ModifyConfig(r.k8sConfigAccess, *tc.startConfig, false)
			if err != nil {
				t.Fatal(err)
			}

			err = r.Run(&cobra.Command{}, tc.mcArg)
			if err != nil {
				if microerror.Cause(err) != tc.expectError {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			} else if tc.expectError != nil {
				t.Fatalf("unexpected success")
			}
		})
	}
}

func TestMCLoginWithInstallation(t *testing.T) {
	testCases := []struct {
		name string

		startConfig *clientcmdapi.Config
		flags       *flag
		token       string

		expectError *microerror.Error
	}{
		// empty start config
		{
			name:        "case 0",
			startConfig: &clientcmdapi.Config{},
			flags:       &flag{},
			token:       "token",
		},
		// filled start config
		{
			name:        "case 1",
			startConfig: createValidTestConfigMC(),
			flags:       &flag{},
			token:       "token",
		},
		// self contained file
		{
			name:        "case 2",
			startConfig: createValidTestConfigMC(),
			flags: &flag{
				SelfContained: "/codename.yaml",
			},
			token: "token",
		},
		// keeping WC context
		{
			name:        "case 3",
			startConfig: createValidTestConfigWC(),
			flags: &flag{
				KeepContext: true,
			},
			token: "token",
		},
		// OIDC flow
		{
			name: "case 4",
			flags: &flag{
				ClusterAdmin:       false,
				CallbackServerPort: 8080,
			},
			startConfig: &clientcmdapi.Config{},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var err error

			configDir, err := os.MkdirTemp("", "loginTest")
			if err != nil {
				t.Fatal(err)
			}
			fs := afero.NewOsFs()
			if len(tc.flags.SelfContained) > 0 {
				tc.flags.SelfContained = configDir + tc.flags.SelfContained
			}

			r := runner{
				k8sConfigAccess: &clientcmd.ClientConfigLoadingRules{
					ExplicitPath: configDir + "/config.yaml",
				},
				flag:   tc.flags,
				stdout: new(bytes.Buffer),
				stderr: new(bytes.Buffer),
				fs:     afero.NewBasePathFs(fs, configDir),
			}
			err = clientcmd.ModifyConfig(r.k8sConfigAccess, *tc.startConfig, false)
			if err != nil {
				t.Fatal(err)
			}
			originConfig, err := r.k8sConfigAccess.GetStartingConfig()
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// generate a key
			key, err := getKey()
			if err != nil {
				t.Fatal(err)
			}

			// mock the OIDC issuer
			var issuer string
			hf := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/auth" {
					http.Redirect(w, r, "http://localhost:8080/oauth/callback?"+r.URL.RawQuery+"&code=codename", http.StatusFound)
				} else if r.URL.Path == "/token" {
					token, err := getToken(issuer, key)
					if err != nil {
						t.Fatal(err)
					}
					w.Header().Set("Content-Type", "text/plain")
					_, err = io.WriteString(w, token)
					if err != nil {
						t.Fatal(err)
					}
				} else if r.URL.Path == "/keys" {
					webKey, err := getJSONWebKey(key)
					if err != nil {
						t.Fatal(err)
					}
					w.Header().Set("Content-Type", "application/json")
					_, err = io.WriteString(w, webKey)
					if err != nil {
						t.Fatal(err)
					}
				} else {
					w.Header().Set("Content-Type", "application/json")
					_, err = io.WriteString(w, strings.ReplaceAll(getIssuerData(), "ISSUER", issuer))
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			s := httptest.NewServer(http.HandlerFunc(hf))
			defer s.Close()

			issuer = s.URL

			r.setLoginOptions(ctx, []string{"codename"})

			err = r.loginWithInstallation(ctx, tc.token, CreateTestInstallationWithIssuer(issuer))

			if err != nil {
				if microerror.Cause(err) != tc.expectError {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			} else if tc.expectError != nil {
				t.Fatalf("unexpected success")
			}

			targetConfig, err := r.k8sConfigAccess.GetStartingConfig()
			if err != nil {
				t.Fatal(err)
			}
			if tc.flags.KeepContext && targetConfig.CurrentContext != originConfig.CurrentContext {
				t.Fatalf("expected to keep context %s, got context %s", originConfig.CurrentContext, targetConfig.CurrentContext)
			}
			if len(tc.flags.SelfContained) > 0 {
				if _, err := os.Stat(configDir + "/codename.yaml"); err != nil {
					t.Fatalf("expected self-contained config file: %s", err)
				}
				if !reflect.DeepEqual(targetConfig, originConfig) {
					t.Fatal("expected origin config to not be modified.")
				}
			}

		})
	}
}

func createValidTestConfigWC() *clientcmdapi.Config {
	const (
		server = "https://anything.com:8080"
		token  = "the-token"
	)

	config := clientcmdapi.NewConfig()
	config.Clusters["gs-codename-cluster"] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.AuthInfos["gs-codename-cluster-user"] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	config.Contexts["gs-codename-cluster"] = &clientcmdapi.Context{
		Cluster:  "gs-codename-cluster",
		AuthInfo: "gs-codename-cluster-user",
	}
	config.CurrentContext = "gs-codename-cluster"

	return config
}

func createValidTestConfigMC() *clientcmdapi.Config {
	const (
		server = "https://anything.com:8080"
		token  = "the-token"
	)

	config := clientcmdapi.NewConfig()
	config.Clusters["gs-codename"] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.AuthInfos["gs-user-codename"] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	config.Contexts["gs-codename"] = &clientcmdapi.Context{
		Cluster:  "gs-codename",
		AuthInfo: "gs-user-codename",
	}
	config.CurrentContext = "gs-codename"

	return config
}

func getToken(issuer string, key *rsa.PrivateKey) (string, error) {
	idToken, err := getRawToken(issuer, key)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Add("id_token", idToken)
	params.Add("access_token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9")
	return params.Encode(), nil
}

func CreateTestInstallationWithIssuer(issuer string) *installation.Installation {
	return &installation.Installation{
		K8sApiURL:         "https://g8s.codename.eu-west-1.aws.gigantic.io",
		K8sInternalApiURL: "https://g8s.codename.internal.eu-west-1.aws.gigantic.io",
		AuthURL:           issuer,
		Provider:          "aws",
		Codename:          "codename",
		CACert:            "-----BEGIN CERTIFICATE-----\nsomething\notherthing\nlastthing\n-----END CERTIFICATE-----",
	}
}

func getRawToken(issuer string, key *rsa.PrivateKey) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	claims := make(jwt.MapClaims)
	claims["iss"] = issuer
	claims["aud"] = clientID
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	token.Claims = claims
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func getIssuerData() string {
	return `{
		"issuer": "ISSUER",
		"authorization_endpoint": "ISSUER/auth",
		"token_endpoint": "ISSUER/token",
		"jwks_uri": "ISSUER/keys",
		"userinfo_endpoint": "ISSUER/userinfo",
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
	}`
}

func getKey() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func getJSONWebKey(key *rsa.PrivateKey) (string, error) {
	jwk := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       &key.PublicKey,
				Algorithm: "RS256",
			},
		}}
	json, err := json.Marshal(jwk)
	if err != nil {
		return "", err
	}
	return string(json), nil
}
