package mcluster

import (
	"reflect"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/modifier"
	sopsmod "github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/modifier/sops"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/common"
	mctmpl "github.com/giantswarm/kubectl-gs/v5/internal/gitops/structure/management-cluster/templates"
)

// NewManagementCluster creates a new Management Cluster directory
// structure.
func NewManagementCluster(config common.StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-cluster/MC_NAME
	mcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-cluster/MC_NAME/organizations
	orgsDir := key.ResourcePath(mcDir, key.OrganizationsDirName())

	// Holds management-cluster/MC_NAME/secrets
	secretsDir := key.ResourcePath(mcDir, key.SecretsDirName())

	// Holds management-cluster/MC_NAME/.sops.keys
	sopsDir := key.ResourcePath(mcDir, key.SopsKeysDirName())

	// Adding a new Management Cluster is simple. We start at the
	// `management-clusters/MC_NAME` and then add definition and few
	// out of the basic directories.
	fsObjects := []*creator.FsObject{creator.NewFsObject(mcDir, nil, 0)}

	err = common.AppendFromTemplate(&fsObjects, mcDir, mctmpl.GetManagementClusterTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, creator.NewFsObject(secretsDir, nil, 0))
	err = common.AppendFromTemplate(&fsObjects, secretsDir, mctmpl.GetManagementClusterSecretsTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, creator.NewFsObject(sopsDir, nil, 0))
	err = common.AppendFromTemplate(&fsObjects, sopsDir, mctmpl.GetManagementClusterSOPSTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, creator.NewFsObject(orgsDir, nil, 0))

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
	}

	if !reflect.DeepEqual(config.EncryptionKeyPair, encryption.KeyPair{}) {
		encPath := key.EncryptionRegex(mcDir, "secrets")
		fsModifiers := map[string]modifier.Modifier{
			key.SopsConfigFileName(): sopsmod.SopsModifier{
				RulesToAdd: []map[string]interface{}{
					sopsmod.NewRule("", encPath, config.EncryptionKeyPair.Fingerprint),
				},
			},
		}

		creatorConfig.PostModifiers = fsModifiers
	}

	return &creatorConfig, nil
}
