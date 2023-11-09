package login

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"k8s.io/utils/ptr"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/installation"
	"github.com/giantswarm/kubectl-gs/v2/test/kubeconfig"
	testoidc "github.com/giantswarm/kubectl-gs/v2/test/oidc"
)

func TestLogin(t *testing.T) {
	testCases := []struct {
		name               string
		startConfig        *clientcmdapi.Config
		startConfigCreator func(issuer string) *clientcmdapi.Config
		mcArg              []string
		flags              *flag
		stdin              string
		contextOverride    string
		expectError        *microerror.Error
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
		{
			name:  "case 29",
			flags: &flag{},
			mcArg: []string{"gs-codename"},
			stdin: "\n",
			startConfigCreator: func(issuer string) *clientcmdapi.Config {
				return kubeconfig.CreateProviderTestConfig(
					"codename",
					"gs-codename",
					"gs-codename-user-device",
					"https://anything.com:8080",
					clientID,
					"id-token",
					issuer,
					"refresh-token")
			},
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

			var s *testoidc.MockOidcServer

			startConfig := tc.startConfig
			if startConfig == nil && tc.startConfigCreator != nil {
				config := testoidc.MockOidcServerConfig{
					ClientID:             clientID,
					InstallationCodename: "codename",
					TokenFailures:        3,
				}
				s = testoidc.NewServer(config)
				err = s.Start(t)
				if err != nil {
					t.Fatal(err)
				}
				startConfig = tc.startConfigCreator(s.Issuer())
			}

			defer func() {
				if s != nil {
					s.Stop()
				}
			}()

			var in *os.File
			if tc.stdin != "" {
				in, err = os.CreateTemp("", "stdin")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = in.WriteString(tc.stdin); err != nil {
					t.Fatal(err)
				}
				if _, err = in.Seek(0, 0); err != nil {
					t.Fatal(err)
				}
			}

			defer func() {
				if in != nil {
					_ = os.Remove(in.Name())
				}
			}()

			r := runner{
				commonConfig: commonconfig.New(cf),
				stdout:       new(bytes.Buffer),
				stdin:        in,
				flag:         tc.flags,
				fs:           afero.NewBasePathFs(fs, configDir),
			}
			k8sConfigAccess := r.commonConfig.GetConfigAccess()
			err = clientcmd.ModifyConfig(k8sConfigAccess, *startConfig, false)
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

		input string

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
				LoginTimeout:       500 * time.Microsecond,
			},
			startConfig:    &clientcmdapi.Config{},
			expectError:    authResponseTimedOutError,
			expectedOutput: "\nYour authentication flow timed out after 500µs. Please execute the same command again.\nYou can use the --login-timeout flag to configure a longer timeout interval, for example --login-timeout=0s.\n",
		},
		// Device auth flow
		{
			name: "case 6",
			flags: &flag{
				ClusterAdmin: false,
				DeviceAuth:   true,
				LoginTimeout: 60 * time.Second,
			},
			input:          "\n",
			startConfig:    &clientcmdapi.Config{},
			expectedOutput: "Press ENTER when ready to continue\nLogged in successfully as '-device' on cluster 'codename'.\n\n",
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

			var in *os.File
			if tc.input != "" {
				in, err = os.CreateTemp("", "stdin")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = in.WriteString(tc.input); err != nil {
					t.Fatal(err)
				}
				if _, err = in.Seek(0, 0); err != nil {
					t.Fatal(err)
				}
			}

			defer func() {
				if in != nil {
					_ = os.Remove(in.Name())
				}
			}()

			r := runner{
				commonConfig: commonconfig.New(cf),
				flag:         tc.flags,
				stdout:       out,
				stderr:       out,
				stdin:        in,
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

			s := testoidc.NewServer(testoidc.MockOidcServerConfig{
				ClientID: clientID,
			})
			err = s.Start(t)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Stop()

			r.setLoginOptions(ctx, &[]string{"codename"})

			err = r.loginWithInstallation(ctx, tc.token, CreateTestInstallationWithIssuer(s.Issuer()))

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

	clusterKey := "gs-" + clustername
	userKey := "gs-user-" + clustername
	config := clientcmdapi.NewConfig()
	config.Clusters[clusterKey] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.Contexts[clusterKey] = &clientcmdapi.Context{
		Cluster:  clusterKey,
		AuthInfo: userKey,
	}
	config.CurrentContext = clusterKey
	if authProvider {
		config.AuthInfos[userKey] = &clientcmdapi.AuthInfo{
			AuthProvider: &clientcmdapi.AuthProviderConfig{
				Config: map[string]string{ClientID: clientid,
					Issuer:       server,
					IDToken:      token,
					RefreshToken: refreshToken,
				},
			},
		}
	} else {
		config.AuthInfos[userKey] = &clientcmdapi.AuthInfo{
			Token: token,
		}
	}

	return config
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
