package migration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"

	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

const (
	pruneAnnotationKey = "kustomize.toolkit.fluxcd.io/prune"
	pruneDisabled      = "disabled"

	pruneComment = `It applies itself, so renaming it would make it prune itself, and its
finalizer would delete everything it applied. The annotation leaves it, as
it does each organisation's namespace Kustomization.`

	// availableGate is the health check kubectl-gs wrote before it gated a
	// cluster's Kustomization on the control plane.
	availableGate = "status.conditions.exists(c, c.type == 'Available' && c.status == 'True')"

	// controlPlaneGate is the health check gitops-template has.
	controlPlaneGate = "status.conditions.exists(c, c.type == 'ControlPlaneAvailable' && c.status == 'True')" +
		" && status.conditions.exists(c, c.type == 'RemoteConnectionProbe' && c.status == 'True')"

	gateComment = `Applies the cluster and its base apps, in cluster/. Ready once Cluster API
reports the cluster's control plane available and reachable, which is all
the apps need. The Kustomization in %s-apps.yaml depends on this one, so
the secondary apps install once the cluster is up. It doesn't wait for
` + "`Available`" + `, which counts every node pool: one node pool short of its
minimum would hold back every change to the apps.`
)

// editDocument changes a document in place, and returns what to report, and
// whether it changed it. An empty report says nothing.
type editDocument func(object *yaml.Node) (string, bool)

// editFluxKustomizations applies edit to every Flux Kustomization of the file
// that applies dir or a directory under it, document by document, keeping
// every other document byte for byte, as fixFluxPath does.
func editFluxKustomizations(fs *afero.Afero, repoPath, file, dir string, edit editDocument) ([]string, error) {
	data, err := fs.ReadFile(file)
	if err != nil || !strings.Contains(string(data), dir) {
		return nil, microerror.Mask(err)
	}

	documents, err := splitDocuments(data, rel(repoPath, file))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var out []byte
	var changes []string
	changed := false
	for _, d := range documents {
		report, edited := "", false
		if appliesDir(d.node, dir) {
			report, edited = edit(d.node)
		}
		if report != "" {
			changes = append(changes, fmt.Sprintf("%s: %s", rel(repoPath, file), report))
		}
		if !edited {
			out = append(out, d.bytes()...)
			continue
		}

		rewritten, err := d.rewrite(rel(repoPath, file))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		out = append(out, rewritten...)
		changed = true
	}

	if !changed {
		return changes, nil
	}

	return changes, microerror.Mask(fs.WriteFile(file, out, 0644))
}

// appliesDir reports whether the object is a Flux Kustomization applying dir
// or a directory under it.
func appliesDir(object *yaml.Node, dir string) bool {
	if object == nil {
		return false
	}
	path := fluxPath(object)
	if path == nil {
		return false
	}
	current := strings.TrimPrefix(path.Value, "./")

	return current == dir || strings.HasPrefix(current, dir+"/")
}

// protectManagementCluster gives the management cluster's Kustomization the
// prune: disabled annotation a newly generated one has, unless it has a
// prune annotation already.
func protectManagementCluster(fs *afero.Afero, repoPath, mcPath string) ([]string, error) {
	file := filepath.Join(repoPath, mcPath, gitopskey.FluxKustomizationFileName(filepath.Base(mcPath)))

	exists, err := fs.Exists(file)
	if err != nil || !exists {
		return nil, microerror.Mask(err)
	}

	changes, err := editFluxKustomizations(fs, repoPath, file, filepath.ToSlash(mcPath), disablePruning)

	return changes, microerror.Mask(err)
}

func disablePruning(object *yaml.Node) (string, bool) {
	metadata := mapValue(object, "metadata")
	if metadata == nil {
		return "", false
	}

	annotations := mapValue(metadata, "annotations")
	if annotations == nil {
		key := scalar("annotations")
		key.HeadComment = pruneComment
		annotations = &yaml.Node{Kind: yaml.MappingNode}
		metadata.Content = append([]*yaml.Node{key, annotations}, metadata.Content...)
	}

	value := mapValue(annotations, pruneAnnotationKey)
	switch {
	case value == nil:
		annotations.Content = append(annotations.Content, scalar(pruneAnnotationKey), scalar(pruneDisabled))
		return fmt.Sprintf("annotated %s: %s, so Flux never prunes the management cluster's Kustomization", pruneAnnotationKey, pruneDisabled), true
	case value.Value != pruneDisabled:
		return fmt.Sprintf("left, its %s annotation is %q, not %q", pruneAnnotationKey, value.Value, pruneDisabled), false
	}

	return "", false
}

// gateControlPlane replaces the health check kubectl-gs used to write in the
// cluster's Kustomizations, on the Cluster being Available, with the one on
// its control plane. A health check on the Cluster that differs is the
// user's, and is left.
func gateControlPlane(fs *afero.Afero, repoPath, wcPath string) ([]string, error) {
	dir := filepath.Dir(filepath.Join(repoPath, wcPath))

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	edit := func(object *yaml.Node) (string, bool) {
		return replaceGate(object, filepath.Base(wcPath))
	}

	var changes []string
	for _, e := range entries {
		if e.IsDir() || !isYAML(filepath.Ext(e.Name())) || isKustomizationFile(e.Name()) {
			continue
		}

		c, err := editFluxKustomizations(fs, repoPath, filepath.Join(dir, e.Name()), filepath.ToSlash(wcPath), edit)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, c...)
	}

	return changes, nil
}

func replaceGate(object *yaml.Node, wc string) (string, bool) {
	spec := mapValue(object, "spec")
	if spec == nil {
		return "", false
	}

	for _, expr := range sequence(mapValue(spec, "healthCheckExprs")) {
		current := mapValue(expr, "current")
		if !isClusterCheck(expr) || current == nil {
			continue
		}

		switch current.Value {
		case availableGate:
			current.Value = controlPlaneGate
			current.Style = yaml.FoldedStyle
			updateGateComment(spec, wc)
			return "waits for the cluster's control plane, rather than for the Cluster to be Available", true
		case controlPlaneGate:
			continue
		}

		return "health check on the Cluster left, as it differs from the one kubectl-gs wrote", false
	}

	return "", false
}

func isClusterCheck(expr *yaml.Node) bool {
	apiVersion, kind := mapValue(expr, "apiVersion"), mapValue(expr, "kind")

	return apiVersion != nil && kind != nil && kind.Value == "Cluster" && strings.HasPrefix(apiVersion.Value, "cluster.x-k8s.io/")
}

// updateGateComment rewrites the comment kubectl-gs wrote above the health
// check, which says what it waits for. Any other comment is the user's.
func updateGateComment(spec *yaml.Node, wc string) {
	for i := 0; i+1 < len(spec.Content); i += 2 {
		key := spec.Content[i]
		if key.Value == "healthCheckExprs" && strings.Contains(key.HeadComment, "reports the cluster Available") {
			key.HeadComment = fmt.Sprintf(gateComment, wc)
		}
	}
}
