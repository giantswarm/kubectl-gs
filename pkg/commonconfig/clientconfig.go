package commonconfig

import (
	"os"
	"path/filepath"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// GetClientConfig gets ClientConfig from Kubeconfig andd override flags
func GetClientConfig(fs afero.Fs) (*CommonConfig, error) {
	configFlags := getConfigOverrideFlags()
	kubeconfig := *configFlags.KubeConfig

	if kubeconfig != "" {
		exists, err := afero.Exists(fs, kubeconfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if exists {
			paths := filepath.SplitList(kubeconfig)
			if len(paths) > 1 {
				return nil, microerror.Maskf(invalidConfigError, "Kubeconfig file not found. '%s' looks like a path. \nPlease use the env var KUBECONFIG if you want to check for multiple configuration files", kubeconfig)
			}
		} else {
			return nil, microerror.Maskf(invalidConfigError, "Kubeconfig file '%s' not found", kubeconfig)
		}
	}
	return New(configFlags), nil
}

func getConfigOverrideFlags() *genericclioptions.ConfigFlags {
	var flags pflag.FlagSet
	var config *genericclioptions.ConfigFlags
	{
		flags = pflag.FlagSet{}
		configFlags := genericclioptions.NewConfigFlags(true)
		config = configFlags
		configFlags.AddFlags(&flags)
		flags.Parse(os.Args)
	}
	return config
}
