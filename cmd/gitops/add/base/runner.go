package app

import (
	"context"
	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	templateapp "github.com/giantswarm/kubectl-gs/v2/pkg/template/app"
	"io"
	"strconv"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/openstack"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/base"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/common"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"

	providers "github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/capa"
	capg "github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider/templates/gcp"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

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
	config := common.StructureConfig{
		Provider: r.flag.Provider,
	}

	var err error
	config.ClusterBaseTemplates, err = generateClusterBaseTemplates(config)
	if err != nil {
		return microerror.Mask(err)
	}

	// TODO move new method to structure package?
	creatorConfig, err := base.NewClusterBase(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig.Stdout = r.stdout

	// TODO These are repeated across all commands, possible to dry it?
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
	case key.ProviderGCP:
		return generateCapGClusterBaseTemplates(config)
	case key.ProviderOpenStack:
		return generateCapOClusterBaseTemplates(config)
	}

	return common.ClusterBaseTemplates{}, invalidProviderError
}

// TODO Go through / share common logic
func generateClusterAppCrTemplate(appName string) (string, error) {
	template, err := templateapp.NewAppCR(templateapp.Config{
		Name:                    appName,
		AppName:                 "${cluster_name}",
		Catalog:                 "cluster",
		InCluster:               true,
		Namespace:               "org-${organization}",
		Version:                 "${cluster_version}",
		UserConfigConfigMapName: "${cluster_name}-config",
	})

	if err != nil {
		return "", err
	}

	return string(template), nil
}

// TODO Go through / share common logic
func generateDefaultAppsAppCrTemplate(appName string) (string, error) {
	template, err := templateapp.NewAppCR(templateapp.Config{
		Name:                    appName,
		AppName:                 "${cluster_name}-default-apps",
		Cluster:                 "${cluster_name}",
		Catalog:                 "cluster",
		DefaultingEnabled:       false,
		InCluster:               true,
		Namespace:               "org-${organization}",
		Version:                 "${default_apps_version}",
		UserConfigConfigMapName: "${cluster_name}-default-apps-config",
		UseClusterValuesConfig:  true,
		ExtraLabels: map[string]string{
			k8smetadata.ManagedBy: "cluster",
		},
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

	clusterConfig := providers.BuildCapaClusterConfig(providers.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
	})
	clusterValues, err := capa.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	defaultAppsAppCr, err := generateDefaultAppsAppCrTemplate("default-apps-aws")

	if err != nil {
		return clusterBaseTemplates, err
	}

	defaultAppsValues, err := capa.GenerateDefaultAppsValues(capa.DefaultAppsConfig{
		ClusterName:  "${cluster_name}",
		Organization: "${organization}",
	})

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues
	clusterBaseTemplates.DefaultAppsAppCr = defaultAppsAppCr
	clusterBaseTemplates.DefaultAppsValues = defaultAppsValues

	return clusterBaseTemplates, nil
}

func generateCapGClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterAppCr, err := generateClusterAppCrTemplate("cluster-gcp")

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapgClusterConfig(providers.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
	})
	clusterValues, err := capg.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	defaultAppsAppCr, err := generateDefaultAppsAppCrTemplate("default-apps-gcp")

	if err != nil {
		return clusterBaseTemplates, err
	}

	defaultAppsValues, err := capg.GenerateDefaultAppsValues(capg.DefaultAppsConfig{
		ClusterName:  "${cluster_name}",
		Organization: "${organization}",
	})

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues
	clusterBaseTemplates.DefaultAppsAppCr = defaultAppsAppCr
	clusterBaseTemplates.DefaultAppsValues = defaultAppsValues

	return clusterBaseTemplates, nil
}

func generateCapOClusterBaseTemplates(structureConfig common.StructureConfig) (common.ClusterBaseTemplates, error) {
	clusterBaseTemplates := common.ClusterBaseTemplates{}

	clusterAppCr, err := generateClusterAppCrTemplate("cluster-openstack")

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterConfig := providers.BuildCapoClusterConfig(providers.ClusterConfig{
		Name:         "${cluster_name}",
		Organization: "${organization}",
	}, 1)
	clusterValues, err := openstack.GenerateClusterValues(clusterConfig)

	if err != nil {
		return clusterBaseTemplates, err
	}

	defaultAppsAppCr, err := generateDefaultAppsAppCrTemplate("default-apps-openstack")

	if err != nil {
		return clusterBaseTemplates, err
	}

	defaultAppsValues, err := openstack.GenerateDefaultAppsValues(openstack.DefaultAppsConfig{
		ClusterName:  "${cluster_name}",
		Organization: "${organization}",
	})

	if err != nil {
		return clusterBaseTemplates, err
	}

	clusterBaseTemplates.ClusterAppCr = clusterAppCr
	clusterBaseTemplates.ClusterValues = clusterValues
	clusterBaseTemplates.DefaultAppsAppCr = defaultAppsAppCr
	clusterBaseTemplates.DefaultAppsValues = defaultAppsValues

	return clusterBaseTemplates, nil
}
