package common

import (
	"path/filepath"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

const (
	missingNamespace = "%s does not exist. What the organisation keeps in its namespace, its workload clusters' Kustomizations and keys among them, is applied by the Kustomization in namespace.yaml, from namespace/, which `kubectl gs gitops add organization` writes. An organisation created with an older kubectl-gs needs them added first, see " + namespaceDocs

	namespaceDocs = "https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md#move-the-workload-cluster-kustomizations"
)

// RequireNamespace is a creator pre-validator for the path of an
// organisation's namespace.yaml. It refuses an organisation without its
// namespace Kustomization, or the namespace/ directory it applies.
func RequireNamespace(fs *afero.Afero, path string) error {
	for _, p := range []string{path, filepath.Join(filepath.Dir(path), key.NamespaceDirName(), key.SigsKustomizationFileName())} {
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
