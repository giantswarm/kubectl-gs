package base

import (
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/common"
	wctmpl "github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/workload-cluster/templates"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
)

func NewClusterBase(config common.StructureConfig) (*creator.CreatorConfig, error) {
	clusterBaseDir := key.ClusterBasePath(config.Provider)

	fsObjects := []*creator.FsObject{
		creator.NewFsObject(clusterBaseDir, nil, 0),
	}

	//baseKusFile := key.ResourcePath(clusterBaseDir, key.SigsKustomizationFileName())

	err := common.AppendFromTemplate(&fsObjects, clusterBaseDir, wctmpl.GetClusterDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	//fsModifiers := map[string]modifier.Modifier{
	//	baseKusFile: sigskusmod.KustomizationModifier{
	//		ResourcesToAdd: resources,
	//	},
	//}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
		//PostModifiers: fsModifiers,
		PreValidators: map[string]func(fs *afero.Afero, path string) error{
			//appDir: func(fs *afero.Afero, path string) error {
			//	ok, err := fs.Exists(path)
			//	if err != nil {
			//		return microerror.Mask(err)
			//	}
			//
			//	if !ok {
			//		return nil
			//	}
			//
			//	return microerror.Maskf(creator.ValidationError, appExists, config.AppName)
			//},
		},
	}

	return &creatorConfig, nil
}
