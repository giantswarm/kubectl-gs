package migration

import (
	"regexp"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// moveDocs describes the moves files alone can't make safely.
const moveDocs = "https://github.com/giantswarm/gitops-template/blob/main/docs/how-to/upgrade-fork.md#move-the-workload-cluster-kustomizations"

// version3 drops the `mapi` directory of workload clusters, and gives each
// organisation a Kustomization of its own for its namespace.
//
// It gates the clusters' Kustomizations on their control plane, where they
// still have the health check kubectl-gs wrote before, and gives the
// management cluster's Kustomization the prune: disabled annotation, so an
// upgraded repository is as protected as a newly generated one.
//
// It also gives the management cluster's secrets a kustomization.yaml that
// lists and protects every key there, which the move of the keys of the
// workload clusters needs, as the management cluster's Kustomization prunes
// by then.
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
	Description: "Workload clusters drop the mapi directory, list what their Kustomizations apply and gate them on the control plane, each organisation gets a Kustomization for its namespace, and the management cluster's Kustomization is never pruned.",
	Kinds:       []string{metadata.LayerManagementCluster, metadata.LayerOrganization, metadata.LayerWorkloadCluster},
	Apply:       applyVersion3,
	Relocate:    relocateMapi,
	ManualSteps: version3ManualSteps,
}

const version3ManualSteps = `The workload clusters' Kustomizations, and their keys in MC_NAME/secrets, are
still where they were: in ` + "`default`" + `, applied by the management cluster's
Kustomization. Move them into each organisation's namespace with the
three commits of "Move the workload cluster Kustomizations", and let Flux
apply each commit before you push the next:

` + moveDocs + `

It needs MC_NAME.yaml to have ` + "`prune: true`" + ` by its second commit, so that Flux
deletes the old Kustomizations in ` + "`default`" + `. Its first commit sets it. Don't
move the Kustomizations in one commit: Flux would delete the old ones, and
every app on the workload clusters with them, before the new ones exist.

Until then, ` + "`kubectl gs gitops add workload-cluster`" + ` and
` + "`kubectl gs gitops add encryption --workload-cluster`" + ` refuse these
organisations.`

func applyVersion3(fs *afero.Afero, repoPath string, layer metadata.Layer, env Env) ([]string, error) {
	switch layer.Kind {
	case metadata.LayerWorkloadCluster:
		return upgradeWorkloadCluster(fs, repoPath, layer.Path, env)
	case metadata.LayerOrganization:
		changes, err := addNamespace(fs, repoPath, layer.Path, env)
		return changes, microerror.Mask(err)
	case metadata.LayerManagementCluster:
		return upgradeManagementCluster(fs, repoPath, layer.Path, env)
	}

	return nil, nil
}

// upgradeWorkloadCluster moves the cluster out of `mapi`, and gates its
// Kustomization on its control plane, as a newly generated one is.
func upgradeWorkloadCluster(fs *afero.Afero, repoPath, wcPath string, env Env) ([]string, error) {
	changes, err := migrateLayout(fs, repoPath, wcPath, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	gated, err := gateControlPlane(fs, repoPath, wcPath, env)

	return append(changes, gated...), microerror.Mask(err)
}

// upgradeManagementCluster protects the management cluster's keys and its
// own Kustomization, as a newly generated management cluster is.
func upgradeManagementCluster(fs *afero.Afero, repoPath, mcPath string, env Env) ([]string, error) {
	changes, err := protectSecrets(fs, repoPath, mcPath, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	protected, err := protectManagementCluster(fs, repoPath, mcPath, env)

	return append(changes, protected...), microerror.Mask(err)
}

var mapiPath = regexp.MustCompile(`^(.*/workload-clusters/[^/]+)/mapi(/.*)?$`)

// relocateMapi follows the layers in a workload cluster's `mapi` directory,
// its apps, up into the cluster's directory.
func relocateMapi(path string) string {
	return mapiPath.ReplaceAllString(path, "$1$2")
}
