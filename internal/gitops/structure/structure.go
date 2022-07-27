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
	wctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/workload-cluster"
)

// NewManagementCluster creates a new App directory structure.
func NewApp(config AppConfig) ([]*creator.FsObject, map[string]creator.Modifier, error) {
	var err error

	// We start from the `apps` directory despite the fact this directory
	// should already exist at this point. We then create the app directory
	// and add bunch of files there, depending on the configuration provided.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(key.DirectoryClusterApps, nil),
		creator.NewFsObject(key.GetWcAppDir(config.Name), nil),
	}

	fileObjects, err := addFilesFromTemplate(
		key.GetWcAppDir(config.Name),
		apptmpl.GetAppDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, fileObjects...)

	// Once files are added, we then need to add resources to the
	// `apps/kustomization.yaml`, either one by one, or the whole
	// directory.
	resources := make([]string, 0)
	if config.Base == "" {
		resources = append(resources, fmt.Sprintf("%s/appcr.yaml # aaa", config.Name))
	} else {
		resources = append(resources, config.Name)
	}

	if config.Base == "" && config.UserValuesConfigMap != "" {
		resources = append(resources, fmt.Sprintf("%s/configmap.yaml", config.Name))
	}

	if config.Base == "" && config.UserValuesSecret != "" {
		resources = append(resources, fmt.Sprintf("%s/secret.yaml", config.Name))
	}

	// Create Kustomization post modifiers
	mods := map[string]creator.Modifier{
		key.GetWcAppsKustomizationFile(): creator.KustomizationModifier{
			ResourcesToAdd: resources,
		},
	}

	return fsObjects, mods, nil
}

// NewImageUpdate configures app for automated updates
func NewAutomaticUpdate(config AutomaticUpdateConfig) ([]*creator.FsObject, map[string]creator.Modifier, error) {
	var err error

	// We start from the `WC_NAME` directory, because we must create the
	// `automatic-updates` directory.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(key.DirectoryAutomaticUpdates, nil),
	}

	fileObjects, err := addFilesFromTemplate(
		key.DirectoryAutomaticUpdates,
		updatetmpl.GetAutomaticUpdatesTemplates,
		config,
	)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	fsObjects = append(fsObjects, fileObjects...)

	// Next we add `Image*` resources to the `APP_NAME` directory
	fileObjects, err = addFilesFromTemplate(
		key.GetWcAppDir(config.App),
		updatetmpl.GetAppImageUpdatesTemplates,
		config,
	)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	// Once files are added, we then need to add resources to the
	// `apps/kustomization.yaml`, and also update App CR with the
	// automatic updates comments
	resources := []string{
		fmt.Sprintf("%s/imagepolicy.yaml", config.App),
		fmt.Sprintf("%s/imagerepository.yaml", config.App),
	}

	// Create Kustomization and App post modifiers
	mods := map[string]creator.Modifier{
		key.GetWcAppsKustomizationFile(): creator.KustomizationModifier{
			ResourcesToAdd: resources,
		},
		fmt.Sprintf("%s/%s/%s", key.DirectoryClusterApps, config.App, "appcr.yaml"): creator.AppModifier{
			ImagePolicy: fmt.Sprintf("%s-%s", config.WorkloadCluster, config.App),
		},
	}

	return fsObjects, mods, nil
}

// NewManagementCluster creates a new Management Cluster directory
// structure.
func NewManagementCluster(config McConfig) ([]*creator.FsObject, error) {
	var err error

	fsObjects := []*creator.FsObject{
		creator.NewFsObject(config.Name, nil),
	}

	fileObjects, err := addFilesFromTemplate(config.Name, mctmpl.GetManagementClusterTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	fsObjects = append(
		fsObjects,
		[]*creator.FsObject{
			creator.NewFsObject(key.GetAnySecretsDir(config.Name), nil),
			creator.NewFsObject(key.GetMCsSopsDir(config.Name), nil),
			creator.NewFsObject(key.GetMCsOrgsDir(config.Name), nil),
		}...,
	)

	return fsObjects, nil
}

// NewOrganization creates a new Organization directory
// structure.
func NewOrganization(config OrgConfig) ([]*creator.FsObject, error) {
	var err error

	// Create `ORG_NAME` directory and add `ORG_NAME.yaml`manifest
	// containing Organization CR
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(config.Name, nil),
	}

	fileObjects, err := addFilesFromTemplate(
		config.Name,
		orgtmpl.GetOrganizationDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	// Create `workload-cluster` directory and populate it with
	// empty `kustomization.yaml`.
	fsObjects = append(
		fsObjects,
		creator.NewFsObject(key.GetOrgWCsDir(config.Name), nil),
	)

	fileObjects, err = addFilesFromTemplate(
		key.GetOrgWCsDir(config.Name),
		orgtmpl.GetWorkloadClustersDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	return fsObjects, nil
}

// NewWorkloadCluster creates a new Workload Cluster directory
// structure.
func NewWorkloadCluster(config WcConfig) ([]*creator.FsObject, map[string]creator.Modifier, error) {
	var err error

	// Create Dir pointing to the `workload-clusters` directory. This should
	// already exist at this point, as a result of Organization creation, but
	// we need to point to this directory anyway in order to drop Kustomization
	// there.
	fsObjects := []*creator.FsObject{
		creator.NewFsObject(key.DirectoryWorkloadClusters, nil),
	}

	// Add Kustomization CR to the `workload-clusters` directory and other
	// files if needed. Currently only Kustomization CR is considered.
	fileObjects, err := addFilesFromTemplate(
		key.DirectoryWorkloadClusters,
		wctmpl.GetWorkloadClusterDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	// Create `WC_NAME` specific directory and then add `apps` and `cluster`
	// directories there.
	// If base has been specified by the user, then in addition to the above, populate
	// the `cluster` directory with cluster definition, possibly enriching it with
	// user configuration when specified as well.
	fsObjects = append(
		fsObjects,
		[]*creator.FsObject{
			creator.NewFsObject(key.GetOrgWcDir(config.Name), nil),
			creator.NewFsObject(key.GetOrgWcAppsDir(config.Name), nil),
			creator.NewFsObject(key.GetOrgWcClusterDir(config.Name), nil),
		}...,
	)

	// The `apps/*` pre-configuration
	fileObjects, err = addFilesFromTemplate(
		key.GetOrgWcAppsDir(config.Name),
		wctmpl.GetAppsDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	// The `cluster/*` files, aka cluster definition, including `kustomization.yaml`,
	// patches, etc.
	fileObjects, err = addFilesFromTemplate(
		key.GetOrgWcClusterDir(config.Name),
		wctmpl.GetClusterDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	// After creating all the files and directories, we need creator to run
	// post modifiers, so that cluster is included into `workload-clusters/kustomization.yaml`
	// for example.
	mods := map[string]creator.Modifier{
		key.GetOrgWCsKustomizationFile(): creator.KustomizationModifier{
			ResourcesToAdd: []string{
				fmt.Sprintf("%s.yaml", config.Name),
			},
		},
	}

	return fsObjects, mods, nil
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

		if len(content.Bytes()) <= 1 {
			continue
		}

		fsObjects = append(
			fsObjects,
			creator.NewFsObject(
				fmt.Sprintf("%s/%s", path, name.String()),
				content.Bytes(),
			),
		)
	}

	return fsObjects, nil
}
