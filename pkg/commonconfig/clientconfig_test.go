package commonconfig

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"gotest.tools/v3/assert"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/test/kubeconfig"
)

const (
	flagKubeconfig = "kubeconfig"
	flagContext    = "context"
)

const (
	existingConfigA   = "config_a.yaml"
	existingConfigB   = "config_b.yaml"
	nonExistingConfig = "config_c.yaml"
	selectedContextA  = "clean"
	selectedContextB  = "arbitraryname"
	unselectedContext = "gs-anothercodename"
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
			expectContext:    selectedContextA,
		},
		{
			kubeconfig:       existingConfigB,
			expectKubeconfig: existingConfigB,
			expectContext:    selectedContextB,
		},
		{
			kubeconfig:         existingConfigB,
			kubeconfigOverride: existingConfigA,
			expectKubeconfig:   existingConfigA,
			expectContext:      selectedContextA,
		},
		{
			kubeconfig:         existingConfigB,
			kubeconfigOverride: nonExistingConfig,
			expectError:        true,
		},
		{
			kubeconfig:       existingConfigA,
			contextOverride:  unselectedContext,
			expectKubeconfig: existingConfigA,
			expectContext:    unselectedContext,
		},
		{
			kubeconfig:         existingConfigA,
			kubeconfigOverride: existingConfigB,
			contextOverride:    unselectedContext,
			expectKubeconfig:   existingConfigB,
			expectContext:      unselectedContext,
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

			commonconfig, err := GetClientConfig(afero.NewOsFs())
			if tc.expectError && err == nil {
				t.Fatalf("unexpected success")
			} else if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if commonconfig != nil {
				k8sConfigAccess := commonconfig.ToRawKubeConfigLoader()
				f := k8sConfigAccess.ConfigAccess().GetExplicitFile()
				p := k8sConfigAccess.ConfigAccess().GetLoadingPrecedence()
				assert.Assert(t, p[0] == fmt.Sprintf("%s/%s", configDir, tc.expectKubeconfig))
				if tc.kubeconfigOverride != "" {
					assert.Assert(t, f == p[0])
				} else {
					assert.Assert(t, f == "")
				}
				if tc.contextOverride != "" {
					configflag, _ := commonconfig.GetConfigFlags()
					if err != nil {
						t.Fatal(err)
					}
					assert.Assert(t, *configflag.Context == tc.expectContext)
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
