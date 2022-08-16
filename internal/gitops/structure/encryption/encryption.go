package structure

import (
	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier"
	fluxkusmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/flux-kustomization"
	secmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/secret"
	sopsmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/sops"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/common"
)

const (
	masterPrefix = "master"
)

// NewEncryption configures repository for the new SOPS key pair
func NewEncryption(config common.StructureConfig) (*creator.CreatorConfig, error) {
	// Holds management-clusters/MC_NAME
	mcDir := key.BaseDirPath(config.ManagementCluster, "", "")

	// Holds management-clusters/MC_NAME/secrets
	secretsDir := key.ResourcePath(mcDir, key.SecretsDirName())

	// Holds management-clusters/MC_NAME/.sops.keys
	sopsDir := key.ResourcePath(mcDir, key.SopsKeysDirName())

	// Either MC_NAME or WC_NAME
	keyPrefix := key.SopsKeyPrefix(config.ManagementCluster, config.WorkloadCluster)

	// Holds management-clusters/MC_NAME/.sops.keys/PREFIX.FINGERPRINT.asc
	sopsPubKeyFile := key.ResourcePath(
		sopsDir,
		key.SopsKeyName(keyPrefix, config.EncryptionKeyPair.Fingerprint),
	)

	// Holds management-clusters/MC_NAME/secrets/PREFIX.gpgkey.enc.yaml
	sopsPrvKeyFile := key.ResourcePath(
		secretsDir,
		key.SopsSecretFileName(keyPrefix),
	)

	// Makes sure the management-clusters/MC_NAME/.sops.keys/PREFIX.FINGERPRINT.asc
	// gets created
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(sopsPubKeyFile, []byte(config.EncryptionKeyPair.PublicData)),
	}

	// Post modifiers make sure other files get the right values necessary
	// to enable encryption for the given path.
	fsModifiers := map[string]modifier.Modifier{
		// Add private SOPS key to the `sopsPrvKeyFile` Secret
		sopsPrvKeyFile: secmod.SecretModifier{
			KeysToAdd: map[string]string{
				key.SopsKeyName(keyPrefix, config.EncryptionKeyPair.Fingerprint): config.EncryptionKeyPair.PrivateData,
			},
		},
	}

	if config.WorkloadCluster != "" {
		// Construct path to the WC_NAME.yaml file.
		orgDir := key.BaseDirPath(config.ManagementCluster, config.Organization, "")
		wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())
		wcKusFile := key.ResourcePath(wcsDir, key.FluxKustomizationFileName(config.WorkloadCluster))

		// Add decryption field to the Flux Kustomization CR for the
		// Workload Cluster
		fsModifiers[wcKusFile] = fluxkusmod.KustomizationModifier{
			DecryptionToAdd: key.SopsSecretName(keyPrefix),
		}
	} else {
		// Construct path to the MC_NAME.yaml file
		mcKusFile := key.ResourcePath(mcDir, key.FluxKustomizationFileName(config.ManagementCluster))

		// Add decryption field to the Flux Kustomization CR for the
		// Management Cluster
		fsModifiers[mcKusFile] = fluxkusmod.KustomizationModifier{
			DecryptionToAdd: key.SopsSecretName(masterPrefix),
		}
	}

	encPath := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	encPath = key.EncryptionRegex(encPath, config.EncryptionTarget)

	fsModifiers[key.SopsConfigFileName()] = sopsmod.SopsModifier{
		RulesToAdd: []map[string]interface{}{
			sopsmod.NewRule("", encPath, config.EncryptionKeyPair.Fingerprint),
		},
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
	}

	return &creatorConfig, nil
}
