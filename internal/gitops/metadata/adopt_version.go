package metadata

import (
	"path/filepath"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

// lastMapiVersion is the last structure version whose workload clusters had
// a `mapi` directory, or no `kustomization.yaml` of their own, so that Flux
// applied all of the cluster's directory, `out-of-band` included.
const lastMapiVersion = 2

// workloadClusterVersion returns the structure version to record a workload
// cluster at, from its layout and from what it deploys.
func workloadClusterVersion(fs *afero.Afero, repoPath, wcDir string) (int, error) {
	mapiDir := key.ResourcePath(wcDir, key.MapiDirName())

	version, err := adoptedVersion(fs, repoPath,
		key.ResourcePath(wcDir, key.ClusterDirName()),
		key.ResourcePath(wcDir, key.AppsDirName()),
		key.ResourcePath(mapiDir, key.ClusterDirName()),
		key.ResourcePath(mapiDir, key.AppsDirName()),
	)
	if err != nil || version < lastMapiVersion {
		return version, microerror.Mask(err)
	}

	layout, err := layoutVersion(fs, filepath.Join(repoPath, wcDir))
	if err != nil {
		return 0, microerror.Mask(err)
	}

	return min(version, layout), nil
}

// layoutVersion returns lastMapiVersion for a workload cluster directory
// with a `mapi` directory, or with `cluster` or `apps` but no
// `kustomization.yaml`, and the current version otherwise.
func layoutVersion(fs *afero.Afero, dir string) (int, error) {
	for _, d := range []string{key.MapiDirName(), key.ClusterDirName(), key.AppsDirName()} {
		exists, err := dirExists(fs, filepath.Join(dir, d))
		if err != nil {
			return 0, microerror.Mask(err)
		}
		if !exists {
			continue
		}
		if d == key.MapiDirName() {
			return lastMapiVersion, nil
		}

		exists, err = fs.Exists(filepath.Join(dir, key.SigsKustomizationFileName()))
		if err != nil {
			return 0, microerror.Mask(err)
		}
		if !exists {
			return lastMapiVersion, nil
		}
	}

	return StructureVersion, nil
}

// managementClusterVersion returns lastMapiVersion for a management cluster
// whose secrets/ holds files but no kustomization.yaml, which structure version 3
// brought to protect its keys, and the current version otherwise.
func managementClusterVersion(fs *afero.Afero, repoPath, mcDir string) (int, error) {
	secretsDir := filepath.Join(repoPath, mcDir, key.SecretsDirName())

	empty, err := isEmptyDir(fs, secretsDir)
	if err != nil || empty {
		return StructureVersion, microerror.Mask(err)
	}

	exists, err := fs.Exists(filepath.Join(secretsDir, key.SigsKustomizationFileName()))
	if err != nil {
		return 0, microerror.Mask(err)
	}
	if !exists {
		return lastMapiVersion, nil
	}

	return StructureVersion, nil
}

// organizationVersion returns lastMapiVersion for an organisation without
// its namespace Kustomization, which structure version 3 brought, and the
// current version otherwise.
func organizationVersion(fs *afero.Afero, repoPath, orgDir string) (int, error) {
	exists, err := fs.Exists(filepath.Join(repoPath, orgDir, key.NamespaceFileName()))
	if err != nil {
		return 0, microerror.Mask(err)
	}
	if !exists {
		return lastMapiVersion, nil
	}

	return StructureVersion, nil
}

// isEmptyDir reports whether dir is missing or holds nothing.
func isEmptyDir(fs *afero.Afero, dir string) (bool, error) {
	exists, err := dirExists(fs, dir)
	if err != nil || !exists {
		return true, microerror.Mask(err)
	}

	empty, err := fs.IsEmpty(dir)

	return empty, microerror.Mask(err)
}
