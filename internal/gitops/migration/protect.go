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
	pruneComment = `It applies itself, so renaming it would make it prune itself, and its
finalizer would delete everything it applied. The annotation leaves it, as
it does each organisation's namespace Kustomization.`
)

// editDocument changes a document in place, and returns what to report, and
// whether it changed it. An empty report says nothing.
type editDocument func(object *yaml.Node) (string, bool)

// appliedDir selects the Flux Kustomizations that apply a directory, relative
// to the repository root.
type appliedDir string

// selects reports whether the object is a Flux Kustomization applying the
// directory.
func (a appliedDir) selects(object *yaml.Node) bool {
	if object == nil {
		return false
	}
	p := fluxPath(object)
	if p == nil {
		return false
	}

	return path.Clean(p.Value) == string(a)
}

// editFluxKustomizations applies edit to the Flux Kustomizations of the file
// that `applied` picks, document by document, keeping every other document
// byte for byte, as fixFluxPath does.
func editFluxKustomizations(fs *afero.Afero, repoPath, file string, applied appliedDir, edit editDocument) ([]string, error) {
	data, err := fs.ReadFile(file)
	if err != nil || !strings.Contains(string(data), string(applied)) {
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

	changes, err := editFluxKustomizations(fs, repoPath, file, appliedDir(filepath.ToSlash(mcPath)), disablePruning)

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

	value := mapValue(annotations, pruneAnnotation)
	switch {
	case value == nil:
		annotations.Content = append(annotations.Content, scalar(pruneAnnotation), scalar(pruneDisabled))
		return fmt.Sprintf("annotated %s: %s, so Flux never prunes the management cluster's Kustomization", pruneAnnotation, pruneDisabled), true
	case value.Value != pruneDisabled:
		return fmt.Sprintf("left, its %s annotation is %q, not %q", pruneAnnotation, value.Value, pruneDisabled), false
	}

	return "", false
}
