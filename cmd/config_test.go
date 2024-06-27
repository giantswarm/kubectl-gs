package cmd

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/v4/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v4/test/kubeconfig"
)

const (
	flagKubeconfig = "kubeconfig"
	flagContext    = "context"
)

const (
	existingConfigA   = "config_a.yaml"
	existingConfigB   = "config_b.yaml"
	selectedContextA  = "clean"
	selectedContextB  = "arbitraryname"
	unselectedContext = "gs-anothercodename"
)

type testCommand struct {
	commonConfig *commonconfig.CommonConfig
	configFlags  *genericclioptions.RESTClientGetter
}

func TestGetCommonConfig(t *testing.T) {
	testCases := []struct {
		kubeconfig         string
		kubeconfigOverride string
		contextOverride    string

		expectKubeconfig string
		expectContext    string
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

			f := &flag{}
			c := &cobra.Command{}
			f.Init(c)
			testCommand := testCommand{
				configFlags: &f.config,
			}
			err = c.PersistentFlags().Parse(os.Args)
			if err != nil {
				t.Fatal(err)
			}
			testCommand.getCommonConfig()

			if testCommand.commonConfig != nil {
				k8sConfigAccess := testCommand.commonConfig.GetConfigFlags().ToRawKubeConfigLoader()
				f := k8sConfigAccess.ConfigAccess().GetExplicitFile()
				p := k8sConfigAccess.ConfigAccess().GetLoadingPrecedence()
				assert.Assert(t, p[0] == fmt.Sprintf("%s/%s", configDir, tc.expectKubeconfig))
				if tc.kubeconfigOverride != "" {
					assert.Assert(t, f == p[0])
				} else {
					assert.Assert(t, f == "")
				}
				if tc.contextOverride != "" {
					assert.Assert(t, testCommand.commonConfig.GetContextOverride() == tc.expectContext)
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

func (t *testCommand) getCommonConfig() {
	t.commonConfig = commonconfig.New(*t.configFlags)
}
