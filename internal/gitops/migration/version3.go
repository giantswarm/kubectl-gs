package migration

import (
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// namespaceDocs describes the moves files alone can't make safely.
const namespaceDocs = "https://github.com/giantswarm/gitops-template/blob/main/docs/reference/repo-structure.md#move-the-workload-cluster-kustomizations"

// version3 drops the `mapi` directory of workload clusters, and gives each
// organisation a Kustomization of its own for its namespace.
//
// It makes the changes that are safe to make in files and push at once: the
// clusters' Kustomizations apply the same objects after it as before, but
// for what was in `out-of-band`, and the new namespace Kustomization applies
// nothing yet. Moving the clusters' Kustomizations into the organisation's
// namespace isn't: Flux would delete the old ones, and what they applied,
// before the new ones exist, unless `deletionPolicy: Orphan` is applied
// first, in a commit of its own. Their keys have to be copied with sops. So
// that is left to the user, as ManualSteps says.
var version3 = Migration{
	From:        2,
	Description: "Workload clusters drop the mapi directory and list what their Kustomizations apply, and each organisation gets a Kustomization for its namespace.",
	Kinds:       []string{metadata.LayerOrganization, metadata.LayerWorkloadCluster},
	Apply:       applyVersion3,
	Relocate:    relocateMapi,
	ManualSteps: version3ManualSteps,
}

const version3ManualSteps = `The workload clusters' Kustomizations are still in ` + "`default`" + `. Moving them into
the organisation's namespace, where its namespace Kustomization applies them,
can't be done safely in one commit: Flux would delete the old ones, and with
them every app on the clusters, before the new ones exist. For each
organisation, use two commits, and let Flux apply the first before you push
the second:

1. Add ` + "`deletionPolicy: Orphan`" + ` to the clusters' Kustomizations in
   workload-clusters/. Add the .sops.yaml rule for the clusters' keys in the
   organisations' namespace/secrets, ahead of the other rules, and copy each key
   Secret the Kustomizations decrypt with from MC_NAME/secrets into the
   organisation's namespace/secrets, in the organisation's namespace, with sops.
2. List ../workload-clusters in namespace/kustomization.yaml instead of the
   organisation's kustomization.yaml, and change the Kustomizations: no
   namespace, ` + "`sourceRef.namespace: default`" + `, ` + "`organization: \"${organization}\"`" + `,
   the organisation's namespace Kustomization in ` + "`dependsOn`" + `, the namespace
   of a Cluster health check ` + "`org-${organization}`" + `, and no ` + "`deletionPolicy`" + `.

Until then, don't add workload clusters to these organisations: kubectl-gs
writes them for the organisation's namespace. The steps, with the commands:
` + namespaceDocs

func applyVersion3(fs *afero.Afero, repoPath string, layer metadata.Layer) ([]string, error) {
	env := EnvOf(fs)

	switch layer.Kind {
	case metadata.LayerWorkloadCluster:
		changes, err := migrateLayout(fs, repoPath, layer.Path, env)
		return changes, microerror.Mask(err)
	case metadata.LayerOrganization:
		changes, err := addNamespace(fs, repoPath, layer.Path, env)
		return changes, microerror.Mask(err)
	}

	return nil, nil
}

var mapiPath = regexp.MustCompile(`^(.*/workload-clusters/[^/]+)/mapi(/.*)?$`)

// relocateMapi follows the layers in a workload cluster's `mapi` directory,
// its apps, up into the cluster's directory.
func relocateMapi(path string) string {
	return mapiPath.ReplaceAllString(path, "$1$2")
}
