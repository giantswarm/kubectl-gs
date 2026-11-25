package migration

import (
	"fmt"
	"path"
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

// appliedDir selects the Flux Kustomizations that apply a directory.
type appliedDir struct {
	// dir is the directory, relative to the repository root.
	dir string
	// under selects those applying a directory under it too.
	under bool
}

// selects reports whether the object is a Flux Kustomization applying the
// directory, or one under it if under is set.
func (a appliedDir) selects(object *yaml.Node) bool {
	if object == nil {
		return false
	}
	p := fluxPath(object)
	if p == nil {
		return false
	}
	current := path.Clean(p.Value)

	return current == a.dir || a.under && strings.HasPrefix(current, a.dir+"/")
}

// editFluxKustomizations applies edit to the Flux Kustomizations of the file
// that `applied` picks, document by document, keeping every other document
// byte for byte, as fixFluxPath does.
func editFluxKustomizations(fs *afero.Afero, repoPath, file string, applied appliedDir, edit editDocument) ([]string, error) {
	data, err := fs.ReadFile(file)
	if err != nil || !strings.Contains(string(data), applied.dir) {
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
		if applied.selects(d.node) {
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

// protectManagementCluster gives the management cluster's Kustomization the
// prune: disabled annotation a newly generated one has, unless it has a
// prune annotation already. Only the Kustomization applying the management
// cluster's directory itself is: one applying a directory under it stays
// prunable, as a generated one is. A file git ignores is left alone.
func protectManagementCluster(fs *afero.Afero, repoPath, mcPath string, env Env) ([]string, error) {
	file := filepath.Join(repoPath, mcPath, gitopskey.FluxKustomizationFileName(filepath.Base(mcPath)))
	if env.IsIgnored(rel(repoPath, file)) {
		return nil, nil
	}

	exists, err := fs.Exists(file)
	if err != nil || !exists {
		return nil, microerror.Mask(err)
	}

	changes, err := editFluxKustomizations(fs, repoPath, file, appliedDir{dir: filepath.ToSlash(mcPath)}, disablePruning)

	return changes, microerror.Mask(err)
}

func disablePruning(object *yaml.Node) (string, bool) {
	metadata := mapValue(object, "metadata")
	if metadata == nil {
		return "", false
	}

	annotations := mapValue(metadata, "annotations")
	switch {
	case annotations == nil:
		key := scalar("annotations")
		key.HeadComment = pruneComment
		annotations = &yaml.Node{Kind: yaml.MappingNode}
		metadata.Content = append([]*yaml.Node{key, annotations}, metadata.Content...)
	case annotations.Kind == yaml.ScalarNode && annotations.ShortTag() == "!!null":
		// `annotations: null`, or nothing: no annotations yet. The node
		// becomes an empty mapping, keeping its comments.
		annotations.Kind, annotations.Tag, annotations.Value, annotations.Style = yaml.MappingNode, "", "", 0
	case annotations.Kind != yaml.MappingNode:
		return "left, its metadata.annotations is not a mapping", false
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
// user's, and is left, as is a file git ignores.
func gateControlPlane(fs *afero.Afero, repoPath, wcPath string, env Env) ([]string, error) {
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
		file := filepath.Join(dir, e.Name())
		if e.IsDir() || !isYAML(filepath.Ext(e.Name())) || isKustomizationFile(e.Name()) || env.IsIgnored(rel(repoPath, file)) {
			continue
		}

		c, err := editFluxKustomizations(fs, repoPath, file, appliedDir{dir: filepath.ToSlash(wcPath), under: true}, edit)
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

		switch {
		case current.Value == availableGate && isGeneratedGate(expr):
			current.Value = controlPlaneGate
			current.Style = yaml.FoldedStyle
			updateGateComment(spec, wc)
			return "waits for the cluster's control plane, rather than for the Cluster to be Available", true
		case current.Value == controlPlaneGate:
			continue
		}

		return "health check on the Cluster left, as it differs from the one kubectl-gs wrote", false
	}

	return "", false
}

// isGeneratedGate reports whether a health check on the Cluster is the entry
// kubectl-gs wrote, which has apiVersion, kind and current alone. One with
// expressions of its own, such as failed or inProgress, is the user's as a
// whole: its current is kept with them.
func isGeneratedGate(expr *yaml.Node) bool {
	return len(expr.Content) == 6
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
