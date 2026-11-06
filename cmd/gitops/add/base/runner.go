package app

import (
	"context"
	"fmt"
	"io"
	"strconv"

	clustercommon "github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider/templates/capv"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider/templates/capz"
	"github.com/giantswarm/kubectl-gs/v6/internal/deploychart"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/base"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"

	providers "github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/provider/templates/capa"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
)

// vsphereCredentialsSecretName is the Secret `kubectl gs template cluster`
// takes the vCenter credentials from by default.
const vsphereCredentialsSecretName = "vsphere-credentials"

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
	case key.ProviderVSphere:
		return generateCapVClusterBaseTemplates(config)
	}

	return common.ClusterBaseTemplates{}, invalidProviderError
}

// generateClusterResources returns the OCIRepository and HelmRelease that
// deploy the release chart. The release chart has no version of its own, it
// is published with the version of the release it holds.
func generateClusterResources(releaseChart string, extraValuesFrom ...deploychart.ValuesFromReference) (string, error) {
	ociRepo, helmRelease, err := clustercommon.BuildClusterFluxResources(
		clustercommon.ClusterConfig{
			Name:           "${cluster_name}",
			Organization:   "${organization}",
			ReleaseVersion: "${release}",
		},
		releaseChart,
		"${cluster_name}-config",
		extraValuesFrom...,
	)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return fmt.Sprintf("%s---\n%s", ociRepo, helmRelease), nil
}

func generateCapAClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterResources, err := generateClusterResources(providers.ReleaseAWSRepoName)
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
	})
	// The release chart carries the release version in its default values.
	clusterConfig.Global.Release = nil
	clusterValues, err := capa.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterResources = clusterResources
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}

func generateCapVClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	// The release chart reads the vCenter credentials from the values, so the
	// HelmRelease has to pass the credentials Secret along.
	clusterResources, err := generateClusterResources(
		providers.ReleaseVsphereRepoName,
		deploychart.ValuesFromReference{Kind: "Secret", Name: vsphereCredentialsSecretName},
	)
	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapvClusterConfig(clustercommon.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
	})
	clusterConfig.Global.Release = nil
	clusterValues, err := capv.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterResources = clusterResources
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}

func generateCapZClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterResources, err := generateClusterResources(providers.ReleaseAzureRepoName)
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
	clusterConfig.Global.Release = nil
	clusterValues, err := capz.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterResources = clusterResources
	clusterBaseTemplates.ClusterValues = clusterValues

	return clusterBaseTemplates, nil
}
