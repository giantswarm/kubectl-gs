package common

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

const mapiCluster = "workload cluster `%s` still keeps its manifests in `%s`, which Flux applies instead of what this command writes. Run `kubectl gs gitops upgrade` to move them out of `%s` first."

// RefuseMapiCluster returns the pre-validator, keyed by the workload cluster's
// directory, that refuses to write into a cluster that still keeps its
// manifests in `WC_NAME/mapi`. The cluster's Kustomization applies `mapi`, so
// Flux would never apply what is written next to it. Keyed by the cluster's
// directory, it runs before the checks on what the directory holds.
func RefuseMapiCluster(config StructureConfig) (string, func(*afero.Afero, string) error) {
	wcDir := key.BaseDirPath(config.ManagementCluster, config.Organization, config.WorkloadCluster)
	mapiDir := key.ResourcePath(wcDir, key.MapiDirName())

	return wcDir, func(fs *afero.Afero, path string) error {
		exists, err := fs.DirExists(key.ResourcePath(path, key.MapiDirName()))
		if err != nil {
			return microerror.Mask(err)
		}
		if !exists {
			return nil
		}

		return microerror.Maskf(creator.ValidationError, mapiCluster, config.WorkloadCluster, mapiDir, key.MapiDirName())
	}
}
