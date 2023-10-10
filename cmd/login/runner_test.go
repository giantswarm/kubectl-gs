package login

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"k8s.io/utils/ptr"

	"github.com/giantswarm/microerror"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/installation"
	"github.com/giantswarm/kubectl-gs/v2/test/kubeconfig"
)

func TestLogin(t *testing.T) {
	testCases := []struct {
		name            string
		startConfig     *clientcmdapi.Config
		mcArg           []string
		flags           *flag
		contextOverride string
		expectError     *microerror.Error
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
			name: "case 11",
			flags: &flag{
				WCCertTTL: "8h",
			},
			mcArg:       []string{"codename"},
			startConfig: createValidTestConfig("", true),
		},
		// Valid starting config with authprovider info, self contained
		{
			name: "case 12",
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
			name: "case 13",
			flags: &flag{
				WCCertTTL:     "8h",
				SelfContained: "/codename.yaml",
			},
			mcArg:       []string{"codename"},
			startConfig: createValidTestConfig("", true),
			expectError: unknownUrlError,
		},
		// Logging into non default context with argument
		{
			name:        "case 14",
			startConfig: kubeconfig.CreateNonDefaultTestConfig(),
			mcArg:       []string{"arbitraryname"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: contextDoesNotExistError,
		},
		// Logging into non default context without argument
		{
			name:        "case 15",
			startConfig: kubeconfig.CreateNonDefaultTestConfig(),
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		// Logging in without argument
		{
			name:        "case 16",
			startConfig: createValidTestConfig("", false),
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		// Logging in without argument using context flag
		{
			name:        "case 17",
			startConfig: kubeconfig.AddExtraContext(createValidTestConfig("", false)),
			flags: &flag{
				WCCertTTL: "8h",
			},
			contextOverride: *ptr.To[string]("gs-anothercodename"),
		},
		// Logging in without argument using context flag but context does not exist
		{
			name:        "case 18",
			startConfig: createValidTestConfig("", false),
			flags: &flag{
				WCCertTTL: "8h",
			},
			contextOverride: *ptr.To[string]("gs-anothercodename"),
			expectError:     contextDoesNotExistError,
		},
		// Logging in with argument using context flag
		{
			name:        "case 19",
			startConfig: createValidTestConfig("", false),
			mcArg:       []string{"codename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			contextOverride: *ptr.To[string]("gs-anothercodename"),
		},
		// Existing WC context
		{
			name:        "case 20",
			startConfig: createValidTestConfig("-cluster", false),
			mcArg:       []string{"codename", "cluster"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		{
			name:        "case 21",
			startConfig: createValidTestConfig("-cluster", false),
			mcArg:       []string{"gs-codename-cluster"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		// Too many arguments
		{
			name:        "case 22",
			startConfig: createValidTestConfig("-cluster", false),
			mcArg:       []string{"codename", "cluster", "somethingelse"},
			flags: &flag{
				WCCertTTL: "8h",
			},
			expectError: invalidConfigError,
		},
		// Existing WC clientcert context
		{
			name:        "case 23",
			startConfig: createValidTestConfig("-cluster-clientcert", false),
			mcArg:       []string{"gs-codename-cluster-clientcert"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		{
			name:        "case 24",
			startConfig: createValidTestConfig("-cluster-clientcert", false),
			mcArg:       []string{"codename", "cluster"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		// Existing MC clientcert context
		{
			name:        "case 25",
			startConfig: createValidTestConfig("-clientcert", false),
			mcArg:       []string{"gs-codename-clientcert"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		{
			name:        "case 26",
			startConfig: createValidTestConfig("-clientcert", false),
			mcArg:       []string{"codename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		{
			name:        "case 27",
			startConfig: createValidTestConfig("-clientcert", false),
			mcArg:       []string{"gs-codename"},
			flags: &flag{
				WCCertTTL: "8h",
			},
		},
		// Logging in without argument using context flag without gs- prefix
		{
			name:        "case 28",
			startConfig: kubeconfig.CreateNonDefaultTestConfig(),
			flags: &flag{
				WCCertTTL: "8h",
			},
			contextOverride: *ptr.To[string]("arbitraryname"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "loginTest")
			if err != nil {
				t.Fatal(err)
			}
			cf := genericclioptions.NewConfigFlags(true)
			cf.KubeConfig = ptr.To[string](fmt.Sprintf("%s/config.yaml", configDir))
			if tc.contextOverride != "" {
				cf.Context = &tc.contextOverride
			}
			fs := afero.NewOsFs()
			if len(tc.flags.SelfContained) > 0 {
				tc.flags.SelfContained = configDir + tc.flags.SelfContained
			}
			r := runner{
				commonConfig: commonconfig.New(cf),
				stdout:       new(bytes.Buffer),
				flag:         tc.flags,
				fs:           afero.NewBasePathFs(fs, configDir),
			}
			k8sConfigAccess := r.commonConfig.GetConfigAccess()
			err = clientcmd.ModifyConfig(k8sConfigAccess, *tc.startConfig, false)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			r.setLoginOptions(ctx, &tc.mcArg)
			err = r.run(ctx, &cobra.Command{}, tc.mcArg)
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

		expectError    *microerror.Error
		expectedOutput string
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
				LoginTimeout:       60 * time.Second,
			},
			startConfig: &clientcmdapi.Config{},
			skipInCI:    true,
		},
		// Fail in case the OIDC flow does not finish within time period specified in the flag
		{
			name: "case 5",
			flags: &flag{
				ClusterAdmin:       false,
				CallbackServerPort: 8080,
				LoginTimeout:       1 * time.Millisecond,
			},
			startConfig:    &clientcmdapi.Config{},
			expectError:    authResponseTimedOutError,
			expectedOutput: "\nYour authentication flow timed out after 1ms. Please execute the same command again.\nYou can use the --login-timeout flag to configure a longer timeout interval, for example --login-timeout=0s.\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			cf := genericclioptions.NewConfigFlags(true)
			cf.KubeConfig = ptr.To[string](fmt.Sprintf("%s/config.yaml", configDir))
			fs := afero.NewOsFs()
			if len(tc.flags.SelfContained) > 0 {
				tc.flags.SelfContained = configDir + tc.flags.SelfContained
			}

			out := new(bytes.Buffer)

			r := runner{
				commonConfig: commonconfig.New(cf),
				flag:         tc.flags,
				stdout:       out,
				stderr:       out,
				fs:           afero.NewBasePathFs(fs, configDir),
			}
			k8sConfigAccess := r.commonConfig.GetConfigAccess()
			err = clientcmd.ModifyConfig(k8sConfigAccess, *tc.startConfig, false)
			if err != nil {
				t.Fatal(err)
			}
			originConfig, err := k8sConfigAccess.GetStartingConfig()
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
					token, err := generateAndGetToken(issuer, key)
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

			r.setLoginOptions(ctx, &[]string{"codename"})

			err = r.loginWithInstallation(ctx, tc.token, CreateTestInstallationWithIssuer(issuer))

			if err != nil {
				if microerror.Cause(err) != tc.expectError {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			} else if tc.expectError != nil {
				t.Fatalf("unexpected success")
			}

			targetConfig, err := k8sConfigAccess.GetStartingConfig()
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

			if tc.expectedOutput != "" {
				outStr := out.String()
				if !strings.Contains(outStr, tc.expectedOutput) {
					t.Fatalf("output does not contain expected string:\nvalue: %s\nexpected string: %s\n", outStr, tc.expectedOutput)
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

func generateAndGetToken(issuer string, key *rsa.PrivateKey) (string, error) {
	idToken, err := getRawToken(issuer, key)
	if err != nil {
		return "", err
	}
	return getToken(idToken, ""), nil
}

func getToken(idToken string, refreshToken string) string {
	params := url.Values{}
	params.Add("id_token", idToken)
	params.Add("access_token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9")
	if refreshToken != "" {
		params.Add("refresh_token", refreshToken)
	}
	return params.Encode()
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
