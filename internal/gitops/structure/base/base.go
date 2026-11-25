package base

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
	base "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/base/templates"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const legacyBaseExists = "a cluster base for %s already exists at %s, where kubectl-gs wrote it before the base moved to %s. " +
	"A second base would not be recorded in the repository metadata. " +
	"Keep using the existing one by passing %s to `kubectl gs gitops add workload-cluster --base`."

func NewClusterBase(config common.StructureConfig) (*creator.CreatorConfig, error) {
	clusterBaseDir := key.ClusterBasePath(config.Provider)

	fsObjects := []*creator.FsObject{
		creator.NewFsObject(key.BaseTemplatesDirName(), nil, 0),
		creator.NewFsObject(key.ClusterBaseTemplatesPath(), nil, 0),
		creator.NewFsObject(clusterBaseDir, nil, 0),
	}

	templates := base.GetClusterBaseTemplates
	err := common.AppendFromTemplate(&fsObjects, clusterBaseDir, templates, config)

	if err != nil {
		return nil, microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
		MetadataLayer: &metadata.Layer{
			Kind: metadata.LayerClusterBase,
			Path: clusterBaseDir,
		},
		PreValidators: map[string]func(fs *afero.Afero, path string) error{
			key.LegacyClusterBasePath(config.Provider): refuseLegacyBase(config.Provider),
		},
	}

	return &creatorConfig, nil
}

// refuseLegacyBase stops a base being written for a provider whose base is
// still at the earlier `bases/clusters/PROVIDER/template` path, so that the
// repository doesn't end up with two bases for the provider.
func refuseLegacyBase(provider string) func(fs *afero.Afero, path string) error {
	return func(fs *afero.Afero, path string) error {
		exists, err := fs.DirExists(path)
		if err != nil {
			return microerror.Mask(err)
		}
		if !exists {
			return nil
		}

		legacyPath := key.LegacyClusterBasePath(provider)

		return microerror.Maskf(creator.ValidationError, legacyBaseExists, provider, legacyPath, key.ClusterBasePath(provider), legacyPath)
	}
}
