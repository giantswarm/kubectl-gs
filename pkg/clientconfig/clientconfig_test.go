package clientconfig

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/giantswarm/kubectl-gs/test/kubeconfig"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"gotest.tools/v3/assert"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	existingConfigA   = "config_a.yaml"
	existingConfigB   = "config_b.yaml"
	nonExistingConfig = "config_c.yaml"
	SelectedContextA  = "clean"
	SelectedContextB  = "arbitraryname"
	UnselectedContext = "gs-anothercodename"
)

func TestGetClientConfig(t *testing.T) {
	testCases := []struct {
		kubeconfig         string
		kubeconfigOverride string
		contextOverride    string

		expectKubeconfig string
		expectContext    string
		expectError      bool
	}{
		{
			kubeconfig:       existingConfigA,
			expectKubeconfig: existingConfigA,
			expectContext:    SelectedContextA,
		},
		{
			kubeconfig:       existingConfigB,
			expectKubeconfig: existingConfigB,
			expectContext:    SelectedContextB,
		},
		{
			kubeconfig:         existingConfigB,
			kubeconfigOverride: existingConfigA,
			expectKubeconfig:   existingConfigA,
			expectContext:      SelectedContextA,
		},
		{
			kubeconfig:         existingConfigB,
			kubeconfigOverride: nonExistingConfig,
			expectError:        true,
		},
		{
			kubeconfig:       existingConfigA,
			contextOverride:  UnselectedContext,
			expectKubeconfig: existingConfigA,
			expectContext:    SelectedContextA,
		},
		{
			kubeconfig:         existingConfigA,
			kubeconfigOverride: existingConfigB,
			contextOverride:    UnselectedContext,
			expectKubeconfig:   existingConfigB,
			expectContext:      SelectedContextB,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "clientConfigTest")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv(clientcmd.RecommendedConfigPathEnvVar, fmt.Sprintf("%s/%s", configDir, tc.kubeconfig))
			passArguments(configDir, tc.kubeconfigOverride, tc.contextOverride)
			err = writeKubeconfigFiles(configDir)
			if err != nil {
				t.Fatal(err)
			}

			clientConfig, err := GetClientConfig(afero.NewOsFs())
			if tc.expectError && err == nil {
				t.Fatalf("unexpected success")
			} else if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if clientConfig != nil {
				k8sConfigAccess := clientConfig.ConfigAccess()
				f := k8sConfigAccess.GetExplicitFile()
				p := k8sConfigAccess.GetLoadingPrecedence()
				assert.Assert(t, p[0] == fmt.Sprintf("%s/%s", configDir, tc.expectKubeconfig))
				if tc.kubeconfigOverride != "" {
					assert.Assert(t, f == p[0])
				} else {
					assert.Assert(t, f == "")
				}
				if tc.contextOverride != "" {
					config, err := clientConfig.RawConfig()
					if err != nil {
						t.Fatal(err)
					}
					assert.Assert(t, config.CurrentContext == tc.expectContext)
				}
			}
		})
	}
}

func writeKubeconfigFiles(configDir string) error {
	err := clientcmd.WriteToFile(*kubeconfig.AddExtraContext(kubeconfig.CreateValidTestConfig()), fmt.Sprintf("%s/%s", configDir, existingConfigA))
	if err != nil {
		return microerror.Mask(err)
	}
	err = clientcmd.WriteToFile(*kubeconfig.AddExtraContext(kubeconfig.CreateNonDefaultTestConfig()), fmt.Sprintf("%s/%s", configDir, existingConfigB))
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func passArguments(configDir string, kubeconfig string, context string) {
	args := []string{}
	if kubeconfig != "" {
		args = append(args, []string{
			fmt.Sprintf("--%s", flagKubeconfig),
			fmt.Sprintf("%s/%s", configDir, kubeconfig),
		}...)
	}
	if context != "" {
		args = append(args, []string{
			fmt.Sprintf("--%s", flagContext),
			context,
		}...)
	}
	os.Args = args
}
