package structure

import (
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier"
	appmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/app"
	sigskusmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/sigs-kustomization"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/common"
	updatetmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/updates/templates"
)

// NewImageUpdate configures app for automated updates
func NewAutomaticUpdate(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/automatic-updates
	autoUpdatesDir := key.ResourcePath(wcDir, key.AutoUpdatesDirName())

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps/APP_NAME
	appDir := key.ResourcePath(appsDir, config.AppName)

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps/appcr.yaml
	appCrFile := key.ResourcePath(appDir, key.AppCRFileName())

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps/kustomization.yaml
	appsKusFile := key.ResourcePath(appsDir, key.SigsKustomizationFileName())

	// We start from the `WC_NAME` directory, because we must create the
	// `automatic-updates` directory there if it does not exist.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(autoUpdatesDir, nil),
	}

	err = common.AppendFromTemplate(&fsObjects, autoUpdatesDir, updatetmpl.GetAutomaticUpdatesTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Next we add `Image*` resources to the `APP_NAME` directory.
	err = common.AppendFromTemplate(&fsObjects, appDir, updatetmpl.GetAppImageUpdatesTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Once files are added, we then need to add resources to the
	// `apps/kustomization.yaml`, and also update App CR with the
	// automatic updates comments.
	resources := []string{
		fmt.Sprintf("%s/%s", config.AppName, key.ImagePolicyFileName()),
		fmt.Sprintf("%s/%s", config.AppName, key.ImageRepositoryFileName()),
	}

	// Create Kustomization and App CR post modifiers to drop the needed
	// changes to the respective resources.
	fsModifiers := map[string]modifier.Modifier{
		appsKusFile: sigskusmod.KustomizationModifier{
			ResourcesToAdd: resources,
		},
		appCrFile: appmod.AppModifier{
			ImagePolicyToAdd: map[string]string{
				fmt.Sprintf("org-%s", config.Organization): fmt.Sprintf("%s-%s", config.WorkloadCluster, config.AppName),
			},
		},
	}

	fsValidators := map[string]creator.Validator{
		key.ResourcePath(appDir, key.SigsKustomizationFileName()): creator.KustomizationValidator{
			ReferencesBase: true,
		},
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
		PreValidators: fsValidators,
	}

	return &creatorConfig, nil
}
