package encryption

import (
	"fmt"
	"path"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier"
	fluxkusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/flux-kustomization"
	secmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/secret"
	sigskusmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/sigs-kustomization"
	sopsmod "github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier/sops"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
)

const (
	masterPrefix = "master"

	missingMasterKey = "the workload cluster's key is encrypted with the master key of the management cluster, and %s has no rule for it. Add the master key first, with `kubectl gs gitops add encryption --generate --management-cluster %s`"

	// keysKustomization lists SOPS keys, which Flux keeps when they are
	// deleted from Git: without them it can't decrypt anything.
	keysKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
buildMetadata: [originAnnotations]
kind: Kustomization
# Flux decrypts with the keys here: deleting one stops GitOps for what it
# decrypts.
commonAnnotations:
  kustomize.toolkit.fluxcd.io/prune: disabled
labels:
  - pairs:
      giantswarm.io/managed-by: flux
      # nobody can delete them, Flux included, until this label is removed
      giantswarm.io/prevent-deletion: "true"
resources: []
`
)

// paths are the files and directories a new key pair touches.
type paths struct {
	mcDir   string
	orgDir  string
	sopsDir string

	// keyPrefix is MC_NAME, or WC_NAME for a workload cluster's key.
	keyPrefix string
}

// NewEncryption configures repository for the new SOPS key pair.
//
// The private key of the management cluster, or of one of its organisations,
// goes into the master key Secret in `MC_NAME/secrets`, in `default`. The one
// of a workload cluster goes into its organisation's `namespace/secrets`, in
// the organisation's namespace, where the cluster's Kustomizations run, and
// is encrypted with the master key, which the organisation's namespace
// Kustomization decrypts it with.
func NewEncryption(config common.StructureConfig) (*creator.CreatorConfig, error) {
	p := paths{
		mcDir:     key.BaseDirPath(config.ManagementCluster, "", ""),
		orgDir:    key.BaseDirPath(config.ManagementCluster, config.Organization, ""),
		keyPrefix: key.SopsKeyPrefix(config.ManagementCluster, config.WorkloadCluster),
	}
	p.sopsDir = key.ResourcePath(p.mcDir, key.SopsKeysDirName())

	var creatorConfig *creator.CreatorConfig
	if config.WorkloadCluster != "" {
		creatorConfig = workloadClusterKey(config, p)
	} else {
		creatorConfig = masterKey(config, p)
	}

	// Makes sure the management-clusters/MC_NAME/.sops.keys/PREFIX.FINGERPRINT.asc
	// gets created.
	creatorConfig.FsObjects = append(creatorConfig.FsObjects,
		creator.NewFsObject(p.sopsDir, nil, 0),
		creator.NewFsObject(
			key.ResourcePath(p.sopsDir, key.SopsKeyName(p.keyPrefix, config.EncryptionKeyPair.Fingerprint)),
			[]byte(config.EncryptionKeyPair.PublicData),
			0,
		),
	)

	return creatorConfig, nil
}

// masterKey adds the key of the management cluster, or of an organisation, to
// the master key Secret, which the management cluster's Kustomization
// decrypts with, and the organisation's namespace Kustomization for what the
// organisation keeps in its namespace.
func masterKey(config common.StructureConfig, p paths) *creator.CreatorConfig {
	secretsDir := key.ResourcePath(p.mcDir, key.SecretsDirName())
	secretFile := key.SopsSecretFileName(p.keyPrefix)

	// The creator never overwrites an existing file, so a Secret that already
	// holds keys keeps them, and a kustomization.yaml its resources.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(secretsDir, nil, 0),
		creator.NewFsObject(key.ResourcePath(secretsDir, secretFile), []byte(emptySecret(key.SopsSecretName(masterPrefix), "default")), 0),
		creator.NewFsObject(key.ResourcePath(secretsDir, key.SigsKustomizationFileName()), []byte(keysKustomization), 0),
	}

	fsModifiers := map[string]modifier.Modifier{
		key.ResourcePath(secretsDir, secretFile):                      addKey(config, p),
		key.ResourcePath(secretsDir, key.SigsKustomizationFileName()): sigskusmod.KustomizationModifier{ResourcesToAdd: []string{secretFile}},
		key.ResourcePath(p.mcDir, key.FluxKustomizationFileName(config.ManagementCluster)): fluxkusmod.KustomizationModifier{
			DecryptionToAdd: key.SopsSecretName(masterPrefix),
		},
		key.SopsConfigFileName(): sopsmod.SopsModifier{RulesToAdd: []map[string]interface{}{targetRule(config)}},
	}

	// What the organisation keeps in its namespace, namespace.yaml applies
	// and decrypts.
	if config.Organization != "" && isNamespaceTarget(config.EncryptionTarget) {
		fsModifiers[key.ResourcePath(p.orgDir, key.NamespaceFileName())] = fluxkusmod.KustomizationModifier{
			DecryptionToAdd: key.SopsSecretName(masterPrefix),
		}
		fsObjects = append(fsObjects, namespaceSecrets(fsModifiers, p, nil)...)
	}

	return &creator.CreatorConfig{FsObjects: fsObjects, PostModifiers: fsModifiers}
}

// workloadClusterKey adds the key of a workload cluster to its own Secret, in
// the organisation's namespace/secrets, which the cluster's Kustomizations
// decrypt with.
func workloadClusterKey(config common.StructureConfig, p paths) *creator.CreatorConfig {
	wcsDir := key.ResourcePath(p.orgDir, key.WorkloadClustersDirName())
	namespace := fmt.Sprintf("org-%s", config.Organization)
	secretFile := key.SopsSecretFileName(p.keyPrefix)
	secretName := key.SopsSecretName(p.keyPrefix)

	secretPath := key.ResourcePath(namespaceSecretsDir(p), secretFile)
	fsModifiers := map[string]modifier.Modifier{secretPath: addKey(config, p)}
	fsObjects := namespaceSecrets(fsModifiers, p, creator.NewFsObject(secretPath, []byte(emptySecret(secretName, namespace)), 0))

	// The organisation's namespace Kustomization decrypts the key with the
	// master key, and the cluster's Kustomizations decrypt the cluster's files
	// with the key. Only clusters with a base have an apps Kustomization.
	fsModifiers[key.ResourcePath(p.orgDir, key.NamespaceFileName())] = fluxkusmod.KustomizationModifier{
		DecryptionToAdd: key.SopsSecretName(masterPrefix),
	}
	fsModifiers[key.ResourcePath(wcsDir, key.FluxKustomizationFileName(config.WorkloadCluster))] = fluxkusmod.KustomizationModifier{
		DecryptionToAdd: secretName,
	}
	fsModifiers[key.ResourcePath(wcsDir, key.FluxKustomizationFileName(config.WorkloadCluster+"-apps"))] = modifier.IfExists{
		Modifier: fluxkusmod.KustomizationModifier{DecryptionToAdd: secretName},
	}

	masterRegex := key.EncryptionRegex(p.mcDir, key.SecretsDirName())
	fsModifiers[key.SopsConfigFileName()] = modifier.Chain{
		sopsmod.MasterKeyRuleModifier{MasterPathRegex: masterRegex, PathRegex: key.WorkloadClusterKeysRegex(p.mcDir)},
		sopsmod.SopsModifier{RulesToAdd: []map[string]interface{}{targetRule(config)}},
	}

	// No MetadataLayer here on purpose: enabling encryption configures a
	// workload cluster that is already recorded, rather than creating a layer
	// of its own.
	return &creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
		PreValidators: map[string]func(*afero.Afero, string) error{
			key.SopsConfigFileName(): func(fs *afero.Afero, path string) error {
				data, err := fs.ReadFile(path)
				if err != nil {
					return microerror.Mask(err)
				}
				ok, err := sopsmod.HasMasterKey(data, masterRegex)
				if err != nil {
					return microerror.Mask(err)
				}
				if !ok {
					return microerror.Maskf(creator.ValidationError, missingMasterKey, path, config.ManagementCluster)
				}
				return nil
			},
		},
	}
}

// namespaceSecrets sets up the organisation's namespace/secrets, protected
// from deletion and listed in namespace/kustomization.yaml, with the secret
// in it, when given.
func namespaceSecrets(fsModifiers map[string]modifier.Modifier, p paths, secret *creator.FsObject) []*creator.FsObject {
	kustomizationFile := key.ResourcePath(namespaceSecretsDir(p), key.SigsKustomizationFileName())

	fsObjects := []*creator.FsObject{creator.NewFsObject(namespaceSecretsDir(p), nil, 0)}
	if secret != nil {
		fsObjects = append(fsObjects, secret)
		fsModifiers[kustomizationFile] = sigskusmod.KustomizationModifier{ResourcesToAdd: []string{path.Base(secret.RelativePath)}}
	}
	fsObjects = append(fsObjects, creator.NewFsObject(kustomizationFile, []byte(keysKustomization), 0))

	namespaceKustomization := key.ResourcePath(key.ResourcePath(p.orgDir, key.NamespaceDirName()), key.SigsKustomizationFileName())
	fsModifiers[namespaceKustomization] = sigskusmod.KustomizationModifier{ResourcesToAdd: []string{key.SecretsDirName()}}

	return fsObjects
}

// namespaceSecretsDir is the organisation's namespace/secrets.
func namespaceSecretsDir(p paths) string {
	return key.ResourcePath(key.ResourcePath(p.orgDir, key.NamespaceDirName()), key.SecretsDirName())
}

// addKey adds the private key to the Secret holding it.
func addKey(config common.StructureConfig, p paths) secmod.SecretModifier {
	return secmod.SecretModifier{
		KeysToAdd: map[string]string{
			key.SopsKeyName(p.keyPrefix, config.EncryptionKeyPair.Fingerprint): config.EncryptionKeyPair.PrivateData,
		},
	}
}

// targetRule encrypts the files in the target directory with the new key.
func targetRule(config common.StructureConfig) map[string]interface{} {
	encPath := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	encPath = key.EncryptionRegex(encPath, config.EncryptionTarget)

	return sopsmod.NewRule("", encPath, config.EncryptionKeyPair.Fingerprint)
}

// isNamespaceTarget reports whether the target is the organisation's
// namespace/secrets, which namespace.yaml applies.
func isNamespaceTarget(target string) bool {
	return strings.Trim(target, "/") == key.NamespaceDirName()+"/"+key.SecretsDirName()
}

// emptySecret returns the Secret that stores the private SOPS keys, before
// any key is added to it.
func emptySecret(name, namespace string) string {
	return fmt.Sprintf("apiVersion: v1\nkind: Secret\nmetadata:\n    name: %s\n    namespace: %s\n", name, namespace)
}
