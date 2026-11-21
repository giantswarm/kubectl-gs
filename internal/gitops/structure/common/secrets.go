package common

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
)

const incompleteKustomization = "%s does not exist, and %s holds %s. A kustomization.yaml listing only %s would take the other files out of what Flux applies, and Flux would delete them. Add a kustomization.yaml that lists every file there first, with the `kustomize.toolkit.fluxcd.io/prune: disabled` annotation and the `giantswarm.io/prevent-deletion` label, as gitops-template's management-clusters/MC_NAME/secrets/kustomization.yaml has"

// RequireCompleteKustomization returns a creator pre-validator for the path
// of a kustomization.yaml the run creates listing only file. It refuses when
// the kustomization.yaml doesn't exist yet and its directory holds other
// files, which Flux applies now, as it generates a kustomization.yaml for a
// directory without one, and would delete once they are no longer listed.
func RequireCompleteKustomization(file string) func(*afero.Afero, string) error {
	return func(fs *afero.Afero, path string) error {
		exists, err := fs.Exists(path)
		if err != nil || exists {
			return microerror.Mask(err)
		}

		dir := filepath.Dir(path)
		entries, err := fs.ReadDir(dir)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		for _, e := range entries {
			if e.Name() != file {
				return microerror.Maskf(creator.ValidationError, incompleteKustomization, path, dir, e.Name(), file)
			}
		}

		return nil
	}
}
