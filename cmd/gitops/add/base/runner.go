package app

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig

	flag   *flag
	logger micrologger.Logger
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
	fmt.Println("Hello world!")

	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	err = provider.WriteCAPATemplate(ctx, client, r.stdout, provider.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
		Namespace:    "${organization}",
		App: provider.AppConfig{
			ClusterCatalog: "cluster",
			//ClusterVersion:     "0.20.3",
			ClusterVersion:     "${cluster_version}",
			DefaultAppsCatalog: "cluster",
			//DefaultAppsVersion: "0.12.4",
			DefaultAppsVersion: "${default_apps_version}",
		},
	})
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Println("Good bye world!")
	return nil
}
