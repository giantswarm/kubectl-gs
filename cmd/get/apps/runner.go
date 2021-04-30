package apps

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

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/pkg/output"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs

	provider string
	service  app.Interface

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

	var namespace string
	{
		namespace, _, err = r.flag.config.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var appResource app.Resource
	{
		options := app.GetOptions{
			Name:      name,
			Namespace: namespace,
		}
		{
			if len(args) > 0 {
				options.Name = strings.ToLower(args[0])
			}

			if r.flag.AllNamespaces {
				options.Namespace = metav1.NamespaceAll
			} else {
				options.Namespace, _, err = r.flag.config.ToRawKubeConfigLoader().Namespace()
				if err != nil {
					return microerror.Mask(err)
				}
			}
		}

		appResource, err = r.service.GetObject(ctx, options)
		if app.IsNotFound(err) {
			return microerror.Maskf(notFoundError, fmt.Sprintf("An app '%s/%s' cannot be found.\n", options.Namespace, options.Name))
		} else if app.IsNoResources(err) && output.IsOutputDefault(r.flag.print.OutputFormat) {
			r.printNoResourcesOutput()

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	err = r.printOutput(appResource)
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

	serviceConfig := app.Config{
		Client: client,
	}
	r.service, err = app.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
