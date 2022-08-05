package clusters

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	configFlags  *genericclioptions.RESTClientGetter
	flag         *flag
	logger       micrologger.Logger
	fs           afero.Fs

	provider string
	service  cluster.Interface

	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	r.commonConfig = commonconfig.New(*r.configFlags)
	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	{
		if r.provider == "" {
			r.provider, err = r.commonConfig.GetProvider()
			if err != nil {
				return microerror.Mask(err)
			}
		}

		err = r.getService(r.commonConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var resource cluster.Resource
	{
		options := cluster.GetOptions{
			Provider: r.provider,
		}
		{
			if len(args) > 0 {
				options.Name = strings.ToLower(args[0])
			}

			if r.flag.AllNamespaces {
				options.Namespace = metav1.NamespaceAll
			} else {
				options.Namespace, _, err = r.commonConfig.ToRawKubeConfigLoader().Namespace()
				if err != nil {
					return microerror.Mask(err)
				}
			}
		}

		resource, err = r.service.Get(ctx, options)
		if cluster.IsNotFound(err) {
			return microerror.Maskf(notFoundError, fmt.Sprintf("A cluster with name '%s' cannot be found.\n", options.Name))
		} else if cluster.IsNoResources(err) && output.IsOutputDefault(r.flag.print.OutputFormat) {
			r.printNoResourcesOutput()

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	err = r.printOutput(resource)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) getService(config *commonconfig.CommonConfig) error {
	if r.service != nil {
		return nil
	}

	client, err := config.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	serviceConfig := cluster.Config{
		Client: client.CtrlClient(),
	}
	r.service = cluster.New(serviceConfig)

	return nil
}
