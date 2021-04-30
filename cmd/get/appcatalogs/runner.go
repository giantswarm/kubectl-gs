package appcatalogs

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/appcatalog"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/appcatalogentry"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs

	provider string
	service  appcatalog.Interface

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

	config := commonconfig.New(r.flag.config)
	{
		err = r.getService(config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var name string
	{
		if len(args) > 0 {
			name = strings.ToLower(args[0])
		}
	}

	var appCatalogResource appcatalog.Resource
	{
		options := appcatalog.GetOptions{
			Name: name,
		}
		appCatalogResource, err = r.service.GetObject(ctx, options)
		if appcatalogentry.IsNotFound(err) {
			return microerror.Maskf(notFoundError, fmt.Sprintf("An appcatalog '%s' cannot be found.\n", options.Name))
		} else if appcatalogentry.IsNoResources(err) && output.IsOutputDefault(r.flag.print.OutputFormat) {
			r.printNoResourcesOutput()

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	err = r.printOutput(appCatalogResource)
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

	serviceConfig := appcatalog.Config{
		Client: client,
	}
	r.service, err = appcatalog.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
