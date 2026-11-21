package migration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

// protectSecrets gives the management cluster's secrets a kustomization.yaml
// listing every file there, so the management cluster's Kustomization applies
// the same as before, with the `kustomize.toolkit.fluxcd.io/prune: disabled`
// annotation and the `giantswarm.io/prevent-deletion` label, as
// gitops-template's has: Flux decrypts with the keys there.
func protectSecrets(fs *afero.Afero, repoPath, mcPath string, env Env) ([]string, error) {
	dir := filepath.Join(repoPath, mcPath, gitopskey.SecretsDirName())
	path := filepath.Join(dir, gitopskey.SigsKustomizationFileName())

	exists, err := fs.DirExists(dir)
	if err != nil || !exists {
		return nil, microerror.Mask(err)
	}
	exists, err = fs.Exists(path)
	if err != nil || exists {
		return nil, microerror.Mask(err)
	}

	names, err := listDir(fs, dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	resources, err := resourcesOf(fs, repoPath, dir, names, env, "the kustomization.yaml of the management cluster's secrets")
	if err != nil || len(resources) == 0 {
		return nil, microerror.Mask(err)
	}

	err = fs.WriteFile(path, []byte(secretsKustomization(resources)), 0644)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: added, lists and protects %s", rel(repoPath, path), strings.Join(resources, ", "))}, nil
}

func secretsKustomization(resources []string) string {
	var b strings.Builder
	b.WriteString(`apiVersion: kustomize.config.k8s.io/v1beta1
buildMetadata: [originAnnotations]
kind: Kustomization
# Flux decrypts with the keys here: deleting one stops GitOps for what it
# decrypts.
commonAnnotations:
  kustomize.toolkit.fluxcd.io/prune: disabled
labels:
  - pairs:
      giantswarm.io/managed-by: flux
      # nobody can delete them, Flux included, until this label is removed
      giantswarm.io/prevent-deletion: "true"
resources:
`)
	for _, r := range resources {
		fmt.Fprintf(&b, "  - %s\n", r)
	}

	return b.String()
}
