package structure

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	apptmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/app"
	updatetmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/automatic-updates"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/common"
	mctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/management-cluster"
	orgtmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/organization"
	roottmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/root"
	wctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/workload-cluster"
)

// Initialize create a basic directory structure for the repository
func Initialize() (*creator.CreatorConfig, error) {
	//var err error

	mcsDir := key.ManagementClustersDirName()

	// We initialize repository with the `management-clusters` directory
	// and SOPS configuration file.
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
func NewApp(config AppConfig) (*creator.CreatorConfig, error) {
	var err error

	// Helpers
	wcDir := key.WcDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())
	appDir := key.ResourcePath(appsDir, config.Name)
	appsKusFile := key.ResourcePath(appsDir, key.KustomizationFileName())

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

	// Once files are added, we then need to add resources to the
	// `apps/kustomization.yaml`, either one by one when no base is
	// used, or as a whole directory when it is used.
	resources := make([]string, 0)
	if config.Base == "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.Name, key.AppCRFileName()))
	} else {
		resources = append(resources, config.Name)
	}

	if config.Base == "" && config.UserValuesConfigMap != "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.Name, key.ConfigMapFileName()))
	}

	if config.Base == "" && config.UserValuesSecret != "" {
		resources = append(resources, fmt.Sprintf("%s/%s", config.Name, key.SecretFileName()))
	}

	// Create Kustomization post modifiers
	fsModifiers := map[string]creator.Modifier{
		appsKusFile: creator.KustomizationModifier{
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
func NewAutomaticUpdate(config AutomaticUpdateConfig) (*creator.CreatorConfig, error) {
	var err error

	// bunch of helpers
	wcDir := key.WcDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	autoUpdatesDir := key.ResourcePath(wcDir, key.AutoUpdatesDirName())
	appsDir := key.ResourcePath(wcDir, key.AppsDirName())
	appDir := key.ResourcePath(appsDir, config.App)
	appCrFile := key.ResourcePath(appDir, key.AppCRFileName())
	appsKusFile := key.ResourcePath(appsDir, key.KustomizationFileName())

	// We start from the `WC_NAME` directory, because we must create the
	// `automatic-updates` directory there if it does not exist.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(autoUpdatesDir, nil),
	}

	err = appendFromTemplate(&fsObjects, autoUpdatesDir, updatetmpl.GetAutomaticUpdatesTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Next we add `Image*` resources to the `APP_NAME` directory
	err = appendFromTemplate(&fsObjects, appDir, updatetmpl.GetAppImageUpdatesTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Once files are added, we then need to add resources to the
	// `apps/kustomization.yaml`, and also update App CR with the
	// automatic updates comments
	resources := []string{
		fmt.Sprintf("%s/%s", config.App, key.ImagePolicyFileName()),
		fmt.Sprintf("%s/%s", config.App, key.ImageRepositoryFileName()),
	}

	// Create Kustomization and App CR post modifiers
	fsModifiers := map[string]creator.Modifier{
		appsKusFile: creator.KustomizationModifier{
			ResourcesToAdd: resources,
		},
		appCrFile: creator.AppModifier{
			ImagePolicy: fmt.Sprintf("%s-%s", config.WorkloadCluster, config.App),
		},
	}

	fsValidators := map[string]creator.Validator{
		key.ResourcePath(appDir, key.KustomizationFileName()): creator.KustomizationValidator{
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

// NewManagementCluster creates a new Management Cluster directory
// structure.
func NewManagementCluster(config McConfig) (*creator.CreatorConfig, error) {
	var err error

	mcDir := key.McDirPath(config.Name)
	secretsDir := key.ResourcePath(mcDir, key.SecretsDirName())
	sopsDir := key.ResourcePath(mcDir, key.SopsKeysDirName())
	orgsDir := key.ResourcePath(mcDir, key.OrganizationsDirName())

	// Adding a new Management Cluster is simple. We start at the
	// `management-clusters/MC_NAME` and then add definition and few
	// of the basic directories.
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
func NewOrganization(config OrgConfig) (*creator.CreatorConfig, error) {
	var err error

	orgDir := key.OrgDirPath(config.ManagementCluster, config.Name)
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
func NewWorkloadCluster(config WcConfig) (*creator.CreatorConfig, error) {
	var err error

	// Helpers
	orgDir := key.OrgDirPath(config.ManagementCluster, config.Organization)
	wcsDir := key.ResourcePath(orgDir, key.WorkloadClustersDirName())
	wcDir := key.WcDirPath(config.ManagementCluster, config.Organization, config.Name)

	// We start at the `workload-clusters` directory. This should already
	// exist at this point, as a result of Organization creation, but we
	// need to point to this directory anyway in order to drop Kustomization
	// there.
	fsObjects := []*creator.FsObject{creator.NewFsObject(wcsDir, nil)}

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
			creator.NewFsObject(key.ResourcePath(wcDir, key.AppsDirName()), nil),
			creator.NewFsObject(key.ResourcePath(wcDir, key.ClusterDirName()), nil),
		}...,
	)

	// The `apps/*` directory pre-configuration including kustomization.yaml and
	// kubeconfig patch.
	err = appendFromTemplate(
		&fsObjects,
		key.ResourcePath(wcDir, key.AppsDirName()),
		wctmpl.GetAppsDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// The `cluster/*` files, aka cluster definition, including `kustomization.yaml`,
	// patches, etc.
	err = appendFromTemplate(
		&fsObjects,
		key.ResourcePath(wcDir, key.ClusterDirName()),
		wctmpl.GetClusterDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// After creating all the files and directories, we need creator to run
	// post modifiers, so that cluster is included into `workload-clusters/kustomization.yaml`
	// for example.
	fsModifiers := map[string]creator.Modifier{
		key.ResourcePath(wcsDir, key.KustomizationFileName()): creator.KustomizationModifier{
			ResourcesToAdd: []string{
				fmt.Sprintf("%s.yaml", config.Name),
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

		// Instead of conditioning in the template package and not returning an
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
