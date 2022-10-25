package root

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/common"
	roottmpl "github.com/giantswarm/kubectl-gs/v2/internal/gitops/structure/root/templates"
)

const (
	gitHooksDir = ".git/hooks"
)

// Initialize create a basic directory structure for the repository
func Initialize() (*creator.CreatorConfig, error) {
	var err error

	// Holds management-clusters
	mcsDir := key.ManagementClustersDirName()

	// We initialize repository with the `management-clusters` directory
	// and SOPS configuration file only.
	fsObjects := []*creator.FsObject{creator.NewFsObject(mcsDir, nil, 0)}

	err = common.AppendFromTemplate(&fsObjects, "", roottmpl.GetRepositoryRootTemplates, nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = common.AppendFromTemplate(&fsObjects, gitHooksDir, roottmpl.GetRepositoryGitTemplates, nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
	}

	return &creatorConfig, nil
}
