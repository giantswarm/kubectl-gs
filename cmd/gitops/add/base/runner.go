package app

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/capa"
	capg "github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/gcp"

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

	err := r.writeCapaTemplate(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	//err := r.writeCapgTemplate(ctx)
	//if err != nil {
	//	return microerror.Mask(err)
	//}

	//err := r.writeCapvTemplate(ctx)
	//if err != nil {
	//	return microerror.Mask(err)
	//}

	//err := r.writeCapzTemplate(ctx)
	//if err != nil {
	//	return microerror.Mask(err)
	//}

	fmt.Println("Good bye world!")
	return nil
}

func (r *runner) writeCapaTemplate(ctx context.Context) error {
	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	// ********************************************************************************

	clusterValues, err := capa.GenerateClusterValues(capa.ClusterConfig{
		ClusterName:  "${cluster_name}",
		Organization: "${organization}",
	})
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Println(clusterValues)

	defaultAppsValues, err := capa.GenerateDefaultAppsValues(capa.DefaultAppsConfig{
		ClusterName:  "${cluster_name}",
		Organization: "${organization}",
	})
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Println(defaultAppsValues)

	// ********************************************************************************

	return provider.WriteCAPATemplate(ctx, client, r.stdout, provider.ClusterConfig{
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
}

func (r *runner) writeCapgTemplate(ctx context.Context) error {
	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	// ********************************************************************************

	clusterValues, err := capg.GenerateClusterValues(capg.ClusterConfig{
		ClusterName:  "${cluster_name}",
		Organization: "${organization}",
	})
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Println(clusterValues)

	defaultAppsValues, err := capg.GenerateDefaultAppsValues(capg.DefaultAppsConfig{
		ClusterName:  "${cluster_name}",
		Organization: "${organization}",
	})
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Println(defaultAppsValues)

	// ********************************************************************************

	return provider.WriteGCPTemplate(ctx, client, r.stdout, provider.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
		Namespace:    "${organization}",
		App: provider.AppConfig{
			ClusterCatalog:     "cluster",
			ClusterVersion:     "${cluster_version}",
			DefaultAppsCatalog: "cluster",
			DefaultAppsVersion: "${default_apps_version}",
		},
	})
}

// Issues:
// - no ready cluster for it to test with?
//func (r *runner) writeCapvTemplate(ctx context.Context) error {
//	client, err := r.commonConfig.GetClient(r.logger)
//	if err != nil {
//		return microerror.Mask(err)
//	}
//
//	return provider.WriteVSphereTemplate(ctx, client, r.stdout, provider.ClusterConfig{
//		Name:         "${cluster_name}",
//		Organization: "${organization}",
//		Namespace:    "${organization}",
//		App: provider.AppConfig{
//			ClusterCatalog:     "cluster",
//			ClusterVersion:     "${cluster_version}",
//			DefaultAppsCatalog: "cluster",
//			DefaultAppsVersion: "${default_apps_version}",
//		},
//	})
//}

// Issues:
//   - the organization name must be valid and exist on the MC that the command runs against
//   - the cluster name must be valid: "lowercase RFC 1123 subdomain"
func (r *runner) writeCapzTemplate(ctx context.Context) error {
	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	return provider.WriteCAPZTemplate(ctx, client, r.stdout, provider.ClusterConfig{
		//Name: "${cluster_name}",
		// needed because of: https://github.com/giantswarm/kubectl-gs/blob/43bcc221f67194ceee51965f2c7b485808726cf9/cmd/template/cluster/provider/common.go#L185
		// so we would need to disable that or need a replacing logic, value must be a "lowercase RFC 1123 subdomain"
		Name: "placeholder-cluster-name",
		//Organization: "${organization}",
		Organization: "test-philippe",
		Namespace:    "${organization}",
		App: provider.AppConfig{
			ClusterCatalog:     "cluster",
			ClusterVersion:     "${cluster_version}",
			DefaultAppsCatalog: "cluster",
			DefaultAppsVersion: "${default_apps_version}",
		},
	})
}
