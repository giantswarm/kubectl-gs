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
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/appcatalog"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs

	service appcatalog.Interface

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

	var name, selector string
	{
		if len(args) > 0 {
			name = strings.ToLower(args[0])
			selector = fmt.Sprintf("application.giantswarm.io/catalog=%s,latest=true", name)
		} else {
			selector = r.flag.LabelSelector
		}
	}

	var labelSelector labels.Selector
	{
		labelSelector, err = labels.Parse(selector)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appCatalogResource appcatalog.Resource
	{

		options := appcatalog.GetOptions{
			Name:          name,
			LabelSelector: labelSelector,
		}
		appCatalogResource, err = r.service.Get(ctx, options)
		if appcatalog.IsNotFound(err) {
			return microerror.Maskf(notFoundError, fmt.Sprintf("An appcatalog '%s' cannot be found.\n", options.Name))
		} else if app.IsNoMatch(err) {
			r.printNoMatchOutput()
			return nil
		} else if appcatalog.IsNoResources(err) && output.IsOutputDefault(r.flag.print.OutputFormat) {
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
