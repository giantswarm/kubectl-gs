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
		flags       *flag
		expectError *microerror.Error
	}{
		// Empty starting config, logging into MC using codename
		{
			name:        "case 0",
			startConfig: &clientcmdapi.Config{},
			mcArg:       []string{"codename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: contextDoesNotExistError,
		},
		// Empty starting config, logging into MC using context
		{
			name:        "case 1",
			startConfig: &clientcmdapi.Config{},
			mcArg:       []string{"gs-codename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: contextDoesNotExistError,
		},
		// Valid starting config, logging into MC using codename
		{
			name:        "case 2",
			startConfig: createValidTestConfig("", false),
			mcArg:       []string{"gs-codename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		{
			name:        "case 3",
			startConfig: createValidTestConfig("", false),
			mcArg:       []string{"gs-othercodename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: contextDoesNotExistError,
		},
		// Valid starting config, logging into MC using context
		{
			name:        "case 4",
			startConfig: createValidTestConfig("", false),
			flags: &flag{
				WCCertTTL: "8h",
			},
			mcArg: []string{"codename"},
		},
		{
			name:        "case 5",
			startConfig: createValidTestConfig("", false),
			mcArg:       []string{"othercodename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: contextDoesNotExistError,
		},
		// Valid starting config, logging into MC using URL
		{
			name:        "case 6",
			startConfig: createValidTestConfig("", false),
			mcArg:       []string{"https://anything.com"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: unknownUrlError,
		},
		// Valid starting config, Try to reuse existing context
		{
			name:        "case 7",
			startConfig: createValidTestConfig("", false),
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		{
			name:        "case 8",
			startConfig: createValidTestConfig("-cluster", false),
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		// Empty starting config, Try to reuse existing context
		{
			name:        "case 9",
			startConfig: &clientcmdapi.Config{},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: selectedContextNonCompatibleError,
		},
		// Valid starting config with authprovider info, reuse context
		{
			name: "case 10",
			flags: &flag{
				WCCertTTL: "8h",
			},
			startConfig: createValidTestConfig("", true),
		},
		// Valid starting config with authprovider info
		{
			name: "case 10",
			flags: &flag{
				WCCertTTL: "8h",
			},
			mcArg:       []string{"codename"},
			startConfig: createValidTestConfig("", true),
		},
		// Valid starting config with authprovider info, self contained
		{
			name: "case 11",
			flags: &flag{
				WCCertTTL:     "8h",
				SelfContained: "/codename.yaml",
			},
			mcArg:       []string{"codename"},
			startConfig: createValidTestConfig("", true),
			expectError: unknownUrlError,
		},
		// Valid starting config with authprovider info, self contained
		{
			name: "case 11",
			flags: &flag{
				WCCertTTL:     "8h",
				SelfContained: "/codename.yaml",
			},
			mcArg:       []string{"codename"},
			startConfig: createValidTestConfig("", true),
			expectError: unknownUrlError,
		},
		// Logging into non default context
		{
			name:        "case 12",
			startConfig: createNonDefaultTestConfig(),
			mcArg:       []string{"arbitraryname"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: contextDoesNotExistError,
		},
		// Logging into non default context
		{
			name:        "case 13",
			startConfig: createNonDefaultTestConfig(),
			mcArg:       []string{"arbitraryname"},
			flags: &flag{
				WCCertTTL:      "8h",
				EnforceContext: true,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
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
				stdout: new(bytes.Buffer),
				flag:   tc.flags,
				fs:     afero.NewBasePathFs(fs, configDir),
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
		skipInCI    bool

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
			startConfig: createValidTestConfig("", false),
			flags:       &flag{},
			token:       "token",
		},
		// self contained file
		{
			name:        "case 2",
			startConfig: createValidTestConfig("", false),
			flags: &flag{
				SelfContained: "/codename.yaml",
			},
			token: "token",
		},
		// keeping WC context
		{
			name:        "case 3",
			startConfig: createValidTestConfig("-cluster", false),
			flags: &flag{
				KeepContext: true,
			},
			token: "token",
		},
		// OIDC flow
		// This case is skipped in the CI because the oidc function enforces opening a browser window
		// TODO: refactor the function to not enforce that anymore, run this test case in CI
		{
			name: "case 4",
			flags: &flag{
				ClusterAdmin:       false,
				CallbackServerPort: 8080,
			},
			startConfig: &clientcmdapi.Config{},
			skipInCI:    true,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if _, ok := os.LookupEnv("CI"); ok {
				if tc.skipInCI {
					t.Skip()
				}
			}
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

func createValidTestConfig(wcSuffix string, authProvider bool) *clientcmdapi.Config {
	const (
		server       = "https://anything.com:8080"
		token        = "the-token"
		clientid     = "id"
		refreshToken = "the-fresh-token"
	)
	clustername := "codename" + wcSuffix

	config := clientcmdapi.NewConfig()
	config.Clusters["gs-"+clustername] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.Contexts["gs-"+clustername] = &clientcmdapi.Context{
		Cluster:  "gs-" + clustername,
		AuthInfo: "gs-user-" + clustername,
	}
	config.CurrentContext = "gs-" + clustername
	if authProvider {
		config.AuthInfos["gs-user-"+clustername] = &clientcmdapi.AuthInfo{
			AuthProvider: &clientcmdapi.AuthProviderConfig{
				Config: map[string]string{ClientID: clientid,
					Issuer:       server,
					IDToken:      token,
					RefreshToken: refreshToken,
				},
			},
		}
	} else {
		config.AuthInfos["gs-user-"+clustername] = &clientcmdapi.AuthInfo{
			Token: token,
		}
	}

	return config
}

func createNonDefaultTestConfig() *clientcmdapi.Config {
	const (
		server      = "https://anything.com:8080"
		clientkey   = "the-key"
		clientcert  = "the-cert"
		clustername = "arbitraryname"
	)

	config := clientcmdapi.NewConfig()
	config.Clusters[clustername] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.Contexts[clustername] = &clientcmdapi.Context{
		Cluster:  clustername,
		AuthInfo: "user-" + clustername,
	}
	config.CurrentContext = clustername
	config.AuthInfos["user-"+clustername] = &clientcmdapi.AuthInfo{
		ClientCertificateData: []byte(clientcert),
		ClientKeyData:         []byte(clientkey)}
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
