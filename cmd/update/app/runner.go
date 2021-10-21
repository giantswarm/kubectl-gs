package app

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/catalog"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger

	service app.Interface

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

	namespace, _, err := r.flag.config.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return microerror.Mask(err)
	}

	name := r.flag.Name //strings.ToLower(args[0])
	version := r.flag.Version

	patchOptions := app.PatchOptions{
		Namespace: namespace,
		Name:      name,
		Version:   version,
	}

	err = r.service.Patch(ctx, patchOptions)
	if app.IsNotFound(err) {
		return microerror.Maskf(notFoundError, fmt.Sprintf("An app with name '%s' cannot be found in the '%s' namespace.\n", patchOptions.Name, patchOptions.Namespace))
	} else if catalog.IsNoResources(err) {
		return microerror.Maskf(noResourcesError, fmt.Sprintf("No AppCatalogEntry CR found for the given version: '%s'\n", patchOptions.Version))
	} else if err != nil {
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

	serviceConfig := app.Config{
		Client: client,
	}
	r.service, err = app.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
