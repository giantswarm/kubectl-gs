package wcluster

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure"
	commonkey "github.com/giantswarm/kubectl-gs/internal/key"
)

type runner struct {
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
	var err error

	config := structure.WcConfig{
		Base:               r.flag.Base,
		ClusterRelease:     r.flag.ClusterRelease,
		DefaultAppsRelease: r.flag.DefaultAppsRelease,
		ManagementCluster:  r.flag.ManagementCluster,
		Name:               r.flag.Name,
		Organization:       r.flag.Organization,
		RepositoryName:     r.flag.RepositoryName,
	}

	if r.flag.ClusterUserConfig != "" {
		config.ClusterUserConfig, err = commonkey.ReadConfigMapYamlFromFile(
			afero.NewOsFs(),
			r.flag.ClusterUserConfig,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		config.ClusterUserConfig = strings.TrimSpace(config.ClusterUserConfig)
	}
	if r.flag.DefaultAppsUserConfig != "" {
		config.DefaultAppsUserConfig, err = commonkey.ReadConfigMapYamlFromFile(
			afero.NewOsFs(),
			r.flag.DefaultAppsUserConfig,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		config.DefaultAppsUserConfig = strings.TrimSpace(config.DefaultAppsUserConfig)
	}

	dir, err := structure.NewWorkloadCluster(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig := filesystem.CreatorConfig{
		Directory: dir,
		Stdout:    r.stdout,
	}

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		creatorConfig.Path = fmt.Sprintf("%s/", localPathFlag.Value.String())
	}
	creatorConfig.Path += key.WorkloadClustersDirectory(r.flag.ManagementCluster, r.flag.Organization)

	creator := filesystem.NewCreator(creatorConfig)

	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
