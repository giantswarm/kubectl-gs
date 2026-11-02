package wcluster

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	structure "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/workload-cluster"
	commonkey "github.com/giantswarm/kubectl-gs/v6/internal/key"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	fs     afero.Fs
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

	config := common.StructureConfig{
		ClusterBase:       r.flag.Base,
		Release:           r.flag.Release,
		ManagementCluster: r.flag.ManagementCluster,
		WorkloadCluster:   r.flag.Name,
		Organization:      r.flag.Organization,
		SkipMAPI:          r.flag.SkipMAPI,
		RepositoryName:    r.flag.RepositoryName,
	}

	localPath := ""
	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		localPath = localPathFlag.Value.String()
	}

	if r.flag.Base != "" {
		baseDir := filepath.Join(localPath, r.flag.Base)
		exists, err := afero.DirExists(r.fs, baseDir)
		if err != nil {
			return microerror.Mask(err)
		}
		if !exists {
			return microerror.Maskf(invalidFlagsError, "--%s %q not found under %q", flagBase, r.flag.Base, localPath)
		}

		config.ClusterChart, err = clusterChartName(r.fs, baseDir)
		if err != nil {
			return microerror.Mask(err)
		}
		if config.ClusterChart == "" && r.flag.Release != "" {
			fmt.Fprintf(r.stderr, "Base %q holds no cluster App, so --%s is only set as the `release` variable of the Flux Kustomization. Set the release in the base's values.\n", r.flag.Base, flagRelease)
		}
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

	creatorConfig, err := structure.NewWorkloadCluster(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig.Stdout = r.stdout

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	creatorConfig.Path = localPath

	creator := creator.NewCreator(*creatorConfig)

	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
