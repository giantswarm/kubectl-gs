package metadata

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

// AdoptionResult reports what walking an existing repository turned up.
type AdoptionResult struct {
	Metadata *RepositoryMetadata

	// Skipped lists directories that sit where a layer was expected but that
	// do not look like one. They are reported rather than swallowed, because
	// the repository structure allows layouts this walk does not recognise:
	// `organizations/` and `workload-clusters/` are optional in
	// gitops-template's repo_structure.md, so a flatter repository is legal
	// and will simply not be adopted.
	Skipped []string
}

// Adopt infers the layers of a repository that predates the metadata file, and
// pins them at the current structure version.
//
// Note what this cannot do: it has no way of knowing which structure version
// the repository was actually generated with, so it assumes the current one.
// Drift that already exists at the point of adoption stays invisible; only
// drift introduced afterwards is detected.
func Adopt(fs *afero.Afero, repoPath string) (*AdoptionResult, error) {
	result := &AdoptionResult{
		Metadata: New(),
		Skipped:  []string{},
	}

	err := adoptClusterBases(fs, repoPath, result)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = adoptManagementClusters(fs, repoPath, result)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	result.Metadata.Sort()
	sort.Strings(result.Skipped)

	return result, nil
}

// adoptClusterBases records `bases/clusters/PROVIDER/template` directories,
// as created by `gitops add base`.
func adoptClusterBases(fs *afero.Afero, repoPath string, result *AdoptionResult) error {
	providers, err := subDirs(fs, repoPath, key.ClusterBaseTemplatesPath())
	if err != nil {
		return microerror.Mask(err)
	}

	for _, provider := range providers {
		basePath := key.ClusterBasePath(provider)

		exists, err := dirExists(fs, filepath.Join(repoPath, basePath))
		if err != nil {
			return microerror.Mask(err)
		}
		if !exists {
			result.Skipped = append(result.Skipped, key.ClusterBaseProviderPath(provider))
			continue
		}

		result.Metadata.Upsert(Layer{Kind: LayerClusterBase, Path: basePath})
	}

	return nil
}

// adoptManagementClusters walks `management-clusters/*` and everything below it.
func adoptManagementClusters(fs *afero.Afero, repoPath string, result *AdoptionResult) error {
	mcs, err := subDirs(fs, repoPath, key.ManagementClustersDirName())
	if err != nil {
		return microerror.Mask(err)
	}

	for _, mc := range mcs {
		result.Metadata.Upsert(Layer{Kind: LayerManagementCluster, Path: key.BaseDirPath(mc, "", "")})

		// Apps are only walked under workload clusters: `gitops add app`
		// requires both --organization and --workload-cluster, so an `apps`
		// directory anywhere else was not created by it.
		err = adoptOrganizations(fs, repoPath, mc, result)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// adoptOrganizations walks `management-clusters/MC/organizations/*`.
func adoptOrganizations(fs *afero.Afero, repoPath, mc string, result *AdoptionResult) error {
	orgsDir := key.ResourcePath(key.BaseDirPath(mc, "", ""), key.OrganizationsDirName())

	orgs, err := subDirs(fs, repoPath, orgsDir)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, org := range orgs {
		result.Metadata.Upsert(Layer{Kind: LayerOrganization, Path: key.BaseDirPath(mc, org, "")})

		err = adoptWorkloadClusters(fs, repoPath, mc, org, result)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// adoptWorkloadClusters walks
// `management-clusters/MC/organizations/ORG/workload-clusters/*`.
func adoptWorkloadClusters(fs *afero.Afero, repoPath, mc, org string, result *AdoptionResult) error {
	wcsDir := key.ResourcePath(key.BaseDirPath(mc, org, ""), key.WorkloadClustersDirName())

	wcs, err := subDirs(fs, repoPath, wcsDir)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, wc := range wcs {
		wcDir := key.BaseDirPath(mc, org, wc)
		result.Metadata.Upsert(Layer{Kind: LayerWorkloadCluster, Path: wcDir})

		// Apps live under `WC_NAME/mapi/apps` when the mapi directory is in
		// use, and under `WC_NAME/apps` when the cluster was added with
		// `--skip-mapi`. Both are checked, so adoption does not depend on
		// knowing which flag was used at creation time.
		err = adoptApps(fs, repoPath, key.ResourcePath(wcDir, key.MapiDirName()), result)
		if err != nil {
			return microerror.Mask(err)
		}

		err = adoptApps(fs, repoPath, wcDir, result)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// adoptApps records the `apps/*` directories holding an App CR under the given
// parent directory.
func adoptApps(fs *afero.Afero, repoPath, parentDir string, result *AdoptionResult) error {
	appsDir := key.ResourcePath(parentDir, key.AppsDirName())

	apps, err := subDirs(fs, repoPath, appsDir)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, app := range apps {
		appDir := key.ResourcePath(appsDir, app)

		// `gitops add app` writes an App CR when no base is used, and a
		// Kustomization referencing the base when one is. The two are mutually
		// exclusive, so either marks the directory as a layer it created.
		isApp := false
		for _, marker := range []string{key.AppCRFileName(), key.SigsKustomizationFileName()} {
			exists, err := fs.Exists(filepath.Join(repoPath, appDir, marker))
			if err != nil {
				return microerror.Mask(err)
			}
			if exists {
				isApp = true
				break
			}
		}

		if !isApp {
			result.Skipped = append(result.Skipped, appDir)
			continue
		}

		result.Metadata.Upsert(Layer{Kind: LayerApp, Path: appDir})
	}

	return nil
}

// subDirs lists the names of the directories directly under dir, relative to
// the repository root. A missing directory yields no names and no error, so
// that callers can probe optional parts of the structure.
func subDirs(fs *afero.Afero, repoPath, dir string) ([]string, error) {
	path := filepath.Join(repoPath, dir)

	exists, err := dirExists(fs, path)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !exists {
		return nil, nil
	}

	entries, err := fs.ReadDir(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	names := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	return names, nil
}

func dirExists(fs *afero.Afero, path string) (bool, error) {
	info, err := fs.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return info.IsDir(), nil
}
