package base

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/key"
	base "github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/base/templates"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/common"
)

func NewClusterBase(config common.StructureConfig) (*creator.CreatorConfig, error) {
	clusterBaseDir := key.ClusterBasePath(config.Provider)

	fsObjects := []*creator.FsObject{
		creator.NewFsObject(key.BaseTemplatesDirName(), nil, 0),
		creator.NewFsObject(key.ClusterBaseTemplatesPath(), nil, 0),
		creator.NewFsObject(key.ClusterBaseProviderPath(config.Provider), nil, 0),
		creator.NewFsObject(clusterBaseDir, nil, 0),
	}

	templates := base.GetClusterBaseTemplates
	err := common.AppendFromTemplate(&fsObjects, clusterBaseDir, templates, config)

	if err != nil {
		return nil, microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
	}

	return &creatorConfig, nil
}
