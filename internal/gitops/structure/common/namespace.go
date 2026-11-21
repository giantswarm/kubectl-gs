package common

import (
	"path/filepath"
	"slices"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

const (
	missingNamespace = "%s does not exist. What the organisation keeps in its namespace, its workload clusters' Kustomizations and keys among them, is applied by the Kustomization in namespace.yaml, from namespace/, which `kubectl gs gitops add organization` writes. For an organisation created with an older kubectl-gs, run `kubectl gs gitops upgrade` first"

	clustersNotMoved = "%s doesn't list ../workload-clusters: the organisation's workload clusters' Kustomizations haven't moved into its namespace yet, and kubectl-gs writes clusters and their keys for that namespace. Move them first, see " + moveDocs

	moveDocs = "https://github.com/giantswarm/gitops-template/blob/main/docs/how-to/upgrade-fork.md#move-the-workload-cluster-kustomizations"
)

// RequireNamespace is a creator pre-validator for the path of an
// organisation's namespace.yaml. It refuses an organisation without its
// namespace Kustomization, or the namespace/ directory it applies.
func RequireNamespace(fs *afero.Afero, path string) error {
	for _, p := range []string{path, namespaceKustomization(path)} {
		ok, err := fs.Exists(p)
		if err != nil {
			return microerror.Mask(err)
		}
		if !ok {
			return microerror.Maskf(creator.ValidationError, missingNamespace, p)
		}
	}

	return nil
}

// RequireMovedClusters is a creator pre-validator for the path of an
// organisation's namespace.yaml. On top of RequireNamespace, it refuses an
// organisation whose workload clusters' Kustomizations haven't moved into
// its namespace, as `kubectl gs gitops upgrade` leaves them.
func RequireMovedClusters(fs *afero.Afero, path string) error {
	err := RequireNamespace(fs, path)
	if err != nil {
		return microerror.Mask(err)
	}

	data, err := fs.ReadFile(namespaceKustomization(path))
	if err != nil {
		return microerror.Mask(err)
	}

	var kustomization struct {
		Resources []string `json:"resources"`
	}
	err = yaml.Unmarshal(data, &kustomization)
	if err != nil {
		return microerror.Mask(err)
	}
	if !slices.Contains(kustomization.Resources, "../"+key.WorkloadClustersDirName()) {
		return microerror.Maskf(creator.ValidationError, clustersNotMoved, namespaceKustomization(path))
	}

	return nil
}

func namespaceKustomization(namespaceFile string) string {
	return filepath.Join(filepath.Dir(namespaceFile), key.NamespaceDirName(), key.SigsKustomizationFileName())
}
