package deploychart

import (
	"fmt"

	"github.com/pmezard/go-difflib/difflib"
	"sigs.k8s.io/yaml"
)

// DiffManifests computes a unified diff between an existing resource (from the cluster)
// and a desired manifest. Both inputs are YAML bytes.
// Returns an empty string if no differences are found.
func DiffManifests(resourceName string, existing, desired []byte) (string, error) {
	var existingMap map[string]any
	if err := yaml.Unmarshal(existing, &existingMap); err != nil {
		return "", fmt.Errorf("unmarshaling existing resource: %w", err)
	}

	var desiredMap map[string]any
	if err := yaml.Unmarshal(desired, &desiredMap); err != nil {
		return "", fmt.Errorf("unmarshaling desired resource: %w", err)
	}

	cleanForDiff(existingMap)
	cleanForDiff(desiredMap)

	existingYAML, err := yaml.Marshal(existingMap)
	if err != nil {
		return "", fmt.Errorf("marshaling existing for diff: %w", err)
	}

	desiredYAML, err := yaml.Marshal(desiredMap)
	if err != nil {
		return "", fmt.Errorf("marshaling desired for diff: %w", err)
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(existingYAML)),
		B:        difflib.SplitLines(string(desiredYAML)),
		FromFile: resourceName + " (existing)",
		ToFile:   resourceName + " (desired)",
		Context:  3,
	}

	return difflib.GetUnifiedDiffString(diff)
}

// cleanForDiff removes server-managed fields that would cause spurious diffs.
func cleanForDiff(obj map[string]any) {
	delete(obj, "status")

	metadata, ok := obj["metadata"].(map[string]any)
	if !ok {
		return
	}
	delete(metadata, "managedFields")
	delete(metadata, "resourceVersion")
	delete(metadata, "uid")
	delete(metadata, "creationTimestamp")
	delete(metadata, "generation")
	delete(metadata, "selfLink")

	// Strip well-known annotations added by apply mechanisms.
	annotations, ok := metadata["annotations"].(map[string]any)
	if ok {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
	// Remove empty annotations map to avoid spurious diff against no-annotations desired state.
	if annot, ok := metadata["annotations"].(map[string]any); ok && len(annot) == 0 {
		delete(metadata, "annotations")
	}

	// Same for labels.
	if labels, ok := metadata["labels"].(map[string]any); ok && len(labels) == 0 {
		delete(metadata, "labels")
	}
}
