package app

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

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
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

	config := structure.AppConfig{
		App:             r.flag.App,
		Base:            r.flag.Base,
		Catalog:         r.flag.Catalog,
		Name:            r.flag.Name,
		Namespace:       r.flag.Namespace,
		Organization:    r.flag.Organization,
		WorkloadCluster: r.flag.WorkloadCluster,
		Version:         r.flag.Version,
	}

	if config.Name == "" {
		config.Name = config.App
	}

	if r.flag.UserValuesConfigMap != "" {
		config.UserValuesConfigMap, err = commonkey.ReadConfigMapYamlFromFile(
			afero.NewOsFs(),
			r.flag.UserValuesConfigMap,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		config.UserValuesConfigMap = strings.TrimSpace(config.UserValuesConfigMap)
	}
	if r.flag.UserValuesSecret != "" {
		byteData, err := commonkey.ReadSecretYamlFromFile(
			afero.NewOsFs(),
			r.flag.UserValuesSecret,
		)
		if err != nil {
			return microerror.Mask(err)
		}

		config.UserValuesSecret = strings.TrimSpace(string(byteData))
	}

	fsObjects, fsModifiers, err := structure.NewApp(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		Stdout:        r.stdout,
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
	}

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		creatorConfig.Path = fmt.Sprintf("%s/", localPathFlag.Value.String())
	}
	creatorConfig.Path += key.WorkloadClusterAppDirectory(
		r.flag.ManagementCluster,
		r.flag.Organization,
		r.flag.WorkloadCluster,
	)

	creator := creator.NewCreator(creatorConfig)

	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
