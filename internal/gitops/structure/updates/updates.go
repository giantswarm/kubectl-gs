package updates

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/appcr"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier"
	ocirepomod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/oci-repository"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const (
	appCRNotSupported = "`%s` app is deployed with an App CR, which kubectl-gs no longer supports. Move it to a HelmRelease first, see %s"
	missingOCIRepo    = "`%s` app has no `%s` file. For an app from a base, set `spec.ref.semver` on the base's OCIRepository."
)

// NewAutomaticUpdate makes the app's OCIRepository follow a semver range, so
// Flux upgrades the app to the newest chart version in the range.
func NewAutomaticUpdate(config common.StructureConfig) (*creator.CreatorConfig, error) {
	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	if !config.SkipMAPI {
		wcDir = key.ResourcePath(wcDir, key.MapiDirName())
	}

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/[mapi]/apps/APP_NAME
	appDir := key.ResourcePath(key.ResourcePath(wcDir, key.AppsDirName()), config.AppName)

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/[mapi]/apps/APP_NAME/oci-repository.yaml
	ociRepoFile := key.ResourcePath(appDir, key.OCIRepositoryFileName())

	// No MetadataLayer here on purpose: automatic updates configure an app
	// that is already recorded, rather than creating a layer of their own.
	creatorConfig := creator.CreatorConfig{
		PostModifiers: map[string]modifier.Modifier{
			ociRepoFile: ocirepomod.SemverModifier{
				Range: config.AppVersionRange,
			},
		},
		PreValidators: map[string]func(fs *afero.Afero, path string) error{
			appDir: func(fs *afero.Afero, path string) error {
				ok, err := fs.DirExists(path)
				if err != nil {
					return microerror.Mask(err)
				}
				if !ok {
					return microerror.Maskf(creator.ValidationError, missingOCIRepo, config.AppName, key.OCIRepositoryFileName())
				}

				files, err := appcr.Files(fs, path)
				if err != nil {
					return microerror.Mask(err)
				}
				if len(files) > 0 {
					return microerror.Maskf(creator.ValidationError, appCRNotSupported, config.AppName, appcr.MigrationDocs)
				}

				ok, err = fs.Exists(key.ResourcePath(path, key.OCIRepositoryFileName()))
				if err != nil {
					return microerror.Mask(err)
				}
				if !ok {
					return microerror.Maskf(creator.ValidationError, missingOCIRepo, config.AppName, key.OCIRepositoryFileName())
				}

				return nil
			},
		},
	}

	return &creatorConfig, nil
}
