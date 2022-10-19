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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/filesystem/creator"
	structure "github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/app"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/common"
	commonkey "github.com/giantswarm/kubectl-gs/v2/internal/key"
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

	targetNamespace := r.flag.TargetNamespace
	if targetNamespace == "" {
		targetNamespace = r.flag.Namespace
	}

	config := common.StructureConfig{
		App:               r.flag.App,
		AppBase:           r.flag.Base,
		AppCatalog:        r.flag.Catalog,
		AppName:           r.flag.Name,
		AppNamespace:      targetNamespace,
		AppVersion:        r.flag.Version,
		ManagementCluster: r.flag.ManagementCluster,
		Organization:      r.flag.Organization,
		SkipMAPI:          r.flag.SkipMAPI,
		WorkloadCluster:   r.flag.WorkloadCluster,
	}

	r.setTimeouts(&config)

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

func (r *runner) setTimeouts(config *common.StructureConfig) {
	if r.flag.InstallTimeout != 0 {
		config.AppInstallTimeout = &metav1.Duration{Duration: r.flag.InstallTimeout}
	}
	if r.flag.RollbackTimeout != 0 {
		config.AppRollbackTimeout = &metav1.Duration{Duration: r.flag.RollbackTimeout}
	}
	if r.flag.UninstallTimeout != 0 {
		config.AppUninstallTimeout = &metav1.Duration{Duration: r.flag.UninstallTimeout}
	}
	if r.flag.UpgradeTimeout != 0 {
		config.AppUpgradeTimeout = &metav1.Duration{Duration: r.flag.UpgradeTimeout}
	}
}
