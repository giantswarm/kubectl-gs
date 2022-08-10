package structure

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier"
	appmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/app"
	fluxkusmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/flux-kustomization"
	secmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/secret"
	sigskusmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/sigs-kustomization"
	sopsmod "github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier/sops"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	apptmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/app"
	updatetmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/automatic-updates"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/common"
	mctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/management-cluster"
	orgtmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/organization"
	roottmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/root"
	wctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/workload-cluster"
)

const (
	masterPrefix = "master"
)

// Initialize create a basic directory structure for the repository
func Initialize() (*creator.CreatorConfig, error) {
	// Holds management-clusters
	mcsDir := key.ManagementClustersDirName()

	// We initialize repository with the `management-clusters` directory
	// and SOPS configuration file only.
	fsObjects := []*creator.FsObject{creator.NewFsObject(mcsDir, nil)}

	err := appendFromTemplate(&fsObjects, "", roottmpl.GetRepositoryRootTemplates, nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
	}

	return &creatorConfig, nil
}

// NewApp creates a new App directory structure.
func NewApp(config StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps/APP_NAME
	appDir := key.ResourcePath(appsDir, config.AppName)

	// Holds management-clusters/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps/kustomization.yaml
	appsKusFile := key.ResourcePath(appsDir, key.SigsKustomizationFileName())

	// We start from the `apps` directory despite the fact this directory
	// should already exist at this point. We then create the `APP_NAME` directory
	// and add bunch of files there, depending on the configuration provided.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(appsDir, nil),
		creator.NewFsObject(appDir, nil),
	}

	err = appendFromTemplate(&fsObjects, appDir, apptmpl.GetAppDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Once files are added, we then need to add resources to the `apps/kustomization.yaml`,
	// either one by one when no base is used, or as a whole directory when it is used.
	resources := make([]string, 0)
	if config.AppBase == "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.AppName, key.AppCRFileName()))
	} else {
		resources = append(resources, config.AppName)
	}

	if config.AppBase == "" && config.AppUserValuesConfigMap != "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.AppName, key.ConfigMapFileName()))
	}

	if config.AppBase == "" && config.AppUserValuesSecret != "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.AppName, key.SecretFileName()))
	}

	// Create Kustomization post modifiers that actually drops the needed changes
	// to the Kustomization CR
	fsModifiers := map[string]modifier.Modifier{
		appsKusFile: sigskusmod.KustomizationModifier{
			ResourcesToAdd: resources,
		},
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
	}

	return &creatorConfig, nil
}

// NewImageUpdate configures app for automated updates
func NewAutomaticUpdate(config StructureConfig) (*creator.CreatorConfig, error) {
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

	err = appendFromTemplate(&fsObjects, autoUpdatesDir, updatetmpl.GetAutomaticUpdatesTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Next we add `Image*` resources to the `APP_NAME` directory.
	err = appendFromTemplate(&fsObjects, appDir, updatetmpl.GetAppImageUpdatesTemplates, config)
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

// NewEncryption configures repository for the new SOPS key pair
func NewEncryption(config StructureConfig) (*creator.CreatorConfig, error) {
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

// NewManagementCluster creates a new Management Cluster directory
// structure.
func NewManagementCluster(config StructureConfig) (*creator.CreatorConfig, error) {
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
	fsObjects := []*creator.FsObject{creator.NewFsObject(mcDir, nil)}

	err = appendFromTemplate(&fsObjects, mcDir, mctmpl.GetManagementClusterTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, creator.NewFsObject(secretsDir, nil))
	err = appendFromTemplate(&fsObjects, secretsDir, mctmpl.GetManagementClusterSecretsTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, creator.NewFsObject(sopsDir, nil))
	err = appendFromTemplate(&fsObjects, sopsDir, mctmpl.GetManagementClusterSOPSTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, creator.NewFsObject(orgsDir, nil))

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
	}

	return &creatorConfig, nil
}

// NewOrganization creates a new Organization directory
// structure.
func NewOrganization(config StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME
	orgDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters
	wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())

	// Create `ORG_NAME` directory and add `ORG_NAME.yaml`manifest
	// containing Organization CR definition.
	fsObjects := []*creator.FsObject{creator.NewFsObject(orgDir, nil)}

	err = appendFromTemplate(&fsObjects, orgDir, orgtmpl.GetOrganizationDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create `workload-cluster` directory and populate it with an
	// empty `kustomization.yaml`.
	fsObjects = append(fsObjects, creator.NewFsObject(wcsDir, nil))
	err = appendFromTemplate(&fsObjects, wcsDir, orgtmpl.GetWorkloadClustersDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects: fsObjects,
	}

	return &creatorConfig, nil
}

// NewWorkloadCluster creates a new Workload Cluster directory
// structure.
func NewWorkloadCluster(config StructureConfig) (*creator.CreatorConfig, error) {
	var err error

	// Holds management-cluster/MC_NAME
	mcDir := key.BaseDirPath(config.ManagementCluster, "", "")

	// Holds management-cluster/MC_NAME/secrets
	secretsDir := key.ResourcePath(mcDir, key.SecretsDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME
	orgDir := key.BaseDirPath(config.ManagementCluster, config.Organization, "")

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters
	wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/apps
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/WC_NAME/cluster
	clusterDir := key.ResourcePath(wcDir, key.ClusterDirName())

	// Holds management-cluster/MC_NAME/organizations/ORG_NAME/workload-clusters/kustomization.yaml
	wcsKusFile := key.ResourcePath(wcsDir, key.SigsKustomizationFileName())

	fsObjects := []*creator.FsObject{}

	// Adding the `MC_NAME/secrets/WC_NAME.gpgkey.enc.yaml. We start here since it's the
	// highest layer in the structure. We add an empty secret there, that serves
	// as the SOPS private keys storage. It is empty by default and can be populated by the
	// `add encryption` command.
	// The need for this here may be questioned, but the rationale is: it is much easier and
	// structured this way. If it is not added here, empty, then it would have to be created
	// when encryption is requested later on. But when that happens, we do not have a way to
	// tell whether something exist or not here in the `structure` package. We can only tell
	// creator to create something or modify it, assuming it exists. So instead of tweaking
	// the creator's logic, I decided to drop an empty Secret and edit it with creator's
	// post modifiers.
	err = appendFromTemplate(&fsObjects, secretsDir, wctmpl.GetSecretsDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// We then continue at the `workload-clusters` directory. This should already
	// exist at this point, as a result of Organization creation, but we
	// need to point to this directory anyway in order to drop Kustomization
	// there.
	fsObjects = append(fsObjects, creator.NewFsObject(wcsDir, nil))

	// Add Kustomization CR to the `workload-clusters` directory and other
	// files if needed. Currently only Kustomization CR is considered.
	err = appendFromTemplate(&fsObjects, wcsDir, wctmpl.GetWorkloadClusterDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create `WC_NAME` specific directory and then add `apps` and `cluster`
	// directories there.
	// If base has been specified by the user, then in addition to the above, populate
	// the `cluster` directory with cluster definition, possibly enriching it with
	// user configuration when specified as well.
	fsObjects = append(
		fsObjects,
		[]*creator.FsObject{
			creator.NewFsObject(wcDir, nil),
			creator.NewFsObject(appsDir, nil),
			creator.NewFsObject(clusterDir, nil),
		}...,
	)

	// The `apps/*` directory pre-configuration including kustomization.yaml and
	// kubeconfig patch.
	err = appendFromTemplate(&fsObjects, appsDir, wctmpl.GetAppsDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// The `cluster/*` files, aka cluster definition, including `kustomization.yaml`,
	// patches, etc.
	err = appendFromTemplate(&fsObjects, clusterDir, wctmpl.GetClusterDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// After creating all the files and directories, we need creator to run
	// post modifiers, so that cluster is included into `workload-clusters/kustomization.yaml`
	// file.
	fsModifiers := map[string]modifier.Modifier{
		wcsKusFile: sigskusmod.KustomizationModifier{
			ResourcesToAdd: []string{
				fmt.Sprintf("%s.yaml", config.WorkloadCluster),
			},
		},
	}

	creatorConfig := creator.CreatorConfig{
		FsObjects:     fsObjects,
		PostModifiers: fsModifiers,
	}

	return &creatorConfig, nil
}

// addFilesFromTemplate add files from the given template to the
// given directory.
func addFilesFromTemplate(path string, templates func() []common.Template, config interface{}) ([]*creator.FsObject, error) {
	var err error

	fsObjects := make([]*creator.FsObject, 0)
	for _, t := range templates() {
		// First, we template the name of the file
		nameTemplate := template.Must(template.New("name").Parse(t.Name))
		var name bytes.Buffer
		err = nameTemplate.Execute(&name, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		contentTemplate := template.Must(template.New("files").Funcs(sprig.TxtFuncMap()).Parse(t.Data))

		// Next, we template the file content
		var content bytes.Buffer
		err = contentTemplate.Execute(&content, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// Instead of conditioning in the `template` package and not returning an
		// empty file, we return empty files and condition here, effectively
		// removing them from the structure set.
		if len(content.Bytes()) <= 1 {
			continue
		}

		file := name.String()
		if path != "" {
			file = fmt.Sprintf("%s/%s", path, file)
		}

		fsObjects = append(
			fsObjects,
			creator.NewFsObject(
				file,
				content.Bytes(),
			),
		)
	}

	return fsObjects, nil
}

func appendFromTemplate(dst *[]*creator.FsObject, path string, templates func() []common.Template, config interface{}) error {
	fileObjects, err := addFilesFromTemplate(path, templates, config)
	if err != nil {
		return microerror.Mask(err)
	}

	*dst = append(*dst, fileObjects...)

	return nil
}
