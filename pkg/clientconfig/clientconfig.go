package clientconfig

import (
	"os"
	"path/filepath"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	flagKubeconfig = "kubeconfig"
	flagContext    = "context"
)

type configOverrides struct {
	kubeConfig string
	context    string
}

// GetClientConfig gets ClientConfig from Kubeconfig andd override flags
func GetClientConfig(fs afero.Fs) (clientcmd.ClientConfig, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	configFlags := getConfigOverrideFlags()

	// Override context
	if configFlags.context != "" {
		configOverrides.CurrentContext = configFlags.context
	}

	// No need to override kubeconfig
	if configFlags.kubeConfig == "" {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides), nil
	}

	// Override kubeconfig
	exists, err := afero.Exists(fs, configFlags.kubeConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if exists {
		paths := filepath.SplitList(configFlags.kubeConfig)
		if len(paths) > 1 {
			return nil, microerror.Maskf(invalidConfigError, "Kubeconfig file not found. '%s' looks like a path. \nPlease use the env var KUBECONFIG if you want to check for multiple configuration files", configFlags.kubeConfig)
		}
		loadingRules.ExplicitPath = configFlags.kubeConfig
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides), nil
	}
	return nil, microerror.Maskf(invalidConfigError, "Kubeconfig file '%s' not found", configFlags.kubeConfig)
}

func getConfigOverrideFlags() configOverrides {
	var flags pflag.FlagSet
	var overrides configOverrides
	{
		flags = pflag.FlagSet{}
		configFlags := genericclioptions.NewConfigFlags(true)
		configFlags.AddFlags(&flags)
		flags.Parse(os.Args)
	}
	overrides.context, _ = flags.GetString(flagContext)
	overrides.kubeConfig, _ = flags.GetString(flagKubeconfig)
	return overrides
}
