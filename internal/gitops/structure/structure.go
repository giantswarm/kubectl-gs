package structure

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem"
	"github.com/giantswarm/kubectl-gs/internal/gitops/key"
	mctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/management-cluster"
	orgtmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/organization"
	wctmpl "github.com/giantswarm/kubectl-gs/internal/gitops/structure/templates/workload-cluster"
)

func NewManagementCluster(config McConfig) ([]*filesystem.FsObject, error) {
	var err error

	fsObjects := []*filesystem.FsObject{
		&filesystem.FsObject{
			RelativePath: config.Name,
		},
	}

	fileObjects, err := addFilesFromTemplate(config.Name, mctmpl.GetManagementClusterTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	fsObjects = append(
		fsObjects,
		[]*filesystem.FsObject{
			&filesystem.FsObject{
				RelativePath: fmt.Sprintf("%s/%s", config.Name, key.DirectorySecrets),
			},
			&filesystem.FsObject{
				RelativePath: fmt.Sprintf("%s/%s", config.Name, key.DirectorySOPSPublicKeys),
			},
			&filesystem.FsObject{
				RelativePath: fmt.Sprintf("%s/%s", config.Name, key.DirectoryOrganizations),
			},
		}...,
	)

	return fsObjects, nil
}

func NewOrganization(config OrgConfig) ([]*filesystem.FsObject, error) {
	var err error

	// Create `ORG_NAME` directory and add `ORG_NAME.yaml`manifest
	// containing Organization CR
	fsObjects := []*filesystem.FsObject{
		&filesystem.FsObject{
			RelativePath: config.Name,
		},
	}

	fileObjects, err := addFilesFromTemplate(config.Name, orgtmpl.GetOrganizationDirectoryTemplates, config)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	// Create `workload-cluster` directory and populate it with
	// empty `kustomization.yaml`.
	fsObjects = append(
		fsObjects,
		&filesystem.FsObject{
			RelativePath: fmt.Sprintf("%s/%s", config.Name, key.DirectoryWorkloadClusters),
		},
	)

	fileObjects, err = addFilesFromTemplate(
		fmt.Sprintf("%s/%s", config.Name, key.DirectoryWorkloadClusters),
		orgtmpl.GetWorkloadClustersDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	return fsObjects, nil
}

func NewWorkloadCluster(config WcConfig) ([]*filesystem.FsObject, error) {
	var err error

	// Create Dir pointing to the `workload-clusters` directory. This should
	// already exist at this point, as a result of Organization creation, but
	// we need to point to this directory anyway in order to drop Kustomization
	// there.
	fsObjects := []*filesystem.FsObject{
		&filesystem.FsObject{
			RelativePath: key.DirectoryWorkloadClusters,
		},
	}

	fileObjects, err := addFilesFromTemplate(
		key.DirectoryWorkloadClusters,
		wctmpl.GetWorkloadClusterDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	// Create `WC_NAME` specific directory and then add `apps` and `cluster`
	// directories there.
	// If base has been specified by the user, then in addition to the above, populate
	// the `cluster` directory with cluster definition, possibly enriching it with
	// user configuration when specified as well.
	fsObjects = append(
		fsObjects,
		[]*filesystem.FsObject{
			&filesystem.FsObject{
				RelativePath: fmt.Sprintf("%s/%s", key.DirectoryWorkloadClusters, config.Name),
			},
			&filesystem.FsObject{
				RelativePath: fmt.Sprintf("%s/%s/%s", key.DirectoryWorkloadClusters, config.Name, key.DirectoryClusterApps),
			},
			&filesystem.FsObject{
				RelativePath: fmt.Sprintf("%s/%s/%s", key.DirectoryWorkloadClusters, config.Name, key.DirectoryClusterDefinition),
			},
		}...,
	)

	// The `cluster/*` files, aka cluster definition
	fileObjects, err = addFilesFromTemplate(
		fmt.Sprintf("%s/%s/%s", key.DirectoryWorkloadClusters, config.Name, key.DirectoryClusterDefinition),
		wctmpl.GetClusterDirectoryTemplates,
		config,
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fsObjects = append(fsObjects, fileObjects...)

	return fsObjects, nil
}

func addFilesFromTemplate(path string, templates func() map[string]string, config interface{}) ([]*filesystem.FsObject, error) {
	var err error

	fsObjects := make([]*filesystem.FsObject, 0)
	for n, t := range templates() {
		nameTemplate := template.Must(template.New("name").Parse(n))
		var name bytes.Buffer
		err = nameTemplate.Execute(&name, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		contentTemplate := template.Must(template.New("files").Funcs(sprig.TxtFuncMap()).Parse(t))

		var content bytes.Buffer
		err = contentTemplate.Execute(&content, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		fsObjects = append(
			fsObjects,
			&filesystem.FsObject{
				RelativePath: fmt.Sprintf("%s/%s.yaml", path, name.String()),
				Data:         content.Bytes(),
			},
		)
	}

	return fsObjects, nil
}
