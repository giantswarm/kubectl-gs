package app

import (
	"fmt"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v6/internal/deploychart"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/appcr"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier"
	sigskusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/sigs-kustomization"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	apptmpl "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/app/templates"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const (
	appExists        = "`%s` app is already configured for the cluster."
	baseDeploysAppCR = "base %s deploys an App CR in %s, which kubectl-gs no longer supports. Move it to a HelmRelease first, see %s"
)

// NewApp creates a new App directory structure.
func NewApp(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	if !config.SkipMAPI {
		wcDir = key.ResourcePath(wcDir, key.MapiDirName())
	}

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/[mapi]/apps
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/[mapi]/apps/APP_NAME
	appDir := key.ResourcePath(appsDir, config.AppName)

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/[mapi]/apps/kustomization.yaml
	appsKusFile := key.ResourcePath(appsDir, key.SigsKustomizationFileName())

	// We start from the `apps` directory despite the fact this directory
	// should already exist at this point. We then create the `APP_NAME` directory
	// and add a bunch of files there, depending on the configuration provided.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(appsDir, nil, 0),
		creator.NewFsObject(appDir, nil, 0),
	}

	if config.AppBase == "" {
		config.AppResources, err = appResources(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	err = common.AppendFromTemplate(&fsObjects, appDir, apptmpl.GetAppDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Once files are added, we then need to add resources to the `apps/kustomization.yaml`,
	// either one by one when no base is used, or as a whole directory when it is used.
	resources := make([]string, 0)
	if config.AppBase == "" {
		resources = append(resources,
			fmt.Sprintf("%s/%s", config.AppName, key.OCIRepositoryFileName()),
			fmt.Sprintf("%s/%s", config.AppName, key.HelmReleaseFileName()),
		)
	} else {
		resources = append(resources, config.AppName)
	}

	if config.AppBase == "" && config.AppUserValuesConfigMap != "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.AppName, key.ConfigMapFileName()))
	}

	if config.AppBase == "" && config.AppUserValuesSecret != "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.AppName, key.SecretFileName()))
	}

	// Create Kustomization post modifiers that actually drops the needed changes
	// to the Kustomization CR
	fsModifiers := map[string]modifier.Modifier{
		appsKusFile: sigskusmod.KustomizationModifier{
			ResourcesToAdd: resources,
		},
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerApp,
			Path: appDir,
		},
		PreValidators: map[string]func(fs *afero.Afero, path string) error{
			appDir: func(fs *afero.Afero, path string) error {
				ok, err := fs.Exists(path)
				if err != nil {
					return microerror.Mask(err)
				}

				if !ok {
					return nil
				}

				return microerror.Maskf(creator.ValidationError, appExists, config.AppName)
			},
		},
	}

	// The user values patch only targets a HelmRelease, so it would match
	// nothing in a base deploying an App CR.
	if config.AppBase != "" {
		creatorConfig.PreValidators[config.AppBase] = func(fs *afero.Afero, path string) error {
			ok, err := fs.DirExists(path)
			if err != nil || !ok {
				return microerror.Mask(err)
			}

			files, err := appcr.Files(fs, path)
			if err != nil {
				return microerror.Mask(err)
			}
			if len(files) > 0 {
				return microerror.Maskf(creator.ValidationError, baseDeploysAppCR, config.AppBase, files[0], appcr.MigrationDocs)
			}

			return nil
		}
	}

	return &creatorConfig, nil
}

// appResources returns the OCIRepository pulling the app's chart and the
// HelmRelease installing it in the workload cluster, or in the management
// cluster for an in-cluster app. Their defaults follow the resources the App
// CR migration generates, so apps added here and migrated apps look alike.
func appResources(config common.StructureConfig) (common.AppResources, error) {
	name := fmt.Sprintf("%s-%s", config.WorkloadCluster, config.AppName)
	namespace := fmt.Sprintf("org-%s", config.Organization)
	inCluster := config.InCluster == "true"

	ociRepository := deploychart.BuildOCIRepository(deploychart.OCIRepositoryOptions{
		Name:        name,
		Namespace:   namespace,
		ClusterName: config.WorkloadCluster,
		URL:         fmt.Sprintf("%s/%s", strings.TrimSuffix(config.AppOCIURLPrefix, "/"), config.App),
		Version:     config.AppVersion,
		Interval:    "10m",
	})

	var valuesFrom []deploychart.ValuesFromReference
	if config.AppUserValuesConfigMap != "" {
		valuesFrom = append(valuesFrom, deploychart.ValuesFromReference{Kind: "ConfigMap", Name: name + "-user-values", ValuesKey: "values"})
	}
	if config.AppUserValuesSecret != "" {
		valuesFrom = append(valuesFrom, deploychart.ValuesFromReference{Kind: "Secret", Name: name + "-user-values", ValuesKey: "values"})
	}

	serviceAccountName := ""
	if inCluster {
		serviceAccountName = deploychart.BundleServiceAccountName
	}

	helmRelease := deploychart.BuildHelmRelease(deploychart.HelmReleaseOptions{
		Name:               name,
		Namespace:          namespace,
		ClusterName:        config.WorkloadCluster,
		ChartName:          config.AppName,
		TargetNamespace:    config.AppNamespace,
		StorageNamespace:   config.AppNamespace,
		CreateNamespace:    true,
		Interval:           "5m",
		Timeout:            "10m",
		ManagementCluster:  inCluster,
		ServiceAccountName: serviceAccountName,
		InstallRemediation: &deploychart.RemediationPolicy{
			Retries: 10,
		},
		UpgradeRemediation: &deploychart.RemediationPolicy{
			Retries:              10,
			RemediateLastFailure: true,
			Strategy:             "rollback",
		},
		ValuesFrom:       valuesFrom,
		InstallTimeout:   duration(config.AppInstallTimeout),
		RollbackTimeout:  duration(config.AppRollbackTimeout),
		UninstallTimeout: duration(config.AppUninstallTimeout),
		UpgradeTimeout:   duration(config.AppUpgradeTimeout),
	})

	ociRepositoryYAML, err := deploychart.MarshalManifest(ociRepository)
	if err != nil {
		return common.AppResources{}, microerror.Mask(err)
	}
	helmReleaseYAML, err := deploychart.MarshalManifest(helmRelease)
	if err != nil {
		return common.AppResources{}, microerror.Mask(err)
	}

	return common.AppResources{
		HelmRelease:   string(helmReleaseYAML),
		OCIRepository: string(ociRepositoryYAML),
	}, nil
}

func duration(d *metav1.Duration) string {
	if d == nil {
		return ""
	}
	return d.Duration.String()
}
