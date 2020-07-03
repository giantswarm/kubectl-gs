package login

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/installation"
)

const (
	flagK8sAPIURL = "k8s-api-url"
)

type flag struct {
	K8sAPIURL string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.K8sAPIURL, flagK8sAPIURL, "", `The installation K8s API URL`)
}

func (f *flag) Validate() error {
	{
		if f.K8sAPIURL == "" {
			return microerror.Maskf(invalidFlagError, "--%s flag must not be empty", flagK8sAPIURL)
		}
		if valid := installation.ValidateK8sAPIUrl(f.K8sAPIURL); !valid {
			return microerror.Maskf(invalidFlagError, "--%s flag must be a valid URL", flagK8sAPIURL)
		}
	}

	return nil
}
