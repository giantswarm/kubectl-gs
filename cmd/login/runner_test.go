package login

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/giantswarm/kubectl-gs/pkg/installation"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
		name        string
		startConfig *clientcmdapi.Config
		flags       *flag
		expectError *microerror.Error
	}{
		// empty start config
		{
			name:        "case 0",
			startConfig: &clientcmdapi.Config{},
			flags:       &flag{},
		},
		// filled start config
		{
			name:        "case 1",
			startConfig: createValidTestConfigMC(),
			flags:       &flag{},
		},
		// self contained file
		{
			name:        "case 2",
			startConfig: createValidTestConfigMC(),
			flags: &flag{
				SelfContained: "/codename.yaml",
			},
		},
		// keeping WC context
		{
			name:        "case 3",
			startConfig: createValidTestConfigWC(),
			flags: &flag{
				KeepContext: true,
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
				flag:   tc.flags,
				stdout: new(bytes.Buffer),
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

			ctx := context.TODO()
			r.setLoginOptions(ctx, []string{"codename"})

			err = r.loginWithInstallation(ctx, "token", CreateTestInstallation())
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

func CreateTestInstallation() *installation.Installation {
	return &installation.Installation{
		K8sApiURL:         "https://g8s.codename.eu-west-1.aws.gigantic.io",
		K8sInternalApiURL: "https://g8s.codename.internal.eu-west-1.aws.gigantic.io",
		AuthURL:           "https://dex.g8s.codename.eu-west-1.aws.gigantic.io",
		Provider:          "aws",
		Codename:          "codename",
		CACert:            "-----BEGIN CERTIFICATE-----\nsomething\notherthing\nlastthing\n-----END CERTIFICATE-----",
	}
}
