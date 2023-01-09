package base

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/key"
	base "github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/base/templates"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/common"
)

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

	//var resources []string
	//for _, tmpl := range templates() {
	//	resources = append(resources, tmpl.Name)
	//}
	//
	//baseKusFile := key.ResourcePath(clusterBaseDir, key.SigsKustomizationFileName())
	//
	//fsModifiers := map[string]modifier.Modifier{
	//	baseKusFile: sigskusmod.KustomizationModifier{
	//		ResourcesToAdd: resources,
	//	},
	//}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
		//PostModifiers: fsModifiers,
		//PreValidators: map[string]func(fs *afero.Afero, path string) error{
		//	//appDir: func(fs *afero.Afero, path string) error {
		//	//	ok, err := fs.Exists(path)
		//	//	if err != nil {
		//	//		return microerror.Mask(err)
		//	//	}
		//	//
		//	//	if !ok {
		//	//		return nil
		//	//	}
		//	//
		//	//	return microerror.Maskf(creator.ValidationError, appExists, config.AppName)
		//	//},
		//},
	}

	return &creatorConfig, nil
}
