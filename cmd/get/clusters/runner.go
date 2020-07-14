package clusters

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	dataClient "github.com/giantswarm/kubectl-gs/pkg/data/client"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs

	client  *dataClient.Client
	service cluster.Interface

	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(cmd.Context(), cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	{
		err = r.getClient()
		if err != nil {
			return microerror.Mask(err)
		}
		err = r.getService()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	res, err := r.service.V4ListKVM(ctx, &cluster.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Println(len(res.Items))

	return nil
}

func (r *runner) getClient() error {
	if r.client != nil {
		return nil
	}

	restConfig, err := r.flag.config.ToRESTConfig()
	if err != nil {
		return microerror.Mask(err)
	}

	config := dataClient.Config{
		Logger:        r.logger,
		K8sRestConfig: restConfig,
	}

	r.client, err = dataClient.New(config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) getService() error {
	if r.service != nil {
		return nil
	}

	config := cluster.Config{
		Client: r.client,
	}
	var err error
	r.service, err = cluster.New(config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
