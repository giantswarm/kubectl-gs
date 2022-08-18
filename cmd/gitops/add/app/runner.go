package app

import (
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	structure "github.com/giantswarm/kubectl-gs/internal/gitops/structure/app"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/common"
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

	config := common.StructureConfig{
		App:               r.flag.App,
		AppBase:           r.flag.Base,
		AppCatalog:        r.flag.Catalog,
		ManagementCluster: r.flag.ManagementCluster,
		AppName:           r.flag.Name,
		AppNamespace:      r.flag.Namespace,
		Organization:      r.flag.Organization,
		WorkloadCluster:   r.flag.WorkloadCluster,
		AppVersion:        r.flag.Version,
	}

	if config.AppName == "" {
		config.AppName = config.App
	}

	if r.flag.UserValuesConfigMap != "" {
		config.AppUserValuesConfigMap, err = commonkey.ReadConfigMapYamlFromFile(
			afero.NewOsFs(),
			r.flag.UserValuesConfigMap,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		config.AppUserValuesConfigMap = strings.TrimSpace(config.AppUserValuesConfigMap)
	}
	if r.flag.UserValuesSecret != "" {
		byteData, err := commonkey.ReadSecretYamlFromFile(
			afero.NewOsFs(),
			r.flag.UserValuesSecret,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		config.AppUserValuesSecret = strings.TrimSpace(string(byteData))
	}

	creatorConfig, err := structure.NewApp(config)
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

	creator := creator.NewCreator(*creatorConfig)

	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
