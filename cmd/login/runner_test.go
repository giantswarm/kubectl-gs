package login

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestLogin(t *testing.T) {
	testCases := []struct {
		name        string
		startConfig *clientcmdapi.Config
		mcArg       string
		expectError *microerror.Error
	}{
		{
			name:        "case 0",
			startConfig: &clientcmdapi.Config{},
			mcArg:       "codename",
			expectError: contextDoesNotExistError,
		},
		{
			name:        "case 1",
			startConfig: &clientcmdapi.Config{},
			mcArg:       "gs-codename",
			expectError: contextDoesNotExistError,
		},
		{
			name:        "case 2",
			startConfig: createValidTestConfig(),
			mcArg:       "gs-codename",
		},
		{
			name:        "case 3",
			startConfig: createValidTestConfig(),
			mcArg:       "codename",
		},
		{
			name:        "case 4",
			startConfig: createValidTestConfig(),
			mcArg:       "othercodename",
			expectError: contextDoesNotExistError,
		},
		{
			name:        "case 5",
			startConfig: createValidTestConfig(),
			mcArg:       "gs-othercodename",
			expectError: contextDoesNotExistError,
		},
		{
			name:        "case 6",
			startConfig: createValidTestConfig(),
			mcArg:       "https://anything.com",
			expectError: unknownUrlError,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := runner{
				k8sConfigAccess: &clientcmd.ClientConfigLoadingRules{
					ExplicitPath: "test/config.yaml",
				},
				stdout: new(bytes.Buffer),
				flag: &flag{
					WCCertTTL: "8h",
				},
			}
			err := clientcmd.ModifyConfig(r.k8sConfigAccess, *tc.startConfig, false)
			if err != nil {
				t.Fatal(err)
			}

			err = r.Run(&cobra.Command{}, []string{tc.mcArg})
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

func createValidTestConfig() *clientcmdapi.Config {
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
