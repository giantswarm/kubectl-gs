package app

import (
	"context"
	"io"
	"strconv"

	templateapp "github.com/giantswarm/kubectl-gs/v4/pkg/template/app"

	clustercommon "github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/provider/templates/capv"
	"github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/provider/templates/capz"
	"github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/structure/base"
	"github.com/giantswarm/kubectl-gs/v4/internal/gitops/structure/common"
	"github.com/giantswarm/kubectl-gs/v4/internal/key"

	providers "github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/provider/templates/capa"
	capg "github.com/giantswarm/kubectl-gs/v4/cmd/template/cluster/provider/templates/gcp"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v4/pkg/commonconfig"
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
	config := common.StructureConfig{
		Provider:            r.flag.Provider,
		Region:              r.flag.Region,
		AzureSubscriptionID: r.flag.AzureSubscriptionID,
	}

	var err error
	config.ClusterBaseTemplates, err = generateClusterBaseTemplates(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig, err := base.NewClusterBase(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig.Stdout = r.stdout

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		creatorConfig.Path = localPathFlag.Value.String()
	}

	creatorObj := creator.NewCreator(*creatorConfig)

	err = creatorObj.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func generateClusterBaseTemplates(config common.StructureConfig) (common.ClusterBaseTemplates, error) {
	switch config.Provider {
	case key.ProviderCAPA:
		return generateCapAClusterBaseTemplates(config)
	case key.ProviderCAPZ:
		return generateCapZClusterBaseTemplates(config)
	case key.ProviderGCP:
		return generateCapGClusterBaseTemplates(config)
	case key.ProviderOpenStack:
		return generateCapOClusterBaseTemplates(config)
	case key.ProviderVSphere:
		return generateCapVClusterBaseTemplates(config)
	}

	return common.ClusterBaseTemplates{}, invalidProviderError
}

func generateClusterAppCrTemplate(appName string) (string, error) {
	template, err := templateapp.NewAppCR(templateapp.Config{
		Name:                    appName,
		AppName:                 "${cluster_name}",
		Catalog:                 "cluster",
		InCluster:               true,
		Namespace:               "org-${organization}",
		Version:                 "",
		UserConfigConfigMapName: "${cluster_name}-config",
	})

	if err != nil {
		return "", err
	}

	return string(template), nil
}

func generateCapAClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterAppCr, err := generateClusterAppCrTemplate("cluster-aws")

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapaClusterConfig(clustercommon.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
		AWS: clustercommon.AWSConfig{
			MachinePool: clustercommon.AWSMachinePoolConfig{
				Name: "nodepool0",
			},
		},
		ReleaseVersion: "${release}",
	})
	clusterValues, err := capa.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}

func generateCapGClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterAppCr, err := generateClusterAppCrTemplate("cluster-gcp")

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapgClusterConfig(clustercommon.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
		GCP: clustercommon.GCPConfig{
			MachineDeployment: clustercommon.GCPMachineDeployment{
				Name: "machine-pool0",
			},
		},
	})
	clusterValues, err := capg.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}

func generateCapOClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterAppCr, err := generateClusterAppCrTemplate("cluster-openstack")

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapoClusterConfig(clustercommon.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
	}, 1)
	clusterValues, err := openstack.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}

func generateCapVClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterAppCr, err := generateClusterAppCrTemplate("cluster-vsphere")

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapvClusterConfig(clustercommon.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
	})
	clusterValues, err := capv.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}

func generateCapZClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterAppCr, err := generateClusterAppCrTemplate("cluster-azure")

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapzClusterConfig(clustercommon.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
		Region:       structureConfig.Region,
		Azure: clustercommon.AzureConfig{
			SubscriptionID: structureConfig.AzureSubscriptionID,
		},
	})
	clusterValues, err := capz.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}
