package orgs

import (
	"context"
	"io"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger
	fs           afero.Fs

	service organization.Interface

	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	{
		err = r.getService()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var resource organization.Resource
	{
		options := organization.GetOptions{}
		{
			if len(args) > 0 {
				options.Name = strings.ToLower(args[0])
			}
		}

		resource, err = r.service.Get(ctx, options)
		if organization.IsNotFound(err) {
			return microerror.Maskf(notFoundError, "An organization with name '%s' cannot be found.\n", options.Name)
		} else if organization.IsNoResources(err) && output.IsOutputDefault(r.flag.print.OutputFormat) {
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

func (r *runner) getService() error {
	if r.service != nil {
		return nil
	}

	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	serviceConfig := organization.Config{
		Client: client,
	}
	r.service, err = organization.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
