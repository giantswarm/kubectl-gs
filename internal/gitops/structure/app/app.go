package structure

import (
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier"
	sigskusmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/sigs-kustomization"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	apptmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/app/templates"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/common"
)

// NewApp creates a new App directory structure.
func NewApp(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	if !config.SkipMAPI {
		wcDir = key.ResourcePath(wcDir, key.MapiDirName())
	}

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps/APP_NAME
	appDir := key.ResourcePath(appsDir, config.AppName)

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps/kustomization.yaml
	appsKusFile := key.ResourcePath(appsDir, key.SigsKustomizationFileName())

	// We start from the `apps` directory despite the fact this directory
	// should already exist at this point. We then create the `APP_NAME` directory
	// and add bunch of files there, depending on the configuration provided.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(appsDir, nil, 0),
		creator.NewFsObject(appDir, nil, 0),
	}

	err = common.AppendFromTemplate(&fsObjects, appDir, apptmpl.GetAppDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Once files are added, we then need to add resources to the `apps/kustomization.yaml`,
	// either one by one when no base is used, or as a whole directory when it is used.
	resources := make([]string, 0)
	if config.AppBase == "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.AppName, key.AppCRFileName()))
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
	}

	return &creatorConfig, nil
}
